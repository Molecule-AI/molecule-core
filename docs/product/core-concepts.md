# Core Concepts

## Workspace

The fundamental unit of the platform. A workspace is:

- A **role** (e.g. "Marketing", "Developer PM", "QA") — what this position in the org chart does
- A **container** that holds one AI agent (swappable without changing the role)
- An **A2A server** with a public endpoint and an Agent Card
- Optionally a **team** — it can contain sub-workspaces recursively

A workspace appears as a single node on the canvas regardless of whether it contains one agent or an entire team. The internal structure is opaque to parent workspaces — exactly as A2A intends.

### Why This Matters

From the outside, a workspace containing a single agent and a workspace containing a team of five agents look **identical**. Both have the same A2A endpoint. Both publish an Agent Card. The parent workspace delegates without knowing or caring what's inside.

**Practical consequence:** users can start with a single "Developer" agent, and when they need more capacity, expand it into a Developer Team (PM + Frontend + Backend + QA) without rewiring anything. The relationship between Business Core and Developer stays the same.

When expanded, the workspace becomes the **team lead** — its agent stays as a coordinator that receives incoming messages and delegates to sub-workspaces. Sub-workspaces can talk to each other and to the team lead, but not to workspaces outside the team. This is recursive — sub-workspaces can themselves expand into teams.

See [Team Expansion](../agent-runtime/team-expansion.md) for the full mechanics.

## Agent

The AI inside a workspace. An agent is swappable — you can replace Claude with GPT-4o or a local Ollama model without changing the workspace role, hierarchy position, or config.

The agent is powered by Deep Agents (LangGraph harness) and can:
- Plan using a TODO list tool
- Use tools
- Spawn sub-agents
- Maintain filesystem-backed memory
- Pause and escalate approval up the hierarchy (human-in-the-loop)

See [System Prompt Structure — Human-in-the-Loop](../agent-runtime/system-prompt-structure.md#human-in-the-loop-hierarchical-approval) for the escalation mechanics.

### Agent Handoff

When an agent is replaced (`AGENT_REPLACED`), the workspace performs a graceful handoff:

1. The outgoing agent wraps up its current task
2. The outgoing agent writes a comprehensive handoff document to the workspace's memory (saved files — current work state, in-progress tasks, decisions made, context)
3. The new agent starts and reads the handoff document from memory
4. The new agent picks up where the old one left off

This is why workspaces always persist their current state and TODO list as files — it's the handoff mechanism. The workspace's memory survives agent replacement (the volume or store persists), so the new agent inherits full context.

## Workspace Bundle

The portable, exportable artifact for a workspace. A `.bundle.json` file contains everything needed to recreate the workspace: system prompt, skills, prompt templates, tool configs, and sub-workspace definitions recursively.

Bundles are the unit of:
- Copy/paste
- Import/export
- Future marketplace

See [Bundle System](../agent-runtime/bundle-system.md) for the full specification.

## Agent Card

A JSON file published at `/.well-known/agent-card.json` on every workspace's A2A endpoint. It describes the workspace's identity, skills, capabilities, input/output modes, and authentication requirements.

This is how:
- Workspaces discover each other
- The canvas renders node UI dynamically
- Calling agents know what skills are available

See [Agent Card](../agent-runtime/agent-card.md) for the full specification.
