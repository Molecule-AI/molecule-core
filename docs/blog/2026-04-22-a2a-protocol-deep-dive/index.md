---
title: "A2A Protocol: Peer-to-Peer Agent Communication"
date: 2026-04-22
slug: a2a-protocol-deep-dive
description: "How Molecule AI's A2A protocol enables peer-to-peer agent communication — no platform relay, direct workspace-to-workspace messaging."
tags: [A2A, protocol, architecture, multi-agent, technical]
author: Molecule AI
og_title: "A2A Protocol: Peer-to-Peer Agent Communication"
og_description: "How Molecule AI's A2A protocol enables peer-to-peer agent communication — no platform relay, direct workspace-to-workspace messaging."
og_image: /assets/blog/2026-04-22-a2a-protocol-deep-dive-og.png
twitter_card: summary_large_image
canonical: https://molecule.ai/blog/a2a-protocol-deep-dive
keywords:
  - A2A protocol
  - agent-to-agent
  - AI agent protocol
  - AI agent architecture
  - multi-agent platform
  - a2a protocol deep dive
  - protocol:
  - peer-to-peer

---
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "A2A Protocol: Peer-to-Peer Agent Communication",
  "datePublished": "2026-04-22",
  "dateModified": "2026-04-22",
  "author": {
    "@type": "Organization",
    "name": "Molecule AI"
  },
  "publisher": {
    "@type": "Organization",
    "name": "Molecule AI",
    "logo": {
      "@type": "ImageObject",
      "url": "https://molecule.ai/logo.png"
    }
  },
  "description": "How Molecule AI's A2A protocol enables peer-to-peer agent communication — no platform relay, direct workspace-to-workspace messaging.",
  "keywords": "A2A protocol, agent-to-agent, AI agent protocol, AI agent architecture, multi-agent platform, a2a protocol deep dive, protocol:, peer-to-peer",
  "url": "https://molecule.ai/blog/a2a-protocol-deep-dive"
}
</script>


# A2A Protocol: Peer-to-Peer Agent Communication

When two AI agents need to work together, how they communicate is an architectural decision with long-term consequences.

Most multi-agent systems use a hub-and-spoke model: every agent-to-agent call routes through a central orchestrator. The orchestrator is always in the message path — it receives, routes, and forwards. That works, but it introduces latency, a single point of failure, and a hard dependency: the orchestrator must be available for any agent to talk to any other agent.

Molecule AI's A2A (Agent-to-Agent) protocol takes a different approach. The platform handles discovery. Messages go workspace-to-workspace. The platform is never in the message path.

This post walks through how that works in detail — the protocol flow, the discovery model, the authentication model, and what peer-to-peer communication means in practice for multi-agent systems.

---

## The Problem with Hub-and-Spoke

Before the alternative, it's worth being precise about what's broken with central orchestration.

When agent A needs to ask agent B to do something in a hub-and-spoke system:
1. A sends a message to the orchestrator
2. The orchestrator decides whether to forward it to B
3. B's response goes back through the orchestrator

That seems fine until you ask: what happens when the orchestrator is down? No agent can reach any other agent. What happens when the orchestrator has 50 agents all sending tasks through it simultaneously? Every task competes for the same relay bandwidth. And what happens when you want to run two agents on different infrastructure — say, one on a cloud VM and one on a developer's laptop? The orchestrator needs to be reachable from both networks.

Hub-and-spoke is simple to reason about but it couples agent communication to platform availability. That's a meaningful constraint for production systems.

---

## How Molecule AI's A2A Works — Protocol Flow

A2A in Molecule AI is direct peer-to-peer communication between workspaces. Here's the exact sequence when workspace A wants to delegate a task to workspace B.

### Step 1: Discovery Request

Workspace A has a task for workspace B. Before it can send anything, it needs B's URL. A asks the platform:

```
GET /registry/discover/{workspace-b-id}
Header: X-Workspace-ID: workspace-a-id
```

### Step 2: Permission Check

The platform runs `CanCommunicate(caller=workspace-a-id, target=workspace-b-id)`. This is the authorization gate. The platform checks whether workspace A is allowed to communicate with workspace B — same org, same hierarchy, not blocked.

### Step 3: URL Resolution

