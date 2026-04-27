"""SDK-based agent executor for Claude Code runtime.

Uses the official `claude-agent-sdk` Python package to invoke the Claude Code
engine programmatically — no subprocess, no stdout parsing, no zombie reap.

Replaces CLIAgentExecutor for the `claude-code` runtime only. Other CLI runtimes
(codex, ollama) keep using `cli_executor.py`.

Benefits over CLI subprocess:
- No per-message ~500ms startup overhead
- No stdout buffering issues
- Native Python session management (no JSON parsing of stdout)
- Real message stream — can surface tool calls in future for live UX
- Cooperative cancel (closes the query async generator on cancel())
- Same Claude Code engine, so plugins / skills / CLAUDE.md still apply

Concurrency model
-----------------
Turns are serialized per-executor via an asyncio.Lock. The old CLI executor
serialized implicitly by spawning one subprocess per message and awaiting it;
the SDK removes that, so we re-introduce serialization explicitly. This keeps
session_id updates race-free and makes cancel() well-defined (there's at most
one active stream at any given moment).
"""

from __future__ import annotations

import asyncio
import logging
import os
import sys
from collections.abc import AsyncIterator, Callable
from dataclasses import dataclass
from typing import TYPE_CHECKING, Any

import yaml

import claude_agent_sdk as sdk

from a2a.server.agent_execution import AgentExecutor, RequestContext
from a2a.server.events import EventQueue
from a2a.helpers import new_agent_text_message

from executor_helpers import (
    CONFIG_MOUNT,
    MEMORY_CONTENT_MAX_CHARS,
    WORKSPACE_MOUNT,
    auto_push_hook,
    brief_summary,
    collect_outbound_files,
    commit_memory,
    extract_attached_files,
    extract_message_text,
    get_a2a_instructions,
    get_hma_instructions,
    get_mcp_server_path,
    get_system_prompt,
    read_delegation_results,
    recall_memories,
    sanitize_agent_error,
    set_current_task,
)

if TYPE_CHECKING:
    from heartbeat import HeartbeatLoop

logger = logging.getLogger(__name__)

_NO_TEXT_MSG = "Error: message contained no text content."
_NO_RESPONSE_MSG = "(no response generated)"
_MAX_RETRIES = 3
_BASE_RETRY_DELAY_S = 5
# Cap for stderr captured from the CLI subprocess in the executor log. Keeps
# log lines bounded while still surfacing enough context to diagnose crashes.
# Fixes #66 (previously the executor logged nothing beyond the generic
# "Check stderr output for details" message).
_PROCESS_ERROR_STDERR_MAX_CHARS = 4096

# Substrings in error messages that indicate a transient failure worth retrying.
_RETRYABLE_PATTERNS = (
    "rate",
    "limit",
    "429",
    "overloaded",
    "capacity",
    "exit code 1",
    "try again",
)

# SDK-wedge state lives in the runtime-side module (runtime_wedge) so
# heartbeat.py and any future cross-cutting consumer can read it without
# importing this adapter-specific executor. Decoupling was the prerequisite
# for moving claude_sdk_executor out of molecule-runtime into the
# claude-code template repo (task #87 — universal-runtime refactor).
#
# Local re-exports keep the in-file call sites (_run_query etc.) terse
# and preserve the historical names so the behavior is identical to
# the pre-extraction version. is_wedged/wedge_reason are also re-exported
# so any external consumer that imported them from this module keeps
# working — heartbeat.py has been updated to import from runtime_wedge
# directly, but a third-party adapter copying our wedge convention may
# still expect them here.
from runtime_wedge import (  # noqa: E402
    clear_wedge as _clear_sdk_wedge_on_success,
    is_wedged,
    mark_wedged as _mark_sdk_wedged,
    reset_for_test as _reset_sdk_wedge_for_test,
    wedge_reason,
)


