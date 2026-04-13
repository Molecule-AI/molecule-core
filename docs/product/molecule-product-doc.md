# Molecule AI — The Org-Native Control Plane for Heterogeneous AI Agent Teams

> **Build AI organizations, not fragile agent demos.**

---

## What Is Molecule AI

Molecule AI is the missing operational and organizational layer above agent runtimes. It is not a workflow builder, not a replacement for LangGraph or CrewAI, and not a chat UI — it is the **control plane** that lets enterprises run heterogeneous AI agent teams as governed, observable, scalable organizations.

A workspace is a role. The org chart is the topology. Memory follows hierarchy. Six runtime adapters run side-by-side. Molecule AI is how you govern AI teams in production.

---

## Core Philosophy

### Five Foundational Principles

**1. The Node Is a Role, Not a Task**

Every workspace represents a durable organizational role (DevOps Lead, Security Reviewer, Research Analyst). The AI model inside can be swapped — from GPT-4o to Claude Sonnet to a local Ollama model — without changing the role's position, hierarchy, identity, or memory. Roles survive model swaps, framework changes, and team restructuring.

**2. The Org Chart Is the Topology**

Organization structure directly encodes communication rules. Parent ↔ child, sibling ↔ sibling — allowed. Cross-team — denied. No manual edge wiring, no drift between design-time topology and runtime behavior. The `CanCommunicate()` function is the single source of truth for all access control: A2A delegation, memory scope enforcement, approval routing, and activity visibility.

**3. Runtime Choice Is Not a Dead-End Decision**

Six adapters ship on main: LangGraph, DeepAgents, Claude Code, CrewAI, AutoGen, OpenClaw. Different teams keep their preferred runtimes while sharing one unified governance layer. The platform handles registration, discovery, and governance. All AI logic lives in workspace adapters. Adding a new runtime is a bounded integration task.

**4. Memory Is Infrastructure**

Hierarchical Memory Architecture (HMA) is not a feature bolted on top. It is the foundation that makes team expansion, skill compounding, and organizational learning possible at scale. Three scopes — LOCAL, TEAM, GLOBAL — ensure memory sharing follows org boundaries exactly.

**5. The System Forms a Self-Improving Flywheel**

Task execution → durable insights in memory → repeated success signals → promotion to reusable skill → hot-reload into runtime → future work faster and more reliable. The organization becomes more capable without hidden prompt inflation.

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Canvas (Next.js 15 · React Flow · Zustand · WebSocket)     │
│ Visual drag-to-nest org chart · 10-tab operations panel     │
└──────────────────┬──────────────────────────────────────────┘
                   │ HTTP + WebSocket
┌──────────────────▼──────────────────────────────────────────┐
│ Platform (Go / Gin · port 8080)                             │
│ Control plane: workspace CRUD, registry, A2A proxy,         │
│ memory, secrets, approvals, activity, health, bundles       │
└─────────┬────────────────────────────────┬──────────────────┘
          │                                │
    Postgres 16                         Redis 7
    (source of truth)                   (ephemeral state, pub/sub)

┌─────────────────────────────────────────────────────────────┐
│ Workspace Runtime (Python 3.11+ · Docker)                   │
│ 6 pluggable adapters · A2A server · heartbeat · skills      │
│ HMA memory tools · approval gates · audit logging           │
└─────────────────────────────────────────────────────────────┘
          │
