# Org-Scoped API Keys — Working Demo

> **PR:** #1105 — `feat(auth): org-scoped API keys`  
> **What it ships:** `workspace-server/internal/handlers/org_tokens.go` — named, revocable org-admin tokens minted from canvas UI or CLI  
> **Acceptance criteria:** working demo + repo link + 1-min screencast or README walkthrough

---

## What This Demo Shows

An org admin can mint a named, revocable API key from the CLI (or canvas). The key is a full-admin bearer token for the tenant platform — it authorizes every admin-gated endpoint on the tenant (all workspaces, org settings, bundles, secrets). The demo shows minting, using, and revoking the key.

**Key facts:**
- 256-bit entropy, base64url encoded. Prefix shown in UI for identification.
- Plaintext returned exactly once at mint time — never stored.
- `org-token:<prefix>` format in audit logs and API responses.
- Revocation is immediate and idempotent.

**Routes:**
| Method | Path | What |
|---|---|---|
| `GET` | `/org/tokens` | List live (non-revoked) tokens |
| `POST` | `/org/tokens` | Mint a new token — plaintext returned once |
| `DELETE` | `/org/tokens/:id` | Revoke a token |

---

## Prerequisites

- Molecule AI platform running (`go run ./cmd/server` from `workspace-server/`)
- Canvas open at `http://localhost:3000`
- Admin session cookie OR an existing org-scoped token
- `curl` and `jq` on the caller machine

---

## Working Demo Script

### 1. Mint a named org token

```bash
PLATFORM="http://localhost:8080"
COOKIE="session=$(curl -s -c - -X POST "$PLATFORM/cp/auth/login" \
  -d 'email=admin@example.com&password=...' | grep session | awk '{print $7}')"

# Mint a named org token
curl -s -X POST "$PLATFORM/org/tokens" \
  -H "Cookie: $COOKIE" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-pipeline-key"}' | jq
```

Response (200):
```json
{
  "id": "otok_xxxxxxxxxxxx",
  "prefix": "mL9kXp2W",
  "name": "ci-pipeline-key",
  "auth_token": "org-token:mL9kXp2WQrZvT8sBmN3cD4eF6gH0iJ1kL9pM3nO5qR7tU0vW1xY2zA3bC4dE5fG",
  "warning": "copy this token now; it will not be shown again"
}
```

**Save the `auth_token` value — it cannot be retrieved again.**

---

### 2. Use the org token to list workspaces

The token is a full-admin bearer. Pass it in the `Authorization` header:

```bash
ORG_TOKEN="org-token:mL9kXp2WQrZvT8sBmN3cD4eF6gH0iJ1kL9pM3nO5qR7tU0vW1xY2zA3bC4dE5fG"

curl -s "$PLATFORM/org/tokens" \
  -H "Authorization: Bearer $ORG_TOKEN" | jq
```

Response — lists all live org tokens:
```json
{
  "tokens": [
    {
      "id": "otok_xxxxxxxxxxxx",
      "prefix": "mL9kXp2W",
      "name": "ci-pipeline-key",
      "created_by": "admin@example.com",
      "created_at": "2026-04-21T00:00:00Z",
      "revoked_at": null
    }
  ],
  "count": 1
}
```

Use the token to call any admin-gated endpoint:

```bash
# List all workspaces in the org
curl -s "$PLATFORM/workspaces" \
  -H "Authorization: Bearer $ORG_TOKEN" | jq '.workspaces[].id'

# List all org bundles
curl -s "$PLATFORM/bundles" \
  -H "Authorization: Bearer $ORG_TOKEN" | jq '.'

# Read workspace secrets
curl -s "$PLATFORM/workspaces/ws-123/secrets/values" \
  -H "Authorization: Bearer $ORG_TOKEN" | jq
```

---

### 3. Revoke the token and confirm 401

```bash
# Revoke via DELETE
curl -s -X DELETE "$PLATFORM/org/tokens/otok_xxxxxxxxxxxx" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -H "Content-Type: application/json" \
  -w "\nHTTP %{http_code}\n"
```

Response:
```
HTTP 200
```

Revocation is immediate. Confirm the token no longer works:

```bash
curl -s "$PLATFORM/org/tokens" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -w "\nHTTP %{http_code}\n"
```

Response:
```
HTTP 401
{"error":"invalid or revoked org api token"}
```

The token is dead. Revocation is permanent — the row is soft-deleted (revoked_at timestamp set). The same plaintext can never authenticate again.

---

### 4. Verify the token shows as revoked in the list

Revoke requires the session cookie (not the org token — org tokens can't revoke other org tokens):

```bash
curl -s "$PLATFORM/org/tokens" \
  -H "Cookie: $COOKIE" | jq '.tokens[0].revoked_at'
```

Response:
```json
"2026-04-21T00:05:00Z"
```

Audit trail: `created_by` on the original mint + `revoked_at` timestamp provide full provenance tracking.

---

## Screencast Outline (1 min)

**0:00–0:10** Terminal: mint a named org token via `POST /org/tokens`. Response shows `auth_token` field — highlight the "copy now" warning.

**0:10–0:25** Terminal: use the token to call `GET /workspaces`. Output shows workspace IDs. Call out `Authorization: Bearer org-token:...` header.

**0:25–0:40** Terminal: revoke the token via `DELETE /org/tokens/:id`. Output: `HTTP 200`. Immediately try to call `/org/tokens` again — `HTTP 401` shown in terminal.

**0:40–0:55** Canvas: switch to Org Settings → API Keys tab. The revoked token shows `revoked_at` timestamp. New token listed alongside any others.

**0:55–1:00** Narration: *"Org-scoped API keys: mint named tokens, use them anywhere, revoke them instantly. Full admin access, one line of curl. No session cookies, no browser. org-token keys, from Molecule AI."*

---

## TTS Narration Script (30s)

> Org-scoped API keys let you mint a full-admin bearer token for the Molecule AI platform — named, revocable, auditable. Mint one from the CLI or canvas, use it to call any admin endpoint, revoke it when you're done. The token is 256 bits of entropy, shown exactly once at mint time, and the revoke is immediate. No session cookies, no browser. Just a bearer token that works until you revoke it. Org API keys, from Molecule AI.

---

## Code Reference

| File | What |
|---|---|
| `workspace-server/internal/handlers/org_tokens.go` | OrgTokenHandler: List, Create, Revoke |
| `workspace-server/internal/orgtoken/tokens.go` | `Issue()`, `Revoke()`, `List()` — token lifecycle |
| `workspace-server/migrations/035_org_api_tokens.up.sql` | Schema: `org_api_tokens` table |
| `canvas/src/components/settings/OrgTokensTab.tsx` | Canvas UI: org token management tab |

**Source:** `workspace-server/internal/handlers/org_tokens.go` (PR #1105)
