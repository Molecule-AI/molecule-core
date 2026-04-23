# Phase 34: Partner API Keys — PMM Positioning Brief
**Owner:** PMM | **Status:** DRAFT (reviewed by Marketing Lead 2026-04-23) | **Date:** 2026-04-22
**Assumptions:** GA date TBD (blocked on Phase 32 completion + infra); partner tiers TBD with PM

---

## Executive Summary

Phase 34 (Partner API Keys) ships a `mol_pk_*` scoped key type that lets CI/CD pipelines, marketplace resellers, and automation tools create and manage Molecule AI orgs via API — without a browser session. This is the foundational capability for three strategic channels: **partner platforms**, **marketplace resellers**, and **enterprise CI/CD automation**. Each channel requires distinct positioning, but all share the same core value prop: *programmatic org provisioning, at scale, without compromising security*.

---

## What Phase 34 Ships (Technical)

| Component | Detail |
|-----------|--------|
| Key type | `mol_pk_*` — SHA-256 hashed in DB, returned in plaintext once on creation |
| Scoping | Org-scoped only; keys cannot access other orgs |
| Rate limiting | Per-key limiter, separate from session limits |
| Audit | `last_used_at` tracking on every request |
| Endpoints | `POST /cp/admin/partner-keys`, `GET /cp/admin/partner-keys`, `DELETE /cp/admin/partner-keys/:id` |
| Secret scanner | `mol_pk_` added to pre-commit secret scanner |
| Onboarding | Partner onboarding guide + two code examples (org lifecycle, CI/CD test org) |

---

## Positioning by Channel

### Channel 1: Partner Platforms

**Buyer:** DevRel + platform integrations lead at platforms that want to embed or white-label Molecule AI as the agent orchestration layer.

**Core message:** *"Molecule AI embeds in 10 lines of code. Provision a full org, attach your branding, and hand the tenant a ready-to-run fleet."*

**Problem:** Platforms that want to offer agent orchestration as a feature today have two bad options — build it themselves (months of work, ongoing maintenance) or integrate via browser sessions (brittle, non-programmatic). Neither scales.

**Solution:** Partner API Keys give platforms a first-class provisioning path. A partner platform calls `POST /cp/admin/partner-keys` with `orgs:create` scope, provisions a white-labeled org for each customer, and hands the customer a dashboard that is already their org, already wired up, already running agents.

**Three claims:**
1. **Zero browser dependency.** Every provisioning action is an API call. Integrations don't break on UI changes.
2. **Scope-isolated by design.** Each partner key is scoped to one org. A compromised key cannot access other tenants or the platform's own infrastructure.
3. **Revocable instantly.** `DELETE /cp/admin/partner-keys/:id` revokes access on the next request. No waiting for session expiry.

**Target dev:** Platform integrations engineer, DevRel who owns partner ecosystem
**CTA:** Request partner access → `docs.molecule.ai/docs/guides/partner-onboarding`

---

### Channel 2: Marketplace Resellers

**Buyer:** Marketplace ops team at cloud marketplaces (AWS Marketplace, GCP Marketplace) or agent framework directories who want to offer one-click Molecule AI org provisioning alongside existing listings.

**Core message:** *"Molecule AI on [Marketplace]: provision in seconds, manage via API, bill through your existing account."*

**Problem:** Marketplaces that list SaaS tools today have to manually provision trials, manage credentials out of band, and reconcile billing. The manual overhead makes Molecule AI a low-margin listing.

**Solution:** Partner API Keys enable fully automated provisioning through marketplace billing APIs. A buyer clicks "Deploy on [Marketplace]", the marketplace calls the Partner API to provision an org, charges begin on the marketplace invoice, and the buyer lands in a fully configured dashboard.

**Three claims:**
1. **Automated provisioning end-to-end.** From click to running org — no manual handoff. ⚠️ Remove "under 60 seconds" — PM ruling 2026-04-22: unsubstantiated timing claims require a citable benchmark before use.
2. **Marketplace-native billing.** Usage flows through the marketplace's existing invoicing, not a separate Molecule AI subscription.
3. **API-first management.** Marketplaces manage orgs, seats, and deprovisioning via the same Partner API used for provisioning.

**Target dev:** Marketplace listing owner, cloud marketplace integrations engineer
**CTA:** List on [Marketplace] → contact partner team

---

### Channel 3: Enterprise CI/CD Automation

