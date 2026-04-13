"""Shared helpers for AgentExecutor implementations.

Used by both CLIAgentExecutor (codex, ollama) and ClaudeSDKExecutor (claude-code).
Provides:
- Memory recall/commit (HTTP to platform /memories endpoints)
- Delegation results consumption (atomic file rename)
- Current task heartbeat updates
- System prompt loading from /configs
- A2A instructions text for system prompt injection (MCP and CLI variants)
- Brief task summary extraction (markdown-aware)
- Error message sanitization (exception classes and subprocess categories)
- Shared workspace path constants and the MCP server path resolver
"""

from __future__ import annotations

import json
import logging
import os
import re
from pathlib import Path
from typing import TYPE_CHECKING, Any

import httpx

if TYPE_CHECKING:
    from heartbeat import HeartbeatLoop


logger = logging.getLogger(__name__)


# ========================================================================
# Constants — workspace container layout
# ========================================================================

WORKSPACE_MOUNT = "/workspace"
CONFIG_MOUNT = "/configs"
DEFAULT_MCP_SERVER_PATH = "/app/a2a_mcp_server.py"
DEFAULT_DELEGATION_RESULTS_FILE = "/tmp/delegation_results.jsonl"
PLATFORM_HTTP_TIMEOUT_S = 5.0
MEMORY_RECALL_LIMIT = 10
MEMORY_CONTENT_MAX_CHARS = 200
BRIEF_SUMMARY_MAX_LEN = 80


def get_mcp_server_path() -> str:
    """Return the path to the stdio MCP server script.

    Overridable via A2A_MCP_SERVER_PATH for tests and non-default layouts.
    """
    return os.environ.get("A2A_MCP_SERVER_PATH", DEFAULT_MCP_SERVER_PATH)


# ========================================================================
# HTTP client (shared, lazily initialised)
# ========================================================================

_http_client: httpx.AsyncClient | None = None


def get_http_client() -> httpx.AsyncClient:
    """Lazy-init a shared httpx client for platform API calls."""
    global _http_client
    if _http_client is None or _http_client.is_closed:
        _http_client = httpx.AsyncClient(timeout=PLATFORM_HTTP_TIMEOUT_S)
    return _http_client


def reset_http_client_for_tests() -> None:
    """Test helper — drop the shared client so the next call rebuilds it.

    Not for production use. Exposed so tests can guarantee a clean slate
    between cases without touching module internals.
    """
    global _http_client
    _http_client = None


# ========================================================================
# Memory recall + commit
# ========================================================================

async def recall_memories() -> str:
    """Recall recent memories from the platform API.

    Returns a newline-joined bullet list of up to MEMORY_RECALL_LIMIT most recent
    memories, or empty string when the platform is unreachable / not configured
    / returns a non-200 / returns an unexpected payload shape.
    """
    workspace_id = os.environ.get("WORKSPACE_ID", "")
    platform_url = os.environ.get("PLATFORM_URL", "")
    if not workspace_id or not platform_url:
        return ""
    try:
        resp = await get_http_client().get(
            f"{platform_url}/workspaces/{workspace_id}/memories",
        )
        if not 200 <= resp.status_code < 300:
            logger.debug(
                "recall_memories: non-2xx response %s from platform",
                resp.status_code,
            )
            return ""
        data = resp.json()
    except Exception as exc:
        logger.debug("recall_memories: request failed: %s", exc)
        return ""
    if not isinstance(data, list) or not data:
        return ""
    lines = [
        f"- [{m.get('scope', '?')}] {m.get('content', '')}"
        for m in data[-MEMORY_RECALL_LIMIT:]
    ]
    return "\n".join(lines)


async def commit_memory(content: str) -> None:
    """Save a memory to the platform API. Best-effort, no error propagation."""
    workspace_id = os.environ.get("WORKSPACE_ID", "")
    platform_url = os.environ.get("PLATFORM_URL", "")
    if not workspace_id or not platform_url or not content:
        return
    try:
        await get_http_client().post(
            f"{platform_url}/workspaces/{workspace_id}/memories",
            json={"content": content, "scope": "LOCAL"},
        )
    except Exception as exc:
        logger.debug("commit_memory: request failed: %s", exc)


# ========================================================================
# Delegation results — written by heartbeat loop, consumed atomically
# ========================================================================

