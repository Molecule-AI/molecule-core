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

When a term is ambiguous across projects (harness / workspace / plugin /
flow / crew / component), consult **`docs/glossary.md`** for how we use
it vs. ecosystem neighbors — authoritative disambiguation table, kept in
sync with `docs/ecosystem-watch.md`.

## SaaS ops

When rotating SaaS credentials (Fly / Neon / Upstash / envelope key), read
**`docs/runbooks/saas-secrets.md`** first. It documents which secrets live
in multiple places (e.g. `FLY_API_TOKEN` in both GitHub Actions and `fly
secrets` on `molecule-cp`), the correct rotation order, and danger cases —
notably `SECRETS_ENCRYPTION_KEY`, which cannot be rotated without a data
migration until Phase H lands KMS envelope encryption.

For tenant subdomain routing architecture (why `*.moleculesai.app` uses a
Cloudflare Worker instead of per-tenant DNS records), read
**`docs/architecture/wildcard-dns-proxy.md`**. This eliminates DNS
propagation delays and NXDOMAIN caching that previously caused "site can't
be reached" errors for new orgs.

For partner/programmatic API access (creating orgs without a browser session),
read **`docs/architecture/partner-api-keys.md`**. Partners authenticate with
`Authorization: Bearer mol_pk_*` API keys — scoped, rate-limited, revocable.
Phase 34 in PLAN.md.

