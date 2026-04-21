# Phase 30 Remote Workspaces — One-Pager

> **For:** Sales + prospects | **Length:** 1 page | **Format:** PDF-ready

---

## What it is

Remote Workspaces let you run Molecule AI agents on your own infrastructure — your laptop, a cloud VM, an on-premises server. They register to your Molecule AI org and appear in Canvas alongside your managed (container) workspaces. Same auth. Same A2A protocol. Same governance.

**The only visible difference:** a purple REMOTE badge on the workspace card.

---

## What changes for the buyer

| | Before Phase 30 | After Phase 30 |
|---|---|---|
| Agent runtime | Platform-managed only | Platform-managed OR self-hosted |
| Fleet visibility | Container workspaces only | Mixed fleet, one Canvas |
| Data residency | Agent compute on Molecule AI infra | Agent compute on your infra |
| Governance model | Identical across runtimes | Identical across runtimes |

---

## What this enables (real use cases)

**Developer teams:** Run a local agent on your laptop for debugging with your IDE, then point the same agent at the org for production tasks. No environment switching.

**Data engineering teams:** Keep raw data on your own AWS/GCP/on-prem infrastructure while using the platform for orchestration. Data residency requirement solved.

**Enterprise platform teams:** Run agents across three clouds — visible in one Canvas, governed by the same org auth. Multi-cloud fleet, single governance plane.

**Existing agent integrations:** Don't want to containerize and redeploy? Register your existing agent with the org. It appears in Canvas without code changes.

---

## What ships with Phase 30

1. **Workspace auth tokens** — 256-bit bearer tokens, minted at registration. No shared secrets.
2. **Token-gated secrets pull** — API keys pulled at boot from the platform. No credentials baked into images.
3. **Reverse proxy (`/cp/*`)** — Allowlist-based same-origin access for internal APIs. Fail-closed.
4. **AdminAuth WorkOS session tier** — 30s positive / 5s negative cache. Tenant-scoped.
5. **AGENTS.md auto-generation** — Auto-generated agent manifest at workspace boot. Peer agents can read each other's identity without system prompts. (AAIF standard.)
6. **Cloudflare Artifacts integration** — Workspace git repos, snapshot/push, fork. "Git for agents."
7. **Remote runtime** — Agent binary connects via WSS. No inbound ports, no VPN. Outbound HTTPS only.
8. **Mixed-fleet Canvas** — Container + remote workspaces visible together, real-time status.

---

## What stays the same

- A2A protocol works across container/remote without code changes
- MCP governance (plugin allowlists, org API keys, audit logs) applies identically
- Org-scoped auth and session-tier controls apply identically
- Canvas, task dispatch, and parent/child relationships work across runtimes

---

## Pricing

Remote workspaces = container workspace pricing at GA. No premium for the remote runtime.

---

## Quick start

```bash
# 1. Install
curl -sSL https://get.moleculesai.app | bash

# 2. Authenticate
molecule login --org your-org

# 3. Bootstrap
molecule workspace init --name my-agent --runtime remote

# 4. It appears in Canvas in ~10 seconds
```

**Docs:** `moleculesai.app/docs/guides/remote-workspaces`
**Launch post:** `moleculesai.app/blog/remote-workspaces-ga`
**Demos:** `moleculesai.app/docs/marketing/demos`

---

## Competitive differentiation

| Competitor | Their claim | Our answer |
|---|---|---|
| Modal / Railway | "Managed infra" | They own compute; we let you own yours |
| Cursor / Copilot | "AI coding assistant" | Single-agent; we do multi-agent coordination |
| CrewAI / Autogen | "Open-source agents" | DIY infra + governance; we give you the platform day one |
| Windsurf / Devin | "Autonomous coding agent" | No org-level governance; we have it built in |

---

*Replace docs links with live URLs before distributing. Quick-start commands need verification against actual `molecule` CLI — Phase 30 storyboard uses Python SDK (`python3 run.py --runtime external`). Pricing section requires PMM confirmation.*
