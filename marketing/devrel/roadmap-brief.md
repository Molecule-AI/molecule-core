# Phase 30 + Roadmap Context Brief — DevRel

> **Sourced from:** `Molecule-AI/internal` — `PLAN.md` (via GitHub API, read-only token)
> **Purpose:** Keep DevRel aligned with roadmap so content and demos anticipate what's coming

---

## Phase 30: Remote Workspaces — What's Shipped

Phase 30 shipped 8 sub-features (30.1–30.8), all GA as of 2026-04-20:

| Sub-feature | What it does |
|---|---|
| 30.1 Workspace auth tokens | 256-bit bearer tokens, minted at registration. Prevents spoofing. |
| 30.2 Secrets pull endpoint | `GET /workspaces/:id/secrets/values` — gated by auth token |
| 30.3 Plugin tarball download | `GET /plugins/:name/download` — remote agent plugin install |
| 30.4 Workspace state polling | `GET /workspaces/:id/state` — fallback for agents behind NAT |
| 30.5 A2A proxy token validation | Mutual auth on `POST /workspaces/:id/a2a` |
| 30.6 Sibling discovery + URL caching | `GET /registry/{parent_id}/peers`, cache sibling URLs |
| 30.7 Poll-liveness for external runtime | 90s offline threshold, behind `REMOTE_LIVENESS_POLLING_ENABLED` |
| 30.8 Remote-agent SDK + docs | `sdk/python/examples/remote-agent/`, Python thin client |

**Out of scope for Phase 30:**
- Mutual TLS from agent → platform (deferred)
- Agent-to-agent mesh across NATs (needs relay — deferred to Phase 31)
- Platform-managed persistent state for remote agents

---

## Phase 31 — Quality + Infra Pass — SHIPPED 2026-04-13

Completed in PRs #1–#8:
- Brand migration (Molecule → Molecule AI)
- Repo structural cleanup
- MCP per-domain split (1697 → 89 lines, 87 tools)
- Canvas dialog unification
- Platform handler decomposition (+47 Go tests, coverage 56.1% → 57.6%)
- Env-var documentation (all 21 vars now documented)
- E2E hardening + CI (`test_api.sh` 62/62, `test_comprehensive_e2e.sh` 67/67)

---

## Phase 32 — Cloud SaaS Launch (2026-Q2/Q3) — IN PROGRESS

**Goal:** Ship Molecule AI as a multi-tenant cloud SaaS (not just self-hosted per-customer).

**Live infrastructure (as of 2026-04-15):**
- Control plane: `https://molecule-cp.fly.dev`
- Tenant app: `molecule-tenant` (Fly)
- Database: **Neon** serverless Postgres (branch-per-org)
- Cache: **Upstash** Redis
- Auth: **WorkOS AuthKit** (`/cp/auth/{signup,login,callback,signout,me}`)
- Billing: Stripe scaffold deployed (no live keys yet — pending Stripe Atlas)
- Registry: `registry.fly.io/molecule-tenant:latest`
- Domain: `moleculesai.app` (Cloudflare routing, DNS pending)
- First real tenant provisioned: org `acme`

**Phase status:**
- A — Foundation (accounts, tokens, domain) ✅
- B — Fly provisioner + Neon branching ✅
- C — WorkOS AuthKit scaffold ✅
- D — Stripe billing scaffold ✅ (live keys pending Stripe Atlas)
- E — Cloudflare + DNS + per-tenant Vercel canvas ✅
- F — Sign-up UX + onboarding ✅ (basic flow done; polish + email pending)
- G — Observability + quotas + admin ✅
- H — Hardening ⏳ partial (KMS envelope encryption ✅, tenant-isolation CI ✅, legal pages ✅; load test + Stripe Atlas + status page custom domain pending)
- I — Launch ⏳ pending Stripe Atlas (~2 week lead)

**Architectural decisions relevant to DevRel messaging:**
- **Open-core split:** `Molecule-AI/molecule-controlplane` (private) handles orgs/signup/billing/provisioner/routing. This public repo stays OSS (tenant binary + plugins + channels).
- **Firecracker > Docker socket:** Fly Machines API replaces raw Docker socket for multi-tenant isolation. Docker path stays for local dev only.
- **Companion repo:** `molecule-controlplane/PLAN.md` has the private roadmap.

**Tier 1 blockers before first external user:**
- Multi-tenancy: `org_id` filter on every row-returning handler
- Human auth + orgs via WorkOS (separate from Phase 30.1 agent bearer tokens)
- Container isolation via Fly Machines (Firecracker microVMs)
- Stripe billing (subscriptions + usage metering)
- Per-org resource quotas
- Managed Postgres (Neon) + Redis (Upstash)
- Secrets at rest via AWS/GCP KMS
- Migration runner extraction (goose as release step)

---

## Upcoming: Phase 33+

**What to watch for:** The backlog (PLAN.md) lists:
- Canvas: Org template import, Workspace search (Cmd+K), Batch operations
- Sandbox: Firecracker/E2B backends
- SDK follow-ups: live tool-call visibility, cost telemetry, cancel UX
- Real webhook mode for channels (webhook vs. polling)
- More channel adapters: Slack (OAuth), Discord (Bot + Gateway), WhatsApp

---

## Known Issues (from `known-issues.md`)

Three issues tracked internally, not yet filed as GitHub issues:

**KI-001 — Telegram `kicked` event doesn't persist disabled state**
- File: `telegram.go:596`
- Severity: Medium
- When the bot is removed from a chat, it keeps retrying sends indefinitely
- Fix: set `enabled = false` on `workspace_channels` row

**KI-002 — Delegation system has no idempotency guard**
- File: `delegation.go`
- Severity: Medium
- Container restart mid-delegation → double execution risk
- Fix: add optional `idempotency_key` to `POST /workspaces/:id/delegate`

**KI-003 — `commit_memory` not surfaced in `activity_logs`**
- File: `memory.py` + `activity.go`
- Severity: Low (debugging quality)
- Memory writes invisible in Canvas "Agent Comms" tab
- Fix: emit `activity_log` entry of type `tool_call` for `commit_memory`

---

## Backlog Highlights for DevRel

The backlog has direct marketing angles:

1. **Canvas: Org template import** — no-code org deployment from Canvas UI (Phase 20.3)
2. **SDK follow-ups** — cost telemetry + live tool-call visibility → enterprise governance story
3. **Delegations list endpoint** — `GET /workspaces/:id/delegations` returns `[]` while `check_delegation_status` shows active. One source of truth needed.
4. **Per-agent repo access** — `workspace_access: none|read_only|read_write` in `org.yaml` — eliminates the "PM couriers documents to reports" workaround
5. **SDK executor stderr swallowing** — every CLI failure is opaque; fix captures stderr, includes first ~1 KB in A2A error response. High priority per PLAN.md.

---

## Ecosystem Watch

`docs/ecosystem-watch.md` is the canonical starting point for research agents doing competitive analysis. Notable projects to track: Holaboss, Hermes, gstack, Letta, Trigger.dev.

---

*Update this doc after token refresh — check PLAN.md for Phase 32 content.*
