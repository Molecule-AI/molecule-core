# Phase 32 — SaaS Federation v2 Competitive Battlecard
**Feature:** SaaS Federation v2 — multi-tenant agent platform with cross-tenant isolation, centralized billing, and org-level governance
**Status:** PMM DRAFT | **Date:** 2026-04-23
**Phase:** 32 (SaaS Federation v2) | **Owner:** PMM
**GA date:** April 30, 2026
**Blocking on:** PM confirmation of beta/GA label, per-tenant feature scope, Stripe Atlas application status

---

## Competitive Context

SaaS Federation v2 is Molecule AI's multi-tenant cloud product — offering organizations their own isolated agent platform with signup in under 5 minutes, workspace-hours billing, and org-level governance that doesn't require self-hosting.

The competitive question this battlecard answers: **how does Molecule AI's multi-tenant SaaS offering compare to LangGraph Cloud and CrewAI's multi-tenant/enterprise options?**

**Note on terminology:** "SaaS Federation v2" refers to the Phase 34 feature (per the messaging matrix). Phase 32 in the build plan covers multi-tenant SaaS infrastructure. The commercial product name for this capability is "Molecule AI SaaS" or "Molecule AI Multi-Tenant." Use "SaaS Federation v2" in internal docs; use "multi-tenant agent platform" or "Molecule AI SaaS" in external copy.

---

## Multi-Tenant Feature Matrix

**Buyer question:** "Can I offer my team or customers a fully isolated agent platform without self-hosting?"

| | Molecule AI SaaS (Phase 32) | LangGraph Cloud | CrewAI |
|---|---|---|---|
| Self-serve signup | ✅ moleculesai.app — signup → org → first workspace < 5 min | ✅ Per-seat SaaS only | ❌ Marketplace listing only |
| Multi-tenant isolation | ✅ Org-level isolation — workspaces, secrets, memory, activity all `org_id`-filtered | ⚠️ Workspace-scoped only (no org hierarchy) | ❌ Single-org teams only |
| Per-tenant auth + org hierarchy | ✅ Parent/child/sibling model, per-workspace tokens | ❌ Per-agent tokens, no hierarchy | ❌ Team-role primitives only |
| Cross-network agent federation | ✅ Phase 30 — external agents register via A2A from any cloud | ❌ Platform-only agents | ❌ Platform-only agents |
| Billing per tenant | ✅ Stripe-backed subscription + workspace-hours metering | Per-seat billing only | Marketplace billing only |
| Self-hosted option | ✅ OSS — same binary, run anywhere | ❌ SaaS-only | ✅ Open source |
| Partner API Key provisioning | ✅ Phase 34 — `mol_pk_*` for programmatic tenant management | ❌ | ❌ |
| Enterprise SSO (WorkOS) | ✅ WorkOS AuthKit — per-org SSO | ❌ Per-user auth only | ⚠️ Enterprise plans with custom auth |

---

## Feature-by-Feature Battlecard

### 1. Multi-Tenant Isolation

**Buyer question:** "If I provision agent workspaces for my team or customers, can I be sure they can't see each other's data?"

| | Molecule AI SaaS | LangGraph Cloud | CrewAI |
|---|---|---|---|
| Org-level isolation model | ✅ `org_id` filter on every row-returning handler | ❌ Workspace-scoped only | ❌ Single-org only |
| Secrets isolation | ✅ `global_secrets` + `workspace_secrets` scoped to org | ⚠️ Environment variables | ⚠️ Team-level secrets |
| Activity log isolation | ✅ `activity_logs` filtered by `org_id` | ⚠️ Per-agent traces | ⚠️ Per-crew logs |
| Cross-tenant data access protection | ✅ Automated red-team CI gate (`isolation_test.go`) | ❌ Not documented | ❌ Not documented |
| Data residency options | ✅ Self-hosted for data residency requirements | ❌ SaaS-only | ✅ Self-hosted option |

**Molecule AI counter:** "LangGraph Cloud and CrewAI are single-organization platforms. Molecule AI SaaS has an org hierarchy that keeps each tenant's workspaces, secrets, memory, and activity logs completely isolated — and we've automated tenant-isolation testing in CI."

---

### 2. Signup and Onboarding Speed

**Buyer question:** "How fast can a new team member or customer get a fully configured agent platform?"

