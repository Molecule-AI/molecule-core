"""Claude Code adapter — wraps the Claude Code CLI as an agent runtime."""

import os
import logging

from adapters.base import BaseAdapter, AdapterConfig
from a2a.server.agent_execution import AgentExecutor

logger = logging.getLogger(__name__)


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
