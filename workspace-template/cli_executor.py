"""CLI-based agent executor for A2A protocol.

Supports CLI agents that accept a prompt and output a response:
- OpenAI Codex: codex --print -p "..."
- Ollama: ollama run <model> "..."
- Custom: any command that reads stdin or accepts -p

NOTE: the `claude-code` runtime no longer routes here. It uses
ClaudeSDKExecutor (see claude_sdk_executor.py) which wraps the
claude-agent-sdk Python package. This executor is reserved for CLI-only
runtimes that don't yet have a programmatic SDK integration.

The runtime is selected via config.yaml:
  runtime: codex | ollama | custom
  runtime_config:
    command: "codex"        # for custom
    args: ["--extra-flag"]  # additional CLI args
    auth_token_env: "OPENAI_API_KEY"
    auth_token_file: ".auth-token"
    timeout: 300
    model: "sonnet"
"""

import asyncio
import atexit
import json
import logging
import os
import shlex
import shutil
import sys
import tempfile
from pathlib import Path

from a2a.server.agent_execution import AgentExecutor, RequestContext
from a2a.server.events import EventQueue
from a2a.utils import new_agent_text_message

from config import RuntimeConfig
from executor_helpers import (
    CONFIG_MOUNT,
    MEMORY_CONTENT_MAX_CHARS,
    WORKSPACE_MOUNT,
    brief_summary,
    classify_subprocess_error,
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

logger = logging.getLogger(__name__)


# Built-in runtime presets.
# The `claude-code` runtime uses ClaudeSDKExecutor (claude_sdk_executor.py)
# and intentionally has no entry here.
RUNTIME_PRESETS: dict[str, dict] = {
    "codex": {
        "command": "codex",
        "base_args": ["--print", "--dangerously-skip-permissions"],
        "prompt_flag": "-p",
        "model_flag": "--model",
        "system_prompt_flag": "--system-prompt",
        "auth_pattern": "env",  # uses OPENAI_API_KEY env var
        "default_auth_env": "OPENAI_API_KEY",
        "default_auth_file": "",
    },
    "ollama": {
        "command": "ollama",
        "base_args": ["run"],
        "prompt_flag": None,  # prompt is positional
        "model_flag": None,   # model is positional after "run"
        "system_prompt_flag": "--system",
        "auth_pattern": None,  # no auth needed
        "default_auth_env": "",
        "default_auth_file": "",
    },
}


class CLIAgentExecutor(AgentExecutor):
    """Executes agent tasks by invoking a CLI tool.

    Works with any CLI agent that accepts a prompt and outputs text.
    """

    def __init__(
        self,
        runtime: str,
        runtime_config: RuntimeConfig,
        system_prompt: str | None = None,
        config_path: str = "/configs",
        heartbeat: "HeartbeatLoop | None" = None,
    ):
        if runtime == "claude-code":
            # Defensive — the adapter should never construct a CLI executor
            # for claude-code. Fail loud rather than silently falling back.
            raise ValueError(
                "claude-code runtime is served by ClaudeSDKExecutor, not "
                "CLIAgentExecutor. Check adapters/claude_code/adapter.py."
            )
        self.runtime = runtime
        self.config = runtime_config
        self.system_prompt = system_prompt
        self.config_path = config_path
        self._heartbeat = heartbeat

        # Resolve preset or use custom
        if runtime in RUNTIME_PRESETS:
            self.preset = RUNTIME_PRESETS[runtime]
        elif runtime == "custom":
            self.preset = {
                "command": runtime_config.command,
                "base_args": [],  # args go in config.args, appended at end
                "prompt_flag": "-p",
                "model_flag": None,
                "system_prompt_flag": None,
                "auth_pattern": None,
                "default_auth_env": "",
                "default_auth_file": "",
            }
        else:
            raise ValueError(f"Unknown runtime: {runtime}. Use: {', '.join(RUNTIME_PRESETS.keys())}, custom")

        # Resolve auth token
        self._auth_token = self._resolve_auth_token()
        self._auth_helper_path: str | None = None
        self._temp_files: list[str] = []  # Track temp files for cleanup

        if self._auth_token and self.preset.get("auth_pattern") == "apiKeyHelper":
            self._auth_helper_path = self._create_auth_helper(self._auth_token)

        # Create MCP config once (reuse across invocations)
        self._mcp_config_path: str | None = None
        if self.preset.get("auth_pattern") in ("apiKeyHelper", "env"):
            mcp_config = json.dumps({
                "mcpServers": {
                    "a2a": {"command": sys.executable, "args": [get_mcp_server_path()]}
                }
            })
            fd, self._mcp_config_path = tempfile.mkstemp(suffix=".json", prefix="a2a-mcp-")
            self._temp_files.append(self._mcp_config_path)  # Track immediately
            os.close(fd)
            with open(self._mcp_config_path, "w") as f:
                f.write(mcp_config)

        # Register cleanup for reliable temp file removal (atexit is more reliable than __del__)
        atexit.register(self._cleanup_temp_files)

        # Verify command exists
        cmd = self.config.command or self.preset["command"]
        if not shutil.which(cmd):
            logger.warning(f"CLI command '{cmd}' not found in PATH")

    def _resolve_auth_token(self) -> str | None:
        """Resolve auth token from env var or file.

        Resolution order:
        1. required_env — first entry that exists in the environment
        2. auth_token_env (deprecated) — explicit env var name
        3. Preset default_auth_env — adapter-declared fallback
        4. auth_token_file (deprecated) — file on disk
        5. Preset default_auth_file — adapter-declared file fallback
        """
        # 1. New path: required_env (first match wins)
        for env_name in (self.config.required_env or []):
            token = os.environ.get(env_name)
            if token:
                return token

        # 2. Legacy: explicit env var from config
        env_name = self.config.auth_token_env or self.preset.get("default_auth_env", "")
        if env_name:
            token = os.environ.get(env_name)
            if token:
                return token

        # 3. Legacy: token file from config
        file_name = self.config.auth_token_file or self.preset.get("default_auth_file", "")
        if file_name:
            token_path = Path(self.config_path) / file_name
            if token_path.exists():
                return token_path.read_text().strip()

        return None

    def _create_auth_helper(self, token: str) -> str:
        """Create a shell script that outputs the auth token (for apiKeyHelper pattern)."""
        fd, helper_path = tempfile.mkstemp(suffix=".sh", prefix="agent-auth-")
        self._temp_files.append(helper_path)  # Track immediately before any exception can leak
        os.close(fd)
        with open(helper_path, "w") as f:
            f.write(f"#!/bin/sh\necho {shlex.quote(token)}\n")
        os.chmod(helper_path, 0o700)
        return helper_path

    def _build_command(self, message: str) -> list[str]:
        """Build the full CLI command from preset + config + message."""
        cmd = self.config.command or self.preset["command"]
        args = list(self.preset.get("base_args", []))

        # Model
        model = self.config.model or None
        model_flag = self.preset.get("model_flag")
        if model and model_flag:
            args.extend([model_flag, model])
        elif model and self.runtime == "ollama":
            # Ollama: model is positional after "run"
            args.append(model)

        # System prompt (+ A2A instructions). The remaining CLI runtimes don't
        # support session resume, so we inject the system prompt on every call.
        system_prompt = get_system_prompt(self.config_path, fallback=self.system_prompt) or ""
        mcp_capable = self.preset.get("auth_pattern") in ("apiKeyHelper", "env")
        a2a_instructions = get_a2a_instructions(mcp=mcp_capable)
        if a2a_instructions:
            system_prompt = (
                f"{system_prompt}\n\n{a2a_instructions}" if system_prompt else a2a_instructions
            )
        system_flag = self.preset.get("system_prompt_flag")
        if system_prompt and system_flag:
            args.extend([system_flag, system_prompt])

        # Auth (apiKeyHelper pattern — reserved for future CLI runtimes)
        if self._auth_helper_path and self.preset.get("auth_pattern") == "apiKeyHelper":
            settings = json.dumps({"apiKeyHelper": self._auth_helper_path})
            args.extend(["--settings", settings])

        # A2A MCP server — inject for MCP-compatible runtimes (created once in __init__)
        if self._mcp_config_path:
            args.extend(["--mcp-config", self._mcp_config_path])

        # Extra args from config (before prompt so flags are parsed correctly)
        args.extend(self.config.args)

        # Prompt (must be last — some CLIs treat final arg as the prompt)
        prompt_flag = self.preset.get("prompt_flag")
        if prompt_flag:
            args.extend([prompt_flag, message])
        else:
            # Positional prompt (ollama)
            args.append(message)

        return [cmd] + args

    async def execute(self, context: RequestContext, event_queue: EventQueue):
        """Execute a task by invoking the CLI agent."""
        user_input = extract_message_text(context.message)
        if not user_input:
            await event_queue.enqueue_event(
                new_agent_text_message("Error: message contained no text content.")
            )
            return

        # Keep a clean copy of the user's actual message for memory BEFORE any
        # delegation or memory injection happens.
        original_input = user_input

        # Show current task on canvas — extract a brief one-line summary
        await set_current_task(self._heartbeat, brief_summary(user_input))

        logger.debug("CLI execute [%s]: %s", self.runtime, user_input[:200])

        # Inject delegation results that arrived since last message
        delegation_context = read_delegation_results()
        if delegation_context:
            user_input = f"[Delegation results received while you were idle]\n{delegation_context}\n\n[New message]\n{user_input}"

        # Auto-recall: inject prior memories into every prompt. (The CLI
        # runtimes don't keep a session, so there's no "first turn" concept.)
        memories = await recall_memories()
        if memories:
            user_input = f"[Prior context from memory]\n{memories}\n\n{user_input}"

        try:
            await self._run_cli(user_input, event_queue)
        finally:
            await set_current_task(self._heartbeat, "")
            # Auto-commit: save the original user request (not the memory-injected version)
            await commit_memory(
                f"Conversation: {original_input[:MEMORY_CONTENT_MAX_CHARS]}"
            )

    async def _run_cli(self, user_input: str, event_queue: EventQueue):
        """Run the CLI subprocess and enqueue the result."""
        cmd = self._build_command(user_input)
        timeout = self.config.timeout or None  # None = no timeout (wait until agent finishes)
        max_retries = 3
        base_delay = 5  # seconds

        # Build env — pass through auth env var if using env pattern
        env = dict(os.environ)
        if self._auth_token and self.preset.get("auth_pattern") == "env":
            # Use first required_env entry, or fall back to legacy auth_token_env
            auth_env = (self.config.required_env or [None])[0] if self.config.required_env else None
            auth_env = auth_env or self.config.auth_token_env or self.preset.get("default_auth_env", "")
            if auth_env:
                env[auth_env] = self._auth_token

        for attempt in range(max_retries):
            proc = None
            try:
                # Run in /workspace if it exists and has content (cloned repo),
                # otherwise /configs (agent config files)
                cwd = (
                    WORKSPACE_MOUNT
                    if os.path.isdir(WORKSPACE_MOUNT) and os.listdir(WORKSPACE_MOUNT)
                    else CONFIG_MOUNT
                )

                proc = await asyncio.create_subprocess_exec(
                    *cmd,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE,
                    env=env,
                    cwd=cwd,
                )
                if timeout:
                    stdout, stderr = await asyncio.wait_for(
                        proc.communicate(), timeout=timeout
                    )
                else:
                    stdout, stderr = await proc.communicate()

                stdout_text = stdout.decode().strip()
                stderr_text = stderr.decode().strip()

                if proc.returncode != 0:
                    logger.error("CLI agent [%s] exit=%d stdout=%s stderr=%s",
                                 self.runtime, proc.returncode,
                                 stdout_text[:200] if stdout_text else "(empty)",
                                 stderr_text[:500] if stderr_text else "(empty)")

                if proc.returncode == 0 or stdout_text:
                    # Success, or non-zero exit but produced output (some CLIs exit 1 with valid output)
                    result = stdout_text
                    if result:
                        await event_queue.enqueue_event(
                            new_agent_text_message(result)
                        )
                        return
                    else:
                        # Empty response — likely rate limited, retry with backoff
                        if attempt < max_retries - 1:
                            delay = base_delay * (2 ** attempt)
                            logger.warning("CLI agent [%s] returned empty (attempt %d/%d), retrying in %ds",
                                           self.runtime, attempt + 1, max_retries, delay)
                            await asyncio.sleep(delay)
                            continue
                        await event_queue.enqueue_event(
                            new_agent_text_message("(no response generated after retries)")
                        )
                        return
                else:
                    error_msg = stderr_text or f"Exit code {proc.returncode}"
                    # Classify once — used both for retry policy and the
                    # sanitized user-facing error message.
                    category = classify_subprocess_error(error_msg, proc.returncode)
                    if category in ("rate_limited", "session_error", "auth_failed") \
                            and attempt < max_retries - 1:
                        delay = base_delay * (2 ** attempt)
                        logger.warning(
                            "CLI agent [%s] %s (attempt %d/%d), retrying in %ds",
                            self.runtime, category, attempt + 1, max_retries, delay,
                        )
                        await asyncio.sleep(delay)
                        continue

                    # Log the full stderr (may contain paths/tokens); surface
                    # only the sanitized category to the user.
                    logger.error("CLI agent error [%s]: %s", self.runtime, error_msg[:500])
                    await event_queue.enqueue_event(
                        new_agent_text_message(sanitize_agent_error(category=category))
                    )
                    return

            except asyncio.TimeoutError:
                logger.error("CLI agent timeout [%s] after %ds", self.runtime, timeout)
                if proc:
                    # Kill and reap the process to prevent zombies
                    try:
                        proc.kill()
                    except ProcessLookupError:
                        pass  # already exited
                    except Exception as kill_err:
                        logger.warning("CLI kill error: %s", kill_err)
                    # Always await wait() to reap zombie, even if kill failed
                    try:
                        await asyncio.wait_for(proc.wait(), timeout=5)
                    except asyncio.TimeoutError:
                        logger.error("CLI agent: proc.wait() also timed out — possible zombie")
                    except Exception as wait_err:
                        logger.warning("CLI wait error: %s", wait_err)
                await event_queue.enqueue_event(
                    new_agent_text_message(sanitize_agent_error(category="timeout"))
                )
                return
            except Exception as exc:
                logger.exception("CLI agent exception [%s]", self.runtime)
                await event_queue.enqueue_event(
                    new_agent_text_message(sanitize_agent_error(exc))
                )
                return

    def _cleanup_temp_files(self):  # pragma: no cover
        """Clean up temp files. Called via atexit for reliable cleanup."""
        for f in self._temp_files:
            try:
                os.unlink(f)
            except OSError:
                pass
        if self._auth_helper_path:
            try:
                os.unlink(self._auth_helper_path)
            except OSError:
                pass

    def __del__(self):  # pragma: no cover
        """Clean up temp files (fallback — prefer atexit-registered _cleanup_temp_files)."""
        for f in getattr(self, "_temp_files", []):
            try:
                os.unlink(f)
            except OSError:
                pass
        if getattr(self, "_auth_helper_path", None):
            try:
                os.unlink(self._auth_helper_path)
            except OSError:
                pass

    async def cancel(self, context: RequestContext, event_queue: EventQueue):  # pragma: no cover
        """Cancel a running task."""
        pass