def read_delegation_results() -> str:
    """Read and consume delegation results written by the heartbeat loop.

    Uses atomic rename to prevent races with the heartbeat writer.
    Returns formatted text suitable for prompt injection, or empty string.
    """
    results_file = Path(
        os.environ.get("DELEGATION_RESULTS_FILE", DEFAULT_DELEGATION_RESULTS_FILE)
    )
    if not results_file.exists():
        return ""
    consumed = results_file.with_suffix(".consumed")
    try:
        results_file.rename(consumed)
    except OSError:
        return ""  # File disappeared between exists() and rename()
    try:
        raw = consumed.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return ""
    finally:
        consumed.unlink(missing_ok=True)

    parts: list[str] = []
    for line in raw.strip().split("\n"):
        if not line.strip():
            continue
        try:
            record = json.loads(line)
        except json.JSONDecodeError:
            continue
        status = record.get("status", "?")
        summary = record.get("summary", "")
        preview = record.get("response_preview", "")
        parts.append(f"- [{status}] {summary}")
        if preview:
            parts.append(f"  Response: {preview[:200]}")
    return "\n".join(parts)


# ========================================================================
# Current task heartbeat update
# ========================================================================

async def set_current_task(heartbeat: "HeartbeatLoop | None", task: str) -> None:
    """Update current task on heartbeat and push immediately via platform API."""
    if heartbeat is not None:
        heartbeat.current_task = task
        heartbeat.active_tasks = 1 if task else 0
    workspace_id = os.environ.get("WORKSPACE_ID", "")
    platform_url = os.environ.get("PLATFORM_URL", "")
    if not (workspace_id and platform_url):
        return
    try:
        try:
            from platform_auth import auth_headers as _auth
            _headers = _auth()
        except Exception:
            _headers = {}
        await get_http_client().post(
            f"{platform_url}/registry/heartbeat",
            json={
                "workspace_id": workspace_id,
                "current_task": task,
                "active_tasks": 1 if task else 0,
                "error_rate": 0,
                "sample_error": "",
                "uptime_seconds": 0,
            },
            headers=_headers,
        )
    except Exception as exc:
        logger.debug("set_current_task: heartbeat push failed: %s", exc)


# ========================================================================
# System prompt loading
# ========================================================================

def get_system_prompt(config_path: str, fallback: str | None = None) -> str | None:
    """Read system-prompt.md from the config dir each call (supports hot-reload).

    Falls back to the provided string if the file doesn't exist.
    """
    prompt_file = Path(config_path) / "system-prompt.md"
    if prompt_file.exists():
        return prompt_file.read_text(encoding="utf-8", errors="replace").strip()
    return fallback


_A2A_INSTRUCTIONS_MCP = """## Inter-Agent Communication
You have MCP tools for communicating with other workspaces:
- list_peers: discover available peer workspaces (name, ID, status, role)
- delegate_task: send a task and WAIT for the response (for quick tasks)
- delegate_task_async: send a task and return immediately with a task_id (for long tasks)
- check_task_status: poll an async task's status and get results when done
- get_workspace_info: get your own workspace info

For quick questions, use delegate_task (synchronous).
For long-running work (building pages, running audits), use delegate_task_async + check_task_status.
Always use list_peers first to discover available workspace IDs.
Access control is enforced — you can only reach siblings and parent/children.

PROACTIVE MESSAGING: Use send_message_to_user to push messages to the user's chat at ANY time:
- Acknowledge tasks immediately: "Got it, delegating to the team now..."
- Send progress updates during long work: "Research Lead finished, waiting on Dev Lead..."
- Deliver follow-up results: "All teams reported back. Here's the synthesis: ..."
This lets you respond quickly ("I'll work on this") and come back later with results.

If delegate_task returns a DELEGATION FAILED message, do NOT forward the raw error to the user.
Instead: (1) try delegating to a different peer, (2) handle the task yourself, or
(3) tell the user which peer is unavailable and provide your own best answer."""


_A2A_INSTRUCTIONS_CLI = """## Inter-Agent Communication
You can delegate tasks to other workspaces using the a2a command:
  python3 /app/a2a_cli.py peers                                  # List available peers
  python3 /app/a2a_cli.py delegate <workspace_id> <task>          # Sync: wait for response
  python3 /app/a2a_cli.py delegate --async <workspace_id> <task>  # Async: return task_id
  python3 /app/a2a_cli.py status <workspace_id> <task_id>         # Check async task
  python3 /app/a2a_cli.py info                                    # Your workspace info

For quick questions, use sync delegate. For long tasks, use --async + status.
Only delegate to peers listed by the peers command (access control enforced)."""


