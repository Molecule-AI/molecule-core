# Hermes Adapter — Shell Design Spec

**Perspective:** DevOps Engineer + Backend Engineer  
**Status:** Draft — pre-implementation  
**Hermes source:** `NousResearch/hermes-agent` (~61k ⭐)  
**Adapter runtime key:** `hermes`

---

## 1. Files Under `workspace-template/adapters/hermes/`

| File | Purpose |
|------|---------|
| `Dockerfile` | Extends `workspace-template:base`; installs `hermes-agent` Python SDK and its deps via pip at image build time |
| `requirements.txt` | Python package list — at minimum `hermes-agent`; pin to a specific release tag for reproducibility |
| `adapter.py` | `HermesAdapter(BaseAdapter)` — implements `name()`, `display_name()`, `description()`, `get_config_schema()`, `setup()`, `create_executor()`; delegates to `_common_setup()` for plugins/skills/tools |
| `__init__.py` | Exports `Adapter = HermesAdapter` — required by the adapter autodiscovery loader in `workspace-template/adapters/__init__.py` |

### `Dockerfile` sketch (no implementation — shape only)

```dockerfile
FROM workspace-template:base
COPY adapters/hermes/requirements.txt /tmp/hermes-requirements.txt
RUN pip install --no-cache-dir -r /tmp/hermes-requirements.txt
```

### `adapter.py` shape

```python
class HermesAdapter(BaseAdapter):
    @staticmethod
    def name() -> str:
        return "hermes"

    async def setup(self, config: AdapterConfig) -> None:
        # validate NOUS_API_KEY or OPENROUTER_API_KEY is set
        # call self._common_setup(config) for plugins/skills/tools
        ...

    async def create_executor(self, config: AdapterConfig) -> AgentExecutor:
        # wrap Hermes SDK session as an A2A AgentExecutor
        ...
```

---

## 2. Platform-Side Changes

### `platform/internal/provisioner/provisioner.go` — `RuntimeImages` map

Add one entry to the existing map:

```go
var RuntimeImages = map[string]string{
    // ... existing entries ...
    "hermes": "workspace-template:hermes",   // ← ADD THIS
}
```

No other platform Go changes are required for the minimal adapter shell. The `runtime` column in the `workspaces` table is a free-form string; no enum migration needed.

### `workspace-template/build-all.sh`

Add `hermes` to the adapter build loop so `build-all.sh` (and the `build-all.sh claude-code`-style single-runtime path) includes it:

```bash
ADAPTERS=(langgraph claude_code openclaw deepagents crewai autogen hermes)
```

---

## 3. Required Environment Variables

| Name | Required | Description |
|------|----------|-------------|
| `NOUS_API_KEY` | Required (unless `OPENROUTER_API_KEY` set) | Nous Research Portal API key — primary model provider for Hermes; obtain from `nousresearch.com` |
| `OPENROUTER_API_KEY` | Optional | Fallback provider; lets operators use any Hermes-supported model via OpenRouter instead of Nous Portal |
| `HERMES_MODEL` | Optional | Model identifier (e.g. `nous-hermes-3`, `openrouter:anthropic/claude-sonnet-4-5`); adapter defaults to `nous-hermes-3` if unset |
| `HERMES_SKILLS_DIR` | Optional | Path inside the container where Hermes looks for skills; defaults to `/configs/skills` — consistent with the Claude Code and DeepAgents adapters |

**Note:** `NOUS_API_KEY` and `OPENROUTER_API_KEY` must be set as workspace secrets via `POST /workspaces/:id/secrets`, not baked into the image. At least one of the two must be present at container start; `setup()` should `raise RuntimeError` early with a clear message if both are absent.

---

## 4. Smallest Viable Adapter — Scope Constraints

This spec covers the **shell only** — the minimum to make a Hermes workspace provision, boot, and accept A2A messages:

- No Hermes learning loop (skill self-improvement) in v1 — that requires persistent storage writes outside `/configs`; defer to a follow-up PR.
- No multi-messenger gateway integration — Hermes's Telegram/Discord/Slack channels are separate from Molecule AI's `/channels` feature; map these later via the channels adapter.
- No FTS5 memory backend — use Molecule AI's existing `commit_memory` / `search_memory` built-in tools for v1; Hermes-native memory can be layered in a subsequent PR.
- The executor wraps one Hermes agent session per workspace, matching the 1:1 workspace→agent model used by all other adapters.
