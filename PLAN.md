# PLAN.md — Molecule AI Build Plan

> Completed phases (1–11, 13–14) are documented in `/docs` and removed from here.
> This file tracks only **in-progress and upcoming work**.

---

## Completed Phases (see /docs for details)

| Phase | Name | Docs |
|-------|------|------|
| 1 | Core Loop | `docs/architecture/architecture.md`, `CLAUDE.md` |
| 2 | E2E Validation | `CLAUDE.md` (build/test commands) |
| 3 | Hierarchy & Communication | `docs/api-protocol/communication-rules.md` |
| 4 | Provisioner | `docs/architecture/provisioner.md` |
| 5 | Agent Management | `CLAUDE.md` (API routes) |
| 6 | Bundle Export/Import | `docs/agent-runtime/bundle-system.md` |
| 7 | Team Expansion | `docs/agent-runtime/team-expansion.md` |
| 8 | Human-in-the-Loop Approvals | `docs/agent-runtime/system-prompt-structure.md` |
| 9 | Hierarchical Memory | `docs/architecture/memory.md` |
| 10 | Observability (Langfuse) | `docs/development/observability.md` |
| 11 | Canvas Polish & UX | `docs/frontend/canvas.md` |
| 13 | Runtime Enhancements | `docs/agent-runtime/workspace-runtime.md` |
| 14 | Production Hardening | `docs/architecture/provisioner.md`, `CLAUDE.md` |
| 15 | Per-Workspace Dir | PR #38 — `workspace_dir` per workspace |
| 16 | Plugin System | PR #39 — per-workspace plugins with registry |
| 17 | Agent GitHub Access | PR #40 — git/gh in images, GITHUB_TOKEN env |
| 18 | File Browser Lazy Loading | PR #37 — depth=1, path traversal protection |
| 19 | MCP Full Coverage | PR #40 — 52→54 tools (plugins, global secrets, pause/resume, org, delegation) |
| 20 | Canvas UX Sprint | PRs #4, #21, #39 — Settings Panel, Onboarding, Plugins UI, Pause/Resume |
| 21 | Claude Agent SDK Migration | PR #48 — `ClaudeSDKExecutor` replaces CLI subprocess |
| 22 | Cron Scheduling | PR #49 — recurring tasks via cron expressions, Canvas Schedule tab |
| 23 | Code Quality & Multi-Provider | PR #50 — model fallback, DeepAgents full SDK, 7 LLM providers, 100% test coverage |
| 24 | Async Delegation | PR #41 — non-blocking delegation with status polling, `check_delegation_status` tool |
| 25 | Social Channels | PR #54 — adapter-based Telegram integration, Canvas Channels tab, 7 MCP tools, hot reload, multi-chat IDs, auto-detect, /start auto-reply, full Telegram Bot API audit fixes |
| 26 | Auth Env Vars | PR #55 — `required_env` config replaces `.auth-token` files, env-var only path; reno-stars 15-agent org template |
| 27 | Channel Polish & Org Auto-link | PR #56 — poller lifetime fix (bgCtx), Restart Pending button (only when needed), org template `channels:` field auto-links Telegram on import |

---

## Phase 12: Code Sandbox — PARTIAL

> MVP done (subprocess + Docker backends). Production backends not started.

- [x] `run_code` tool — `tools/sandbox.py`
- [x] Docker-in-Docker backend (MVP) — throwaway container with resource limits
- [ ] Firecracker backend (production) — MicroVM isolation, faster cold starts
- [ ] E2B backend (cloud) — cloud-hosted via E2B API
- [x] Sandbox config — `SandboxConfig` dataclass in config.py

---

## Phase 20: Canvas UX Sprint — MOSTLY COMPLETE

> UX specs created by UIUX Designer agent. See `docs/ux-specs/` for full specs.

### 20.1 Settings Panel (Global Secrets UI) — DONE
**Spec**: `docs/ux-specs/ux-spec-settings-panel.md`

