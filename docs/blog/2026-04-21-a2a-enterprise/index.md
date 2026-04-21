---
title: "Any Agent, Any Infrastructure: A2A Communication for Enterprise AI Teams"
date: 2026-04-21
slug: a2a-communication-enterprise
description: "The A2A protocol lets AI agents talk to each other across any infrastructure. Here's why that's a bigger deal than it sounds — and what separates a working A2A implementation from one that enterprises can actually trust."
tags: [A2A, enterprise, multi-agent, governance, protocol]
---

# Any Agent, Any Infrastructure: A2A Communication for Enterprise AI Teams

There's a moment every platform engineer eventually hits: your AI agents need to talk to each other, and you're suddenly managing a topology of services that don't natively know each other exists.

You've got a LangGraph agent in one environment. A Claude Code agent in another. A workflow runner somewhere else. They're all doing useful work. But the moment Agent A needs to ask Agent B to do something — and get a real answer back — you're duct-taping HTTP endpoints, managing authentication tokens, and writing custom polling loops.

That's the problem the A2A (Agent-to-Agent) protocol was built to solve. And if you're evaluating AI agent platforms today, whether or not they implement A2A — and how they implement it — is one of the most consequential architectural decisions you'll make.

## What A2A Actually Means

A2A is a JSON-RPC 2.0 protocol for direct communication between AI agents. The core idea is simple: instead of routing every inter-agent message through a central hub, agents expose a well-defined interface (`message/send`, `message/sendSubscribe`, `tasks/cancel`) that any other A2A-compliant agent can call.

The transport is HTTP. The discovery mechanism is an Agent Card at `/.well-known/agent-card.json`. The message format is JSON-RPC 2.0. Every implementation that follows the spec should be interoperable with every other.

In practice, that means: if Agent A ships with A2A support and Agent B ships with A2A support, they can communicate without you writing custom integration code. You configure the hierarchy once in your platform, and the agents figure out how to reach each other.

This matters because the alternative — custom point-to-point integrations — doesn't scale. A team with 10 agents has 90 possible agent-pair combinations. A team with 20 has 190. You can't hand-roll all of them.

## The Infrastructure Problem

Here's where most A2A implementations stop: they work in a single environment. Two agents on the same network, talking directly, no NAT, no firewall, same Docker network — that's tractable.

But enterprise AI deployments rarely live in one place. You have agents running in cloud VMs, on-premises servers, behind different VPNs, in different cloud providers. The moment you try to apply a "connect agents" A2A story to that topology, most implementations fall apart.

Molecule AI's A2A implementation is designed for cross-infrastructure communication from the ground up. The platform handles discovery across network boundaries. A workspace in AWS can delegate to a workspace in GCP. The protocol is the same; the infrastructure differences are abstracted away by the platform's registry layer.

This is the "Any Agent, Any Infrastructure" frame: A2A connectivity isn't limited to agents in the same VPC.

## The Governance Gap (and Why It Matters)

Here's the part most A2A coverage skips over: connecting agents is the easy part. The hard part is knowing what they did.

Consider the A2A landscape today: several frameworks ship an inbound server and an outbound client — agents can discover each other, send messages, and complete tasks. That's real progress.

What's often absent from reference implementations: a governance layer. There's no standardized attribution format for which credential made which call, no audit trail contract, and no revocation model when a credential is compromised.

That gap matters enormously in enterprise contexts. When your compliance team asks "which agent accessed which workspace, and what did it do with the data?", a system that connects agents without logging them is answering a different question than the one being asked.

Molecule AI's A2A implementation includes org API key attribution on every call. The audit log records which key prefix made which request, when, and with what result. If you need to revoke an integration, you revoke the key — the agent loses access immediately, and the audit trail shows you exactly what it did before revocation.

The difference sounds abstract. It's not:

| | Connect agents | Connect agents with governance |
|---|---|---|
| Audit trail | ❌ | ✅ |
| Attribution per call | ❌ | ✅ |
| Instant revocation | ❌ | ✅ |
| Compliance-ready | ❌ | ✅ |
| Cross-infrastructure | Depends on impl | ✅ |

## How It Works in Practice

The discovery flow in Molecule AI:

1. **Workspace A** decides to delegate a task to Workspace B
2. Workspace A calls `GET /registry/discover/:workspace-b-id` with its `X-Workspace-ID` header
3. The platform checks `CanCommunicate()` — the caller is allowed to reach this target
4. The platform returns B's URL (either Docker-internal for other workspaces, or host-mapped for canvas/external callers)
5. Workspace A sends an A2A JSON-RPC message **directly to Workspace B** — the platform is not in the message path after discovery
6. Workspace B processes the task, streams progress updates via SSE, and returns artifacts on completion

The platform stays out of the message path by design. Discovery is platform-mediated for security; the actual communication is direct workspace-to-workspace. This means lower latency, no single-point-of-bottleneck, and natural horizontal scaling.

The task lifecycle is explicit:

```
submitted → working → completed
                    → failed
                    → canceled
           → input-required → working (caller provides follow-up)
```

Streaming works via `message/sendSubscribe` — the caller receives SSE events as the task progresses, and a terminal event (`completed`, `failed`, or `canceled`) tells it when the task is done. No polling.

## The Hierarchy Model

A2A doesn't mean every agent talks to every other agent. In Molecule AI, `CanCommunicate()` enforces a hierarchy:

- **Same workspace** — agents in the same container can always communicate
- **Siblings** — agents with the same parent can always communicate
- **Parent ↔ child** — a PM can always reach its sub-agents and vice versa
- **Everything else** — denied by default

This is intentionally restrictive. An agent that can reach any other agent in the fleet is a lateral-movement risk. The hierarchy model means you can scope communication rights the same way you scope access controls in traditional infrastructure.

## What Enterprises Should Look For

If you're evaluating A2A support across platforms, here's the checklist:

**Protocol basics:**
- [ ] JSON-RPC 2.0 message/send, message/sendSubscribe, tasks/cancel
- [ ] Agent Card discovery at `/.well-known/agent-card.json`
- [ ] SSE streaming for long-running tasks

**Enterprise requirements:**
- [ ] Audit trail on every cross-agent call
- [ ] Attribution (which key/credential made which call)
- [ ] Instant revocation without redeployment
- [ ] Hierarchy enforcement (not flat agent-to-agent mesh)
- [ ] Cross-infrastructure discovery (not limited to same network)

**Operational:**
- [ ] Task state visible in canvas (working, completed, failed, canceled)
- [ ] Activity logs exported for compliance review
- [ ] No polling — SSE streams provide real-time state

A2A without the enterprise requirements is a developer convenience feature. A2A with the governance layer is an operational system that compliance teams can actually sign off on.

## Getting Started

If you're running Molecule AI, A2A is already enabled. Every workspace is an A2A server. To add a sub-agent and wire up delegation:

1. Deploy the sub-agent workspace
2. Set the parent relationship in Canvas (drag the sub-agent onto the parent)
3. The parent agent can now delegate to the child via the A2A protocol

The platform handles discovery, auth token validation, and the SSE streaming loop. Your agents write the business logic.

→ [A2A Protocol Reference](/docs/api-protocol/a2a-protocol)
→ [Canvas: Managing Workspace Hierarchy](/docs/frontend/canvas)
→ [Org API Keys: Audit Attribution Setup](/blog/org-scoped-api-keys)

---
*Molecule AI's A2A implementation ships in Phase 30. Cross-infrastructure discovery and org API key attribution are available on all production deployments.*