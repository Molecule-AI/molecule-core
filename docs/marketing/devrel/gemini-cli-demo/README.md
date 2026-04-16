# Gemini CLI Runtime Adapter — Live Demo

> **Feature:** [`feat(adapters): add gemini-cli runtime adapter`](https://github.com/Molecule-AI/molecule-core/pull/379)  
> **Adapter path:** `workspace-template/adapters/gemini_cli/`  
> **Runtime key:** `gemini-cli`

This demo provisions a Gemini CLI workspace on Molecule AI, sends it a task via
the A2A proxy, and prints the result — all in about 60 seconds.

---

## What you'll need

| Requirement | Where to get it |
|-------------|----------------|
| Running Molecule AI platform | See [Quickstart](../../docs/quickstart.md) |
| Admin bearer token | Printed on first `go run ./cmd/server` startup |
| `GEMINI_API_KEY` | [Google AI Studio → Get API key](https://aistudio.google.com/apikey) |
| Python ≥ 3.11 + pip | `python --version` |
| `@google/gemini-cli` Docker image built | `bash workspace-template/build-all.sh gemini-cli` |

---

## Step-by-step walkthrough

### 1 — Build the adapter image (one-time)

```bash
# From the repo root
bash workspace-template/build-all.sh gemini-cli
```

Expected output: `Successfully tagged workspace-template:gemini-cli`

This installs `@google/gemini-cli@0.38.1` globally inside the container and
wires the A2A MCP server into `~/.gemini/settings.json` at boot. The adapter
seeds `GEMINI.md` from `system-prompt.md` so the agent has role context on
first message.

---

### 2 — Set environment variables

```bash
export PLATFORM_URL=http://localhost:8080   # your running platform
export PLATFORM_TOKEN=<admin-bearer-token>  # printed at startup
export GEMINI_API_KEY=<your-api-key>        # NEVER hardcode this
```

The demo script reads all credentials from env vars — no secrets in source.

---

### 3 — Run

```bash
make run
# or: pip install httpx && python demo.py
```

---

## Expected output

```
[1] Creating gemini-cli workspace...
  created  id=a1b2c3d4-5678-...

[2] Storing GEMINI_API_KEY as workspace secret (value never logged)...
  secret stored

[3] Waiting for workspace to come online (up to 90 s)...
  online in ~18 s

[4] Sending task via A2A proxy...
  Task: "List the three biggest advantages of Google Gemini 2.5 Pro ..."

[5] Gemini CLI agent reply:

  1. Gemini 2.5 Pro's one-million-token context window lets it ingest entire
     codebases in a single pass, eliminating the repeated context-loading
     overhead GPT-4o requires.
  2. Its native multimodal input natively processes screenshots and diagrams
     alongside code, so UI-driven debugging tasks need no preprocessing step.
  3. Google's function-calling latency benchmarks show lower P99 for
     tool-call round-trips, which compounds in ReAct loops across many steps.

[6] Deleting demo workspace...
  workspace deleted

Demo complete.
```

---

## How it works — under the hood

```
demo.py
  │
  ├─ POST /workspaces          → platform creates Docker container
  │    runtime: gemini-cli       adapter.setup() writes ~/.gemini/settings.json
  │                               seeds GEMINI.md from system-prompt.md
  │
  ├─ PUT  /workspaces/:id/secrets → GEMINI_API_KEY stored AES-256-GCM
  │
  ├─ GET  /workspaces/:id  (poll) → waits for status=="online"
  │    (workspace registers via POST /registry/register)
  │
  ├─ POST /workspaces/:id/a2a  → JSON-RPC 2.0  method: message/send
  │    platform proxies to gemini CLI subprocess
  │    CLI runs: gemini --yolo --model gemini-2.5-flash -p "<task>"
  │    MCP tools (delegate_task, commit_memory, …) available via settings.json
  │
  └─ DELETE /workspaces/:id    → container removed
```

### Key adapter decisions (from PR #379)

| Decision | Why |
|----------|-----|
| `~/.gemini/settings.json` for MCP | Gemini CLI ignores `--mcp-config`; adapter merges A2A server entry on `setup()`, preserving user's existing MCP tools |
| `GEMINI.md` as memory file | Equivalent of `CLAUDE.md` for Claude Code; seeded from `system-prompt.md` on first boot so agents start with role context |
| `--yolo` flag | Non-interactive mode — auto-approves all tool calls, required for headless subprocess execution |
| `gemini-2.5-flash` for demo | Faster boot; switch to `gemini-2.5-pro` for production workspaces needing deeper reasoning |

---

## Swap in a different model

```bash
# In demo.py, change runtime_config.model:
"model": "gemini-2.5-pro",   # full reasoning
"model": "gemini-2.0-flash",  # fastest, cheapest
```

Or set it per-workspace via the Molecule AI canvas → Config → Runtime.

---

## Multi-provider example

Once you have a `gemini-cli` workspace running alongside a `claude-code` workspace,
you can delegate tasks between them transparently — the A2A protocol is runtime-agnostic:

```python
# From your orchestrator workspace (claude-code, hermes, etc.)
result = delegate_task(
    workspace_id="<gemini-cli-workspace-id>",
    task="Summarise the attached diff and suggest three test cases.",
)
```

No code changes needed. The orchestrator doesn't know (or care) which model
is running on the other side.

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Workspace stuck in `provisioning` | Check `docker images` for `workspace-template:gemini-cli`; re-run `build-all.sh gemini-cli` if missing |
| `failed` status immediately | Check platform logs: `GEMINI_API_KEY` missing or `npm install -g @google/gemini-cli` failed during image build |
| A2A call times out | `gemini-cli` cold-start on first task can take 15–20 s; increase `timeout=120` in demo.py if needed |
| `code 422` on workspace create | Platform requires `runtime: "gemini-cli"` to be in `RUNTIME_PRESETS`; confirm you're on main after PR #379 |

---

## Related

- [PR #379 — gemini-cli runtime adapter](https://github.com/Molecule-AI/molecule-core/pull/379)
- [Tutorial: Running a Gemini CLI Workspace](../../docs/tutorials/gemini-cli-runtime.md) *(PR #509)*
- [Adapter source](../../workspace-template/adapters/gemini_cli/adapter.py)
- [CLI executor preset](../../workspace-template/cli_executor.py)
- [A2A proxy API reference](../../docs/api-reference.md#a2a-proxy)