| | Molecule AI SaaS | LangGraph Cloud | CrewAI |
|---|---|---|---|
| Self-serve signup | ✅ < 5 minutes — signup → org → first workspace | ✅ Per-seat provisioning | ❌ Sales-driven / marketplace only |
| Pre-configured workspace templates | ✅ Org templates with defaults, plugins, system prompt | ⚠️ Per-workspace config only | ⚠️ Crew templates |
| Platform-instantiated agent runtime | ✅ Fly Machines boot in < 1 second | ⚠️ Cloud-hosted, variable | ⚠️ Cloud-hosted |
| Import from org template | ✅ Canvas UI org template import | ❌ | ❌ |

**Molecule AI counter:** "Molecule AI SaaS is the only multi-tenant agent platform where a new tenant gets a fully configured org with their own auth, templates, and workspace defaults in under 5 minutes — without talking to sales."

---

### 3. Billing and Economics

**Buyer question:** "Can I pay for agent platform usage per workspace-hour, and can I offer this to my end customers as part of my product?"

| | Molecule AI SaaS | LangGraph Cloud | CrewAI |
|---|---|---|---|
| Workspace-hours billing | ✅ Stripe-backed metering | ❌ Per-seat only | ❌ Marketplace billing only |
| Per-tenant cost tracking | ✅ Per-org usage visible in admin panel | ❌ Shared billing | ❌ Shared billing |
| Reseller / marketplace billing | ✅ Stripe Connect (future) + Partner API Keys (Phase 34) | ❌ | ⚠️ CrewAI Enterprise marketplace |
| Cost predictability | ✅ Fly Machines pricing documented per workspace-hour | Per-seat unpredictable at scale | Per-seat pricing |
| Free tier / trial | ✅ Per-plan free tier | ✅ Free tier | ⚠️ Enterprise trials |

**Molecule AI counter:** "LangGraph Cloud charges per seat — which means every agent in your org counts the same, regardless of usage. Molecule AI SaaS bills per workspace-hour, so you pay for what runs. For platform builders offering agent orchestration as a product, Partner API Keys (Phase 34) lets you provision and bill end customers programmatically."

---

### 4. Enterprise Controls and Compliance

**Buyer question:** "Can enterprise IT and compliance teams get the access controls and audit trail they need without self-hosting?"

| | Molecule AI SaaS | LangGraph Cloud | CrewAI |
|---|---|---|---|
| Enterprise SSO | ✅ WorkOS AuthKit — per-org SSO | ❌ | ⚠️ Enterprise plans |
| Role-based access control | ✅ Org admin / workspace admin / member tiers | ⚠️ Per-user roles | ⚠️ Team roles |
| Org-level audit trail | ✅ Immutable `structure_events` per org | ⚠️ Per-agent traces only | ⚠️ Per-crew logs |
| Data residency | ✅ Self-hosted option for data-residency requirements | ❌ SaaS-only | ✅ Self-hosted option |
| SOC 2 / compliance certifications | ⏳ In progress (Tier 4) | ⚠️ Enterprise compliance programs | ⚠️ Enterprise compliance programs |
| Platform Instructions (org governance) | ✅ Enterprise plans — system-prompt governance | ❌ No equivalent | ❌ No equivalent |
| Tool Trace (execution visibility) | ✅ All plans — execution record in every A2A response | ⚠️ LangSmith required | ❌ Manual callbacks only |

**Molecule AI counter:** "Most multi-tenant agent platforms give you shared billing and call it enterprise readiness. Molecule AI SaaS adds org-level audit trails, WorkOS SSO, Platform Instructions for system-prompt governance, and Tool Trace for full execution visibility — without requiring self-hosting."

---

### 5. Platform Builder / Reseller Story

**Buyer question:** "Can I embed Molecule AI as the agent platform for my SaaS product and manage my customers as tenants?"

| | Molecule AI SaaS + Phase 34 | LangGraph Cloud | CrewAI |
|---|---|---|---|
| Programmatic tenant provisioning | ✅ Partner API Keys (`mol_pk_*`) | ❌ | ❌ |
| Tenant isolation + governance | ✅ Platform Instructions + Tool Trace per tenant | ❌ | ❌ |
| Multi-tenant billing | ✅ Stripe-backed + Partner API Key billing hooks | ❌ | ⚠️ CrewAI Enterprise marketplace |
| White-label / branding | ✅ Tenant canvas with own branding | ❌ | ❌ |
| API-first (no browser dependency) | ✅ Full API for tenant lifecycle | ❌ | ❌ |

