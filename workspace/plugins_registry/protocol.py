"""Protocol + context types for per-runtime plugin adaptors.

Each plugin ships (or has registered for it) a per-runtime adaptor implementing
``PluginAdaptor``. The platform resolves the adaptor for ``(plugin_name, runtime)``
via :func:`plugins_registry.resolve` and calls ``install(ctx)`` to wire the
plugin into a workspace.

The :class:`InstallContext` deliberately gives adaptors ONLY the hooks they
need (``register_tool``, ``register_subagent``, ``append_to_memory``) — it
does not leak runtime internals. This keeps adaptors thin and lets the
workspace runtime adapter (claude_code, deepagents, …) own its own state.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Callable, Protocol, runtime_checkable


# Default filename for the runtime's long-lived memory file. Claude Code
# and DeepAgents both read CLAUDE.md natively; other runtimes override via
# BaseAdapter.memory_filename() and that value flows through
# InstallContext.memory_filename so adaptors don't hardcode the name.
DEFAULT_MEMORY_FILENAME = "CLAUDE.md"

# Subdirectory under /configs where skills get installed.
SKILLS_SUBDIR = "skills"


@dataclass
class InstallContext:
    """Hooks + state passed to every PluginAdaptor.install() call.

    Adaptors should treat unknown verbs as no-ops on runtimes that don't
    support them (e.g. ``register_subagent`` is a no-op on Claude Code).
    """

    configs_dir: Path
    """Workspace's /configs directory (where CLAUDE.md, plugins/, skills/ live)."""

    workspace_id: str
    """Workspace UUID — useful for per-workspace state or logging."""

    runtime: str
    """Runtime identifier (``claude_code``, ``deepagents``, …)."""

    plugin_root: Path
    """Path to the plugin's directory (where plugin.yaml + content lives)."""

    memory_filename: str = DEFAULT_MEMORY_FILENAME
    """Runtime's long-lived memory file (populated from
    :meth:`BaseAdapter.memory_filename`). Adaptors pass this to
    :attr:`append_to_memory` instead of hardcoding a filename so runtimes
    with non-standard memory files (e.g. ``AGENTS.md``) work unchanged."""

    register_tool: Callable[[str, Callable[..., Any]], None] = field(
        default=lambda name, fn: None
    )
    """Register a callable as a runtime tool. No-op on runtimes without a
    dynamic tool registry — those runtimes pick tools up at startup via
    filesystem scan instead."""

    register_subagent: Callable[[str, dict[str, Any]], None] = field(
        default=lambda name, spec: None
    )
    """Register a sub-agent specification (DeepAgents-only). No-op elsewhere."""

    append_to_memory: Callable[[str, str], None] = field(
        default=lambda filename, content: None
    )
    """Append text to a runtime memory file (e.g. CLAUDE.md). The default
    no-op lets adaptors run in test harnesses that don't have a real
    workspace filesystem."""

    logger: logging.Logger = field(default_factory=lambda: logging.getLogger(__name__))


@dataclass
class InstallResult:
    """Outcome of a PluginAdaptor.install() call."""

    plugin_name: str
    runtime: str
    source: str  # "registry" | "plugin" | "raw_drop"
    files_written: list[str] = field(default_factory=list)
    tools_registered: list[str] = field(default_factory=list)
    subagents_registered: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)


@runtime_checkable
class PluginAdaptor(Protocol):
    """Contract every per-runtime adaptor must implement."""

    plugin_name: str
    runtime: str

    async def install(self, ctx: InstallContext) -> InstallResult:
        ...

    async def uninstall(self, ctx: InstallContext) -> None:
        ...
