# Platform Instructions — Plan Gating Note
**Date:** 2026-04-23
**Source:** Code review + staged blog post
**Status:** CONFIRMED — Enterprise-only

---

**Platform Instructions is gated to Enterprise plans.**

Code basis: All CRUD endpoints (`POST/PUT/DELETE/GET /instructions`) live under `adminInstr` — a router group protected by `AdminAuth` middleware. `AdminAuth` requires org-level admin credentials (WorkOS session or org API key). This means only org admins can create or modify Platform Instructions rules.

The `/instructions/resolve` endpoint (used by workspace agents at startup) uses `wsAuth` — workspace-level authentication. Workspaces can retrieve their own resolved instructions, but the CRUD API that creates/modifies rules requires org admin access.

The staged blog post (`docs/blog/2026-04-23-platform-instructions-governance/index.md` on `marketing/phase-34-launch-prep`, line 94) states explicitly: "Platform Instructions are available on **Enterprise plans**."

There is no feature flag or plan-tier check in the handler code — the gating is enforced by `AdminAuth` middleware on the CRUD routes. This means the restriction is architectural, not a config flag that could be flipped.

**Implication for social copy:** Posts 4 and 5 in the Phase 34 GA social thread ("Platform Instructions: Enterprise plans") are correct. Do not remove the plan qualifier.

---

*Source: `workspace-server/internal/router/router.go:376` (AdminAuth on CRUD routes), `workspace-server/internal/handlers/instructions.go` (no plan/tier/enitle/enterprise references — gating is purely middleware-enforced). Staged blog post line 94.*