**Buyer:** DevOps / Platform engineering team at enterprises that want to spin up ephemeral test orgs as part of CI pipelines, run integration tests against a fresh Molecule AI org per PR, or automate org provisioning for dev/staging environments.

**Core message:** *"Test against a real org, every commit, without touching the production fleet."*

**Problem:** Enterprise teams building on Molecule AI today have to either share test orgs (flaky, data contamination) or manually provision ephemeral orgs per test run (slow, non-automatable). Neither supports a high-velocity CI/CD workflow.

**Solution:** Partner API Keys + CI/CD example in the onboarding guide gives platform teams a fully automated org lifecycle per pipeline run: `POST` to create org → run tests → `DELETE` to teardown. Each PR gets a clean org. No cross-contamination. No manual cleanup.

**Three claims:**
1. **Per-PR ephemeral orgs.** Each pipeline run gets a fresh org with default settings. Tests run in isolation. No shared-state flakiness.
2. **Automated teardown.** `DELETE /cp/admin/partner-keys/:id` deprovisions the org and stops billing immediately.
3. **No browser required.** The entire lifecycle — create, configure, test, teardown — is one or two API calls. CI/CD-native from day one.

**Target dev:** Platform engineer, DevOps lead, CI/CD team
**CTA:** CI/CD integration guide → `docs.molecule.ai/docs/guides/partner-onboarding#cicd-example`

---

## Cross-Channel Positioning

All three channels share a single technical differentiator that should appear in every channel's collateral:

> **Partner API Keys are org-scoped, scope-enforced, and revocable in one call.** A `mol_pk_*` key cannot escape its org boundary. Compromised keys cost one `DELETE` to neutralize. This is not a personal access token with a org-wide blast radius — it is an infrastructure credential designed for the partner tier.

---

## Phase 30 Linkage

Phase 30 (Remote Workspaces) shipped the per-workspace auth token model (`mol_ws_*`). Phase 34 extends that model to the *platform tier* with `mol_pk_*` — partner/platform-level keys that provision and manage orgs. Cross-sell opportunity: every Phase 34 org comes with Phase 30 remote workspace capability at no additional configuration.

---

## Collateral Needed

| Asset | Owner | Status |
|-------|-------|--------|
| Partner onboarding guide (`docs/guides/partner-onboarding.md`) | DevRel / PM | Not started |
| CI/CD example (org lifecycle + test teardown) | DevRel | Not started |
| Partner API Keys landing page section | Content Marketer | Not started |
| Marketplace listing copy | Content Marketer | Not started |
| Battlecard update (add Phase 34 row) | PMM | ✅ Done — staging `docs/marketing/battlecard/phase-34-partner-api-keys-battlecard.md` |
| Partner tier pricing page | Marketing Lead / PM | TBD |

---

## Open Questions for PM / Marketing Lead

| # | Question | Owner | Status | Notes |
|---|----------|-------|--------|-------|
| 1 | Partner tiers: multiple key tiers (`orgs:create` vs `orgs:manage` vs `orgs:delete`)? Pricing model? | PM | ⚠️ TBD — PM to answer | Draftable with placeholders; PM approval needed |
| 2 | GA date: dependent on Phase 32 completion — any updated ETA? | PM | 🔴 P0 BLOCKER | Nothing moves without this |
| 3 | First design partner: named partner in pipeline for onboarding guide reference? | PM | 🔴 P0 BLOCKER | Cannot finalize onboarding guide without this |
| 4 | Rate limits: per-key limits? Do limits vary by tier? | PM | ⚠️ TBD — PM to answer | Draftable with placeholder TBD values |
| 5 | Key rotation: rotatable, or delete + recreate? | PM | ⚠️ TBD — PM to answer | Delete + recreate is the current model; rotate is an enhancement |

**2026-04-23 update (PMM):** Q1, Q4, Q5 are advanceable with TBD placeholders — PM only needs to confirm or adjust. Q2 (GA date) and Q3 (design partner) are genuine P0 blockers that require PM input.

---

## Competitive Context

No direct competitor has a published Partner API Key program at the agent orchestration layer. CrewAI and AutoGen focus on developer-seat pricing. LangGraph Cloud uses per-user licensing with no partner provisioning tier. This is a first-mover opportunity to own the "agent platform-as-a-backend" positioning before the category standardizes.

**Risk:** If AWS/GCP/Azure absorb agent orchestration into their managed AI platforms (Phase 30 risk, tracked in ecosystem-watch), the partner platform channel may shift to OEM relationships rather than API-key-based reselling. Monitor for cloud provider announcements.