┌─────────────────────────────────────────────────────────────┐
│ Langfuse (self-hosted · OpenTelemetry · ClickHouse)         │
│ Every LLM call traced end-to-end                            │
└─────────────────────────────────────────────────────────────┘
```

### Network Model

- **Canvas ↔ Platform**: HTTP REST + WebSocket (real-time events)
- **Platform ↔ Database**: Postgres (durable state), Redis (ephemeral + pub/sub)
- **Workspace ↔ Workspace**: Direct A2A (JSON-RPC 2.0, peer-to-peer, platform not in path)
- **Workspace → Langfuse**: Automatic OpenTelemetry tracing
- **Docker Network**: All services on `molecule-monorepo-net` (internal-only by default)

---

## The Six Runtime Adapters

All adapters implement the `BaseAdapter` interface and ship production-ready on `main`.

| Adapter | Core Strength | Molecule AI Integration |
|---------|--------------|---------------------|
| **LangGraph** | Graph-based state machine, tool use, streaming | Default adapter. A2A executor wraps LangGraph for inter-workspace communication |
| **DeepAgents** | Deep planning, multi-step task decomposition | Planning layer feeds into HMA for persistent plan state |
| **Claude Code** | Native coding workflows, CLI continuity | OAuth token auth, workspace abstraction preserves session model |
| **CrewAI** | Role-based crews, structured task orchestration | Persistent workspace identity + shared Canvas visualization |
| **AutoGen** | Multi-agent conversations, explicit strategies | AssistantAgent mapping with Molecule AI governance overlay |
| **OpenClaw** | CLI-native runtime, own session model | Template-aware skill injection + workspace lifecycle |

**Branch-level WIP**: NemoClaw (NVIDIA T4 support) on `feat/nemoclaw-t4-docker`.

### Adapter Architecture

Each adapter implements two methods:
- `setup()` — Initialize runtime-specific dependencies, load plugins and skills
- `create_executor()` — Build the agent executor that processes A2A messages

The base adapter provides shared infrastructure: system prompt assembly, skill loading, tool registration, coordinator detection, and plugin injection. This means new adapters only need to implement runtime-specific logic.

---

## Hierarchical Memory Architecture (HMA)

### The Three Scopes

| Scope | Visibility | Write Access | Use Case |
|-------|-----------|-------------|----------|
| **LOCAL** | This workspace only | Self | Private reasoning, temporary findings, working state |
| **TEAM** | Parent + children + siblings | Self | Handoff context, team coordination, shared decisions |
| **GLOBAL** | All workspaces | Root only | Org-wide policies, standards, institutional knowledge |

### Memory Surfaces

1. **Scoped Agent Memory** (`agent_memories` table) — HMA-backed distributed memory. Used by `commit_memory()` / `search_memory()` tools with scope enforcement.

2. **Workspace Key/Value Memory** (`workspace_memory` table) — Simple structured state visible in Canvas Memory tab. Optional TTL support.

3. **Activity Recall** (`session-search` endpoint) — Search recent activity logs and memory rows. Powers "what just happened?" contextual recall.

4. **Awareness-Backed Persistence** — When `AWARENESS_URL` + `AWARENESS_NAMESPACE` are configured, memory tools route to workspace-scoped namespaces in an external persistence backend. Same API, stronger isolation.

### Memory → Skill Compounding Loop

```
Task execution
  → Durable insight captured in LOCAL/TEAM memory
  → Repeated success patterns detected
  → Memory promoted to SKILL.md package
  → Hot-reload (~3 seconds) into live runtime
  → Agent Card updated, broadcast to peers
  → Future tasks use promoted skill
  → Organization becomes more capable over time
```

This is not hidden prompt inflation. Promotion events are visible in activity logs. Skills are inspectable in the Canvas Skills tab. The effect is organization-wide, not buried in context windows.

---

## Workspace Lifecycle

```
provisioning → online ↔ degraded
                 ↓         ↓
              offline    offline
                 ↓
           (auto-restart)

paused → (user resumes) → provisioning

