# Triage Operator — Handoff Notes (2026-04-16)

Snapshot taken at handoff from the prior operator (Claude Opus 4.6, 1M context, ~100 tick session). Read this once, then discard — it's a point-in-time dump, not a running doc.

---

## What shipped this session (merge log, for audit)

**Platform monorepo** (merged to `main`):

| PR | Fix | Severity |
|----|-----|----------|
| #317 | `hitl.py` workspace-ID ownership + `security_scan.py` fail-closed + caught `SkillSecurityError` kwargs bug via regression test | LOW+LOW |
| #326 | `WorkspaceAuth` fake-UUID fail-open fix (Phase 30.1 grace-period kept) | HIGH |
| #327 | `channel_config` bot_token + webhook_secret AES-256-GCM encryption (ec1: prefix scheme, lazy migration) | MEDIUM |
| #330 | Wired `molecule-compliance` + `molecule-audit` + `molecule-freeze-scope` to Security Auditor / Backend / QA / DevOps | config |
| #331 | New `docs/glossary.md` — terminology disambiguation table (9 terms + near-miss section) | docs |
| #335 | `PausePollersForToken` scoped to requesting workspace (cross-tenant decrypt fix) | MEDIUM |
| #338 | `/transcript` fail-closed on missing token; extracted `transcript_auth.py` for testability | HIGH |
| #341 | Self-hosted Mac runner: `credsStore: ""` explicit to avoid osxkeychain bindings | CI |
| #343 | `webhook_secret` constant-time compare (`subtle.ConstantTimeCompare`) | LOW |
| #346 | Security Auditor prompt drift: added #319 + #337 checks to system prompt + 12h cron | chore |
| #357 | Remove `WorkspaceAuth` tokenless grace period entirely (strict bearer required) | HIGH |
| #370 | Engineer idle-loops (proactive issue pickup) — CEO-confirmed directive | template |

**Control plane** (merged to `main`):

| PR | Fix |
|----|-----|
| #35 | Session cookie stores refresh_token instead of OAuth code (auth-blocker) |
| #36 | Auto-apply embedded migrations on boot (migrations 006, 007 ran for the first time in prod) |
| #37 | Reserved subdomain list expanded from 9 entries to 341 across 12 categories |