If permission is granted, the platform returns B's URL. How it resolves the URL depends on who's asking:

- **From another workspace (Docker-internal):** Returns `ws:{id}:internal_url` from Redis — containers reach each other directly by hostname on the Docker network. No NAT, no port forwarding.
- **From Canvas or external caller:** Returns `ws:{id}:url` from Redis — the host-mapped URL (the ephemeral `127.0.0.1:PORT` bound by the provisioner). Canvas can't reach Docker-internal URLs directly.

If the URL isn't in Redis cache, the platform reads from Postgres and refreshes the cache.

### Step 4: Direct Message

Workspace A sends the task directly to workspace B — HTTP POST, JSON-RPC 2.0. The platform is not in the message path. It will never see this request.

```json
{
  "jsonrpc": "2.0",
  "id": "task-789",
  "method": "message/send",
  "params": {
    "message": {
      "role": "user",
      "parts": [
        { "kind": "text", "text": "Review the pull request at molecule-core#342 and summarize the changes" }
      ],
      "messageId": "msg-456"
    }
  }
}
```

### Step 5: Response

Workspace B processes the task and responds. For long-running tasks, B streams progress back to A via SSE — `working` events with intermediate updates. When done, a terminal event fires (`completed`, `failed`, or `canceled`) and any artifacts are returned:

```json
{
  "status": "completed",
  "artifacts": [
    {
      "type": "text/plain",
      "content": "PR #342 adds 214 lines. Key changes: refactors registry/discover to use Postgres instead of in-memory map, adds CanCommunicate() permission check on all discover endpoints, and updates Redis cache invalidation to trigger on workspace state changes. LGTM with one suggestion on the cache invalidation edge case."
    }
  ]
}
```

The platform sees none of this.

---

## On-Demand Discovery: Why Topology Isn't Pushed at Startup

A2A uses on-demand discovery, not push-at-startup. Here's why that matters.

When a workspace boots, it fetches peer Agent Cards (at `/.well-known/agent-card.json`) to build its system prompt — but it doesn't get A2A URLs at startup. It gets them at the moment it decides to delegate.

The reason: **topology changes while agents run**. Sub-workspaces get added, removed, come online and go offline. If you push URLs at startup, you need to also push every topology change to every affected workspace and keep them in sync. That's complex and fragile.

On-demand fits naturally with how agents work. An agent only needs to know another workspace's URL at the moment it decides to delegate — not before. The platform resolves it, caches it (in Redis, by workspace ID), and the message goes direct.

The cache holds the URL until the workspace container restarts or the URL changes. The platform's registry re-reads from Postgres on cache miss.

---

## Authentication: Discovery-Time Validation

The platform's authorization gate runs at discovery time. When workspace A calls `GET /registry/discover/:id`, the platform checks `CanCommunicate(A, B)` and returns B's URL only if the permission check passes.

After that — once A has B's URL — direct A2A calls are unauthenticated in the MVP.

This is acceptable because:
- All workspaces are provisioned by the same platform on trusted infrastructure
- Docker network isolation (`molecule-monorepo-net`) limits what can reach workspace endpoints
- The tool is self-hosted; the operator controls the network boundary

The known gap is in the cache: once A has cached B's URL, nothing stops A from calling B directly even after the hierarchy changes and A is no longer supposed to reach B. The cached URL remains valid until the container restarts or the URL changes.

The post-MVP fix is platform-issued signed tokens scoped to the caller/target pair — issued on discovery, validated on every A2A request. When the hierarchy changes, old tokens expire and new discovery attempts are blocked by `CanCommunicate()`.

---

## The Task Lifecycle

Every A2A message creates a task with a defined lifecycle:

```
submitted → working → completed
                    → failed
                    → canceled
           → input-required → working (caller provides follow-up)
```

The caller chooses how to wait:

```python
# Synchronous — short tasks, caller blocks until terminal state
result = await a2a.send({
    "method": "message/send",
    "params": { "message": { "role": "user", "parts": [...] } }
})
# Returns when completed or failed. No SSE streaming.

# Streaming — long tasks, caller subscribes to SSE progress
async for event in a2a.subscribe({
    "method": "message/sendSubscribe",
    "params": { "message": { "role": "user", "parts": [...] } }
}):
    if event["status"] == "working":
        print(event["message"])  # intermediate progress

    if event["status"] in ("completed", "failed", "canceled"):
        result = event["artifacts"]
        break  # terminal event — stream ends here
```

