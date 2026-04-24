# Phase 34 — Positioning One-Pager
**Feature group:** Partner API Keys, Tool Trace, Platform Instructions, SaaS Federation v2
**GA date:** April 30, 2026
**Status:** INTERNAL DRAFT — for PMM review and press kit use
**Owner:** PMM
**Last updated:** 2026-04-23

---

## One-Sentence Positioning Statement

Molecule AI Phase 34 gives enterprise teams the platform-native primitives — programmable access, built-in observability, and pre-execution governance — required to run AI agents in production, without the bolt-on integrations that add latency, maintenance burden, and security gaps.

---

## Target Audience

| | Role | What they care about |
|--|------|----------------------|
| **Primary** | Platform Engineering / DevOps leads | Shipping reliable agent infrastructure: observability, CI/CD integration, multi-environment support |
| **Primary** | Enterprise IT / Security Governance | Controlling agent behavior before it happens: policy enforcement, audit trails, compliance |
| **Secondary** | Partner / Marketplace integrations engineers | Embedding Molecule AI as the orchestration layer for their platform or marketplace |
| **Secondary** | Developer advocates / DevRel | Demonstrating enterprise-grade capabilities to prospective enterprise buyers |

---

## Problem We Solve

Enterprise teams adopting AI agents face three compounding failures at once:

1. **Observability gaps** — Agents run and produce outputs, but teams have no structured record of *what the agent actually did*: which tools it called, with what inputs, in what order. Debugging is reverse-engineering from outputs. Cross-platform observability (Langfuse, Datadog) adds a pipeline but misses A2A-level agent behavior.

2. **Governance gaps** — Agent behavior policies are enforced *after* the agent has already acted — filtering outputs, blocking writes post-hoc. Governance that only works after the fact is governance that failed. Enterprise IT and compliance teams need controls that shape behavior *before* the first token is generated.

3. **Integration gaps** — Platforms that want to embed agent orchestration programmatically face a choice between building it themselves (months of work) or using browser sessions (brittle, non-programmatic). CI/CD teams need ephemeral test orgs per PR. Neither is solved by existing agent platforms.

---

## Our Solution — Phase 34 Angle

Phase 34 ships four features that address each failure at the platform layer — not as integrations, not as SDKs, not as post-hoc configuration:

- **Partner API Keys** (`mol_pk_*`) — Scoped, revocable API tokens that let partner platforms, CI/CD pipelines, and marketplace resellers programmatically provision and manage Molecule AI orgs. No browser. No manual handoff.
- **Tool Trace** — `tool_trace[]` in every A2A `Message.metadata`. A structured, run_id-paired execution record: tool name, inputs, output previews, timing. No SDK, no sidecar, no sampling.
- **Platform Instructions** — Workspace-scoped system prompt rules that take effect at startup. Governance happens before the first turn, not after an incident.
- **SaaS Federation v2** — Multi-tenant control plane architecture: isolated orgs, cross-tenant guardrails, centralized billing for enterprise and marketplace deployments.

**The Phase 34 angle:** These four features work together. A partner platform provisions an org via Partner API Keys, configures Platform Instructions for their tenants, gets full observability via Tool Trace, and operates it all inside a SaaS Federation v2 multi-tenant control plane. This is a coherent enterprise stack — not four unrelated features.

---

## Key Differentiators vs. Competitors

| Differentiator | LangGraph Cloud | CrewAI | Molecule AI Phase 34 |
|---------------|----------------|--------|----------------------|
| Built-in agent observability (no SDK) | ❌ | ❌ | **✅ Tool Trace** |
| Pre-execution governance (system prompt level) | ❌ | ❌ | **✅ Platform Instructions** |
| Programmatic partner org provisioning | ❌ (seat licensing only) | ❌ (marketplace listing only) | **✅ Partner API Keys** |
| CI/CD-native ephemeral orgs | ❌ | ❌ | **✅ Partner API Keys + CI/CD example** |
| Multi-tenant SaaS control plane | ❌ | ❌ | **✅ SaaS Federation v2** |
| A2A-native protocol | ✅ (in-progress, Q2-Q3 2026) | ❌ | **✅ live today** |

**Counter-framing for sellers:**
> "LangGraph Cloud and CrewAI are end-user platforms. Molecule AI is infrastructure your platform builds on — with the governance and observability built in, not bolted on."

---

## Proof Points

| Claim | Evidence |
|-------|----------|
| Molecule AI is the only agent platform with built-in execution tracing | `tool_trace[]` in `Message.metadata` — no SDK, no sidecar. LangGraph and CrewAI require Langfuse/Helicone instrumentation. |
| Platform Instructions enforce governance before agents run | Workspace startup path prepends rules to system prompt. Policy takes effect before first token generated. |
| Partner API Keys enable programmatic org provisioning | `POST /cp/admin/partner-keys` creates orgs via API. Keys are SHA-256 hashed, org-scoped, rate-limited, revocable via `DELETE`. |
| Ephemeral test orgs per PR are fully automated | CI/CD example in partner onboarding guide: `POST` create → run tests → `DELETE` teardown. No manual cleanup, no shared-state contamination. |
| SaaS Federation v2 enables multi-tenant isolation | Tutorial at `docs/marketing/launches/pr-1613-saas-federation-v2.md`. Org-scoped keys + control plane boundary. |
| Design partner (Acme Corp) validates enterprise readiness | Acme Corp integration (design partner, name pending PM confirmation). Reference use case: partner-provisioned orgs for Acme's customer base. |

---

## Internal Use Notes

> **2026-04-23 override:** Internal notes previously flagged all Phase 34 features as BETA. Community FAQ (`phase-34-community-faq.md`, Community Manager-owned, approved) and approved social copy are the authoritative external-facing sources. Updated to reconcile:
> - **Partner API Keys:** GA April 30, 2026 — "generally available" language is correct per community FAQ table and approved social copy.
> - **Tool Trace:** GA — community FAQ uses no Beta designation. All plans.
> - **Platform Instructions:** GA — Enterprise plans only. Confirmed via code review: AdminAuth-gated at router.go:376. Community FAQ updated accordingly. Enterprise IT is primary ICP — plan gate is consistent with buyer audience, not a hidden limitation.
> - **SaaS Federation v2:** REMOVED from community announcement and social copy pending PM confirmation. Do not reference in external materials.
> - Do not use "Acme Corp" in any externally published copy — placeholder only. Confirm partner name with PM before press release.
> - Phase 30 linkage: Phase 30 shipped `mol_ws_*` (per-workspace auth). Phase 34 extends to `mol_pk_*` (partner-level keys). Cross-sell: "Phase 30 workspace isolation + Phase 34 partner scoping — the only platform with both."
