---
title: "A2A Protocol Deep-Dive: How the Agent-to-Agent Standard Actually Works"
date: "2026-04-22"
slug: "a2a-protocol-deep-dive"
description: "A practical, code-first walkthrough of the A2A (Agent-to-Agent) protocol — peer discovery, JSON-RPC messaging, SSE streaming, and task cancellation — and what separates a reference implementation from one that enterprises can rely on."
tags: [A2A, protocol, technical, multi-agent, JSON-RPC, enterprise]
---

# A2A Protocol Deep-Dive: How the Agent-to-Agent Standard Actually Works

*If you read our [A2A Enterprise post](/blog/a2a-communication-enterprise), here's how the protocol works under the hood.*

The A2A (Agent-to-Agent) protocol is a JSON-RPC 2.0 standard for direct communication between AI agents. Where the Model Context Protocol (MCP) connects agents to tools, A2A connects agents to each other — and it does so without routing every message through a central hub.

Most introductions to A2A focus on the "what" and the "why." This post focuses on the "how." We'll walk through the protocol's core primitives — discovery, task submission, streaming, and cancellation — with real JSON payloads you can use as a reference.

---

## The Three Primitives

A2A defines four methods. The three that cover 95% of use cases:

1. **`message/send`** — submit a task and wait for the result (synchronous)
2. **`message/sendSubscribe`** — submit a task and stream updates back (async via SSE)
3. **`tasks/cancel`** — cancel a running task

Every method is a JSON-RPC 2.0 request over HTTP. Responses follow the same spec. The transport is deliberately minimal: HTTP + JSON. If you've built a REST API, you can work with A2A.

---

## Agent Discovery: The Agent Card

Before Agent A can call Agent B, it needs to know B exists and how to reach it. A2A uses an **Agent Card** — a JSON document at `/.well-known/agent-card.json` that every compliant agent publishes.

```json
{
  "agentId": "research-agent-01",
  "name": "Research Agent",
  "description": "Handles web research, code search, and document retrieval",
  "url": "https://agent.example.com",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false
  },
  "skills": ["web-search", "code-search", "document-retrieval"]
}
```

In Molecule AI, the Agent Card is automatically maintained for every workspace. The platform's registry handles discovery — you call `GET /registry/discover/:workspace-id` and the platform returns the target's URL, with the platform staying out of the message path after discovery.

---

## Sending a Task: `message/send`

The simplest A2A interaction is a single request → single response. Agent A submits a task to Agent B and waits for the result.

**Request (Agent A → Agent B):**

```json
{
  "jsonrpc": "2.0",
  "id": "req-456",
  "method": "message/send",
  "params": {
    "message": {
      "role": "user",
      "parts": [
        {
          "type": "text",
          "text": "Research MCP server ecosystem — focus on enterprise adoption signals"
        }
      ]
    },
    "metadata": {
      "taskId": "task-789",
      "callerWorkspaceId": "pm-agent-01"
    }
  }
}
```

**Response (Agent B → Agent A):**

```json
{
  "jsonrpc": "2.0",
  "id": "req-456",
  "result": {
    "taskId": "task-789",
    "state": "completed",
    "artifacts": [
      {
        "parts": [
          {
            "type": "text",
            "text": "MCP server ecosystem research complete. Key signals: Chrome DevTools MCP (Google, 35.9k★), Slack MCP (community), GitHub MCP (community). Enterprise adoption accelerating via governance-layer differentiation."
          }
        ]
      }
    ]
  }
}
```

The task lifecycle:

```
submitted → working → completed
                    → failed
                    → canceled
           → input-required → working (caller provides follow-up)
```

`input-required` is the key edge case: if Agent B needs clarification, it pauses and signals this state. Agent A can then provide the missing context and the task resumes.

---

## Streaming: `message/sendSubscribe`

Long-running tasks don't wait for completion — they stream progress. `message/sendSubscribe` opens an SSE (Server-Sent Events) channel back to the caller.

**Request:**

```json
{
  "jsonrpc": "2.0",
  "id": "req-789",
  "method": "message/sendSubscribe",
  "params": {
    "message": {
      "role": "user",
      "parts": [
        {
          "type": "text",
          "text": "Run a full security audit on PR #2341"
        }
      ]
    }
  }
}
```

**Stream events (Agent B → Agent A, over SSE):**

```
event: task
data: {"jsonrpc":"2.0","id":"req-789","params":{"state":"working","taskId":"task-101"}}

event: task
data: {"jsonrpc":"2.0","id":"req-789","params":{"state":"working","taskId":"task-101","progress":{"message":"Scanning dependency tree..."}}}

event: task
data: {"jsonrpc":"2.0","id":"req-789","params":{"state":"working","taskId":"task-101","progress":{"message":"Running SAST checks..."}}}

event: task
data: {"jsonrpc":"2.0","id":"req-789","result":{"taskId":"task-101","state":"completed","artifacts":[{"parts":[{"type":"text","text":"Security audit complete. 3 medium findings. Full report attached."}]}]}}
```

