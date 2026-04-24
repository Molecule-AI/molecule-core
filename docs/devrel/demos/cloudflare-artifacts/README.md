# Cloudflare Artifacts — Demo
**Issue:** #1479 | **Source:** PR #641 | **Handler:** `workspace-server/internal/handlers/artifacts.go`

---

## What This Demo Shows

1. Attach a Cloudflare Artifacts Git repo to a workspace
2. Mint a short-lived git credential (shown once, never stored server-side)
3. Clone, write a file, commit, push — every agent run becomes a Git commit
4. Fork the repo before a risky experiment

**Time:** ~3 min | **Requirements:** `pip install requests`

---

## Quick Start

```bash
export PLATFORM_URL=https://your-deployment.moleculesai.app
export WORKSPACE_TOKEN=your-workspace-token
export WORKSPACE_ID=your-workspace-id

python demo.py
```

### Offline mode (no platform needed)

```bash
python demo.py
# Simulated responses — no credentials required
```

---

## Step-by-Step Walkthrough

### Step 1 — Attach a Cloudflare Artifacts repo

```bash
curl -s -X POST "$PLATFORM_URL/workspaces/$WORKSPACE_ID/artifacts" \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "agent-demo", "description": "Demo workspace"}' | jq .
```

```json
{
  "id": "wa_abc123",
  "workspace_id": "ws-demo-001",
  "cf_repo_name": "molecule-ws-demo",
  "cf_namespace": "molecule-prod",
  "remote_url": "https://artifacts.cloudflare.net/git/molecule-ws-demo",
  "created_at": "2026-04-23T10:00:00Z"
}
```

### Step 2 — Mint a short-lived Git credential

```bash
curl -s -X POST "$PLATFORM_URL/workspaces/$WORKSPACE_ID/artifacts/token" \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope": "write", "ttl": 3600}' | jq .
```

```json
{
  "token_id": "tok_xyz789",
  "token": "cf_tok_abc123...",
  "scope": "write",
  "expires_at": "2026-04-23T11:00:00Z",
  "clone_url": "https://x:cf_tok_abc123...@artifacts.cloudflare.net/git/molecule-ws-demo.git",
  "message": "Save this token — it cannot be retrieved again."
}
```

> **Copy the token now.** It's shown exactly once at mint time.

### Step 3 — Git clone, write, commit, push

```bash
git clone https://x:$TOKEN@artifacts.cloudflare.net/git/molecule-ws-demo.git demo-workspace
cd demo-workspace

# Agent writes its work as a Git commit
echo "# Agent Run — $(date)" > AGENT_SNAPSHOT.md
git add AGENT_SNAPSHOT.md
git commit -m "feat: agent run snapshot"
git push origin main
```

### Step 4 — Fork before a risky experiment

```bash
curl -s -X POST "$PLATFORM_URL/workspaces/$WORKSPACE_ID/artifacts/fork" \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "agent-demo/experiment", "default_branch_only": true}' | jq .
```

```json
{
  "fork": {"name": "agent-demo/experiment", "namespace": "molecule-prod"},
  "object_count": 14
}
```

Fork succeeds → merge back. Fork fails → discard. Main stays clean.

---

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| POST | `/workspaces/:id/artifacts` | Attach/create CF Artifacts repo |
| GET | `/workspaces/:id/artifacts` | Get linked repo info |
| POST | `/workspaces/:id/artifacts/token` | Mint short-lived git credential |
| POST | `/workspaces/:id/artifacts/fork` | Fork the workspace's primary repo |
| DELETE | `/workspaces/:id/artifacts` | Detach the linked repo |

---

## Key Design Decisions

### Credentials never stored server-side
The plaintext token is returned **only at mint time** — `stripCredentials()` removes `x:<token>@` from the URL before the DB row is written. The token value is never persisted.

### Per-call token minting
Each `git push` uses a fresh short-lived credential. The token TTL defaults to 3600s (1 hour), max 7 days. Compromised token = 1-hour window.

### SSRF protection
Import URLs (`POST /workspaces/:id/artifacts` with `import_url`) must use `https://`. Any other scheme returns 400.

### Forks not recorded in DB
`POST /artifacts/fork` creates a CF-side fork but does not write a `workspace_artifacts` row — the caller owns the fork, not the platform.

---

## Files

- `demo.py` — Runnable Python demo (simulated + live modes)
- `demo.md` — Original script-style demo (`docs/marketing/devrel/demos/cloudflare-artifacts-demo.md`)
- Handler: `workspace-server/internal/handlers/artifacts.go`
- Tests: `workspace-server/internal/handlers/artifacts_test.go`
