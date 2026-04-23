# CRITICAL: Phase 34 GA vs Beta Label Conflict

**Date:** 2026-04-23  
**Status:** 🔴 UNRESOLVED — Requires PM or PMM decision before April 30 publish  
**Raised by:** Marketing Lead  

---

## The Conflict

Two sets of documents make contradictory claims about the Phase 34 launch label:

### Internal positioning docs say BETA

`docs/marketing/briefs/phase34-positioning.md` (lines 83–85):
> "Partner API Keys are **BETA** — do not claim GA in press materials."  
> "Tool Trace and Platform Instructions shipped via PR #1686 — **BETA**."  
> "SaaS Federation v2 — **BETA** or **EARLY ACCESS**, pending PM label confirmation."

`docs/marketing/briefs/phase34-messaging-matrix.md`:
> Partner API Keys HN/Reddit framing: "**Do NOT claim GA.** Use 'beta' or 'now available.'"  
> Tool Trace HN/Reddit framing: "Be honest: **this is a beta feature.**"

### Approved external-facing launch posts say GA

| File | Language |
|------|----------|
| `docs/marketing/launches/phase-34-hn-show-hn.md` | "Molecule AI – every agent tool call now logged in A2A response (no SDK, **GA today**)" — APPROVED ML 2026-04-23 |
| `docs/marketing/launches/phase-34-reddit-post.md` | "Molecule AI Phase 34: built-in agent execution tracing + programmatic org provisioning API (**GA today**)" — APPROVED ML 2026-04-23 |
| `docs/marketing/social/2026-04-26-phase34-ga-launch/social-copy.md` | "🚀 **Phase 34 is GA.**" — APPROVED ML 2026-04-23 |
| `docs/marketing/launches/phase-34-community-announcement.md` | "Partner API Keys (`mol_pk_*`) — **GA April 30**" |
| `docs/marketing/blog/2026-04-30-partner-api-keys-ga.md` | Blog title contains "**GA**" — published on April 30 |

---

## Resolution Required

**PM or PMM must confirm the authoritative launch label for each feature before April 30:**

| Feature | Internal brief says | External posts say | Confirm: |
|---------|--------------------|--------------------|---------|
| Tool Trace | Beta | GA | GA / Beta? |
| Platform Instructions | Beta | GA | GA / Beta? |
| Partner API Keys | Beta — Do NOT claim GA | GA today | GA / Beta? |
| SaaS Federation v2 | BETA or Early Access (TBD) | "improved multi-org federation" (vague) | GA / Beta / EA? |

---

## If Decision Is GA (External posts are correct)

No changes needed to launch assets. Update internal positioning brief and messaging matrix to reflect GA status. Delete or update the internal BETA language so it doesn't contradict future copy.

---

## If Decision Is Beta (Internal briefs are correct)

The following files must be edited before April 30 publish:

1. **`docs/marketing/launches/phase-34-hn-show-hn.md`**
   - Title: "GA today" → "now in beta" or "beta today"
   - Body: "GA today" → "shipping in beta today"

2. **`docs/marketing/launches/phase-34-reddit-post.md`**
   - Title: "GA today" → "now available in beta"
   - Body: soften GA language throughout

3. **`docs/marketing/social/2026-04-26-phase34-ga-launch/social-copy.md`**
   - Post 1: "Phase 34 is GA" → "Phase 34 is live" or "Phase 34 is in beta"
   - Multiple posts reference "GA" — audit required

4. **`docs/marketing/launches/phase-34-community-announcement.md`**
   - "Partner API Keys — GA April 30" → "Partner API Keys — beta April 30"

5. **`docs/marketing/blog/2026-04-30-partner-api-keys-ga.md`**
   - Rename file and update title — remove "GA" from title
   - Update body language throughout

6. **`docs/marketing/launches/partner-onboarding-guide.md`**
   - Update any "GA" references to "beta"

7. **`docs/marketing/launches/phase-34-community-faq.md`**
   - Q&A answers referencing GA status

---

## Context Note

The GA label was used in all external launch assets written 2026-04-23 during the Phase 34 launch prep sprint. The internal positioning briefs (also 2026-04-23) predate the external copy and may reflect an earlier working assumption. The external assets reflect a subsequent (implicit) GA decision.

**Most likely resolution:** PM confirms GA, internal briefs are updated to reflect the final decision.  
**Worst case:** PM confirms Beta, 7 external files need editing before April 30.

PM + PMM: please confirm via `/docs/marketing/briefs/phase34-ga-vs-beta-conflict.md`.

---

*Marketing Lead 2026-04-23. PM and PMM unreachable via A2A at time of writing (delegation failures). Surfaced for human decision.*
