# Staging full-SaaS E2E

`tests/e2e/test_staging_full_saas.sh` provisions a fresh org per run, exercises the workspace lifecycle end-to-end, then tears the org down and asserts leak-free. Runs in CI via `.github/workflows/e2e-staging-saas.yml`.

## What it covers

| Step | What it verifies |
|---|---|
| 1. Accept terms (POST `/cp/auth/accept-terms`) | Session cookie valid, ToS gate honours idempotent replay |
| 2. Create org (POST `/cp/orgs`) | Slug validation, member insert, billing gate, quota |
| 3. Wait for provisioning | CP tenant EC2 boot + cloudflared tunnel + DNS + TLS (~5–10 min cold) |
| 4. Tenant health (GET `/health` on new tenant URL) | Cert chain OK, TenantGuard + session-auth wired |
| 5. Provision parent workspace | SaaS provision path (CP RunInstances, EC2 bootstrap, runtime register) |
| 6. Provision child workspace under parent | `parent_id` relationship, team-hierarchy |
| 7. Wait both online | Workspace sweeper + register handler + token bootstrap |
| 8. A2A round-trip (POST `/workspaces/:id/a2a`) | Full LLM loop — registration, MCP tools, provider auth, response shape |
| 9. HMA memory write+read | `/memories` scope routing, awareness namespace, persistence |
| 9b. Peers + activity smoke | Route registration + activity-log write path |
| 10. Teardown | `DELETE /cp/admin/tenants/:slug` + leak assertion |

If any step fails, the EXIT trap tears down the org anyway.

## Required GitHub Actions secrets

Both are at **Settings → Secrets and variables → Actions → Repository secrets**:

### `MOLECULE_STAGING_SESSION_COOKIE`

A valid `molecule_cp_session` cookie for a **test user** that:

- is on the staging beta allowlist (or `BETA_GATE_ENABLED=false` on staging)
- has already accepted the current terms version (the script re-accepts idempotently but can't bootstrap from unaccepted)
- has under-quota owned orgs

**How to extract:**

1. In an incognito window, sign in at `https://staging-api.moleculesai.app/cp/auth/login` with the test user.
2. DevTools → Application → Cookies → `https://staging-api.moleculesai.app`
3. Copy the `molecule_cp_session` value (base64-looking blob).
4. Paste as the secret value. Do not include the `molecule_cp_session=` prefix.

**Rotation:** WorkOS sessions don't expire until the user signs out or the refresh token revokes. A 90-day rotation schedule is safe.

### `MOLECULE_STAGING_ADMIN_TOKEN`

The `CP_ADMIN_API_TOKEN` env var currently set on the Railway **staging** molecule-platform → controlplane service.

**How to extract:**

```
railway variables --service controlplane --environment staging --kv | grep CP_ADMIN_API_TOKEN
```

Used exclusively for teardown (`DELETE /cp/admin/tenants/:slug`) and leak detection (`GET /cp/admin/orgs`). Write access, treat like prod admin.

## Running locally

```
export MOLECULE_CP_URL=https://staging-api.moleculesai.app
export MOLECULE_SESSION_COOKIE="…"
export MOLECULE_ADMIN_TOKEN="…"
# Optional: keep the org for post-mortem inspection
export E2E_KEEP_ORG=1
bash tests/e2e/test_staging_full_saas.sh
```

`E2E_KEEP_ORG=1` skips teardown so you can poke at the provisioned tenant yourself. **Never set this in CI** — staging will fill with orphans.

## Cost

- Full run: ~20 min wall clock
- Compute: ~12 min of t3.small tenant EC2 + ~4 min of per-workspace EC2 × 2 = ~20 t3.small-minutes ≈ **$0.007/run**
- Daily (nightly cron + PR runs ≈ 5/day): **~$0.04/day**
- Hard timeout (30 min workflow timeout + per-request curl timeouts) caps runaway cost

## Known gaps (follow-ups)

- Canvas UI tabs not covered — separate Playwright workflow in `e2e-staging-canvas.yml` (todo)
- Delegation end-to-end (parent calls `delegate_task` MCP tool against child) — not in this run because it needs a real LLM loop and doubles runtime cost
- Claude Code runtime test — currently only Hermes is exercised to keep wall time down; pass `runtime: claude-code` via workflow_dispatch to test it
- No screenshot/trace capture on failure — add if CI signal is noisy
