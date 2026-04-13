# Handoff to fresh Claude Code session — Molecule AI / `molecule-monorepo`

You're picking up where the previous session left off. The project just rebranded from "Starfire" (public hackathon repo) to "Molecule AI" (private commercial repo at `github.com/Molecule-AI/molecule-monorepo`). This handoff is the previous session's accumulated context — memory entries, operational rules, and current state.

---

## 1. Who the user is

- **Hongming Wang** — solo founder + automation engineer at a Vancouver renovation business (Reno Stars). Two GitHub accounts: `HongmingWang-Rabbit` (main) and `airenostars` (Reno Stars business). Same person.
- **Working style:** terse, direct. Wants you to get to the point. Doesn't like filler. Will tell you when you're being too cautious.
- **Dual hat:** founder of Molecule AI (this product) + customer of Molecule AI (uses `org-templates/molecule-dev/` and `org-templates/reno-stars/` to dogfood his own product against his renovation business). The reno-stars template = real revenue work. Treat with care.

---

## 2. Project state right now (2026-04-13)

### Repo identity
- **Public hackathon repo (frozen):** `github.com/ZhanlinCui/Starfire-AgentTeam` — BSL 1.1, still public, NOT archived yet. Will likely be archived once the new repo is fully validated.
- **New private commercial repo:** `github.com/Molecule-AI/molecule-monorepo`
- **Local path:** `/Users/hongming/Documents/GitHub/molecule-monorepo`
- **License:** BSL 1.1, Licensor "Molecule AI", auto-converts to Apache 2.0 on 2029-01-01. Additional Use Grant prohibits competing products in the "organizational control plane for heterogeneous AI agent teams" space.

### Brand mapping (already done — do NOT redo)
| Old | New |
|---|---|
| `Starfire` / `starfire` / `STARFIRE` | `Molecule AI` / `molecule` / `MOLECULE` |
| `Agent Molecule` / `agent-molecule` | `Molecule AI` / `molecule` |
| `agent_molecule_status.py` | `molecule_ai_status.py` |
| `org-templates/starfire-dev/` | `org-templates/molecule-dev/` |
| `org-templates/starfire-worker-gemini/` | `org-templates/molecule-worker-gemini/` |
| `plugins/starfire-dev/` | `plugins/molecule-dev/` |
| `sdk/python/starfire_plugin/` | `sdk/python/molecule_plugin/` |
| `sdk/python/starfire_agent/` | `sdk/python/molecule_agent/` |
| Go module: `github.com/agent-molecule/platform` | `github.com/Molecule-AI/molecule-monorepo/platform` |
| MCP package: `@starfire/mcp-server`, binary `starfire-mcp` | `@molecule-ai/mcp-server`, binary `molecule-mcp` |
| Postgres DB: `agentmolecule` | `molecule` |
| Env vars: `STARFIRE_*` | `MOLECULE_*` (full rename, NO backward-compat shim) |

**One name preserved intentionally:** `starfire-test-plugin` — that's a real external GitHub repo (`HongmingWang-Rabbit/starfire-test-plugin`) used to validate the github:// plugin install path. Do NOT rename references to it.

### Verified green on the new repo (last verified ~30 min before this handoff)
- `cd platform && go test -race -count=1 ./...` — all packages
- `cd workspace-template && python3 -m pytest` — **1129 passed, 9 skipped, 2 xfailed**
- `cd sdk/python && python3 -m pytest` — **132 passed**
- `cd canvas && npm test -- --run` — **352 passed (18 files)**
- `cd canvas && npm run build` — clean
- `cd mcp-server && npm run build` — clean
- Platform server boots, `/health` returns `{"status":"ok"}`

### Docker images
6 of 8 rebuilt fresh against the new repo: `workspace-template:base`, `:claude-code`, `:langgraph`, `:deepagents`, `:autogen`, `:hermes`. **`openclaw` and `crewai` may still be building when you start** — check `tail /tmp/molecule-build.log` and `docker images | grep workspace-template` to confirm. They're heavy (3-5 GB each, 5+ min builds).

### Infra
- Old `starfire-agentteam-*` containers were stopped during migration.
- New infra is running: `docker compose -f docker-compose.infra.yml ps` should show postgres / redis / langfuse all healthy.
- All 21 migrations applied to the fresh `molecule` DB.
- DB has zero workspaces and zero secrets — fresh start.

