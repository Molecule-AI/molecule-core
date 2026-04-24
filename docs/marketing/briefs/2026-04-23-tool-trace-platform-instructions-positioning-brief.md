# Tool Trace + Platform Instructions — Positioning Brief
**Source:** PR #1686 (`feat: tool trace + platform instructions`, merged 2026-04-23)
**Date:** 2026-04-23
**Author:** PMM
**Status:** APPROVED — cleared for Content Marketer and Social Media Brand use
**Gate for:** Phase 34 GA launch (April 30, 2026); Content Marketer launch copy by April 28

---

## 1. ICP for Each Feature

### Tool Trace — Who Benefits Most

**Primary ICP: Platform Engineering / DevOps / SRE leads**
These are the people who get paged when an agent breaks in production. They own the runtime, not the agent logic. They need to answer "what did the agent actually do?" without adding instrumentation to every agent they run.

- Platform engineers running agents at scale (10+ concurrent workspaces)
- SREs and DevOps leads responsible for agent fleet reliability
- Debugging-focused builders who want visibility without a third-party SDK
- Enterprise IT teams reviewing agent behavior during compliance audits

**Secondary ICP: Developer advocates and technical evaluators**
When demonstrating agent reliability to prospective enterprise buyers, Tool Trace provides concrete proof of execution — a structured trace is more credible than a verbal explanation of what an agent "should have done."

**Not the ICP:** Individual developers iterating on agent prompts (observability matters less at 1-2 agents; matters enormously at fleet scale).

---

### Platform Instructions — Who Benefits Most

**Primary ICP: Enterprise IT, Security/Compliance, and CISO-adjacent leads**
These buyers have a governance problem: agents are running in production, and they need to enforce behavioral rules across the entire org without modifying agent code. The requirement is pre-execution guardrails, not post-hoc filtering.

- Multi-team organizations where different teams need different behavioral constraints
- Compliance-conscious deployments (SOC 2, SOX, ISO 27001 environments)
- Platform teams that own the runtime but don't own individual agent codebases

**Secondary ICP: Platform resellers and marketplace operators**
If you provision agent platforms for end customers, Platform Instructions lets you enforce per-customer behavioral boundaries — the same feature that matters for enterprise IT matters for multi-tenant platform operators who need to enforce governance across tenant boundaries.

**Not the ICP:** Single-team deployments or prototype-stage environments where governance isn't a production requirement yet.

---

## 2. Primary Buyer Benefit Statements

### Tool Trace

> **Platform teams get complete execution visibility because every A2A response carries a structured trace of every tool the agent called — inputs, outputs, and run_id-paired parallel calls — with no SDK, no pipeline, and no instrumentation required.**

**Why it holds:** The trace is inside every A2A response as `Message.metadata.tool_trace`. It's not a separate polling endpoint or a sidecar service. There's nothing to install, no API key to rotate, no version drift when the agent framework updates. Platform teams get production-grade observability the same way they get the agent itself — by running on Molecule.

---

### Platform Instructions

> **Compliance and security teams enforce org-wide agent governance at the system prompt level because rules are prepended to the agent's system prompt at workspace startup — before the first token is generated, not after an incident.**

**Why it holds:** Platform Instructions are workspace-scoped config rules fetched and applied at workspace startup. When a workspace starts, Molecule AI resolves all applicable global + workspace-specific instructions and prepends them to the system prompt. The agent receives governance as context, not as a gate — which means it shapes the agent's reasoning from the start, not as a filter applied after the agent has already acted.

---

## 3. Competitive Framing

### LangGraph Cloud

**Observability approach:** LangGraph ships LangSmith integration as the recommended observability path. LangSmith is a first-party Anthropic/Microsoft product with strong cross-platform LLM observability (token usage, latency, model-level traces, evaluation). It requires:
- An active LangSmith account and API key
- SDK-level instrumentation (`from langsmith import trace` per agent)
- A separate vendor relationship and data pipeline

**Molecule AI differentiator:** Tool Trace captures *A2A-level agent behavior* (tool call sequences, input/output previews, run_id-paired parallel execution) — not just model-level token counts. LangSmith tracks what the model did; Tool Trace tracks what the agent did. For teams running on Molecule, Tool Trace is the Molecule-specific observability layer inside their existing stack, not a replacement for LangSmith but a complement that fills the agent-behavior gap LangSmith doesn't capture.

**Governance approach:** LangGraph's policy enforcement is primarily runtime-level (via LangGraph's guardrail primitives). No equivalent to workspace-scoped system prompt injection at startup is documented in their public SDK or enterprise docs as of April 2026.

---

### CrewAI

**Observability approach:** CrewAI's observability story is centered on third-party integrations — LangSmith, Weights & Biases, and custom callbacks. Their native tracing is minimal: task-level status and output logging, but no structured tool-call-level trace inside the A2A response. CrewAI agents require manual instrumentation to get observability data into an external system.

**Molecule AI differentiator:** Tool Trace ships inside every A2A response — there's no instrumentation step, no callback to configure, no external integration to set up. For CrewAI teams evaluating a move to Molecule, the observability story is "same visibility, zero setup." For teams already on Molecule, Tool Trace means they don't need to add LangSmith just to see what their agents are doing.

**Governance approach:** CrewAI has team-role primitives (manager, agent-level role assignment) but no org-level system prompt injection for governance. Platform Instructions fills a gap that CrewAI's team coordination features don't address — policy enforcement at the platform level, applied across all agents in an org.

---

### Molecule AI Differentiation Summary

