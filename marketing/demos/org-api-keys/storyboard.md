# Screencast Storyboard — Org-Scoped API Keys

> **PR:** #1105 | **Feature:** `org_tokens.go` | **Duration:** 60 seconds
> **Format:** Terminal-led, clean dark theme

---

## Pre-roll (0:00–0:04)

**Canvas — full screen**
Org settings panel open. Org Settings → API Keys tab visible with one existing token listed.

Narration (0:00–0:04):
> "Every org has admin operations — listing workspaces, managing secrets, deploying bundles. Org-scoped API keys let you do all of it, from the CLI, without a browser."

**Camera:** Static Canvas frame. 3-second hold.

---

## Moment 1 — Mint the token (0:04–0:16)

**Cut to:** Terminal window, dark theme.

Prompt: `admin@platform:~$`

```bash
PLATFORM="https://acme.moleculesai.app"

curl -s -X POST "$PLATFORM/org/tokens" \
  -H "Cookie: session=admin@example.com..." \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-pipeline-key"}' | jq
```

**Terminal output:**

```json
{
  "id": "otok_a1b2c3d4e5f6",
  "prefix": "mL9kXp2W",
  "name": "ci-pipeline-key",
  "auth_token": "org-token:mL9kXp2WQrZvT8sBmN3cD4eF6gH0iJ1kL9pM3nO5qR7tU0vW1xY2zA3bC4dE5fG",
  "warning": "copy this token now; it will not be shown again"
}
```

**Camera:** Type-in animation. Highlight `auth_token` value and `warning` field — amber ring, 1s hold.

Narration (0:05–0:13):
> "One POST, a name, and the token is minted. 256 bits of entropy. The plaintext is shown exactly once — copy it now."

**Callout text (bottom-left):**
`One-time display. Never stored.`

---

## Moment 2 — Use the token (0:16–0:30)

**Terminal continues:**

```bash
ORG_TOKEN="org-token:mL9kXp2WQrZvT8sBmN3cD4eF6gH0iJ1kL9pM3nO5qR7tU0vW1xY2zA3bC4dE5fG"

# List all workspaces
curl -s "$PLATFORM/workspaces" \
  -H "Authorization: Bearer $ORG_TOKEN" | jq '.count'
```

**Terminal output:**

```json
{"count": 7}
```

**Terminal continues:**

```bash
# List all org tokens (as audit check)
curl -s "$PLATFORM/org/tokens" \
  -H "Authorization: Bearer $ORG_TOKEN" | jq '.tokens[] | "\(.name) — \(.created_by)"'
```

**Terminal output:**

```
"ci-pipeline-key — admin@example.com"
```

**Camera:** Run the two curl commands. Show JSON output. Hold on the org token listing with `created_by` attribution.

Narration (0:16–0:26):
> "Use it anywhere. Authorization header, full admin access. Every workspace, every bundle, every secret. Audit trail shows who minted it."

---

## Moment 3 — Revoke and confirm 401 (0:30–0:50)

**Terminal:**

```bash
# Revoke the token
curl -s -X DELETE "$PLATFORM/org/tokens/otok_a1b2c3d4e5f6" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -w "\nHTTP %{http_code}\n"
```

**Terminal output:**

```
HTTP 200
```

**Terminal immediately:**

```bash
# Confirm it's dead
curl -s "$PLATFORM/workspaces" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -w "\nHTTP %{http_code}\n"
```

**Terminal output:**

```
{"error":"invalid or revoked org api token"}
HTTP 401
```

**Camera:** Full revoke sequence. Hold on `HTTP 401` in red.

Narration (0:32–0:44):
> "Revoke it. One DELETE. The token dies immediately — the 401 confirms it. The same plaintext will never work again."

---

## Moment 4 — Canvas audit trail (0:50–0:56)

**Cut to:** Canvas — Org Settings → API Keys tab.

The revoked token now shows `revoked_at: 2026-04-21T00:04:30Z`. The `ci-pipeline-key` token is listed alongside any others with `created_by: admin@example.com`.

Narration (0:50–0:54):
> "The canvas shows the full audit trail. Who minted it, when, when it was revoked. Named tokens, full admin scope, instant revocation."

---

## Close (0:56–1:00)

**Terminal clean frame.**

Narration (0:56–0:58):
> "Org API keys — mint, use, revoke. No session cookies. No browser. Full admin access from the CLI."

**End card:**

```
Org-Scoped API Keys
workspace-server/internal/handlers/org_tokens.go — molecule-core#1105
```

**Fade to black.**

---

## Production Notes

- **Terminal theme:** Dark, SF Mono / JetBrains Mono 14pt, same as other demos.
- **HTTP status:** Use `curl -w "\nHTTP %{http_code}\n"` in all terminal demos to show status codes inline.
- **Callout style:** Amber ring `#E8A000`, 1s fade-in/out.
- **401 highlight:** Show the HTTP status in red (`\u001b[31m` ANSI if supported, or just text highlight).
- **Canvas cutaway:** Pre-record the Org Settings → API Keys tab with a live token in the list.
- **VO pacing:** Read against the timeline — the 0:32–0:44 revoke sequence is the climax; VO should land on "401" for emphasis.