No polling. The SSE stream always ends with a terminal event — the caller knows the task is done without needing to check separately.

---

## What This Means for Developers

Three practical properties follow from peer-to-peer A2A:

**Lower latency.** Messages go workspace-to-workspace. There's no platform relay in the path — no serialization-deserialization on the platform side, no extra hop. For short tasks, this is noticeable.

**Fault isolation.** If workspace B fails, that doesn't cascade through the orchestrator. Workspace A gets a failed task response. Other agents continue unaffected. The orchestrator in hub-and-spoke becomes a single point of failure by design.

**Platform independence.** A2A works as long as workspaces can reach each other. If the platform's API layer has issues, discovery may be affected — but agents that have already resolved peer URLs continue communicating. The platform orchestrates; it doesn't proxy.

---

## A2A and MCP: Complementary Protocols

MCP (Model Context Protocol) connects agents to external tools and data sources. A2A connects agents to each other. They operate at different layers.

MCP is agent-to-tool: a browser automation skill, a database connector, a webhook handler. A2A is agent-to-agent: one workspace delegating to another, sharing artifacts, coordinating on a task.

A workspace can implement MCP tools *and* be an A2A peer. A fleet can have some agents specialized for particular tool categories (via MCP skills) while others coordinate work across teams (via A2A). The protocols are complementary, not competing.

---

## Frequently Asked Questions

**What's the difference between A2A and MCP?**

MCP connects an agent to external resources — tools, APIs, data sources. A2A connects two agents to each other. Think of MCP as agent-to-tool and A2A as agent-to-agent. They layer: a workspace can be an A2A peer *and* implement MCP tools for its own sub-agents to call.

**Does A2A work between workspaces on different infrastructure?**

Yes. The discovery step returns the correct URL based on the caller's network context — Docker-internal URLs for intra-network callers, host-mapped URLs for cross-network callers. Remote workspaces on a laptop, a cloud VM, or an on-prem server all register via the same flow and are reachable via A2A from any other workspace in the org.

**What happens if a workspace goes offline mid-delegation?**

If the target workspace is unreachable when the task arrives, the task is rejected with a `connection refused` error. A2A does not queue or retry failed deliveries — that's the caller's responsibility to handle. For long-running tasks, the SSE stream provides progress events so the caller can monitor status and retry the delegation if needed.

**Is A2A traffic encrypted?**

Traffic between workspaces is not encrypted by the protocol itself in the MVP — it relies on network isolation. Docker containers reach each other over the `molecule-monorepo-net` network, which is not exposed externally. Remote workspaces use the platform's routing layer. For deployments where workspace-to-workspace traffic crosses untrusted networks, add TLS termination at the load balancer.

**How is A2A authentication handled?**

Authorization runs at discovery time — the platform checks `CanCommunicate(A, B)` before returning B's URL. After that, direct A2A calls are unauthenticated in the MVP. The post-MVP plan is platform-issued signed tokens scoped to the caller/target pair, validated on every A2A request.

**Does A2A work with LangChain, CrewAI, or other agent frameworks?**

A2A is a Molecule AI workspace-to-workspace protocol. Non-Molecule AI agents can participate as remote workspaces if they implement the A2A client interface (HTTP POST with JSON-RPC 2.0, SSE streaming for long tasks). The A2A client spec is language-agnostic — any HTTP client with SSE support can participate.

---

## CTA

Molecule AI's A2A protocol has been GA since Phase 30 (2026-04-20). The implementation is documented, the JSON-RPC format is stable, and the protocol works across mixed infrastructure — Docker containers and remote workspaces alike.

→ [A2A Protocol docs](/docs/api-protocol/a2a-protocol.md)
→ [Remote Workspaces guide](/docs/guides/remote-workspaces.md)
→ [Molecule AI on GitHub](https://github.com/Molecule-AI/molecule-core)

---

*Molecule AI is open source. A2A shipped in Phase 30. DevRel Engineer to verify JSON-RPC examples before publishing.*

*By the Molecule AI Team*