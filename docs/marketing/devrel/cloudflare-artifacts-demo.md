# Cloudflare Artifacts — DevRel Demo README

**Source:** `workspace-server/internal/handlers/artifacts.go` + `docs/devrel/demos/cloudflare-artifacts/demo.sh`
**Feature:** Cloudflare Artifacts git integration for AI agent workspaces (PR #641, shipped Apr 2026)
**Screencast:** ~60s (see shot list below)
**Run time:** ~2 min with live credentials

---

## What this demo shows

Every Molecule AI workspace can now have its own **Git repository on Cloudflare's edge** — no credential management, no self-hosted Git server. The agent works, commits get written, history stays auditable.

### The 4-step workflow

```bash
# Step 1 — Attach a repo to your workspace (1 API call)
curl -X POST https://platform.moleculesai.app/workspaces/$WORKSPACE_ID/artifacts \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" \
  -d '{"name": "agent-snapshots"}'

# Returns: repo ID + git remote URL (e.g. https://x:***@hash.artifacts.cloudflare.net/git/agent-snapshots.git)

# Step 2 — Mint a short-lived git credential (1 API call, expires in 1h by default)
curl -X POST https://platform.moleculesai.app/workspaces/$WORKSPACE_ID/artifacts/token \
  -H "Authorization: Bearer $WORKSPACE_TOKEN"

# Returns: { "token": "...", "expires_at": "...", "clone_url": "..." }

# Step 3 — Clone, write, commit, push
git clone https://x:$TOKEN@hash.artifacts.cloudflare.net/git/agent-snapshots.git
cd agent-snapshots
echo "# Agent run — $(date)" >> SNAPSHOT.md
git add . && git commit -m "feat: agent snapshot" && git push

# Step 4 — Fork before a risky change (isolated experiment branch)
curl -X POST https://platform.moleculesai.app/workspaces/$WORKSPACE_ID/artifacts/fork \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" \
  -d '{"name": "agent-snapshots-experiment"}'
```

### What makes this different from a regular Git repo

| | Regular Git | Cloudflare Artifacts |
|---|---|---|
| Setup | Create account, manage keys, configure SSH | 1 API call, no account needed |
| Credential lifetime | Long-lived (rotate manually) | Short-lived, minted per-session (TTL: 1h–7d) |
| Scope | Global (whole platform) | Per-repo, from a single CF namespace |
| Where | Self-hosted or GitHub/GitLab | Managed by Cloudflare, on the edge |
| Audit trail | Partial | Full Cloudflare audit log on every access |

---

## Screencast shot list (60 seconds)

| Time | On-screen | What you say |
|---|---|---|
| 0–10s | Canvas → Workspaces → Artifacts tab (empty) | "Every Molecule AI workspace can now have its own Git repo on Cloudflare's edge." |
| 10–25s | Terminal: Step 1 curl → JSON response | "One API call creates the repo and returns a git remote URL." |
| 25–40s | Terminal: git clone → write snapshot → commit → push | "The agent writes its work as a Git commit. Every run is versioned." |
| 40–50s | Run fork curl → show both repos in Canvas | "Before a risky change, the agent forks — the main branch stays clean." |
| 50–60s | Canvas: show commit history, Artifacts tab | "All of this is visible from Canvas — no terminal required." |

---

## Phase 30 video production spec — Status

**File:** `marketing/devrel/phase30-video-production.md`
**Status:** ❌ Not found in repo. The file does not exist at any path matching `**/phase30-video-production.md`.

**Recommendation:** Create the spec at `docs/marketing/devrel/phase30-video-production.md` before Phase 30 campaign assets go live, or confirm it's stored in the internal `Molecule-AI/internal` repo (which requires credentials we don't have).

---

## Prerequisites for the demo

- `WORKSPACE_TOKEN` — from Canvas → Workspace → API Keys
- `WORKSPACE_ID` — from Canvas → Workspace → Settings
- Platform env vars: `CF_ARTIFACTS_API_TOKEN` + `CF_ARTIFACTS_NAMESPACE` (server-side, not in the demo script)
- Tools: `curl`, `jq`, `git`

**Run:**
```bash
git clone https://github.com/Molecule-AI/molecule-core.git
cd molecule-core/docs/devrel/demos/cloudflare-artifacts
export PLATFORM_URL="https://platform.moleculesai.app"  # or your self-hosted URL
export WORKSPACE_ID="ws_xxxxxxxxxxxx"
export WORKSPACE_TOKEN="mk_live_xxxxxxxxxxxxxxxxxxxxxx"
chmod +x demo.sh && bash demo.sh
```