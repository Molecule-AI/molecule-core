# Phase 34 — Partner API Keys Competitive Battlecard
**Feature:** `mol_pk_*` — partner-scoped org provisioning API key
**Status:** PMM DRAFT | **Date:** 2026-04-22
**Phase:** 34 | **Owner:** PMM
**Blocking on:** Phase 32 completion + PM input on partner tiers + GA date

---
## Competitive Context

No direct competitor has a published Partner API Key program at the agent orchestration layer. This is a first-mover opportunity. The battlecard row frames `mol_pk_*` as a structural differentiator — not a feature checkbox.

**Competitor landscape (updated 2026-04-22):**

| Competitor | Partner / API Program | Org Provisioning | CI/CD Org Lifecycle | Self-Hosted |
|------------|----------------------|-----------------|---------------------|-------------|
| LangGraph Cloud | Per-user SaaS licensing | ❌ | ❌ | ❌ (SaaS-only) |
| CrewAI | Enterprise marketplace (live) | ❌ | ❌ | ✅ (open source) |
| AutoGen (Microsoft) | None | ❌ | ❌ | ✅ (open source) |
| AWS/GCP managed | OEM resale programs (separate) | N/A | N/A | N/A |
| **Molecule AI Phase 34** | **Partner API Keys** | **✅ `POST /cp/admin/partner-keys`** | **✅ Ephemeral orgs per PR** | **✅** |

---

## Feature-by-Feature Battlecard

### 1. Partner Platform Integration

**Buyer question:** "Can I embed Molecule AI as the agent orchestration layer for my platform?"

| | Molecule AI Phase 34 | LangGraph Cloud | CrewAI |
|---|---|---|---|
| Programmatic org provision | ✅ `mol_pk_*` | ❌ per-user seat licensing only | ❌ marketplace listing only |
| Org-scoped keys | ✅ — key cannot escape its org boundary | N/A | N/A |
| Partner onboarding guide | ⏳ DevRel in progress | ❌ | ❌ |
| White-label / branding | ✅ via partner-provisioned orgs | ❌ | ❌ |
| API-first (no browser dependency) | ✅ | ❌ | ❌ |

**Molecule AI counter:** "LangGraph Cloud and CrewAI are end-user platforms. Molecule AI is infrastructure your platform builds on."

---

### 2. CI/CD / Automation

**Buyer question:** "Can my pipeline spin up test orgs per PR?"

| | Molecule AI Phase 34 | LangGraph Cloud | CrewAI |
|---|---|---|---|
| Ephemeral test orgs | ✅ via `POST` + `DELETE` partner key | ❌ | ❌ |
| Per-PR isolation | ✅ — each run gets a fresh org | ❌ | ❌ |
| Automated teardown | ✅ — `DELETE /cp/admin/partner-keys/:id` stops billing | ❌ | ❌ |
| No shared-state contamination | ✅ | ❌ | ❌ |
| CI/CD example in docs | ⏳ DevRel in progress | ❌ | ❌ |

**Molecule AI counter:** "CrewAI's marketplace is for consuming agents. Molecule AI's partner API is for provisioning infrastructure."

---

### 3. Marketplace / Reseller

**Buyer question:** "Can I resell Molecule AI through my marketplace?"

| | Molecule AI Phase 34 | AWS Marketplace (reseller) | GCP Marketplace |
|---|---|---|---|
| Automated provisioning | ✅ via Partner API | ✅ | ✅ |
| Marketplace-native billing | ⏳ PM to confirm | ✅ | ✅ |
| Partner API + marketplace billing | ⏳ PM to confirm | N/A | N/A |
| Programmatic org lifecycle | ✅ | ✅ | ✅ |

**Note:** Phase 34 delivers the API side. Marketplace-native billing integration (AWS/GCP) is PM-to-confirm.

---

## Positioning Claims

**Lead claim:** "Molecule AI is the only agent platform with a first-class partner provisioning API. `mol_pk_*` keys let you build agent marketplaces, CI/CD integrations, and white-label platforms on top of Molecule AI — without a browser session."

**Supporting claims:**
1. **Org-scoped by design** — `mol_pk_*` keys cannot escape their org boundary. Compromised keys neutralize with one API call.
2. **CI/CD-native** — ephemeral test orgs per PR. No shared state. No manual cleanup.
3. **Platform-first** — LangGraph charges per seat. CrewAI offers marketplace listing. Molecule AI offers an API to build either.

**Risks to monitor:**
- AWS/GCP/Azure publish their own partner/OEM programs → Phase 34 becomes table stakes faster
- CrewAI ships partner API → first-mover advantage closes

---

## Language to Avoid

- Do not claim "only platform with partner API" unless verified (check CrewAI, LangGraph, AutoGen GitHub)
- Do not mention specific pricing tiers until PM confirms
- Do not promise marketplace billing integration until PM confirms

---

## Update Triggers

| Event | Action |
|-------|--------|
| CrewAI launches partner API | Update lead claim → "first agent platform with partner API" |
| AWS/GCP publish agent OEM program | Add OEM row, frame Molecule AI as OEM alternative |
| Phase 34 GA date confirmed | Open social copy brief, notify Social Media Brand |
| DevRel ships partner onboarding guide | File social copy task for Content Marketer |

---

## Phase 30 Linkage

Phase 30 shipped `mol_ws_*` (per-workspace auth tokens). Phase 34 extends to `mol_pk_*` (partner/platform-level keys). Battlecard cross-sell: "Phase 30 workspace isolation + Phase 34 partner scoping — the only platform with both."

---

*PMM draft 2026-04-22 — pending PM input on partner tiers, GA date, and marketplace billing confirmation*