**Molecule AI counter:** "LangGraph Cloud and CrewAI are platforms you use. Molecule AI SaaS + Partner API Keys is a platform you build on. If you want to offer agent orchestration as a feature in your product — provision tenants, enforce their governance rules, see their execution traces, and bill them programmatically — that's what Phase 34 + Phase 32 together deliver. No competitor has this stack."

---

## Positioning Claims

**Lead claim:** ✅ FIRST-MOVER (verified per Research Lead competitive audit, 2026-04-22) — "Molecule AI is the first agent platform with a multi-tenant SaaS product that combines org-level isolation, WorkOS SSO, Stripe billing, and a Partner API Key layer for platform builders — letting you provision, govern, and bill agent tenants without self-hosting."

> **Rationale:** LangGraph Cloud and CrewAI are single-organization SaaS platforms. Neither has a Partner API Key layer (Phase 34), org-level governance via Platform Instructions, or a Stripe-billing integration for per-tenant metering. Molecule AI's combination of SaaS Federation v2 (Phase 32) + Partner API Keys (Phase 34) + Platform Instructions (Phase 34) is first-mover. Use "first-mover" framing — a competitor could ship this tomorrow.

**Supporting claims:**
1. **5-minute tenant provisioning** — signup → org → first workspace in under 5 minutes, no sales call
2. **Tenant isolation verified in CI** — `isolation_test.go` automated red-team test in CI gate
3. **API-first platform building** — Partner API Keys + Stripe billing = complete programmatic tenant lifecycle
4. **No self-hosting required** — Fly Machines, Neon, Upstash managed by Molecule AI; self-hosted option available for data residency

**Risks to monitor:**
- LangGraph ships enterprise multi-tenancy → update lead claim to "first agent platform with native multi-tenancy + Partner API Key layer"
- CrewAI Enterprise ships reseller billing → update reseller story to "programmatic billing" differentiator
- Stripe Atlas application delayed → Phase 32 GA moves with Stripe timeline

---

## Language to Avoid

- ~~Do not claim "only platform with multi-tenant agent platform"~~ — use "first-mover" or "first to combine" framing
- Do not claim "GA" until Stripe Atlas is live and PM confirms
- Do not promise specific compliance certifications (SOC 2, FedRAMP) until confirmed by PM
- Do not mention specific pricing tiers until PM confirms

---

## Update Triggers

| Event | Action |
|---|---|
| Stripe Atlas approved | Update billing claims to "Stripe-backed subscription" |
| LangGraph Cloud ships enterprise multi-tenancy | Update lead claim → "first to combine multi-tenancy + Partner API Keys" |
| CrewAI Enterprise ships reseller billing | Update platform builder row |
| SaaS Federation v2 GA confirmed | Update status → APPROVED, open social copy task |
| DevRel ships multi-tenant demo | File social copy task for Content Marketer |

---

## Connection to Phase 34

SaaS Federation v2 (Phase 32) and Partner API Keys (Phase 34) are the commercial stack for platform builders:

- **Phase 32 SaaS Federation v2** → the infrastructure: multi-tenant isolation, WorkOS SSO, Stripe billing, Fly Machines provisioning, tenant canvas
- **Phase 34 Partner API Keys** → the provisioning layer: `mol_pk_*` for programmatic tenant creation and management
- **Phase 34 Platform Instructions** → the governance layer: enforce per-tenant behavioral rules at the system prompt level
- **Phase 34 Tool Trace** → the observability layer: full execution visibility per tenant

Combined: "Molecule AI gives platform builders observability, control, and provisioning in one stack."

---

## Related PMM Assets

| Asset | Status | Notes |
|---|---|---|
| Phase 34 battlecard (`phase-34-partner-api-keys-battlecard.md`) | ✅ Ready | Partner API Keys positioning |
| Phase 34 messaging matrix (`phase34-messaging-matrix.md`) | ✅ Ready | All Phase 34 features |
| Phase 34 positioning brief (`2026-04-23-*.md`) | ✅ Ready | ICP + buyer benefit statements |
| SaaS Federation v2 social copy | ⏳ Not started | Awaiting GA confirmation |
| SaaS Federation v2 blog post | ⏳ Not started | Awaiting PM + DevRel input |

---

*PMM draft 2026-04-23 — Phase 32 SaaS Federation v2 battlecard*
*Source: PLAN.md Phase 32 (Cloud SaaS launch, 2026-Q2/Q3), ecosystem-watch.md (updated 2026-04-22)*
*Reference: Phase 34 battlecard structure, competitors.md*