def get_a2a_instructions(mcp: bool = True) -> str:
    """Return inter-agent communication instructions for system-prompt injection.

    Pass `mcp=True` (default) for MCP-capable runtimes (Claude Code via SDK,
    Codex). Pass `mcp=False` for CLI-only runtimes (Ollama, custom) that have
    to call a2a_cli.py as a subprocess.
    """
    return _A2A_INSTRUCTIONS_MCP if mcp else _A2A_INSTRUCTIONS_CLI


# ========================================================================
# Misc text helpers
# ========================================================================

_MARKDOWN_FENCE = "```"
_MARKDOWN_HR = "---"


_BRIEF_SUMMARY_MIN_LEN = 4  # 1 char + 3-char ellipsis


def brief_summary(text: str, max_len: int = BRIEF_SUMMARY_MAX_LEN) -> str:
    """Extract a one-line task summary for the canvas card display.

    Strips markdown headers (#, ##, ###), bold/italic markers (**, __),
    and skips code fences and horizontal rules. Returns the first meaningful
    line, truncated with an ellipsis when it exceeds `max_len`.

    `max_len` is clamped to at least 4 (one real character plus a 3-char
    ellipsis) so degenerate callers can't produce negative slice indices.
    """
    max_len = max(max_len, _BRIEF_SUMMARY_MIN_LEN)
    for raw_line in text.split("\n"):
        line = raw_line.strip()
        while line.startswith("#"):
            line = line[1:]
        line = line.strip()
        if not line or line.startswith(_MARKDOWN_FENCE) or line == _MARKDOWN_HR:
            continue
        line = line.replace("**", "").replace("__", "")
        if len(line) > max_len:
            return line[: max_len - 3] + "..."
        return line
    return text[:max_len]


def extract_message_text(message: Any) -> str:
    """Extract text from an A2A message (handles both .text and .root.text patterns)."""
    parts = getattr(message, "parts", None) or []
    text_parts: list[str] = []
    for part in parts:
        text = getattr(part, "text", None)
        if text:
            text_parts.append(text)
            continue
        root = getattr(part, "root", None)
        if root is not None:
            root_text = getattr(root, "text", None)
            if root_text:
                text_parts.append(root_text)
    return " ".join(text_parts).strip()


# Word-boundary patterns for subprocess stderr classification. Using word
# boundaries avoids false positives like "author" matching "auth" or
# "generate" matching "rate".
_RATE_LIMIT_RE = re.compile(r"\brate\b|\b429\b|\boverloaded\b", re.IGNORECASE)
_AUTH_RE = re.compile(r"\bauth(?:entication|orization)?\b|\bapi[_-]?key\b", re.IGNORECASE)
_SESSION_RE = re.compile(r"\bsession\b|\bno conversation found\b", re.IGNORECASE)


def classify_subprocess_error(stderr_text: str, exit_code: int | None) -> str:
    """Map a subprocess stderr blob to a short, user-safe category tag.

    The full stderr goes to the workspace logs via `logger.error`; only the
    category is surfaced to the user to avoid leaking tokens, internal paths,
    or stack traces in the chat UI. Used with `sanitize_agent_error` to
    produce a user-facing message for subprocess failures.
    """
    if _RATE_LIMIT_RE.search(stderr_text):
        return "rate_limited"
    if _AUTH_RE.search(stderr_text):
        return "auth_failed"
    if _SESSION_RE.search(stderr_text):
        return "session_error"
    if exit_code is not None and exit_code != 0:
        return f"exit_{exit_code}"
    return "subprocess_error"


def sanitize_agent_error(
    exc: BaseException | None = None,
    category: str | None = None,
) -> str:
    """Render an agent-side failure into a user-safe error message.

    Either pass an exception (class name is used as the tag) or an explicit
    category string (e.g. from `classify_subprocess_error`). If both are
    given, `category` wins. If neither, the tag defaults to "unknown".

    The message body is deliberately dropped — exception messages and
    subprocess stderr frequently leak stack traces, paths, tokens, and
    API keys. Full detail is available in the workspace logs via
    `logger.exception()` / `logger.error()`.
    """
    if category:
        tag = category
    elif exc is not None:
        tag = type(exc).__name__
    else:
        tag = "unknown"
    return f"Agent error ({tag}) — see workspace logs for details."