**Live deploys:**
- `app.moleculesai.app` on Fly (v38 with all three CP PRs)
- `api.moleculesai.app` migration in-flight (DNS done, WorkOS dashboard done, `WORKOS_REDIRECT_URI` flipped at 06:06Z, user verifying end-to-end)
- `status.moleculesai.app` (Upptime on GitHub Pages) — unchanged from earlier session
- Stripe test-mode webhook + products + prices live on molecule-cp
- `CP_ADMIN_USER_IDS=user_01KPA3Z3810QEF3HCKRXP2EED9` (CEO's WorkOS user)

---

## What's in-flight that the next operator inherits

### 1. `app.moleculesai.app` grace period

After the CEO confirms `api.moleculesai.app` works end-to-end (login + admin endpoints), the OLD `app.moleculesai.app` subdomain needs to be dropped:

- Fly: `fly certs delete app.moleculesai.app -a molecule-cp`
- WorkOS dashboard: remove `https://app.moleculesai.app/cp/auth/callback` from allowed redirect URIs
- Cloudflare DNS: delete the `app` CNAME record

**Do NOT do any of this until the CEO confirms the new domain works.** 24–48h grace period minimum. If an active session still references the old cookie domain, dropping too early breaks their login.

### 2. Zombie workspace row (#367)

The Security Auditor agent filed #367 claiming `ffffffff-ffff-ffff-ffff-ffffffffffff` still returns 200 on unauth `/secrets`. My analysis: **stale probe** — no local platform is running on this host (`lsof -iTCP:8080` empty), so the auditor's probe must have hit an old process. My triage comment pointed this out and asked for live re-verification against a fresh `./platform/server` binary.

Next operator: if the CEO rebuilds + runs the local platform, re-probe:

```bash
curl -s -o /dev/null -w "%{http_code}" \
  http://localhost:8080/workspaces/ffffffff-ffff-ffff-ffff-ffffffffffff/secrets
```

Expected: **401** (because PR #357 removed the tokenless grace period). If 200, there's a real bug in the routing layer we haven't found.

### 3. Open design calls — CEO deciding

These are feature/plugin/research proposals. The next operator should NOT pick them up without explicit CEO instruction. They are listed here so the next operator can reference them quickly:

| Issue | Class | My recommendation |
|-------|-------|-------------------|
| #126 / #243 | Slack adapter for DevOps + Security Auditor | Build small (one webhook pattern, not full Slack app); confirm scope with CEO |
| #239 | Provisioner recovery for `failed` workspaces with missing config volume | Lean Option 1 (auto-reap + log) |
| #245 | Telegram channel for Security Auditor + DevOps | Already shipped via #246 |
| #258 | `molecule-sandbox` plugin (subprocess/docker/e2b) | Three separate plugins per CEO tick-032 direction |
| #274 | Witness/Deacon/Dogs three-tier health pattern | Layer 1 scaffolding only, ~6h |
| #286 | `investment-committee` template | Vertical pattern — valuable if there's a customer; skip otherwise |
| #294 | IATP signed delegation | Couple with #311 ADK spike |
| #298 | `molecule-plugin-github` | ~2h pickup, wraps github-mcp-server |
| #302 | Bloom behavioral eval hook | Skip, diminishing returns |
| #305 | Per-workspace token budget cap | Defer until billing model changes |
| #309 | `browser-use` plugin | Defer, overlaps with #281 |
| #311 | Google ADK A2A spike | Research spike, not code |
| #313 | Workspace-as-MCP-server | Phase-H design spike |
| #315 | HERMES_OVERLAYS two-layer provider | Research |
| #323 | `mcp-agent` plugin | Defer unless Research Lead bottleneck is real |
| #332 | `gemini-cli` runtime adapter | Defer until a user asks; ~4-6h |
| #333 | PM goal-decomposition skill | Minimal-scope, ~6h if picked up |
| #345 | `molecule-temporal` plugin | Defer — temporal_workflow.py already ships per-workspace |
| #347 | `molecule-governance` plugin | Pick up if MS AGT compliance matters to sales |
| #348 | Agent Protocol exposure spike | Research only |
| #349 | HITL structured feedback types | **Pickable** — concrete value, ~4h |
| #361 | Memory tiers (L0-L4) | **Pickable with 2 answers**: TEXT+CHECK vs enum, L0 enforced vs advisory |
| #362 | OpenSRE DevOps integrations | Research spike, need 3 target integrations from CEO |
| #364–368 | Recent plugin proposals (telemetry / trailofbits / awareness / budget / zombie / eco) | Mostly design calls; #368 budget enforcement is pickable |

### 4. Cron-learnings is the read-first file

`~/.claude/projects/-Users-hongming-Documents-GitHub-molecule-monorepo/memory/cron-learnings.jsonl` has ~52 ticks of operational history. The next operator reads the **last 20 lines** at the start of every tick (enforced by the SessionStart hook if installed, or by Step 0 of `playbook.md`).

Key cron-learnings conventions:
- `tick_id` format: `manual-NNN` for /triage runs, `overnight-NNN` for cron autonomous runs
- `category` is always `workflow` for now — reserved for future (`incident`, `config`, `research`)
- `next_action` must be CONCRETE and actionable by either the CEO or the next tick. Vague "continue monitoring" is a waste of disk.

### 5. Secrets status (for ops continuity)

| Secret | Where | Rotation |
|--------|-------|----------|
| `FLY_API_TOKEN` | GitHub Actions + `fly secrets` on `molecule-cp` | Both places, together |
| `SECRETS_ENCRYPTION_KEY` | molecule-cp | **Cannot rotate** until Phase H KMS envelope lands — see `docs/runbooks/saas-secrets.md` |
| `WORKOS_API_KEY` | molecule-cp | WorkOS dashboard only |
| `STRIPE_API_KEY` | molecule-cp | Currently TEST-MODE `sk_test_51TMJEV...`. Flip to live when CEO completes Canadian federal incorporation |
| `RESEND_API_KEY` | molecule-cp | Resend dashboard |
| `CP_ADMIN_USER_IDS` | molecule-cp | Comma-separated WorkOS user_ids — currently `user_01KPA3Z3810QEF3HCKRXP2EED9` |

### 6. Known unreliable signals

- **Mac mini self-hosted runner** has a history of 2+ hour queue latency. If CI pending > 30 min, prefer merging via local `go test -race ./...` + explicit CEO approval over waiting.
- **Security Auditor agent probes** sometimes run against stale platform binaries. Always confirm "which process / when" before treating a finding as current.
- **Eco-watch agent PRs** (e.g. #334, #350) are usually doc-only additions to `docs/ecosystem-watch.md`. Verified-merge is fine if the diff is pure docs.

---

## Open questions the next operator should NOT answer — escalate

- Stripe live-mode cutover timing
- App-UI subdomain layout (what goes at `app.moleculesai.app` once the CEO's other agent ships the landing page)
- Whether to add `schema_migrations` tracking table to the control plane migration runner
- Investment-committee template go/no-go (#286)

---

## Goodbye note

This was a ~100-tick session. I shipped 15 PRs across the two repos, caught two HIGH auth fail-opens the security auditor missed (#318 fake-UUID + #351 tokenless grace), two auth-blocker bugs in the control plane (wrong-cookie-contents + missing migration runner), and one directive-claim verification that held a PR for 10 minutes until the CEO confirmed (#370).

The philosophy that held up best across the whole session: **verify before claiming done.** Three different 401-loop bugs (#336, #351, WorkOS refresh-token) were all the same class — a claim of success that was technically true for the step the agent observed but false for the downstream step the agent didn't re-check. The operator who reads `playbook.md` Step 2 carefully will catch these before I did.

The philosophy that was hardest to hold: **don't pick up design calls.** The backlog looks like easy wins; each proposal says "small scope, clear fix." Most are 2-hour conversations with the CEO disguised as 2-hour engineering tickets. Reading the philosophy file's rule #7 (two-issue cap) + rule #9 (when you don't know, don't guess) is how you stay in-scope.

Good luck. Append your own goodbye note when you hand off.

— Claude Opus 4.6, 2026-04-16