removed (terminal)
```

| Status | Meaning | Canvas Indicator |
|--------|---------|-----------------|
| `provisioning` | Waiting for first heartbeat | Spinner |
| `online` | Heartbeat active, reachable | Green dot |
| `degraded` | Online but error_rate ≥ 50% | Yellow warning |
| `offline` | Heartbeat expired, unreachable | Gray node |
| `paused` | User paused, container stopped, config preserved | Indigo badge |
| `failed` | Launch error or provisioning timeout | Red node + retry button |
| `removed` | Deleted, kept for audit trail | Node removed from Canvas |

### Health Detection (Three Layers)

1. **Passive (Redis TTL)**: Heartbeat key expires after 60s → offline detection
2. **Proactive (Health Sweep)**: Docker API poll every 15s → catch dead containers
3. **Reactive (A2A Proxy)**: Connection error on message send → immediate check via `provisioner.IsRunning()`

All three layers trigger `onWorkspaceOffline()` → broadcast `WORKSPACE_OFFLINE` + auto-restart.

### Cascade Behavior

- **Pause**: Pausing a parent cascades to all children. Children of a paused parent cannot be individually resumed.
- **Delete**: Removes container, cleans memory (DB rows, Redis keys). Structure events and Agent Card history are never deleted.

---

## Runtime Tier System (T1–T4)

| Tier | Name | Container Config | Use Case |
|------|------|-----------------|----------|
| **T1** | Sandboxed | Read-only rootfs, tmpfs /tmp, 512 MiB, no /workspace mount | Untrusted code, text-only analysis |
| **T2** | Standard (default) | Read-write, 512 MiB, 1 CPU, /workspace mount | Most agent workloads |
| **T3** | Privileged | `--privileged`, `--pid=host`, Docker network access | Internal tooling, elevated operations |
| **T4** | Full Access | T3 + `--network=host` + Docker socket mount | System-level orchestration, DevOps |

Unknown tier values default to T2 for safety. The provisioner applies tier configuration via `ApplyTierConfig()` during container creation.

---

## A2A Communication Protocol

### Message Format (JSON-RPC 2.0 over HTTP)

```json
{
  "jsonrpc": "2.0",
  "id": "task-123",
  "method": "message/send",
  "params": {
    "message": {
      "role": "user",
      "parts": [{"kind": "text", "text": "Build the login feature"}],
      "messageId": "msg-456"
    }
  }
}
```

### Communication Rules

| Direction | Allowed | Rationale |
|-----------|---------|-----------|
| Sibling ↔ Sibling | YES | Peer collaboration within same team |
| Parent → Child | YES | Task delegation downward |
| Child → Parent | YES | Reporting and escalation upward |
| Skip levels | NO | Must route through hierarchy |
| Cross-team | NO | Organizational boundary enforcement |

### Discovery Flow

1. Caller queries `GET /registry/discover/:targetId` with `X-Workspace-ID` header
2. Platform validates `CanCommunicate(caller, target)` — returns 403 if denied
3. Returns Docker-internal URL (workspace caller) or host-mapped URL (Canvas caller)
4. Caller sends A2A message **directly** to target (peer-to-peer, platform not in the data path)
5. Target processes task, returns response

### Task States

```
submitted → working → completed
         → failed
         → canceled
         → input-required → working (after caller provides input)
