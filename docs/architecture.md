# Architecture Overview

Molecule AI is a distributed platform for orchestrating AI agent teams. Three components form the core system, connected by HTTP, WebSocket, and JSON-RPC protocols.

## System Components

```
Browser ──HTTP/WS──> Canvas (Next.js :3000)
                        │
                    HTTP + WS
                        │
                    Platform (Go :8080)
                     ┌──┴──┐
                 Postgres  Redis
                     │
                 Docker API
                     │
           ┌─────────┼─────────┐
       Agent-1    Agent-2    Agent-N
      (Python)   (Python)   (Python)
           └──A2A JSON-RPC 2.0──┘
```

### Canvas (Next.js 15 + React Flow)

The browser-based visual UI. Built with `@xyflow/react` v12, Zustand for state, and Tailwind CSS.

- Renders workspaces as draggable nodes on a canvas
- Connects to Platform via REST (`http://localhost:8080`) and WebSocket (`ws://localhost:8080/ws`)
- Sends user messages to agents through the Platform's A2A proxy
- Receives real-time updates via WebSocket events (status changes, agent messages, A2A responses)

Source: `canvas/`

### Platform (Go / Gin)

The control plane. Manages workspace lifecycle, provisions containers, proxies A2A communication, and broadcasts events.

Key responsibilities:
- **Workspace CRUD** -- create, list, update, delete workspaces
- **Container provisioning** -- starts Docker containers for each workspace agent, injects secrets as env vars
- **A2A proxy** -- forwards JSON-RPC requests from canvas to workspace agents, avoiding CORS/Docker network issues
- **Registry** -- agents self-register on startup, send heartbeats, update their AgentCard
- **Discovery** -- workspaces discover peers via hierarchy-based access control rules
- **WebSocket hub** -- broadcasts events to canvas clients (all events) and workspace clients (filtered by access)
- **Secrets management** -- global (`/settings/secrets`) + workspace-level encrypted secrets (AES-256-GCM) with inheritance (workspace overrides global)
- **Liveness monitoring** -- 3-layer health detection: passive (Redis TTL), proactive (Docker health sweep), reactive (A2A proxy check)

Source: `workspace-server/`

### Workspace Runtime (Python)

The execution engine for individual agents. Each workspace runs in its own Docker container.

- Loads config from `/configs/config.yaml`
- Discovers the appropriate adapter (LangGraph, Claude Code, etc.)
- Wraps the agent in an A2A server (using `a2a-sdk`)
- Self-registers with Platform on startup (`POST /registry/register`)
- Sends periodic heartbeats (`POST /registry/heartbeat`)
- Communicates with other workspaces via A2A JSON-RPC 2.0

Source: `workspace/`

## Message Flow

### User sends a message to an agent

```
1. User types in ChatTab
2. Canvas sends POST /workspaces/:id/a2a with JSON-RPC body
3. Platform resolves workspace URL (cache or DB)
4. Platform wraps body in JSON-RPC 2.0 envelope if needed
5. Platform forwards to agent container (5-min timeout for canvas, 30-min for agent-to-agent)
6. Agent processes via LangGraph/adapter, returns JSON-RPC response
7. Platform broadcasts A2A_RESPONSE via WebSocket (canvas-initiated requests only)
8. Platform logs activity asynchronously
9. Canvas receives A2A_RESPONSE event, extracts text, displays in ChatTab
```

### Agent-to-agent delegation

```
1. Agent A calls message/send targeting Agent B
2. Request goes through Platform A2A proxy (POST /workspaces/:id/a2a with X-Workspace-ID header)
3. Platform verifies access via CanCommunicate(callerID, targetID)
4. Platform forwards to Agent B's container (30-min timeout)
5. Agent B responds, Platform returns response to Agent A
6. Activity logged for both workspaces
```

## Core Concepts

### Workspace

The fundamental unit. A workspace represents an organizational **role** (not a task). Each workspace:
- Has a unique UUID, name, role description, and tier (1-4)
- Runs in its own Docker container
- Exposes a single A2A endpoint
- Can be expanded into a sub-team (Team Lead + children)
- Has a lifecycle: `provisioning` -> `online` -> `degraded` -> `offline` -> `removed`

### Agent Card

An A2A protocol discovery document. Each workspace agent publishes an AgentCard containing:
- Name, description, version
- URL endpoint
- Capabilities (streaming, push notifications)
- Skills (id, name, description, tags, examples)
- Supported input/output modes

Updated via `POST /registry/update-card` and broadcast as `AGENT_CARD_UPDATED`.

### A2A Protocol (Agent-to-Agent)

Industry-standard JSON-RPC 2.0 protocol for agent communication:
- `message/send` -- synchronous request/response
- `message/stream` -- SSE streaming variant
- `tasks/get` -- poll async task status

