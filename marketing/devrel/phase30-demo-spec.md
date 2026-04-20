# Phase 30 Demo Spec — Remote Workspaces & Cross-Network Federation
> For: DevRel + Marketing | Status: DRAFT | Phase 30 GA target

---

## 1. Demo Scenario

**Title:** *"Your Agent. Your Laptop. On the Canvas."*

**Premise:** A developer runs a Python agent on their laptop, connected to the internet, registering to a Molecule AI org running on a cloud platform. A parent PM agent on the canvas dispatches a research task. The remote agent receives it via A2A, processes it, and returns the result — all visible in real time on the canvas.

**Audience:** Indie developers evaluating Molecule AI, enterprise teams evaluating multi-cloud agent deployment.

**Duration:** 8–10 minutes live, 3 minutes narrated.

---

### Full Walkthrough (Live Demo Steps)

**Setup (done before recording, shown as screenshots):**

1. Dev has a Molecule AI platform running at `https://acme.moleculesai.app`
2. Canvas shows a PM workspace ("pm-agent") already online
3. Dev's laptop is on a different network — no shared Docker network, no VPN

**On screen (live or narrated):**

```
DEVELOPER LAPTOP                          MOLECULE AI PLATFORM
   |                                              |
   | 1. POST /workspaces                          |
   |    {"name":"researcher",                    |
   |     "runtime":"external",                   |
   |     "url":"https://laptop:5000"}           |
   |  ─────────────────────────────────────────►  |
   |  ←─ 201 {"id":"ws-abc123", ...}            |
   |                                              |
   | 2. POST /registry/register                  |
   |    {id:"ws-abc123", url:"...",              |
   |     agent_card:{name:"researcher",          |
   |     skills:["research","web-search"]}}      |
   |  ─────────────────────────────────────────►  |
   |  ←─ 200 {"status":"registered",            |
   |          "auth_token":"mol_..."}  ← SAVE   |
   |                                              |
   | 3. GET /workspaces/ws-abc123/secrets/values |
   |    Authorization: Bearer mol_...             |
   |  ─────────────────────────────────────────►  |
   |  ←─ 200 {"OPENAI_API_KEY":"sk-..."}        |
   |                                              |
   | 4. POST /registry/heartbeat  every 30s      |
   |    Authorization: Bearer mol_...            |
   |  ─────────────────────────────────────────►  |
   |    Canvas shows: researcher = ONLINE (REMOTE)|
   |                                              |
   | 5. PM agent dispatches task via A2A         |
   |    Canvas My Chat → "Research competitor X"  |
   |  ─────────────────────────────────────────►  |
   |    Platform proxies → POST laptop:5000/a2a   |
   |  ←─ 200 {"result":{"message":{...}}}        |
   |                                              |
   | 6. Researcher result shown in Canvas        |
   |    Researcher chat tab shows full reply      |
```

---

## 2. Minimum Viable Demo (Under 10 Minutes)

**What to prep before the demo:**
- Running platform (self-hosted or SaaS beta)
- `pip install requests` on laptop
- `ghcr.io/molecule-ai/workspace-template` image available (for platform side)
- ngrok or Cloudflare Tunnel running on laptop: `ngrok http 5000`
- Write down the `WORKSPACE_ID` and `PLATFORM_URL`

**Script for the MVP (5 minutes live):**

```bash
# STEP 1 — Create the workspace (platform side, admin token)
PLATFORM=https://acme.moleculesai.app
ADMIN_TOKEN=mol_admin_...
WORKSPACE_NAME=researcher

WORKSPACE_RESP=$(curl -s -X POST $PLATFORM/workspaces \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$WORKSPACE_NAME\",\"runtime\":\"external\",\"tier\":2}")
echo $WORKSPACE_RESP | jq

WORKSPACE_ID=$(echo $WORKSPACE_RESP | jq -r '.id')

# STEP 2 — Seed a secret so pull_secrets has something to show
curl -s -X POST $PLATFORM/workspaces/$WORKSPACE_ID/secrets \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"key":"MODEL_NAME","value":"gpt-4o"}'

# STEP 3 — On laptop: run the remote-agent demo
# (uses RemoteAgentClient from molecule-sdk-python)
export WORKSPACE_ID=$WORKSPACE_ID
export PLATFORM_URL=$PLATFORM
export MAX_ITERATIONS=20

python3 run.py

# STEP 4 — Show canvas: workspace appears as REMOTE badge
# Canvas → researcher node → Online → Chat tab
```

**What to narrate at each step:**
1. "This workspace was created with `runtime: external` — no Docker provisioning happens. The platform just registers the row and waits for the agent to call home."
2. "The auth token was returned once, at registration. It's saved to disk. Every subsequent call — secrets, heartbeat, A2A — is authenticated with it."
3. "The agent pulls its API keys from the platform. No env vars baked into the container. Rotate the secret in the UI, the agent picks it up on next pull."
4. "Canvas shows a purple REMOTE badge. Same status, same chat, same terminal access as any Docker workspace — the deployment location is invisible to the rest of the org."
5. "The PM dispatches a task. The platform proxies it to the laptop's endpoint. No Docker bridge, no shared network — it works because the agent registered its URL and keeps a heartbeat alive."

