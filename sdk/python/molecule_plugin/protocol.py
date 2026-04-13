"""Adaptor protocol — kept in sync with workspace-template/plugins_registry/protocol.py.

SDK authors depend only on this module so their plugin repos don't need to
pull in the full workspace-template package. At runtime the platform's own
``plugins_registry`` loads the adaptor; the two ``InstallContext`` shapes are
structurally identical so the Protocol check passes.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Callable, Protocol, runtime_checkable


# Kept in sync with workspace-template/plugins_registry/protocol.py.
DEFAULT_MEMORY_FILENAME = "CLAUDE.md"
SKILLS_SUBDIR = "skills"


@dataclass
class InstallContext:
    """Hooks + state passed to every PluginAdaptor.install() call."""

    configs_dir: Path
    """Workspace's /configs directory (where memory file, plugins/, skills/ live)."""

    workspace_id: str
    """Workspace UUID — useful for per-workspace state or logging."""

    runtime: str
    """Runtime identifier (``claude_code``, ``deepagents``, …)."""

    plugin_root: Path
    """Path to the plugin's directory (where plugin.yaml + content lives)."""

    memory_filename: str = DEFAULT_MEMORY_FILENAME
    """Runtime's long-lived memory file. Populated by the runtime's
    :meth:`BaseAdapter.memory_filename`; adaptors pass this to
    :attr:`append_to_memory` rather than hardcoding a filename."""

    register_tool: Callable[[str, Callable[..., Any]], None] = field(
        default=lambda name, fn: None
    )
    """Register a callable as a runtime tool. No-op on runtimes without
    a dynamic tool registry — those runtimes pick tools up at startup
    via filesystem scan instead."""

    register_subagent: Callable[[str, dict[str, Any]], None] = field(
        default=lambda name, spec: None
    )
    """Register a sub-agent specification (DeepAgents-only). No-op elsewhere."""

    append_to_memory: Callable[[str, str], None] = field(
        default=lambda filename, content: None
    )
    """Append text to a runtime memory file. The default no-op lets
    adaptors run in test harnesses without a real workspace filesystem."""

    logger: logging.Logger = field(default_factory=lambda: logging.getLogger(__name__))


@dataclass
class InstallResult:
    plugin_name: str
    runtime: str
    source: str
    files_written: list[str] = field(default_factory=list)
    tools_registered: list[str] = field(default_factory=list)
    subagents_registered: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)


@runtime_checkable
class PluginAdaptor(Protocol):
    plugin_name: str
    runtime: str

    async def install(self, ctx: InstallContext) -> InstallResult:
        ...

    async def uninstall(self, ctx: InstallContext) -> None:
        ...
