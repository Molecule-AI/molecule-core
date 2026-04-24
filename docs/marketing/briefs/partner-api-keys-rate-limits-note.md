# Partner API Keys — Rate Limits Note
**Date:** 2026-04-23 | **Owner:** PMM | **Source:** Code review
**File:** `docs/marketing/briefs/partner-api-keys-rate-limits-note.md`

---

## Rate Limit

**Default rate limit: 60 requests per minute per Partner API Key.**

This is the default configured in `docs/architecture/partner-api-keys.md` (line 219):

> "Partner keys have per-key rate limits (default: 60 req/min, configurable)."

The rate limiter is a separate middleware from the session-based rate limiter, so partner traffic does not compete with browser user rate limits. Each partner key has its own independent bucket.

**Source:** `docs/architecture/partner-api-keys.md` (architecture design doc)
**Go implementation reference:** `workspace-server/internal/middleware/ratelimit.go` — generic token bucket rate limiter (used for PartnerKeys with configurable rate + interval per key instance)

---

## Rate Limit Behavior

- **Per-key, not per-org** — Each `mol_pk_*` key has its own rate limit counter. One key going over limit does not affect other partner keys.
- **Separate from session rate limiter** — Partner API Key traffic uses a separate rate limit bucket from organic browser/session traffic.
- **Configurable** — The rate limit is set per key at creation time (the `rate_limit` field in the key creation payload). Default is 60 req/min if not specified.
- **429 response on exceed:** `{"error": "rate limit exceeded", "retry_after": <seconds>}`
- **Rate limit headers:** `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` on all responses.

---

## Known Gaps

- **Go implementation not found** — The rate limiter middleware (`ratelimit.go`) is a generic token bucket. The partner-specific rate limit enforcement (keying on `partner.ID`) was not found in a concrete handler file — the architecture doc describes the pattern but no `partner_key_ratelimit.go` or equivalent was found in `workspace-server/internal/`. PM should confirm the actual Go implementation before citing specific endpoint behavior.
- **Rate limit ceiling not documented** — Maximum configurable rate per key not found. PM must confirm if there is a platform-level cap.

---

## Implication for DevRel Demo

The `[RATE LIMIT TBD]` placeholder in `docs/devrel/phase-34-partner-api-keys-demo.md` can be filled with: **60 req/min per key (default, configurable)** — sourced from architecture doc.

Update the demo script: after PM confirms Go implementation, update this note with the specific file + line number.

---

*Source: `docs/architecture/partner-api-keys.md` lines 217–232. PM confirmation needed on Go implementation file + line number before citing in externally published copy.*
