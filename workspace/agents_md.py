"""AGENTS.md auto-generation for Molecule AI workspaces.

Implements the AAIF / Linux Foundation AGENTS.md standard so that peer agents
and orchestration tools can discover this workspace's identity, role, A2A
endpoint, and available tools without reading the full system prompt.

Usage::

    from agents_md import generate_agents_md

    generate_agents_md(config_dir="/configs", output_path="/workspace/AGENTS.md")

The function is called automatically at container startup (see main.py).
"""

import logging
import os
from pathlib import Path

logger = logging.getLogger(__name__)


def generate_agents_md(config_dir: str, output_path: str) -> None:
    """Generate (or regenerate) AGENTS.md from the workspace config.yaml.

    Always overwrites ``output_path`` — no stale-file guard.  Re-calling
    after editing config.yaml produces a fresh file reflecting the changes.

    Args:
        config_dir: Directory containing config.yaml (same convention as
            ``load_config`` in config.py).
        output_path: Absolute path where AGENTS.md will be written.
            The parent directory is expected to exist.
    """
    from config import load_config

    cfg = load_config(config_dir)

    # ── A2A Endpoint ─────────────────────────────────────────────────────────
    # AGENT_URL env var takes priority (production deployments behind a proxy).
    # Otherwise derive from the configured a2a.port (default 8000).
    endpoint = os.environ.get("AGENT_URL") or f"http://localhost:{cfg.a2a.port}/a2a"

    # ── Role ─────────────────────────────────────────────────────────────────
    # Fall back to description when the role field is absent so legacy
    # config.yaml files (without a role key) still produce meaningful output.
    role = cfg.role if cfg.role else cfg.description

    # ── MCP Tools ────────────────────────────────────────────────────────────
    # tools (skill names) + plugins (installed plugin names) form the combined
    # capability surface visible to peer agents.
    all_tools = list(cfg.tools) + list(cfg.plugins)
    if all_tools:
        tools_section = "\n".join(f"- {t}" for t in all_tools)
    else:
        tools_section = "None"

    content = (
        f"# {cfg.name}\n"
        f"\n"
        f"**Role:** {role}\n"
        f"\n"
        f"## Description\n"
        f"{cfg.description}\n"
        f"\n"
        f"## A2A Endpoint\n"
        f"{endpoint}\n"
        f"\n"
        f"## MCP Tools\n"
        f"{tools_section}\n"
    )

    Path(output_path).write_text(content, encoding="utf-8")
    logger.info("Generated AGENTS.md at %s for workspace %r", output_path, cfg.name)
