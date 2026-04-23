# Cloudflare Artifacts — DevRel Demo

**Issue:** [#1479](https://github.com/Molecule-AI/molecule-core/issues/1479) |
**Screencast:** ~60s walkthrough |
**Run time:** ~2 min (manual) / ~30s (dry-run with mock env)

This demo shows the full Cloudflare Artifacts workflow for a Molecule AI workspace:
attach a git repo, mint a short-lived credential, clone, write a snapshot, commit, push,
and fork for an experiment branch.

---

## Prerequisites

| Variable | Where to get it | Scope |
|---|---|---|
| `WORKSPACE_TOKEN` | Molecule AI Canvas → Workspace → API Keys | Workspace-level bearer token |
| `WORKSPACE_ID` | Molecule AI Canvas → Workspace → Settings | Workspace UUID |
| `PLATFORM_URL` | Self-hosted: your deployment URL. Cloud: `https://platform.moleculesai.app` | Platform base URL |
| `CF_ARTIFACTS_API_TOKEN` | Cloudflare Dashboard → API Tokens → Create Token (Templates: Artifacts Edit) | Platform env var (server-side) |
| `CF_ARTIFACTS_NAMESPACE` | Cloudflare Dashboard → Artifacts → Namespace ID | Platform env var (server-side) |

> **Note:** `CF_ARTIFACTS_API_TOKEN` and `CF_ARTIFACTS_NAMESPACE` are platform-level env vars — they do not appear in the demo script. The demo only calls the Molecule AI platform API; Cloudflare credentials are managed server-side.

### Required tools

```bash
curl jq git
# macOS
brew install curl jq git
# Linux (Debian/Ubuntu)
sudo apt-get install curl jq git
```

---

## Setup

```bash
# 1. Clone this repo
git clone https://github.com/Molecule-AI/molecule-core.git
cd molecule-core/docs/devrel/demos/cloudflare-artifacts

# 2. Set required env vars
export PLATFORM_URL="https://platform.moleculesai.app"   # or your self-hosted URL
export WORKSPACE_ID="ws_xxxxxxxxxxxx"                    # from Canvas → Workspace → Settings
export WORKSPACE_TOKEN="mk_live_xxxxxxxxxxxxxxxxxxxxxx"  # from Canvas → Workspace → API Keys

# 3. Make demo.sh executable and run it
chmod +x demo.sh
bash demo.sh
```

---

## Expected Output

### Step 1 — Attach repo

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  STEP 1: Attach a new Artifacts repo to workspace ws_xxx
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[12:34:56] Repo created:
[12:34:56]   id         : repo_abc123xxxxxxxx
[12:34:56]   name       : demo-1745200000
[12:34:56]   remote_url : https://x:***@hash.artifacts.cloudflare.net/git/repo-abc123.git
```

### Step 2 — Mint credential

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  STEP 2: Mint a short-lived git credential
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[12:34:57] Credential minted (username=x, token=***xxxxxx)
```

### Step 3 — Clone, write, commit, push

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  STEP 3: Clone repo · write agent snapshot · commit · push
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[12:34:58] Push succeeded
[12:34:58] Files in working tree:
[12:34:58] AGENT_SNAPSHOT.md
```

### Step 4 — Fork

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  STEP 4: Fork the repo for an experiment branch
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[12:35:00] Fork created: id=repo_abc123experiment
[12:35:00]   fork remote : https://x:***@hash.artifacts.cloudflare.net/git/repo-abc123experiment...
[12:35:00]   next step   : git clone <fork_url> && cd <repo> && git push
```

### Step 5 — Verify

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  STEP 5: Verify — list workspace Artifacts repos
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[12:35:01] Current workspace repos:
[12:35:01]   repo_abc123            demo-1745200000           (2026-04-21), remote: https://x:***@hash.artifacts.cloudflare.net/git/repo-abc123...
[12:35:01]   repo_abc123experiment  demo-1745200000-experiment (2026-04-21), remote: https://x:***@hash.artifacts.cloudflare.net/git/repo-abc123experiment...
```

---

## Screencast Shot List (~60 seconds)

| Time | On-screen | Audio |
|---|---|---|
| 0–10s | Canvas → Workspaces → Artifacts tab (empty) | "Every Molecule AI workspace can now have its own Git repo on Cloudflare's edge." |
| 10–25s | Terminal: run Step 1 curl → JSON response | "One API call creates the repo and returns a git remote URL." |
| 25–40s | Terminal: git clone → write AGENT_SNAPSHOT.md → commit → push | "The agent writes its work as a Git commit. Every run is versioned." |
| 40–50s | Run fork curl → show both repos in Canvas | "Before a risky change, the agent forks — the main branch stays clean." |
| 50–60s | Canvas: show commit history, point to Artifacts tab | "All of this is visible from Canvas — no terminal required for your team." |

---

## Troubleshooting

### `403 Forbidden` or `401 Unauthorized`

Workspace token is invalid or expired. Generate a fresh token at **Canvas → Workspace → API Keys**.

### `503 Cloudflare Artifacts not configured`

The platform server is missing `CF_ARTIFACTS_API_TOKEN` or `CF_ARTIFACTS_NAMESPACE`. This is a server-side configuration issue — contact your platform admin.

### `404 Not Found` on `/artifacts` endpoints

The platform version does not include the Artifacts integration. Ensure you're running `main` with `workspace-server/internal/handlers/artifacts.go` present.

### `Failed to create repo` with valid credentials

Check that the Cloudflare API token has **Artifacts Write** scope and the namespace ID is correct in Cloudflare Dashboard → Artifacts.

### First git push fails with "refusing to push to unrelated history"

Run `git pull origin main --allow-unrelated-histories` before pushing, or `git push -f` if the remote is empty and you want to establish it as the canonical history.

---

## Files

```
cloudflare-artifacts/
├── demo.sh          # Runnable bash demo (self-contained)
└── README.md        # This file
```

---

## Related Resources

- **Blog post:** [Give Your AI Agent a Git Repository](https://moleculesai.app/blog/cloudflare-artifacts-molecule-ai)
- **API reference:** [Platform API → Artifacts](/docs/api-protocol/platform-api)
- **Cloudflare Artifacts docs:** [developers.cloudflare.com/artifacts](https://developers.cloudflare.com/artifacts/)
- **Source:** `workspace-server/internal/handlers/artifacts.go` on `main`
