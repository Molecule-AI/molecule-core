# Ecosystem Research Outcomes

**Input:** [`docs/ecosystem-watch.md`](./ecosystem-watch.md) — Holaboss
(`holaboss-ai/holaboss-ai`), Hermes Agent (`NousResearch/hermes-agent`),
gstack (`garrytan/gstack`).

**Method:** Molecule AI-dev team coordination — PM fan-out to Research Lead
(3 analysts) and Dev Lead (6 engineers). Full research corpus archived
under `/tmp/eco_research/` during synthesis; raw outputs are what the
team actually said. Cross-referenced against real repo files before
listing any file path in this doc.

**Date:** 2026-04-12

---

## Top-5 platform improvements (prioritized)

Ranking is by convergence across team members + impact for the hours
spent. All effort tags are S (≤1 day), M (1–3 days), L (≥1 week).

### 1. Memory: Postgres FTS + namespace scoping — **S, high impact**

Replace the `content ILIKE '%q%'` sequential scan in
`workspace-server/internal/handlers/memories.go:Search` with a `tsvector`
generated column, GIN index, and `ts_rank` ordering. Add a
`namespace VARCHAR(50) DEFAULT 'general'` column plus the
`(workspace_id, namespace)` composite index. Ship as migration
`workspace-server/migrations/017_memories_fts_namespace.sql`. Purely
additive — old rows get `namespace = 'general'`, new query params
(`?q=`, `?namespace=`) are optional, no breaking change.

Converged across Backend, QA, Frontend, UIUX — everyone proposed
some flavour of this. Combines Hermes's FTS5 recall pattern with
Holaboss's `knowledge/{facts,procedures,blockers,reference}/`
namespace model. Canvas can render namespace-grouped accordions
against the same endpoint with zero backend changes after day one.

**Ecosystem citations:** Hermes — "FTS5 + LLM-summarization for
cross-session recall — cheap, no vector-store overhead"; Holaboss —
filesystem-as-memory hierarchy.

### 2. Workspace hibernation: idle watchdog + auto-pause — **M, DevOps win**

DevOps Engineer's proposal: add a `_idle_watchdog` background job
in `workspace/entrypoint.sh` that reads `/tmp/.last_activity`
(written by `main.py` on each A2A request) and calls the existing
`POST /workspaces/:id/pause` after `IDLE_SHUTDOWN_MINUTES` (default
30). Platform's existing liveness monitor handles resume on next task;
no new Go code required — this is a shell + one `main.py` line + an
env var. Enables Hermes-style serverless-ish behaviour for agents
that only wake for cron audits (Security Auditor, QA Engineer).
Pairs naturally with Proposal 3 below.

**Ecosystem citation:** Hermes — Daytona / Modal serverless backends
with hibernation.

### 3. Parallel adapter builds — **S, QoL**

`workspace/build-all.sh` builds the 6 adapter images
sequentially (~15 min wall-clock). They all `FROM
workspace-template:base` with no inter-adapter dependency — swap the
Step 3 loop for background jobs + `wait`, log each build to
`/tmp/build_<tag>.log`. Cuts total build time to ~5–7 minutes.
Prerequisite for hibernate/wake feeling snappy (Proposal 2).

### 4. Plugin manifest: permissions + version floor + config schema — **S, spec-alignment**

Extend `pluginInfo` in `workspace-server/internal/handlers/plugins.go`
with `permissions []string` (e.g. `env:GITHUB_TOKEN`,
`path:/workspace/repo`, `docker:CAP`), `min_platform_version`
(semver floor enforced at install time when `PLATFORM_VERSION`
env is set), and `config_schema json.RawMessage` (stored raw so
canvas can render a form without re-parsing). All three are
additive — missing keys unmarshal to zero values. Positions
Molecule AI ahead of the agentskills.io spec picking up
permissions semantics, and mirrors Holaboss's
`workspace.yaml`-forces-prompts-into-AGENTS.md principle
(config stays machine-readable).

