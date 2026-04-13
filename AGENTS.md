# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Project Overview

Molecule AI is a platform for orchestrating AI agent workspaces that form an organizational hierarchy. Workspaces register with a central platform, communicate via A2A protocol, and are visualized on a drag-and-drop canvas.

## Architecture

```
Canvas (Next.js :3000) ←WebSocket→ Platform (Go :8080) ←HTTP→ Postgres + Redis
                                                                  ↑
                                   Workspace A ←──A2A──→ Workspace B
                                   (pluggable runtimes)
                                        ↑ register/heartbeat ↑
                                        └───── Platform ─────┘
```

Three main components:
- **Platform** (`platform/`): Go/Gin control plane — workspace CRUD, registry, discovery, WebSocket hub, liveness monitoring
- **Canvas** (`canvas/`): Next.js 15 + React Flow (@xyflow/react v12) + Zustand + Tailwind — visual workspace graph
- **Workspace Runtime** (`workspace-template/`): A2A runtime layer with pluggable adapters — LangGraph, DeepAgents, Claude Code, CrewAI, AutoGen, OpenClaw — registers with platform and sends heartbeats

## Build & Run Commands

### Infrastructure
```bash
./infra/scripts/setup.sh    # Start Postgres, Redis, Langfuse; run migrations
./infra/scripts/nuke.sh     # Tear down everything, remove volumes
```

### Platform (Go)
```bash
cd platform
go build ./cmd/server       # Build
go run ./cmd/server          # Run (requires Postgres + Redis running)
```
Must run from `platform/` directory (not repo root). Env vars: `DATABASE_URL`, `REDIS_URL`, `PORT` (defaults: postgres://dev:dev@localhost:5432/molecule?sslmode=prefer, redis://localhost:6379, 8080).

### Canvas (Next.js)
```bash
cd canvas
npm install
npm run dev                  # Dev server on :3000
npm run build && npm start   # Production
```
Env vars: `NEXT_PUBLIC_PLATFORM_URL` (default http://localhost:8080), `NEXT_PUBLIC_WS_URL` (default ws://localhost:8080/ws).

### Integration Tests
```bash
bash test_api.sh             # Runs 34 API tests against localhost:8080
```
Requires platform running. Tests full CRUD, registry, heartbeat, discovery, peers, access control, events, degraded/recovery lifecycle.

### Docker Compose
```bash
docker compose -f docker-compose.infra.yml up -d    # Infra only
docker compose up                                     # Full stack
```

## Key Architectural Patterns

### Import Cycle Prevention
The platform uses function injection to avoid Go import cycles between ws, registry, and events packages:
- `ws.NewHub(canCommunicate AccessChecker)` — Hub accepts `registry.CanCommunicate` as a function
- `registry.StartLivenessMonitor(ctx, onOffline OfflineHandler)` — Liveness accepts broadcaster callback
- Wiring happens in `platform/cmd/server/main.go`

### Communication Rules (`registry/access.go`)
`CanCommunicate(callerID, targetID)` determines if two workspaces can talk:
- Same workspace → allowed
- Siblings (same parent_id) → allowed
- Root-level siblings (both parent_id IS NULL) → allowed
- Parent ↔ child → allowed
- Everything else → denied

### JSONB Gotcha
When inserting Go `[]byte` (from `json.Marshal`) into Postgres JSONB columns, you must:
1. Convert to `string()` first
2. Use `::jsonb` cast in SQL

lib/pq treats `[]byte` as `bytea`, not JSONB.

### WebSocket Events Flow
1. Action occurs (register, heartbeat, etc.)
2. `broadcaster.RecordAndBroadcast()` inserts into `structure_events` table + publishes to Redis pub/sub
3. Redis subscriber relays to WebSocket hub
4. Hub broadcasts to canvas clients (all events) and workspace clients (filtered by CanCommunicate)

### Canvas State Management
- Initial load: HTTP fetch from `GET /workspaces` → Zustand hydrate
- Real-time updates: WebSocket events → `applyEvent()` in Zustand store
- Position persistence: `onNodeDragStop` → `PATCH /workspaces/:id` with `{x, y}`

### Workspace Lifecycle
`provisioning` → `online` (on register) → `degraded` (error_rate > 0.5) → `online` (recovered) → `offline` (Redis TTL expired) → `removed` (deleted)

## Platform API Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /health | inline |
| POST/GET/PATCH/DELETE | /workspaces[/:id] | workspace.go |
| POST | /registry/register | registry.go |
| POST | /registry/heartbeat | registry.go |
| POST | /registry/update-card | registry.go |
| GET | /registry/discover/:id | discovery.go |
| GET | /registry/:id/peers | discovery.go |
| POST | /registry/check-access | discovery.go |
| GET | /events[/:workspaceId] | events.go |
| GET | /ws | socket.go |

## Database

5 migration files in `platform/migrations/`. Key tables: `workspaces` (core entity with status, agent_card JSONB, heartbeat columns), `canvas_layouts` (x/y position), `structure_events` (append-only event log), `agents`, `workspace_secrets`.

The platform auto-discovers and runs migrations on startup from several candidate paths.

<!-- AWARENESS_RULES_START -->
# Awareness Memory Integration

MANDATORY agent policy - follow for every task.

## Awareness Memory Integration (MANDATORY)

awareness_* = cross-session persistent memory (past decisions, knowledge, tasks).
Other tools = current codebase navigation (file search, code index).
Use BOTH - they serve different purposes.

STEP 1 - SESSION START:
  Call awareness_init(source="codex") -> get session_id, review context.
  If active_skills[] is returned: skill = reusable procedure done 2+ times;
  summary = injectable instruction, methods = steps. Apply matching skills to tasks.

STEP 2 - RECALL BEFORE WORK (progressive disclosure):
  1. awareness_recall(semantic_query=..., keyword_query=..., detail='summary') → lightweight index.
  2. Review summaries/scores, pick relevant IDs.
  3. awareness_recall(detail='full', ids=[...]) → expand only what you need.

STEP 3 - RECORD EVERY CHANGE:
  After EVERY code edit, decision, or bug fix:
  awareness_record(content=<detailed natural language description>,
    insights={knowledge_cards:[...], action_items:[...], risks:[...]})
  Content should be RICH and DETAILED — include reasoning, key code snippets,
  user quotes, alternatives considered, and files changed. Do NOT compress into
  a single-line summary. The content IS the memory — more detail = better recall.
  Include insights to create searchable knowledge in ONE step (recommended).
  Skipping = permanent data loss.

STEP 4 - CATEGORY GUIDE (for insights.knowledge_cards):
  - decision = choice made between alternatives.
  - problem_solution = bug/problem plus the fix that resolved it.
  - workflow = process, setup, or configuration steps only.
  - pitfall = blocker, warning, or limitation without a fix yet.
  - insight = reusable pattern or general learning.
  - skill = reusable procedure done 2+ times; summary = injectable instruction, methods = steps.
  - key_point = important technical fact when nothing else fits.
  Never default everything to workflow.

STEP 5 - SESSION END:
  awareness_record(content=[step1, step2, ...], insights={...}) with final summary.

BACKFILL (if applicable):
  If MCP connected late: awareness_record(content=<transcript>)

RULES VERSION: Pass rules_version="2" to awareness_init so the server knows you have these rules.
If the server returns _setup_action, the rules have been updated — follow the instruction to re-sync.

NOTE: memory_id from X-Awareness-Memory-Id header. source/actor/event_type auto-inferred.

## Codex-Specific Notes

- Call awareness_init at task start before reading any files.

- After each code patch, call awareness_record with the change description.
<!-- AWARENESS_RULES_END -->
