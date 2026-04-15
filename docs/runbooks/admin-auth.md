# Admin auth middleware reference

Two Gin middleware variants gate admin-style routes on the platform. Pick the
right one — they have different security contracts.

## `middleware.AdminAuth(db.DB)` — strict bearer-only

Required for any route where a forged request could:

- Leak prompts or memory (`GET /bundles/export/:id`, `GET /events*`)
- Create or mutate workspaces (`POST /workspaces`, `DELETE /workspaces/:id`, `POST /bundles/import`, `POST /templates/import`, `POST /org/import`)
- Leak operational intelligence (`GET /admin/liveness`)
- Touch approvals, secrets, or schedules at the cross-workspace level

**Contract:**

1. Reads `Authorization: Bearer <token>` and validates against `workspace_auth_tokens` via `wsauth.ValidateAnyToken`
2. **No fallback.** Missing or invalid bearer → 401
3. Lazy-bootstrap fail-open: if `HasAnyLiveTokenGlobal` returns 0 (fresh install / rolling upgrade), the route is open. First token issued to any workspace activates enforcement for every route.

**DO NOT use Origin header or session-cookie fallbacks here.** That reopens every route to curl-based spoofing — CORS is a browser-only defence, not a server-side auth signal.

## `middleware.CanvasOrBearer(db.DB)` — softer, canvas-friendly

**Only** for cosmetic routes where a forged request has zero data / security impact.

Currently used on:

| Route | Why soft is OK |
|-------|----------------|
| `PUT /canvas/viewport` | Viewport corruption resets on the next browser refresh. No data exposure, no resource creation. |

**Contract:**

1. Reads `Authorization: Bearer <token>` first. If present but **invalid**, returns 401 — **no fall-through** to the Origin path. (This was a CanvasOrBearer bug fixed during code review; preserved as the invariant.)
2. Empty bearer → check `Origin` header against `CORS_ORIGINS` env var. Exact-match only. Empty Origin does not pass.
3. Lazy-bootstrap fail-open identical to `AdminAuth`.

**The Origin check is NOT a strict auth boundary.** Any non-browser client (curl, an attacker tool) can forge the `Origin` header. CORS protects the browser from reading the response, not the server from receiving the request. Apply `CanvasOrBearer` only to routes where a curl attacker with knowledge of the canvas origin could do nothing harmful.

### When to add a new route to `CanvasOrBearer`

Ask these three questions. **All three** must be yes or the route belongs behind strict `AdminAuth`:

1. Can a browser at `https://<tenant>.moleculesai.app` need this route without a bearer token? (If not, just use `AdminAuth` — browsers can send bearers via the session-cookie auth flow once that lands.)
2. If a non-browser attacker forged `Origin: https://<tenant>.moleculesai.app`, would the worst-case outcome be purely cosmetic — recoverable with a browser refresh and no data exposure?
3. Is there no tenant isolation concern (cross-org data leak) on this route?

If yes/yes/yes → `CanvasOrBearer` is acceptable. Document the rationale in the PR that adds it, and add the route to the table above in the same PR.

## Relationship to `WorkspaceAuth`

`WorkspaceAuth` is the `/workspaces/:id/*` sub-route middleware. Different contract entirely: it binds a bearer token to a specific workspace ID so workspace A's token can't hit workspace B's sub-routes. Used for all `/workspaces/:id/*` paths except the A2A proxy (which has its own `CanCommunicate` access-control layer).

AdminAuth accepts **any** valid workspace bearer (it's a global gate). WorkspaceAuth accepts only the bearer for the **specific** `:id` in the URL path.

## Known gap (Phase H follow-up)

`CanvasOrBearer` is a tactical fix for the #168 canvas-regression problem. The proper long-term path is **session-cookie-accepting AdminAuth**: extend `AdminAuth` to validate the `mcp_session` cookie via `auth.Provider.VerifySession` (WorkOS in prod, DisabledProvider in dev). That would give the full list of admin routes browser compatibility without an Origin-based workaround. Tracked as a Phase H item once the SaaS control plane is the primary deployment surface.

## Related PRs and issues

- #138 — first canvas regression (PATCH /workspaces/:id), fixed with field-level authz in the handler (`WorkspaceHandler.Update`)
- #164 — CRITICAL anonymous workspace creation via unauthenticated `POST /bundles/import`
- #165 — HIGH topology disclosure via unauthenticated `GET /events` and `GET /bundles/export/:id`
- #166 — MEDIUM viewport corruption / liveness leak
- #167 — first auth-gate batch, strict `AdminAuth` on 5 routes
- #168 — canvas regression from the strict gating
- #190 — HIGH unauthenticated `POST /templates/import`
- #194 — rejected Origin-fallback approach (would have reopened #164)
- #203 — the `CanvasOrBearer` middleware, route-split approach, only on `PUT /canvas/viewport`
- #228 — code-review follow-up: CanvasOrBearer invalid-bearer fall-through fix
