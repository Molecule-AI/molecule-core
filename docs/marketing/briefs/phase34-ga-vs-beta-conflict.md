# CRITICAL: Phase 34 GA vs Beta Label Conflict

**Date:** 2026-04-23  
**Status:** 🔴 UNRESOLVED — Requires PM decision before April 30 publish  
**Raised by:** Marketing Lead + Community Manager  
**Hold:** Phase 34 GA posting ON HOLD pending PM sign-off

---

## The Conflict

Two sets of documents make contradictory claims about the Phase 34 launch label:

### Internal positioning docs say BETA

`docs/marketing/briefs/phase34-positioning.md`:
> *"Partner API Keys are **BETA** — do not claim GA in press materials. Use 'now available in beta' or 'shipping April 30, 2026.'"*  
> *"Tool Trace and Platform Instructions shipped via PR #1686 — **BETA**."*  
> *"SaaS Federation v2 — **BETA** or **EARLY ACCESS**, pending PM label confirmation."*

`docs/marketing/briefs/phase34-messaging-matrix.md`:
> *"HN/Reddit framing: 'Be honest: this is a beta feature.'"* (Tool Trace)  
> *"SaaS Federation v2: **Do NOT draft community copy for this feature** until PM confirms the GA/beta/alpha label."*

### Approved external-facing launch posts say GA

| File | Language |
|------|----------|
| `docs/marketing/launches/phase-34-hn-show-hn.md` | "GA today" — APPROVED ML 2026-04-23 |
| `docs/marketing/launches/phase-34-reddit-post.md` | "GA today" — APPROVED ML 2026-04-23 |
| `docs/marketing/social/2026-04-26-phase34-ga-launch/social-copy.md` | "Phase 34 is GA" — APPROVED ML 2026-04-23 |
| `docs/marketing/launches/phase-34-community-announcement.md` | "Partner API Keys — GA April 30" |
| `docs/marketing/blog/2026-04-30-partner-api-keys-ga.md` | Blog title contains "GA" |

---

## Feature-by-Feature Conflict Table

| Feature | Internal brief | External assets | Severity |
|---------|---------------|-----------------|----------|
| Tool Trace | BETA | "live now" / GA | 🔴 HIGH |
| Platform Instructions | BETA | "live now" / GA | 🔴 HIGH |
| Partner API Keys | BETA (shipping Apr 30) | "GA April 30" | 🟡 LOW — aligns on date |
| SaaS Fed v2 | BETA or EARLY ACCESS | "live now" | 🟠 MEDIUM |

---

## Questions for PM

1. Is Tool Trace **GA or BETA**? (Internal brief says BETA, external assets say live)
2. Is Platform Instructions **GA or BETA**? (Internal brief says BETA, external assets say live)
3. Is SaaS Fed v2 ready for external community copy at all? (Brief says do NOT draft copy)
4. Should Partner API Keys be "BETA — GA April 30" or "shipping April 30"?

---

## Resolution Paths

### If Decision Is GA (external posts are correct)
No changes needed to launch assets. Update internal positioning brief and messaging matrix to reflect GA status.

### If Decision Is Beta (internal briefs are correct)
The following files must be edited before April 30 publish:

1. `docs/marketing/launches/phase-34-hn-show-hn.md` — soften "GA today" → "now in beta" / "shipping today"
2. `docs/marketing/launches/phase-34-reddit-post.md` — soften "GA today" throughout
3. `docs/marketing/social/2026-04-26-phase34-ga-launch/social-copy.md` — audit all GA references
4. `docs/marketing/launches/phase-34-community-announcement.md` — "GA April 30" → "beta April 30"
5. `docs/marketing/blog/2026-04-30-partner-api-keys-ga.md` — rename file + update title and body
6. `docs/marketing/launches/partner-onboarding-guide.md` — update any GA references
7. `docs/marketing/launches/phase-34-community-faq.md` — Q&A answers referencing GA status

---

## Context Note

The GA label was used in all external launch assets written 2026-04-23. The internal positioning briefs (also 2026-04-23) predate the external copy and may reflect an earlier working assumption.

**Most likely resolution:** PM confirms GA, internal briefs are updated to match.  
**Worst case:** PM confirms Beta, 7 external files need editing before April 30.

---

## Current Posture

All Phase 34 GA posting is **ON HOLD** pending PM sign-off. Assets are committed locally on `marketing/phase-34-launch-prep`. Once PM resolves the GA vs Beta question, the hold can lift.

Response queue (`phase-34-community-response-queue.md`) is built and ready for when the hold lifts.

---

*Marketing Lead + Community Manager, 2026-04-23. PM must resolve before any public Phase 34 launch post.*
