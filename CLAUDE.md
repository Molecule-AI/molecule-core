# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Molecule AI is a platform for orchestrating AI agent workspaces that form an organizational hierarchy. Workspaces register with a central platform, communicate via A2A protocol, and are visualized on a drag-and-drop canvas.

## Ecosystem Context

Before research, strategy, or design work, skim **`docs/ecosystem-watch.md`** —
it catalogs adjacent agent projects (Holaboss, Hermes, gstack, …) with
overlap / differentiation / terminology-collision notes. Cross-referenced
from `PLAN.md` and `README.md`; it's the canonical starting point for
"what else is out there."

## Architecture

```
Canvas (Next.js :3000) ←WebSocket→ Platform (Go :8080) ←HTTP→ Postgres + Redis
                                                                  ↑
                                   Workspace A ←──A2A──→ Workspace B
                                   (Python agents)
                                        ↑ register/heartbeat ↑
                                        └───── Platform ─────┘
```

Four main components:
- **Platform** (`platform/`): Go/Gin control plane — workspace CRUD, registry, discovery, WebSocket hub, liveness monitoring
- **Canvas** (`canvas/`): Next.js 15 + React Flow (@xyflow/react v12) + Zustand + Tailwind — visual workspace graph
- **Workspace Runtime** (`workspace-template/`): Unified Docker image with pluggable adapter system — supports LangGraph, Claude Code, OpenClaw, DeepAgents, CrewAI, AutoGen. Adapters in `workspace-template/adapters/`. Deps installed at startup via `entrypoint.sh`.
- **molecli** (`platform/cmd/cli/`): Go TUI dashboard (Bubbletea + Lipgloss) — real-time workspace monitoring, event log, health overview, delete/filter operations

## Build & Run Commands

### Infrastructure
```bash
./infra/scripts/setup.sh    # Start Postgres, Redis, Langfuse; run migrations
./infra/scripts/nuke.sh     # Tear down everything, remove volumes
```

### Platform (Go)
```bash
cd platform
go build ./cmd/server       # Build server
go run ./cmd/server          # Run server (requires Postgres + Redis running)
go build -o molecli ./cmd/cli  # Build TUI dashboard
./molecli                    # Run TUI dashboard (requires platform running)
```
Must run from `platform/` directory (not repo root). Env vars: `DATABASE_URL`, `REDIS_URL`, `PORT`, `PLATFORM_URL` (default `http://host.docker.internal:PORT` — passed to agent containers so they can reach the platform), `SECRETS_ENCRYPTION_KEY` (optional AES-256, 32 bytes), `CONFIGS_DIR` (auto-discovered), `PLUGINS_DIR` (deprecated — plugins are now installed per-workspace via API; the `plugins/` registry at repo root is auto-discovered), `ACTIVITY_RETENTION_DAYS` (default `7`), `ACTIVITY_CLEANUP_INTERVAL_HOURS` (default `6`), `CORS_ORIGINS` (comma-separated, default `http://localhost:3000,http://localhost:3001`), `RATE_LIMIT` (requests/min, default `600`), `WORKSPACE_DIR` (optional — global fallback host path for `/workspace` bind-mount; overridden by per-workspace `workspace_dir` column in DB; if neither is set, each workspace gets an isolated Docker named volume), `AWARENESS_URL` (optional — if set, injected into workspace containers along with a deterministic `AWARENESS_NAMESPACE` derived from workspace ID), `MOLECULE_IN_DOCKER` (optional — set to `1` when the platform itself runs inside Docker so the A2A proxy rewrites `127.0.0.1:<port>` URLs to container hostnames; auto-detected via `/.dockerenv`).

