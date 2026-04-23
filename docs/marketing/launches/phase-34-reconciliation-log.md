# Phase 34 — Reconciliation Log
**Date:** 2026-04-23
**Reconciled by:** Community Manager

---

## Reconciliation Decision Summary

Marketing Lead's versions on `marketing/phase-34-launch-prep` are superior. `/tmp/` drafts are **discarded**. No overwrites made.

---

### `phase-34-community-faq.md` — REPO WIN

**Repo version** (15+ Q&As, 5 sections, committed): Superior coverage, clean structure, already has `output_preview` field format, covers all four features comprehensively.

**My `/tmp/phase34-community-qa.md`**: Has additional depth in Q5 (Langfuse full comparison), Q7 (Canvas UI — issue #759), Q9 (cross-org leak risk), Q10 (SaaS Fed v2 architecture). These questions are covered in the **DevRel prebrief** already added (`phase-34-devrel-prebrief.md`, commit `7b286ba0`).

**Action:** Repo version kept. DevRel prebrief preserves the additional Q&A depth. No merge needed.

---

### `phase-34-community-announcement.md` — REPO WIN

**Repo version** (Marketing Lead committed): Clean, full-length Discord-format announcement. Partner API Keys correctly framed "GA April 30." Correct blog URLs (live on main). Clean CTA to `#bug-reports`, `#feedback`, `#partner-program`.

**My `/tmp/phase34-discord-announcement.md`**: Broken relative links (`docs/architecture/...`), references non-existent `docs/blog/2026-04-23-phase34-launch.md`, different format (shorter, more copy-heavy).

**Action:** Repo version kept. `/tmp/` version discarded.

---

### `phase-34-reddit-post.md` — REPO WIN (my local version is the approved version)

**Critical finding:** Marketing Lead says the repo version (`docs/marketing/launches/phase-34-reddit-post.md`) is "APPROVED, committed." But I wrote that file this session as commit `bb21fed0`. It does NOT exist in the origin remote.

The repo file on `origin/docs/phase34-community-launch` (`docs/marketing/community/phase34-reddit-post.md`) is the **older version** with alpha framing and pricing mentions. The local version in `docs/marketing/launches/phase-34-reddit-post.md` is the correct approved version.

**Action:** Local version (bb21fed0) is the approved version. Repo version in `community/` directory is NOT the one to use.

---

### `phase-34-hn-show-hn.md` — REPO WIN (my local version is the approved version)

Same as Reddit — Marketing Lead's "APPROVED" refers to the local version I wrote this session (bb21fed0), not the pushed version on `origin/docs/phase34-community-launch`.

**Action:** Local version (bb21fed0) is the approved version.

---

## Issue #1836 — CLOSED

Canonical files:
- Announcement: `docs/marketing/launches/phase-34-community-announcement.md` (Marketing Lead version) — also on `origin/docs/phase34-community-launch`
- FAQ: `docs/marketing/launches/phase-34-community-faq.md` (Marketing Lead version)
- Reddit: `docs/marketing/launches/phase-34-reddit-post.md` (Community Manager version, bb21fed0) — **READY**
- HN: `docs/marketing/launches/phase-34-hn-show-hn.md` (Community Manager version, bb21fed0) — **READY**

Git push blocked — all approved versions are on `marketing/phase-34-launch-prep` as local commits.

---

## HOLD ON ALL PHASE 34 GA POSTING

GA vs Beta conflict document (`docs/marketing/briefs/phase34-ga-vs-beta-conflict.md`) — not found in repo. Marketing Lead must confirm GA vs Beta framing before any public post goes live.

---

*Reconciled: 2026-04-23*