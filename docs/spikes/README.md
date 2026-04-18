# Spike #745 — Anthropic Managed Agents as a Molecule Executor

**Parent issue:** #742 — "Third executor option: Anthropic Managed Agents"  
**Spike issue:** #745

## What We Evaluated

Anthropic's Managed Agents beta (`managed-agents-2026-04-01`) lets you create
persistent agent objects, spin up per-task sessions, and stream execution events
via SSE — all hosted on Anthropic's infrastructure. The key question for Molecule
is: *can this replace (or complement) the self-hosted Docker workspace executor?*

---

## Demo

`demo.py` exercises the full lifecycle:

```
ANTHROPIC_API_KEY=sk-ant-... python demo.py
```

What it measures:

| Phase | What we time |
|---|---|
| `environment create` | Provisioning a cloud execution environment |
| `agent create` | Storing the agent config (model, system prompt, tools) |
| `cold start` | `sessions.create()` → session ready |
| `turn 1 RTT` | User message → SSE drain → `session.status_idle` |
| `turn 2 RTT` | Same, plus implicit state recall check |

State continuity is verified by injecting a unique token in turn 1 and
asserting the agent quotes it back in turn 2. Exit code 0 = pass, 1 = fail.

---

## Integration Assessment

### 1. Provisioner changes

Molecule's provisioner today calls `docker.NewClient()`, pulls an image,
creates a container with resource limits, and waits for `/registry/register`
from inside the container. A Managed Agents executor would replace that
entire path:

```
current:  docker pull → container run → heartbeat register
proposed: agents.create() → sessions.create() → SSE stream
```

A new `runtime: "managed-agent"` value in `workspaces.runtime` would branch
the provisioner. The workspace row would store `agent_id` (persistent) and
`session_id` (ephemeral per-run) instead of a Docker container ID.

**Migration effort:** medium.  
A new `ManagedAgentProvisioner` can be added alongside the existing Docker
provisioner without touching the common path. The primary cost is the
integration layer described below.

---

### 2. A2A routing — the blocking architectural conflict

This is the hard blocker. Molecule's A2A proxy (`POST /workspaces/:id/a2a`)
resolves `ws.agent_url` and forwards an HTTP POST to the running container.
Every workspace has a persistent, addressable HTTP endpoint.

Managed Agents sessions communicate exclusively through the Anthropic SSE API —
there is no per-session URL that the platform can proxy to. The session is a
streaming consumer, not a server.

Bridging the gap requires one of:

**Option A — Long-poll bridge (complex, fragile)**  
Keep a goroutine open per session holding the SSE stream. When an A2A message
arrives, inject it via `sessions.events.send()` and wait for the next
`agent.message` event. Map response back to A2A caller.  
Risk: the goroutine dies, the session becomes unreachable, and A2A callers time out
with no clear error path.

**Option B — Managed Agents as leaf-only workers (scope reduction)**  
Only use Managed Agents for workspaces that *receive* tasks (no outbound A2A).
The platform queues work, opens a session, streams the result, and closes the
session. No live bridge needed.  
Risk: many real workspaces delegate to peers — leaf-only scope limits
applicability to batch/one-shot agents.

**Option C — Hybrid: MCP bridge**  
Anthropic agents can call MCP servers. The platform exposes its A2A proxy as
an MCP server; the agent's MCP tool calls translate back to A2A messages.  
Risk: this inverts the call direction (agent calls platform instead of
platform-to-agent) and breaks the current workspace-to-workspace trust model.
Security review required before shipping.

---

### 3. Cost model

Managed Agents sessions are charged on top of standard token pricing — the
platform receives its own compute costs. For comparison, the Docker path uses
a customer-supplied model key with zero platform markup.

The cold-start latency (environment + session creation) measured in the demo
adds overhead before the first token. For interactive canvas workflows where
workspaces are expected to be long-lived ("always on"), this model is a poor
fit. For batch workspaces that run occasionally, it may save infrastructure
cost.

---

### 4. API gaps (as of 2026-04-17)

| Molecule requirement | Managed Agents support |
|---|---|
| Persistent HTTP endpoint for A2A | **No** — SSE only |
| Heartbeat / liveness signal | **Partial** — session status via poll or SSE, but no proactive push to the platform |
| Resource limits (memory, CPU) | **No** — environment config offers only `networking` |
| Custom Docker image | **No** — Anthropic-managed base image only |
| `workspace_dir` bind-mount | **No** — files uploaded via `client.beta.files` API |
| Bearer token auth per workspace | **No** — auth is Anthropic API key, not per-workspace token |
| Plugin system (arbitrary pip installs) | **No** — built-in `agent_toolset_20260401` or custom tool callbacks |
| Runtime detection (`config.yaml` introspection) | **Not applicable** — config lives in agent object |

---

## Ship/No-Ship Recommendation

### Decision: **No-ship for the primary executor. Spike further as a batch worker.**

**Rationale:**

1. **A2A proxy is the load-bearing constraint.** Molecule's value proposition
   is multi-workspace orchestration. A workspace executor that can't be reached
   by other workspaces over A2A is not a Molecule workspace — it's a standalone
   call to the Anthropic API with extra steps.

2. **No persistent endpoint = no topology.** The canvas shows workspaces as
   nodes that communicate. A Managed Agents session has no addressable URL; the
   canvas can't represent it as a live peer.

3. **Cold start is non-trivial.** Preliminary measurements from the demo show
   environment + session creation adding visible latency before the first token.
   For the "always-on" UX the canvas targets, this is noticeable.

4. **Scope would be a dead end.** Shipping Managed Agents as a leaf-only,
   no-A2A executor today means two provisioner paths diverge. The Managed Agents
   path can never grow to full parity without Anthropic exposing a persistent
   addressable URL. We'd be maintaining a permanently limited path.

### What to do instead

- **Phase H (planned):** Consider Managed Agents as the execution target for
  *scheduled* tasks only (`workspace_schedules` cron rows). A cron fire could
  spin up a session, run the prompt, stream the result, and self-report via
  `/activity`. No live A2A needed. Effort: ~2 weeks.

- **Watch the API.** If Anthropic ships a stable URL per session (like a
  webhook delivery endpoint), re-evaluate. The MCP bridge angle (Option C above)
  also becomes more viable once Molecule's MCP server is feature-complete.

---

## Rough Effort Estimate (if we did ship)

| Component | Effort |
|---|---|
| `ManagedAgentProvisioner` (create/start/stop session) | 3–5 days |
| A2A bridge goroutine (Option A) | 5–8 days |
| Heartbeat adapter (translate SSE status to `/registry/heartbeat`) | 2–3 days |
| Canvas: hide A2A tab for managed-agent workspaces | 1 day |
| Tests, migration, docs | 3–4 days |
| **Total** | **~3 weeks** |

Even at 3 weeks, the result is a permanently limited path with no A2A and no
resource controls. Not recommended.

---

## Files

| File | Purpose |
|---|---|
| `demo.py` | Runnable spike script — auth, provision, session, two turns, timing |
| `README.md` | This assessment |
