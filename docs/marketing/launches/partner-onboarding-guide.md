# Partner API Keys — Onboarding Guide

**Feature:** `mol_pk_*` partner-scoped API keys  
**GA Date:** April 30, 2026  
**Audience:** Platform integrators, marketplace resellers, CI/CD DevOps teams  
**Status:** DRAFT — awaiting PM confirmation on rate limits and partner tiers

---

## Prerequisites

Before calling the Partner API:

1. **Admin access** to a Molecule AI organization
2. **An admin token** (`ADMIN_TOKEN` or an org-scoped key with admin privileges)
3. **HTTPS client** — `curl`, your language's HTTP library, or any REST client

No additional setup, no browser session, no SDK to install.

---

## Step 1: Create Your First Partner Key

A partner key (`mol_pk_*`) lets your platform programmatically provision and manage Molecule AI orgs on behalf of your users.

```bash
curl -X POST https://api.molecule.ai/cp/admin/partner-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-platform-integration",
    "description": "Production partner key for MyPlatform org provisioning"
  }'
```

**Response:**

```json
{
  "id": "pk_8ch4r5xx",
  "name": "my-platform-integration",
  "key": "mol_pk_live_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
  "created_at": "2026-04-30T09:00:00Z"
}
```

**Store the key immediately** — Molecule AI stores only the SHA-256 hash. The plaintext key is returned once and cannot be retrieved again.

---

## Step 2: Provision an Org

With your partner key, provision a Molecule AI org for a customer or pipeline run:

```bash
curl -X POST https://api.molecule.ai/cp/admin/partner-keys \
  -H "Authorization: Bearer mol_pk_live_XXXXXXXXXXXXXXXX" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "customer-acme-prod",
    "description": "Acme Corp production environment"
  }'
```

The provisioned org is isolated — the partner key is scoped to its own org boundary and cannot touch any other org's resources.

---

## Step 3: Manage the Org Lifecycle

List your partner keys to see what's active:

```bash
curl https://api.molecule.ai/cp/admin/partner-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Revoke a key when you no longer need it (billing stops immediately):

```bash
curl -X DELETE https://api.molecule.ai/cp/admin/partner-keys/pk_8ch4r5xx \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Revocation is **immediate** — the key stops working on the very next request. No propagation delay.

---

## CI/CD Use Case: Ephemeral Test Orgs Per PR

This is the most common enterprise pattern — a fresh isolated org per CI run, with automated teardown:

```yaml
# .github/workflows/integration-test.yml
name: Integration Tests

on: [pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Provision ephemeral test org
        id: provision
        run: |
          RESPONSE=$(curl -sf -X POST https://api.molecule.ai/cp/admin/partner-keys \
            -H "Authorization: Bearer ${{ secrets.MOL_ADMIN_TOKEN }}" \
            -H "Content-Type: application/json" \
            -d "{\"name\": \"ci-pr-${{ github.event.number }}\", \"description\": \"PR #${{ github.event.number }} test org\"}")
          echo "key_id=$(echo $RESPONSE | jq -r .id)" >> $GITHUB_OUTPUT
          echo "::add-mask::$(echo $RESPONSE | jq -r .key)"
          echo "org_key=$(echo $RESPONSE | jq -r .key)" >> $GITHUB_OUTPUT

      - name: Run integration tests
        run: pytest tests/integration/
        env:
          MOLECULE_API_KEY: ${{ steps.provision.outputs.org_key }}

      - name: Teardown test org
        if: always()   # runs even if tests fail
        run: |
          curl -sf -X DELETE \
            https://api.molecule.ai/cp/admin/partner-keys/${{ steps.provision.outputs.key_id }} \
            -H "Authorization: Bearer ${{ secrets.MOL_ADMIN_TOKEN }}"
```

**What this gives you:**
- Each PR gets a completely isolated Molecule AI org
- No shared state between test runs
- Billing stops the moment `DELETE` is called
- `if: always()` ensures teardown happens even on test failure

---

## Security Best Practices

| Practice | Why |
|----------|-----|
| Store `mol_pk_*` keys in secrets management (Vault, AWS Secrets Manager, GitHub Actions secrets) | Never commit to source control — `mol_pk_` is in the pre-commit secret scanner |
| One key per integration | Revoking one key doesn't affect any other integration |
| Rotate by delete + recreate | There is no key rotation endpoint — revoke the old key and create a new one |
| Monitor `last_used_at` before revoking | Ensures you don't revoke a key that's actively in use |
| Use descriptive `name` and `description` | Makes auditing easier — you'll know what each key was for |

**Rate limits:** [RATE LIMIT TBD — PM to confirm from controlplane config]

---

## Key Properties Reference

| Property | Value |
|----------|-------|
| Key prefix | `mol_pk_live_` (production), `mol_pk_test_` (test environments) |
| Storage | SHA-256 hashed in database — plaintext returned once on creation only |
| Scope | Org-scoped — cannot access resources outside the provisioned org |
| Revocation | Immediate — synchronous check on every request |
| Audit | `last_used_at` updated on every request |
| Secret scanning | `mol_pk_` pattern is in the pre-commit scanner |

---

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/cp/admin/partner-keys` | Create a new partner key |
| `GET` | `/cp/admin/partner-keys` | List all partner keys for the org |
| `GET` | `/cp/admin/partner-keys/:id` | Get details for a specific key |
| `DELETE` | `/cp/admin/partner-keys/:id` | Revoke a key (immediate) |

---

## Get Support

**Partner questions:** Join `#partner-program` in the [Molecule AI Discord](https://discord.gg/molecule-ai)  
**Docs:** [https://docs.molecule.ai/api/partner-keys](https://docs.molecule.ai/api/partner-keys)  
**Issues:** [https://github.com/Molecule-AI/molecule-core/issues](https://github.com/Molecule-AI/molecule-core/issues)

---

*Draft — Marketing Lead 2026-04-23. Awaiting PM confirmation on rate limits (`[RATE LIMIT TBD]`) and partner tiers before final publish.*