- [x] Gear icon in canvas top bar (Cmd+, shortcut)
- [x] Slide-over drawer (480px, right-anchored)
- [x] Service groups (GitHub, Anthropic, OpenRouter, Custom)
- [x] CRUD: add, view (masked), edit, delete secrets
- [x] Empty state with guided setup
- [x] Unsaved changes guard on close

### 20.2 Onboarding / Deploy Interception — DONE
**Spec**: `docs/ux-specs/ux-spec-onboarding-interception.md`

- [x] Pre-deploy secret check — detect missing API keys per runtime
- [x] Missing Keys Modal — inline form, only asks for what's needed
- [x] Provisioning timeout → named error state with recovery actions
- [x] No dead ends — every error has a fix action

### 20.3 Canvas UI Improvements — PARTIAL
**Spec**: `docs/ux-specs/ux-spec-canvas-improvements.md`

- [x] Plugins install/uninstall in Skills tab (PR #39)
- [x] Pause/resume from context menu
- [x] Org template import from canvas (PR — `OrgTemplatesSection` in TemplatePalette)
- [ ] Workspace search (Cmd+K)
- [ ] Batch operations

---

## Phase 30: SaaS — Remote Workspaces & Cross-Network Federation — IN PROGRESS

**Goal:** let a Python agent running on a laptop in another city boot,
register, authenticate, accept A2A from its parent PM on the platform,
and appear on the canvas as a first-class workspace.

**Why now:** the self-hostable single-box model has landed; the next
meaningful expansion is letting orgs span machines and networks. This
is the step that turns Molecule AI from "Docker-compose on one box" into
a multi-tenant SaaS-shaped product.

**Design thesis:** ride the existing `runtime='external'` escape hatch.
Every Docker-touching handler already short-circuits when a workspace
is external. We don't need a parallel subsystem — we need to close
four small gaps and add per-workspace auth. See
[`docs/remote-workspaces-readiness.md`](docs/remote-workspaces-readiness.md)
for the full code audit.

### Shipping order (eight bounded steps, ~2 weeks to GA)

- [x] **30.1 Workspace auth tokens** — foundation; prevents spoofing.
  New `workspace_auth_tokens` table; `POST /registry/register` issues
  a token; middleware validates `Authorization: Bearer <token>` on
  `/registry/heartbeat`, `/registry/update-card`. Lazy bootstrap so
  in-flight workspaces upgrade gracefully. Transparent to local
  containers — provisioner carries the token through the existing env-var
  pattern. No feature flag.

- [x] **30.2 Secrets pull endpoint** — `GET /workspaces/:id/secrets/values`
  returns decrypted secrets JSON, gated by the 30.1 token. Local agents
  can use it too (removes env-at-create coupling for rotating secrets).

- [x] **30.3 Plugin tarball download** — `GET /plugins/:name/download`
  returns a tarball; agent unpacks locally. Replaces Docker-exec plugin
  install for remote agents. Behind `REMOTE_PLUGIN_DOWNLOAD_ENABLED`.

- [x] **30.4 Workspace state polling** — `GET /workspaces/:id/state`
  returns `{status, paused, deleted_at, pending_events[]}` as a drop-in
  for the WebSocket feed remote agents can't reach. Behind
  `REMOTE_STATE_POLLING_ENABLED`.

- [x] **30.5 A2A proxy token validation** — the proxy enforces the caller's
  auth token on `POST /workspaces/:id/a2a`. Mutual auth between agents.

- [x] **30.6 Direct sibling discovery + URL caching** — agents call
  `GET /registry/{parent_id}/peers` once, cache sibling URLs, call them
  directly for A2A. Resilient to brief platform outages.

- [x] **30.7 Poll-liveness for external runtime** — `LivenessChecker`
  interface in `registry/`; `PollLiveness` marks offline if no heartbeat
  in 90s. Docker checker becomes one implementation, poll-liveness
  another. Health sweep routes by runtime. Behind
  `REMOTE_LIVENESS_POLLING_ENABLED`.

- [x] **30.8 Remote-agent SDK + docs** — `sdk/python/molecule_agent/`
  thin client: register → pull secrets → run A2A loop → poll state →
  heartbeat. Working `sdk/python/examples/remote-agent/` a new user can run on a
  laptop. Remove the three feature flags. Remote workspaces become GA.

### Out of scope for Phase 30

- Mutual TLS / platform-identity verification from the agent side.
  Agent trusts any platform URL in its env. Defer until real multi-
  tenant deployment forces the question.
- Agent-to-agent mesh across NATs. Direct sibling calls only work when
  siblings are reachable from each other. Behind-NAT ↔ behind-NAT needs
  a relay — defer to Phase 31.
- Platform-managed persistent state for remote agents. Remote agents
  own their filesystem; platform never mounts.

### Success criteria

- `sdk/python/examples/remote-agent/` boots on a laptop disconnected from the
  platform's LAN, registers, receives a task from parent PM via A2A,
  returns a result, appears on the canvas.
- `tests/e2e/test_federation.sh` spawns a second platform instance +
  remote agent pointing at the first; both platforms see the agent as
  a workspace in the right state.
- Spoofing test: attempt to impersonate a workspace with a guessed ID
  but no token → 401.

---

## Phase 31 — Quality + Infra Pass (Q2 2026) — SHIPPED 2026-04-13

Completed in PRs #1–#8 and documented in `docs/edit-history/2026-04-13.md`:

- [x] **Brand migration cleanup** — LICENSE "Agent Molecule" → "Molecule AI"; new icon assets (PR #1).
- [x] **Repo structural cleanup** — moved `examples/remote-agent/` → `sdk/python/examples/`, `docs/superpowers/plans/` → `plugins/superpowers/plans/`; deleted empty `platform/plugins/`; gitignored `.agents/`, `platform/workspace-configs-templates/`, `backups/`, `logs/`, `test-results/`; added READMEs under `tests/` and `docs/` (PR #3).
- [x] **MCP per-domain split** — `mcp-server/src/index.ts` 1697 → 89 lines; 12 per-domain modules in `src/tools/`; shared `src/api.ts`; startup log now reports 87 tools (PRs #2, #4, #7).
- [x] **Canvas dialog unification** — native `confirm()`/`alert()` replaced with `ConfirmDialog` in 7 sites; new `singleButton` prop + 5 tests (vitest 352 → 357).
- [x] **Platform handler decomposition** — 4 oversize functions (`proxyA2ARequest`, `Delegate`, `Discover`, `SessionSearch`) split into testable helpers; +47 Go tests; `handlers` coverage 56.1% → 57.6%.
- [x] **Env-var documentation** — `.env.example` gained 11 previously-undocumented vars; all 21 distinct `os.Getenv`/`envx.*` keys now documented.
- [x] **E2E hardening + CI** — Phase 30.1 bearer auth + Phase 30.6 `X-Workspace-ID` requirements baked into `test_api.sh` (62/62) and `test_comprehensive_e2e.sh` (67/67); shared `_lib.sh` + `_extract_token.py`; new CI jobs `e2e-api` and `shellcheck`; `setup-go` gains module cache (PRs #5, #7, #8).

---

## PR Workflow Rules

All PRs must follow this checklist:

1. **Branch**: Never push to main. Always create a feature/fix branch.
2. **Code Review**: Run `/code-review` skill and fix all issues before requesting merge.
3. **Tests**: All existing tests must pass. New features require new tests.
4. **Documentation**: Run `/update-docs` skill. Every PR must update:
   - `docs/edit-history/` session log
   - Relevant docs in `docs/` (API, architecture, frontend, etc.)
   - `CLAUDE.md` if routes, env vars, or commands changed
   - `PLAN.md` if the work completes a phase or adds new items
5. **E2E Test**: Rebuild, restart service, and manually verify before reporting done.
6. **QA Review**: QA Engineer reviews for edge cases, plan compliance, and documentation completeness before CEO merge approval.
7. **CEO Approval**: Only the CEO approves merges. Never merge without explicit approval.

---

## Ecosystem Awareness

Adjacent projects worth tracking (Holaboss, Hermes, gstack, …) are catalogued
in **[`docs/ecosystem-watch.md`](docs/ecosystem-watch.md)**. Skim quarterly,
add entries liberally, and when one of those projects ships something we
should react to, file a "Signals to react to" line in that doc and create a
Backlog entry below pointing at it. Agents doing research or strategy work
should read `docs/ecosystem-watch.md` first — it's the canonical starting
point for "what else is out there."

---

## Backlog (prioritized)

1. **Canvas: Org template import** — Phase 20.3 (deploy org from canvas UI)
2. **Canvas: Workspace search (Cmd+K)** — Phase 20.3 (quick find)
3. **Canvas: Batch operations** — Phase 20.3 (multi-select delete/restart)
4. **Sandbox: Firecracker/E2B backends** — Phase 12 (production isolation)
5. **NemoClaw adapter** — stub exists at `adapters/nemoclaw/`, no implementation yet
6. **Remote plugin registry** — install plugins from npm/git (currently local only)
7. **Agent git worktrees** — per-agent branches without full clone
8. **SDK follow-ups** — live tool-call visibility, cost telemetry, cancel UX, governance hooks
9. **Real webhook mode for channels** — Phase 27 candidate. Currently polling-only; webhook needs:
   - `mode: "webhook"|"polling"` config field
   - `PUBLIC_URL` env var
   - Platform calls `setWebhook` on channel create (with random `webhook_secret`), `deleteWebhook` on delete
   - Canvas toggle to enable webhook mode (only when PUBLIC_URL is set)
   - Polling works fine for ≤hundreds of bots; webhook needed at thousands+ scale or for serverless
10. **More channel adapters** — Slack (OAuth + Events API), Discord (Bot + Gateway), WhatsApp (Cloud API)
11. **Delegations list endpoint mismatch** — `GET /workspaces/:id/delegations` returns `[]` while the agent's internal `check_delegation_status` shows active/completed delegations. One source of truth.
12. **YAML-configurable per-agent repo access** — new `workspace_access: none|read_only|read_write` field in `org.yaml` + `:ro` bind-mount for research agents; eliminates the "PM couriers documents to reports" workaround.
13. **SDK executor swallows subprocess stderr** — `workspace-template/claude_sdk_executor.py` surfaces only "Command failed with exit code 1 / Check stderr output for details" when the `claude` CLI crashes, making every failure opaque. Capture stderr, log at ERROR, include first ~1 KB in the A2A error response. **High priority** — blocked real debugging during PLAN.md coordination on 2026-04-12.
14. **Agent MCP client defaults to `localhost:8080`** — inside a workspace container, `localhost` is the container itself, not the platform — so `mcp__molecule__*` tools fail with "platform unreachable." Inject `MOLECULE_URL=${PLATFORM_URL}` into every container at provision time and change the MCP client default to `http://host.docker.internal:8080`. **High priority** — blocks agents from calling platform tools (e.g. PM couldn't restart its own reports).

> Note: items 11–14 previously carried sequential refs `#64`–`#67`. Those refs were placeholder enumeration, not GitHub issues. They now collide with actual merged PRs and issues with different scopes, so the refs were removed in 2026-04-14 tick-5. If/when these items get prioritized, file real GitHub issues for them.
15. **Workspace `restart_prompt` — user-defined restart context (#19 Layer 2)** — GitHub issue **#66** (new 2026-04-14 tick-4 follow-up to PR #65 which shipped Layer 1). Let `config.yaml` / `org.yaml` declare a user-authored `restart_prompt` that is delivered alongside the platform-generated restart-context system message — e.g. "re-read your CLAUDE.md, re-hydrate TODOs from memory, resume the active delegation." Layer 1 (platform state snapshot) already ships; Layer 2 adds the user-defined side.

### Recently launched (2026-04-14 tick-4)
- **GitHub issue #15** — Provisioner: auto-refresh `CLAUDE_CODE_OAUTH_TOKEN` from `global_secrets` on workspace restart → **DONE** via PR #64 (`SetGlobal` / `DeleteGlobal` now fan out `RestartByID` to every affected workspace).
- **GitHub issue #19 Layer 1** — Platform-generated restart context → **DONE** via PR #65 (synthetic A2A `message/send` with `metadata.kind=restart_context`, `system:restart-context` caller prefix, 30s re-register wait). Layer 2 deferred to issue #66 (see Backlog item 15 above).

### Recently launched (2026-04-14 tick-8)
- **Phase 32 PR #1** — `TenantGuard` middleware (PR #78, merged `57a05686`). Public repo's only SaaS hook: when `MOLECULE_ORG_ID` env is set, non-allowlisted requests require matching `X-Molecule-Org-Id` header or 404. Unset → passthrough (self-hosted unchanged). Allowlist is exact-match: `/health` + `/metrics`. Paired with the private `Molecule-AI/molecule-controlplane` repo scaffolded this tick (Fly Machines provisioner stub, `/cp/orgs` CRUD, subdomain→fly-replay router, migrations 001-003 for `organizations`/`org_instances`/`org_members`). +6 `TestTenantGuard_*` tests. Phase 32 plan: follow-up PRs wire real Fly provisioner, WorkOS AuthKit, Stripe, Cloudflare, signup UX — all in the private repo except the single public middleware.

### Recently launched (2026-04-14 tick-7)
- **GitHub issue #24** — Runtime-added workspace_schedules drift on org re-import → **DONE** via PR #76 (new `source` column on `workspace_schedules` via migration `022`; org/import now upserts with `ON CONFLICT (workspace_id, name) DO UPDATE ... WHERE source='template'`, so runtime-added rows survive re-imports; legacy rows backfilled to `'template'`; +3 tests).
- **GitHub issue #51** — PM hardcoded audit-category routing → **DONE** via PR #75 (generic `category_routing:` block in `org-templates/<name>/org.yaml` `defaults` + per-workspace override; rendered into each workspace's `config.yaml` via `renderCategoryRoutingYAML` using `yaml.Node` + `yaml.Marshal` for safe escaping; PM prompt replaced with generic config-lookup; +6 tests).
- **PR #74** — `org-templates/molecule-dev/org.yaml` role overrides shrunk to just the deltas now that UNION semantics (PR #71) are in effect — removes verbose re-listing of defaults across PM, Research Lead, Research sub-roles, Security Auditor, UIUX Designer.

### Recently launched (2026-04-14 tick-6)
- **GitHub issue #68** — Per-workspace `plugins:` REPLACE semantics caveat → **DONE** via PR #71 (`mergePlugins` helper in `platform/internal/handlers/org.go` now UNIONs per-workspace with `defaults.plugins`; `!plugin` or `-plugin` prefix on a per-workspace entry opts a default out; +5 `TestPlugins_*` tests). Role overrides in `org-templates/*/org.yaml` can now declare just the delta instead of restating every default.

### Recently launched (2026-04-14 tick-5)
- **PR #70** — Wired the 12 modular plugins from PR #63 (tick-4) into the default `molecule-dev` org template. `defaults.plugins` expands from 3 → 9 (safety hooks + operational-memory skills become universal); PM role gains `molecule-workflow-triage` + `molecule-workflow-retro`, Security Auditor gains `molecule-skill-code-review` + `molecule-skill-cross-vendor-review` + `molecule-skill-llm-judge`. Verbose per-role re-listing is a consequence of REPLACE (not UNION) semantics in `platform/internal/handlers/org.go`; union-semantics proposal tracked as issue **#68**.
- **PR #69** — Backlog items 11–14 stripped of stale sequential refs `#64`–`#67` (see footnote near item 15 above).

---

## Test Coverage

| Stack | Tests | Framework |
|-------|-------|-----------|
| Go (platform) | 726 | `go test -race` (raw PASS lines incl. subtests; +6 top-level `Test*` this tick: #64 secrets auto-restart x2, #65 restart-context x4) |
| Python (workspace) | 1,140 | pytest |
| Canvas (frontend) | 357 | Vitest |
| SDK (python) | 132 | pytest |
| MCP server | 97 | Jest |
| **Total** | **2,452** | |

E2E: 67/67 comprehensive checks passing, 62/62 API tests (also gated in CI `e2e-api` job), shellcheck-clean across all 5 E2E scripts.

---

## Team Assignments

| Agent | Current Focus |
|-------|--------------|
| PM | Sprint coordination, backlog prioritization |
| Dev Lead | Engineering planning, PR review |
| UIUX Designer | UX specs for Phase 20 (DONE — 5 specs delivered) |
| Frontend Engineer | Phase 20.3 remaining items (org import, search, batch) |
| Backend Engineer | Sandbox production backends, API completeness |
| QA Engineer | **Review every PR for docs + plan compliance** |
| DevOps Engineer | CI/CD, Docker image optimization |
| Security Auditor | API key handling, path traversal, auth review |

---

## Next Steps

1. Frontend Engineer implements remaining Phase 20.3 items (org import from canvas, Cmd+K search)
2. Backend Engineer scopes Firecracker/E2B sandbox backends (Phase 12)
3. QA Engineer reviews PR #52 for docs compliance before merge
4. All agents use `GITHUB_TOKEN` env var to clone repo, branch, and create PRs

---

## Future Work — Plugin Adaptor System

Landed (see `feat/plugin-adaptor-registry` and `feat/agentskills-compliance`):
per-runtime plugin adaptors, hybrid resolver (registry > plugin-shipped >
raw-drop), `AgentskillsAdaptor` covering rule+skill plugins for all
runtimes, `/plugins?runtime=` filter, `/workspaces/:id/plugins/available`
endpoint, `molecule-plugin` SDK, gemini org parity with molecule-dev,
and **full agentskills.io spec compliance** for all first-party skills
(installable in Claude Code, Cursor, Codex, and ~35 other skill-compatible
tools — see `docs/plugins/agentskills-compat.md`).

Deferred, not blocking:

- **Upstream `runtime-adapters/` extension to agentskills.io spec** —
  once we've lived with our own per-runtime adapter model for ~month,
  propose it as a spec extension to `agentskills/agentskills` so other
  tools can share Molecule AI-authored adaptors.
- **Install-from-GitHub-URL flow** — `POST /plugins/install {git_url}` that
  clones a repo into the registry, validates the manifest, and runs the
  adaptor through a sandbox. Needs signature/version pinning and a review
  of the adaptor-execution threat model before shipping.
- **Promote-to-default UI** — today, promoting a community plugin to
  "curated" means manually copying its `adapters/<runtime>.py` into
  `workspace-template/plugins_registry/<plugin>/`. Later add a canvas
  button + PR template that opens an upstream PR automatically.
- **Plugin packs** — manifest that lists other plugins to bundle
  (`superpowers-pack` → install `superpowers-tdd` + `superpowers-debug` + …).
  Skip until a real user asks; first-party plugins are small enough to
  install individually today.
- **Hot-reload on DeepAgents** — upstream docs say skills/sub-agents are
  startup-only; would need platform-level container restart on plugin
  file change. Defer until users complain.
- **Atomic split of first-party plugins** — `superpowers` and `ecc` still
  ship as multi-skill bundles. Pipeline already supports splitting but
  non-urgent.
- **Sub-agent plugins for non-DeepAgents runtimes** — Claude Code /
  LangGraph don't have a native sub-agent feature; emulating via
  tool-routing is possible but invasive. Defer.
- **Workspace install tracking table** — a `workspace_plugin_installs`
  table would let uninstall call the adaptor's `uninstall()` path
  reliably. Today uninstall is a `rm -rf /configs/plugins/<name>` which
  leaves copied skill dirs behind. Low user impact.
- **Shared org-template `system-prompt.md` via `_shared/`** — DRY molecule-dev
  and molecule-worker-gemini. Drift risk; revisit at 3+ orgs.

## Phase 32 — Cloud SaaS launch (2026-Q2/Q3)

Goal: ship Molecule AI as a multi-tenant cloud SaaS (not just
self-hosted per-customer). Ordered by dependency + ROI.

### Tier 1 — blocks multi-tenant launch

- [ ] **Multi-tenancy**: `organizations` table, `org_id` FK +
  `WHERE org_id = $caller_org` filter on every row-returning
  handler (`workspaces`, `workspace_secrets`, `global_secrets`,
  `activity_logs`, `structure_events`, `agent_memories`,
  `workspace_schedules`, `workspace_channels`). Middleware resolves
  caller's org from session token → ctx. Full security audit of
  tenant isolation before first external user.
- [ ] **Human auth + orgs**: **WorkOS AuthKit** (NOT build-yourself,
  NOT Clerk — WorkOS treats per-org SSO as first-class; Clerk
  treats it as an upsell). Keep Phase 30.1 bearer tokens for
  machine-to-machine (agents). Stripe integration via WorkOS hooks.
- [ ] **Container isolation**: replace raw-Docker-socket provisioner
  with **Fly Machines API** (Firecracker microVMs, per-workspace
  isolation, sub-second boot, pay-per-second). Today's shared
  `/var/run/docker.sock` is an RCE-to-host footgun that cannot ship
  multi-tenant. `provisioner` interface stays — only backend swaps.
  Docker path remains for local dev.
- [ ] **Stripe billing**: subscriptions + usage metering
  (workspace-hours, LLM-token pass-through, storage), trial flow,
  dunning, invoices.
- [ ] **Per-org resource quotas**: tier memory/CPU is configurable
  (PR #58) but unenforced at provision time. Add per-org ceilings:
  max workspaces, max concurrent-running, max total memory.
- [ ] **Managed Postgres + Redis**: move off `docker-compose` for
  prod. **Neon** (serverless, branch-per-PR) for Postgres; **Upstash**
  for Redis. Alternative: drop Redis entirely — `LISTEN/NOTIFY`
  + advisory locks cover heartbeat TTL + URL cache.
- [ ] **Secrets at rest via KMS**: current `SECRETS_ENCRYPTION_KEY`
  is a single static AES-256 key. Move to **AWS/GCP KMS**-backed
  envelope encryption; the `secrets_encryption_version` table slot
  is already reserved for rotation.
- [ ] **Migration runner out of app boot**: a bad migration
  currently crashes platform boot with no rollback. Extract to
  **goose** as a release step / init container. Auto-discovery
  runner stays for dev mode only.

### Tier 1 follow-ups (before customer #1)

- [ ] **Observability**: wire `/metrics` to a scraper (Grafana
  Cloud or self-hosted). Add **Sentry** for Go + Next.js error
  tracking. Langfuse stays for LLM traces.
- [ ] **Rate limiting per-org**: global `RATE_LIMIT=600/min` is a
  shared bucket today. Needs per-org + per-endpoint buckets.
- [ ] **Cloudflare in front**: WAF + CDN + DDoS. Free tier covers
  pre-revenue.
- [ ] **Sign-up / onboarding flow**: landing → signup → first
  workspace in 60 seconds. No such flow today.
- [ ] **Transactional email**: Resend or Postmark.
- [ ] **Admin panel**: view orgs, suspend accounts, see usage,
  issue refunds. SQL-only at first; UI by ~50 orgs.
- [ ] **Privacy policy + ToS + DPA**: real ones, vetted. GDPR /
  CCPA data-export + deletion endpoints (workspace-export already
  exists; need org-level).

### Tier 2 — tech-stack upgrades (high ROI, non-blocking)

- [ ] **Go platform**: migrate `lib/pq` → **pgx/v5** (1–2 days;
  `lib/pq` in maintenance since ~2021). Then **sqlc** incrementally
  for new queries — keeps the no-ORM philosophy + typed Go.
- [ ] **Platform async: River** (Postgres-backed, Go-native job
  queue). Delegation dispatch, `workspace_schedules` cron, future
  billing events + webhook fan-out all migrate cleanly. **NOT**
  Temporal — Temporal already ships in workspace-template as an
  agent tool; keep the separation.
- [ ] **Frontend: TanStack Query** for server state. Zustand keeps
  pure UI state. Stops reimplementing cache / refetch / dedup. WS
  updates flow via `qc.setQueryData`. Single highest-ROI frontend
  refactor.
- [ ] **Turbopack for `next build`**: one flag, 2–5× cold-build
  speedup.
- [ ] **Python workspace runtime → uv**: `uv pip install` in
  `entrypoint.sh` cuts workspace cold-start 10–100×. User-visible
  latency win.
- [ ] **Python MCP client inside runtime**: today `mcp-server/`
  exposes the platform as an MCP server; agents inside workspaces
  can't yet consume external MCP servers. Closing the gap joins
  the winning 2026 ecosystem.
- [ ] **shadcn/ui CLI convention**: already Radix + Tailwind;
  adopt `npx shadcn add …` passively for new components.
  No rewrite.

### Tier 3 — explicitly NOT doing

- **Kubernetes**: company-of-one cannot run K8s. Fly Machines
  covers isolation without the ops tax.
- **ORM** (GORM / ent / bun): raw-SQL + sqlc covers every case.
- **Framework swap** (Next → Vite / TanStack Start): 2-week
  rewrite buys nothing users see.
- **Auth-from-scratch**: every hour on auth is an hour not on
  product.
- **Canvas library swap** (xyflow → tldraw): xyflow is still the
  correct tool for typed node graphs.

### Tier 4 — compliance / enterprise (when revenue lands)

- [ ] SOC 2 via Drata / Vanta
- [ ] Status page (Betterstack or Instatus)
- [ ] Staging environment that mirrors prod
- [ ] Blue-green / canary deploy pipeline
- [ ] Per-org backup + point-in-time restore
- [ ] Load testing (`hey` / `vegeta`) — current per-node ceiling
  unknown

### Success criteria for Phase 32

- Customer can sign up at moleculesai.app, create an org, deploy their
  first workspace, send their first message in < 5 minutes.
- Two orgs on the same cluster cannot observe each other's
  workspaces, secrets, memory, or activity — verified by automated
  tenant-isolation test + manual red-team.
- Fly Machines cost per active workspace-hour documented and
  reproducible.
- Stripe-backed subscription + usage-based add-ons working end-to-
  end in sandbox.
- One paying design partner on the cluster, paying a real invoice.

---

## Infra footnote — Temporal

`docker-compose.infra.yml` now includes Temporal (`:7233` gRPC, `:8233` Web
UI) backing `workspace-template/builtin_tools/temporal_workflow.py` for
durable long-running agent workflows. All infra services share the
`molecule-monorepo-net` Docker network, which `infra/scripts/setup.sh`
creates idempotently. Temporal currently runs with **no auth** on
`0.0.0.0:7233` — dev-only; any production deployment must front it with
mTLS, API keys, or a reverse proxy before exposing the cluster.
