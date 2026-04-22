# Screencast Storyboard — Cloudflare Artifacts Integration

> **PR:** #641 | **Feature:** `POST/GET /workspaces/:id/artifacts`, `/artifacts/fork`, `/artifacts/token` | **Duration:** 60 seconds
> **Format:** Terminal-led, clean dark theme

---

## Pre-roll (0:00–0:04)

**Canvas — full screen**
Single workspace card in Canvas: `data-agent [ONLINE]`. Status: `idle`.

Narration (0:00–0:04):
> "This data-agent has been running for three hours. It has context, task state, memory. What happens when it disconnects?"

**Camera:** Static Canvas frame. 3-second hold. No cursor.

---

## Moment 1 — Attach a CF Artifacts repo (0:04–0:16)

**Cut to:** Terminal window, dark theme.

Prompt: `agent@data-agent:~$`

```bash
WORKSPACE_ID="ws-data-agent-001"
PLATFORM="https://acme.moleculesai.app"
TOKEN="Bearer ws-token-xxx"

curl -s -X POST "$PLATFORM/workspaces/$WORKSPACE_ID/artifacts" \
  -H "Authorization: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "data-agent-snapshots",
    "description": "Versioned snapshots of data-agent workspace"
  }' | jq
```

**Terminal output (JSON, formatted):**

```json
{
  "id": "art-uuid-789",
  "workspace_id": "ws-data-agent-001",
  "cf_repo_name": "data-agent-snapshots",
  "cf_namespace": "acme-production",
  "remote_url": "https://hash.artifacts.cloudflare.net/git/data-agent-snapshots.git",
  "created_at": "2026-04-21T00:00:10Z"
}
```

**Camera:** Cursor to `remote_url` field, highlight ring. Hold 1s.

Narration (0:06–0:14):
> "One API call attaches a Cloudflare Artifacts git repo to the workspace. A remote URL is returned — no CF dashboard required."

**Callout text (bottom-left):**
`Git for agents. No separate setup.`

---

## Moment 2 — Mint a credential, clone the repo (0:16–0:28)

**Terminal continues:**

```bash
# Mint a short-lived git credential
TOKEN_RESP=$(curl -s -X POST "$PLATFORM/workspaces/$WORKSPACE_ID/artifacts/token" \
  -H "Authorization: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope": "write", "ttl": 3600}')

CLONE_URL=$(echo $TOKEN_RESP | jq -r '.clone_url')

# Clone the workspace repo
git clone "$CLONE_URL" /tmp/data-agent-snapshots
```

**Terminal output:**

```
Cloning into '/tmp/data-agent-snapshots'...
remote: Enumerating objects: 12, done.
remote: Counting objects: 100% | (12/12), done.
Receiving objects: 100% | (12/12), 12.00 KiB, done.
```

**Camera:** Scroll through git clone output. Brief hold on `Receiving objects: 100%`. Clean finish.

Narration (0:18–0:26):
> "A short-lived git credential is minted — valid for one hour. The agent clones the repo. Cloudflare Artifacts handles the git transport."

---

## Moment 3 — Agent writes a snapshot (0:28–0:44)

**Terminal continues:**

```bash
cd /tmp/data-agent-snapshots

# Agent writes its state to the repo
echo "# Workspace State — 2026-04-21" > snapshot.md
echo "current_task: analyzing sales pipeline Q1" >> snapshot.md
echo "data_sources_analyzed: 8" >> snapshot.md
echo "key_findings: [revenue-drop-may, churn-signal-3pc, upsell-opportunity]" >> snapshot.md
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
 Counting objects: 100% | (3/3), done.
 Writing objects: 100% | (3/3), done.
 remote: success
```

**Camera:** Full commit → push sequence. Hold on `remote: success`. Green checkmark indicator.

Narration (0:30–0:40):
> "The agent writes a snapshot — current task, data sources, key findings — commits and pushes. The state is now in Cloudflare Artifacts. Versioned. Recoverable."

**Callout text:**
`Versioned agent state — every push is a checkpoint.`

---

## Moment 4 — Fork the repo for a new workspace (0:44–0:54)

**Terminal:**

```bash
# A new researcher workspace forks the data-agent's repo
curl -s -X POST "$PLATFORM/workspaces/$WORKSPACE_ID/artifacts/fork" \
  -H "Authorization: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "researcher-from-data-agent",
    "description": "Forked from data-agent workspace",
    "default_branch_only": true
  }' | jq
```

**Terminal output:**

```json
{
  "fork": {
    "name": "researcher-from-data-agent",
    "namespace": "acme-production",
    "remote_url": "https://hash2.artifacts.cloudflare.net/git/researcher-from-data-agent.git"
  },
  "object_count": 47,
  "remote_url": "https://hash2.artifacts.cloudflare.net/git/researcher-from-data-agent.git"
}
```

**Camera:** Highlight the `remote_url` and `object_count` fields. Hold 2s.

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

## Production Notes

- **Terminal theme:** Same as AGENTS.md storyboard — dark, SF Mono / JetBrains Mono 14pt.
- **Canvas cutaway (pre-roll + close):** Use dev canvas with one workspace in active state. Pre-record before the session.
- **Camera:** Screenflow / Camtasia. 1440×900 record → 1080p export.
- **Callout text:** Amber ring `#E8A000`, 1s fade-in/out, positioned bottom-left at 90% opacity on semi-transparent dark background.
- **Green success indicator:** On the `git push` moment, use a green ring pulse (`#22C55E`) for the `remote: success` line — 1.5s hold.
- **JSON jq output:** Use `jq` with a custom `.絹` (color) filter or `--monochrome-output` to keep it clean and readable in dark theme.
- **VO recording:** Match VO session with AGENTS.md storyboard — use the same voice talent and consistent pacing.
- **Music:** No music. Consider a subtle single-tone click at 0:04 (repo attached) and 0:54 (end card) for visual rhythm.
- **Speed:** The curl/git clone/push sequence should run at 2x playback in moments 1–4 for pacing. VO rides over the cuts.
