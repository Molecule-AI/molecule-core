# Screencast Storyboard — Cloudflare Artifacts Integration
**PR:** #641 | **Feature:** `POST/GET /workspaces/:id/artifacts`, `/artifacts/fork`, `/artifacts/token`
**Duration:** 60 seconds | **Format:** Terminal-led, clean dark theme

---

## Pre-roll (0:00–0:04)

**Canvas — full screen**
Single workspace card: `data-agent [ONLINE]`, status: `idle`.

Narration (0:00–0:04):
> "This data-agent has been running for three hours. It has context, task state, memory. What happens when it disconnects?"

**Camera:** Static Canvas frame. 3-second hold. No cursor.

---

## Moment 1 — Attach a CF Artifacts repo (0:04–0:16)

**Terminal:** `agent@data-agent:~$`

```bash
WORKSPACE_ID="ws-data-agent-001"
PLATFORM="https://acme.moleculesai.app"
TOKEN="Bearer ws-token-xxx"

curl -s -X POST "$PLATFORM/workspaces/$WORKSPACE_ID/artifacts" \
  -H "Authorization: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "data-agent-snapshots", "description": "Versioned snapshots of data-agent workspace"}' \
  | jq
```

**Terminal output:**
```json
{
  "id": "art-uuid-789",
  "workspace_id": "ws-data-agent-001",
  "cf_repo_name": "data-agent-snapshots",
  "remote_url": "https://hash.artifacts.cloudflare.net/git/data-agent-snapshots.git",
  "created_at": "2026-04-21T00:00:10Z"
}
```

**Camera:** Cursor to `remote_url`, highlight ring. Hold 1s.

Narration (0:06–0:14):
> "One API call attaches a Cloudflare Artifacts git repo to the workspace. A remote URL is returned — no CF dashboard required."

**Callout text (bottom-left):**
`Git for agents. No separate setup.`

---

## Moment 2 — Mint a credential, clone the repo (0:16–0:28)

```bash
TOKEN_RESP=$(curl -s -X POST "$PLATFORM/workspaces/$WORKSPACE_ID/artifacts/token" \
  -H "Authorization: $TOKEN" -H "Content-Type: application/json" \
  -d '{"scope": "write", "ttl": 3600}')

CLONE_URL=$(echo $TOKEN_RESP | jq -r '.clone_url')
git clone "$CLONE_URL" /tmp/data-agent-snapshots
```

**Terminal output:**
```
Cloning into '/tmp/data-agent-snapshots'...
Receiving objects: 100% | (12/12), 12.00 KiB, done.
```

**Camera:** Scroll through git clone output. Hold on `Receiving objects: 100%`.

Narration (0:18–0:26):
> "A short-lived git credential is minted — valid for one hour. The agent clones the repo. Cloudflare Artifacts handles the git transport."

---

## Moment 3 — Agent writes a snapshot (0:28–0:44)

```bash
cd /tmp/data-agent-snapshots
echo "# Workspace State — 2026-04-21" > snapshot.md
echo "current_task: analyzing sales pipeline Q1" >> snapshot.md
echo "uptime_seconds: 10800" >> snapshot.md
echo "last_status: COMPLETE" >> snapshot.md
git add snapshot.md
git commit -m "snapshot: pipeline analysis complete — 3 key findings"
git push origin main
```

**Terminal output:**
```
[main abc1234] snapshot: pipeline analysis complete — 3 key findings
 1 file changed, 5 insertions(+)
 remote: success
```

**Camera:** Full commit → push. Hold on `remote: success`. **Green ring pulse `#22C55E`**.

Narration (0:30–0:40):
> "The agent writes a snapshot — current task, data sources, key findings — commits and pushes. The state is now in Cloudflare Artifacts. Versioned. Recoverable."

**Callout text:**
`Versioned agent state — every push is a checkpoint.`

---

## Moment 4 — Fork the repo for a new workspace (0:44–0:54)

```bash
curl -s -X POST "$PLATFORM/workspaces/$WORKSPACE_ID/artifacts/fork" \
  -H "Authorization: $TOKEN" -H "Content-Type: application/json" \
  -d '{"name": "researcher-from-data-agent", "description": "Forked from data-agent workspace", "default_branch_only": true}' \
  | jq
```

**Terminal output:**
```json
{
  "fork": {"name": "researcher-from-data-agent", "namespace": "acme-production", "remote_url": "..."},
  "object_count": 47,
  "remote_url": "https://hash2.artifacts.cloudflare.net/git/researcher-from-data-agent.git"
}
```

**Camera:** Highlight `remote_url` and `object_count`. Hold 2s.

Narration (0:45–0:52):
> "Another agent forks the repo — a separate, isolated copy. 47 objects transferred. The new workspace can clone it and continue from the same point."

---

## Close (0:54–1:00)

**Terminal clean frame.** Cursor at prompt.

Narration (0:54–0:58):
> "Every workspace can have its own git history. Snapshot state, version it, fork it into a new agent. Git for agents, built into the platform."

**End card:**
```
Cloudflare Artifacts Integration
workspace-server/internal/handlers/artifacts.go — molecule-core#641
```
**Fade to black.**

---

## Production Spec

| Spec | Value |
|------|-------|
| Terminal theme | Same as AGENTS.md storyboard — dark, SF Mono 14pt / JetBrains Mono 13pt |
| Canvas cutaway | Dev canvas localhost:3000, pre-record before session |
| Camera | Screenflow / Camtasia, 1440×900 → 1080p export |
| JSON output | `jq --monochrome-output` or custom monochrome filter for dark theme |
| Callout highlight | Amber ring `#E8A000`, 1s fade-in/out |
| Green success | Green ring `#22C55E` on `remote: success` line, 1.5s hold |
| VO voice | Match AGENTS.md storyboard — same voice talent, consistent pacing |
| Music | None |
| Sound FX | Subtle single-tone click at 0:04 (repo attached) and 0:54 (end card) |
| Playback speed | curl/git/push sequence at 2x during Moments 1–4 |