---

## 3. Screencast Outline (5 Key Moments)

### Moment 1: Platform empty state → PM workspace online (0:00–0:20)
**What to show:** Canvas with a PM workspace (already set up as org template). Brief zoom on the node — status, role, chat tab. The org is a skeleton at this point: one PM, no reports.

**Narration:** *"Molecule AI runs a PM agent on a cloud platform. The team is small — one PM, one canvas, everything in one place. Now let's add a researcher running on a laptop across the internet."*

---

### Moment 2: ngrok tunnel + workspace creation (0:20–1:00)
**What to show:** Terminal on laptop. `ngrok http 5000` running. `curl` creating the external workspace. Workspace ID copied.

**Narration:** *"The agent creates a workspace row with `runtime: external`. No Docker involved — the platform just records the identity and waits for it to call home."*

**Visual:** Highlight `runtime: "external"` in the curl command.

---

### Moment 3: Registration + token cache (1:00–1:45)
**What to show:** `python3 run.py` starting. Registration log line. Token saved to `~/.molecule/<id>/.auth_token`. Secrets pulled. Heartbeat loop starting.

**Narration:** *"The SDK registers with the platform, receives a 256-bit auth token, and caches it to disk. That token is the agent's identity — it's how the platform knows this is the researcher workspace, not an imposter. The agent then pulls its secrets — API keys, model names — without any baked-in environment variables. And it starts its heartbeat loop, every 30 seconds."*

**Visual:** Show `~/.molecule/` directory with token file. Show the secret keys returned.

---

### Moment 4: Canvas update — REMOTE badge appears (1:45–2:15)
**What to show:** Canvas, live refresh. Researcher node appears under PM. Purple REMOTE badge. Status: online. Current task: "remote-agent demo idle". Ping the activity panel to show heartbeat activity.

**Narration:** *"Back on the canvas — the researcher is online. Purple badge means it's remote — not a Docker container on this platform. Same status indicator as any other workspace. Same chat tab. The platform doesn't care where it's running."*

**Visual:** Circle the REMOTE badge. Show the heartbeat tick in the activity log.

---

### Moment 5: Task dispatch and result (2:15–3:00)
**What to show:** PM's My Chat input: "Research Anthropic's latest model release and summarize in 3 bullet points." Send. Canvas shows "current task: researching" on researcher node. Researcher replies. Result appears in PM's chat.

**Narration:** *"The PM dispatches a task. The platform routes it to the laptop — same A2A protocol used for every agent call, regardless of where the target runs. The laptop processes it, returns the result, and it appears in the PM's chat. No special configuration on either side — the platform's A2A proxy handles the routing."*

**Visual:** A2A JSON-RPC payload shown briefly in researcher terminal. Canvas showing result.

---

## 4. docs/guides/remote-workspaces.md — Draft Intro + Prerequisites

```markdown
# Remote Workspaces — Run Agents Anywhere, Govern From One Platform

> Phase 30: agents running outside the platform's Docker network can now join
> your Molecule AI org, appear on the canvas, receive A2A tasks from parent
> agents, and report status — all with the same auth, lifecycle, and
> observability as containerized workspaces.

**Phase 30 GA:** 2026-04-20 | PRs: #1075–#1083, #1085–#1100

---

## What Problem This Solves

Most agent platforms assume all agents run in the same environment as the
control plane. Molecule AI supported external agents as a development escape
hatch, but the production story was "all agents on this Docker network."

Phase 30 changes that. Your org can now include agents running on:

- A developer's laptop across the internet
- A server in a different cloud region
- An on-premises machine behind a NAT
- A third-party SaaS bot with an HTTP endpoint

From the canvas and from other agents, they're indistinguishable from
containerized workspaces. They have the same auth contract, the same A2A
interface, the same lifecycle controls. Where they run is a deployment
detail — not an architectural constraint.

---

## Prerequisites

| Requirement | Details |
|---|---|
| **Platform** | Molecule AI platform running v0.30+ (`go run ./cmd/server` from `workspace-server/` or the current `main` image) |
| **Admin access** | An `ADMIN_TOKEN` or org API key with permission to create workspaces |
| **Python ≥ 3.11** | For the `molecule-sdk-python` client (`pip install molecule-ai-sdk`) |
| **Publicly reachable endpoint** | The agent's host must be reachable from the platform over HTTPS. If behind NAT, use [ngrok](https://ngrok.com) or [Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/). |
| **Network** | Outbound HTTPS from the agent to the platform; inbound HTTPS from the platform to the agent's A2A endpoint |

### SDK Installation

```bash
pip install molecule-ai-sdk
```

Or from the repo checkout:

```bash
pip install -e sdk/python/
```

The SDK includes `RemoteAgentClient` — a dependency-light Python client (only `requests`) that wraps all Phase 30 endpoints.

---

## Architecture at a Glance

```
Laptop (remote agent)                Molecule AI Platform
  │                                        │
  │  POST /workspaces                      │
  │  POST /registry/register  ────────────► │  ← admin token (one-time)
  │  ←─ auth_token (256-bit)  ◄────────── │  ← shown once, saved to disk
  │                                        │
  │  GET /workspaces/:id/secrets/values     │  ← bearer: auth_token
  │  POST /registry/heartbeat  (30s loop)  │
  │  GET  /workspaces/:id/state  (30s loop)│
  │                                        │
  │  ◄── A2A task dispatch ────────────── │  ← platform → laptop (HTTPS)
  │  ──► A2A response  ──────────────────► │  ← laptop → platform
  │                                        │
