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
| `RESEND_API_KEY` | `fly secrets` on `molecule-cp` | Resend REST API token used by `internal/email.ResendProvider` — GDPR erasure confirmation today; welcome + plan-change emails later. Empty → `DisabledProvider` silently no-ops all sends |
| `RESEND_FROM_EMAIL` | `fly secrets` on `molecule-cp` | RFC-5322 From line, typically `"Molecule AI <noreply@moleculesai.app>"`. Must resolve to a Resend-verified domain or sends fail with `403 domain not verified` |
| `STRIPE_API_KEY` | `fly secrets` on `molecule-cp` | `sk_live_…` secret key used by `internal/billing.StripeProvider` for customer/subscription/checkout mutations + GDPR Art. 17 cascade |
| `STRIPE_WEBHOOK_SECRET` | `fly secrets` on `molecule-cp` | `whsec_…` used by `internal/billing.verifySignature` to reject forged webhook calls. Rotated independently from the API key — Stripe treats them as separate secrets |
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

## Rotation procedure — RESEND_API_KEY

Low-blast-radius rotation — the only consumer is the transactional-email
path and sends fail loudly (the cascade logs `purge confirmation email
failed`) without breaking user-facing flows.

1. In Resend dashboard → API Keys → create a new key scoped to
   "molecule-cp production", e.g. name
   `molecule-cp-rotation-$(date +%Y%m%d)`.
2. Stage the replacement on Fly (not immediately live):
   ```
   flyctl secrets set --app molecule-cp \
     --stage RESEND_API_KEY='re_...'
   ```
   `--stage` holds the secret for the next deploy instead of restarting
   machines immediately. Skip `--stage` if you want a rolling restart
   right now.
3. Redeploy (or wait for the next image publish) — machines pick up the
   new key.
4. Trigger a real send to verify: delete a disposable test org via
   `DELETE /cp/orgs/test-rotate` and confirm the Resend dashboard shows
   the event in Emails → Logs within a minute.
5. Revoke the old key in the Resend dashboard.

### Blast-radius note

The GDPR Art. 17 cascade sends a best-effort confirmation email after
purge succeeds; a failed send is logged but does **not** flip the 204
response (purge data is already gone). This means a broken
`RESEND_API_KEY` silently skips confirmation emails — monitor the
`purge confirmation email failed` log line after any rotation.

### Domain verification

`RESEND_FROM_EMAIL` must come from a Resend-verified domain or every
send returns `403 domain not verified`. Domain verification lives in
Resend dashboard → Domains → Add Domain; Resend gives you 3 DNS records
(SPF, DKIM, DMARC) to add to the DNS provider for `moleculesai.app`.
**Do not rotate the From address without confirming the new domain is
verified** — there's no server-side check at deploy time.

## Rotation procedure — STRIPE_API_KEY + STRIPE_WEBHOOK_SECRET

These are independent Stripe secrets. Rotating one does **not** affect
the other — they can be rotated on separate schedules.

1. Stripe dashboard → Developers → API keys → **Roll key** on the live
   secret key. Stripe gives you a new `sk_live_…`.
2. Stage on Fly:
   ```
   flyctl secrets set --app molecule-cp \
     --stage STRIPE_API_KEY='sk_live_...'
   ```
3. Redeploy, then verify: hit
   `https://molecule-cp.fly.dev/cp/billing/checkout` from an authenticated
   test session and confirm the returned checkout URL redirects to a
   valid Stripe-hosted page.
4. Stripe auto-revokes the old key after rolling — no manual revoke
   step.

For `STRIPE_WEBHOOK_SECRET`:

1. Stripe dashboard → Developers → Webhooks → the molecule-cp endpoint →
   **Roll secret**.
2. Stripe shows you BOTH old and new secret for a 24-hour overlap window.
   Copy the new `whsec_…`.
3. Stage + deploy on Fly as above.
4. Inside the overlap window, send a Stripe CLI test event:
   ```
   stripe trigger customer.subscription.updated \
     --forward-to https://molecule-cp.fly.dev/webhooks/stripe
   ```
   If the signature-verification layer accepts it (no `400 invalid
   signature` in Fly logs), the new secret is live.
5. Wait for the overlap window to expire or click "Delete old secret"
   in Stripe dashboard.

## Emergency contacts

- **Fly**: billing dashboard at fly.io → Support
- **Neon**: console.neon.tech → Support
- **Upstash**: upstash.com → Support
- **Resend**: resend.com/dashboard → Help (email-only support, ~24h turnaround)
- **Stripe**: stripe.com/support → live chat
- **GHCR**: github.com/orgs/Molecule-AI (org admins)
