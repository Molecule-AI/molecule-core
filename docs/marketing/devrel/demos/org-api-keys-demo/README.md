# Org-Scoped API Keys — Demo Script
**Issue:** org-api-keys-launch campaign | **Source:** PR #1105 | **Acceptance:** Working demo + repo link

---

## What This Demo Shows

1. List existing org tokens (`GET /org/tokens`)
2. Mint a new scoped token (`POST /org/tokens`)
3. Use the org token for a workspace-scoped API call
4. Verify cross-org access is blocked
5. Revoke a token (`DELETE /org/tokens/:id`)

**Time:** ~3 min | **Tools:** Python, curl | **Setup:** `PLATFORM_URL`, `ORG_TOKEN`

---

## Quick Start

```bash
export PLATFORM_URL=https://your-deployment.moleculesai.app
export ORG_TOKEN=your-org-scoped-token   # must be org-admin level

python demo.py
```

### Offline mode (no platform needed)

```bash
python demo.py
# Uses simulated responses — no credentials required
```

---

## Step-by-Step Walkthrough

### Step 1 — List org tokens

```bash
curl -s -X GET https://your-deployment.moleculesai.app/org/tokens \
  -H "Authorization: Bearer $ORG_TOKEN" | jq
```

```json
{
  "tokens": [
    {"id": "tok_abc001", "name": "slack-integration", "scope": "read"},
    {"id": "tok_abc002", "name": "ci-pipeline-key",    "scope": "write"}
  ],
  "total": 2
}
```

### Step 2 — Mint a new token

```bash
curl -s -X POST https://your-deployment.moleculesai.app/org/tokens \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-pipeline-key", "scope": "write", "expires_in_days": 90}' | jq
```

```json
{
  "id": "tok_xyz789",
  "token": "org_tk_abc123...def456",
  "scope": "write",
  "expires_at": "2026-07-01T00:00:00Z",
  "message": "Save this token — it cannot be retrieved again."
}
```

> **Copy the token now.** It's shown exactly once.

### Step 3 — Use the org token

```bash
# Attach an Artifacts repo to a workspace — scoped to this org
curl -s -X POST https://your-deployment.moleculesai.app/workspaces/ws-demo-001/artifacts \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-deploy-snapshots"}' | jq
```

### Step 4 — Cross-org rejection

```bash
# Attempt to access a workspace in a different org
curl -s -X GET https://your-deployment.moleculesai.app/workspaces/ws-external/executions \
  -H "Authorization: Bearer $ORG_TOKEN" | jq
```

```json
{"error": "unauthorized", "message": "token org does not match target workspace org"}
```

### Step 5 — Revoke

```bash
curl -s -X DELETE https://your-deployment.moleculesai.app/org/tokens/tok_xyz789 \
  -H "Authorization: Bearer $ORG_TOKEN"
# → 204 No Content (token is dead immediately)
```

---

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| GET | `/org/tokens` | List all org tokens |
| POST | `/org/tokens` | Mint a new token |
| DELETE | `/org/tokens/:id` | Revoke a token |

### Token body (POST)

```json
{
  "name": "my-ci-token",
  "scope": "write",        // "read" or "write"
  "expires_in_days": 30    // 1–365, default 30
}
```

### Response fields

| Field | Description |
|-------|-------------|
| `id` | Internal token ID (use for revoke) |
| `token` | Plaintext token — shown **once only** |
| `scope` | `read` or `write` |
| `expires_at` | ISO 8601 expiry timestamp |
| `created_at` | ISO 8601 creation timestamp |
| `last_used` | ISO 8601 last use (null if never used) |

---

## Key Design Decisions

### Plaintext shown once

The plaintext token value is returned **only at mint time**. The server stores only the SHA-256 hash. This means:

- There is no "show token" endpoint — if you lose it, revoke and re-mint
- The token value can't be recovered from the server
- Bruteforce protection via crypto-random token generation

### Org-level scoping

An org token grants access to **any workspace in its organization**. This is different from workspace tokens which are scoped to one workspace. Org tokens are for integrations (CI/CD, Slack bots, monitoring scripts) that need to operate across the org.

### Immediate revocation

Revoking a token immediately invalidates it — no grace period, no cached sessions. The next request with that token returns 401.

### Cross-org enforcement

Every platform handler that validates a token must check `token.org_id == workspace.org_id`. Handlers that don't perform this check are a security bug — the org-api-keys PR should add a middleware-level enforcement.

---

## Files

- Demo script: `docs/marketing/devrel/demos/org-api-keys-demo/demo.py`
- Screencast storyboard: `marketing/devrel/demos/screencasts/storyboard-org-api-keys.md`
- Social copy: `docs/marketing/campaigns/org-api-keys-launch/social-copy.md`
- OG image: `docs/assets/blog/2026-04-25-org-scoped-api-keys-og.png`
