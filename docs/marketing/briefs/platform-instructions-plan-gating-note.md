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

`docs/marketing/social/2026-04-26-phase34-ga-launch/social-copy.md` updated 2026-04-23 by Marketing Lead to remove Enterprise-only language from Post 5 and LinkedIn.

---

*Marketing Lead 2026-04-23.*
