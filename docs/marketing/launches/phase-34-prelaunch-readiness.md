# Phase 34 — Pre-Launch Readiness Summary
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Date:** 2026-04-24
**Git push:** BLOCKED (token) — all commits local on `marketing/phase-34-launch-prep`

---

## ⚠️ Gates — Must Resolve Before Any Public Post

### Gate 1: GA vs Beta Label Conflict
**File:** `docs/marketing/briefs/phase34-ga-vs-beta-conflict.md`
**Status:** PM must decide — escalated, awaiting response
**Impact:** All four Phase 34 features have inconsistent GA/Beta framing between internal briefs (BETA) and external assets ("live now"). Cannot post Discord, Reddit, or HN until PM confirms labeling.

### Gate 2: Git Push Token
**Status:** `ghs_*` GitHub App tokens expire after 60 min with no refresh wired up — Issue #1933 (P0, opened Apr 24)
**Impact:** Nothing can reach `origin/` until P0 fix is shipped and token refresh works reliably
**Note:** Content is complete — only blocked on infrastructure fix

### Gate 3: X Credentials (Issue #1865)
**Status:** No mol-ops response
**Impact:** Social posts (X/LinkedIn) cannot go out on April 30 unless credentials land
**Mitigation:** Reddit + HN do not require X credentials — Day 2 posts can proceed

---

## ✅ Assets Ready to Post (hold pending Gate 1)

### Discord
| Asset | File | Status |
|-------|------|--------|
| Announcement | `phase-34-community-announcement.md` | ✅ Ready — on `origin/docs/phase34-community-launch` (PR #1860 blocked on stale CI) |
| FAQ | `phase-34-community-faq.md` | ✅ Ready — on `origin/marketing/phase-34-launch-prep` |
| Runbook | `phase-34-discord-runbook.md` | ✅ Ready — committed locally |
| Response queue | `phase-34-community-response-queue.md` | ✅ Ready — committed locally |

### Day 2 External (Reddit + HN)
| Asset | File | Status |
|-------|------|--------|
| Reddit post | `phase-34-reddit-post.md` | ✅ Ready (bb21fed0) — GA framing, no pricing, no alpha |
| HN Show HN | `phase-34-hn-show-hn.md` | ✅ Ready (bb21fed0) — honest caveats, correct GA scope |

### Social Copy (blocked on X credentials + Gate 1)
| Asset | File |
|-------|------|
| Tool Trace social | `phase-34-tool-trace-social-copy.md` |
| Platform Instructions social | `phase-34-platform-instructions-social-copy.md` |
| Partner API Keys social | `phase-34-partner-api-keys-social-copy.md` |
| SaaS Fed v2 social | `phase-34-saas-fed-v2-social-copy.md` |

### Internal / Reference
| Asset | File |
|-------|------|
| Launch handoff | `phase-34-handoff.md` |
| Asset inventory | `phase-34-asset-inventory.md` |
| DevRel prebrief (10 Q&As) | `phase-34-devrel-prebrief.md` |
| Reconciliation log | `phase-34-reconciliation-log.md` |
| GA vs Beta conflict doc | `phase34-ga-vs-beta-conflict.md` |

---

## ❌ Do Not Use — Pushed Versions on `origin/docs/phase34-community-launch`

The Reddit and HN posts in `docs/marketing/community/` on `origin/docs/phase34-community-launch` have been superseded. They contain:
- Wrong `tool_trace` schema (includes `reasoning` field — not real)
- Wrong Platform Instructions API format (`{"type": "instruction"...}` — not real)
- "This is alpha" framing for Tool Trace

**Canonical versions:** `docs/marketing/launches/phase-34-reddit-post.md` and `docs/marketing/launches/phase-34-hn-show-hn.md` (both bb21fed0)

---

## ✅ Design Partner Guard — Confirmed Clean

All Phase 34 community-facing assets correctly exclude design partner names:
- Approved Reddit + HN posts: "Do not name any design partners" in posting notes
- Discord runbook + handoff: "No design partner names in copy" in pre-launch checklist
- `Acme Corp` only appears in internal positioning briefs as a confirmed placeholder

---

## 📋 PLAN.md Context (from eco-watch branches)

Phase 30 (Remote Workspaces): IN PROGRESS
Phase 32 (Cloud SaaS launch): Infrastructure live, Phase A-I tracking
Phase 34: Not in public PLAN.md — likely in private `molecule-controlplane` repo
KI-001: Telegram kicked event (no fix yet)
KI-002: Delegation idempotency guard (no fix yet)
KI-003: commit_memory not in activity_logs (no fix yet)
P0 #1933 (Apr 24): GH_TOKEN expires after 60min, refresh not wired up — fleet token dead, queues stalled

---

## Day 2 Plan (April 30 ~16:00 UTC)

1. Reddit r/MachineLearning: `phase-34-reddit-post.md` (bb21fed0) — no X needed
2. HN Show HN: `phase-34-hn-show-hn.md` (bb21fed0) — no X needed, pin tool_trace code snippet
3. X/LinkedIn: blocked on X credentials (issue #1865)
4. Monitor 2h (Reddit) / 3h (HN), 30-min reply SLA

---

*Last updated: 2026-04-24 00:00 UTC*