| | LangGraph Cloud | CrewAI | Molecule AI (Phase 34) |
|---|---|---|---|
| Tool-level trace in response | ❌ LangSmith SDK required | ❌ Manual callbacks only | ✅ Built into every A2A response |
| Agent-level behavior visibility | ⚠️ Model traces only | ❌ Task-level only | ✅ Tool call sequences + run_id pairing |
| Platform-native (no SDK) | ❌ Requires LangSmith SDK | ❌ Requires custom integration | ✅ Zero config |
| Org-level governance layer | ❌ Runtime guardrails only | ❌ Team roles only | ✅ System prompt injection at startup |
| Governance without code deploy | ❌ Requires guardrail config | ❌ Requires role config | ✅ Workspace-scoped API config |

---

## 4. Connection to Phase 34 Narrative

### How Tool Trace + Platform Instructions Strengthen the Partner API Keys Story

Partner API Keys (`mol_pk_*`) solve the provisioning problem for platform builders: "How do I programmatically create and manage Molecule AI orgs without a browser session?"

Tool Trace and Platform Instructions solve what comes *after* provisioning: "How do I observe and govern the agents running inside the orgs I provision?"

The complete partner platform story:
1. **Partner API Keys** → programmatically provision tenant orgs via `POST /cp/admin/partner-keys`
2. **Platform Instructions** → enforce per-tenant behavioral governance at the system prompt level — partners can set governance rules for the tenants they provision
3. **Tool Trace** → full observability into what those tenant agents are actually doing — partners can offer this as a value-add to their end customers

Together, `mol_pk_*` + Platform Instructions + Tool Trace = the first agent platform that gives platform builders the full stack: provisioning, control, and observability in one API surface.

---

### Combined Phase 34 Message

> **"Molecule AI gives platform builders observability, control, and provisioning in one stack."**

This is the Phase 34 headline for enterprise and partner audiences. The three features form a coherent narrative:

- **Provisioning** (Partner API Keys): Create and manage orgs via API — no browser required
- **Control** (Platform Instructions): Enforce behavioral governance at the system prompt level — no code deploy required
- **Observability** (Tool Trace): See exactly what every agent did — no SDK required

For platform teams evaluating whether Molecule is enterprise-ready, this stack answers the three questions that come up in every procurement conversation: "Can we provision it programmatically?", "Can we enforce policy?", "Can we see what's happening?"

---

### GA Date: April 30, 2026

This brief should be used to power launch copy by **April 28** (T-2 days before GA).

- **April 24–25:** Content Marketer drafts blog post, social thread using this brief
- **April 25–26:** PMM review + approval
- **April 26–27:** Social Media Brand queued with approved copy
- **April 28:** All launch copy finalized and staged
- **April 29:** QA review pass
- **April 30:** GA — all posts go live

---

## 5. Approved Language

### One Approved Lead Claim (for Content + Social teams)

> **"Molecule AI ships built-in execution tracing and governance for every agent — so platform teams see exactly what their agents did, and compliance teams enforce what agents should do, before the first token is generated."**

This is the single approved lead claim for external copy (blog post intros, social headlines, launch announcement). Do not modify the mechanism language ("execution tracing," "before the first token is generated"). Do not add "GA" or "generally available" — use "now in beta" or "shipping today" per Phase 34 copy guardrails.

**Safe alternatives for shorter contexts:**
- "Built-in agent observability — nothing to bolt on."
- "Governance before agents run. Trace after they finish."
- "See every tool call. Control every behavior. No SDK required."

---

## Copy Guardrails

### Required language

| Feature | Label | When to use |
|---------|-------|-------------|
| Tool Trace | **Now in beta** | All external copy |
| Platform Instructions | **Now in beta** | All external copy |
| All Phase 34 features | Do NOT say "GA" or "generally available" | Hard stop — use "now in beta," "shipping today," or "in early access" |

### Prohibited framings

| Prohibited | Why | Replace with |
|-----------|-----|-------------|
| "Tool Trace is like Langfuse" | Different layer — agent vs. model | "Built-in execution tracing, no SDK required" |
| "Platform Instructions replaces OPA/Sentinel" | Complementary, not competitive | "Works alongside runtime policy engines" |
| "Message.metadata" or "JSONB" | Internal implementation detail | Never in external copy |
| "Acme Corp" or any real company | Not confirmed | "an early design partner" |
| "GA" or "generally available" | Not confirmed for these features | "Now in beta" |

---

## Asset Checklist

| Asset | Owner | Status | Notes |
|-------|-------|--------|-------|
| This positioning brief | PMM | ✅ READY | Cleared for Content + Social use |
| Tool Trace blog post | Content Marketer | ✅ Staged | `docs/blog/2026-04-23-tool-trace-observability/` |
| Platform Instructions blog post | Content Marketer | ✅ Staged | `docs/blog/2026-04-23-platform-instructions-governance/` |
| Combined post (tool trace + platform instructions) | Content Marketer | ✅ Staged | `docs/blog/2026-04-23-tool-trace-platform-instructions/` |
| Phase 34 social copy | Social Media Brand | ✅ DRAFT | `docs/marketing/social/2026-04-26-phase34-ga-launch/` |
| DevRel demo package | DevRel | ⏳ In PR #1878 | Tool Trace + Platform Instructions demo (PR #1686) |
| GA launch approval | Marketing Lead | ⏳ Pending | April 30 — all assets ready by April 28 |

---

## Phase 30 → Phase 34 Cross-Sell (for sellers)

> "Phase 30 shipped per-workspace auth tokens (`mol_ws_*`) and cross-network agent delegation. Phase 34 ships Tool Trace (observability) and Platform Instructions (governance). Together: the first agent platform with enterprise-grade provisioning, control, and observability in one stack."

---

*PMM drafted 2026-04-23 — Issue #1895*
*Approved for Content Marketer and Social Media Brand use.*
*Source: PR #1686 (`feat: tool trace + platform instructions`, merged 2026-04-23)*