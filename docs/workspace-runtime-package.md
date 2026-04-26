# Workspace Runtime PyPI Package

## Overview

The shared workspace runtime infrastructure has **one editable source** and
**one published artifact**:

1. **Source of truth (monorepo, editable):** `workspace/` — every runtime
   change lands here. Edit it like any other monorepo code.
2. **Published artifact (PyPI, generated):** [`molecule-ai-workspace-runtime`](https://pypi.org/project/molecule-ai-workspace-runtime/)
   — produced by `.github/workflows/publish-runtime.yml` on every
   `runtime-vX.Y.Z` tag push. Do NOT edit this independently — it gets
   overwritten on every publish.

The legacy sibling repo `molecule-ai-workspace-runtime` (the GitHub repo, as
distinct from the PyPI package) is no longer the source-of-truth and should
be treated as a publish artifact only. It can be archived or used as a
read-only mirror.

## Why this shape

The 8 workspace template repos (claude-code, langgraph, hermes, etc.) each
build their own Docker image and `pip install molecule-ai-workspace-runtime`
from PyPI. PyPI is the right distribution channel — semver, reproducible
builds, no submodule dance per-repo. But the runtime ALSO needs to evolve
in lock-step with the platform's wire protocol (queue shape, A2A metadata,
event payloads). Shipping cross-cutting protocol changes as separate
runtime + platform PRs in two repos creates ordering pain and broken
intermediate states.

The monorepo + auto-publish split gives both: edit cross-cutting changes
in one PR, publish the runtime artifact via a tag.

## What's in the package

Everything in `workspace/*.py` plus the `adapters/`, `builtin_tools/`,
`plugins_registry/`, `policies/`, `skill_loader/` subpackages. Build
artifacts (`Dockerfile`, `*.sh`, `pytest.ini`, `requirements.txt`) are
excluded.

The build script rewrites bare imports so the published package is a
proper Python namespace:

```
# In monorepo workspace/:
from a2a_client import discover_peer
from builtin_tools.memory import store

# In published molecule_runtime/ (auto-rewritten at publish time):
from molecule_runtime.a2a_client import discover_peer
from molecule_runtime.builtin_tools.memory import store
```

The closed allowlist of rewritten module names lives in
`scripts/build_runtime_package.py` (`TOP_LEVEL_MODULES` + `SUBPACKAGES`).
Add a new top-level module to workspace/? Add it to the allowlist in the
same PR.

## Adapter repos

Each of the 8 adapter template repos contains:
- `adapter.py` — runtime-specific `Adapter` class
- `requirements.txt` — `molecule-ai-workspace-runtime>=0.1.X` + adapter deps
- `Dockerfile` — standalone image with `ENV ADAPTER_MODULE=adapter` and
  `ENTRYPOINT ["molecule-runtime"]`

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

Standalone adapter repos set `ENV ADAPTER_MODULE=adapter` in their
Dockerfile. The runtime's `get_adapter()` checks this env var first:

```python
# In molecule_runtime/adapters/__init__.py
def get_adapter(runtime: str) -> type[BaseAdapter]:
    adapter_module = os.environ.get("ADAPTER_MODULE")
    if adapter_module:
        mod = importlib.import_module(adapter_module)
        return getattr(mod, "Adapter")
    raise KeyError(...)
```

## Publishing a new version

```bash
# From any local checkout of monorepo, after merging your runtime change:
git tag runtime-v0.1.6
git push origin runtime-v0.1.6
```

The `publish-runtime` workflow takes over — checks out the tag, runs
`scripts/build_runtime_package.py --version 0.1.6`, builds wheel + sdist,
runs a smoke import to catch broken rewrites, and uploads to PyPI via
the `PYPI_TOKEN` repo secret.

For dev/test releases without tagging, dispatch the workflow manually
with an explicit version (e.g. `0.1.6.dev1` — PEP 440 dev/rc/post forms
are accepted).

After publish, the 8 template repos pick up the new version on their
next `:latest` rebuild. To force-pull immediately, bump the pin in each
template's `requirements.txt`.

## End-to-end CD chain

The full chain from monorepo merge → workspace containers running new code:

