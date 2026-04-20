---
title: "One Canvas, Every Agent: Remote AI Agents and Fleet Visibility on Molecule AI"
date: 2026-04-20
slug: remote-ai-agents
description: "Your Claude Code laptop, your LangGraph cloud instance, and your OpenClaw server — all on the same canvas. Phase 30 ships per-workspace bearer tokens and unified fleet visibility for heterogeneous AI agent fleets."
tags: [platform, remote-agents, fleet-management, a2a]
---

# One Canvas, Every Agent: Remote AI Agents and Fleet Visibility on Molecule AI

> "Our agents need to talk to each other even when they're in different clouds — and we need to see the whole fleet in one place without stitching together five different dashboards."
>
> — Infrastructure lead at a mid-stage SaaS company, describing what they needed before finding Molecule AI Phase 30

That's the problem. Not a hypothetical one.

When your AI agents span your laptop, an AWS EC2 instance, a company's on-premise server, and a contractor's development environment — you need one answer to three questions: Where are my agents right now? What are they doing? And are they actually who they say they are?

Molecule AI Phase 30 ships the answer to all three.

## The Fleet Visibility Problem

Every AI agent platform works fine when your agents are in one place. Docker containers on the same host, all visible to the same canvas, all on the same network. That was Molecule AI up until Phase 29.

But real organizations don't look like that. Your engineering org probably has agents running:

- In CI/CD pipelines (GitHub Actions, AWS CodeBuild)
- On developer laptops for local iteration
- In cloud VMs on AWS, GCP, or Azure
- Behind company firewalls on on-premise infrastructure
- In SaaS integrations that need to participate in your agent hierarchy

Before Phase 30, each of those was invisible to the others. Your CI agent couldn't see your production agents. Your on-premise agent couldn't receive instructions from the PM agent running in the cloud. And you — the operator — had no single view of the whole fleet.

## Phase 30: One Canvas, Every Agent

Phase 30 makes three things possible for the first time:

1. **Any agent, anywhere, on the same canvas.** Remote agents running outside Docker — on any machine, any cloud, any network — register with the platform and appear in your canvas with the same status indicators, activity feeds, and chat interfaces as your local agents.

2. **Unified A2A communication across network boundaries.** Agents in different clouds, behind different firewalls, on different continents can send each other A2A messages through the platform's proxy — with the same permission rules that govern local agents.

3. **Per-workspace bearer tokens.** Every remote agent gets its own cryptographic identity. No shared credentials. No guessing which agent made an API call. No all-or-nothing credential revocation.

The emotional hook is fleet visibility. The technical foundation that makes it work is the auth model.

## How Remote Agents Join the Fleet

A remote agent — running on any machine with an HTTP endpoint — joins your Molecule AI org in six steps.

### Step 1: Create the external workspace

Your platform admin creates an external workspace record via the REST API:

```bash
curl -X POST https://your-platform.molecule.ai/workspaces \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>" \
  -d '{
    "name": "CI Build Agent",
    "role": "ci-agent",
    "runtime": "external",
    "external": true,
    "url": "https://ci-agent.example.com",
    "tier": 2
  }'
```

The response returns a workspace ID. The `runtime: "external"` flag tells the platform not to provision a Docker container — this workspace runs on your infrastructure.

### Step 2: Agent registers and receives a bearer token

The agent calls `POST /registry/register` with its workspace ID and agent card:

```bash
curl -X POST https://your-platform.molecule.ai/registry/register \
  -H "Content-Type: application/json" \
  -d '{
    "id": "<workspace-id>",
    "url": "https://ci-agent.example.com",
    "agent_card": {
      "name": "CI Build Agent",
      "description": "Runs tests and reports results to the PM agent",
      "skills": ["ci", "testing", "reporting"],
      "runtime": "external"
    }
  }'
```

The response includes an `auth_token` — shown **exactly once**, never stored by the platform. The agent must persist this token. Every subsequent authenticated call to the platform uses it.

### Registration in Python

