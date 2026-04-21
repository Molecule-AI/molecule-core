# A2A Protocol v1 — Deep-Dive Content Brief
**Owner:** Marketing Lead | **Authoring:** Content Marketer
**Source:** PMM + A2A Protocol docs (docs/api-protocol/a2a-protocol.md)
**Status:** BRIEF — ready for Content Marketer to execute
**Timeline:** Execute before LangGraph A2A GA announcement (Q2-Q3 2026)
**Urgency:** HIGH — 72h window to publish before LangGraph GA ships their A2A narrative

---

## Background

Molecule AI shipped A2A (Agent-to-Agent) protocol GA in Phase 30 (2026-04-20). This is a deep-dive technical content piece designed to:

1. **Own the A2A narrative** before LangGraph GA ships their competing A2A story
2. **Educate developers** on how Molecule AI's A2A works at the protocol level
3. **Differentiate on architecture** — platform NOT in the message path, peer-to-peer workspaces

LangGraph A2A GA targeting Q2-Q3 2026. The window to establish Molecule AI as the canonical A2A reference is open **now**.

---

## Content Goal

Position Molecule AI's A2A implementation as the most architecturally clean peer-to-peer agent communication protocol available — with specific technical depth that makes it the reference implementation developers point to.

---

## Target Audience

- **Primary:** AI/ML engineers evaluating agent frameworks, building multi-agent systems
- **Secondary:** Platform engineers, DevOps leads evaluating agent orchestration infrastructure
- **Tertiary:** LangChain/CrewAI users evaluating alternatives
- **Tone:** Technical depth. Code-first. This is a protocol explainer, not a feature announcement.

---

## Target Keywords

| Priority | Keyword | Intent |
|----------|---------|--------|
| P0 | "A2A protocol" | Informational — own the canonical definition |
| P0 | "agent-to-agent protocol" | Informational — broader intent |
| P1 | "Molecule AI A2A" | Brand + technical |
| P1 | "A2A vs A2A" / "MCP vs A2A" | Comparison — capture migration queries |
| P2 | "multi-agent communication" | Informational |

---

## Content Angle

**Title:** How Molecule AI's A2A Protocol Works: Peer-to-Peer Agent Communication

**Core argument:** Most agent-to-agent communication is hub-and-spoke — all messages route through a central orchestrator. Molecule AI's A2A is peer-to-peer. The platform handles discovery. Messages go workspace-to-workspace. The platform is never in the message path.

**Why this matters:** Hub-and-spoke introduces latency, a single point of failure, and a dependency on the platform's availability for every agent-to-agent call. Peer-to-peer means agents communicate directly — the platform orchestrates, but doesn't proxy.

---

## Content Outline

### 1. Intro — The Multi-Agent Communication Problem
Why agents need to talk to each other (specialized agents, task decomposition, distributed workflows). Why most implementations are hub-and-spoke. What peer-to-peer changes.

**~200 words**

### 2. How Molecule AI's A2A Works — Architecture Deep-Dive
Walk through the actual protocol flow:
1. Workspace A decides to delegate to Workspace B
2. Workspace A asks platform: `GET /registry/discover/:id` (with `X-Workspace-ID` header)
3. Platform checks `CanCommunicate()` permission
4. Platform returns B's URL (Docker-internal or host-mapped, depending on caller type)
5. Workspace A sends JSON-RPC 2.0 message **directly** to Workspace B — no platform in the path
6. Workspace B streams SSE progress, returns artifacts when done

Show the JSON-RPC message format. Show the Redis key resolution. Show the permission check.

**~400 words + code examples**

### 3. The Discovery Model — On-Demand, Not Pushed
Why topology is not pushed at startup (topology changes while agents run; push-at-startup requires constant re-push). Molecule AI resolves peer URLs on-demand — an agent only asks for another agent's URL at the moment it decides to delegate.

**~200 words**

### 4. Authentication — Discovery-Time Validation
How `CanCommunicate()` permission checking works at discovery. MVP: unauthenticated direct calls post-discovery (Docker network isolation). Post-MVP: short-lived signed tokens scoped to caller/target pair.

**~200 words**

### 5. Task Lifecycle
Walk through the task lifecycle: task creation → progress streaming via SSE → artifact return. Show what an SSE stream looks like.

**~200 words**

### 6. What This Means for Developers
Practical implications: lower latency (no platform proxy), fault isolation (workspace failure doesn't cascade through orchestrator), platform independence (A2A works as long as workspaces can reach each other).

**~150 words**

### 7. CTA
Link to A2A protocol docs, workspace API reference, Phase 30 Remote Workspaces docs.

---

## Competitive Framing

**LangGraph A2A comparison (include in body, don't lead with):**

LangGraph's A2A GA (when it ships) will compete directly. Key architectural difference:

| | Molecule AI A2A | LangGraph A2A |
|---|---|---|
| Message routing | Peer-to-peer — platform not in path | TBD — likely platform-mediated |
| Discovery | On-demand, permission-checked at resolve | TBD |
| Authentication | CanCommunicate() at discovery + post-MVP signed tokens | TBD |
| Phase 30 ship date | **GA since 2026-04-20** | Targeting Q2-Q3 2026 |

Frame: Molecule AI's A2A has been GA for weeks. The protocol is spec'd, shipped, and documented. When LangGraph ships, developers will compare — make sure the architectural difference is already understood.

---

## Format

- **Type:** Technical blog post / deep-dive
- **Length:** ~1,400 words
- **Code blocks:** 4–5 (JSON-RPC examples, SSE stream example, Redis key resolution)
- **Screenshots/diagrams:** Architecture diagram showing peer-to-peer vs hub-and-spoke

---

## CTA Links

| Link | Value |
|------|-------|
| A2A Protocol docs | /docs/api-protocol/a2a-protocol.md |
| Remote Workspaces | /docs/guides/remote-workspaces.md |
| GitHub | github.com/Molecule-AI/molecule-core |

---

## Dependencies

- **This brief:** Marketing Lead (done)
- **Content draft:** Content Marketer
- **Code review:** DevRel Engineer (verify JSON-RPC examples, Redis key names)
- **Architecture diagram:** Social Media Brand or DevRel
- **Legal review:** None expected (technical explainer, no customer data)

---

## Success Metrics

- SERP position for "A2A protocol" — target #1 within 2 weeks
- SERP position for "agent-to-agent protocol" — target top 3
- Referral from LangChain/CrewAI comparison queries
- GitHub Discussion / community engagement

---

## Timing

Publish before LangGraph A2A GA ships. Even a rough draft published a week before LangGraph GA establishes first-mover SEO advantage. Coordinate with DevRel for community launch (HN, Reddit, LinkedIn) on publish day.

---

*Brief by Marketing Lead based on PMM brief + A2A protocol docs. Content Marketer to draft. DevRel to review code examples.*
