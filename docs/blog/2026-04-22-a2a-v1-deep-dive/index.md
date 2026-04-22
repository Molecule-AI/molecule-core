---
title: "How Molecule AI's A2A Protocol Works: Peer-to-Peer Agent Communication"
date: "2026-04-22"
slug: "a2a-protocol-deep-dive"
description: "A code-first walkthrough of the A2A (Agent-to-Agent) protocol — peer discovery, JSON-RPC messaging, SSE streaming, and task lifecycle — and what separates a reference implementation from one enterprises can rely on."
og_title: "How Molecule AI's A2A Protocol Works: Peer-to-Peer Agent Communication"
og_description: "A code-first walkthrough of how Molecule AI's A2A protocol handles peer discovery, JSON-RPC messaging, SSE streaming, and task lifecycle — without routing traffic through a central hub."
og_image: /docs/assets/blog/2026-04-22-a2a-v1-deep-dive-og.png
tags: [A2A, protocol, technical, multi-agent, JSON-RPC, enterprise]
keywords: [A2A protocol, agent-to-agent protocol, Molecule AI A2A, MCP vs A2A, multi-agent communication, A2A JSON-RPC, A2A SSE streaming]
canonical: https://docs.molecule.ai/blog/a2a-protocol-deep-dive
---

<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "How Molecule AI's A2A Protocol Works: Peer-to-Peer Agent Communication",
  "description": "A code-first walkthrough of the A2A (Agent-to-Agent) protocol — peer discovery, JSON-RPC messaging, SSE streaming, and task lifecycle.",
  "author": { "@type": "Organization", "name": "Molecule AI" },
  "datePublished": "2026-04-22",
  "publisher": {
    "@type": "Organization",
    "name": "Molecule AI",
    "logo": { "@type": "ImageObject", "url": "https://molecule.ai/logo.png" }
  }
}
</script>

# How Molecule AI's A2A Protocol Works: Peer-to-Peer Agent Communication

*If you read our [A2A Enterprise post](/blog/a2a-enterprise-any-agent-any-infrastructure), here's how the protocol works under the hood.*

The first multi-agent demo is always hub-and-spoke. Agent A talks to an orchestrator. Orchestrator talks to Agent B. It works. Then the agent count grows, and the orchestrator becomes the bottleneck — in latency, in failure modes, and in observability.

Most A2A (Agent-to-Agent) introductions spend their word count explaining why this is a problem. This post is different. It shows you how Molecule AI solves it — with real JSON payloads, real protocol flows, and the architectural decisions that make it production-ready.

---

## The Multi-Agent Communication Problem

The hub-and-spoke model works until it doesn't. When all agent-to-agent traffic routes through a central orchestrator, three things compound:

**Latency scales with hops.** Every task that requires two agents passes through the orchestrator twice — once in, once out. Add a third agent in the chain and the orchestrator becomes the critical path for every response.

**Failure modes cascade.** An orchestrator failure doesn't just stop one agent — it stops every delegation in the system. The orchestrator is a single point of failure for every cross-agent call.

**Observability collapses.** The audit log shows "orchestrator → Agent B" with no visibility into why. If Agent A inside the orchestrator made the delegation decision, the audit trail doesn't show it.

Molecule AI's A2A is peer-to-peer. The platform handles *discovery* — finding whether Agent B exists and whether Agent A is allowed to call it. The message itself goes workspace-to-workspace. The platform is never in the message path.

---

## The Four A2A Methods

A2A is JSON-RPC 2.0 over HTTP. Every request is a JSON object with `jsonrpc`, `method`, and `params`. Every response is a JSON-RPC result or error. The transport is deliberately minimal: if you've built a REST API, you already understand the shape.

The protocol defines four methods:

| Method | Purpose |
|--------|---------|
| `message/send` | Submit a task synchronously — returns the result |
| `message/sendSubscribe` | Submit a task, stream progress via SSE |
| `tasks/get` | Retrieve current task state (idempotency) |
| `tasks/cancel` | Cancel a running task |

`message/sendSubscribe` covers the majority of production use cases. Here's what it looks like:

```json
// Client → Server (workspace B)
POST /a2a
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "method": "message/sendSubscribe",
  "params": {
    "message": {
      "messageId": "msg_01hx9fk3z2k7...",
      "role": "user",
      "parts": [{ "type": "text", "text": "Run the security audit on repo acme/core" }]
    },
    "taskId": "task_01hx9fk3z2k7...",
    "metadata": {
      "callerWorkspaceId": "ws_01hx8abcd...",
      "orgApiKeyPrefix": "mole_a1b2",
      "idempotencyKey": "audit-delegation-20260422-01"
    }
  },
  "id": 1
}
```

The `metadata` field carries the governance layer — `callerWorkspaceId` identifies who made the call, `orgApiKeyPrefix` traces it to the integration owner, and `idempotencyKey` ensures the delegation isn't processed twice even if the network retries.

---

## Agent Discovery: The Registry, Not the Agent Card

Most A2A explainers introduce the *Agent Card* — a JSON document at `/.well-known/agent-card.json` that every compliant agent publishes. That's the spec. Molecule AI's implementation starts one step earlier.

Before Agent A can send a message to Agent B, it needs B's reachable URL. That URL depends on where the two workspaces are deployed — they might both be local Docker services on the same machine, one might be an EC2 Instance Connect SSH workspace, another might be a Fly Machine in a different region. The calling agent doesn't need to know any of this.

