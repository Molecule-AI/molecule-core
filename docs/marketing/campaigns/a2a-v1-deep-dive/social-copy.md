# A2A v1 Deep-Dive — Social Copy
Campaign: a2a-v1-deep-dive | Blog: `docs/blog/2026-04-22-a2a-v1-deep-dive/`
Slug: `a2a-protocol-deep-dive`
Publish day: Coordinate with Marketing Lead (Day 3/4, Apr 22–23)
Assets: None required — code-first posts
Hashtags: #A2A #MCP #AIAgents #AgenticAI #MoleculeAI
UTM: `?utm_source=twitter&utm_medium=social&utm_campaign=a2a-protocol-deep-dive`

---

## X Thread — 5 posts

### Post 1 — Hook (protocol problem)
Hub-and-spoke agent communication works great — until it doesn't.

Every agent-to-agent call through a central orchestrator adds latency, a failure point, and an audit log that only shows "orchestrator → agent B."

Molecule AI's A2A is peer-to-peer. The platform handles discovery. Messages go workspace-to-workspace.

Here's how it actually works:

→ https://docs.molecule.ai/blog/a2a-protocol-deep-dive

---

### Post 2 — The four methods
A2A is JSON-RPC 2.0 over HTTP. Four methods covers 95% of use cases:

• message/send — submit task, get result
• message/sendSubscribe — submit + SSE progress stream
• tasks/get — retrieve task state
• tasks/cancel — wire-level interrupt

Real payloads:

```json
{
  "jsonrpc": "2.0",
  "method": "message/sendSubscribe",
  "params": {
    "message": {
      "messageId": "msg_01hx9fk3z2k7...",
      "parts": [{ "type": "text", "text": "Run the security audit" }]
    },
    "metadata": {
      "callerWorkspaceId": "ws_01hx8abcd...",
      "orgApiKeyPrefix": "mole_a1b2"
    }
  }
}
```

→ https://docs.molecule.ai/blog/a2a-protocol-deep-dive

---

### Post 3 — Discovery
How does Agent A find Agent B?

A calls: GET /registry/discover/:workspace_b_id
Platform checks: CanCommunicate(A, B)?
Platform returns: B's reachable URL

Then A sends the JSON-RPC message directly to B.

The platform is never in the message path. Discovery is on-demand — A asks for B's URL at the moment it decides to delegate, not at startup.

→ https://docs.molecule.ai/blog/a2a-protocol-deep-dive

---

### Post 4 — SSE streaming + cancellation
The task lifecycle in Molecule AI A2A:

1. Caller submits task with idempotency key
2. Target streams progress via SSE (Server-Sent Events)
3. Caller receives artifacts as they're generated
4. Caller sends tasks/cancel — target stops mid-execution

tasks/cancel isn't just a flag. It's a wire-level interrupt. The target workspace stops the agent mid-run when it receives the signal.

→ https://docs.molecule.ai/blog/a2a-protocol-deep-dive

---

### Post 5 — Cross-infrastructure
The practical blocker for multi-cloud agent orchestration is usually networking, not the protocol.

Molecule AI's A2A registry resolves Docker-internal, EC2 Instance Connect SSH, and Fly Machine URLs transparently. Neither side needs a direct network path to the other.

Both sides only need outbound access to the platform endpoint.

No VPN. No VPC peering. No mesh.

→ https://docs.molecule.ai/blog/a2a-protocol-deep-dive

---

## LinkedIn — Single Post

**Title:** How Molecule AI's A2A protocol actually works — with real JSON payloads

Most agent-to-agent communication explainers stay at the architecture level. Here's the protocol.

The A2A (Agent-to-Agent) protocol reached v1.0, and every major agent platform is scrambling to add support. Most are bolting it on.

What does it look like to build A2A as a first-class architectural primitive — not a feature?

The key insight: in Molecule AI, the platform handles *discovery*, not *routing*. When Agent A wants to delegate to Agent B, it asks the platform registry for B's reachable URL. The registry checks that A is authorized to call B, returns the URL, and steps out of the message path entirely.

Agent A then sends a JSON-RPC 2.0 message directly to Agent B's workspace. The platform doesn't proxy the message. It never appears in the latency path.

This is the architectural difference between "A2A support" and "A2A as a platform primitive." The former routes through the control plane. The latter uses the control plane for permission-gated discovery and then gets out of the way.

The practical implications for platform teams:

→ Lower delegation latency (no control plane proxy)
→ Fault isolation (workspace failure doesn't cascade through an orchestrator)
→ Cross-infrastructure delegation without a VPN (both sides only need outbound access to the platform endpoint)
→ Wire-level task cancellation (tasks/cancel stops compute mid-run, not just flags a task for future cancellation)

For teams evaluating agent frameworks: the protocol spec is the same everywhere. The implementation differences — permission model, audit trail, cancellation semantics, discovery architecture — are where production readiness lives.

Deep-dive with real JSON payloads in the post.

→ https://docs.molecule.ai/blog/a2a-protocol-deep-dive

---

**Hashtags:** #A2A #MCP #MultiAgent #AgenticAI #MoleculeAI #PlatformEngineering #AIInfrastructure
**CTA:** Bookmark for when you're evaluating agent frameworks — the protocol explainer covers what the spec doesn't.
