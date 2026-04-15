"""Claude Code adapter — wraps the Claude Code CLI as an agent runtime."""

import json
import os
import logging
from pathlib import Path

from adapters.base import BaseAdapter, AdapterConfig
from a2a.server.agent_execution import AgentExecutor

logger = logging.getLogger(__name__)

# Cap one transcript response at 1000 lines so a paranoid client can't OOM
# the workspace by polling /transcript?limit=999999.
_TRANSCRIPT_MAX_LIMIT = 1000


class ClaudeCodeAdapter(BaseAdapter):

    @staticmethod
    def name() -> str:
        return "claude-code"

    @staticmethod
    def display_name() -> str:
        return "Claude Code"

    @staticmethod
    def description() -> str:
        return "Claude Code CLI — full agentic coding with hooks, CLAUDE.md, auto-memory, and MCP support"

    @staticmethod
    def get_config_schema() -> dict:
        return {
            "model": {"type": "string", "description": "Claude model (e.g. sonnet, opus, haiku)", "default": "sonnet"},
            "required_env": {"type": "array", "description": "Required env vars", "default": ["CLAUDE_CODE_OAUTH_TOKEN"]},
            "timeout": {"type": "integer", "description": "Timeout in seconds (0 = no timeout)", "default": 0},
        }

    async def setup(self, config: AdapterConfig) -> None:
        """Install plugins via the per-runtime adaptor registry.

        The legacy claude-code-specific ``inject_plugins()`` override is gone:
        each plugin now ships (or has registered in the platform registry) a
        per-runtime adaptor, and ``BaseAdapter.install_plugins_via_registry``
        routes installs through it. The Claude Code SDK still reads
        ``CLAUDE.md`` and ``/configs/skills/`` natively, and the default
        :class:`AgentskillsAdaptor` writes to both.
        """
        from plugins import load_plugins
        workspace_plugins_dir = os.path.join(config.config_path, "plugins")
        plugins = load_plugins(
            workspace_plugins_dir=workspace_plugins_dir,
            shared_plugins_dir=os.environ.get("PLUGINS_DIR", "/plugins"),
        )
        await self.install_plugins_via_registry(config, plugins)

    async def create_executor(self, config: AdapterConfig) -> AgentExecutor:
        from claude_sdk_executor import ClaudeSDKExecutor

        # Load system prompt if exists
        system_prompt = config.system_prompt
        if not system_prompt:
            prompt_file = os.path.join(config.config_path, "system-prompt.md")
            if os.path.exists(prompt_file):
                with open(prompt_file) as f:
                    system_prompt = f.read()

        # runtime_config may arrive as a dict (from main.py vars(...)) or as a
        # RuntimeConfig dataclass. Read `model` defensively from either shape.
        rc = config.runtime_config
        if isinstance(rc, dict):
            model = rc.get("model") or "sonnet"
        else:
            model = getattr(rc, "model", None) or "sonnet"

        return ClaudeSDKExecutor(
            system_prompt=system_prompt,
            config_path=config.config_path,
            heartbeat=config.heartbeat,
            model=model,
        )

    async def transcript_lines(self, since: int = 0, limit: int = 100) -> dict:
        """Read the live Claude Code session transcript.

        Claude Code writes every session to
        ``$HOME/.claude/projects/<cwd-as-dirname>/<session-uuid>.jsonl`` —
        every line is a JSON event (user/assistant/tool_use/attachment/etc).
        We pick the most-recently-modified .jsonl in the projects dir for
        the agent's working directory, then return ``[since:since+limit]``.

        Returns ``supported: True`` even if no .jsonl exists yet (empty
        ``lines`` + ``cursor=0``) so the canvas can show "agent hasn't
        produced output yet" instead of "feature unavailable".
        """
        limit = max(1, min(limit, _TRANSCRIPT_MAX_LIMIT))
        since = max(0, since)

        # Resolve the projects-dir name. Claude Code maps cwd → dirname by
        # replacing "/" with "-" (so "/configs" → "-configs"). The exact
        # rule lives inside the CLI binary, but the leading-dash + path-
        # without-trailing-slash pattern is stable across versions.
        #
        # Match ClaudeSDKExecutor._resolve_cwd: prefer /workspace if populated,
        # else /configs. Override via CLAUDE_PROJECT_CWD for tests.
        WORKSPACE_MOUNT = "/workspace"
        CONFIG_MOUNT = "/configs"
        cwd_override = os.environ.get("CLAUDE_PROJECT_CWD")
        if cwd_override:
            cwd = cwd_override
        elif os.path.isdir(WORKSPACE_MOUNT) and os.listdir(WORKSPACE_MOUNT):
            cwd = WORKSPACE_MOUNT
        else:
            cwd = CONFIG_MOUNT

        # Normalize: strip trailing slash, replace path separators with "-"
        cwd_norm = cwd.rstrip("/") or "/"
        projdir_name = cwd_norm.replace("/", "-")  # "/configs" → "-configs"

        home = Path(os.environ.get("HOME", "/home/agent"))
        projdir = home / ".claude" / "projects" / projdir_name
        result_base = {
            "runtime": self.name(),
            "supported": True,
            "lines": [],
            "cursor": since,
            "more": False,
            "source": str(projdir),
        }

        if not projdir.is_dir():
            return result_base

        # Pick most-recently-modified .jsonl
        candidates = sorted(projdir.glob("*.jsonl"), key=lambda p: p.stat().st_mtime, reverse=True)
        if not candidates:
            return result_base
        target = candidates[0]
        result_base["source"] = str(target)

        lines = []
        more = False
        try:
            with target.open("r") as f:
                for i, raw in enumerate(f):
                    if i < since:
                        continue
                    if len(lines) >= limit:
                        more = True
                        break
                    raw = raw.strip()
                    if not raw:
                        continue
                    try:
                        lines.append(json.loads(raw))
                    except json.JSONDecodeError:
                        # Skip malformed lines but keep cursor advancing
                        lines.append({"_parse_error": True, "_raw": raw[:200]})
        except OSError as exc:
            logger.warning("transcript_lines: read failed for %s: %s", target, exc)
            return result_base

        result_base["lines"] = lines
        result_base["cursor"] = since + len(lines)
        result_base["more"] = more
        return result_base