```
1. Merge PR with workspace/ changes to main
   ↓
2. .github/workflows/auto-tag-runtime.yml fires
   ↓ reads PR labels (release:major/minor) or defaults to patch
   ↓ pushes runtime-vX.Y.Z tag
   ↓
3. .github/workflows/publish-runtime.yml fires (on the tag)
   ↓ builds wheel via scripts/build_runtime_package.py
   ↓ smoke-imports the wheel
   ↓ uploads to PyPI
   ↓ cascade job fires repository_dispatch (event-type: runtime-published)
   ↓ to all 8 workspace-template-* repos
   ↓
4. Each template's publish-image.yml fires (on repository_dispatch)
   ↓ rebuilds Dockerfile (which pip-installs the new PyPI version)
   ↓ pushes ghcr.io/molecule-ai/workspace-template-<runtime>:latest
   ↓
5. Production hosts run scripts/refresh-workspace-images.sh
   OR an operator hits POST /admin/workspace-images/refresh on the platform
   ↓ docker pull all 8 :latest tags
   ↓ remove + force-recreate any running ws-* containers using a refreshed image
   ↓ canvas re-provisions the workspaces on next interaction
```

Steps 1-4 are fully automated. Step 5 is one-click: a single curl or shell
command. SaaS deployments typically wire step 5 into their normal deploy
pipeline (every release pulls fresh images on every host); local dev fires
it manually after a runtime release lands.

### Required secrets

| Secret | Where | Why |
|---|---|---|
| `PYPI_TOKEN` | molecule-core repo | Twine upload auth (PyPI) |
| `TEMPLATE_DISPATCH_TOKEN` | molecule-core repo | Fine-grained PAT with `actions:write` on the 8 template repos. Without it the `cascade` job warns and exits clean — PyPI still publishes; templates just don't auto-rebuild. |

### Step 5 specifics

**Local dev (compose stack):**
```bash
bash scripts/refresh-workspace-images.sh                  # all runtimes
bash scripts/refresh-workspace-images.sh --runtime claude-code
bash scripts/refresh-workspace-images.sh --no-recreate    # pull only, leave containers
```

**Via platform admin endpoint (any deploy):**
```bash
curl -X POST "$PLATFORM/admin/workspace-images/refresh"
curl -X POST "$PLATFORM/admin/workspace-images/refresh?runtime=claude-code"
curl -X POST "$PLATFORM/admin/workspace-images/refresh?recreate=false"
```

The endpoint pulls + recreates from inside the platform container, so it
needs Docker socket access (the compose stack mounts
`/var/run/docker.sock` already) AND GHCR auth on the host's docker config
(`docker login ghcr.io` once per host). On a fresh host without GHCR auth,
the pull step warns per runtime and the response surfaces the failures.

## Local dev (build the package without publishing)

```bash
python3 scripts/build_runtime_package.py --version 0.1.0-local --out /tmp/runtime-build
cd /tmp/runtime-build
python -m build              # produces dist/*.whl + dist/*.tar.gz
pip install dist/*.whl       # install into a venv to test locally
```

This is the same pipeline CI runs. Use it to validate import-rewrite
correctness before pushing a `runtime-v*` tag.

## Writing a new adapter

1. Create a new standalone repo `molecule-ai-workspace-template-<runtime>`
2. Copy `adapter.py` pattern from any existing adapter repo
3. Change imports: `from molecule_runtime.adapters.base import BaseAdapter, AdapterConfig`
4. Create `requirements.txt` with `molecule-ai-workspace-runtime>=0.1.0` + your deps
5. Create `Dockerfile` with `ENV ADAPTER_MODULE=adapter` and
   `ENTRYPOINT ["molecule-runtime"]`
6. Register the runtime name in the platform's known runtimes list

## Migration note

Prior to this workflow, the runtime was duplicated across monorepo
`workspace/` AND a sibling repo `molecule-ai-workspace-runtime`, with no
sync mechanism. That caused 30+ files to drift between the two trees and
tonight's chat-leak / queued-classification fixes existed only in the
monorepo copy until manually ported.

If you have an old local checkout of `molecule-ai-workspace-runtime`, treat
it as outdated. The monorepo `workspace/` is now authoritative; the PyPI
artifact is rebuilt from it on every `runtime-v*` tag.
