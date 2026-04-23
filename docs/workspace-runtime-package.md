# Workspace Runtime PyPI Package

## Overview

The shared workspace runtime infrastructure lives in two places:

1. **Source of truth (monorepo):** `workspace/` — this is where all development happens
2. **Published package:** [`molecule-ai-workspace-runtime`](https://pypi.org/project/molecule-ai-workspace-runtime/) on PyPI

## What's in the package

Everything in `workspace/` except adapter-specific code:

- `molecule_runtime/` — all shared `.py` files (main.py, config.py, heartbeat.py, etc.)
- `molecule_runtime/adapters/` — `BaseAdapter`, `AdapterConfig`, `SetupResult`, `shared_runtime`
- `molecule_runtime/builtin_tools/` — delegation, memory, approvals, sandbox, telemetry
- `molecule_runtime/skill_loader/` — skill loading + hot-reload
- `molecule_runtime/plugins_registry/` — plugin discovery and install pipeline
- `molecule_runtime/policies/` — namespace routing policies
- Console script: `molecule-runtime` → `molecule_runtime.main:main_sync`

## Adapter repos

Each of the 8 adapter repos now contains:
- `adapter.py` — runtime-specific `Adapter` class
- `requirements.txt` — `molecule-ai-workspace-runtime>=0.1.0` + adapter deps
- `Dockerfile` — standalone image (no longer extends workspace-template:base)

| Adapter | Repo |
|---------|------|
| claude-code | https://github.com/Molecule-AI/molecule-ai-workspace-template-claude-code |
| langgraph | https://github.com/Molecule-AI/molecule-ai-workspace-template-langgraph |
| crewai | https://github.com/Molecule-AI/molecule-ai-workspace-template-crewai |
| autogen | https://github.com/Molecule-AI/molecule-ai-workspace-template-autogen |
| deepagents | https://github.com/Molecule-AI/molecule-ai-workspace-template-deepagents |
| hermes | https://github.com/Molecule-AI/molecule-ai-workspace-template-hermes |
| gemini-cli | https://github.com/Molecule-AI/molecule-ai-workspace-template-gemini-cli |
| openclaw | https://github.com/Molecule-AI/molecule-ai-workspace-template-openclaw |

## Adapter discovery (ADAPTER_MODULE)

Standalone adapter repos set `ENV ADAPTER_MODULE=adapter` in their Dockerfile.
The runtime's `get_adapter()` checks this env var first:

```python
# In molecule_runtime/adapters/__init__.py
def get_adapter(runtime: str) -> type[BaseAdapter]:
    adapter_module = os.environ.get("ADAPTER_MODULE")
    if adapter_module:
        mod = importlib.import_module(adapter_module)
        return getattr(mod, "Adapter")
    # Fall back to built-in subdirectory scan (monorepo local dev)
    ...
```

## Publishing a new version

```bash
cd workspace-template
# 1. Bump version in pyproject.toml
# 2. Sync to molecule-ai-workspace-runtime repo
# 3. Tag and push — CI publishes to PyPI via PYPI_TOKEN secret
```

Or manually:
```bash
cd workspace-template
python -m build
python -m twine upload dist/*
```

## Writing a new adapter

1. Create a new standalone repo `molecule-ai-workspace-template-<runtime>`
2. Copy `adapter.py` pattern from any existing adapter repo
3. Change imports: `from molecule_runtime.adapters.base import BaseAdapter, AdapterConfig`
4. Create `requirements.txt` with `molecule-ai-workspace-runtime>=0.1.0` + your deps
5. Create `Dockerfile` with `ENV ADAPTER_MODULE=adapter` and `ENTRYPOINT ["molecule-runtime"]`
6. Register the runtime name in the platform's known runtimes list
