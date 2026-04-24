# Git for Agents: Cloudflare Artifacts Integration

**Source:** PR #641 (feat(platform): Cloudflare Artifacts demo integration #595), merged 2026-04-17
**Issue:** #1174
**Status:** Draft v1

---

Your AI agent has been working for three hours. It wrote tests, refactored a module, and left a summary in your workspace. Then your laptop died.

Without a shared version history, that work was in memory — gone. With Cloudflare Artifacts, it doesn't have to be.

Molecule AI's Cloudflare Artifacts integration treats every workspace snapshot as a first-class Git commit. Agents can branch, fork, push, and pull their own work — collaborating with peer agents or rolling back to a known-good state — without you touching a terminal.

---

## What Is Cloudflare Artifacts?

Cloudflare Artifacts is Cloudflare's "Git for agents" storage layer — a versioned, collaborative object store for AI agent workspaces. Each workspace gets a bare Git repository on CF's edge, and agents interact with it through a typed REST API.

Key properties:
- **Versioned** — every snapshot is a Git commit, accessible and diffable
- **Branching** — agents can fork an isolated copy before experimental changes
- **Short-lived credentials** — Git tokens minted on demand, revoked automatically
- **Edge-hosted** — hosted on Cloudflare's global network; low-latency access from wherever your agents run

This is a first-mover integration. As of 2026-04-17, no other AI agent platform has shipped a Git-backed workspace snapshot feature. The [Cloudflare blog post](https://blog.cloudflare.com/artifacts-git-for-agents-beta/) has the full context.

---

## How It Works in Molecule AI

The integration adds four operations to the workspace API:

| Operation | What it does |
|-----------|-------------|
| `POST /artifacts/repos` | Create a Git repo for the workspace |
| `POST /artifacts/repos/:name/fork` | Fork an isolated copy (branch-equivalent) |
| `POST /artifacts/repos/:name/import` | Bootstrap from an external Git URL |
| `POST /artifacts/tokens` | Mint a short-lived Git credential |

All tokens expire automatically. The Go client handles the credential lifecycle — tokens are never stored, never logged.

---

## Why It Matters for Agentic Workflows

Without versioned snapshots, AI agent work is ephemeral. Here's what that costs:

- **No rollback** — a bad agent decision means re-running from scratch
- **No collaboration** — two agents can't share a working context without manual handoff
- **No audit trail** — you can see what the agent did, but not what it changed

Cloudflare Artifacts changes all three. The workspace filesystem becomes a proper Git working tree. Every action is a commit. Branching is a first-class API call.

This is especially powerful for:

- **Multi-agent pipelines** — an agent writes to a feature branch, a reviewer agent pulls and approves, you merge to main
- **Long-running tasks** — checkpoint snapshots so a crash doesn't mean starting over
- **Experimentation** — fork before a risky refactor, delete the fork if it fails, keep the main clean

---

## Setup

```bash
# Set Cloudflare credentials
export CLOUDFLARE_API_TOKEN="your-cf-api-token"
export CLOUDFLARE_ARTIFACTS_NAMESPACE="your-namespace"

# Create a repo for the workspace
curl -X POST https://your-deployment.moleculesai.app/artifacts/repos \
  -H "Authorization: Bearer $ORG_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-workspace", "description": "Dev agent workspace"}'

# Fork before an experimental change
curl -X POST https://your-deployment.moleculesai.app/artifacts/repos/my-workspace/fork \
  -H "Authorization: Bearer $ORG_API_KEY" \
  -d '{"name": "my-workspace/experiment"}'
```

From the Molecule AI Canvas, navigate to **Workspaces → Your Workspace → Artifacts** to view repos, fork branches, and manage credentials visually.

---

## The Bigger Picture

Cloudflare Artifacts is part of the MCP governance layer. The combination of MCP tool-calling with versioned storage gives agents the primitives they need for production-grade workflows: capability discovery (via AGENTS.md), tool access (via MCP), and state persistence (via Cloudflare Artifacts).

Your agents stop being stateless. They become participants in a versioned, collaborative system — with the audit trail, rollback capability, and multi-agent coordination that production deployments require.

---

**Docs:** [Cloudflare Artifacts setup](/docs/guides/cloudflare-artifacts)
**PR:** [PR #641 on GitHub](https://github.com/Molecule-AI/molecule-core/pull/641)
