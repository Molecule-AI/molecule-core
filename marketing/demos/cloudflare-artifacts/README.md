# Cloudflare Artifacts — Working Demo

> **PR:** #641 — Cloudflare Artifacts demo integration  
> **What it ships:** `POST/GET /workspaces/:id/artifacts`, `POST /workspaces/:id/artifacts/fork`, `POST /workspaces/:id/artifacts/token`  
> **Concept:** "Git for agents" — versioned workspace snapshot storage  
> **Acceptance criteria:** working demo showing workspace snapshot to/from Cloudflare Artifacts + 1-min screencast

---

## What This Demo Shows

A workspace links to a Cloudflare Artifacts git repo. The agent can push snapshots (git commits) and later fork the repo to bootstrap a new workspace. This is versioned workspace state — like `git init` for agent memory.

**The flow:**
1. Attach a CF Artifacts repo to a workspace (or import an existing Git repo)
2. Mint a short-lived git credential via the platform
3. Agent clones the repo, writes a snapshot, pushes
4. Fork the repo to bootstrap a new workspace

---

## Prerequisites

- Molecule AI platform with `CF_ARTIFACTS_API_TOKEN` and `CF_ARTIFACTS_NAMESPACE` set
- A running workspace with a bearer token
- `git` and `curl` on the caller machine

---

## Working Demo Script

### 1. Attach / create a CF Artifacts repo to a workspace

```bash
# Admin token or workspace token
WORKSPACE_ID=ws-abc123
PLATFORM=https://acme.moleculesai.app
TOKEN=your-workspace-or-admin-token

# Create (or import) the repo
curl -s -X POST "$PLATFORM/workspaces/$WORKSPACE_ID/artifacts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-workspace-snapshots",
    "description": "Versioned snapshots of workspace state"
  }' | jq
```

Response (201):
```json
{
  "id": "art-uuid-456",
  "workspace_id": "ws-abc123",
  "cf_repo_name": "my-workspace-snapshots",
  "cf_namespace": "my-namespace",
  "remote_url": "https://hash.artifacts.cloudflare.net/git/my-workspace-snapshots.git",
  "description": "Versioned snapshots of workspace state",
  "created_at": "2026-04-20T12:00:00Z"
}
```

The repo was created in Cloudflare Artifacts and linked to the workspace. No separate CF dashboard login needed.

---

### 2. Import an existing GitHub repo instead

```bash
curl -s -X POST "$PLATFORM/workspaces/$WORKSPACE_ID/artifacts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "researcher-agent",
    "description": "Researcher agent workspace",
    "import_url": "https://github.com/myorg/researcher-agent.git",
    "import_branch": "main",
    "import_depth": 1
  }' | jq
```

The platform calls the CF Artifacts API to import the GitHub repo. The workspace now has a full git history of the agent's work.

---

### 3. Mint a git credential

```bash
curl -s -X POST "$PLATFORM/workspaces/$WORKSPACE_ID/artifacts/token" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope": "write", "ttl": 3600}' | jq
```

Response:
```json
{
  "token": "cf_at_xxxxx...xxxx",
  "scope": "write",
  "expires_at": "2026-04-20T13:00:00Z",
  "clone_url": "https://x:cf_at_xxxxx...xxxx@artifacts.cloudflare.net/git/my-workspace-snapshots.git",
  "message": "Save this token — it cannot be retrieved again."
}
```

The `clone_url` is the authenticated git remote. Use it directly:

```bash
git clone https://x:cf_at_xxxxx@artifacts.cloudflare.net/git/my-workspace-snapshots.git
```

The token is scoped to this workspace's repo only. It expires in 1 hour (configurable up to 7 days).

---

### 4. Clone, snapshot, push

```bash
# Clone the workspace repo
git clone "https://x:cf_at_xxxxx@artifacts.cloudflare.net/git/my-workspace-snapshots.git" \
  /tmp/workspace-snapshots

cd /tmp/workspace-snapshots

# Agent writes a snapshot: memory dump, active task state, config
echo "current_task: researching competitor X" > snapshot.md
echo "uptime_seconds: 3600" >> snapshot.md
echo "memory_summary: analyzed 12 sources, 3 key findings" >> snapshot.md

git add snapshot.md
git commit -m "snapshot: researching competitor X — 3 findings ready"
git push origin main
```

The workspace state is now in Cloudflare Artifacts — versioned, accessible to other workspaces, recoverable.

---

### 5. Fork the repo for a new workspace

```bash
# Researcher wants to start from the PM's workspace snapshot
curl -s -X POST "$PLATFORM/workspaces/ws-pm-123/artifacts/fork" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "researcher-from-pm",
    "description": "Forked from pm-agent workspace",
    "default_branch_only": true
  }' | jq
```

Response:
```json
{
  "fork": {
    "name": "researcher-from-pm",
    "namespace": "my-namespace",
    "remote_url": "https://hash2.artifacts.cloudflare.net/git/researcher-from-pm.git"
  },
  "object_count": 47,
  "remote_url": "https://hash2.artifacts.cloudflare.net/git/researcher-from-pm.git"
}
```

The forked repo is a separate Cloudflare Artifacts repository with the full snapshot history. A new workspace can clone it and pick up where the PM left off.

---

## Screencast Outline (1 min)

**0:00–0:10** Canvas: a workspace is online. Terminal: `curl POST /workspaces/:id/artifacts` — repo created, response shows CF Artifacts remote URL.

**0:10–0:25** Terminal: mint a git credential. `clone_url` shown in response. `git clone` runs, repo clones in <5s.

**0:25–0:40** Agent writes a workspace snapshot to the repo. `echo` → `git add` → `git commit` → `git push`. Output shows the push succeeded.

**0:40–0:55** Canvas: fork call. `POST /workspaces/:id/artifacts/fork` → new repo created in CF Artifacts. The new workspace ID is returned.

**0:55–1:00** Narration: *"Every workspace can have its own git history. Snapshot state, version it, fork it into a new agent. Git for agents, built into the platform."*

---

## TTS Narration Script (30s)

> Cloudflare Artifacts turns your Molecule AI workspace into a versioned git repository. Attach a repo, mint a short-lived credential, and the agent can push snapshots — memory dumps, task state, config — and other agents can fork the history to bootstrap from the same point. No external git service configuration. No separate dashboard. The platform manages the credential lifecycle and the repo link. Versioned agent state, built into the platform. That's the first-mover advantage: Git for agents, from Molecule AI.

---

## API Reference

| Method | Path | What |
|---|---|---|
| `POST` | `/workspaces/:id/artifacts` | Attach/create CF Artifacts repo |
| `GET` | `/workspaces/:id/artifacts` | Get linked repo info |
| `POST` | `/workspaces/:id/artifacts/fork` | Fork repo to new workspace |
| `POST` | `/workspaces/:id/artifacts/token` | Mint short-lived git credential |

**Source:** `workspace-server/internal/handlers/artifacts.go` (PR #641)