**Ecosystem citations:** Hermes — "if `agentskills.io` spec picks
up mass adoption → align our plugin manifest"; Holaboss —
`workspace.yaml` rejects inline prompt bodies.

### 5. Fail-secure encryption at boot — **S, security critical**

Security Auditor's top proposal. Today `SECRETS_ENCRYPTION_KEY`
is optional — when unset, the platform boots and silently falls
back to storing secrets in plaintext. Flip to fail-secure: if the
binary is built with `go build -tags prod` (or `MOLECULE_ENV=prod`
is set), refuse to start without a 32-byte key and log a loud
abort. Dev builds retain the current fallback with a startup
warning. Small, surgical change in `workspace-server/internal/crypto/aes.go`
+ `cmd/server/main.go` init; unit test already exists to verify
encryption path.

**Ecosystem citation:** gstack CSO persona — OWASP A02:2021
(cryptographic failures), STRIDE "Tampering / Information
Disclosure."

---

## Per-agent improvement proposals

Each of the 9 team members produced 2–3 concrete proposals mapped to
real file paths. Summary here; full proposals live in the captured
research (happy to expand any). Proposals adopted into Top-5 above are
marked ✅.

### Market Analyst (Holaboss axis, 3,164 chars)

- ✅ Structured filesystem memory layer (→ Top-5 #1)
- Compaction-boundary artifact for long-horizon single-agent mode —
  **defer** (we're multi-agent; only useful if we add a persistent
  PM-only mode).
- Section-based prompt assembly with per-section cache fingerprints —
  **consider** once Claude-SDK prompt-caching becomes a cost line item.

### Technical Researcher (Hermes axis, 3,874 chars)

- ✅ Nudge-to-persist memory pattern → exposed in UIUX Proposal 2 below.
- ✅ FTS5 recall (→ Top-5 #1).
- ✅ Daytona/Modal-style hibernation (→ Top-5 #2).
- Honcho dialectic user-modelling backend — **evaluate for PM role
  only**; too invasive to bolt onto every workspace.
- `hermes claw migrate` graceful-import pattern — **add to backlog**
  if we ever deprecate a runtime adapter.

### Competitive Intelligence (gstack axis, 2,974 chars)

- Weekly Retro Synthesis command (`/retro`) — **CEO-side skill, see
  below**.
- `/freeze`, `/guard`, `/unfreeze` architectural guardrails — see
  QA proposal 3.
- Lift CSO (OWASP + STRIDE) and Designer (AI-slop detection) role
  prompts into our Security Auditor and UIUX system-prompts as
  attributed additions — **S effort, high leverage**.

### Frontend Engineer (6,154 chars)

1. **Namespaced Memory Browser** — `canvas/src/components/tabs/MemoryTab.tsx`
   parses `namespace:key` naming into grouped accordions. Zero backend
   change for MVP; converges with BE Top-5 #1.
2. **"Save as memory" nudge in ActivityTab** —
   `canvas/src/components/tabs/ActivityTab.tsx` renders an inline
   "Save as memory →" link on `task_complete` and `skill_promo`
   events; clicks pre-populate the MemoryTab add form. Hermes
   closed-learning-loop pattern.
3. **[3rd proposal in full output]** — available on request.

### Backend Engineer (12,938 chars, the longest output)

1. ✅ Memory FTS + namespace (→ Top-5 #1)
2. ✅ Plugin manifest extension (→ Top-5 #4)
3. Schedule import/export via bundle system — **M**; currently
   `workspace_schedules` rows are orphaned on `bundles/export`. Small
   handler change in `workspace-server/internal/handlers/bundle.go`.

### DevOps Engineer (6,761 chars)

1. ✅ Idle watchdog auto-pause (→ Top-5 #2)
2. ✅ Parallel adapter builds (→ Top-5 #3)
3. Per-adapter CI stages (build + smoke-test each image in its own
   GitHub-Actions matrix job) — **M**; currently adapter images only
   get built locally.

### Security Auditor (8,875 chars, strongest deliverable)

1. ✅ Fail-secure encryption at boot (→ Top-5 #5)
2. Remove `test:*` from production `systemCallerPrefixes` — **S**.
   `workspace-server/internal/handlers/a2a_proxy.go:50` currently whitelists
   the literal prefix `test:` in every environment; it's an
   access-control bypass waiting to be exploited. Guard behind
   `MOLECULE_ENV != prod`.
3. Plugin supply-chain hardening — mandate `plugin.yaml` presence
   and reject staged trees containing executable bits (`+x`) outside
   `skills/*/hook.sh`. **S**; adds a preflight in
   `workspace-server/internal/plugins/localresolver.go`.

### QA Engineer (6,395 chars)

1. Filesystem memory namespace isolation (tests enforcing namespace
   separation) — **S**, complements Top-5 #1.
2. Autonomous skill-creation loop + FTS5 recall test suite — **M**;
   Hermes self-improvement pattern needs explicit coverage before
   landing.
3. Freeze / Guard / Unfreeze architectural guardrail tests — **M**;
   ports gstack's `/freeze` primitive as enforced test fixtures
   (e.g. a `/freeze` on the auth middleware fails CI if any handler
   modifies it without an override flag).

### UIUX Designer (14,285 chars, the longest engineering output)

1. Namespaced Memory Browser — same as FE proposal 1; the two
   should be implemented as one ticket, UIUX leads, FE executes.
2. Clickable Skill Promotion Nudge on node card — surfaces Hermes's
   skill-promotion pattern at the canvas-graph level. **S**.
3. Inline `initial_prompt` body warning in ConfigTab —
   `canvas/src/components/tabs/ConfigTab.tsx` flags when
   `initial_prompt:` has inline body text >200 chars with a
   "Extract to AGENTS.md" lint-style hint. **S**; encodes the
   Holaboss principle that config should stay machine-readable.

---

## CEO workflow improvements

Patterns to adopt at the **Claude Code CLI** layer (the CEO's
interface), not inside the Molecule AI platform itself.

### New skills to add under `.claude/skills/`

1. **`/retro`** — lifted directly from gstack. Generates a weekly
   retrospective by reading `git log --since='7 days ago' --oneline
   --shortstat`, the merged PR list, the set of closed issues, and
   the activity logs across the org. Outputs a markdown doc under
   `docs/retros/<YYYY-MM-DD>.md`. High leverage at near-zero cost;
   gstack validated the pattern at 70k⭐.
2. **`/freeze <path>`** — sets a repo-level flag (a file under
   `.claude/freezes/`) that any future code-review or edit skill
   reads. When the next CEO-driven change touches a frozen path,
   the edit skill blocks with a clear message. Adopted from
   gstack's `/freeze` / `/guard` / `/unfreeze` trio.
3. **`/verify-refs`** — explicit helper for the "verify before
   citing" discipline we encoded in the team's hardened prompts.
   Takes a draft message, finds `#NNN` / `sha:hex` / `path:` refs,
   runs `gh issue view`, `git log -1`, `cat` respectively, and
   reports any mismatches before the CEO sends.

### Settings / Hooks changes

- **Pre-tool hook on `Bash` commands that match `git push origin main`**
  — reject unless `FORCE_PUSH_MAIN=1` is exported. This session we
  caught ourselves (and PM) about to commit to `main` twice. A
  hook makes the rule programmatic.
- **Status line / telemetry counter** for MCP tool failures — so
  PR breakage from upstream MCP-client issues (e.g. #67) surfaces
  in the prompt, not only when we try to use it.

### Process / conventions

- When briefing PM on a fan-out task: always include the explicit
  workspace IDs and instruction to **inline documents** — even
  though this is now encoded in the hardened system prompts
  (PR #69), the reminder at task-issue time saves a round-trip.
- Treat `Agent error (ProcessError)` as a **platform bug**, not a
  transient failure. Restart the affected workspace, note the
  incident in the issue tracker referencing #66 and #71 until
  they land.

---

## Ecosystem signals to monitor (next quarter)

Items to watch on `docs/ecosystem-watch.md` and in the repos directly:

- **agentskills.io spec finalisation** — if the upstream spec
  locks in permissions semantics, our `plugin.yaml` should
  conform on the first release day. Today's Top-5 #4 positions
  us to lead rather than follow.
- **Hermes multi-agent / A2A addition** — would put us in direct
  overlap on the core thesis. Signal: any Nous Research blog
  post or commit matching `a2a|delegate|subagent_a2a`.
- **gstack parallel / multi-session** — if gstack ships
  simultaneous Claude Code workers + routing between them,
  their 70k⭐ head start converts into direct competition.
  Signal: any `/multi-*` command in the `garrytan/gstack` repo
  or a Garry Tan post showing it.
- **Holaboss → A2A** — Holaboss shipping workspace-to-workspace
  messaging would put them in the "AI company" shape we occupy
  today. Signal: a `workspace.yaml` `connections:` field or a
  `holaboss a2a` subcommand.
- **Atropos RL trajectory format** — if Nous standardises the
  schema for RL training-data export, our activity logs should
  adopt it so users can export Molecule AI runs for training.

---

## Explicit non-adoptions

Decisions to NOT copy, with reasons, so we don't revisit them:

- **Holaboss single-active-agent-per-workspace shape** — incompatible
  with our core thesis. Keep the concept of workspace-as-container
  but don't collapse to a single agent.
- **Hermes six-backend abstraction** (Docker / SSH / Daytona /
  Singularity / Modal) — our Docker provisioner is the right
  ceiling for v1. Serverless hibernation (Top-5 #2) buys us 80%
  of the cost win without the plumbing.
- **gstack's Claude-Code-native-only execution model** — gstack is
  a prompt library living inside one Claude Code session. Adopting
  that shape would eliminate our multi-agent / multi-runtime
  differentiation. We borrow specific role prompts, not the
  architecture.
- **`workspace.yaml` banning inline prompts at the schema level**
  — Holaboss rejects inline prompt bodies at parse time. We ship a
  UIUX *warning* instead (UIUX proposal 3) so existing templates
  keep working. The principle is right; the enforcement mechanism
  is too blunt for a platform that already has shipped templates
  out in the wild.
- **Compaction-boundary artifact** — solves long-horizon single-agent
  cost. We are multi-agent with per-workspace checkpointing already;
  this would be complexity for no direct gain.

---

## Process observations (meta)

Notes on how this coordination went that inform future runs:

1. **`#66` (opaque stderr) and `#71` (initial_prompt replay crash)
   are blocking team coordination.** Every fresh org import today
   started with ProcessError cascades. Until these land, any
   multi-agent research task requires host-side intervention
   (touching `.initial_prompt_done`, restarting crashed workspaces).
2. **`#65` (per-agent repo-access YAML) would eliminate the
   inline-documents workaround** that every Hard-Learned Rule we
   just added to the prompts tells the team to do. This is the
   single highest-leverage platform improvement on the list.
3. **Capturing raw analyst outputs from the activity log is a valid
   fallback** when PM crashes mid-synthesis. All 9 outputs in this
   doc were retrieved from `GET /workspaces/:id/activity` after the
   PM/RL/DL round-trip failed. Worth surfacing in platform docs
   as the "recovery" path.
4. **The hardened system prompts (PR #69) are already paying off**:
   Research Lead and Dev Lead both immediately fanned out in
   parallel with delegation IDs, rather than attempting solo
   synthesis. The "always fan out" rule is doing work.

---

## Next actions

Recommend proceeding in this order, each as its own PR:

1. **Ship #71 fix** (initial_prompt marker up-front) — unblocks all
   future fresh org imports.
2. **Ship #66 fix** (stderr capture) — restores debuggability.
3. **Top-5 #1** (memory FTS + namespace) — highest-convergence
   team ask, cleanest migration.
4. **Top-5 #5** (fail-secure encryption) — security-critical, trivial.
5. **CEO `/retro` skill** — near-zero effort, compounding weekly.

Everything else in this doc flows from there.