```python
import requests, os, time, threading

PLATFORM_URL = os.environ["PLATFORM_URL"]
AGENT_URL     = os.environ["AGENT_URL"]      # e.g. "https://my-agent.ngrok.io"
ADMIN_TOKEN   = os.environ["ADMIN_TOKEN"]   # platform admin token

# Step 1: create external workspace
workspace = requests.post(
    f"{PLATFORM_URL}/workspaces",
    json={"name": "CI Agent", "runtime": "external",
          "external": True, "url": AGENT_URL},
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"}
).json()
ws_id = workspace["id"]

# Step 2: register — receive bearer token
reg = requests.post(
    f"{PLATFORM_URL}/registry/register",
    json={"id": ws_id, "url": AGENT_URL,
          "agent_card": {"name": "CI Agent", "runtime": "external"}}
).json()
auth_token = reg["auth_token"]   # save this — shown once

# Heartbeat every 30s
def heartbeat():
    while True:
        requests.post(f"{PLATFORM_URL}/registry/heartbeat",
                      json={"workspace_id": ws_id, "error_rate": 0.0,
                            "active_tasks": 0, "current_task": "",
                            "uptime_seconds": int(time.time() - start)},
                      headers={"Authorization": f"Bearer {auth_token}"})
        time.sleep(30)

start = time.time()
threading.Thread(target=heartbeat, daemon=True).start()
```

### Registration in Node.js

```javascript
const PLATFORM = process.env.PLATFORM_URL;
const AGENT_URL = process.env.AGENT_URL;
const ADMIN = process.env.ADMIN_TOKEN;

const create = await fetch(`${PLATFORM}/workspaces`, {
  method: "POST",
  headers: { "Authorization": `Bearer ${ADMIN}`, "Content-Type": "application/json" },
  body: JSON.stringify({ name: "CI Agent", runtime: "external", external: true, url: AGENT_URL })
});
const { id: wsId } = await create.json();

const reg = await fetch(`${PLATFORM}/registry/register`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ id: wsId, url: AGENT_URL,
        agent_card: { name: "CI Agent", runtime: "external" } })
});
const { auth_token } = await reg.json(); // save — returned once

// Heartbeat every 30s
setInterval(async () => {
  await fetch(`${PLATFORM}/registry/heartbeat`, {
    method: "POST",
    headers: { "Authorization": `Bearer ${auth_token}`, "Content-Type": "application/json" },
    body: JSON.stringify({ workspace_id: wsId, error_rate: 0.0,
          active_tasks: 0, current_task: "", uptime_seconds: 0 })
  });
}, 30_000);
```

Full examples with A2A message handling are in the [External Agent Registration Guide](/docs/guides/external-agent-registration).

### Step 3: Pull secrets on demand

Remote agents don't get secrets baked in at container boot. They pull them on demand:

```bash
curl https://your-platform.molecule.ai/workspaces/<workspace-id>/secrets \
  -H "Authorization: Bearer <auth-token>"
```

This returns the decrypted secrets scoped to this workspace — API keys, credentials, anything the platform has stored. The agent uses these to authenticate with its LLM provider, external services, or any tool it needs to do its job.

### Step 4: Start the heartbeat loop

The agent sends a heartbeat every 30 seconds to stay visible on the canvas:

```bash
curl -X POST https://your-platform.molecule.ai/registry/heartbeat \
  -H "Authorization: Bearer <auth-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_id": "<workspace-id>",
    "error_rate": 0.0,
    "active_tasks": 1,
    "current_task": "Running test suite on PR #412",
    "uptime_seconds": 3600
  }'
```

If the platform receives no heartbeat for 60 seconds, the workspace transitions to **offline** on the canvas. This is the liveness signal — visible, real-time, consistent across local and remote agents alike.

### Step 5: Send and receive A2A messages

Remote agents communicate with the rest of the fleet through the platform's A2A proxy. Both sides are authenticated:

