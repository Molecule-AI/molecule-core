# Screencast Storyboard — Org-Scoped API Keys
**PR:** #1105 | **Feature:** `GET/POST/DELETE /org/tokens`, `OrgTokensTab.tsx`
**Duration:** 60 seconds | **Format:** Canvas UI-led + terminal, dark zinc theme

---

## Pre-roll (0:00–0:04)

**Canvas — Settings panel open, Org Tokens tab active**
Tokens list is empty. Prompt at bottom: `admin@acme-platform:~$`

Narration (0:00–0:04):
> "Every platform has the same problem: one shared admin key, full access, no audit trail. Org-scoped tokens change that."

**Camera:** Static Settings frame. 3-second hold. No cursor movement.

---

## Moment 1 — Mint a token from the canvas (0:04–0:18)

**Canvas:** Org Tokens tab. "Create new token" button highlighted with amber ring `#E8A000`.

Cursor clicks "Create new token". Name input field appears. Types `ci-pipeline-key`.

Camera: Name field highlighted. "Create" button pulses.

Click "Create".

**Terminal overlay appears (bottom-left slide-in):**
```bash
# Token created — plaintext shown exactly once
ORG_TOKEN="org_tk_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
PREFIX="ci-pipeline"  # stored for reference only

echo "Token created: $PREFIX... (shown once, never retrievable)"
# → Token created: ci-pipeline... (shown once, never retrievable)
```

**Canvas:** New token appears in list with prefix `ci-pipeline••••••`, created just now, last used: never.

Narration (0:08–0:16):
> "One click in the canvas. The plaintext token is shown exactly once — copy it now or it's gone. No more shared global keys."

**Camera:** Clip to token list item. `ci-pipeline••••••` row highlighted. Created-at timestamp shown. Hold 2s.

---

## Moment 2 — Use the token in CI (0:18–0:30)

**Terminal:** Full-screen, clean.

```bash
# CI pipeline — deploys to workspace ws-acme-001
curl -s -X POST "$PLATFORM/workspaces/ws-acme-001/artifacts" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-deploy-snapshots"}' \
  | jq
```

**Terminal output:**
```json
{
  "id": "art-uuid-999",
  "workspace_id": "ws-acme-001",
  "cf_repo_name": "ci-deploy-snapshots",
  "remote_url": "https://hash.artifacts.cloudflare.net/git/ci-deploy-snapshots.git",
  "created_at": "2026-04-23T00:00:00Z"
}
```

Narration (0:19–0:28):
> "The CI pipeline uses the org token to attach an artifacts repo — scoped to this org, nothing else. No global ADMIN_TOKEN needed."

**Camera:** Full commit → push sequence. Hold on JSON output. `remote_url` highlighted with amber ring. Hold 1.5s.

---

## Moment 3 — Workspace isolation (0:30–0:42)

**Terminal:**
```bash
# Try to access a different workspace — still within same org
curl -s -X GET "$PLATFORM/workspaces/ws-other-team/artifacts" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  | jq
```

**Terminal output:**
```json
{
  "id": "art-uuid-888",
  "workspace_id": "ws-other-team",
  "cf_repo_name": "other-team-snapshots",
  "remote_url": "https://hash.artifacts.cloudflare.net/git/other-team-snapshots.git"
}
```

Narration (0:31–0:38):
> "Same org token, different workspace — still works. Org-scoped means every workspace in the org is accessible, but nothing outside it."

**Camera:** JSON output highlighted. Hold on `workspace_id` field. Amber ring 1.5s.

```bash
# What about a workspace in a DIFFERENT org?
curl -s -X GET "$PLATFORM/workspaces/ws-external/executions" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  | jq -r '.error // .message'
```

**Terminal output:**
```json
{"error": "unauthorized", "message": "token org does not match target workspace org"}
```

Narration (0:40–0:42):
> "Different org — access denied. The token knows its scope."

**Camera:** Error JSON. Red ring `#EF4444` on `unauthorized` field. 1.5s hold.

---

## Moment 4 — Revoke (0:42–0:52)

**Canvas:** Org Tokens tab. `ci-pipeline••••••` row. "Revoke" button hovered.

Click "Revoke". Confirm dialog appears.

Click "Confirm Revoke".

**Terminal:**
```bash
# Token is now invalid
curl -s -X GET "$PLATFORM/workspaces/ws-acme-001/artifacts" \
  -H "Authorization: Bearer $ORG_TOKEN" \
  | jq -r '.error // .message'
# → token revoked or not found
```

**Canvas:** Token row disappears from list.

Narration (0:44–0:50):
> "One click to revoke. The token is dead immediately — no grace period, no cached sessions."

**Camera:** Empty tokens list. Hold 2s.

---

## Close (0:52–1:00)

**Canvas:** Org Tokens tab — empty state message: "No org tokens yet. Create one to get started."

Narration (0:52–0:56):
> "Org-scoped API keys. Mint from the canvas, use in CI, revoke in one click. Replace your shared admin key."

**End card:**
```
Org-Scoped API Keys
canvas/src/components/settings/OrgTokensTab.tsx — molecule-core#1105
Org tokens: GET/POST/DELETE /org/tokens
```

**Fade to black.**

---

## Production Spec

| Spec | Value |
|------|-------|
| Terminal theme | Dark, SF Mono 14pt / JetBrains Mono 13pt |
| Canvas theme | Zinc-900 background, same as existing Phase 30 assets |
| Camera | Camtasia / ScreenFlow, 1440×900 → 1080p export |
| JSON output | Monochrome jq filter, same as CF Artifacts storyboard |
| Amber highlight | `#E8A000` ring on key output fields, 1.5s hold |
| Red error highlight | `#EF4444` on `unauthorized` field |
| Green success | `#22C55E` on token created confirmation |
| VO voice | Same talent as CF Artifacts storyboard, consistent pacing |
| Music | None |
| Sound FX | Subtle click at 0:04 (Create clicked), click at 0:52 (Revoke confirmed) |
| Playback speed | curl sequences at 1.5x during Moments 2–3 |
| Canvas recording | Pre-record on localhost:3000 with test tokens before session |