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
from collections.abc import AsyncIterator
from dataclasses import dataclass
from typing import TYPE_CHECKING, Any

import claude_agent_sdk as sdk

from a2a.server.agent_execution import AgentExecutor, RequestContext
from a2a.server.events import EventQueue
from a2a.utils import new_agent_text_message

from executor_helpers import (
    CONFIG_MOUNT,
    MEMORY_CONTENT_MAX_CHARS,
    WORKSPACE_MOUNT,
    brief_summary,
    commit_memory,
    extract_message_text,
    get_a2a_instructions,
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


def _format_process_error(exc: BaseException) -> str:
    """Render a Claude-SDK ProcessError (or any ClaudeSDKError) with its full
    captured context — exit code, stderr, exception type. Plain strings for
    non-SDK exceptions fall back to str(exc).

    Bounded at _PROCESS_ERROR_STDERR_MAX_CHARS so a runaway CLI can't spam
    the log. Used by the executor's error path (fixes #66 — the SDK's
    ProcessError carries `.stderr`/`.exit_code` attributes that the previous
    code silently discarded, leaving every CLI crash with an identical
    "Check stderr output for details" message in the workspace log).
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
        """Compose system prompt from file + A2A delegation instructions."""
        base = get_system_prompt(self.config_path, fallback=self.system_prompt)
        a2a = get_a2a_instructions(mcp=True)
        if base and a2a:
            return f"{base}\n\n{a2a}"
        return base or a2a

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

    def _build_options(self) -> Any:
        """Build ClaudeAgentOptions.

        No allowed_tools allowlist — bypassPermissions grants full access,
        matching the old CLI `--dangerously-skip-permissions` so Claude can
        use every built-in tool (Task, TodoWrite, NotebookEdit, BashOutput/
        KillShell, ExitPlanMode, etc.) plus all MCP tools.

        The MCP server launcher uses `sys.executable` so tests and alternate
        virtual-env layouts don't depend on a `python3` shim being on PATH.
        """
        mcp_servers = {
            "a2a": {
                "command": sys.executable,
                "args": [get_mcp_server_path()],
            }
        }
        return sdk.ClaudeAgentOptions(
            model=self.model,
            permission_mode="bypassPermissions",
            cwd=self._resolve_cwd(),
            mcp_servers=mcp_servers,
            system_prompt=self._build_system_prompt(),
            resume=self._session_id,
        )

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
                elif isinstance(message, sdk.ResultMessage):
                    sid = getattr(message, "session_id", None)
                    if sid:
                        session_id = sid
                    result_text = getattr(message, "result", None)
        finally:
            self._active_stream = None
        text = result_text if result_text is not None else "".join(assistant_chunks)
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
        if not user_input:
            await event_queue.enqueue_event(new_agent_text_message(_NO_TEXT_MSG))
            return

        async with self._run_lock:
            response_text = await self._execute_locked(user_input)

        # Enqueue outside the lock so the next queued turn can start
        # preparing its prompt while this turn's response ships. Event
        # ordering is preserved per-queue by the A2A server, so no races.
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
        await set_current_task(self.heartbeat, brief_summary(user_input))
        logger.debug("SDK execute [claude-code]: %s", user_input[:200])

        prompt = self._prepare_prompt(user_input)
        prompt = await self._inject_memories_if_first_turn(prompt)

        response_text: str = ""
        try:
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
                    response_text = sanitize_agent_error(exc)
                    break
        finally:
            await set_current_task(self.heartbeat, "")
            await commit_memory(
                f"Conversation: {original_input[:MEMORY_CONTENT_MAX_CHARS]}"
            )

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
