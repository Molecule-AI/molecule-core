# Marketing Social Queue — Status Tracker
**Owner:** PMM | **Last updated:** 2026-04-24 mid-cycle
**Purpose:** Single source of truth for all social copy status across campaigns.

---

## Phase 34 GA Launch (April 30, 2026)

### 2026-04-26 — Phase 34 GA: Tool Trace + Platform Instructions ⚠️ PENDING APPROVAL (T-4)
- **File:** `2026-04-26-phase34-ga-launch/social-copy.md`
- **Status:** PMM pre-write complete. Awaiting Marketing Lead approval. Platform Instructions: confirmed Enterprise plans only (AdminAuth-gated, per plan-gating note). Community FAQ corrected.
- **Content:** 6-post X thread + LinkedIn post
- **Owner:** PMM → Social Media Brand
- **Blocking:** Marketing Lead approval + X credentials
- **Canonical:** `docs.molecule.ai/blog/tool-trace-platform-instructions`

### 2026-04-30 — Phase 34 GA: Partner API Keys ✅ APPROVED (Marketing Lead)
- **File:** `2026-04-30-phase-34-ga-launch/social-copy.md`
- **Status:** APPROVED by Marketing Lead 2026-04-23. GA language confirmed (community FAQ + updated positioning brief). Ready for Social Media Brand execution.
- **Content:** 5-post X thread
- **Owner:** Social Media Brand
- **Blocking:** X credentials
- **Canonical:** `docs.molecule.ai/blog/partner-api-keys`
- **Note:** Issue #1829 — Tool Trace/Platform Instructions thread already posted Apr 23. Partner API Keys thread targets Apr 30 GA date.

---

## Phase 30 Social Campaign — Archive (April 21–23, past)

### 2026-04-21 — Chrome DevTools MCP 🟡 PAST — posting status unknown
- **File:** `2026-04-21-chrome-devtools-mcp/social-copy.md`
- **Status:** Copy was ready. Whether it was posted is unconfirmed — X credentials missing blocked all Phase 30 publishing.
- **Owner:** PMM → Social Media Brand

### 2026-04-21 — Cloudflare Artifacts 🟡 PAST — posting status unknown
- **File:** `2026-04-21-cloudflare-artifacts/social-copy.md`
- **Status:** PMM pre-write complete. Whether it was posted is unconfirmed.
- **Owner:** PMM → Social Media Brand

