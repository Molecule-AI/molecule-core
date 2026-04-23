# Partner API Keys — CI/CD Lifecycle Example
**Phase 34 DevRel Deliverable** | Status: DRAFT — awaiting PM calibration values

This is the DevRel reference implementation for the Partner API Keys CI/CD integration pattern. It shows how a partner would integrate Molecule AI into their automated pipelines using org-scoped partner tokens.

> **Calibration pending from PM:** partner tier rate limits, GA date, first design partner name, per-key org/workspace creation limits, key rotation policy. The pattern below is complete; the placeholder values marked ⚡TBD⚡ will be calibrated once PM answers arrive.

---

## The Partner CI/CD Problem

Enterprise partners integrating with Molecule AI need to:
1. Authenticate CI/CD pipelines without human involvement
2. Rotate tokens without downtime
3. Scope tokens to specific orgs and workspaces
4. Stay within partner-tier rate limits

The current model (shared secret) doesn't support any of these. Partner API keys do.

---

## Workflow: Partner Onboarding a CI/CD Pipeline

### Step 1 — Create a Partner Org

```bash
# Partner's DevOps creates their org in Molecule AI
curl -X POST https://platform.moleculesai.app/orgs \
  -H "Authorization: Bearer $MKTPLATFORM_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acme-partner",
    "tier": "partner_standard",
    "contact_email": "devops@acme.example.com"
  }'
```

Response:
```json
{
  "id": "org_acme_partner_001",
  "name": "acme-partner",
  "tier": "partner_standard",
  "rate_limit": 1000,
  "workspace_limit": 50,
  "api_key_limit": 25,
  "created_at": "2026-04-25T00:00:00Z"
}
```

⚡TBD⚡ **Calibration needed:** `partner_standard` tier's rate limits, workspace limit, and API key limit.

---

### Step 2 — Create a CI/CD Service Token

```bash
# Acme's DevOps creates a token scoped to their org
curl -X POST https://platform.moleculesai.app/orgs/org_acme_partner_001/tokens \
  -H "Authorization: Bearer $MKTPLATFORM_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ci-pipeline-acme",
    "scope": "org_write",
    "ttl_seconds": 2592000,
    "description": "Acme Corp CI/CD pipeline — auto-rotated"
  }'
```

Response (shown once, never retrievable again):
```json
{
  "id": "tok_acme_cicd_001",
  "name": "ci-pipeline-acme",
  "prefix": "mk_live_acme_cicd",
  "token": "ak_live_4j8k2m...Xx9p1",
  "scope": "org_write",
  "ttl_seconds": 2592000,
  "expires_at": "2026-05-25T00:00:00Z",
  "created_by": "admin@acme.example.com"
}
```

> **Security note:** Store the plaintext token in your CI/CD secrets manager (GitHub Secrets, Vault, AWS Secrets Manager). It is shown once and never retrievable.

---

### Step 3 — Use the Token in CI/CD

```yaml
# .github/workflows/molecule-agent.yml
name: Molecule AI Agent Pipeline

on:
  push:
    branches: [main]

env:
  MOLECULE_ORG_ID: org_acme_partner_001
  MOLECULE_API_KEY: ${{ secrets.MOLECULE_CI_PIPELINE_TOKEN }}

jobs:
  agent-run:
    runs-on: ubuntu-latest
    steps:
      - name: Provision workspace
        run: |
          WS=$(curl -s -X POST https://platform.moleculesai.app/workspaces \
            -H "Authorization: Bearer $MOLECULE_API_KEY" \
            -H "Content-Type: application/json" \
            -d '{"name": "ci-workspace-${{ github.run_id }}", "runtime": "hermes"}' \
            | jq -r '.id')
          echo "Workspace: $WS"

      - name: Run agent task
        env:
          WORKSPACE_ID: $WS
        run: |
          curl -s -X POST "https://platform.moleculesai.app/workspaces/$WORKSPACE_ID/tasks" \
            -H "Authorization: Bearer $MOLECULE_API_KEY" \
            -H "Content-Type: application/json" \
            -d '{"task": "analyze", "input": "${{ github.sha }}"}'
```

Every API call is tagged with `mk_live_acme_cicd` in the audit log — Acme's security team can trace every pipeline run.

---

### Step 4 — Token Rotation (automated)

