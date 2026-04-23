# Phase 34 — Taglines + Messaging Matrix
**Feature group:** Partner API Keys, Tool Trace, Platform Instructions, SaaS Federation v2
**GA date:** April 30, 2026
**Owner:** PMM | **Status:** INTERNAL DRAFT
**Last updated:** 2026-04-23

---

## 3 Candidate Taglines

### Tagline A — Production-grade (emphasizes enterprise reliability)
> **"Production-grade AI agents. Nothing to bolt on."**

**Use for:** Press releases, homepage hero, paid placements, enterprise sales decks.
**Why it works:** Directly addresses the enterprise buyer's #1 objection — "this is great for prototypes but can I run it in production?" — without overclaiming features. "Nothing to bolt on" is a dig at competitors (LangGraph, CrewAI) that require Langfuse, Helicone, or custom observability pipelines.

---

### Tagline B — Observability/visibility (emphasizes transparency)
> **"See exactly what your AI agents did. Every tool. Every call. Every time."**

**Use for:** DevOps-focused channels, technical blog intros, SOC 2 / compliance audience, tool trace launch announcement.
**Why it works:** Speaks directly to the platform engineering persona — the person who gets paged at 2am when something breaks. "Every tool. Every call. Every time." is specific and falsifiable, which builds credibility with technical audiences. It names the feature (Tool Trace) without making it a product name.

---

### Tagline C — Aspirational (emphasizes enterprise enablement)
> **"Your AI fleet. Your rules. Your cloud."**

**Use for:** LinkedIn, enterprise social, brand campaigns, vision statements.
**Why it works:** Three short declarative sentences that speak to three distinct buyer anxieties: managing at scale ("fleet"), controlling behavior ("rules"), and infrastructure autonomy ("your cloud"). Works for Platform Instructions, Partner API Keys, and SaaS Federation v2 simultaneously — it's a Phase 34 group tagline, not a single-feature tagline.

---

## Messaging Matrix — 4 Features

---

### Feature 1: Partner API Keys (`mol_pk_*`)

| | |
|--|--|
| **Pain it solves** | Partner platforms, CI/CD pipelines, and marketplace resellers cannot programmatically provision or manage Molecule AI orgs — they must use browser sessions or build custom integrations from scratch. This makes Molecule AI unembeddable for any platform that wants to offer agent orchestration as a feature. |
| **Who cares** | Platform integrations engineers, DevRel leads building partner ecosystems, CI/CD DevOps teams, marketplace listing owners (AWS/GCP Marketplace) |
| **One-liner** | Programmatic org provisioning via API — no browser required, no manual handoff. |
| **Proof point** | `POST /cp/admin/partner-keys` creates a fully configured org with one API call. Keys are scoped to the org they create, rate-limited, revocable with `DELETE /cp/admin/partner-keys/:id`. Ephemeral CI test orgs: `POST` → run tests → `DELETE` → clean billing. |
| **HN/Reddit framing** | "Molecule AI now lets partners provision orgs via API — the same week Acme Corp [design partner, placeholder] ships their integration." Do NOT claim GA. Use "beta" or "now available." |
| **What to soft-pedal** | Specific partner tiers and pricing (PM not confirmed). Marketplace billing integration status (PM to confirm). Do not mention "Acme Corp" in published copy. |

---

### Feature 2: Tool Trace

| | |
|--|--|
| **Pain it solves** | When an agent breaks in production, teams have no structured record of what it did — only the final output. Reverse-engineering from outputs is slow, error-prone, and impossible to automate. Third-party observability tools (Langfuse, Helicone, Datadog) miss A2A-level agent behavior and require SDK instrumentation. |
| **Who cares** | Platform engineers, DevOps leads, SREs, enterprise IT debugging production incidents |
| **One-liner** | Built-in execution tracing for every A2A task — no SDK, no sidecar, no sampling. |
| **Proof point** | `tool_trace[]` in every `Message.metadata` — array of `{tool, input, output_preview, run_id}` entries. Entries written to `activity_logs.tool_trace` as JSONB. run_id pairs concurrent calls so parallel traces don't merge. Platform-native: ships with the A2A response, no instrumentation required. |
| **HN/Reddit framing** | Lead with the developer experience: "Tool Trace ships today in Molecule AI. Every agent turn now includes a structured record of every tool called — inputs, output previews, run_id-paired for parallel calls." Be honest: this is a beta feature. |
| **What to soft-pedal** | Technical implementation details (run_id pairing schema, JSONB storage format). Overlap with Langfuse/Helicone — frame as complementary, not competitive. |