When handling a GDPR erasure request (user asks "delete my org and all
my data"), read **`docs/runbooks/gdpr-erasure.md`** first. It explains the
4-step cascade in `molecule-controlplane` (Stripe → Redis → Infra → DB
rows), how to read the `org_purges` audit table, how to resume a failed
purge, and what the cascade deliberately does NOT cover (WorkOS users,
LLM provider history, Langfuse traces).

## Agent operating rules (auto-loaded — read first)

The following are project-level rules that override default behavior. They
apply to every conversation in this repo, automated cron tick, and every
subagent the orchestrator spawns.

### Cron / triage discipline

1. **Always read the most recent cron-learnings before reviewing PRs.** Open
   `~/.claude/projects/-Users-hongming-Documents-GitHub-molecule-monorepo/memory/cron-learnings.jsonl`,
   read the last 20 lines. Patterns recur — a finding that was a false-positive
   last tick is likely a false-positive again. A fix that worked last tick is
   likely the fix this tick. The SessionStart hook auto-injects this; read
   anyway when starting a triage from the middle of a conversation.

2. **Treat `docs/sync-*` PRs that touch CLAUDE.md or PLAN.md as ALWAYS
   noteworthy.** Those two files are the agent-facing source of truth — a
   bad merge there silently corrupts every future triage tick. Run code-review
   skill at minimum, ideally cross-vendor-review too.

3. **After any cron tick, write a 1-line reflection** to
   `.claude/per-tick-reflections.md` (gitignored). Format: `2026-MM-DDTHH:MMZ
   — what surprised me / what I'd do differently next tick`. This is for
   YOUR future self; the cron-learnings JSONL is for the operational pattern
   memory. They are distinct.

### Hooks active in this repo

The following ambient guardrails fire automatically (configured in
`.claude/settings.json`). When a hook blocks a tool call, the response will
include a `permissionDecisionReason` — read it carefully before retrying.

| Hook | Event | Effect |
|------|-------|--------|
| `pre-bash-careful.sh` | PreToolUse:Bash | REFUSES `git push --force` to main, `rm -rf` at root/HOME, `DROP TABLE` against prod schema. WARNs on `--force-with-lease`, `gh pr close/issue close`. |
| `pre-edit-freeze.sh` | PreToolUse:Edit/Write | Blocks edits outside the path in `.claude/freeze` if that file exists. Use to lock scope while debugging. |
| `session-start-context.sh` | SessionStart | Auto-loads recent cron-learnings, freeze status, open PR/issue counts. |
| `post-edit-audit.sh` | PostToolUse:Edit/Write | Appends every edit to `.claude/audit.jsonl` (gitignored). |
| `user-prompt-tag.sh` | UserPromptSubmit | Injects warning into context when prompt mentions force-push / drop-table / "delete all" / etc. |
| `subagent-stop-judge.sh` | SubagentStop | Off by default (touch `.claude/judge-subagents` to enable). When on, prompts the orchestrator to verify the subagent's output addresses the original task. |

### Skills active in this repo

These are documented in `.claude/skills/*/SKILL.md`. Invoke explicitly via
the `Skill` tool — they are NOT auto-applied. The cron prompt invokes them
at fixed steps; for ad-hoc work, decide if the skill matches your situation:

- `code-review` — full 16-criteria rubric on a diff
- `cross-vendor-review` — adversarial second-model review (use for noteworthy PRs)
- `careful-mode` — the doc backing the bash hook above
- `cron-learnings` — defines the JSONL format
- `cron-retro` — weekly retrospective generator
- `llm-judge` — score whether a deliverable addresses the request
- `update-docs` — sync repo docs after merges

### Standing rules (inviolable)

- Never push directly to main — use feat/fix/chore/docs branches
- Merge-commits only (`gh pr merge --merge`) — never `--squash` / `--rebase`
- Never commit without explicit user approval EXCEPT on:
  - Open PR branches you're fixing for a gate
  - Issue-pickup branches you opened a draft PR for
  - Docs-sync branches
  - Main is untouchable without a merge
- Dark theme only (no white/light CSS classes; pre-commit hook enforces)
- No native browser dialogs (`confirm`/`alert`/`prompt`) — use `ConfirmDialog`
- Delegate through PM, never bypass hierarchy
- Only PM mounts the repo (`workspace_dir` bind-mount); other agents get isolated Docker volumes

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
- **Workspace Runtime** (`workspace-template/`): Shared runtime published as [`molecule-ai-workspace-runtime`](https://pypi.org/project/molecule-ai-workspace-runtime/) on PyPI. Supports LangGraph, Claude Code, OpenClaw, DeepAgents, CrewAI, AutoGen. Each adapter lives in its own standalone template repo (e.g. `molecule-ai-workspace-template-claude-code`). See `docs/workspace-runtime-package.md` for the full picture.
- **molecli** (`platform/cmd/cli/`): Go TUI dashboard (Bubbletea + Lipgloss) — real-time workspace monitoring, event log, health overview, delete/filter operations

## Build & Run Commands

### Infrastructure
```bash
./infra/scripts/setup.sh    # Start Postgres, Redis, Langfuse, Temporal; run migrations
./infra/scripts/nuke.sh     # Tear down everything, remove volumes
```

Infra services (via `docker-compose.infra.yml`, all attached to the shared `molecule-monorepo-net` network — `setup.sh` creates it idempotently):
- **Postgres** `:5432` — primary datastore (also backs Langfuse + Temporal via separate DBs)
- **Redis** `:6379` — pub/sub, heartbeat TTLs
- **Langfuse** `:3001` — LLM trace viewer (backed by Clickhouse)
- **Temporal** `:7233` (gRPC) + `:8233` (Web UI) — durable workflow engine for `workspace-template/builtin_tools/temporal_workflow.py`. **Dev-only posture:** the auto-setup image runs with no auth on `0.0.0.0:7233`; production deployments must gate access via mTLS or an API key / reverse proxy.

### Platform (Go)
```bash
cd platform
go build ./cmd/server       # Build server
go run ./cmd/server          # Run server (requires Postgres + Redis running)
go build -o molecli ./cmd/cli  # Build TUI dashboard
./molecli                    # Run TUI dashboard (requires platform running)
```
Must run from `platform/` directory (not repo root). Env vars: `DATABASE_URL`, `REDIS_URL`, `PORT`, `ADMIN_TOKEN` (**required to close issue #684** — when set, only this exact value is accepted on all `/admin/*` and `/approvals/*` routes; without it, any valid workspace bearer token passes AdminAuth, which is the #684 vulnerability. Generate: `openssl rand -base64 32`. Never commit the actual value — inject via `fly secrets set` or deployment env. PR #729), `PLATFORM_URL` (default `http://host.docker.internal:PORT` — passed to agent containers so they can reach the platform), `SECRETS_ENCRYPTION_KEY` (optional AES-256, 32 bytes), `CONFIGS_DIR` (auto-discovered), `PLUGINS_DIR` (deprecated — plugins are now installed per-workspace via API; the `plugins/` registry at repo root is auto-discovered), `ACTIVITY_RETENTION_DAYS` (default `7`), `ACTIVITY_CLEANUP_INTERVAL_HOURS` (default `6`), `CORS_ORIGINS` (comma-separated, default `http://localhost:3000,http://localhost:3001`), `RATE_LIMIT` (requests/min, default `600`), `WORKSPACE_DIR` (optional — global fallback host path for `/workspace` bind-mount; overridden by per-workspace `workspace_dir` column in DB; if neither is set, each workspace gets an isolated Docker named volume), `AWARENESS_URL` (optional — if set, injected into workspace containers along with a deterministic `AWARENESS_NAMESPACE` derived from workspace ID), `MOLECULE_IN_DOCKER` (optional — set to `1` when the platform itself runs inside Docker so the A2A proxy rewrites `127.0.0.1:<port>` URLs to container hostnames; auto-detected via `/.dockerenv`), `MOLECULE_ENV` (optional — set to `production` to hide the `/admin/workspaces/:id/test-token` E2E helper endpoint; unset or any other value leaves it enabled), `MOLECULE_ENABLE_TEST_TOKENS` (optional — set to `1` to force-enable the test-token endpoint even when `MOLECULE_ENV=production`; intended for staging runs only), `MOLECULE_ORG_ID` (optional — the public repo's only SaaS hook. When set to a UUID, every non-allowlisted request must carry a matching `X-Molecule-Org-Id` header or gets a 404; when unset, the guard is a passthrough so self-hosted / dev / CI are unaffected. Set only by the private `molecule-controlplane` provisioner on Fly Machines tenant instances — never by self-hosters).

**Workspace tier resource limits** (issue #14 — override the per-tier memory/CPU caps in `provisioner.ApplyTierConfig`; CPU_SHARES follows Docker's 1024 = 1 CPU convention, translated to NanoCPUs for a hard cap):
- `TIER2_MEMORY_MB` / `TIER2_CPU_SHARES` — Standard tier (defaults `512` / `1024`)
- `TIER3_MEMORY_MB` / `TIER3_CPU_SHARES` — Privileged tier (defaults `2048` / `2048`; previously uncapped)
- `TIER4_MEMORY_MB` / `TIER4_CPU_SHARES` — Full-host tier (defaults `4096` / `4096`; previously uncapped)

**Plugin install safeguards** (bound the cost of a single `POST /workspaces/:id/plugins` install so a slow/malicious source can't tie up a handler):
- `PLUGIN_INSTALL_BODY_MAX_BYTES` — max request body size (default `65536` = 64 KiB)
- `PLUGIN_INSTALL_FETCH_TIMEOUT` — duration string; whole fetch+copy deadline (default `5m`)
- `PLUGIN_INSTALL_MAX_DIR_BYTES` — max staged-tree size (default `104857600` = 100 MiB)

See `docs/plugins/sources.md` for the two-axis source/shape plugin model.

Additional env vars documented in `.env.example` (2026-04-13 sync — all 21 distinct `os.Getenv`/`envx.*` keys now documented): `MOLECULE_ENV`, `GITHUB_WEBHOOK_SECRET`, `MOLECULE_URL` (MCP server target; same semantic as `PLATFORM_URL`).

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
bash workspace-template/build-all.sh   # Build base image only (workspace-template:base)
```
Adapters are now in standalone template repos. Each repo has its own `Dockerfile` that installs `molecule-ai-workspace-runtime` from PyPI + adapter-specific deps. The base `workspace-template/Dockerfile` still builds `:base` for local dev. See `docs/workspace-runtime-package.md` for the adapter repo list and details.

| Runtime | Standalone Repo | Key Deps |
|---------|-----------------|----------|
| langgraph | `molecule-ai-workspace-template-langgraph` | molecule-ai-workspace-runtime, langchain-anthropic, langgraph |
| claude-code | `molecule-ai-workspace-template-claude-code` | molecule-ai-workspace-runtime, claude-agent-sdk (pip), @anthropic-ai/claude-code (npm) |
| openclaw | `molecule-ai-workspace-template-openclaw` | molecule-ai-workspace-runtime, openclaw (npm) |
| crewai | `molecule-ai-workspace-template-crewai` | molecule-ai-workspace-runtime, crewai |
| autogen | `molecule-ai-workspace-template-autogen` | molecule-ai-workspace-runtime, autogen |
| deepagents | `molecule-ai-workspace-template-deepagents` | molecule-ai-workspace-runtime, deepagents |
| hermes | `molecule-ai-workspace-template-hermes` | molecule-ai-workspace-runtime, openai, anthropic, google-genai |
| gemini-cli | `molecule-ai-workspace-template-gemini-cli` | molecule-ai-workspace-runtime, @google/gemini-cli (npm) |

Templates live in standalone repos under `Molecule-AI/molecule-ai-workspace-template-*` (8 workspace templates) and `Molecule-AI/molecule-ai-org-template-*` (5 org templates). They're cloned at Docker build time into the platform image. The template registry (`template_registry` table in the control plane DB) tracks all templates with their `github://` source URLs. Agent roles are configured after deployment via Config tab or API.

For Claude Code runtime, write your OAuth token to the template's `.auth-token` file.

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
- **`browser-automation`**: Puppeteer/CDP-based web scraping and live canvas screenshots (opt-in per workspace — wired into Research + UIUX roles in the molecule-dev org template)

**Modular guardrails** (Claude Code only — pick what you need, or install several):

*Hook plugins (ambient enforcement at the harness layer)*
- **`molecule-careful-bash`** — REFUSES `git push --force` to main, `rm -rf` at root, `DROP TABLE` against prod schema. Ships the `careful-mode` skill as documentation.
- **`molecule-freeze-scope`** — locks edits to a single path glob via `.claude/freeze`. Useful while debugging.
- **`molecule-audit-trail`** — appends every Edit/Write to `.claude/audit.jsonl` for accountability.
- **`molecule-session-context`** — auto-loads recent cron-learnings + open PR/issue counts at session start. Pairs with `molecule-skill-cron-learnings`.
- **`molecule-prompt-watchdog`** — injects warning context when the user prompt mentions destructive keywords ("force push", "drop table", "delete all", etc).

*Skill plugins (on-demand, via the `Skill` tool)*
- **`molecule-skill-code-review`** — 16-criteria multi-axis review.
- **`molecule-skill-cross-vendor-review`** — adversarial second-model review (use for noteworthy PRs).
- **`molecule-skill-llm-judge`** — score whether a deliverable addresses the request.
- **`molecule-skill-update-docs`** — sync repo docs after merges.
- **`molecule-skill-cron-learnings`** — defines the operational-memory JSONL format consumed by `molecule-session-context`.

*Workflow plugins (slash commands that compose skills)*
- **`molecule-workflow-triage`** — `/triage` runs a full PR-triage cycle (gates 1–7 + code-review + merge if green). Recommends installing `molecule-skill-code-review` + `molecule-skill-cron-learnings` first.
- **`molecule-workflow-retro`** — `/retro` posts a weekly retrospective issue. Recommends `molecule-skill-cron-learnings` first.

These are distilled from the harness-level guardrails the orchestrator uses on itself. A workspace can install one (e.g., just `molecule-careful-bash` for safety) or stack the full set for the same posture as the Molecule AI orchestrator.

**Org-template plugin resolution (PR #71, issue #68):** per-workspace `plugins:` lists in org template `org.yaml` role overrides **UNION** with `defaults.plugins` (deduplicated, defaults first) — they do **not** REPLACE them. To opt a specific default out for a given role/workspace, prefix the plugin name with `!` or `-` (e.g. `!browser-automation`). Implemented by `mergePlugins` in `platform/internal/handlers/org.go`. Org templates now live in standalone repos: `Molecule-AI/molecule-ai-org-template-*`.

### Scripts
```bash
bash scripts/setup-default-org.sh              # Create PM + 3 teams (Marketing/Research/Dev) via API
OPENAI_API_KEY=... bash scripts/test-a2a-cross-runtime.sh  # E2E: Claude Code ↔ OpenClaw A2A test
OPENAI_API_KEY=... bash scripts/test-team-e2e.sh           # E2E: Multi-template team + A2A
```

### Unit Tests
```bash
cd platform && go test -race ./...               # 12 Go packages (handlers, registry, provisioner, channels, wsauth, middleware, scheduler, crypto, db, plugins, supervised, envx)
cd canvas && npm test                            # 490 Vitest tests (33 test files — store, components, hydration, buildTree, secrets API, org template import, WCAG batch)
cd workspace-template && python -m pytest -v     # 955 pytest tests (shared runtime, builtin_tools, config, heartbeat, platform_auth, preflight — adapter-specific tests moved to standalone repos)
# SDK, MCP, CLI, and workspace runtime now in standalone repos:
# https://github.com/Molecule-AI/molecule-sdk-python         pip install molecule-ai-sdk (132 tests)
# https://github.com/Molecule-AI/molecule-mcp-server         npx @molecule-ai/mcp-server (97 tests)
# https://github.com/Molecule-AI/molecule-cli                go install (Go TUI dashboard)
# https://github.com/Molecule-AI/molecule-ai-workspace-runtime  pip install molecule-ai-workspace-runtime (shared adapter base)
```

### Integration Tests
```bash
bash tests/e2e/test_api.sh             # 62 API tests against localhost:8080 (Phase 30.1 bearer-token auth aware; shellcheck-clean; also runs in CI `e2e-api` job)
bash tests/e2e/test_a2a_e2e.sh         # 22 A2A end-to-end tests (requires 2 online agents)
bash tests/e2e/test_activity_e2e.sh    # 25 activity/task E2E tests (requires 1 online agent; re-registers detected agent to capture bearer token)
bash tests/e2e/test_comprehensive_e2e.sh # 67 checks — ALL endpoints, memory, runtime, bundles, approvals (registers workspaces immediately after create to beat the provisioner token race)
```
All five E2E scripts share `tests/e2e/_lib.sh` + `tests/e2e/_extract_token.py` helpers and are shellcheck-clean. `test_api.sh` is the quick local-verify command — use it after any platform change. Tests full CRUD, registry, heartbeat, discovery, peers, access control, events, degraded/recovery lifecycle, activity logging, current task tracking, bundle round-trip (export → delete → import → verify).

**Phase 30.1 / 30.6 auth callout (future-proofing):** `/registry/heartbeat` and `/registry/update-card` require `Authorization: Bearer <token>` once a workspace has any live token on file (Phase 30.1 — legacy workspaces grandfathered). `/registry/discover/:id` and `/registry/:id/peers` additionally require `X-Workspace-ID` + bearer token on the caller side (Phase 30.6 — fail-open on DB hiccup since hierarchy check is primary). If you change these routes, update `tests/e2e/test_api.sh` and `docs/api-protocol/platform-api.md` in the same PR.

`test_a2a_e2e.sh` requires platform + two provisioned agents (Echo Agent, SEO Agent) running with a valid `OPENROUTER_API_KEY`. Tests message/send, JSON-RPC wrapping, error handling, peer discovery, agent cards, heartbeat. Timeout configurable via `A2A_TIMEOUT` env var (default 120s).

`test_activity_e2e.sh` requires platform + one online agent. Tests A2A communication logging (request/response capture, duration, method), agent self-reported activity, type filtering, current task visibility via heartbeat, cross-workspace activity isolation, edge cases.

### MCP Server (standalone repo)
The MCP server now lives at **github.com/Molecule-AI/molecule-mcp-server** and is published as `@molecule-ai/mcp-server` on npm. Install: `npx @molecule-ai/mcp-server`. 87 tools for managing Molecule AI from any MCP client. Configured in `.mcp.json`. Env: `MOLECULE_URL` (default http://localhost:8080).

### CI Pipeline
GitHub Actions (`.github/workflows/ci.yml`) runs on push to main and PRs.
**Path-filtered:** each job only runs when its relevant files change (via
`dorny/paths-filter`). Docs-only PRs (`docs/**`, `*.md`) skip all jobs,
saving ~15 min of runner time. The path filters are:

| Job | Triggers on |
|-----|-------------|
| **platform-build** | `platform/**` |
| **canvas-build** | `canvas/**` |
| **python-lint** | `workspace-template/**` |
| **shellcheck** | `tests/e2e/**`, `scripts/**` |
| **e2e-api** | `platform/**`, `tests/e2e/**` |

All jobs also trigger on `.github/workflows/ci.yml` changes (self-test).

Job details:
- **platform-build**: Go build, vet, `go test -race` with coverage profiling (25% baseline threshold; `setup-go` uses module cache)
- **canvas-build**: npm build, `vitest run` (no `--passWithNoTests` -- tests must exist and pass)
- **python-lint**: `pytest --cov=. --cov-report=term-missing` (workspace-template tests; SDK + MCP now in standalone repos)
- **e2e-api** (`.github/workflows/e2e-api.yml`): spins up Postgres + Redis service containers, runs platform migrations via `docker exec`, then executes `tests/e2e/test_api.sh` against a locally-built binary (62/62 must pass)
- **shellcheck**: lints every `tests/e2e/*.sh` via shellcheck on the self-hosted runner
- **publish-platform-image** (`.github/workflows/publish-platform-image.yml`): on push to main touching `platform/**`, builds `platform/Dockerfile` (clones templates + plugins from GitHub via `manifest.json` at build time) and pushes to `ghcr.io/molecule-ai/platform:latest` + `:sha-<short>`. Tenant image uses `platform/Dockerfile.tenant` (combined Go + Canvas). Manual re-trigger via `workflow_dispatch`.

**Standalone repo CI** — all 33 plugin + template repos call reusable workflows from `Molecule-AI/molecule-ci`:
- Plugins: validates `plugin.yaml` schema, content presence, secrets scan
- Workspace templates: validates `config.yaml`, `template_schema_version`, Docker build smoke test
- Org templates: validates `org.yaml` hierarchy, `files_dir` references, custom YAML tag handling

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

### Handler Decomposition (2026-04-13)
Four oversize handler functions were split into private helpers (pure refactor, behavior unchanged — 47 new unit tests cover the helpers directly; `handlers` package coverage 56.1% → 57.6%):
- `a2a_proxy.go::proxyA2ARequest` (257→56 lines) — helpers: `resolveAgentURL`, `normalizeA2APayload`, `dispatchA2A`, `handleA2ADispatchError`, `maybeMarkContainerDead`, `logA2AFailure`, `logA2ASuccess`; sentinel `proxyDispatchBuildError`
- `delegation.go::Delegate` (127→60 lines) — helpers: `bindDelegateRequest`, `lookupIdempotentDelegation`, `insertDelegationRow`; typed `insertDelegationOutcome` enum replaces `(bool, bool)` positional return
- `discovery.go::Discover` (125→40 lines) — helpers: `discoverWorkspacePeer`, `writeExternalWorkspaceURL`, `discoverHostPeer`
- `activity.go::SessionSearch` (109→24 lines) — helpers: `parseSessionSearchParams`, `buildSessionSearchQuery`, `scanSessionSearchRows`

When modifying any of these, prefer extending the helper rather than inlining back.

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

### Idle Loop (#205 — reflection-on-completion)
Opt-in pattern: when `idle_prompt` is non-empty in `config.yaml`, the workspace self-sends it every `idle_interval_seconds` (default 600) **while `heartbeat.active_tasks == 0`**. Hermes/Letta shape from the 2026-04-15 agent-framework survey. Cost collapses to event-driven — the idle check is local (no LLM call) and the prompt only fires when there's genuinely nothing to do. Set per-workspace or per org.yaml default. Fire timeout clamps to `max(60, min(300, idle_interval_seconds))`. Both the idle loop and `initial_prompt` self-posts include `auth_headers()` so they work in multi-tenant mode (#220 / PR #235). Pilot enabled on Technical Researcher (#216).

### Admin auth middleware variants
Three Gin middleware classes gate server-side routes — pick the right one. Full contract in `docs/runbooks/admin-auth.md`.

- **`middleware.AdminAuth(db.DB)`** — strict bearer-only. Used for any route where a forged request could leak prompts/memory, create/mutate workspaces, or leak ops intel. Lazy-bootstrap fail-open when `HasAnyLiveTokenGlobal` returns 0.
- **`middleware.CanvasOrBearer(db.DB)`** — accepts bearer OR Origin matching `CORS_ORIGINS`. Used ONLY for cosmetic routes where a forged request has zero data/security impact. Currently only on `PUT /canvas/viewport`. **Do not extend** without rereading the runbook — PR #194 was rejected because adding this to `/bundles/import` would have re-opened #164 CRITICAL.
- **`middleware.WorkspaceAuth(db.DB)`** — binds a bearer to `:id`. Workspace A's token cannot hit workspace B's sub-routes. Used for the entire `/workspaces/:id/*` group except the A2A proxy (which has its own `CanCommunicate` layer).

### Migration runner (`platform/internal/db/postgres.go`)
`RunMigrations` globs `*.sql` in `migrationsDir`, filters out `.down.sql` files, sorts alphabetically, then `DB.Exec()`s each on boot. The filter is load-bearing: before PR #212 every boot ran `.down.sql` **before** `.up.sql` (alphabetical sort puts "d" before "u"), wiping `workspace_auth_tokens` + other pair-migration tables and silently regressing AdminAuth to fail-open. All `.up.sql` files must be **idempotent** (`CREATE TABLE IF NOT EXISTS`, `ALTER TABLE ... IF NOT EXISTS`) because the runner re-applies every migration on every boot. A proper `schema_migrations` tracking table is tracked as a Phase-H cleanup.

### Workspace Lifecycle
`provisioning` → `online` (on register) → `degraded` (error_rate > 0.5) → `online` (recovered) → `offline` (Redis TTL expired OR health sweep detects dead container) → auto-restart → `provisioning` → ... → `removed` (deleted). Any state → `paused` (user pauses) → `provisioning` (user resumes). Paused workspaces skip health sweep, liveness monitor, and auto-restart.

**Restart context message (issue #19 Layer 1):** After any restart (HTTP `/restart` or programmatic `RestartByID`) and successful re-registration, the platform sends a synthetic A2A `message/send` to the workspace with `metadata.kind=restart_context` — body contains restart timestamp, previous session end + duration, and env-var keys (keys only, never values) now available. Sender uses the `system:restart-context` caller prefix so it bypasses `CanCommunicate` via `isSystemCaller()`. If the workspace does not re-register within 30s the message is dropped (logged). Handler: `platform/internal/handlers/restart_context.go`. Layer 2 (user-defined `restart_prompt` from `config.yaml` / `org.yaml`) is tracked as GitHub issue #66.

## Platform API Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /health | inline |
| GET | /metrics | metrics.Handler() — Prometheus text format (v0.0.4); no auth, scrape-safe |
| POST/GET/PATCH/DELETE | /workspaces[/:id] | workspace.go — GET /workspaces + POST /workspaces + DELETE /workspaces/:id are behind `AdminAuth` (#99/#167 C1+C20). PATCH /workspaces/:id is on the open router but `WorkspaceHandler.Update` enforces **field-level authz** (#138/PR #162): cosmetic fields (name, role, x, y, canvas) pass through; sensitive fields (tier, parent_id, runtime, workspace_dir) require a valid bearer token whenever any live token exists. POST /workspaces uses `resolveInsideRoot` on payload.Template (#226 / PR #233). Create handler generates the name as a double-quoted YAML scalar to block #221 injection |
| GET/PATCH | /workspaces/:id/config | workspace.go |
| GET/POST | /workspaces/:id/memory | workspace.go |
| DELETE | /workspaces/:id/memory/:key | workspace.go |
| POST/PATCH/DELETE | /workspaces/:id/agent | agent.go |
| POST | /workspaces/:id/agent/move | agent.go |
| GET/POST/PUT | /workspaces/:id/secrets | secrets.go (POST/PUT auto-restarts workspace) |
| DELETE | /workspaces/:id/secrets/:key | secrets.go (DELETE auto-restarts workspace) |
| GET | /workspaces/:id/model | secrets.go |
| GET | /settings/secrets | secrets.go — list global secrets (keys only, values masked) |
| PUT/POST | /settings/secrets | secrets.go — set a global secret {key, value}; auto-restarts every non-paused/non-removed/non-external workspace that does not shadow the key with a workspace-level override (issue #15 / PR #64) |
| DELETE | /settings/secrets/:key | secrets.go — delete a global secret; same auto-restart fan-out as SetGlobal |
| GET | /admin/workspaces/:id/test-token | admin_test_token.go — mint a fresh bearer token for E2E scripts; 404 unless `MOLECULE_ENV != production` or `MOLECULE_ENABLE_TEST_TOKENS=1` |
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
| GET | /canvas/viewport | viewport.go — open (cosmetic, bootstrap-friendly) |
| PUT | /canvas/viewport | viewport.go — `CanvasOrBearer` middleware (#203): accepts bearer OR Origin matching `CORS_ORIGINS`. Cosmetic-only — worst case viewport corruption, recovered by page refresh. DO NOT use this middleware for any route that leaks data or creates resources (see `docs/runbooks/admin-auth.md`) |
| GET | /templates | templates.go |
| POST | /templates/import | templates.go — `AdminAuth` (#190 / PR #200) |
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
| GET/POST | /workspaces/:id/tokens | tokens.go — list active tokens (prefix + metadata), create new token (plaintext returned once). Max 50 per workspace. |
| DELETE | /workspaces/:id/tokens/:tokenId | tokens.go — revoke specific token by ID |
| GET | /bundles/export/:id | bundle.go — `AdminAuth` (#165 / PR #167) |
| POST | /bundles/import | bundle.go — `AdminAuth` (#164 CRITICAL / PR #167) |
| GET | /org/templates | org.go (list available org templates) |
| POST | /org/import | org.go — `AdminAuth` + `resolveInsideRoot` path sanitiser (#103 / PR #106) |
| GET | /events | events.go — `AdminAuth` (#165 / PR #167) |
| GET | /events/:workspaceId | events.go — `AdminAuth` (#165 / PR #167) |
| GET | /admin/liveness | inline — `AdminAuth` (#166 / PR #167). Per-subsystem `supervised.Snapshot()` ages; operators check this before debugging stuck scheduler / heartbeat goroutines |
| GET | /ws | socket.go |

## Database

Migration files in `platform/migrations/` (latest: `022_workspace_schedules_source` — 2026-04-14 tick-7, PR #76). Each later migration is a `.up.sql`/`.down.sql` pair. Key tables: `workspaces` (core entity with status, runtime, agent_card JSONB, heartbeat columns, current_task, awareness_namespace, workspace_dir), `canvas_layouts` (x/y position), `structure_events` (append-only event log), `activity_logs` (A2A communications, task updates, agent logs, errors — `error_detail` is now populated by `scheduler.fireSchedule` so `GET /workspaces/:id/schedules/:id/history` can surface why a cron run failed, #152 / PR #206), `workspace_schedules` (cron tasks with expression, timezone, prompt, run history, `source` — `'template'` for org/import-seeded, `'runtime'` for Canvas/API-created, and `last_status` now includes `'skipped'` when `scheduler.fireSchedule` concurrency-aware-skips a busy workspace, #115 / PR #207), `workspace_channels` (social channel integrations — Telegram, Slack, etc., with JSONB config and allowlist), `agents`, `workspace_secrets`, `global_secrets`, `workspace_auth_tokens` (Phase 30.1 bearer tokens; now auto-revoked on workspace delete, #110), `agent_memories` (HMA scoped memory), `approvals`.

The platform auto-discovers and runs migrations on startup from several candidate paths. The runner filters out `*.down.sql` files — see the "Migration runner" section above for the history of PR #212 and why this filter is load-bearing.

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
