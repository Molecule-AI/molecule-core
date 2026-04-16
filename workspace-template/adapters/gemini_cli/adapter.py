"""Gemini CLI adapter — wraps Google's Gemini CLI as an agent runtime.

Gemini CLI (github.com/google-gemini/gemini-cli, ~101k stars, Apache 2.0)
is structurally identical to the Claude Code adapter: a single-agent agentic
CLI with file/shell tools, MCP support, and a ReAct loop — backed by Gemini
instead of Claude.

Key differences from claude-code:
- Auth: GEMINI_API_KEY env var (no OAuth token needed)
- Memory file: GEMINI.md (equivalent of Claude Code's CLAUDE.md)
- MCP config: ~/.gemini/settings.json (not via --mcp-config flag)
- Executor: CLIAgentExecutor (no Python SDK; uses gemini CLI subprocess)
"""

import json
import logging
import os
import sys
from pathlib import Path

from a2a.server.agent_execution import AgentExecutor

from adapters.base import BaseAdapter, AdapterConfig

logger = logging.getLogger(__name__)


class GeminiCLIAdapter(BaseAdapter):

    @staticmethod
    def name() -> str:
        return "gemini-cli"

    @staticmethod
    def display_name() -> str:
        return "Gemini CLI"

    @staticmethod
    def description() -> str:
        return (
            "Google Gemini CLI — agentic coding with file/shell tools, "
            "MCP support, and a ReAct loop backed by Gemini models"
        )

    @staticmethod
    def get_config_schema() -> dict:
        return {
            "model": {
                "type": "string",
                "description": "Gemini model (e.g. gemini-2.5-pro, gemini-2.5-flash)",
                "default": "gemini-2.5-pro",
            },
            "required_env": {
                "type": "array",
                "description": "Required env vars",
                "default": ["GEMINI_API_KEY"],
            },
            "timeout": {
                "type": "integer",
                "description": "Timeout in seconds (0 = no timeout)",
                "default": 0,
            },
        }

    def memory_filename(self) -> str:
        """Gemini CLI reads GEMINI.md as its persistent context file."""
        return "GEMINI.md"

    async def setup(self, config: AdapterConfig) -> None:
        """Wire MCP server into ~/.gemini/settings.json and seed GEMINI.md.

        Gemini CLI does not accept an --mcp-config flag; instead, MCP servers
        are declared in ~/.gemini/settings.json under the "mcpServers" key.
        This method merges the A2A MCP server into that file, preserving any
        existing keys (e.g. user's own MCP tools).

        Also seeds GEMINI.md from system-prompt.md if GEMINI.md is absent,
        so the agent has role context on first boot.
        """
        from executor_helpers import get_mcp_server_path

        # -- MCP wiring --------------------------------------------------
        gemini_dir = Path.home() / ".gemini"
        gemini_dir.mkdir(parents=True, exist_ok=True)
        settings_path = gemini_dir / "settings.json"

        settings: dict = {}
        if settings_path.exists():
            try:
                settings = json.loads(settings_path.read_text())
            except Exception as exc:
                logger.warning("gemini-cli: could not parse %s: %s", settings_path, exc)
                settings = {}

        settings.setdefault("mcpServers", {})
        settings["mcpServers"]["a2a"] = {
            "command": sys.executable,
            "args": [get_mcp_server_path()],
        }

        try:
            settings_path.write_text(json.dumps(settings, indent=2))
            logger.info("gemini-cli: wrote MCP config to %s", settings_path)
        except OSError as exc:
            logger.warning("gemini-cli: could not write %s: %s", settings_path, exc)

        # -- GEMINI.md seed ----------------------------------------------
        gemini_md = Path(config.config_path) / "GEMINI.md"
        system_prompt_file = Path(config.config_path) / "system-prompt.md"
        if not gemini_md.exists() and system_prompt_file.exists():
            try:
                gemini_md.write_text(system_prompt_file.read_text())
                logger.info("gemini-cli: seeded GEMINI.md from system-prompt.md")
            except OSError as exc:
                logger.warning("gemini-cli: could not seed GEMINI.md: %s", exc)

    async def create_executor(self, config: AdapterConfig) -> AgentExecutor:
        from cli_executor import CLIAgentExecutor
        from config import RuntimeConfig

        rc = config.runtime_config
        if isinstance(rc, dict):
            model = rc.get("model") or "gemini-2.5-pro"
            timeout = int(rc.get("timeout") or 0)
        else:
            model = getattr(rc, "model", None) or "gemini-2.5-pro"
            timeout = int(getattr(rc, "timeout", None) or 0)

        runtime_config = RuntimeConfig(
            model=model,
            timeout=timeout,
            required_env=["GEMINI_API_KEY"],
        )

        return CLIAgentExecutor(
            runtime="gemini-cli",
            runtime_config=runtime_config,
            system_prompt=config.system_prompt,
            config_path=config.config_path,
            heartbeat=config.heartbeat,
        )