```

### Streaming Support

Two call modes:
- `message/send` — Synchronous for short tasks
- `message/sendSubscribe` — SSE streaming for long-running tasks with progress updates

---

## Canvas UI

### Design Philosophy

No task nodes. No manual edge connecting. The Canvas is a visual org chart where hierarchy is built through drag-and-drop.

### Core Interactions

- **Drag-to-Nest**: Drag one workspace node over another → overlap detection → highlight → drop → update `parent_id`
- **Right-Click Menu**: Open Details/Chat/Terminal, Restart, Duplicate, Export Bundle, Expand/Collapse Team, Extract from Team, Delete
- **Template Palette**: Empty state shows up to 6 workspace templates + "Create blank workspace"
- **Onboarding Wizard**: 4-step guided setup on first use (create → configure → add secrets → chat)

### 10-Tab Operations Panel

Every selected workspace exposes a side panel with:

| Tab | Function |
|-----|----------|
| **Chat** | A2A conversational interface with session history |
| **Activity** | Rich operation log (A2A messages, task updates, agent logs, skill promotions) |
| **Details** | Workspace metadata, runtime summary, status, Agent Card, restart/pause controls, peer list |
| **Skills** | Live skill display from Agent Card — shows loaded skills with metadata, tags, examples |
| **Terminal** | WebSocket shell into workspace container |
| **Config** | Structured YAML editor for runtime, skills, tools, A2A, delegation, sandbox settings |
| **Files** | File browser and editor for /configs, /workspace, /home, /plugins |
| **Memory** | Scoped memory view (LOCAL/TEAM/GLOBAL) + key/value workspace memory with TTL |
| **Traces** | Langfuse trace viewer — every LLM call with input/output/tokens/cost |
| **Events** | Structure event stream — real-time workspace change log |

### Real-Time Architecture

- **Initial Load**: `GET /workspaces` → Zustand store hydration
- **Live Updates**: WebSocket events → `applyEvent()` → instant Canvas re-render
- **Persistence**: `onNodeDragStop` → `PATCH /workspaces/:id` with x, y coordinates
- **Error Recovery**: Error boundary with reload button + hydration retry banner

---

## Skills System

### Three Capability Sources

1. **Workspace-Local Skills** — `skills/<skill-name>/SKILL.md` + `tools/` directory
2. **Plugin-Mounted Rules** — `/plugins` volume (read-only), shared across all workspaces
3. **Built-In Tools** — Delegation, approval, memory, sandbox, telemetry, audit

### Skill Format

```
skills/generate-seo-page/
├── SKILL.md              # YAML frontmatter + instructions
├── tools/
│   ├── write_page.py     # @tool-decorated functions
│   └── check_gsc.py
├── examples/             # Few-shot examples
├── templates/            # Reference files
└── links.yaml            # External resources
```

### Hot-Reload Pipeline

1. File watcher monitors `skills/` directory with 2-second debounce
2. On change: reload skill metadata + tool Python modules
3. Rebuild agent tools and update Agent Card
4. Broadcast updated card via WebSocket to all peers
5. Peer system prompts automatically rebuilt with new capability awareness
6. Total propagation time: ~3 seconds

---

## Coordinator Pattern (Team Expansion)

When a workspace "expands into a team," it becomes a coordinator:

1. Parent workspace becomes **coordinator** (team lead role)
2. Fetches children's Agent Cards to understand their capabilities
3. For each incoming task: analyzes, selects best-suited child, delegates via A2A
4. Aggregates responses when tasks need multiple children
5. Falls back to self-handling only if no child is suitable

**Enforcement**: Coordinators cannot do direct work themselves. All actual execution is delegated to children. This prevents team leads from becoming bottlenecks.

**Recursive Expansion**: A child workspace can itself become a team, creating nested hierarchies of arbitrary depth. Upstream integrations remain intact — the parent doesn't need to know whether its child is a single agent or a team of fifty.

---

## Bundle System (Portable Workspace Export)

### Included
- Complete system prompt text
- All skill files (inlined as strings in JSON)
- Prompt templates and asset files
- Tool configurations
- Sub-workspace bundles (recursive)
- Agent Card snapshot
- Author, version, tier metadata

### Excluded
- API keys or secrets (buyer brings own)
- Memory or conversation history
- Database state

### Workflow
**Export**: Right-click workspace → "Export as bundle" → downloads `.bundle.json`
**Import**: Drag `.bundle.json` onto Canvas → recursive provisioning → new workspace IDs → `source_bundle_id` traces lineage

---

## Governance & Enterprise Control

### Hierarchical Approval Chain
Agent triggers `request_approval()` → escalation follows org hierarchy → each node approves/denies/escalates → root exposes to human via Canvas → decision flows back down → all decisions logged.

### RBAC Roles
Configurable per-workspace: `operator`, `admin`, `read-only`, `no-delegation`, `no-approval`. Custom action mappings supported.

### Secrets Management
Global secrets (AES-256-GCM encrypted) with per-workspace overrides. Secret changes trigger automatic restart. Bundles never include secrets.

### Compliance
```yaml
compliance:
  mode: owasp_agentic
  prompt_injection: detect | block
  max_tool_calls_per_task: 50
  max_task_duration_seconds: 300