### Open PRs / issues
- **Open PRs on new repo:** 0 (just initial commit)
- **Open issues:** 0
- **Open PRs on old public repo:** 0 (all resolved before migration)

---

## 3. How the user works with you (the CEO's standing rules)

These are accumulated feedback memories from prior sessions — read all of them, they're load-bearing.

### Git workflow
- **Never push directly to `main`.** Always create a `feat/...`, `fix/...`, `chore/...` branch and PR. (One exception during the migration: the user explicitly OK'd direct pushes to the old repo's main for the housekeeping commits because we were about to leave that repo.)
- **Merge with `--merge` only.** Never `--squash`, never `--rebase`. Preserves commit attribution.
- **You MAY merge PRs autonomously** if you personally verified all of: (1) CI green, (2) line-level review clean, (3) design-philosophy fit, (4) security review clean, (5) actual full tests run by you (not "tests exist" — "I ran them just now and they passed"). Wait for CEO approval ONLY for noteworthy cases: ambiguous design call, irreversible migration, large blast radius, anything touching auth/billing/data deletion.
- **Never commit without explicit user approval** (separate from merge — refers to authoring commits in the working tree).
- **Loop "skip" must comment.** If hourly maintenance (the `/loop` skill) skips a PR, the FIRST skip per session must leave a PR comment with the specific blocker. Silent skips strand PRs indefinitely.

### Testing discipline
- **Manual browser/E2E testing required**, not optional. Unit tests + green CI ≠ working feature. Use Chrome MCP (`mcp__claude-in-chrome__*`) or Playwright (canvas/playwright.config.ts is set up) for any UI-touching change. If both are unavailable, **STOP and report** — don't claim it's verified.
- **E2E tests must verify data flow**, not just UI structure. "Button exists" passes when the feature is broken. Test: create real data → wait → verify content renders.
- **Test long-lived state.** If a feature spawns a goroutine in a CRUD handler, write a test that triggers the spawn, cancels the spawning request context, then asserts the goroutine is STILL alive. Don't pass `c.Request.Context()` to long-lived goroutines.
- **Reload + restart before reporting "done."** After ANY platform Go change or canvas TypeScript change: rebuild → kill old → start new → manually verify on the running service. Telling the user "done" with a stale binary running has happened repeatedly and is unacceptable.

### Architecture / philosophy
- **Multi-agent**, not single-agent. Per-workspace isolation. A2A for sibling communication. Memory as files. Runtime-agnostic plugins. Hierarchy-based access control (CanCommunicate in `registry/access.go`).
- **Always delegate through PM.** Never bypass hierarchy by sending A2A directly to Frontend Engineer / QA / Dev Lead. CEO → PM → team. This is the platform's value proposition; bypassing PM defeats the point.
- **Only PM mounts the repo.** PM gets `workspace_dir` bind-mount; all other agents get isolated Docker named volumes for `/workspace`. Don't set the global `WORKSPACE_DIR` env var.
- **Cross-reference new docs.** When adding a top-level doc under `docs/`, wire it into `PLAN.md` + `README.md` (+ `README.zh-CN.md` mirror) + `CLAUDE.md`. A doc not linked from those three is invisible to agents.
- **No native browser dialogs.** Never use `confirm()`, `alert()`, or `prompt()` in canvas code. Use the `ConfirmDialog` component at `canvas/src/components/ConfirmDialog.tsx`.

### Operational discipline
- **Check provisioning failures.** If a workspace is stuck in "provisioning" >30s, run `docker logs ws-<id>` and diagnose. Never report "still provisioning, will be online shortly" without verifying.
- **Monitor infra while team works.** When agents are delegated work, your job is infra monitoring (heartbeats, delegation chains, container health, activity logs) — not micromanaging their implementation.
- **Report monitoring findings.** Don't run silent background loops. After each check: brief summary. Even "13/13 online, no issues" is fine. Never `run_in_background` and forget.
- **Coordinate with PM.** Before significant work: A2A check-in with PM. After completing: share results so PM can update backlog and inform the team.
- **`.awareness/` is gitignored.** Local agent state, never tracked. Already covered by `.gitignore`. If you ever see `git ls-files .awareness/` return rows, `git rm --cached -r .awareness/` and commit.

---

## 4. Operator PII situation (read this before doing anything with reno-stars)

The `org-templates/reno-stars/` template was scrubbed of PII just before migration. Real values were replaced with env-var references:

