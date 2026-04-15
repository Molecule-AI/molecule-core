# SaaS secret rotation — runbook

Where each secret lives, why, and the **full rotation procedure** so a partial
update doesn't silently break production.

## Secret map

| Secret | Location(s) | Purpose |
|---|---|---|
| `FLY_API_TOKEN` | **(a)** `molecule-monorepo` GitHub Actions secret (push image to `registry.fly.io/molecule-tenant`) + **(b)** `fly secrets` on `molecule-cp` app (control plane creates + deletes tenant Fly Machines) | Any Fly Machines API call |
| `NEON_API_KEY` | `fly secrets` on `molecule-cp` | Create + delete tenant Neon branches |
| `DATABASE_URL` | `fly secrets` on `molecule-cp` | Control-plane Postgres connection (Neon `cool-sea-89357706`) |
| `TENANT_REDIS_URL` | `fly secrets` on `molecule-cp` | Injected into every tenant container as `REDIS_URL` |
| `SECRETS_ENCRYPTION_KEY` | `fly secrets` on `molecule-cp` | AES-256 key wrapping tenant DB/Redis URLs in `org_instances` (provisioner + tenant use this) |
| `GITHUB_TOKEN` | Built-in GitHub Actions token | GHCR push; rotated automatically |

## Coupled secrets — MUST rotate together

`FLY_API_TOKEN` is the one secret duplicated across systems. Rotating **only
one** will cause **silent** breakage:

- Rotating **only (a) GHA** → image publish workflow fails, but no alert; control plane keeps provisioning from the stale `latest` tag.
- Rotating **only (b) Fly secrets** → control plane's Fly API calls start erroring (`401`), tenant provisioning fails, but image publishes keep succeeding so everything *looks* fine on the build side.

## Rotation procedure — FLY_API_TOKEN

1. Generate new token:
   ```
   flyctl tokens create deploy --name molecule-cp-rotation-$(date +%Y%m%d)
   ```
2. Update **both** locations (order matters — Fly secrets first, then GHA):
   ```
   # (b) Fly secrets — triggers zero-downtime redeploy
   flyctl secrets set --app molecule-cp FLY_API_TOKEN='FlyV1 fm2_...'

   # (a) GitHub Actions secret — next workflow run uses new token
   echo 'FlyV1 fm2_...' | gh secret set FLY_API_TOKEN --repo Molecule-AI/molecule-monorepo
   ```
3. Verify:
   ```
   # Control plane can reach Fly API:
   curl https://molecule-cp.fly.dev/health
   # Trigger image publish (dispatches workflow, pushes to both registries):
   gh workflow run publish-platform-image.yml --repo Molecule-AI/molecule-monorepo
   gh run list --repo Molecule-AI/molecule-monorepo --workflow publish-platform-image --limit 1
   ```
4. Revoke the old token:
   ```
   flyctl tokens list
   flyctl tokens revoke <id-of-old-token>
   ```

## Rotation procedure — NEON_API_KEY

1. Create replacement key in Neon console → Account Settings → API Keys.
2. Update Fly secrets:
   ```
   flyctl secrets set --app molecule-cp NEON_API_KEY='napi_...'
   ```
3. Trigger a test provision (dry run — create + delete):
   ```
   curl -X POST https://molecule-cp.fly.dev/cp/orgs \
     -H 'Content-Type: application/json' \
     -d '{"slug":"keytest-'$(date +%s)'","name":"Rotation test"}'
   # Wait 60s, inspect logs:
   flyctl logs --app molecule-cp --no-tail | tail -30
   # Clean up the test org via DELETE once live
   ```
4. Revoke old key in Neon console.

## Rotation procedure — SECRETS_ENCRYPTION_KEY

**DANGEROUS**: rotating this key will invalidate every encrypted row in
`org_instances.database_url_encrypted` + `redis_url_encrypted`. Every tenant
becomes unreachable until re-provisioned.

Mitigation: we intentionally defer real KMS + key-rotation to Phase H. Until
then, **do not rotate this key unless compromised.** If compromise, procedure is:

1. Generate new key: `openssl rand -hex 32`
2. Set new key on `molecule-cp`.
3. For every row in `org_instances`: re-provision the tenant (creates fresh
   Neon branch + Fly machine). The old encrypted URLs are un-decryptable but
   irrelevant — we mint fresh ones.
4. Migration to rotate encrypted columns in-place (decrypt-with-old → encrypt-
   with-new) is Phase H work and requires envelope encryption with KMS.

## Rotation procedure — DATABASE_URL (control plane)

The Neon `molecule-cp` project has a stable primary endpoint. Rotate only if:
- Neon forces a migration
- The connection-URI password is leaked

Procedure: regenerate URI via Neon API → `flyctl secrets set DATABASE_URL=...`.
Zero-downtime (Fly applies secret via rolling restart).

## Emergency contacts

- **Fly**: billing dashboard at fly.io → Support
- **Neon**: console.neon.tech → Support
- **Upstash**: upstash.com → Support
- **GHCR**: github.com/orgs/Molecule-AI (org admins)
