# Partner API Keys — Rate Limits Research Note

**Date:** 2026-04-23  
**Author:** Marketing Lead (direct code search)  
**Status:** NOT FOUND IN molecule-core — escalated to PM

---

## Finding

Partner key rate limits are **not configurable from `molecule-core`**. The `/cp/admin/partner-keys` endpoint is served by the private `molecule-controlplane` repository, which is not checked out in the Marketing Lead workspace.

## What IS in molecule-core

The workspace-server `router.go` has a global IP-based rate limiter:
- Default: **600 requests/minute per IP** (configurable via `RATE_LIMIT` env var)
- The partner-key routes (`/cp/admin/*`) are NOT proxied through the workspace-server tenant proxy (`cpProxyAllowedPrefixes` in `cp_proxy.go` does not include `/cp/admin/partner-keys`)
- Partner-key-specific rate limiting (the PMM brief mentions "per-key limiter, separate from session limits") must live in the controlplane

## What's Needed

PM must confirm from the controlplane codebase:
1. Per-key rate limit (requests/min or requests/hour for `POST` and `DELETE` on partner keys)
2. Whether rate limits vary by partner tier

## Marketing Impact

Until PM confirms:
- DevRel demo at `docs/devrel/phase-34-partner-api-keys-demo.md` retains `[RATE LIMIT TBD]` placeholder
- No copy should claim specific rate limits
- Partner onboarding guide should say "Rate limits are enforced per key — contact the partner team for current limits"

---

*Marketing Lead 2026-04-23 — escalated to PM via status report.*