Each `task` event carries the current state. A `completed`, `failed`, or `canceled` state is the terminal event — the SSE channel closes after it.

---

## Cancellation: `tasks/cancel`

If a task is running longer than expected or the caller no longer needs the result:

```json
{
  "jsonrpc": "2.0",
  "id": "req-012",
  "method": "tasks/cancel",
  "params": {
    "taskId": "task-101"
  }
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": "req-012",
  "result": {
    "taskId": "task-101",
    "state": "canceled"
  }
}
```

In Molecule AI, cancellation is immediate — the agent's task loop aborts and the `canceled` state propagates to the canvas. The caller gets confirmation without having to poll.

---

## A Complete Example: Orchestrator → Research Agent

Here's how it looks in practice with Molecule AI's A2A implementation:

```python
import requests

# 1. Discover the research agent
registry_url = f"{PLATFORM_URL}/registry/discover/research-agent-01"
headers = {"Authorization": f"Bearer {auth_token}", "X-Workspace-ID": "pm-agent-01"}
discovery = requests.get(registry_url, headers=headers).json()

# 2. Submit task via A2A
a2a_url = discovery["url"]
payload = {
    "jsonrpc": "2.0",
    "id": "req-456",
    "method": "message/sendSubscribe",
    "params": {
        "message": {
            "role": "user",
            "parts": [{"type": "text", "text": "Research A2A adoption in enterprise AI teams"}]
        }
    }
}
response = requests.post(a2a_url, json=payload, headers=headers, stream=True)

# 3. Stream SSE events
for line in response.iter_lines():
    if line.startswith("data: "):
        event = json.loads(line[6:])
        print(event)
```

The platform's registry handles discovery. Agent A sends directly to Agent B. The platform is not in the message path after discovery — lower latency, no bottleneck, natural horizontal scaling.

---

## The Governance Layer: What A2A Doesn't Specify

The A2A protocol defines communication primitives — it doesn't define who is accountable for what happens in a call. That's the governance layer, and it's where implementations diverge.

**What A2A specifies:**
- Discovery via Agent Cards
- Message format (JSON-RPC 2.0)
- Task lifecycle states
- Streaming via SSE
- Cancellation

**What A2A leaves undefined:**
- Authentication and authorization
- Audit attribution (which credential made which call)
- Revocation (how to cut off an agent's access immediately)
- Cross-infrastructure routing (most A2A reference implementations assume same-network agents)

Molecule AI's A2A implementation adds the governance layer on top of the protocol:

- **Discovery** is platform-mediated — the registry enforces `CanCommunicate()` hierarchy rules
- **Every call** carries the caller's org API key prefix in the audit log
- **Revocation** is immediate — one API call, the key stops working, no redeploy
- **Cross-infrastructure** is built in — a workspace in AWS can delegate to a workspace in GCP

A2A without a governance layer is a developer convenience feature. A2A with the governance layer is an operational system compliance teams can actually sign off on.

---

## Getting Started with Molecule AI A2A

Every Molecule AI workspace is an A2A server. To wire up an agent hierarchy:

1. Deploy the sub-agent workspace
2. Set the parent relationship in Canvas (drag the sub-agent onto the parent)
3. The parent agent can now delegate to the child via A2A

The platform handles discovery, auth token validation, and SSE streaming. Your agents write the business logic.

→ [A2A Protocol Reference](/docs/api-protocol/a2a-protocol)
→ [Canvas: Managing Workspace Hierarchy](/docs/frontend/canvas)
→ [A2A Enterprise Post](/blog/a2a-communication-enterprise) — the positioning angle
→ [Org API Keys: Audit Attribution Setup](/blog/org-scoped-api-keys)

---

## Competitive Table

| Feature | Molecule AI | LangGraph (in-progress) |
|---------|-------------|------------------------|
| JSON-RPC 2.0 message/send | ✅ | ✅ |
| message/sendSubscribe (SSE streaming) | ✅ | ✅ |
| tasks/cancel | ✅ | ✅ |
| Agent Card discovery | ✅ | ✅ |
| Cross-infrastructure A2A | ✅ | ❌ |
| Org API key attribution on every call | ✅ | ❌ |
| Audit trail on cross-agent calls | ✅ | ❌ |
| Instant revocation | ✅ | ❌ |
| Hierarchy enforcement (CanCommunicate) | ✅ | ❌ |
| Task state visible in canvas | ✅ | ❌ |

*LangGraph A2A (PRs #6645, #7113): inbound + outbound client shipping, no governance layer.*

---

*Molecule AI's A2A implementation ships in Phase 30. Cross-infrastructure discovery and org API key attribution are available on all production deployments.*