### 2026-04-22 — EC2 Instance Connect SSH 🟡 PAST — posting status unknown
- **File:** `2026-04-22-ec2-instance-connect-ssh/social-copy.md`
- **Status:** PMM positioning approved (GH #1637). DevRel screenshot + blog still outstanding. Whether posted is unconfirmed.
- **Owner:** PMM → Social Media Brand → DevRel

---

## Phase 30 Social Campaign — Active (April 24–25)

### 2026-04-24 — EC2 Console Output ✅ APPROVED (Marketing Lead) [T-6: publish today]
- **File:** `2026-04-24-ec2-console-output/social-copy.md`
- **Status:** Approved by Marketing Lead 2026-04-22. Ready for Social Media Brand execution.
- **Content:** 4-post X thread + LinkedIn
- **Owner:** Social Media Brand
- **Blocking:** X credentials + visual asset (`ec2-console-output-canvas.png`, 1200×800 dark mode)
- **Campaign position:** Day 4 of Phase 30 social campaign

### 2026-04-25 — Org-Scoped API Keys ✅ APPROVED (Marketing Lead) [T-5: publish tomorrow]
- **File:** `2026-04-25-org-scoped-api-keys/social-copy.md`
- **Status:** Approved by Marketing Lead 2026-04-21. Ready for Social Media Brand execution.
- **Content:** 5-post X thread + LinkedIn
- **Owner:** Social Media Brand
- **Blocking:** X credentials + visual assets (Canvas UI screenshot, before/after credential model, audit log terminal output)
- **Campaign position:** Day 5 of Phase 30 social campaign

---

## Staged on `origin/staging` (unreviewed by PMM)

### MCP Server List — Day 1 ✅ COPY READY
- **File:** `docs/marketing/campaigns/mcp-server-list/social-copy.md` (on staging, commit `0d3ad96`)
- **Status:** Copy complete. Awaiting visual assets + X credentials.
- **Content:** 5-post X thread + LinkedIn
- **Canonical URL:** `docs.molecule.ai/blog/mcp-server-list`
- **Owner:** Social Media Brand
- **Blocking:** Visual assets + X credentials

### Discord Adapter — Day 2 ✅ COPY READY
- **File:** `discord-adapter-social-copy.md` (on staging)
- **Status:** Copy complete. Awaiting Marketing Lead Day 2 approval + X credentials + visual assets.
- **Content:** 4 X variants + LinkedIn + Reddit + HN copy
- **Canonical URL:** `docs.molecule.ai/blog/discord-adapter` (live, PR #1301 merged)
- **Owner:** Social Media Brand → Marketing Lead (Day 2 approval)

### A2A Enterprise Deep-Dive — Day T+1 ⚠️ ON STAGING ONLY
- **File:** `docs/marketing/campaigns/a2a-enterprise-deep-dive/social-copy.md` (on staging only, not main)
- **Status:** COPY READY (PMM-approved, 72h window). Not on origin/main.
- **Content:** 4-post X thread + LinkedIn
- **Canonical URL:** `docs.molecule.ai/blog/a2a-v1-agent-platform`
- **Owner:** PMM → Social Media Brand
- **Blocking:** X credentials + needs to be cherry-picked to origin/main for execution
- **Note:** File needs to be on origin/main before Social Media Brand can execute. Executor must confirm staging access, or file must be cherry-picked to main.

---

## Held / Pending Decision

### Fly.io Deploy Anywhere — Stale (T+6)
- **File:** `fly-deploy-anywhere-social-copy.md` (on staging)
- **Status:** PMM recommendation: Option A (retrospective framing). Decision memo: `fly-deploy-anywhere-decision-memo.md`
- **Campaign position:** Phase 30 social campaign catch-up
- **Decision needed:** Marketing Lead confirmation on Option A framing
- **Blocking:** Marketing Lead decision + X credentials

### Phase 30 (original) — MERGED
- **File:** `phase30-social-copy.md`
- **Status:** MERGED to origin/main. Awaiting Marketing Lead publish approval.
- **Owner:** PMM → Social Media Brand
- **Blocking:** Marketing Lead publish approval + X credentials

---

## Cross-Cutting Blockers (all human-gated)

| Blocker | Affects |
|---|---|
| X credentials (`X_ACCESS_TOKEN` + `X_ACCESS_TOKEN_SECRET`) | ALL posts — Social Media Brand cannot publish anything |
| Marketing Lead publish approvals | Chrome DevTools MCP, Phase 30, Fly.io, Discord Day 2 |
| DevRel terminal screenshot (PR #1545) | EC2 Instance Connect SSH |
| Content Marketer blog post (#1546) | EC2 Instance Connect SSH |
| Visual assets (Canvas screenshots, diagrams) | EC2 Console Output, Org-Scoped API Keys, Chrome DevTools MCP |

---

## Assets: Visual Requirements by Post

| Post | Asset | Source | Status |
|---|---|---|---|
| Chrome DevTools MCP | 3-item checklist graphic | Custom (Lighthouse/Regression/Auth) | Needed |
| Chrome DevTools MCP | Fleet diagram | Reuse `marketing/assets/phase30-fleet-diagram.png` | Ready |
| EC2 Instance Connect SSH | Canvas terminal screenshot | DevRel (PR #1545) | Blocked |
| EC2 Console Output | Canvas screenshot (dark, 1200×800) | Custom | Needed |
| Org-Scoped API Keys | Canvas Org API Keys UI screenshot | Custom | Needed |
| Org-Scoped API Keys | Before/after credential model graphic | Custom | Needed |
| Org-Scoped API Keys | Audit log terminal output | Custom | Needed |
| MCP Server List | Campaign visual | Custom | Needed |
| Discord Adapter | Multi-channel diagram | Custom | Needed |

---

*PMM compiled 2026-04-23. Consolidated from multiple inline social-queue files into single status tracker.*
*Marketing Lead: approve queue items to unblock Social Media Brand for execution once X credentials are restored.*

---

## Battlecards (complete)

| Battlecard | Phase | Status | Pushed |
|---|---|---|---|
| Phase 30 Remote Workspaces | 30 | ✅ PMM DRAFT | `marketing/phase-34-launch-prep` (2026-04-23) |
| Phase 32 SaaS Federation v2 | 32 | ✅ PMM DRAFT | `marketing/phase-34-launch-prep` |
<<<<<<< HEAD
| Phase 34 Partner API Keys | 34 | ✅ PMM DRAFT | `marketing/phase-34-launch-prep` |

---

## Research Files (complete this cycle)

| File | Status | Blocking |
|---|---|---|
| `briefs/saas-fed-v2-what-shipped.md` | ⚠️ NO IMPLEMENTATION FOUND — PM must confirm scope | PM confirmation before battlecard copy |
| `briefs/partner-api-keys-rate-limits-note.md` | ✅ 60 req/min per mol_pk_* key (default, configurable) | PM confirm Go implementation |
| `launches/partner-onboarding-guide.md` | ✅ First pass (831 words, 6 sections) | `[PARTNER TIER TBD]` placeholder |
| `launches/phase-34-community-announcement.md` | ✅ Updated — SaaS Fed v2 section flagged ⚠️ PM REVIEW NEEDED | PM confirm before publish |

---

*PMM compiled 2026-04-23. Updated 2026-04-23 late cycle: research files section added.*
=======
| Phase 34 Partner API Keys | 34 | ✅ PMM DRAFT | `marketing/phase-34-launch-prep` |
>>>>>>> 78109f59 (docs(marketing): update social queue status — battlecard section added)
