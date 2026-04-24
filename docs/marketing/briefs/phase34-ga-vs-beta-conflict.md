# Phase 34 — GA vs Beta Label Conflict
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** PM (decision required)
**Status:** ⚠️ BLOCKING — unresolved
**Date:** 2026-04-23
**Found by:** Community Manager (from `phase34-positioning.md` + `phase34-messaging-matrix.md`)

---

## The Conflict

**Internal positioning briefs say:** BETA
**External launch assets say:** "live now" / GA

---

## What the Internal Briefs Say

From `docs/marketing/briefs/phase34-positioning.md`:

> *"Partner API Keys are **BETA** — do not claim GA in press materials. Use 'now available in beta' or 'shipping April 30, 2026.'"*
>
> *"Tool Trace and Platform Instructions shipped via PR #1686 — **BETA**."*
>
> *"SaaS Federation v2 — **BETA** or **EARLY ACCESS**, pending PM label confirmation."*

From `docs/marketing/briefs/phase34-messaging-matrix.md`:

> *"HN/Reddit framing: 'Be honest: this is a beta feature.'"*
> (referring to Tool Trace)

> *"SaaS Federation v2: **Do NOT draft community copy for this feature** until PM confirms... the GA/beta/alpha label."*

---

## What the External Assets Say

From `docs/marketing/launches/phase-34-community-announcement.md`:

> *"All four features are live now (Partner API Keys GA on April 30)."*

This implies:
- Tool Trace: **LIVE NOW** (no qualifier)
- Platform Instructions: **LIVE NOW** (no qualifier)
- Partner API Keys: GA April 30 (correct — matches brief)
- SaaS Fed v2: **LIVE NOW** (brief says BETA/EARLY ACCESS pending PM)

The Reddit post (`phase-34-reddit-post.md`, approved bb21fed0) uses no qualifier — implies GA.
The HN post (`phase-34-hn-show-hn.md`, approved bb21fed0) says "Phase 34 ships two features" — implies live.

---

## Specific Conflicts

| Feature | Internal brief | External announcement | Severity |
|---------|---------------|----------------------|----------|
| Tool Trace | BETA | "live now" | HIGH — contradicts "beta" guidance |
| Platform Instructions | BETA | "live now" | HIGH — contradicts "beta" guidance |
| Partner API Keys | BETA (shipping Apr 30) | "GA April 30" | LOW — aligns |
| SaaS Fed v2 | BETA or EARLY ACCESS | "live now" | MEDIUM — SaaS Fed v2 community copy not ready |

---

## What This Means for Posting

**Cannot post until PM resolves this:**
- Discord announcement: all four features announced as "live now"
- Reddit/HN posts: Tool Trace framed as shipped/GA
- Social posts (when X credentials are available): would use "live now" framing

**Risk if we post as "GA/live" while internal brief says "BETA":**
- If PM decides "BETA" — external posts must be pulled/edited retroactively
- Community members who see "beta" elsewhere may call out inconsistency
- Enterprise buyers reading "live now" then finding beta labeling may lose trust

---

## Questions for PM

1. Is Tool Trace GA or BETA? (Internal brief says BETA, external assets say live)
2. Is Platform Instructions GA or BETA? (Internal brief says BETA, external assets say live)
3. Is SaaS Fed v2 ready for external community copy at all? (Brief says do NOT draft copy)
4. Should Partner API Keys be "BETA — GA April 30" or "shipping April 30"?

---

## Community Manager Posture

Until PM confirms:
- All Phase 34 GA posting is **ON HOLD**
- Response queue (`phase-34-community-response-queue.md`) is built and ready
- All assets are committed locally on `marketing/phase-34-launch-prep`
- Git push blocked — but even if unblocked, will not post until gate clears

---

*Found: 2026-04-23. PM must resolve before any public launch post.*