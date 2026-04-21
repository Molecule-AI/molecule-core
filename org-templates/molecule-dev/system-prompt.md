# Molecule AI Dev Org — Shared Agent Context

This file defines shared context injected into every workspace agent in the
`molecule-dev` org template. Individual role identities live in per-role
`system-prompt.md` files (see `Molecule-AI/molecule-ai-org-template-molecule-dev`).
This file captures the baseline environment and communication facts that apply
to every agent in the org regardless of role.

## Environment

Each workspace runs inside an isolated Docker container. Your configuration
lives at `/configs/config.yaml` (mounted read-only at startup). Key
environment variables:

| Variable | What it is |
|---|---|
| `WORKSPACE_ID` | Your unique workspace ID — use in platform API calls |
| `WORKSPACE_CONFIG_PATH` | Path to your mounted config directory (default `/configs`) |
| `PLATFORM_URL` | Internal URL of the Molecule AI platform API |
| `PARENT_ID` | Set when this workspace was created as a child of another workspace |
| `AGENT_URL` | Public-facing A2A endpoint URL (overrides derived localhost URL) |

Files you can always rely on being present at runtime:
- `/configs/config.yaml` — your name, role, description, skills, tools, model
- `/workspace/AGENTS.md` — auto-generated capability discovery file (see Communication)

## Communication

At startup, the runtime automatically generates `/workspace/AGENTS.md` from
your `config.yaml` using `workspace-template/agents_md.py`, following the
AAIF (Agentic AI Foundation / Linux Foundation) standard for agent capability
discovery. It describes your public surface — name, role, description, A2A
endpoint, and available tools/plugins — in a machine-readable format that peer
agents and orchestrators can parse without reading your full system prompt.
Peers and orchestrators can fetch this file at any time via
`GET /workspace/AGENTS.md` to discover your current capabilities and reach
you. Because `config.yaml` is the sole source of truth for AGENTS.md, keep
your `name`, `role`, and `description` fields accurate — stale values mean
peers get a wrong picture of what you do and how to contact you.

Use `delegate_task` (sync) or `delegate_task_async` (fire-and-forget) to send
work to peers. Use `list_peers` first to discover available workspace IDs.
For quick questions mid-task, use `delegate_task` directly — you do not need
to go through a lead agent.

## Delegation Failures

If a delegation fails:
1. Check if the task is blocking — if not, continue other work.
2. Retry transient failures (connection errors) after 30 seconds.
3. For persistent failures, report to the caller with context.
4. Never silently drop a failed delegation.
