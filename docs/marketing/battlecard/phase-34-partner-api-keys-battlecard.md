# Phase 34 — Partner API Keys Competitive Battlecard
**Feature:** `mol_pk_*` — partner-scoped org provisioning API key
**Status:** PMM DRAFT | **Date:** 2026-04-22
**Phase:** 34 | **Owner:** PMM
**Blocking on:** PM input on partner tiers + marketplace billing (GA date now confirmed)

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

**Lead claim:** ✅ VERIFIED (Research team audit, 2026-04-23) — "Molecule AI is the **first** agent platform with a first-class partner provisioning API — letting marketplaces, CI/CD pipelines, and automation platforms create and manage Molecule AI orgs via API, without a browser session."

> **Rationale:** Competitive Intel audited LangGraph Cloud, CrewAI, Azure AI Foundry, Dify, Flowise, and n8n. None have a documented programmatic partner org provisioning API equivalent to `mol_pk_*`. Use **"first-mover"** framing (not "only") for legal defensibility — a competitor could launch tomorrow.

**Supporting claims:**
1. **Org-scoped by design** — `mol_pk_*` keys cannot escape their org boundary. Compromised keys neutralize with one API call.
2. **CI/CD-native** — ephemeral test orgs per PR. No shared state. No manual cleanup.
3. **Platform-first** — LangGraph charges per seat. CrewAI offers marketplace listing. Molecule AI offers an API to build either.

**Risks to monitor:**
- AWS/GCP/Azure publish their own partner/OEM programs → Phase 34 becomes table stakes faster
- CrewAI ships partner API → first-mover window closes; update claim to "pioneered" framing

---

## Language to Avoid

- ~~Do not claim "only platform with partner API" unless verified~~ — **RESOLVED:** Use "first-mover" / "first agent platform" language. Do NOT use "only" (legal risk if competitor ships).
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

Phase 30 shipped `mol_ws_*` (per-workspace auth tokens). Phase 34 extends to `mol_pk_*` (partner/platform-level keys). Battlecard cross-sell: ✅ "Phase 30 workspace isolation + Phase 34 partner scoping — **the first agent platform with both layered token scoping and a first-class partner provisioning API.**" — verified 2026-04-23 via competitive audit. Use "first" / "pioneered" framing, not "only".

---

*PMM draft 2026-04-22 — Marketing Lead 2026-04-23 v2: (1) lead claim updated to verified "first-mover" language per Research team competitive audit (LangGraph Cloud, CrewAI, Azure AI Foundry, Dify, Flowise, n8n — no equivalent `mol_pk_*` found), (2) Phase 30 cross-sell updated to "first agent platform with both" framing, (3) Language to Avoid section resolved. GA DATE CONFIRMED: April 30, 2026. Still awaiting PM input on partner tiers and marketplace billing.*

---

## Marketing Lead Review — 2026-04-23

**Status: APPROVED for Sales distribution and launch prep. Two action items below.**

✅ **Lead claim** — "first-mover" framing is correct and Research-verified. Sales team can use this now.
✅ **Phase 30 linkage** — cross-sell claim ("first agent platform with both layered token scoping and a first-class partner provisioning API") is clean and approved.
✅ **GA Date** — April 30, 2026 confirmed. Update Triggers table: mark "Phase 34 GA date confirmed" as ✅ DONE.
✅ **Competitive table** — accurate as of 2026-04-23. Monitor CrewAI and Azure AI Foundry monthly.

⏳ **Action (PM):** Marketplace-native billing rows (AWS/GCP) show "⏳ PM to confirm" — need PM input before Sales uses those rows in enterprise deals. Flagged to PM via issue #1122 routing.

⚠️ **Action (DevRel):** Partner onboarding guide and CI/CD example are marked "⏳ DevRel in progress" — must ship by April 28 to support April 30 launch. Verify DevRel ETA.

**Ready to distribute to:** Sales team, partner AEs. Do NOT share marketplace billing rows externally until PM confirms.