**Plugin install safeguards** (bound the cost of a single `POST /workspaces/:id/plugins` install so a slow/malicious source can't tie up a handler):
- `PLUGIN_INSTALL_BODY_MAX_BYTES` — max request body size (default `65536` = 64 KiB)
- `PLUGIN_INSTALL_FETCH_TIMEOUT` — duration string; whole fetch+copy deadline (default `5m`)
- `PLUGIN_INSTALL_MAX_DIR_BYTES` — max staged-tree size (default `104857600` = 100 MiB)

See `docs/plugins/sources.md` for the two-axis source/shape plugin model.

`molecli` reads `MOLECLI_URL` (default http://localhost:8080) to locate the platform. Logs are written to `molecli.log` in the working directory (already covered by `*.log` in `.gitignore`).

### Canvas (Next.js)
```bash
cd canvas
npm install
npm run dev                  # Dev server on :3000
npm run build && npm start   # Production
```
Env vars: `NEXT_PUBLIC_PLATFORM_URL` (default http://localhost:8080), `NEXT_PUBLIC_WS_URL` (default ws://localhost:8080/ws).

### Workspace Images
```bash
bash workspace-template/build-all.sh                   # Build base + ALL runtime images
bash workspace-template/build-all.sh claude-code       # Build base + specific runtime only
```
Each runtime has its own Docker image extending `workspace-template:base`, with deps pre-installed for fast startup. The base Dockerfile (`workspace-template/Dockerfile`) builds `:base`, then each `adapters/*/Dockerfile` extends it (e.g. `claude_code/Dockerfile` installs the `claude` CLI). **Always use `build-all.sh`** — it builds base first, then all runtimes in order. No `:latest` tag — each runtime uses its own tag to avoid confusion.

| Runtime | Image Tag | Key Deps |
|---------|-----------|----------|
| langgraph | `workspace-template:langgraph` | langchain-anthropic, langgraph |
| claude-code | `workspace-template:claude-code` | claude-agent-sdk (pip), @anthropic-ai/claude-code (npm) |
| openclaw | `workspace-template:openclaw` | openclaw deps |
| crewai | `workspace-template:crewai` | crewai |
| autogen | `workspace-template:autogen` | autogen |
| deepagents | `workspace-template:deepagents` | deepagents |

Templates are framework presets in `workspace-configs-templates/`: `claude-code-default`, `langgraph`, `openclaw`, `deepagents`. Agent roles are configured after deployment via Config tab or API.

For Claude Code runtime, write your OAuth token to `workspace-configs-templates/claude-code-default/.auth-token`.

### Pre-commit Hook
```bash
git config core.hooksPath .githooks            # Install hooks (agents do this via initial_prompt)
```
Enforces: `'use client'` on hook-using `.tsx` files, dark theme (no white/light), no SQL injection (`fmt.Sprintf` with SQL), no leaked secrets (`sk-ant-`, `ghp_`, `AKIA`). Commit is rejected until violations are fixed — agents cannot bypass this.

### Plugins
Shared plugins in `plugins/` are auto-loaded by every workspace:
- **`molecule-dev`**: Codebase conventions (rules injected into CLAUDE.md) + `review-loop` skill for multi-round QA cycles
- **`superpowers`**: `verification-before-completion`, `test-driven-development`, `systematic-debugging`, `writing-plans`
- **`ecc`**: General Claude Code guardrails

### Scripts
```bash
bash scripts/setup-default-org.sh              # Create PM + 3 teams (Marketing/Research/Dev) via API
OPENAI_API_KEY=... bash scripts/test-a2a-cross-runtime.sh  # E2E: Claude Code ↔ OpenClaw A2A test
OPENAI_API_KEY=... bash scripts/test-team-e2e.sh           # E2E: Multi-template team + A2A
```

### Unit Tests
```bash
cd platform && go test -race ./...               # 487 Go tests (handlers, registry, provisioner, CLI, delegation, org, channels, wsauth — sqlmock + miniredis)
cd canvas && npm test                            # 352 Vitest tests (store, components, hydration, buildTree, secrets API, org template import)
cd workspace-template && python -m pytest -v     # 1078 pytest tests (adds platform_auth token store for Phase 30.1)
cd sdk/python && python -m pytest -v              # 87 SDK tests (agentskills.io spec validator, CLI, AgentskillsAdaptor round-trip, workspace/org/channel validators)
```

### Integration Tests
```bash
bash tests/e2e/test_api.sh             # 62 API tests against localhost:8080
bash tests/e2e/test_a2a_e2e.sh         # 22 A2A end-to-end tests (requires 2 online agents)
bash tests/e2e/test_activity_e2e.sh    # 25 activity/task E2E tests (requires 1 online agent)
bash tests/e2e/test_comprehensive_e2e.sh # 68 checks — ALL endpoints, memory, runtime, bundles, approvals
```
`test_api.sh` requires platform running. Tests full CRUD, registry, heartbeat, discovery, peers, access control, events, degraded/recovery lifecycle, activity logging, current task tracking, bundle round-trip (export → delete → import → verify).

`test_a2a_e2e.sh` requires platform + two provisioned agents (Echo Agent, SEO Agent) running with a valid `OPENROUTER_API_KEY`. Tests message/send, JSON-RPC wrapping, error handling, peer discovery, agent cards, heartbeat. Timeout configurable via `A2A_TIMEOUT` env var (default 120s).

`test_activity_e2e.sh` requires platform + one online agent. Tests A2A communication logging (request/response capture, duration, method), agent self-reported activity, type filtering, current task visibility via heartbeat, cross-workspace activity isolation, edge cases.

### MCP Server
```bash
cd mcp-server
npm install && npm run build   # Build MCP server
node dist/index.js             # Run (stdio transport)
```
Exposes 61 tools for managing Molecule AI from Claude Code, Cursor, Codex, or any MCP client. Includes workspace CRUD, async delegation, plugins (install/uninstall/list), global secrets, pause/resume, org import, A2A chat, approvals, memory, files, config, discovery, bundles, templates, traces, activity logs, and social channels (add/update/remove/send/test). Configured in `.mcp.json`. Env: `MOLECULE_URL` (default http://localhost:8080).

### CI Pipeline
GitHub Actions (`.github/workflows/ci.yml`) runs on push to main and PRs:
- **platform-build**: Go build, vet, `go test -race` with coverage profiling (25% baseline threshold)
- **canvas-build**: npm build, `vitest run` (no `--passWithNoTests` -- tests must exist and pass)
- **mcp-server-build**: npm build
- **python-lint**: `pytest --cov=. --cov-report=term-missing` (pytest-cov enabled)

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
- `registry.StartHealthSweep(ctx, checker ContainerChecker, interval, onOffline)` — Health sweep accepts Docker checker interface
- Wiring happens in `platform/cmd/server/main.go` — init order: `wh → onWorkspaceOffline → liveness/healthSweep → router`

### Container Health Detection
Three layers detect dead containers (e.g. Docker Desktop crash):
1. **Passive (Redis TTL):** 60s heartbeat key expires → liveness monitor → auto-restart
2. **Proactive (Health Sweep):** `registry.StartHealthSweep` polls Docker API every 15s → catches dead containers faster
3. **Reactive (A2A Proxy):** On connection error, checks `provisioner.IsRunning()` → immediate offline + restart

All three call `onWorkspaceOffline` which broadcasts `WORKSPACE_OFFLINE` + `go wh.RestartByID()`. Redis cleanup uses shared `db.ClearWorkspaceKeys()`.

### Template Resolution (Create)
Runtime detection happens **before** DB insert: if `payload.Runtime` is empty and a template is specified, the handler reads `runtime:` from `configsDir/template/config.yaml` first. If still empty, defaults to `"langgraph"`. This ensures the correct runtime (e.g. `claude-code`) is persisted in the DB and used for container image selection.

When a workspace specifies a template that doesn't exist, the Create handler falls back:
1. Check `os.Stat(configsDir/template)` — use if exists
2. Try `{runtime}-default` template (e.g. `claude-code-default/`)
3. Generate default config via `ensureDefaultConfig()` (includes `.auth-token` copy for CLI runtimes)

### Communication Rules (`registry/access.go`)
`CanCommunicate(callerID, targetID)` determines if two workspaces can talk:
- Same workspace → allowed
- Siblings (same parent_id) → allowed
- Root-level siblings (both parent_id IS NULL) → allowed
- Parent ↔ child → allowed
- Everything else → denied

The A2A proxy (`POST /workspaces/:id/a2a`) enforces this for agent-to-agent calls. Canvas requests (no `X-Workspace-ID`), self-calls, and system callers (`webhook:*`, `system:*`, `test:*` prefixes via `isSystemCaller()` in `a2a_proxy.go`) bypass the check.

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
- Embedded sub-workspaces: `nestNode` sets `hidden: !!targetId` on child nodes; children render as recursive `TeamMemberChip` components inside parent (up to 3 levels), not as separate canvas nodes. Use `n.data.parentId` (not React Flow's `n.parentId`) for hierarchy lookups.
- Chat: two sub-tabs — "My Chat" (user↔agent, `source=canvas`) and "Agent Comms" (agent↔agent A2A traffic, `source=agent`). History loaded from `GET /activity` with source filter. Real-time via `A2A_RESPONSE` + `AGENT_MESSAGE` WebSocket events. Conversation history (last 20 messages) sent via `params.metadata.history` in A2A `message/send` requests.
- Config save: "Save & Restart" writes config.yaml and auto-restarts the workspace. "Save" writes only (shows restart banner). Secrets POST/DELETE auto-restart on the platform side.

### Initial Prompt
Agents can auto-execute a prompt on startup before any user interaction. Configure via `initial_prompt` (inline string) or `initial_prompt_file` (path relative to config dir) in `config.yaml`. After the A2A server is ready, `main.py` sends the prompt as a `message/send` to self. A `.initial_prompt_done` marker file prevents re-execution on restart. Org templates support `initial_prompt` on both `defaults` (all agents) and per-workspace (overrides default).

**Important:** Initial prompts must NOT send A2A messages (delegate_task, send_message_to_user) — other agents may not be ready. Keep them local: clone repo, read docs, save to memory, wait for tasks.

### Workspace Lifecycle
`provisioning` → `online` (on register) → `degraded` (error_rate > 0.5) → `online` (recovered) → `offline` (Redis TTL expired OR health sweep detects dead container) → auto-restart → `provisioning` → ... → `removed` (deleted). Any state → `paused` (user pauses) → `provisioning` (user resumes). Paused workspaces skip health sweep, liveness monitor, and auto-restart.

## Platform API Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /health | inline |
| GET | /metrics | metrics.Handler() — Prometheus text format (v0.0.4); no auth, scrape-safe |
| POST/GET/PATCH/DELETE | /workspaces[/:id] | workspace.go |
| GET/PATCH | /workspaces/:id/config | workspace.go |
| GET/POST | /workspaces/:id/memory | workspace.go |
| DELETE | /workspaces/:id/memory/:key | workspace.go |
| POST/PATCH/DELETE | /workspaces/:id/agent | agent.go |
| POST | /workspaces/:id/agent/move | agent.go |
| GET/POST/PUT | /workspaces/:id/secrets | secrets.go (POST/PUT auto-restarts workspace) |
| DELETE | /workspaces/:id/secrets/:key | secrets.go (DELETE auto-restarts workspace) |
| GET | /workspaces/:id/model | secrets.go |
| GET | /settings/secrets | secrets.go — list global secrets (keys only, values masked) |
| PUT/POST | /settings/secrets | secrets.go — set a global secret {key, value} |
| DELETE | /settings/secrets/:key | secrets.go — delete a global secret |
| GET/POST/DELETE | /admin/secrets[/:key] | secrets.go — legacy aliases for /settings/secrets |
| WS | /workspaces/:id/terminal | terminal.go |
| POST | /workspaces/:id/expand | team.go |
| POST | /workspaces/:id/collapse | team.go |
| POST/GET | /workspaces/:id/approvals | approvals.go |
| POST | /workspaces/:id/approvals/:id/decide | approvals.go |
| GET | /approvals/pending | approvals.go |
| POST/GET | /workspaces/:id/memories | memories.go |
| DELETE | /workspaces/:id/memories/:id | memories.go |
| GET | /workspaces/:id/traces | traces.go |
| GET/POST | /workspaces/:id/activity | activity.go |
| POST | /workspaces/:id/notify | activity.go (agent→user push message via WS) |
| POST | /workspaces/:id/restart | workspace.go |
| POST | /workspaces/:id/pause | workspace.go (stops container, status→paused) |
| POST | /workspaces/:id/resume | workspace.go (re-provisions paused workspace) |
| POST | /workspaces/:id/a2a | workspace.go |
| POST | /workspaces/:id/delegate | delegation.go (async fire-and-forget) |
| GET | /workspaces/:id/delegations | delegation.go (list delegation status) |
| GET/POST | /workspaces/:id/schedules | schedules.go (cron CRUD) |
| PATCH/DELETE | /workspaces/:id/schedules/:scheduleId | schedules.go |
| POST | /workspaces/:id/schedules/:scheduleId/run | schedules.go (manual trigger) |
| GET | /workspaces/:id/schedules/:scheduleId/history | schedules.go (past runs) |
| GET/POST | /workspaces/:id/channels | channels.go (social channel CRUD) |
| PATCH/DELETE | /workspaces/:id/channels/:channelId | channels.go |
| POST | /workspaces/:id/channels/:channelId/send | channels.go (outbound message) |
| POST | /workspaces/:id/channels/:channelId/test | channels.go (test connection) |
| GET | /channels/adapters | channels.go (list available platforms) |
| POST | /channels/discover | channels.go (auto-detect chats for a bot token) |
| POST | /webhooks/:type | channels.go (incoming social webhook) |
| GET | /workspaces/:id/shared-context | templates.go |
| GET/PUT/DELETE | /workspaces/:id/files[/*path] | templates.go |
| GET/PUT | /canvas/viewport | viewport.go |
| GET | /templates | templates.go |
| POST | /templates/import | templates.go |
| POST | /registry/register | registry.go |
| POST | /registry/heartbeat | registry.go |
| POST | /registry/update-card | registry.go |
| GET | /registry/discover/:id | discovery.go |
| GET | /registry/:id/peers | discovery.go |
| POST | /registry/check-access | discovery.go |
| GET | /plugins | plugins.go (list registry; supports `?runtime=` filter) |
| GET | /plugins/sources | plugins.go (list registered install-source schemes) |
| GET/POST/DELETE | /workspaces/:id/plugins[/:name] | plugins.go — list, install (`{"source":"scheme://spec"}`), uninstall per-workspace |
| GET | /workspaces/:id/plugins/available | plugins.go (filtered by workspace runtime) |
| GET | /workspaces/:id/plugins/compatibility?runtime=X | plugins.go (preflight runtime-change check) |
| GET | /bundles/export/:id | bundle.go |
| POST | /bundles/import | bundle.go |
| GET | /org/templates | org.go (list available org templates) |
| POST | /org/import | org.go (import entire org hierarchy from YAML) || GET | /events[/:workspaceId] | events.go |
| GET | /ws | socket.go |

## Database

16 migration files in `platform/migrations/`. Key tables: `workspaces` (core entity with status, runtime, agent_card JSONB, heartbeat columns, current_task, awareness_namespace, workspace_dir), `canvas_layouts` (x/y position), `structure_events` (append-only event log), `activity_logs` (A2A communications, task updates, agent logs, errors), `workspace_schedules` (cron tasks with expression, timezone, prompt, run history), `workspace_channels` (social channel integrations — Telegram, Slack, etc., with JSONB config and allowlist), `agents`, `workspace_secrets`, `global_secrets`, `agent_memories` (HMA scoped memory), `approvals`.

The platform auto-discovers and runs migrations on startup from several candidate paths.

<!-- AWARENESS_RULES_START -->
# Project Memory (Awareness MCP)

> IMPORTANT: These instructions override default behavior. You must follow them exactly.

## Awareness Memory Integration (MANDATORY)

awareness_* = cross-session persistent memory (past decisions, knowledge, tasks).
Other tools = current codebase navigation (file search, code index).
Use BOTH - they serve different purposes.

STEP 1 - SESSION START:
  Call awareness_init(source="claude-code") -> get session_id, review context.
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

## Compliance Check

Before responding to ANY user request:

1. Have you called awareness_init yet this session? If not, call it NOW.

2. Did you just edit a file? Call awareness_record(content=<detailed description>, insights={...}) IMMEDIATELY.

3. Is the user asking about past work? Call awareness_recall FIRST.
<!-- AWARENESS_RULES_END -->