# Per-tool-use summarizers. Reads the most-useful argument from each
# tool's input dict so the canvas progress feed shows
# `🛠 Read /tmp/foo` instead of the bare tool name. Anything not in the
# table falls through to a generic "🛠 <tool>(…)" line. Order keys by
# tool frequency so a future contributor can see the high-traffic
# tools first.
_TOOL_USE_SUMMARIZERS: dict[str, Callable[[dict], str]] = {
    "Read":  lambda i: f"📄 Read {i.get('file_path', '?')}",
    "Write": lambda i: f"✍️  Write {i.get('file_path', '?')}",
    "Edit":  lambda i: f"✏️  Edit {i.get('file_path', '?')}",
    "Bash":  lambda i: f"⚡ Bash: {(i.get('command') or '')[:80]}",
    "Glob":  lambda i: f"🔍 Glob {i.get('pattern', '?')}",
    "Grep":  lambda i: f"🔍 Grep {i.get('pattern', '?')}",
    "WebFetch": lambda i: f"🌐 WebFetch {i.get('url', '?')}",
    "WebSearch": lambda i: f"🌐 WebSearch {i.get('query', '?')}",
    "Task":  lambda i: f"🤖 Task: {(i.get('description') or '')[:60]}",
    "TodoWrite": lambda _i: "📝 TodoWrite",
}


def _summarize_tool_use(tool_name: str, tool_input: dict) -> str:
    summarizer = _TOOL_USE_SUMMARIZERS.get(tool_name)
    if summarizer:
        try:
            return summarizer(tool_input or {})[:200]
        except Exception:
            pass
    # Generic fallback. Truncated so a tool with a giant input dict
    # doesn't write a 10kB activity row per call.
    return f"🛠 {tool_name}(…)"[:200]


async def _report_tool_use(block: Any) -> None:
    """Fire-and-forget agent_log activity row per tool the SDK invoked,
    so the canvas's MyChat live-progress feed can render each step
    Claude is doing instead of staring at a single spinner.

    Posts directly to /workspaces/:id/activity rather than through
    a2a_tools.report_activity — that helper also pushes a current_task
    heartbeat which would duplicate as a TASK_UPDATED line in the
    chat feed. The workspace card's current_task is already set
    once per turn by the executor's set_current_task(brief_summary)
    call, so the per-tool telemetry stays a chat-only signal.

    Best-effort — any failure (network blip, platform unreachable, the
    block didn't have the attrs we expected) is swallowed silently.
    The tool will still execute regardless; only the progress
    telemetry is lost. Deliberately does NOT raise — a malformed
    block must not abort the message-stream iteration in
    `_run_query`.
    """
    try:
        # Lazy imports to keep this helper non-essential — the
        # executor must still run when the workspace's network/auth
        # plumbing isn't fully set up (e.g. unit tests).
        import httpx
        from a2a_client import PLATFORM_URL, WORKSPACE_ID
        from platform_auth import auth_headers
    except Exception:
        return
    try:
        tool_name = getattr(block, "name", "") or ""
        tool_input = getattr(block, "input", {}) or {}
        if not tool_name:
            return
        summary = _summarize_tool_use(tool_name, tool_input)
        # 5s budget — long enough to absorb a single platform GC
        # pause, short enough that a wedged platform doesn't slow
        # the tool-iteration cadence beyond noticeable.
        async with httpx.AsyncClient(timeout=5.0) as client:
            await client.post(
                f"{PLATFORM_URL}/workspaces/{WORKSPACE_ID}/activity",
                json={
                    "activity_type": "agent_log",
                    "source_id": WORKSPACE_ID,
                    # target_id == source for self-actions. Matches the
                    # convention other self-logged activity rows use
                    # (a2a_receive when the workspace logs its own
                    # outbound reply) so DB consumers joining on
                    # target_id see a well-defined value.
                    "target_id": WORKSPACE_ID,
                    "summary": summary,
                    "status": "ok",
                    "method": tool_name,
                },
                headers=auth_headers(),
            )
    except Exception:
        # Telemetry failures must not break the conversation.
        return


