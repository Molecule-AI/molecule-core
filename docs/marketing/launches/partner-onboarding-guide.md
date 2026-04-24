# Partner Onboarding Guide — First Pass
**Date:** 2026-04-23 | **Owner:** PMM | **Status:** DRAFT — tier names confirmed (blog post live), PM confirm Go implementation
**Scope:** Partner and Enterprise plans | Rate limit: 60 req/min per key (default, configurable — confirmed per architecture doc)

---

## Overview

Partner API Keys (`mol_pk_*`) let you provision and manage Molecule AI orgs programmatically — for CI/CD pipelines, marketplace integrations, or internal automation. This guide covers the end-to-end lifecycle: key creation, org provisioning, configuration, and teardown.

**What you need before starting:**
- A Molecule AI admin account with access to `/cp/admin/partner-keys`
- `curl` or an HTTP client
- Understanding of which scopes your integration needs

---

## 1. Prerequisites

Before calling the Partner API, you need:

| Requirement | How to get it |
|---|---|
| Admin token or org-scoped key | Created in Canvas → Org Settings → API Keys |
| Partner tier access | Partner and Enterprise plans — contact your account team or apply via moleculesai.app/partners |
| Scope assignment | Decide at key creation time which scopes to grant |
| Compliance review (optional) | Enterprise tier may require a security questionnaire |

---

## 2. Creating Your First Partner Key

Keys are created by an org admin. The full key is shown once — store it securely (secret manager, not a spreadsheet).

```bash
# Create a partner API key
curl -X POST https://api.moleculesai.app/cp/admin/partner-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acme-ci-pipeline",
    "scopes": ["orgs:create", "orgs:list", "orgs:delete"],
    "rate_limit": 60
  }'

# Response — key shown ONCE
{
  "id": "pak_01HXKM4...",
  "key": "mol_pk_live_1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7",
  "name": "acme-ci-pipeline",
  "scopes": ["orgs:create", "orgs:list", "orgs:delete"],
  "rate_limit": 60,
  "created_at": "2026-04-23T08:00:00Z"
}
```

Save the `key` value immediately — it is not retrievable after this response.

**Scope reference:**
- `orgs:create` — provision new orgs
- `orgs:list` — list your partner-managed orgs
- `orgs:delete` — deprovision orgs
- `workspaces:create` — create workspaces within an org
- `billing:read` — read subscription status

---

## 3. Org Lifecycle

### Create an org
```bash
ORG_RESPONSE=$(curl -X POST https://api.moleculesai.app/cp/orgs \
  -H "Authorization: Bearer mol_pk_live_1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7" \
  -H "Content-Type: application/json" \
  -d '{"name": "customer-acme", "slug": "customer-acme", "plan": "standard"}')

ORG_ID=$(echo $ORG_RESPONSE | jq -r '.id')
echo "Org ID: $ORG_ID"
```

### Poll provisioning status
```bash
# Poll until status is "active"
STATUS=$(curl -s https://api.moleculesai.app/cp/orgs/$ORG_ID/status \
  -H "Authorization: Bearer mol_pk_live_1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7" \
  | jq -r '.status')

echo "Status: $STATUS"
# → provisioning → active
```

### Redirect the tenant
```
https://app.moleculesai.app/login?org=customer-acme
```

### Teardown
```bash
# Delete the org (irreversible)
curl -X DELETE https://api.moleculesai.app/cp/orgs/$ORG_ID \
  -H "Authorization: Bearer mol_pk_live_1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7"
```

---

## 4. CI/CD Example — GitHub Actions

This pattern spins up an isolated test org per PR, runs your integration tests, and tears it down. Each run gets a clean org — no shared state, no test pollution.

```yaml
name: Molecule AI Integration Tests

on:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Create ephemeral test org
        id: create-org
        run: |
          RESPONSE=$(curl -s -X POST ${{ vars.MOLECULE_API_URL }}/cp/orgs \
            -H "Authorization: Bearer ${{ secrets.MOL_PARTNER_KEY }}" \
            -H "Content-Type: application/json" \
            -d "{\"name\":\"pr-${{ github.event.pull_request.number }}\",\"slug\":\"pr-${{ github.event.pull_request.number }}\"}")
          ORG_ID=$(echo "$RESPONSE" | jq -r '.id')
          echo "ORG_ID=$ORG_ID" >> $GITHUB_OUTPUT
          echo "Created org: $ORG_ID"

      - name: Poll until ready
        run: |
          for i in $(seq 1 30); do
            STATUS=$(curl -s ${{ vars.MOLECULE_API_URL }}/cp/orgs/${{ steps.create-org.outputs.ORG_ID }}/status \
              -H "Authorization: Bearer ${{ secrets.MOL_PARTNER_KEY }}" \
              | jq -r '.status')
            [ "$STATUS" = "active" ] && break
            sleep 2
          done

      - name: Run integration tests
        run: |
          # Use ${{ steps.create-org.outputs.ORG_ID }} in your test config
          npm test -- --org-id=${{ steps.create-org.outputs.ORG_ID }}

      - name: Teardown test org
        if: always()
        run: |
          curl -s -X DELETE ${{ vars.MOLECULE_API_URL }}/cp/orgs/${{ steps.create-org.outputs.ORG_ID }} \
            -H "Authorization: Bearer ${{ secrets.MOL_PARTNER_KEY }}"
```

**Security note:** Store `MOL_PARTNER_KEY` as a GitHub Actions secret — never hardcode it. Use a key scoped to `orgs:create` + `orgs:list` + `orgs:delete` only.

---

## 5. Security Best Practices

1. **One key per integration tier.** Create separate keys for CI/CD, marketplace, and internal automation. If one key is compromised, revoke it without affecting other integrations.

2. **Create-then-revoke for rotation.** Keys cannot be updated — only created and deleted. To rotate: create new key → update integration → revoke old key. This produces a clean audit trail.

3. **Monitor `last_used_at`.** Every call with a Partner API Key is logged with timestamp and caller identity. Check the audit log periodically for unexpected usage.

4. **Scope to minimum required.** A CI pipeline needs `orgs:create` + `orgs:list` + `orgs:delete` — nothing more. Don't grant `billing:read` to a test integration.

5. **Set expiration on non-production keys.** Keys used in test/CI environments can be issued with an `expires_at` timestamp. Expired keys return 401 with a clear message.

---

## 6. Support

- **Partner Discord:** `#partner-program` channel in the Molecule AI Community server — for integration questions, escalation, and partner announcements.
- **Email:** partner@molecule.ai — for enterprise partner inquiries.
- **Docs:** docs.molecule.ai/docs — API reference and architecture docs.

---

*PMM draft 2026-04-23 — first pass. Tier names confirmed via blog post (2026-04-23). Still needs PM confirmation on Go implementation and billing endpoint availability before external distribution.*
