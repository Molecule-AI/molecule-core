---
title: "Heterogeneous Fleet Visibility — One Canvas, Every Runtime"
date: 2026-04-21
slug: heterogeneous-agent-fleet-visibility
description: "Your laptop agent, your AWS EC2 agent, and your on-premises agent all report to the same canvas. Here's what that actually looks like and how it works under the hood."
tags: [platform, fleet-management, remote-agents, canvas, multi-cloud]
---

# Heterogeneous Fleet Visibility — One Canvas, Every Runtime

The hardest part of running a mixed agent infrastructure isn't the agents — it's the visibility. When your PM agent runs on GCP, your data researcher runs on AWS, and your on-premises pipeline agent runs behind your company firewall, you have three different systems to watch. Three different dashboards. Three different failure modes.

Phase 30 solves this structurally. All three agents appear on the same canvas, with the same status indicators, the same activity log, and the same task dispatch interface. The runtime differences are normalized by the platform.

This post explains what that looks like in practice and why the normalization is architectural — not a cosmetic layer.

---

## What you see in Canvas

When you open the workspace list in Canvas, every agent — regardless of where it runs — appears as a card with:

- **Name and tier** — the agent's configured name and role tier
- **Runtime badge** — a purple `REMOTE` badge for non-Docker agents, nothing for Docker containers
- **Status** — `ONLINE`, `DEGRADED`, or `OFFLINE` — based on heartbeat polling, not Docker health checks
- **Current task** — what the agent is doing right now, updated on each heartbeat
- **Active task count** — number of tasks in flight
- **Error rate** — percentage of calls that returned errors on the last poll window

For a Docker agent, the same fields are populated. For a remote agent on your laptop, the same fields are populated. The platform polls each workspace's `GET /workspaces/:id/state` endpoint on the same schedule regardless of runtime.

---

## Why this is architectural, not cosmetic

A cosmetic layer would normalize the display by querying each runtime separately and merging the results in the browser. That approach has two failure modes: the merge logic breaks when one runtime is slow or unreachable, and there's no single source of truth for fleet state.

Molecule AI's normalization is deeper. Every workspace — Docker or remote — implements the same state contract:

```python
# What the platform polls from every workspace
GET /workspaces/:id/state
→ {
    "workspace_id": "ws-abc",
    "status": "online",          # online | degraded | offline
    "current_task": "running research query",
    "active_tasks": 2,
    "error_rate": 0.0,
    "last_seen": "2026-04-21T12:00:00Z"
}
```

This contract is the same regardless of where the agent runs. The platform stores the state. Canvas reads from the platform. If the platform is unreachable, Canvas shows stale data with a warning — but the logic doesn't branch based on runtime type.

The practical benefit: if you need to diagnose why your on-premises agent went offline, you open Canvas. You see the last-seen timestamp, the error rate, and the activity log. You don't open a terminal on the server.

---

## What the runtime badge actually signals

The purple `REMOTE` badge on a workspace card tells you one specific thing: this agent doesn't run in a Docker container managed by the platform. That's it. It doesn't mean the agent is less trusted, less capable, or less monitored.

The badge exists because agents on your laptop or behind a NAT need the platform A2A proxy to receive task dispatches — they can't receive inbound connections. Docker agents can receive dispatches directly. The badge is the reminder: this agent's inbound traffic routes through the proxy.

For operations, this means:
- **Proxy path:** if a remote agent is slow to receive tasks, the bottleneck is the proxy's outbound connection to the agent's registered URL
- **Docker path:** if a Docker agent is slow, the bottleneck is the container's own processing

Both cases surface as task dispatch latency in the activity log. The canvas doesn't distinguish between them for monitoring purposes.

---

## The A2A routing path

When a PM agent dispatches a task to a remote researcher agent, the path looks like this:

```
PM Agent (GCP) → Platform A2A Proxy → Researcher Agent URL (laptop, behind NAT)
```

The platform proxy holds the task dispatch open until the researcher agent polls and claims it, or until the dispatch times out. The proxy is stateful: it knows which dispatch is outstanding and can retry on the researcher's next poll.

For Docker agents, the path is direct:
```
PM Agent (GCP) → Researcher Agent (Docker on platform network)
```

The platform proxy falls out of the path for Docker agents — they have inbound connectivity. The PM agent can call the researcher's internal service URL directly.

Both paths are bearer-authenticated on every hop. The canvas shows both dispatches identically: task queued, task claimed, task complete.

---

## Activity log — one schema, every runtime

The Canvas activity log captures every task dispatch and response in the same schema regardless of which runtime the sender or receiver uses:

```
jane@acme.com → pm-agent: "Run the competitor analysis"
pm-agent → researcher (REMOTE): "pull latest sales numbers"
researcher (REMOTE) → pm-agent: results (3 artifacts)
pm-agent → jane@acme.com: analysis complete (1 artifact)
```

The `(REMOTE)` suffix appears next to the researcher because it's a remote agent. The schema is identical to what you'd see if the researcher were a Docker agent.

This is the auditability benefit of heterogeneous fleet visibility: every action across your entire agent fleet is in one log, in one format, queryable by workspace, time range, or actor. No cross-referencing between separate runtime logs.

---

## Getting started

Register your first remote agent in under five minutes:

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

It appears in Canvas within seconds. The `REMOTE` badge is the only difference.

→ [Remote Workspaces Guide](/docs/guides/remote-workspaces.md)
→ [Phase 30: Run AI Agents Anywhere](/blog/remote-ai-agents)