---

### Feature 3: Platform Instructions

| | |
|--|--|
| **Pain it solves** | Agent governance that only filters outputs after the agent has already acted is governance that failed. Enterprise IT and compliance teams need to shape agent behavior *before* the first token is generated — without requiring a code change or deployment. |
| **Who cares** | Enterprise IT, Security/Compliance leads, Platform Engineering, CISO office |
| **One-liner** | Enforce org-wide agent governance at the system prompt level — before the first turn, not after an incident. |
| **Proof point** | Platform Instructions prepends workspace-scoped rules to the system prompt at startup. Two scopes: global (every workspace in the org) and workspace-specific. Rules take effect before the first agent turn — not after. Policy update requires no code deploy, no agent restart, no application change. |
| **HN/Reddit framing** | Frame as "the missing governance layer for production agents." Avoid overclaiming compliance certifications. Do not compare directly to OPA/Sentinel — say "complements runtime policy engines" not "replaces them." |
| **What to soft-pedal** | Overlap with the existing audit trail panel (Issue #759) — they are complementary (Tool Trace feeds the audit trail). Don't let buyers think they have to choose. Specific policy examples until PM confirms which are GA-ready. |

---

### Feature 4: SaaS Federation v2

| | |
|--|--|
| **Pain it solves** | Enterprises and marketplaces that need to offer agent orchestration to multiple end-customers (tenants) cannot do so safely with a single-tenant architecture: cross-tenant data isolation, centralized billing, org-level access control, and per-tenant audit trails are all required for enterprise procurement. |
| **Who cares** | Enterprise procurement, IT procurement teams, marketplace operators, SaaS resellers, multi-tenant ISVs |
| **One-liner** | Multi-tenant agent platform with cross-tenant isolation, centralized billing, and org-level governance — built for enterprises and marketplaces. |
| **Proof point** | SaaS Federation v2 tutorial at `docs/tutorials/saas-federation` (PR #1613). Org-scoped keys + control plane boundary. Isolated per-tenant workspaces with centralized admin view. |
| **HN/Reddit framing** | ⚠️ **WARNING:** SaaS Federation v2 is listed in Issue #1836 as a Phase 34 feature, but no PMM positioning brief or blog post exists for it yet. Do NOT draft community copy for this feature until PM confirms: (a) what it actually ships, (b) the GA/beta/alpha label, and (c) the primary use case narrative. Current content gap — not ready for external copy. |
| **What to soft-pedal** | Until PM confirms details, do not publish any claims about SaaS Federation v2. |

---

## Feature Cross-Sell Angles

**Phase 30 → Phase 34 linkage (for sellers):**
> "Phase 30 shipped per-workspace auth tokens (`mol_ws_*`). Phase 34 ships partner-level keys (`mol_pk_*`). Together, Molecule AI is the only platform with workspace-level isolation *and* partner-level scoping — enterprise-ready from day one."

**Governance stack (Platform Instructions + Tool Trace):**
> "Platform Instructions shapes what agents do *before* they run. Tool Trace records what they did *after*. Together: governance before, observability after. Nothing leaves production unaccounted for."

**Partner platform stack (Partner API Keys + SaaS Federation v2 + Platform Instructions):**
> "Provision tenants via API. Isolate them in a multi-tenant control plane. Govern their behavior at the system prompt level. Revoke access in one call. That's a complete partner platform — not a collection of features."
