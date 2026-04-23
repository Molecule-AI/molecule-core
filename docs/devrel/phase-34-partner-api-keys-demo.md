# Phase 34 Partner API Keys — DevRel Demo Script
**Phase:** 34 | **Feature:** `mol_pk_*`
**Status:** DRAFT — GA date confirmed (Apr 30). Remaining placeholders: `[DESIGN PARTNER NAME]` (PM), rate limits (PMM lookup in progress)
**Owner:** DevRel Engineer

---

## Demo Prerequisites

1. Molecule AI deployment with admin access
2. `curl` or HTTP client
3. Org admin token (`ADMIN_TOKEN` or org-scoped key)
4. [DESIGN PARTNER NAME]'s account provisioned (or mock org for demo)

---

## Demo Scenario 1 — Partner Key Creation

**What it shows:** How a platform operator provisions a partner-scoped key programmatically.

```
# Create a partner API key
curl -X POST https://your-deployment.moleculesai.app/cp/admin/partner-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "[DESIGN PARTNER NAME] Integration",
    "scopes": ["org:create", "workspace:read"],
    "description": "[DESIGN PARTNER NAME] CI pipeline integration"
  }'

# Response
{
  "id": "pk_live_abc123xyz",
  "name": "[DESIGN PARTNER NAME] Integration",
  "key": "mol_pk_live_abc123xyz789...",
  "scopes": ["org:create", "workspace:read"],
  "created_at": "[TIMESTAMP]",
  "partner_org_id": "[PARTNER ORG ID]"
}
```

**Demo narration:** "This is what a partner-scoped key looks like. Notice the prefix — `mol_pk_`. This is distinct from org-scoped keys and workspace tokens. The partner key can only create orgs within its own scope. It cannot escape to other orgs."

**⚠️ PM NEEDED:** Rate limits per key — add after PM confirms.

---

## Demo Scenario 2 — CI/CD Ephemeral Org (Ephemeral Key Lifecycle)

**What it shows:** How a CI/CD pipeline uses a partner key to spin up a test org per PR, then tears it down.

```
# Step 1: CI job starts — create ephemeral test org
ORG_ID=$(curl -X POST https://your-deployment.moleculesai.app/cp/admin/partner-keys/mol_pk_live_xxx/orgs \
  -H "Authorization: Bearer $PARTNER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "test-pr-$(PR_NUMBER)", "tier": "ephemeral"}' \
  | jq -r '.org_id')

echo "Created test org: $ORG_ID"

# Step 2: Run agent tests against test org
curl -X POST https://your-deployment.moleculesai.app/orgs/$ORG_ID/workspaces \
  -H "Authorization: Bearer $PARTNER_KEY" \
  -d '{"name": "pr-$(PR_NUMBER)-test"}'

# Step 3: CI job ends — teardown
curl -X DELETE https://your-deployment.moleculesai.app/cp/admin/partner-keys/mol_pk_live_xxx/orgs/$ORG_ID \
  -H "Authorization: Bearer $PARTNER_KEY"

echo "Ephemeral org $ORG_ID destroyed — billing stopped"
```

**Demo narration:** "This is the CI/CD use case. Each PR gets its own isolated org — no shared state, no test pollution. When the pipeline finishes, one DELETE call stops the billing immediately. This is what programmatic partner access enables."

**⚠️ PM NEEDED:** Org creation rate limits — add as comments.

---

## Demo Scenario 3 — Key Revocation (Security Demo)

**What it shows:** What happens when a partner key is compromised.

```
# Revoke immediately
curl -X DELETE https://your-deployment.moleculesai.app/cp/admin/partner-keys/mol_pk_live_abc123xyz \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Verify revocation — next request returns 401
curl -X GET https://your-deployment.moleculesai.app/cp/admin/partner-keys \
  -H "Authorization: Bearer mol_pk_live_abc123xyz..."
# → 401 Unauthorized
```

**Demo narration:** "Compromised partner key? One DELETE. The key is dead on the next request. No propagation delay. No cached credentials surviving. That's the security model."

**✅ CONFIRMED:** Revocation is immediate — key is checked synchronously on every request, no async propagation. Access stops on the next inbound request (source: Partner API Keys positioning brief + engineering confirmation).

---

## Demo Scenario 4 — Partner Onboarding Walkthrough

**What it shows:** The full partner onboarding flow from key creation to first API call.

```
# Platform operator creates partner key for new partner
PARTNER_KEY=$(curl -X POST https://your-deployment.moleculesai.app/cp/admin/partner-keys \
  ... | jq -r '.key')

# Partner receives key (out of band — email, dashboard, etc.)
# Partner's CI system uses key to provision their test org

PARTNER_ORG=$(curl -X POST https://your-deployment.moleculesai.app/cp/admin/partner-keys/mol_pk_xxx/orgs \
  -H "Authorization: Bearer $PARTNER_KEY" \
  -d '{"name": "[PARTNER]-production"}')

echo "Partner provisioned: $PARTNER_ORG"

# Partner can now provision workspaces within their own org scope
# They cannot touch other orgs — key is scoped
```

**Demo narration:** "Partner onboarding takes minutes. One key creation, one org provision, partner is live. The key's scope is enforced at every route — partner keys cannot escape their org boundary."

---

## README Structure (for repo walkthrough)

```
## Partner API Keys — Demo

### Prerequisites
...

### Quick Start
1. Create a partner key → Scenario 1
2. Use in CI/CD → Scenario 2
3. Revoke on teardown → Scenario 3

### API Reference
- POST /cp/admin/partner-keys    — create key
- GET  /cp/admin/partner-keys    — list keys
- GET  /cp/admin/partner-keys/:id — get key details
- DELETE /cp/admin/partner-keys/:id — revoke key

### ⚠️ Placeholders to fill after PM confirmation
- [DESIGN PARTNER NAME] — first design partner name
- 2026-04-30 — GA ship date
- Rate limits — pending PM confirmation
- Key rotation policy — pending PM confirmation
```

---

## Storyboard for Screencast

| Scene | Duration | What happens |
|---|---|---|
| 1. Title card | 5s | "Partner API Keys — Phase 34" + date `2026-04-30` |
| 2. Problem framing | 10s | "Your CI pipeline needs test orgs. How do you provision them programmatically?" |
| 3. Key creation | 15s | Terminal: `POST /cp/admin/partner-keys` → key returned |
| 4. Ephemeral org | 20s | CI script: create org → run tests → DELETE org |
| 5. Revocation | 10s | Compromised key → DELETE → 401 |
| 6. CTA | 5s | "Docs → [URL]" + `[DESIGN PARTNER NAME]` logo |

**Total: ~65 seconds** — within 1-min screencast spec.

**Brand audio:** TTS voiceover with light background music. Intro jingle (Phase 30 jingle reuse). Outro: same as Phase 30 CTA music sting.

---

## Open Questions (PM must answer before final)

| # | Question | Impact if unanswered |
|---|---|---|
| 1 | Rate limits per partner key? | Cannot show safe CI/CD limits in demo |
| 2 | Key rotation policy (TTL/forced/manual)? | Cannot document rotation in README |
| 3 | Design partner name? | Cannot name first partner in demo narration |
| 4 | GA date? | ✅ RESOLVED — April 30, 2026 |
| 5 | Partner tier differences? | Cannot differentiate tiers in demo |

*Skeleton by Marketing Lead 2026-04-23 — fill placeholders when PM responds.*