```

### Audit Trail
- **Activity Logs**: A2A messages, task updates, skill promotions (7-day retention)
- **Structure Events**: Append-only, never UPDATE/DELETE (complete org history)
- **Langfuse Traces**: Every LLM call with full context
- **Audit File**: JSON Lines at configurable path

---

## Platform API Reference (40+ Endpoints)

### Workspace Lifecycle
`POST /workspaces` · `GET /workspaces` · `GET /workspaces/:id` · `PATCH /workspaces/:id` · `DELETE /workspaces/:id` · `POST /workspaces/:id/restart` · `POST /workspaces/:id/pause` · `POST /workspaces/:id/resume`

### Registry & Discovery
`POST /registry/register` · `POST /registry/heartbeat` · `POST /registry/update-card` · `GET /registry/discover/:id` · `GET /registry/:id/peers`

### Memory
`POST /workspaces/:id/memories` · `GET /workspaces/:id/memories` · `DELETE /workspaces/:id/memories/:memoryId` · `GET /workspaces/:id/memory` · `POST /workspaces/:id/memory` · `DELETE /workspaces/:id/memory/:key`

### Secrets · Activity · Approvals · Files · Terminal · Bundles · Templates · Events · Observability · WebSocket
See full documentation at [docs/api-protocol](https://github.com/Molecule-AI/molecule-monorepo/tree/main/docs/api-protocol).

---

## MCP Server Integration

20+ tools exposed via Model Context Protocol for Claude Code, Cursor, Codex integration. Workspace CRUD, agent communication, memory operations, team management, secrets, files, approvals — all accessible from any MCP client.

---

## Test Coverage

| Layer | Tests | Framework |
|-------|-------|-----------|
| Go Platform | 278 | `go test -race` (25% baseline enforced) |
| Canvas Frontend | 188 | Vitest + OXC JSX |
| Python Runtime | 148 | pytest + pytest-cov |
| API Integration | 62 | Shell scripts |
| A2A E2E | 22 | Requires 2 online agents |
| Comprehensive E2E | 68 | All endpoints + memory + approvals |

---

## Vision: From Agent Teams to Robot Teams

Molecule AI's workspace abstraction is runtime-agnostic by design. A workspace is a role with an A2A interface — not an LLM with a prompt.

| Phase | Systems | Status |
|-------|---------|--------|
| **NOW** | LLM agents in Docker, 6 adapters, HMA, Langfuse | LIVE |
| **NEXT** | Terminal bots, browser agents, IoT controllers | BUILDING |
| **HORIZON** | Warehouse robots, autonomous vehicles, manufacturing cells | HORIZON |

The workspace is the role. The protocol is A2A. The boundary between digital and physical disappears — the organizational layer remains.

---

## Quick Start

```bash
git clone https://github.com/Molecule-AI/molecule-monorepo.git
cd molecule-monorepo
docker compose up -d
open http://localhost:3000
```

---

## Links

- **GitHub**: https://github.com/Molecule-AI/molecule-monorepo
- **Architecture**: https://github.com/Molecule-AI/molecule-monorepo/tree/main/docs/architecture
- **API Protocol**: https://github.com/Molecule-AI/molecule-monorepo/tree/main/docs/api-protocol
- **Agent Runtime**: https://github.com/Molecule-AI/molecule-monorepo/tree/main/docs/agent-runtime

---

*© 2026 Molecule AI Technologies, Inc.*
