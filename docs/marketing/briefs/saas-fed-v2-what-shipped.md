# SaaS Federation v2 — What Shipped Note
**Date:** 2026-04-23 | **Owner:** PMM
**Source:** Codebase search + docs search | **Status:** INVESTIGATION INCOMPLETE

---

## Summary

**No implementation evidence found for "SaaS Federation v2" as a named feature.**

The term "SaaS Federation v2" appears in marketing materials (Phase 34 messaging matrix, Phase 32 battlecard, positioning briefs) but:
- No tutorial file exists at `docs/tutorials/saas-federation` — the messaging matrix explicitly says it should be at `docs/tutorials/saas-federation` (PR #1613 reference), and this file does not exist.
- No Go files in `workspace-server/` or `platform/` contain the string "federation" or "saas-fed."
- No launch doc exists at `docs/marketing/launches/pr-1613-saas-federation-v2.md`.
- The architecture docs (`docs/architecture/`) contain no mention of "federation."

The feature does appear in marketing copy as a conceptual grouping of Phase 32 + Phase 34 capabilities: multi-tenant isolation, WorkOS SSO, Stripe billing, Fly Machines provisioning, and Partner API Keys. But there is no separate PR #1613 implementation that codifies "SaaS Federation v2" as a discrete unit of work.

**PLAN.md note (2026-04-24):** Phase 34.1–34.4 checkboxes in PLAN.md (`/tmp/PLAN.md` lines 622–661) are all unchecked `[ ]`. Partner API Keys implementation may not be marked shipped in the engineering plan. Phase 33 in PLAN.md is "Tenant Subdomain Routing — MIGRATING TO CLOUDFLARE TUNNEL" — not federation. This reinforces that "SaaS Fed v2" is a marketing grouping, not an engineering phase.

---

## What "SaaS Federation v2" Refers To

Based on the messaging matrix and Phase 34 positioning brief, the term appears to describe the commercial stack of:

| Capability | Source Phase | Status |
|---|---|---|
| Multi-tenant org isolation (`org_id` filter) | Phase 30+ | Live |
| Per-workspace auth tokens (`mol_ws_*`) | Phase 30 | Live |
| WorkOS AuthKit (per-org SSO) | Phase 32 | Live |
| Fly Machines backend provisioning | Phase 30 | Live |
| Neon + Upstash managed backing services | Phase 32 | Live |
| Stripe billing integration | Phase 32 | ⚠️ Stripe Atlas pending |
| Cloudflare Tunnel migration | Phase 33 | MIGRATING (PLAN.md: "MIGRATING TO CLOUDFLARE TUNNEL") — this IS the Phase 33 engineering work, not federation |
| Partner API Keys (`mol_pk_*`) | Phase 34 | Live (Apr 23) |
| SaaS Federation v2 tutorial | PR #1613 | **DOES NOT EXIST** |

The messaging matrix (2026-04-23) explicitly flags this: "Do NOT draft community copy for this feature until PM confirms: (a) what it actually ships, (b) the GA/beta/alpha label, and (c) the primary use case narrative."

---

## Is a Battlecard Safe to Write?

**No.** A battlecard requires a discrete feature with concrete capabilities. "SaaS Federation v2" as a named feature is not verified in the codebase. Writing a battlecard for it now would mean writing marketing copy with no implementation anchor — claims that could be invalidated the moment PM defines the feature scope.

**Safe path forward:**
1. PM must confirm what PR #1613 actually shipped (or whether it was merged at all)
2. If no discrete feature, rename the battlecard to "Multi-Tenant Agent Platform" and anchor claims to Phase 30+ Phase 34 capability stack
3. The existing Phase 32 battlecard at `docs/marketing/battlecard/phase-32-saas-fed-v2-battlecard.md` is written against the conceptual stack, not a named PR — this is a risk

---

*PMM investigation 2026-04-23 — no implementation evidence found for SaaS Federation v2 as a discrete feature.*
*Action: PM must confirm feature scope before external copy is written.*
