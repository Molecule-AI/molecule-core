# Memory Architecture (HMA)

Molecule AI's memory model is built around one principle:

> memory boundaries should follow organizational boundaries.

That is the purpose of **HMA: Hierarchical Memory Architecture**.

## The Three Scopes

| Scope | Meaning | Intended use |
|---|---|---|
| `LOCAL` | visible only to the current workspace | private scratch facts and local recall |
| `TEAM` | visible to the local team boundary | handoffs between a parent and its direct children, or siblings under the same parent |
| `GLOBAL` | readable across the tree; writable only from the root side | org-wide guidance, standards, shared institutional knowledge |

These are the scopes exposed through the runtime memory tools:

- `commit_memory(content, scope)`
- `search_memory(query, scope)`

## What Exists In The Current Implementation

There are **multiple memory surfaces**, and the distinction matters.

### 1. Scoped agent memory (`agent_memories`)

This is the HMA-facing storage used by:

- `POST /workspaces/:id/memories`
- `GET /workspaces/:id/memories`
- runtime tools `commit_memory` / `search_memory`

It stores durable facts with a `LOCAL`, `TEAM`, or `GLOBAL` scope.

### 2. Workspace key/value memory (`workspace_memory`)

This is the simpler key/value surface used by the canvas `Memory` tab:

- `GET /workspaces/:id/memory`
- `POST /workspaces/:id/memory`
- `DELETE /workspaces/:id/memory/:key`

It is useful for structured per-workspace state and optional TTL entries. It is not the same thing as scoped HMA memories.

### 3. Activity recall (`session-search`)

`GET /workspaces/:id/session-search` provides a thin recall surface over recent activity rows and memory rows. It is for “what just happened in this workspace?” rather than long-term semantic storage.

### 4. Awareness-backed persistence

When the runtime receives:

```bash
AWARENESS_URL=...
AWARENESS_NAMESPACE=workspace:<id>
```

the same memory tools keep the same interface, but durable memory writes/reads are routed through the workspace's awareness namespace.

This is the current production direction of the memory boundary: stable tool surface, stronger backend isolation.

## Access Model

Molecule AI's memory rules follow the same hierarchy logic as communication rules:

- `LOCAL` belongs to one workspace
- `TEAM` follows the immediate team boundary
- `GLOBAL` is readable widely but writable only from the root side

The platform-side memory handlers still apply reachability checks for shared/team reads instead of trusting callers blindly.

## Current Schema Reality

The current `agent_memories` migration is intentionally simple:

```sql
CREATE TABLE IF NOT EXISTS agent_memories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id),
    content      TEXT NOT NULL,
    scope        VARCHAR(10) NOT NULL CHECK (scope IN ('LOCAL', 'TEAM', 'GLOBAL')),
    created_at   TIMESTAMPTZ DEFAULT now(),
    updated_at   TIMESTAMPTZ DEFAULT now()
);
```

`pgvector` is **not enabled by default in the shipped migration**. The repo keeps vector support as an optional future extension, not as a current hard dependency. The docs should reflect that explicitly.

## Why This Architecture Matters

| Flat shared memory model | Molecule AI HMA |
|---|---|
| easy to over-share | scopes align to hierarchy |
| unclear ownership | each memory belongs to a workspace and a scope |
| recall and procedure blur together | memory stores facts, skills store repeatable procedure |
| hard to govern | org structure and memory rules reinforce each other |

## Memory To Skill Promotion

Molecule AI intentionally separates:

- **durable fact storage**
- **repeatable operational procedure**

The documented promotion path is:

1. a durable workflow is captured in memory
2. repeated success becomes a signal
3. the workflow is promoted into a skill package
4. the runtime hot-reloads that skill

This is why memory and skills are presented as adjacent systems, not one merged blob.

## Practical Summary

If you need:

- **private agent recall**: use `LOCAL`
- **shared team handoff knowledge**: use `TEAM`
- **org-wide guidance**: use `GLOBAL`
- **simple UI-visible structured state**: use `workspace_memory`
- **recent decision/task recall**: use `session-search`
- **stronger durable isolation**: enable awareness namespaces

## Related Docs

- [Workspace Runtime](../agent-runtime/workspace-runtime.md)
- [Skills](../agent-runtime/skills.md)
- [Communication Rules](../api-protocol/communication-rules.md)
- [Platform API](../api-protocol/platform-api.md)