All agent-to-agent traffic flows through the Platform A2A proxy for access control and observability.

### Hierarchy & Access Control

The organizational structure IS the network topology. `CanCommunicate(callerID, targetID)` rules:
- Same workspace: allowed
- Parent <-> child: allowed
- Siblings (same parent_id): allowed
- Root-level workspaces (both parent_id IS NULL): allowed
- Everything else: denied

### Team Expansion (Fractal Architecture)

Any workspace can recursively expand into a sub-team. From the outside, it still exposes a single A2A endpoint. Inside, a Team Lead coordinates child agents.

```
Before:                     After expand:
┌──────────┐               ┌──────────────────────┐
│ Marketing│               │ Marketing (Team Lead)│
│          │   ──expand──> │  ├─ SEO Agent        │
│          │               │  ├─ Content Writer   │
│          │               │  └─ Analytics Agent  │
└──────────┘               └──────────────────────┘
```

- `POST /workspaces/:id/expand` provisions child workspaces from config
- `POST /workspaces/:id/collapse` removes children, reverting to single workspace
- Children are auto-wired: Team Lead ↔ children can communicate, children are siblings
- On the canvas, children render as chips inside the parent node

### Tiered Security

| Tier | Name | Isolation |
|------|------|-----------|
| 1 | Sandboxed | Read-only root FS, tmpfs /tmp, no /workspace mount |
| 2 | Standard | 512 MiB memory, 1.0 CPU limit |
| 3 | Privileged | Privileged mode, host PID, Docker network |
| 4 | Full Access | Privileged, host PID, host network, Docker socket |

## Database (PostgreSQL)

Key tables:

| Table | Purpose |
|-------|---------|
| `workspaces` | Core entity: id, name, role, tier, status, url, parent_id, agent_card (JSONB), heartbeat timestamps |
| `workspace_secrets` | Per-workspace encrypted secrets (AES-256-GCM). UNIQUE(workspace_id, key) |
| `global_secrets` | Platform-wide secrets. Workspace secrets with same key override globals |
| `activity_logs` | A2A communication logs: source, target, method, request/response bodies, duration, status |
| `agent_memories` | Hierarchical Memory Architecture: LOCAL, TEAM, GLOBAL scoped memories |
| `structure_events` | Append-only event log (WORKSPACE_ONLINE, AGENT_CARD_UPDATED, etc.) |
| `workspace_config` | Arbitrary JSONB config per workspace |
| `workspace_memory` | Key-value store with optional TTL per workspace |
| `canvas_layouts` | Node x/y positions on the canvas |

Migrations: `workspace-server/migrations/` (12 files, auto-applied on startup).

## Directory Structure

```
molecule/
├── canvas/                        # Frontend (Next.js 15)
│   └── src/
│       ├── app/                   # Next.js app router pages
│       ├── components/            # React components (tabs/, workspace-node)
│       ├── store/                 # Zustand stores (canvas, socket, events)
│       ├── hooks/                 # Custom React hooks
│       └── lib/                   # Utilities
├── workspace-server/                      # Backend (Go / Gin)
│   ├── cmd/server/main.go        # Entry point
│   ├── cmd/cli/                   # molecli TUI dashboard
│   ├── internal/
│   │   ├── handlers/              # 24 HTTP handler files
│   │   ├── ws/                    # WebSocket hub + client management
│   │   ├── events/                # Broadcaster (WS + Redis pub/sub)
│   │   ├── db/                    # PostgreSQL + Redis connections
│   │   ├── provisioner/           # Docker container lifecycle
│   │   ├── registry/              # Liveness, health sweep, access rules
│   │   ├── crypto/                # AES-256-GCM encryption
│   │   └── models/                # Data types
│   └── migrations/                # 12 SQL migration files
├── workspace/            # Agent Runtime (Python)
│   ├── main.py                    # Entry point
│   ├── a2a_executor.py            # A2A request handler
│   ├── config.py                  # YAML config loader
│   ├── heartbeat.py               # Platform heartbeat loop
│   ├── adapters/                  # Runtime backends (langgraph, claude-code, ...)
│   └── tools/                     # Agent tools (delegation, sandbox, ...)
├── docker-compose.yml             # Full stack
└── docker-compose.infra.yml       # Infrastructure only (dev)
```

## Supporting Infrastructure

| Service | Image | Purpose |
|---------|-------|---------|
| PostgreSQL 16 | `postgres:16-alpine` | Primary database |
| Redis 7 | `redis:7-alpine` | URL caching, pub/sub, TTL-based liveness |
| Langfuse | `langfuse/langfuse:2` + ClickHouse | LLM call tracing and observability |
| LiteLLM (optional) | `ghcr.io/berriai/litellm` | Unified multi-provider LLM routing |
| Ollama (optional) | `ollama/ollama` | Local model inference |