```bash
curl -X POST https://your-platform.molecule.ai/workspaces/<target-id>/a2a \
  -H "Authorization: Bearer <auth-token>" \
  -H "X-Workspace-ID: <your-workspace-id>" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"type": "text", "text": "PR #412 tests passed. Ready for review."}]
      }
    },
    "id": "req-456"
  }'
```

The `X-Workspace-ID` header is the caller's identity. The platform's `CanCommunicate` check uses it to enforce hierarchy-based access: agents can only message siblings, parents, children, and themselves. No agent can reach an unrelated workspace.

### Step 6: See the whole fleet in one place

The canvas automatically includes remote agents alongside local ones. Remote agents get a purple **REMOTE** badge so you can tell them apart at a glance. Every other canvas feature — status indicators, chat tabs, activity feed, config management — works identically for remote and local agents.

## The Security Model: Per-Workspace Bearer Tokens

Fleet visibility is the hook. Per-workspace bearer tokens are the foundation that makes it safe.

Every remote agent has:

- **A unique 256-bit token** — cryptographic random, returned once at registration, stored as a SHA-256 hash server-side
- **A workspace identity** — bound to the `X-Workspace-ID` header on every A2A call
- **A revocation path** — immediate, per-agent, no downtime for other agents

The `workspace_auth_tokens` table tracks:

| Field | Purpose |
|---|---|
| `token_hash` | SHA-256 of the plaintext. The platform never stores the actual secret. |
| `prefix` | First 8 characters for display and log attribution |
| `workspace_id` | Which agent this token belongs to |
| `created_by` | Provenance: admin-token, session, or org-api-key |
| `last_used_at` | Audit trail: last time this token exercised an API call |
| `revoked_at` | Immediate revocation: the token stops working on the next request |

Two agents in different clouds both have bearer tokens. Both use those tokens to authenticate to the A2A proxy. The proxy validates both tokens before dispatching any message. Mutual auth, end-to-end.

## Where Remote Agents Fit in Your Organization

### CI/CD pipelines

Your CI agent — running in GitHub Actions, CircleCI, or any CI system — joins your org as a first-class workspace. It registers with a bearer token, pulls its secrets, runs your test suite, and reports results to the PM agent. The PM agent sees the CI agent's status on the canvas. When tests fail, the canvas shows you exactly which agent ran them, with full audit attribution.

### Multi-cloud fleets

An agent running in GCP and an agent running in AWS communicate through the platform's A2A proxy. Both are authenticated. Both appear on the same canvas. The GCP agent doesn't need to know the AWS agent's IP address — it just calls the proxy with the workspace ID, and the proxy routes the message.

### On-premise and air-gapped environments

Agents behind a company firewall — or in environments that can't expose a public endpoint — use a polling model. Instead of receiving WebSocket events, they poll `GET /workspaces/:id/state` for platform-initiated events (pause, resume, config changes). They still send A2A messages outbound. They still appear on the canvas.

### SaaS integrations and webhooks

A third-party SaaS service that exposes an A2A-compatible HTTP endpoint can register as an external workspace. It joins the org hierarchy, receives tasks from the PM agent, and returns results — without any Molecule AI infrastructure running on its end.

## What's Next for Remote Agents

Phase 30 shipped the foundation. The remaining work — plugin tarball download, state polling for behind-NAT agents, poll-based liveness monitoring, and sibling URL caching — completes the remote onboarding story over the next phases.

Direct agent-to-agent mesh across NATs (without routing through the platform proxy) is a future phase. For most use cases, the proxy path is already fast enough and doesn't require any infrastructure changes.

## Get Started

Per-workspace bearer tokens and unified canvas fleet visibility are available now on all Molecule AI deployments.

- [External Agent Registration Guide](/docs/guides/external-agent-registration) — full step-by-step with Python and Node.js examples
- [Token Management API](/docs/guides/org-api-keys) — mint, list, and revoke per-workspace tokens
- [Architecture: Remote Workspaces](/docs/architecture/remote-workspaces) — technical deep-dive on the auth model and network topology

Your heterogeneous fleet is waiting. It all fits on one canvas now.
