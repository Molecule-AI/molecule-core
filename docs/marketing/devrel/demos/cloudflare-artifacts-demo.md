# Cloudflare Artifacts — Interactive Demo Script
**Issue:** #1173 | **Source:** PR #641 | **Acceptance:** Working demo + repo link + 1-min screencast

---

## What This Demo Shows

1. Provision a Cloudflare Artifacts Git repo for a workspace
2. Clone it, write a file, push a commit
3. Fork a branch, make a change, merge back

**Time:** ~60 seconds | **Tools:** curl, git, Molecule AI Canvas | **Setup:** `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ARTIFACTS_NAMESPACE`

---

## Demo Script

### Step 1: Create a Repo

```bash
curl -s -X POST https://your-deployment.molecule.ai/artifacts/repos \
  -H "Authorization: Bearer $ORG_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "demo-workspace", "description": "Agent demo workspace"}' | jq .
```

Expected output:
```json
{
  "id": "repo_abc123",
  "name": "demo-workspace",
  "remote_url": "https://x:<TOKEN>@hash.artifacts.cloudflare.net/git/repo-abc123.git",
  "created_at": "2026-04-21T00:00:00Z"
}
```

**Narrative:** "Every Molecule AI workspace can now have its own versioned Git repo on Cloudflare's edge."

---

### Step 2: Clone and Push a Snapshot

```bash
# Clone the repo (TOKEN is embedded in the remote URL from Step 1)
git clone https://x:<TOKEN>@hash.artifacts.cloudflare.net/git/repo-abc123.git demo-workspace
cd demo-workspace

# Write a snapshot note
cat > AGENT_SNAPSHOT.md << 'EOF'
# Agent Run — 2026-04-21

Task: Refactored the auth module. 3 tests added, 1 bug fixed.
Status: Complete. Ready for reviewer agent.
EOF

git add AGENT_SNAPSHOT.md
git commit -m "feat: agent run snapshot — auth module refactor"
git push origin main
```

**Narrative:** "The agent writes its work as a Git commit. Every run is versioned."

---

### Step 3: Fork Before an Experiment

```bash
# Fork the workspace — creates an isolated branch
curl -s -X POST https://your-deployment.molecule.ai/artifacts/repos/demo-workspace/fork \
  -H "Authorization: Bearer $ORG_API_KEY" \
  -d '{"name": "demo-workspace/experiment"}' | jq '.repo.remote_url'
```

```bash
git clone https://x:<TOKEN>@hash.artifacts.cloudflare.net/git/repo-abc123-fork.git exp-workspace
cd exp-workspace

# Experimental change
cat > experimental.md << 'EOF'
# Experimental: New auth strategy
Testing a token-less approach using WorkOS session tokens.
EOF

git add experimental.md
git commit -m "feat(experiment): token-less auth prototype"
git push origin main
```

**Narrative:** "Before a risky change, the agent forks — like a Git branch. If it fails, main stays clean."

---

### Step 4: View in Canvas

Open **Workspaces → demo-workspace → Artifacts** tab:
- See both repos (main + experiment fork)
- View commit history
- Clone or download

**Narrative:** "All of this is visible from the Molecule AI Canvas — no terminal required."

---

## Screencast Outline (~60s)

| Time | Action |
|------|--------|
| 0–10s | Open Canvas → Workspaces → Artifacts tab |
| 10–25s | Run Step 1 curl → show repo created in UI |
| 25–45s | Show git clone + commit + push in terminal |
| 45–55s | Run fork step, show experiment branch in Canvas |
| 55–60s | Zoom commit history — "every agent run is a Git commit" |

---

## Files

- Demo script: `docs/marketing/devrel/demos/cloudflare-artifacts-demo.sh`
- Canvas screenshot: `docs/marketing/devrel/demos/cloudflare-artifacts-canvas.png`