| Var | What it is |
|---|---|
| `OPERATOR_EMAIL` | Operator's contact email |
| `OPERATOR_PHONE` | Display only |
| `OPERATOR_TELEGRAM_ID` | Numeric Telegram user ID |
| `GADS_MCC_ID` | Google Ads MCC account |
| `GADS_CUSTOMER_ID` | Google Ads child account |
| `GCP_PROJECT_ID` | GCP project |
| `GSC_SERVICE_ACCOUNT` | Search Console reporter service account email |

The user must set these as **global_secrets** via the canvas, API (`PUT /settings/secrets`), or MCP (`mcp__starfire__set_global_secret`) for the reno-stars org to work. The platform auto-injects every global_secret as a container env var. See `org-templates/reno-stars/OPERATOR_NOTES.md` for instructions.

---

## 5. Things that are NOT done yet (what the user might ask you about next)

1. **Set operator global_secrets in the new platform.** DB is fresh — zero secrets. Reno-stars won't function until these are populated. The values exist in the user's head / old DB / `org-templates/molecule-worker-gemini/.env`.
2. **Switch live Reno Stars business deployment to point at the new repo + new infra.** Old infra was stopped during migration. If the business automations were running there, they're down right now until you redeploy from the new repo.
3. **Archive the old public repo.** Up to the user when. Recommendation: leave public, archive with a "see new repo" notice once new repo is fully validated.
4. **Consider extracting a sanitized "Renovation Business" reference template** at `org-templates/examples/renovation-saas/` as a customer-facing starter kit. Optional product play discussed but not built.
5. **`openclaw` + `crewai` Docker images may still be rebuilding** — check status when you start.
6. **No CI configured yet on the new repo.** The old repo's GitHub Actions workflows should have copied over (`.github/workflows/`), but they reference the old repo URL in some places. Worth a quick audit.
7. **The `MEMORY.md` index in the previous session's auto-memory** lives at `~/.claude/projects/-Users-hongming-Documents-GitHub-Starfire-AgentTeam/memory/`. The new session under `~/.claude/projects/-Users-hongming-Documents-GitHub-molecule-monorepo/` will start fresh. Worth re-saving the load-bearing rules from §3 above as memory entries in the new project.

---

## 6. Useful commands / paths in the new repo

```bash
# Local repo
cd /Users/hongming/Documents/GitHub/molecule-monorepo

# Build + test sweep
cd platform && go build ./... && go test -race ./...
cd workspace-template && python3 -m pytest -q
cd sdk/python && python3 -m pytest -q
cd canvas && npm test -- --run && npm run build
cd mcp-server && npm run build
bash workspace-template/build-all.sh   # rebuild Docker images

# Infra
bash infra/scripts/setup.sh    # postgres + redis + langfuse + migrations
bash infra/scripts/nuke.sh     # tear down (warn user — wipes volumes)

# Run platform locally
cd platform && go run ./cmd/server   # port 8080

# Run canvas
cd canvas && npm run dev   # port 3000

# Health check
curl http://localhost:8080/health
```

---

## 7. The hourly maintenance loop (`/loop`)

The previous session was running an hourly PR-triage + issue-pickup loop. The cron job (`63a71b1f`) was session-only and died when that session ended. If the user wants it on the new repo, they'll re-invoke `/loop` with the same prompt.

The loop's full prompt is preserved in the conversation history. Key discipline rules baked in:
- STEP 0.5 ambiguity: when blocked, always **comment on the PR before skipping** (memory: `feedback_loop_skip_must_comment.md`)
- Use verified-merge (memory: `feedback_no_merge_pr.md`) — don't bottleneck on CEO approval if all 5 verification boxes are ticked
- Merge-commit only (memory: `feedback_merge_commits.md`)

---

## 8. Recommended first 5 minutes when you start

1. `cd /Users/hongming/Documents/GitHub/molecule-monorepo && git status && git log --oneline -3` — confirm clean state
2. `docker images | grep workspace-template` — confirm 8 fresh images (or report which still need rebuild)
3. `docker compose -f docker-compose.infra.yml ps` — confirm infra healthy
4. `curl -s http://localhost:8080/health` (if platform is running) or skip — platform may not be running unattended
5. Save the load-bearing feedback rules from §3 as memory entries in the new project's memory dir so they persist across sessions in this repo too