Here's how discovery works in Molecule AI:

```
1. Workspace A decides to delegate to Workspace B
2. Workspace A calls: GET /registry/discover/:workspace_b_id
   Header: X-Workspace-ID: ws_01hx8abcd...

3. Platform registry checks: CanCommunicate(ws_01hx8abcd..., ws_01hx9zxy...)
   - Is ws_01hx8abcd... authorized to call ws_01hx9zxy...?
   - Is ws_01hx9zxy... reachable?

4. Registry returns B's current endpoint URL:
   {
     "workspaceId": "ws_01hx9zxy...",
     "url": "http://workspace-b:8080",
     "transport": "docker-internal",
     "version": "1.0"
   }

5. Workspace A sends the JSON-RPC message directly to B — no platform proxy
```

The `CanCommunicate()` permission check is where governance lives. In the current implementation, workspace-to-workspace communication is gated at the platform boundary. An agent can only discover and call workspaces it's authorized to reach.

The critical detail: the platform registry resolves the target URL on-demand, at the moment of delegation. It is not pushed at startup. This matters because:

- A workspace can be stopped, restarted, or migrated without invalidating the registry
- Topology changes don't require re-pushing agent cards across the fleet
- An agent only resolves the peer URL when it actually needs it

This is the "on-demand, not pushed" model that differentiates Molecule AI's A2A from implementations that require topology to be declared upfront.

---

## Authentication: Permission at Discovery, Tokens at Runtime

The permission model has two layers:

**Discovery-time authorization.** `CanCommunicate()` gates whether Workspace A is allowed to discover Workspace B's URL. Unauthorized callers get a `403 Forbidden` — they never learn that B exists.

**Runtime authentication.** Once A has B's URL, the actual JSON-RPC call carries workspace bearer tokens (Phase 30 per-workspace authentication). The target workspace validates the caller's token before processing it. A discovered URL without a valid token is useless.

This means Molecule AI's A2A has no unauthenticated message path. Discovery is gated. Every JSON-RPC call is token-authenticated. There is no mode where an agent can receive an unauthenticated delegation request.

For enterprise buyers, the implication is significant: Molecule AI's A2A cannot be exploited by an agent that wasn't explicitly granted access to the target workspace. The permission model is enforced at two layers, not one.

---

## Task Lifecycle: Progress Streaming and Artifact Return

Once a delegation call is accepted, the task enters an asynchronous lifecycle. The caller doesn't block waiting for a final result — it receives a stream of progress events via Server-Sent Events (SSE), then a final result when the agent finishes.

Here's what the SSE stream looks like (abbreviated):

```
HTTP/1.1 200 OK
Content-Type: text/event-stream

event: WorkProgress
data: {"jsonrpc":"2.0","method":"tasks/progress","params":{"taskId":"task_01hx9fk3z2k7...","status":{"state":"working"},"artial":null}}

event: WorkProgress
data: {"jsonrpc":"2.0","method":"tasks/progress","params":{"taskId":"task_01hx9fk3z2k7...","status":{"state":"working"},"artial":{"parts":[{"type":"text","text":"Cloning repo..."}]}}}

event: WorkProgress
data: {"jsonrpc":"2.0","method":"tasks/progress","params":{"taskId":"task_01hx9fk3z2k7...","status":{"state":"working"},"artial":{"parts":[{"type":"text","text":"Running security scan..."}]}}}

event: TaskCompleted
data: {"jsonrpc":"2.0","method":"tasks/completed","params":{"taskId":"task_01hx9fk3z2k7...","status":{"state":"completed"},"artial":{"parts":[{"type":"text","text":"Security audit complete. 0 critical, 2 medium findings."}]}}}
```

The caller can terminate this stream at any point by sending a `tasks/cancel` request. The target workspace listens for cancellation signals and stops the agent mid-execution if the caller no longer needs the result. This is what "cancelable tasks" means in practice — not just a protocol flag, but a wire-level interrupt that stops compute.

---

## What This Means for Developers

Three practical implications for teams building on Molecule AI's A2A:

**Lower delegation latency.** The platform is not in the message path. Once discovery resolves the peer URL, the JSON-RPC call goes directly to the target workspace. For multi-hop workflows, this eliminates one round-trip per hop compared to hub-and-spoke architectures.

**Fault isolation.** A workspace failure doesn't cascade. If Workspace B crashes mid-delegation, Workspace A receives a connection error — and can retry, escalate, or fall back without affecting other agents. The orchestrator in a hub-and-spoke model would propagate B's failure to every caller waiting on it.

**Cross-infrastructure delegation without a VPN.** Both sides of the connection only need outbound access to the platform endpoint. The platform registry resolves Docker-internal, EC2 Instance Connect SSH, and Fly Machine URLs transparently. Teams don't need VPC peering or a VPN mesh to run agents across cloud providers.

---

## Try It

To delegate work to another workspace, you need the workspace ID and a valid bearer token. Start with the [Remote Workspaces docs](/docs/blog/2026-04-20-remote-workspaces/) to provision a workspace, then check the [A2A Protocol reference](/docs/api-protocol/a2a-protocol/) for the full JSON-RPC method spec.

If you're evaluating agent frameworks and want to understand what peer-to-peer A2A actually looks like in production — not in a demo — the [A2A Enterprise post](/blog/a2a-enterprise-any-agent-any-infrastructure) covers the governance and audit trail layer that the protocol spec doesn't address.