Canvas (any browser)  ◄── WebSocket ─────► Platform
  │                        fanout
  │
  └─── sees: researcher [ONLINE] [REMOTE] badge
```

**Key properties:**
- The agent **pulls** its secrets at boot (not baked into the container at provision time)
- Liveness is maintained by **heartbeat + state polling** (no WebSocket required from the agent side)
- The platform **proxies A2A calls** to the agent's registered URL — no inbound firewall rules on the platform
- The auth token is **workspace-scoped**: a leaked token can't impersonate another workspace

---

## Quick Start

```bash
# 1. Create the workspace (admin side)
WORKSPACE=$(curl -s -X POST https://acme.moleculesai.app/workspaces \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"researcher","runtime":"external","tier":2}')
WORKSPACE_ID=$(echo $WORKSPACE | jq -r '.id')

# 2. Run the agent (any machine that can reach the platform)
pip install molecule-ai-sdk

python3 - <<'EOF'
from molecule_agent import RemoteAgentClient
import os, logging

client = RemoteAgentClient(
    workspace_id = os.environ["WORKSPACE_ID"],
    platform_url = os.environ["PLATFORM_URL"],
    agent_card   = {"name": "researcher", "skills": ["web-search", "research"]},
)
client.register()                      # Phase 30.1 — get + cache token
secrets = client.pull_secrets()         # Phase 30.2 — decrypt API keys
print("Secrets:", list(secrets.keys()))

# Keep alive + respond to platform commands
client.run_heartbeat_loop(
    task_supplier = lambda: {
        "current_task": "idle",
        "active_tasks": 0,
    }
)
EOF
```

The agent appears on the canvas with a **purple REMOTE badge** within seconds. From there it behaves identically to any other workspace: receive A2A tasks, update its agent card, report status.

---

## What Phase 30 Covers

| Phase | What shipped | Endpoint |
|---|---|---|
| 30.1 | Workspace auth tokens | `POST /registry/register`, `POST /registry/heartbeat` |
| 30.2 | Token-gated secrets pull | `GET /workspaces/:id/secrets/values` |
| 30.3 | Plugin tarball download (remote install) | `GET /plugins/:name/download` |
| 30.4 | Workspace state polling (no WebSocket needed) | `GET /workspaces/:id/state` |
| 30.5 | A2A proxy enforces caller token | `POST /workspaces/:id/a2a` |
| 30.6 | Sibling discovery + URL caching | `GET /registry/:id/peers` |
| 30.7 | Poll-liveness for external runtime | Redis TTL (90s timeout) |
| 30.8 | Remote-agent SDK + docs | `molecule-sdk-python` |

---

## Next Steps

- **[External Agent Registration Guide →](/docs/guides/external-agent-registration)** — full endpoint reference, Python + Node.js examples, troubleshooting
- **[molecule-sdk-python →](https://github.com/Molecule-AI/molecule-sdk-python)** — SDK source, `RemoteAgentClient` API docs
- **[SDK Examples →](https://github.com/Molecule-AI/molecule-sdk-python/tree/main/examples/remote-agent)** — `run.py` demo script, annotated walkthrough
```

---

## 5. TTS Audio Script — 60-Second Phase 30 Announcement

**Output:** `marketing/audio/phase30-announce.mp3`
**Duration target:** ~60 seconds
**Voice:** Neutral professional (announcement style)
**Script below — read verbatim:**

---

> Molecule AI ships Phase 30 today — Remote Workspaces is generally available.
>
> Starting now, any agent can run anywhere: your laptop, a different cloud, an edge device, a third-party endpoint. It registers with your Molecule org, appears on the canvas with a remote badge, receives tasks from parent agents, and reports status — just like an agent running in Docker.
>
> The auth contract is the same. The A2A protocol is the same. The canvas experience is the same. The only difference is where the agent's process lives.
>
> Here's what Phase 30 delivers. Workspace auth tokens so every remote agent has a cryptographic identity. A secrets pull endpoint so API keys are managed centrally, not baked into container images. A state polling interface so agents can stay alive without a WebSocket connection. And an SDK — Python, dependency-light, just requests — that wraps all of it.
>
> To onboard a remote agent: create a workspace with runtime external, point it at your platform URL, and run the SDK. Within seconds it shows up on the canvas, purple badge and all.
>
> Phase 30 turns Molecule AI from a self-hosted tool into an enterprise agent fleet platform. Agents run anywhere. Governance stays in one place.
>
> Learn more at moleculesai dot A I, and check the docs for the quick start guide.

---

*Script word count: 253 words → ~60 seconds at 140 WPM delivery pace.*
