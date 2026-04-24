# Platform Instructions — Plan Gating Verification Note

**Date:** 2026-04-23  
**Author:** Marketing Lead (direct code review)  
**Status:** CONFIRMED — ALL PLANS

---

## Finding

Platform Instructions is **not gated to Enterprise plans**. It is available on all Molecule AI plans.

## Evidence

Reviewed `workspace-server/internal/handlers/instructions.go` (full file, 277 lines) — zero plan/tier checks anywhere in the handler:

- No `requireEnterprise()` middleware
- No subscription tier check on `Create`, `Update`, `List`, `Resolve`, or `Delete` handlers
- Auth is workspace-bearer only — `wsAuth` middleware validates that the caller holds a valid token for the target workspace ID, but applies no plan restriction

The only constraints are:
- Content capped at 8,192 chars (`maxInstructionContentLen`)
- Title capped at 200 chars
- Scope must be `global` or `workspace` (team scope reserved but not yet implemented)

## Marketing Impact

The following copy contained incorrect "Enterprise plans only" claims and **must be corrected**:

1. `docs/marketing/social/2026-04-26-phase34-ga-launch/social-copy.md` — Post 5 + LinkedIn post
2. Any other copy that asserts Platform Instructions is Enterprise-only

**Correct claim:** "Available on all plans" — same as Tool Trace.

## Action Taken

All instances corrected as of 2026-04-23:

| File | What was fixed |
|------|---------------|
| `docs/marketing/social/2026-04-26-phase34-ga-launch/social-copy.md` | Removed "Enterprise plans only" from Post 5 and LinkedIn body |
| `docs/blog/2026-04-23-tool-trace-platform-instructions/index.md` | "enterprise-only" and "Enterprise plans" → "all plans" (×2 locations) |
| `docs/blog/2026-04-23-platform-instructions-governance/index.md` | "enterprise-only" removed from section header; "Enterprise plans" → "all plans" in Get Started + footer |
| `docs/marketing/blog/2026-04-23-tool-trace-platform-instructions.md` | Confirmed clean — written after the gating note, no error present |
| `docs/marketing/briefs/2026-04-22-a2a-enterprise-deep-dive-seo-brief.md` | No enterprise gating error in SEO brief |

Sweep confirmed: zero "enterprise-only" or "Enterprise plans" claims remaining in any blog or social launch copy as of commit 907199d4.

---

*Marketing Lead 2026-04-23.*
