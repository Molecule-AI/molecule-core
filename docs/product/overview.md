# Overview

## What Molecule AI Is

Molecule AI is an **org-native orchestration platform for AI agent workspaces**.

The shortest accurate description is:

> A visual org chart plus a control plane for heterogeneous agent teams.

Instead of modeling a system as edges between tasks, Molecule AI models it as **roles inside a hierarchy**. A workspace can be one agent now, a sub-team later, and still keep the same external identity, policy boundary, memory boundary, and position on the canvas.

## What Problem It Solves

Most agent projects are strong at one of these layers, but weak across all of them together:

- runtime flexibility
- topology management
- memory isolation
- operational control
- observability
- reusable skill lifecycle

Molecule AI is the layer that ties those together.

## What Makes It Different

| Dimension | Typical agent tool | Molecule AI |
|---|---|---|
| Primary abstraction | task, chain, graph node | workspace role |
| Topology | manual edges or hard-coded routing | org chart hierarchy |
| Runtime choice | usually one framework | multiple frameworks behind one workspace contract |
| Memory model | flat or loosely namespaced | hierarchy-aware scope + awareness namespace |
| Team growth | rebuild the graph | expand a workspace into a sub-team |
| Ops | mostly left to custom glue | built-in registry, heartbeats, traces, approvals, activity, restart |

## Runtime Compatibility

Current `main` ships adapters for:

- LangGraph
- DeepAgents
- Claude Code
- CrewAI
- AutoGen
- OpenClaw

Branch-level runtime work such as NemoClaw exists separately and should be described as WIP, not merged `main` support.

## Memory And Skills

Molecule AI treats durable memory and reusable procedure as different system layers:

- **memory** stores facts worth recalling later
- **session-search** recovers recent activity and memory rows
- **skills** store repeatable procedures
- **promotion** is the bridge: repeated durable workflows can be elevated from memory into a hot-reloadable skill package

This separation is one of the reasons Molecule AI scales better than “just add another memory store” designs.

## What Molecule AI Is Not

- Not a replacement for LangGraph, CrewAI, AutoGen, Claude Code, or OpenClaw
- Not a visual workflow automation builder where nodes are one-off tasks
- Not just a chat UI over one agent
- Not a model provider
- Not a hosted SaaS-only black box; this repository is the open-source core

## Related Docs

- [Product Narrative](./molecule-product-doc.md)
- [Landing Messaging Report](./landing-messaging-report.md)
- [Quickstart](../quickstart.md)
- [System Architecture](../architecture/architecture.md)
- [Comprehensive Technical Documentation](../architecture/molecule-technical-doc.md)
- [Memory Architecture](../architecture/memory.md)
- [Workspace Runtime](../agent-runtime/workspace-runtime.md)
- [Canvas UI](../frontend/canvas.md)
