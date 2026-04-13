# molecule_plugin — Python SDK for building Molecule AI plugins

A Molecule AI plugin is a directory that bundles rules, skills, and per-runtime
install adaptors. Any plugin that conforms to this contract is installable
on any Molecule AI workspace whose runtime the plugin supports.

## Quick start

Copy `template/` to a new directory and edit:

```
my-plugin/
├── plugin.yaml              # name, version, runtimes, description
├── rules/my-rule.md         # optional — appended to CLAUDE.md at install
├── skills/my-skill/
│   ├── SKILL.md             # instructions injected into the system prompt
│   └── tools/do_thing.py    # optional LangChain @tool functions
└── adapters/
    ├── claude_code.py       # one-liner: `from molecule_plugin import AgentskillsAdaptor as Adaptor`
    └── deepagents.py        # same
```

Validate:

```python
from molecule_plugin import validate_manifest
errors = validate_manifest("my-plugin/plugin.yaml")
assert not errors, errors
```

## CLI

The SDK ships a CLI for validating Molecule AI artifacts before publishing:

```bash
python -m molecule_plugin validate plugin    my-plugin/
python -m molecule_plugin validate workspace workspace-configs-templates/claude-code-default/
python -m molecule_plugin validate org       org-templates/molecule-dev/
python -m molecule_plugin validate channel   channels.yaml
python -m molecule_plugin validate my-plugin/   # kind defaults to 'plugin'
```

Exit code is 0 when valid, 1 when any errors are found — suitable for CI.
Add `-q` / `--quiet` to suppress success lines and emit only errors.

Programmatic equivalents:

```python
from molecule_plugin import (
    validate_plugin,
    validate_workspace_template,
    validate_org_template,
    validate_channel_file,
    validate_channel_config,
)
```

## Per-runtime adaptors — when to write a custom one

The default `AgentskillsAdaptor` handles the common shape: rules go into
the runtime's memory file (CLAUDE.md), skill dirs go into `/configs/skills/`.
That covers most plugins.

Write a custom adaptor when you need to:

- **Register runtime tools dynamically** — call `ctx.register_tool(name, fn)`.
- **Register DeepAgents sub-agents** — call `ctx.register_subagent(name, spec)`.
- **Write to a non-standard memory file** — call `ctx.append_to_memory(filename, content)`.

Minimum custom adaptor:

```python
# adapters/deepagents.py
from molecule_plugin import InstallContext, InstallResult

class Adaptor:
    def __init__(self, plugin_name: str, runtime: str):
        self.plugin_name, self.runtime = plugin_name, runtime

    async def install(self, ctx: InstallContext) -> InstallResult:
        ctx.register_subagent("my-agent", {"prompt": "...", "tools": [...]})
        return InstallResult(plugin_name=self.plugin_name, runtime=self.runtime, source="plugin")

    async def uninstall(self, ctx: InstallContext) -> None:
        pass
```

## Resolution order (understood by the platform)

For `(plugin_name, runtime)`:

1. **Platform registry** — `workspace-template/plugins_registry/<plugin>/<runtime>.py`
   (curated; set by the Molecule AI team for quality-assured plugins).
2. **Plugin-shipped** — `<plugin_root>/adapters/<runtime>.py` (what this SDK helps you build).
3. **Raw-drop fallback** — copies plugin files into `/configs/plugins/<name>/`
   and surfaces a warning; no tools are wired.

You generally ship for path #2. If your plugin becomes popular enough to be
promoted to "default," the Molecule AI team PRs a copy of your adaptor into
the platform registry (path #1) so it survives upstream breakage.

## Testing locally

The SDK ships `AgentskillsAdaptor` as a standalone, unit-testable class:

```python
import asyncio
from pathlib import Path
from molecule_plugin import AgentskillsAdaptor, InstallContext

ctx = InstallContext(
    configs_dir=Path("/tmp/configs"),
    workspace_id="local",
    runtime="claude_code",
    plugin_root=Path("./my-plugin"),
)
asyncio.run(AgentskillsAdaptor("my-plugin", "claude_code").install(ctx))
# check /tmp/configs/CLAUDE.md, /tmp/configs/skills/
```

## Publishing

A plugin is just a directory. Push it to any Git host. Installation via
`POST /plugins/install {git_url}` is on the roadmap — see the platform's
`PLAN.md` under "Install-from-GitHub-URL flow." Until then, plugins are
bundled into the platform by dropping them into `plugins/` at deploy time.

## Supported runtimes

As of 2026-Q2: `claude_code`, `deepagents`, `langgraph`, `crewai`, `autogen`,
`openclaw`. See the live list with:

```bash
curl $PLATFORM_URL/plugins
```