# Substring patterns that classify an exception as the specific
# claude_agent_sdk init-timeout wedge (vs. a rate-limit, transient
# subprocess crash, etc.). Match is case-insensitive on the formatted
# error string. Adding a new pattern here MUST come with a test in
# tests/test_claude_sdk_executor.py — false-positives lock the
# workspace into degraded until the next successful query clears it.
#
# `:initialize` suffix-anchored — the SDK can theoretically time out
# on later control messages (in-flight tool callbacks), but those
# don't leave the SDK in the unrecoverable post-init state we're
# trying to detect. Limit the pattern to the specific wedge.
_WEDGE_ERROR_PATTERNS = (
    "control request timeout: initialize",
)


_SWALLOWED_STDERR_MARKER = "Check stderr output for details"


def _probe_claude_cli_error() -> str | None:
    """Run ``claude --print`` directly and capture its stderr + stdout.

    Used as a fallback when the claude-agent-sdk raises a bare ``Exception``
    with the swallowed "Check stderr output for details" placeholder — that
    happens when the SDK wraps a stream error from the CLI subprocess and
    loses both the ``.stderr`` attribute and the exit code. At that point
    the only way to see the real failure reason (rate limit, auth error,
    network outage, missing token) is to run the CLI ourselves.

    Bounded by a 30s timeout so a hung CLI can't stall the error path.
    Returns None if the probe itself failed (wrong invariant — don't
    corrupt the main error message with probe noise).
    """
    try:
        import subprocess
        # --print reads stdin, prints response, exits. Empty stdin gives the
        # CLI something to work with without triggering an actual model call
        # when it's going to fail anyway.
        proc = subprocess.run(
            ["claude", "--print"],
            input="probe",
            capture_output=True,
            text=True,
            timeout=30,
        )
        if proc.returncode == 0:
            # CLI succeeded — the original error was a transient state that
            # resolved between the SDK failure and our probe. Signal that.
            return "<cli probe succeeded — error was transient>"
        raw = (proc.stderr or "") + (proc.stdout or "")
        raw = raw.strip()
        if not raw:
            return f"<cli exited {proc.returncode} with empty output>"
        if len(raw) > _PROCESS_ERROR_STDERR_MAX_CHARS:
            raw = raw[:_PROCESS_ERROR_STDERR_MAX_CHARS] + "... [truncated]"
        return raw
    except Exception as probe_exc:  # pragma: no cover — best-effort diagnostic
        return f"<probe failed: {type(probe_exc).__name__}: {probe_exc}>"


def _format_process_error(exc: BaseException) -> str:
    """Render a Claude-SDK ProcessError (or any ClaudeSDKError) with its full
    captured context — exit code, stderr, exception type. Plain strings for
    non-SDK exceptions fall back to str(exc).

    Bounded at _PROCESS_ERROR_STDERR_MAX_CHARS so a runaway CLI can't spam
    the log. Used by the executor's error path (fixes #66 — the SDK's
    ProcessError carries `.stderr`/`.exit_code` attributes that the previous
    code silently discarded, leaving every CLI crash with an identical
    "Check stderr output for details" message in the workspace log).

    Fixes #160: when the SDK raises a bare ``Exception`` containing the
    "Check stderr output for details" placeholder (which happens when the
    CLI subprocess emits a stream error the SDK can't categorize — rate
    limit, auth, network), there's no ``.stderr``/``.exit_code`` to read.
    In that case we fall back to running the CLI ourselves via
    ``_probe_claude_cli_error`` so the operator sees the real failure
    reason (e.g. ``You've hit your limit · resets Apr 17``) instead of
    chasing ghosts in the workspace logs.
    """
    parts = [f"{type(exc).__name__}: {exc}"]
    exit_code = getattr(exc, "exit_code", None)
    if exit_code is not None:
        parts.append(f"exit_code={exit_code}")
    stderr = getattr(exc, "stderr", None)
    if stderr:
        trimmed = stderr[:_PROCESS_ERROR_STDERR_MAX_CHARS]
        if len(stderr) > _PROCESS_ERROR_STDERR_MAX_CHARS:
            trimmed += f"... [{len(stderr) - _PROCESS_ERROR_STDERR_MAX_CHARS} more chars truncated]"
        parts.append(f"stderr={trimmed!r}")
    elif exit_code is None and _SWALLOWED_STDERR_MARKER in str(exc):
        # #160: generic exception with the swallowed-stderr placeholder.
        # Probe the CLI directly — this is the only way to surface the real
        # error when the SDK lost it in translation.
        probed = _probe_claude_cli_error()
        if probed:
            parts.append(f"probed_cli_error={probed!r}")
    return " | ".join(parts)


