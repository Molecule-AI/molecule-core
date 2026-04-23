---
title: "Partner API Keys Are Generally Available"
slug: partner-api-keys-ga
date: 2026-04-30
authors: [molecule-ai]
tags: [platform, partner-api, provisioning, phase-34, ci-cd]
description: "mol_pk_* keys are GA. Programmatically provision and manage Molecule AI orgs via API — no browser session, no manual setup, no shared credentials."
og_image: /assets/blog/2026-04-30-partner-api-keys/og.png
---

# Partner API Keys Are Generally Available

Starting today, any platform, CI/CD pipeline, or marketplace can programmatically create and manage Molecule AI organizations via API.

No browser session. No manual setup handoff. No shared `ADMIN_TOKEN` passed around your team. The entire org lifecycle — provision, configure, teardown — is a set of API calls.

---

## The Core API

Partner API Keys (`mol_pk_*`) are a new key type scoped to the org lifecycle. Three endpoints cover the full lifecycle:

```bash
# Provision a new org
curl -X POST https://api.molecule.ai/cp/admin/partner-keys \
  -H "Authorization: Bearer $MOL_PK_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "acme-prod", "description": "Acme Corp production org"}'
```

```json
{
  "id": "pk_8ch4r5xx",
  "key": "mol_pk_live_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
  "org_id": "org_acme_prod",
  "created_at": "2026-04-30T09:00:00Z"
}
```

The key is returned in plaintext once on creation — store it immediately. Molecule AI stores only the SHA-256 hash.

```bash
# List all partner keys
curl https://api.molecule.ai/cp/admin/partner-keys \
  -H "Authorization: Bearer $MOL_PK_TOKEN"

# Revoke a key — org access stops on next request
curl -X DELETE https://api.molecule.ai/cp/admin/partner-keys/pk_8ch4r5xx \
  -H "Authorization: Bearer $MOL_PK_TOKEN"
```

---

## Three Things You Can Build

### 1. Platform integrations: embed agent orchestration as a feature

If you're building a platform that wants to offer AI agent capabilities to your users, Partner API Keys give you a programmatic provisioning path. Your backend calls the Partner API to create a Molecule AI org for each customer — no browser session required on either side.

Each `mol_pk_*` key is **org-scoped**: a key created for one org cannot reach resources in any other org. Compromising one key doesn't expose your other tenants. Revoking access is one `DELETE` call.

### 2. Marketplace resellers: automate the deploy-to-customer flow

For cloud marketplace listings, Partner API Keys enable fully automated org provisioning. When a customer clicks "Deploy", your marketplace integration calls the Partner API, provisions the org, and hands the customer a working dashboard — no manual coordination required.

### 3. CI/CD teams: ephemeral test orgs per PR

This is the use case we hear most from enterprise platform teams.

```yaml
# .github/workflows/integration-test.yml
- name: Provision test org
  run: |
    RESPONSE=$(curl -s -X POST https://api.molecule.ai/cp/admin/partner-keys \
      -H "Authorization: Bearer $MOL_PK_TOKEN" \
      -d '{"name": "ci-pr-${{ github.event.number }}"}')
    echo "ORG_KEY=$(echo $RESPONSE | jq -r .key)" >> $GITHUB_ENV

- name: Run integration tests
  run: pytest tests/integration/
  env:
    MOLECULE_API_KEY: ${{ env.ORG_KEY }}

- name: Teardown test org
  if: always()
  run: |
    KEY_ID=$(curl -s https://api.molecule.ai/cp/admin/partner-keys \
      -H "Authorization: Bearer $MOL_PK_TOKEN" | jq -r '.[0].id')
    curl -X DELETE https://api.molecule.ai/cp/admin/partner-keys/$KEY_ID \
      -H "Authorization: Bearer $MOL_PK_TOKEN"
```

Each PR gets a clean, isolated Molecule AI org. Tests run without shared-state contamination. Teardown happens automatically — billing stops on `DELETE`.

---

## Security model

`mol_pk_*` keys are designed as infrastructure credentials, not user credentials:

- **Org-scoped.** Each key is bound to exactly one org. There is no key that has org-wide blast radius.
- **SHA-256 hashed in storage.** Molecule AI stores only the hash. If the database is compromised, attacker does not get live keys.
- **Revocable immediately.** `DELETE /cp/admin/partner-keys/:id` — access stops on the next inbound request. No session expiry delay.
- **Rate-limited per key.** Per-key rate limiting is separate from session-based limits. High-traffic integrations get predictable headroom.
- **Tracked.** `last_used_at` is updated on every authenticated request. You can see whether a key is in active use before you rotate or revoke it.
- **Pre-commit scanning.** `mol_pk_` is in the secret scanner pattern list. Accidental commits get blocked before they reach the repo.

---

## What else ships today (Phase 34)

Partner API Keys GA is part of Phase 34, which also ships:

- **[Tool Trace](https://docs.molecule.ai/blog/agent-observability-tool-trace-platform-instructions)** — every A2A response includes a `tool_trace` field with a complete record of every tool your agent called. No SDK, no sidecar.
- **[Platform Instructions](https://docs.molecule.ai/blog/agent-observability-tool-trace-platform-instructions)** — set org-wide behavioral rules via API, enforced before every agent turn.
- **SaaS Federation v2** — improved multi-tenant control plane architecture for enterprise and marketplace deployments.

---

## Getting started

Partner API Keys are live at GA today.

```bash
# Read the docs
open https://docs.molecule.ai/api/partner-keys

# Join the partner channel
# #partner-program on the Molecule AI Discord
```

→ [Partner API Keys documentation](https://docs.molecule.ai/api/partner-keys)  
→ [Partner onboarding guide](https://docs.molecule.ai/docs/guides/partner-onboarding)  
→ [Phase 34 release notes](https://docs.molecule.ai/changelog/phase-34)

---

*Phase 34 also includes Tool Trace, Platform Instructions, and SaaS Federation v2. See the [full Phase 34 announcement](https://docs.molecule.ai/blog/phase-34-community-announcement).*