⚡TBD⚡ **Calibration needed:** Rotation policy (forced TTL? manual? grace period?). The pattern below assumes 30-day TTL with 7-day grace period.

```yaml
# .github/workflows/rotate-molecule-token.yml
name: Rotate Molecule AI CI Token

on:
  schedule:
    # Run weekly — well within the 7-day grace period before expiry
    - cron: '0 9 * * 1'
  workflow_dispatch:

jobs:
  rotate:
    runs-on: ubuntu-latest
    steps:
      - name: Create new token
        id: new_token
        run: |
          RESPONSE=$(curl -s -X POST https://platform.moleculesai.app/orgs/org_acme_partner_001/tokens \
            -H "Authorization: Bearer $MKTPLATFORM_ADMIN_TOKEN" \
            -H "Content-Type: application/json" \
            -d '{
              "name": "ci-pipeline-acme",
              "scope": "org_write",
              "ttl_seconds": 2592000
            }')
          echo "token=$(echo $RESPONSE | jq -r '.token')" >> $GITHUB_ENV

      - name: Revoke old token
        run: |
          # List all tokens, revoke any with the same name
          TOKENS=$(curl -s https://platform.moleculesai.app/orgs/org_acme_partner_001/tokens \
            -H "Authorization: Bearer $MKTPLATFORM_ADMIN_TOKEN")
          OLD_TOKEN_ID=$(echo $TOKENS | jq -r '.tokens[] | select(.name == "ci-pipeline-acme" and .id != env.NEW_TOKEN_ID) | .id')
          if [ -n "$OLD_TOKEN_ID" ]; then
            curl -s -X DELETE "https://platform.moleculesai.app/orgs/org_acme_partner_001/tokens/$OLD_TOKEN_ID" \
              -H "Authorization: Bearer $MKTPLATFORM_ADMIN_TOKEN"
            echo "Revoked old token: $OLD_TOKEN_ID"
          fi

      - name: Update GitHub Secret
        uses: GitHub/actions/create-or-update-secret@v4
        with:
          secret-name: MOLECULE_CI_PIPELINE_TOKEN
          value: ${{ env.token }}
```

Zero downtime. The old token is revoked only after the new one is created and tested. The CI pipeline picks up the new secret on its next run.

---

## Tier-Specific Rate Limits (placeholder)

⚡TBD⚡ **Calibration needed from PM.** These are illustrative placeholders:

| Tier | Requests/min | Workspaces | API Keys | Token TTL |
|------|-------------|------------|----------|-----------|
| `partner_starter` | 60 | 5 | 3 | 7 days |
| `partner_standard` | 500 | 25 | 15 | 30 days |
| `partner_enterprise` | 5000 | 200 | 100 | 90 days |

When a partner hits their rate limit, the platform returns `429 Too Many Requests` with a `Retry-After` header. CI/CD pipelines should handle this gracefully:

```bash
# Example: rate-limit-aware API call
until curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $MOLECULE_API_KEY" \
    https://platform.moleculesai.app/workspaces/$WS/tasks \
    -X POST -d '{"task": "analyze"}' | grep -q "200"; do
  echo "Rate limited — backing off 30s..."
  sleep 30
done
```

---

## Visual Asset: Partner CI/CD Flow

```
Partner DevOps
     │
     ▼
POST /orgs  →  creates partner org
     │
     ▼
POST /orgs/:id/tokens  →  creates CI token (stored in Vault/GitHub Secrets)
     │
     ▼
CI Pipeline (GitHub Actions / Jenkins / CircleCI)
     │
     ├──► POST /workspaces  →  provision ephemeral workspace
     ├──► POST /workspaces/:id/tasks  →  run agent task
     └──► Audit log shows mk_live_acme_cicd prefix on every call
     │
     ▼
Token rotation workflow (weekly) — old token revoked, new token created
```

---

## DevRel Checklist for Phase 34 Launch

- [ ] PM calibration values received (tier limits, rotation policy, GA date, partner name)
- [ ] CI/CD example doc updated with real values
- [ ] Partner onboarding tutorial written (or linked from docs)
- [ ] Battlecard updated with CI/CD use case
- [ ] Social copy drafted for partner integration angle

**Source:** `docs/marketing/devrel/partner-api-keys-cicd-example.md`
**Staging path:** ready to commit — awaiting `ghp_` PAT to push