@dataclass
class QueryResult:
    """Outcome of a single `query()` stream.

    `text` is the canonical final response; `session_id` is the id the SDK
    reports in its ResultMessage (used for resume on the next turn).
    """
    text: str
    session_id: str | None


class ClaudeSDKExecutor(AgentExecutor):
    """Executes agent tasks via the claude-agent-sdk programmatic API."""

    def __init__(
        self,
        system_prompt: str | None,
        config_path: str,
        heartbeat: "HeartbeatLoop | None",
        model: str = "sonnet",
    ):
        self.system_prompt = system_prompt
        self.config_path = config_path
        self.heartbeat = heartbeat
        self.model = model
        self._session_id: str | None = None
        self._active_stream: AsyncIterator[Any] | None = None
        # Serializes concurrent execute() calls on the same executor so
        # session_id / _active_stream mutations stay race-free.
        self._run_lock = asyncio.Lock()

    # ------------------------------------------------------------------
    # Prompt + options builders
    # ------------------------------------------------------------------

    def _resolve_cwd(self) -> str:
        """Run in /workspace if it has been populated, otherwise /configs."""
        if os.path.isdir(WORKSPACE_MOUNT) and os.listdir(WORKSPACE_MOUNT):
            return WORKSPACE_MOUNT
        return CONFIG_MOUNT

    def _build_system_prompt(self) -> str | None:
        """Compose system prompt from file + A2A + HMA memory instructions."""
        base = get_system_prompt(self.config_path, fallback=self.system_prompt)
        a2a = get_a2a_instructions(mcp=True)
        hma = get_hma_instructions()
        parts = [p for p in (base, a2a, hma) if p]
        return "\n\n".join(parts) if parts else None

    def _prepare_prompt(self, user_input: str) -> str:
        """Prepend delegation results that arrived while idle."""
        delegation_context = read_delegation_results()
        if delegation_context:
            return (
                "[Delegation results received while you were idle]\n"
                f"{delegation_context}\n\n[New message]\n{user_input}"
            )
        return user_input

    async def _inject_memories_if_first_turn(self, prompt: str) -> str:
        if self._session_id:
            return prompt
        memories = await recall_memories()
        if not memories:
            return prompt
        return f"[Prior context from memory]\n{memories}\n\n{prompt}"

    def _load_config_dict(self) -> dict:
        """Read config.yaml as a raw dict for field-level inspection.

        Returns an empty dict on any I/O or parse error so callers can
        always use ``.get()`` without guards.
        """
        try:
            config_file = os.path.join(self.config_path, "config.yaml")
            with open(config_file) as f:
                return yaml.safe_load(f) or {}
        except Exception:
            return {}

    def _build_options(self) -> Any:
        """Build ClaudeAgentOptions.

        No allowed_tools allowlist — bypassPermissions grants full access,
        matching the old CLI `--dangerously-skip-permissions` so Claude can
        use every built-in tool (Task, TodoWrite, NotebookEdit, BashOutput/
        KillShell, ExitPlanMode, etc.) plus all MCP tools.

        The MCP server launcher uses `sys.executable` so tests and alternate
        virtual-env layouts don't depend on a `python3` shim being on PATH.

        output_config wiring (issue #652)
        ----------------------------------
        Reads ``effort`` and ``task_budget`` from config.yaml and populates
        ``output_config`` on the SDK options before the API call:

        - ``effort`` (str): one of low|medium|high|xhigh|max.  xhigh is the
          Opus 4.7 recommended default for long agentic tasks.
        - ``task_budget`` (int): advisory total-token budget across the full
          agentic loop.  Must be >= 20000 (API minimum) or 0/absent (unset).
          When set, the ``task-budgets-2026-03-13`` beta header is added so
          the API accepts the field.
        """
        mcp_servers = {
            "a2a": {
                "command": sys.executable,
                "args": [get_mcp_server_path()],
            }
        }

        create_kwargs: dict = dict(
            model=self.model,
            permission_mode="bypassPermissions",
            cwd=self._resolve_cwd(),
            mcp_servers=mcp_servers,
            system_prompt=self._build_system_prompt(),
            resume=self._session_id,
        )

        # --- output_config: effort + task_budget (issue #652) ---
        config = self._load_config_dict()
        output_config: dict = {}
        effort = config.get("effort", "")
        task_budget = config.get("task_budget", 0)

        if effort:
            output_config["effort"] = effort  # "low"|"medium"|"high"|"xhigh"|"max"

        if task_budget and int(task_budget) >= 20000:
            output_config["task_budget"] = {
                "type": "tokens",
                "total": int(task_budget),
            }
            betas = list(create_kwargs.get("betas", []))
            if "task-budgets-2026-03-13" not in betas:
                betas.append("task-budgets-2026-03-13")
            create_kwargs["betas"] = betas
        elif task_budget and int(task_budget) > 0:
            # Below minimum — reject clearly before any API call is made.
            raise ValueError(
                f"task_budget must be >= 20000 tokens (got {task_budget})"
            )

        if output_config:
            create_kwargs["output_config"] = output_config

        return sdk.ClaudeAgentOptions(**create_kwargs)

    # ------------------------------------------------------------------
    # Query streaming
    # ------------------------------------------------------------------

    async def _run_query(self, prompt: str, options: Any) -> QueryResult:
        """Drive the SDK query stream and return a QueryResult.

        Prefers ResultMessage.result (the canonical final text — same field
        the CLI's --output-format json used) and only falls back to the
        concatenation of AssistantMessage TextBlocks when result is absent.
        Otherwise pre-tool reasoning and post-tool summary get double-emitted.

        Pure: does not mutate executor state other than setting / clearing
        `self._active_stream` so cancel() can reach in. The caller decides
        whether to persist the returned session_id.
        """
        assistant_chunks: list[str] = []
        result_text: str | None = None
        session_id: str | None = None
        self._active_stream = sdk.query(prompt=prompt, options=options)
        try:
            async for message in self._active_stream:
                if isinstance(message, sdk.AssistantMessage):
                    for block in message.content:
                        if isinstance(block, sdk.TextBlock):
                            assistant_chunks.append(block.text)
                        else:
                            # ToolUseBlock / ServerToolUseBlock are present
                            # on the real SDK but not on the conftest stub —
                            # check by class name to avoid an isinstance()
                            # against a class the stub doesn't define.
                            cls = type(block).__name__
                            if cls in ("ToolUseBlock", "ServerToolUseBlock"):
                                await _report_tool_use(block)
                elif isinstance(message, sdk.ResultMessage):
                    sid = getattr(message, "session_id", None)
                    if sid:
                        session_id = sid
                    result_text = getattr(message, "result", None)
        finally:
            self._active_stream = None
        text = result_text if result_text is not None else "".join(assistant_chunks)
        # Auto-recover the wedge flag — if a previous query() left this
        # process in `_sdk_wedged` and THIS query just completed
        # cleanly, the SDK clearly works again. Clear so the next
        # heartbeat reports runtime_state empty and the platform flips
        # status degraded → online without a manual restart.
        #
        # Gate on actual content from the stream so a degenerate
        # "iterator returned without raising but emitted nothing"
        # case (possible from a partial stream or a stub SDK) doesn't
        # falsely advertise recovery. A real successful query yields
        # at least a ResultMessage (sets result_text) or one
        # AssistantMessage TextBlock (populates assistant_chunks).
        if result_text is not None or assistant_chunks:
            _clear_sdk_wedge_on_success()
        return QueryResult(text=text, session_id=session_id)

    # ------------------------------------------------------------------
    # AgentExecutor interface
    # ------------------------------------------------------------------

    async def execute(self, context: RequestContext, event_queue: EventQueue):
        """Run a turn through the Claude Agent SDK and emit the response.

        Serialized via `self._run_lock` — concurrent A2A messages to the same
        workspace queue rather than racing on `_session_id` / `_active_stream`.
        """
        user_input = extract_message_text(context.message)
        # Surface attached files to claude-code via a manifest in the prompt.
        # Claude Code reads files through its own Read/Glob tools by path —
        # as long as the prompt names the path, the CLI will open them on
        # demand. Same contract every platform runtime uses so the UX is
        # identical across hermes / langgraph / claude-code.
        attached = extract_attached_files(context.message)
        if attached:
            manifest = "\n\nAttached files:\n" + "\n".join(
                f"- {f['name']} ({f['mime_type'] or 'unknown type'}) at {f['path']}"
                for f in attached
            )
            user_input = (user_input + manifest) if user_input else manifest.lstrip()
        if not user_input:
            await event_queue.enqueue_event(new_agent_text_message(_NO_TEXT_MSG))
            return

        async with self._run_lock:
            response_text = await self._execute_locked(user_input)

        # Enqueue outside the lock so the next queued turn can start
        # preparing its prompt while this turn's response ships. Event
        # ordering is preserved per-queue by the A2A server, so no races.
        # If the response mentions /workspace/... files, stage each and
        # emit FileParts alongside the text so the canvas can download.
        outbound = collect_outbound_files(response_text)
        if outbound:
            from a2a.types import FilePart, FileWithUri, Message, Part, Role, TextPart
            import uuid as _uuid
            parts: list = [Part(root=TextPart(text=response_text))] if response_text else []
            for f in outbound:
                parts.append(Part(root=FilePart(file=FileWithUri(
                    uri="workspace:" + f["path"],
                    name=f["name"],
                    mimeType=f["mime_type"],
                ))))
            await event_queue.enqueue_event(Message(
                messageId=_uuid.uuid4().hex,
                role=Role.agent,
                parts=parts,
            ))
        else:
            await event_queue.enqueue_event(new_agent_text_message(response_text))

    @staticmethod
    def _is_retryable(exc: BaseException) -> bool:
        """Check if an SDK exception looks like a transient rate-limit or
        capacity error that's worth retrying with backoff."""
        msg = str(exc).lower()
        return any(p in msg for p in _RETRYABLE_PATTERNS)

    def _reset_session_after_error(self, exc: BaseException) -> None:
        """Clear `_session_id` if the exception looks like a subprocess
        crash (#75). On the next `_build_options()` call `resume=None` is
        passed to the SDK, so the CLI boots a brand-new session instead of
        trying to resume one the previous subprocess left in an
        unrecoverable state.

        Kept in its own method so the policy can evolve (e.g. also clear
        on MessageParseError) without touching the retry loop. Logs at
        INFO when a session was actually cleared; silent when there was
        nothing to reset.
        """
        exc_name = type(exc).__name__
        # Conservative: reset only on subprocess-level failures. Pure
        # rate-limit / capacity errors don't leave the session in a bad
        # state — keep the session_id so the resumed turn preserves
        # conversational continuity.
        is_subprocess_error = (
            exc_name in ("ProcessError", "CLIConnectionError")
            or getattr(exc, "exit_code", None) is not None
            or "exit code" in str(exc).lower()
        )
        if not is_subprocess_error:
            return
        if self._session_id is None:
            return
        logger.info(
            "SDK session reset after %s: clearing session_id so the next "
            "attempt starts fresh (fixes #75 session contamination)",
            exc_name,
        )
        self._session_id = None

    async def _execute_locked(self, user_input: str) -> str:
        """Body of execute() that runs under the run lock.

        Retries transient errors (rate limits, capacity, exit-code-1) up to
        _MAX_RETRIES times with exponential backoff (5s, 10s, 20s).
        """
        # Keep a clean copy of the user's actual message for the memory record,
        # BEFORE any delegation or memory injection.
        original_input = user_input
        logger.debug("SDK execute [claude-code]: %s", user_input[:200])

        prompt = self._prepare_prompt(user_input)

        response_text: str = ""
        try:
            # set_current_task INSIDE the try so active_tasks is always
            # decremented by the finally block even if CancelledError hits
            # during the heartbeat HTTP push. Moving it outside the try
            # created a narrow window where cancellation left active_tasks
            # stuck at 1 forever, permanently blocking queue drain. (#2026)
            await set_current_task(self.heartbeat, brief_summary(user_input))
            prompt = await self._inject_memories_if_first_turn(prompt)
            for attempt in range(_MAX_RETRIES):
                options = self._build_options()
                try:
                    result = await self._run_query(prompt=prompt, options=options)
                    if result.session_id:
                        self._session_id = result.session_id
                    response_text = result.text
                    break  # success
                except Exception as exc:
                    formatted = _format_process_error(exc)
                    # #75: CLI subprocess crashes leave our _session_id
                    # referencing a session the next subprocess can't
                    # resume. Without this reset the next attempt would
                    # crash identically even when the underlying cause
                    # was transient, cascading into "crashed once →
                    # crashes forever until container restart." Clear
                    # the session_id so the next attempt (retry or
                    # next user turn) starts fresh.
                    self._reset_session_after_error(exc)
                    if attempt < _MAX_RETRIES - 1 and self._is_retryable(exc):
                        delay = _BASE_RETRY_DELAY_S * (2 ** attempt)
                        logger.warning(
                            "SDK agent [claude-code] transient error (attempt %d/%d), "
                            "retrying in %ds: %s",
                            attempt + 1, _MAX_RETRIES, delay, formatted,
                        )
                        await asyncio.sleep(delay)
                        continue
                    # Non-retryable or exhausted retries. Log exit_code +
                    # stderr explicitly (fixes #66) so operators don't have
                    # to reproduce the crash manually to find out why the
                    # subprocess died.
                    logger.error("SDK agent error [claude-code]: %s", formatted)
                    logger.exception("SDK agent error [claude-code] — full traceback follows")
                    # Detect the specific claude_agent_sdk init-wedge case
                    # so the heartbeat task can flip the workspace to
                    # `degraded`. Match on the lowercased formatted error;
                    # `formatted` is whatever _format_process_error built,
                    # which already includes both the message and the
                    # exception class name.
                    formatted_lc = formatted.lower()
                    for pat in _WEDGE_ERROR_PATTERNS:
                        if pat in formatted_lc:
                            _mark_sdk_wedged(
                                f"claude_agent_sdk wedge: {formatted[:200]} — restart workspace to recover"
                            )
                            break
                    response_text = sanitize_agent_error(exc)
                    break
        finally:
            await set_current_task(self.heartbeat, "")
            await commit_memory(
                f"Conversation: {original_input[:MEMORY_CONTENT_MAX_CHARS]}"
            )
            # Auto-push unpushed commits and open PR (non-blocking, best-effort).
            await auto_push_hook()

        return response_text or _NO_RESPONSE_MSG

    async def cancel(self, context: RequestContext, event_queue: EventQueue):
        """Cooperatively cancel the currently running turn.

        cancel() targets whatever turn is in flight *right now*, not the
        specific turn the caller may have been looking at when they sent
        the cancel request. If turn A has finished and turn B is already
        running under the run lock by the time cancel arrives, turn B is
        the one that gets aborted. This matches how a "stop" button in a
        chat UI typically behaves (stop whatever is running) and is a
        conscious trade-off against per-turn bookkeeping.

        Implementation: the SDK's query() is an async generator; calling
        aclose() raises GeneratorExit inside the running turn and unwinds
        cleanly. We read `self._active_stream` into a local BEFORE calling
        aclose so the reference can't be reassigned by another turn
        mid-cancel. Best-effort — if no stream is active (cancel arrived
        between turns, or the stream has no aclose), this is a no-op.
        """
        stream = self._active_stream
        if stream is None:
            return
        aclose = getattr(stream, "aclose", None)
        if aclose is None:
            return
        try:
            await aclose()
        except Exception:
            logger.exception("SDK cancel: aclose() raised")
