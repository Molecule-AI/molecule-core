# Organization API Keys — Follow-up Work

> Tracked improvements to the beta `org_api_tokens` system. Each item
> has a rationale + sketch implementation + rough effort estimate.
> Ordered by priority.

## 1. Role scoping (P1 — next after beta signal)

**Problem:** Today every token is full-admin. A token given to a
simple read-only monitoring script is as dangerous as one given to
a deploy bot. No way to hand an AI agent a token that lets it read
workspace state but not nuke the org.

**Proposal:** Add a `role` column to `org_api_tokens`:

```sql
ALTER TABLE org_api_tokens
  ADD COLUMN role TEXT NOT NULL DEFAULT 'admin'
  CHECK (role IN ('admin', 'editor', 'reader'));
```

- `admin` — current behavior (all AdminAuth routes)
- `editor` — workspace CRUD + secrets + approvals, but NOT mint/
  revoke org tokens (closes the self-escalation loop)
- `reader` — GETs only, no mutations

New middleware wrapper `RequireRole(role)` checks token's row
against the route's required minimum. Extend AdminAuth to stash
the resolved role on `c.Set("org_token_role", r)`.

**Effort:** ~200 LOC + migration + UI role-picker in
`OrgTokensTab.tsx`. Breaking change for existing tokens (default
to `admin` preserves behavior).

## 2. Per-workspace binding (P1)

**Problem:** An org-admin token that only needs to touch one
workspace is overkill. AWS IAM equivalent: "this key can only read
bucket foo".

**Proposal:** Optional `workspace_id` FK on the token. When set,
AdminAuth + WorkspaceAuth both accept the token ONLY for routes
scoped to that workspace (`/workspaces/<id>/*`). Tokens with
`workspace_id = NULL` behave as today (full-org).

```sql
ALTER TABLE org_api_tokens
  ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE;
```

Cascade delete means revoking a workspace revokes its scoped
tokens automatically. UI adds a workspace dropdown at mint time.

**Effort:** ~250 LOC. Pairs naturally with role scoping.

## 3. Expiry (P2)

**Problem:** Long-lived tokens are a liability. "Mint this key for
this one deploy and die after 1 hour" is a common ask.

**Proposal:** Optional `expires_at` on the row, enforced in the
hot-path query:

```sql
WHERE token_hash = $1 AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > now())
```

UI: mint form has "Expires in: [Never / 1h / 1d / 30d]" picker.
Show time-left on the list view; flag soon-to-expire in amber.

**Effort:** ~80 LOC. Additive; existing tokens have NULL = never.

## 4. Usage metrics (P2)

**Problem:** `last_used_at` is the only observation we have. Users
want to see what a token is doing — which paths, from which IPs,
how often — so they can detect anomalies.

**Proposal:** Async counter writes on every successful Validate.
New table:

```sql
CREATE TABLE org_api_token_usage (
  token_id       UUID REFERENCES org_api_tokens(id) ON DELETE CASCADE,
  hour           TIMESTAMPTZ NOT NULL,  -- truncated to hour
  request_count  BIGINT NOT NULL DEFAULT 0,
  last_path      TEXT,
  last_ip        INET,
  last_user_agent TEXT,
  PRIMARY KEY (token_id, hour)
);
```

`ON CONFLICT DO UPDATE SET request_count = request_count + 1` —
atomic counter upserts, one row per token-hour. UI graphs last 30
days per token.

**Effort:** ~150 LOC + background sweep to prune >90-day rows.

## 5. Rotation webhooks (P3)

**Problem:** When a user revokes a token, integrations using it
get 401 with no warning. Big ones want "you're about to lose
access, here's 60s to rotate" signals.

**Proposal:** Soft-revoke tier. Revoke now accepts
`?drain_seconds=60`. Token enters a `draining` state (still valid
but a warning header `X-Molecule-Token-Draining: true` is added to
every response). After drain window, fully revoked.

Alternative / complement: webhook URL on the token. POST to it
when revoked. Safer because no drain period.

**Effort:** ~200 LOC. Webhook variant requires retry logic +
delivery audit.

## 6. Capture WorkOS user_id in created_by (P2, quick win)

**Problem:** Today, tokens minted via the canvas UI log
`created_by: "session"` — we know it was a session but not whose.
Post-incident review can't link a token back to a user.

**Proposal:** Thread the WorkOS user_id from the session-auth
verification through to the handler. The CP's
`/cp/auth/tenant-member` already returns `user_id`; stash it on
the gin context in `session_auth.go`; handler reads it for
`created_by`.

```go
// session_auth.go after successful verify
c.Set("session_user_id", body.UserID)

// handler
if v, ok := c.Get("session_user_id"); ok {
    createdBy = "session:" + v.(string)
}
```

**Effort:** ~20 LOC. Unblocks Important follow-up #6 from today's
code review.

## 7. Mint-rate limit (P3)

**Problem:** A compromised session or admin token could mint
thousands of org tokens quickly, making forensic cleanup painful.

**Proposal:** Rate limit mint calls per-org: max N tokens per 5 min.
Existing `middleware/ratelimit` package does exactly this — bind
the limiter to the mint route with a low ceiling.

**Effort:** ~30 LOC. Do this before #5 — revoke-storms could hit
the same pattern.

## 8. Audit log (P2)

**Problem:** Token revocation is logged to stdout. That's fine for
Railway's retention window but ops want a queryable audit log.

**Proposal:** New table `org_token_audit` with (token_id, action,
actor, occurred_at). Write on mint/revoke. Surface in admin
diagnostics endpoint.

**Effort:** ~100 LOC + lightweight read API.

## 9. CLI for local development (P3)

**Problem:** Developers running canvas locally can't easily mint
and use org tokens against their dev tenant because the UI
requires a WorkOS session.

**Proposal:** `molecli org-token create --name <label>` uses
`ADMIN_TOKEN` from env + `MOLECULE_ORG_URL` to mint. Same API,
scripts-friendly.

**Effort:** ~80 LOC in molecli + a line in the docs guide.

## 10. Migrate ADMIN_TOKEN to org_api_tokens table (P4 — long-term)

**Problem:** `ADMIN_TOKEN` as an env var is a special case that
every auth tier has to handle. Once org tokens are feature-
complete (roles, expiry, binding), the env-var token is redundant
and complicates the auth code.

**Proposal:** Bootstrap the tenant by inserting a row labeled
`bootstrap` into `org_api_tokens` at provision time with the
current ADMIN_TOKEN value's hash. Remove the env-var check entirely
from AdminAuth. `ADMIN_TOKEN` becomes just "the initial token that
happens to be stored as a normal row".

Requires: roles + expiry shipped first (bootstrap token needs to
be demarcated as revocable-but-permanent-by-default).

**Effort:** ~150 LOC once prerequisites land.

---

## Tracked issues to file

Each of the above should become a GitHub issue when we're ready to
work it. One-liner label for the batch: `area:org-api-keys`.

## Non-goals

Explicit list of things we do NOT want to add:

- JWT / signed tokens. Opaque bearers + DB lookup is simpler and
  matches every other token type in the system.
- OAuth scopes. We're not a third-party OAuth provider; this is
  for internal integrations only.
- IP allow-lists per token. Captured nominally by the usage log
  (#4) for detection, but enforcement adds operational friction
  (customer VPN changes → all tokens break).
