# Quickstart Guide

This path is aligned to the current repository and current UI. It gets you from clone to a live workspace on the canvas without assuming any extra platform wrapper.

## Prerequisites

- Docker + Docker Compose v2
- Node.js 20+
- Go 1.25+
- One model/API key for the runtime you want to use
  - `ANTHROPIC_API_KEY`
  - `OPENAI_API_KEY`
  - `GOOGLE_API_KEY`
  - or another provider routed through LiteLLM

## Step 1: Clone the repository

```bash
git clone https://github.com/Molecule-AI/molecule-monorepo.git
cd molecule-monorepo
```

## Step 2: Start the shared infrastructure

Recommended:

```bash
./infra/scripts/setup.sh
```

That brings up Postgres, Redis, and Langfuse.

If you only want the raw compose flow:

```bash
docker compose -f docker-compose.infra.yml up -d
```

## Step 3: Start the platform

```bash
cd workspace-server
go run ./cmd/server
```

The control plane listens on `http://localhost:8080`.

## Step 4: Start the canvas

In a new terminal:

```bash
cd canvas
npm install
npm run dev
```

Open `http://localhost:3000`.

## Step 5: Deploy your first workspace

On a fresh canvas, the center empty state shows template cards plus a blank-workspace option.

You can either:

1. Click a template to provision a ready-made workspace.
2. Click `+ Create blank workspace`.

At the same time, the bottom-left onboarding wizard appears and guides the first-run flow.

## Step 6: Add an API key

1. Select the workspace.
2. Open the `Config` tab.
3. Expand `Secrets & API Keys`.
4. Add the API key in either:
   - `This Workspace`
   - `Global (All Workspaces)`

Global keys are inherited by all workspaces. Workspace keys override globals with the same name.

## Step 7: Send the first message

1. Open the `Chat` tab.
2. Send a prompt such as:

```text
What can you help me with in this workspace?
```

Responses are delivered through the platform A2A proxy and pushed back to the canvas through WebSocket events, with polling kept only as recovery fallback.

## What To Try Next

- **Expand to a team:** right-click a workspace and choose `Expand to Team`.
- **Switch runtime:** use `Config -> Runtime` to move between LangGraph, DeepAgents, Claude Code, CrewAI, AutoGen, and OpenClaw.
- **Inspect operations:** check `Activity`, `Traces`, `Events`, and `Terminal`.
- **Use global keys:** configure one provider once in `Secrets & API Keys -> Global`.
- **Import a template:** use the template palette or `POST /templates/import`.
- **Import/export bundles:** duplicate or move workspace trees as `.bundle.json`.
- **Run a remote workspace:** register an agent on your laptop, a cloud VM, or an on-premises server — it appears on the canvas alongside your Docker workspaces. See [Remote Workspaces](/docs/guides/remote-workspaces.md).

---

## Remote Workspaces — Your Laptop as a Runtime

The quickstart above deploys workspaces as Docker containers on the platform. Phase 30 adds a second option: running an agent on your own infrastructure and registering it with the platform.

This lets you:
- **Debug locally** — run an agent in your IDE with your filesystem, your git config, your SSH keys
- **Cross-cloud fleets** — a PM agent on GCP and a researcher agent on AWS, coordinated from the same canvas
- **Existing agents** — register an agent without containerizing or redeploying it

**What the canvas shows:** Remote workspaces appear identically to Docker workspaces — same status indicators, same activity log, same task dispatch interface. The only visual difference is a purple `REMOTE` badge on the workspace card.

**What changes operationally:** Remote agents use the platform A2A proxy for inbound task dispatches (they can't receive inbound connections from the platform). Docker agents receive dispatches directly. Both paths are bearer-authenticated on every hop.

**How to get started:**

```bash
pip install molecule-ai-sdk
```

```python
from molecule_agent import RemoteAgentClient

client = RemoteAgentClient(
    workspace_id="ws-abc123",
    platform_url="https://acme.moleculesai.app",
    agent_card={"name": "researcher", "skills": ["web-search"]},
)
client.register()
client.run_heartbeat_loop(
    task_supplier=lambda: {"current_task": "idle", "active_tasks": 0}
)
```

The agent appears in Canvas within seconds with a purple `REMOTE` badge. From there, it's indistinguishable from a Docker workspace for operational purposes.

→ [Remote Workspaces Guide](/docs/guides/remote-workspaces.md)
→ [Fleet Visibility Guide](/docs/blog/2026-04-21-fleet-visibility.md)

## Troubleshooting

| Problem | What to check |
|---|---|
| Workspace stays offline | Platform is running, Docker is available, and the runtime has valid credentials |
| Template palette is empty | `workspace-configs-templates/` exists and the platform can read it |
| Chat says agent unavailable | Workspace status is not yet `online` or `degraded` |
| No responses from model | Add the correct API key in `Secrets & API Keys` and restart the workspace |
| Want direct DB access | Postgres and Redis are internal-only; use `docker compose exec postgres psql` or `docker compose exec redis redis-cli` |

## Architecture At A Glance

```text
Browser  -->  Canvas (Next.js :3000)
                |
                v
           Platform (Go :8080)
             |       |       |
             |       |       +--> WebSocket events / A2A proxy / templates / bundles
             |       +----------> Redis
             +------------------> Postgres
                                |
                                v
                       Provisioned workspaces
                     (LangGraph / Claude Code / CrewAI / AutoGen / etc.)

Remote agents  -->  A2A Proxy  <--  Docker workspaces
(laptop, cloud, on-prem)       (platform network)
```
