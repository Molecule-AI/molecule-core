# Partner API Keys — Demo
**Phase:** 34 | **Feature:** `mol_pk_*` | **Handler:** `workspace-server/internal/handlers/partner_keys.go`

---

## What This Demo Shows

1. Create a partner-scoped API key with minimal scopes
2. Use the partner key to create an ephemeral org (CI/CD use case)
3. Poll org status, create a workspace
4. Teardown: DELETE the org (billing stops immediately)
5. Revoke a partner key — next request returns 401

**Requirements:** `pip install requests`

---

## Quick Start

```bash
export PLATFORM_URL=https://your-deployment.moleculesai.app
export ADMIN_TOKEN=your-admin-token

python demo.py
```

### Offline mode (no platform needed)

```bash
python demo.py
# Uses simulated responses — no credentials required
```

---

## Step-by-Step Walkthrough

### Step 1 — Create a partner key

```bash
curl -s -X POST "$PLATFORM_URL/cp/admin/partner-keys" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-pipeline-key", "scopes": ["orgs:create", "orgs:list", "workspaces:create"]}' | jq .
```

```json
{
  "id": "pak_01HXKM4ABC",
  "key": "mol_pk_live_abc123xyzci-pipel789...",
  "name": "ci-pipeline-key",
  "scopes": ["orgs:create", "orgs:list", "workspaces:create"],
  "created_at": "2026-04-23T08:00:00Z"
}
```

> Copy the key now — it's shown exactly once.

### Step 2 — Create an ephemeral org

```bash
curl -s -X POST "$PLATFORM_URL/cp/orgs" \
  -H "Authorization: Bearer $PARTNER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "test-pr-123", "plan": "ephemeral"}' | jq .
```

```json
{"id": "org_01HXKM4ABC", "name": "test-pr-123", "status": "provisioning"}
```

### Step 3 — Poll until active, create workspace

```bash
# Poll
curl -s "$PLATFORM_URL/cp/orgs/org_01HXKM4ABC" \
  -H "Authorization: Bearer $PARTNER_KEY" | jq '.status'
# → "active"

# Create workspace in the org
curl -s -X POST "$PLATFORM_URL/cp/orgs/org_01HXKM4ABC/workspaces" \
  -H "Authorization: Bearer $PARTNER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "pr-123-test"}' | jq .
```

### Step 4 — Teardown

```bash
curl -s -X DELETE "$PLATFORM_URL/cp/orgs/org_01HXKM4ABC" \
  -H "Authorization: Bearer $PARTNER_KEY"
# → 204 No Content — billing stops immediately
```

### Step 5 — Revoke a key

```bash
curl -s -X DELETE "$PLATFORM_URL/cp/admin/partner-keys/pak_01HXKM4ABC" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# → 204 No Content — next mol_pk_* request returns 401
```

---

## API Reference

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/cp/admin/partner-keys` | Admin | Create a partner key |
| GET | `/cp/admin/partner-keys` | Admin | List all partner keys |
| DELETE | `/cp/admin/partner-keys/:id` | Admin | Revoke a partner key |
| POST | `/cp/orgs` | Partner key | Create an org |
| GET | `/cp/orgs/:id` | Partner key | Get org status |
| DELETE | `/cp/orgs/:id` | Partner key | Delete an org |
| POST | `/cp/orgs/:id/workspaces` | Partner key | Create workspace in org |

---

## Key Design Decisions

### `mol_pk_*` vs workspace/org tokens

| Token type | Scope |
|-----------|-------|
| Workspace token | One workspace |
| Org token | All workspaces in one org |
| **Partner key (`mol_pk_*`)** | **Org-level operations only** |

Partner keys create orgs and manage their lifecycle. They cannot access workspace sub-routes (secrets, agents, files, etc.).

### Plaintext shown once

The plaintext key is returned only at creation. The server stores SHA-256 hash only. If lost, revoke and recreate.

### Ephemeral org lifecycle

1. `POST /cp/orgs` — org starts in `provisioning` state
2. Poll `GET /cp/orgs/:id` until `status: "active"`
3. Create workspaces, run tests
4. `DELETE /cp/orgs/:id` — immediately stops billing

### Security model

- Keys are org-scoped by design — cannot escape their boundary
- Per-key rate limiter (separate from session limits)
- `last_used_at` tracked on every request
- `mol_pk_` added to pre-commit secret scanner

---

## Files

- `demo.py` — Runnable Python demo (simulated + live modes)
- Handler: `workspace-server/internal/handlers/partner_keys.go`
- Blog: `docs/blog/2026-04-23-partner-api-keys/`
- Battlecard: `docs/marketing/battlecard/phase-34-partner-api-keys-battlecard.md`
- TTS narration: `docs/devrel/phase-34-partner-api-keys-screencast-narration.mp3`
