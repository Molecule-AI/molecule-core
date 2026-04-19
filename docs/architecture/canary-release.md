# Canary release pipeline

How a workspace-server code change reaches the prod tenant fleet — and how to stop it if something's wrong.

## The loop

```
PR merged to staging → main
      │
      ▼
publish-workspace-server-image.yml   ← pushes :staging-<sha> ONLY
      │                                (NOT :latest — prod is untouched)
      ▼
Canary tenants auto-update to :staging-<sha>
      │   (5-min auto-updater cycle on each canary EC2)
      ▼
canary-verify.yml waits 6 min, runs scripts/canary-smoke.sh
      │
      ├─► GREEN → crane tag :staging-<sha> → :latest
      │                                       │
      │                                       ▼
      │                           Prod tenants auto-update within 5 min
      │
      └─► RED   → :latest stays on prior good digest
                  GitHub Step Summary flags the rejected sha
                  Ops fixes forward OR rolls back manually
```

## Canary fleet

Lives in a separate AWS account (`molecule-canary`, `004947743811`) via an assumed role (`MoleculeStagingProvisioner`). The CP's `is_canary` org flag routes provisioning there; every other org goes to the default staging account. See `docs/architecture/saas-prod-migration-2026-04-19.md` for the account bootstrap.

Canary tenants are configured to pull `:staging-<sha>` (not `:latest`) via `TENANT_IMAGE` on their provisioner, so they ingest each new build before prod does.

## Smoke suite

`scripts/canary-smoke.sh` hits each canary tenant (URL + ADMIN_TOKEN pair) and asserts:

- `/admin/liveness` returns a subsystems map (tenant booted, AdminAuth reachable)
- `/workspaces` returns a JSON array (wsAuth + DB healthy)
- `/memories/commit` + `/memories/search` round-trip (encryption + scrubber)
- `/events` admin read (C4 fail-closed proof)
- `/admin/liveness` without bearer → 401 (C4 regression gate)

Expand by editing the script — each `check "name" "expected" "$response"` call is one line.

## Adding a canary tenant

1. `POST /cp/orgs` — create the org normally (is_canary defaults to false)
2. `POST /cp/admin/orgs/<slug>/canary` with `{"is_canary": true}` — admin only, refuses to flip if already provisioned
3. Re-trigger provision (or delete + recreate if the org was already provisioned into staging) — the fresh EC2 lands in account `004947743811`

Then set repo secrets:
- `CANARY_TENANT_URLS` — append the new tenant's URL
- `CANARY_ADMIN_TOKENS` — append its ADMIN_TOKEN in the same position

## Rolling back `:latest`

When canary was green but something surfaces post-promotion, retag `:latest` to a prior digest:

```bash
export GITHUB_TOKEN=ghp_...    # write:packages
scripts/rollback-latest.sh 4c1d56e  # retags both platform + tenant images
```

`scripts/rollback-latest.sh` pre-checks that `:staging-<sha>` exists before moving `:latest`, and verifies the digest after the move. Prod tenants pick up the rolled-back image on their next 5-min auto-update.

A post-mortem should always include:
- the commit sha that broke
- why canary didn't catch it (new code path the smoke suite doesn't exercise?)
- whether the smoke suite should grow a new check to prevent the same class of bug

## What this gate doesn't catch

- Bugs that only surface under prod-only data (customer workloads with scale or shape canary doesn't produce). Canary uses real traffic shapes but can't simulate weeks of accumulated state.
- Config drift between canary and prod (different env-var values, different feature flags). Keep canary's config deltas minimal and documented.
- Cross-tenant interactions — canary tenants run in their own AWS account, so a bug that only appears when two tenants compete for a shared resource won't reproduce here.

When these miss, `rollback-latest.sh` is the escape hatch.
