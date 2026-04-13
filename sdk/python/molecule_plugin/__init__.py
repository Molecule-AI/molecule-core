"""Molecule AI plugin SDK — build plugins installable on any Molecule AI workspace.

A plugin is a directory containing a ``plugin.yaml`` manifest and one or more
per-runtime adaptors under ``adapters/<runtime>.py``. The Molecule AI platform
resolves and installs the right adaptor at workspace startup.

This SDK exposes:

* :class:`PluginAdaptor` — the Protocol every adaptor must satisfy.
* :class:`InstallContext`, :class:`InstallResult` — data classes passed to
  ``install()`` and returned from it.
* :class:`AgentskillsAdaptor` — a drop-in adaptor for plugins that ship
  agentskills.io-format skills + Molecule AI's optional rules (covers the
  vast majority of cases).
* :data:`PLUGIN_YAML_SCHEMA` — the manifest schema for validation tooling.

Example: a minimal plugin that's installable on Claude Code and DeepAgents

.. code-block:: text

    my-plugin/
    ├── plugin.yaml
    ├── rules/my-rule.md
    ├── skills/my-skill/SKILL.md
    └── adapters/
        ├── claude_code.py   # `from molecule_plugin import AgentskillsAdaptor as Adaptor`
        └── deepagents.py    # same one-liner

Full docs + cookiecutter template: see ``sdk/python/README.md``.
"""

from __future__ import annotations

# Re-export from the runtime registry so plugins have a single import path.
# The workspace-template package is not pip-installable yet; the SDK duplicates
# the Protocol definition so community authors can build against it without
# depending on the runtime. When a plugin is installed in a workspace, the
# runtime's own ``plugins_registry`` is what actually executes the adaptor —
# these types are structurally compatible (duck-typed via Protocol).

from .protocol import (  # noqa: F401
    InstallContext,
    InstallResult,
    PluginAdaptor,
)
from .builtins import AgentskillsAdaptor, SKIP_ROOT_MD  # noqa: F401
from .manifest import (  # noqa: F401
    PLUGIN_YAML_SCHEMA,
    SKILL_COMPAT_MAX,
    SKILL_DESC_MAX,
    SKILL_NAME_MAX,
    SKILL_NAME_RE,
    parse_skill_md,
    validate_manifest,
    validate_plugin,
    validate_skill,
)
from .protocol import DEFAULT_MEMORY_FILENAME, SKILLS_SUBDIR  # noqa: F401
from .workspace import (  # noqa: F401
    SUPPORTED_RUNTIMES,
    ValidationError,
    validate_workspace_template,
)
from .org import validate_org_template  # noqa: F401
from .channel import (  # noqa: F401
    SUPPORTED_CHANNEL_TYPES,
    validate_channel_config,
    validate_channel_file,
)

__version__ = "0.1.0"

__all__ = [
    "AgentskillsAdaptor",
    "DEFAULT_MEMORY_FILENAME",
    "InstallContext",
    "InstallResult",
    "PLUGIN_YAML_SCHEMA",
    "PluginAdaptor",
    "SKILLS_SUBDIR",
    "SKILL_COMPAT_MAX",
    "SKILL_DESC_MAX",
    "SKILL_NAME_MAX",
    "SKILL_NAME_RE",
    "SKIP_ROOT_MD",
    "SUPPORTED_CHANNEL_TYPES",
    "SUPPORTED_RUNTIMES",
    "ValidationError",
    "parse_skill_md",
    "validate_channel_config",
    "validate_channel_file",
    "validate_manifest",
    "validate_org_template",
    "validate_plugin",
    "validate_skill",
    "validate_workspace_template",
    "__version__",
]
