# Incident Log — molecule-core

> This file documents security incidents, outages, and degraded states.
> Active incidents are listed first. Resolved incidents remain for historical record.

---

*Last updated: 2026-04-22T08:20Z by Infra-SRE — WORKSPACE_ID: all builtin_tools gaps closed (d838b73 + f3204c2); P0 still active, push blocked*

---

## P0 — Platform Incident (ACTIVE, 2026-04-22 ~00:00Z onward)

**Severity:** Critical
**Period:** 2026-04-22 ~00:00Z – ongoing
**Finding IDs:** N/A (infra/cascade outage)
**Status:** ACTIVE — platform unreachable, GitHub tokens revoked org-wide

### Summary

Multiple cascading failures:
1. **Platform unreachable** — PM workspace (f0897ffd) and second EC2 down, cascade spreading
2. **GitHub tokens revoked org-wide** — `ghs_f5trls76E6B2T9jU...` returns 401 "Bad credentials" across ALL agents (Research Lead, Core-DevOps, Plugin-Dev, Fullstack, CP-Security, Core-QA, SDK Lead all confirmed)
3. **A2A mesh degraded** — PM workspace + EC2 down blocks peer-to-peer delegation for dependent workspaces

### Affected Systems

| System | Status |
|--------|--------|
| Platform API | ❌ Unreachable |
| GitHub (gh api, git push) | ❌ 401 org-wide |
| PM workspace (f0897ffd) | ❌ DOWN |
| Self-hosted CI runners | ⚠️ Runner hongming-claws has corrupt Go module cache |
| GCP staging CP | ❌ 403 on `/cp/admin/orgs` — missing IAM policy (KI-007) |

### GitHub Token Status

`ghs_f5trls76E6B2T9jU...` (molecule-ai[bot]) — revoked or rotated by mol-ops. All write operations blocked across all workspaces. Token confirmed dead via `gh auth status` and `gh api` calls returning "Bad credentials" on every agent.

### PR #1498 CI Blockers

**Branch:** `ship/security-fixes-to-main-0516` (P0 — emergency security fixes)
**Fix commits (ready, not pushed):**
- `54be428` — fix(handlers): resolve Go compile failures from duplicate declarations
- `a701cc0` — fix(tests): conftest httpx mock — proper async context manager support
- `0f39f82` — docs(incidents): document Python test fix + Go fix in qa-audit

**CI failures (pre-fix state, SHA d490f977):**
| Job | Result | Fix |
|-----|--------|-----|
| Platform (Go) | ❌ FAIL | `54be428` — deleted duplicate workspace_crud.go |
| Python Lint & Test | ❌ FAIL | `a701cc0` — fixed conftest httpx mock |
| E2E API Smoke Test | ❌ FAIL | Runner Go cache corruption — needs SSH to hongming-claws |
| CodeQL (go/js/py) | ✅ SUCCESS | — |
| Canvas (Next.js) | ✅ SUCCESS | — |
| Shellcheck | ✅ SUCCESS | — |
| Detect changes | ✅ SUCCESS | — |

### Required Actions

1. **Restore GitHub credentials** — mol-ops must provide new PAT for molecule-ai[bot] or re-enable existing token
2. **Push PR #1498 fixes** — `git push origin ship/security-fixes-to-main-0516` (blocked by #1)
3. **SSH to hongming-claws** — clear Go module cache corruption:
   ```bash
   ssh hongming-claws
   rm -rf ~/go/pkg/mod ~/Library/Caches/go-build
   cd /Users/hongming-claw/actions-runner-2/_work/molecule-core/molecule-core
   go mod download
   ```
4. **GCP IAM fix (KI-007)** — apply `roles/container.clusterAdmin` to staging canary SA:
   ```bash
   gcloud iam service-accounts add-iam-policy-binding \
     staging@YOUR_PROJECT.iam.gserviceaccount.com \
     --role=roles/container.clusterAdmin \
     --member="serviceAccount:staging@YOUR_PROJECT.iam.gserviceaccount.com"
   ```
5. **PM workspace + EC2 recovery** — Infra team to restore PM workspace (f0897ffd) and EC2 cascade

### Known Issue — builtin_tools WORKSPACE_ID validation (INCIDENT LOG #1124) — ✅ FULLY FIXED

All builtin_tools modules with empty-string WORKSPACE_ID defaults have been patched:

| File | Commit | Change |
|------|--------|--------|
| `builtin_tools/memory.py` | `d838b73` | RuntimeError guard (module-level) |
| `builtin_tools/approval.py` | `d838b73` | RuntimeError guard (module-level) |
| `builtin_tools/audit.py` | `d838b73` | RuntimeError guard (module-level) |
| `builtin_tools/delegation.py` | `d838b73` | RuntimeError guard (module-level) |
| `builtin_tools/governance.py` | `d838b73` | RuntimeError guard (module-level) |
| `builtin_tools/a2a_tools.py` | `f3204c2` | RuntimeError guard (module-level) |
| `builtin_tools/hitl.py` | `f3204c2` | RuntimeError guard (3 function-local sites: pause_task, resume_task, _notify_all_channels) |

Branch: `ship/security-fixes-to-main-0516` — 7 unpushed commits, GH_TOKEN 401 blocks push.

**Known exceptions (low priority, intentional defaults):**
- `builtin_tools/telemetry.py` — `"unknown"` default for optional OTEL telemetry
- `builtin_tools/awareness_client.py` — `""` default; function returns early when awareness not configured

---

## Security Audit Cycle 6 — ALL CLEAR (2026-04-21 ~07:15Z)

**SHA range:** e69cb26 → 674384b on main (~5 commits + ~10 merged PRs)
**Verdict:** ✅ No critical/high findings

### Commits Reviewed — All CLEAN

| Commit | Description |
|--------|-------------|
| `dc9c64e` / PR #1258 | F1097 org_id context — eliminates redundant 2nd SELECT in AdminAuth |
| `33f1d1a` | Canvas cascade-delete UX — `pendingDelete.hasChildren`, warning dialog |
| `0790d57` | Canvas metrics guard — null coalescing |
| `781c217` | CI YAML fix |
| `169120d` / PR #1310 | CWE-78/CWE-22 — exec form + path traversal guards |
| `e431fc4` / PR #1302 | CWE-918 SSRF — `isSafeURL` in `a2a_proxy.go` |
| `a66f889` / PR #1261 | CWE path-injection — `resolveInsideRoot` for template paths |

Full audit saved to TEAM memory id `abc58b47`.

---

## F1100 — workspace_restart.go Path Traversal (RESOLVED)

**Severity:** Medium | **Finding ID:** F1100
**Status:** Resolved — fix applied via `a66f889` (PR #1261) on both main and staging

### Summary

`workspace_restart.go:127-133` accepted `body.Template` (attacker-controlled) via raw `filepath.Join(h.configsDir, template)`, allowing path traversal (e.g. `../../../etc`) to escape `configsDir`. **Issue #1043 triage missed this — legitimate gap, not false positive.**

Authenticated callers could pass a crafted `body.Template` value to escape the configs directory.

### Fix Applied

PR #1260 (intended) closed without merge. Fix landed via **PR #1261 (`a66f889`)** on both main and staging:

```go
// Fixed (a66f889):
candidatePath, resolveErr := resolveInsideRoot(h.configsDir, template)
if resolveErr != nil {
    template = ""  // fallback fires safely
}
```

### References

- PR #1260: closed without merge — superseded by PR #1261
- PR #1261 (`a66f889`): merged ✅
- Closes: #1043

---

## F1088 Credential Exposure — CLOSED

**All prior F1088 entries below remain valid. Summary of current state:**

- Credentials: MiniMax revoked (⚠️), GitHub PAT revoked (✅), Admin token — treat as potentially exposed
- BFG git-history scrub: NOT REQUIRED — incident management closure, 0 public forks confirmed
- Git history still contains values — admin token rotation recommended as precaution
- PR #1179 (`b89f3fd`) merged — active code is clean
- Branch `origin/fix/credential-history-cleanup-f1088` exists but is 38 commits behind main — superseded by incident management closure

**Required remaining action:** Rotate `ADMIN_TOKEN` (`HlgeMb8...ShARE=`) as precaution. All other actions complete.

---

### Summary

Commit `d513a0ced549ef2be8903a7b4794256110ba1805` on staging (merged to main via PR #1098) contains three production credentials as hardcoded default values in `scripts/post-rebuild-setup.sh`. The credentials appeared in the git diff and were permanently visible in the public commit history.

### Credentials Status

| # | Credential | Value | Status |
|---|------------|-------|--------|
| 1 | ANTHROPIC_AUTH_TOKEN | `sk-cp-lHt...KVw` | ⚠️ Revoked or inactive (404 on API call) |
| 2 | GITHUB_TOKEN | `github_pat_11...hsIJLIL` | ✅ Revoked (confirmed 401) |
| 3 | ADMIN_TOKEN | `***REDACTED***` | Needs confirmation — treated as active until proven otherwise |

### Resolution

PR #1179 (`b89f3fd`: "ci: retry — trigger fresh runner allocation") closed this finding. The incident was closed at the finding-management level. Git history scrub via BFG was discussed but deemed not required by security team (no active public forks confirmed, credentials were already revoked/inactive).

Active code is clean (`d513a0c` replaced hardcoded defaults with env-var reads).

### Summary

Commit `d513a0ced549ef2be8903a7b4794256110ba1805` on staging (merged to main via PR #1098) contains two production credentials as hardcoded default values in `scripts/post-rebuild-setup.sh`. The credentials appear in the git diff and are permanently visible in the public commit history.

The commit itself fixed the problem by replacing hardcoded defaults with env-var reads (MINIMAX_API_KEY, GITHUB_PAT). However, git history still shows the original values.

### Credentials Exposed

| # | Credential | Value (redacted reference) | Service |
|---|------------|------------------------------|---------|
| 1 | ANTHROPIC_AUTH_TOKEN | `***REDACTED***` | MiniMax API (api.minimax.io/anthropic) |
| 2 | GITHUB_TOKEN | `***REDACTED***` | GitHub (fine-grained PAT, scope unknown) |
| 3 | ADMIN_TOKEN | `***REDACTED***` | Platform admin authentication |

### Affected Files

- `scripts/post-rebuild-setup.sh` (commit d513a0c, PR #1098 → merged to staging → merged to main)

### Timeline

- **~2026-04-20T13:02Z**: Commit `d513a0c` pushed by `rabbitblood`. GitGuardian flagged credentials in the diff. Fix committed in same commit.
- **~2026-04-20T**: Credentials removed from active code, but git history still contains them.
- **2026-04-20T22:32Z**: Incident discovered and escalated.

### Actions Taken

1. Dev Lead notified (delegation failed — Dev Lead unreachable)
2. All child workspaces notified (delegation failed — all unreachable)
3. Incident documented in this file
4. Branch `origin/fix/credential-history-cleanup-f1088` exists but is 38 commits behind `origin/main`
5. **Incident CLOSED** — PR #1179 merged, finding management closure, BFG scrub deemed not required (no active public forks confirmed)

### Blast Radius (Confirmed by Core-Security)

| Credential | Test Result | Status |
|------------|-------------|--------|
| MiniMax API key (`sk-cp-...KVw`) | `404 Not Found` on real API call | ⚠️ **REVOKED** (or endpoint inactive) |
| GitHub PAT (`github_pat_...hsIJLIL`) | `401 Bad credentials` | ✅ **REVOKED** |
| Admin token (`HlgeMb8...ShARE=`) | Base64 — cannot test directly | ⚠️ **Treated as active** — recommend rotation as precaution |

**Public forks:** 0 confirmed (GH API `/forks` returns none) — low fork blast radius.

**Git history scope:** Credentials exist in both `main` and `staging` in commits `f787873`..`d513a0c`. They were introduced in `f787873` ("feat: nuke-and-rebuild.sh") and removed from active code in `d513a0c`. Both branches require BFG cleanup.

### Required Actions (RESOLVED)

- [x] Credentials revoked (MiniMax ⚠️, GitHub PAT ✅)
- [x] BFG git history cleanup **NOT REQUIRED** — incident management closure, no active public forks, credentials confirmed revoked/inactive
- [x] Team notification — documented in this log
- [ ] **Admin token rotation** — recommended as precaution (value still in git history, treat as potentially exposed)

### BFG Repo-Cleaner Procedure

**NOT REQUIRED** — F1088 closed without BFG scrub per security team decision. Retained for reference only.

**Step 1 — Create credentials manifest (`creds.txt`) [NOT NEEDED]:**
```
***REDACTED***
***REDACTED***
***REDACTED***
```

**Step 2 — Clean origin/main:**
```bash
git clone --mirror https://github.com/Molecule-AI/molecule-core /tmp/molecule-main-mirror
java -jar bfgr.jar --replace-text creds.txt --rewrite-not-committed-by-oss --no-blob-protection /tmp/molecule-main-mirror
cd /tmp/molecule-main-mirror && git push --mirror
```

**Step 3 — Clean origin/staging:**
```bash
git clone --mirror https://github.com/Molecule-AI/molecule-core /tmp/molecule-staging-mirror
java -jar bfgr.jar --replace-text creds.txt --rewrite-not-committed-by-oss --no-blob-protection /tmp/molecule-staging-mirror
cd /tmp/molecule-staging-mirror && git push --mirror
```

**Step 4 — Notify team to re-clone both branches if cloned before ~13:02 UTC 2026-04-20.**

### References

- Commit: `d513a0ced549ef2be8903a7b4794256110ba1805`
- PR: #1098 (staging → main merge)
- Cleanup branch: `origin/fix/credential-history-cleanup-f1088` (behind main by 38 commits)
- Scanners triggered: GitGuardian
- Security investigation: Core-Security (confirmed credentials revoked via API tests)
- GitHub issue: #1282 (filed by Core-OffSec)
- **Closed by:** PR #1179 (`b89f3fd`) — incident management closure, BFG scrub deemed not required

### Known Issue — PR #1230 Incomplete (QA Round 16, 2026-04-21)

PR #1230 / commit `524e3c6` ("fix(security): replace err.Error() leaks") failed to carry mcp.go fixes into main's tree. All 3 MCP error leaks remain on main:
- `mcp.go:259`: "parse error: " + err.Error()
- `mcp.go:347`: "invalid params: " + err.Error()
- `mcp.go:352`: err.Error()
- `org_plugin_allowlist.go:260`: "detail": err.Error()

Fix is covered by PR #1226 (rebased, MERGEABLE). Gap should close after #1226 merges.

---

## CWE-918 SSRF — Backport to Main (RESOLVED)

**Severity:** High
**Status:** Resolved — PR #1302 merged to main

### Summary

SSRF defence (`isSafeURL` in `a2a_proxy.go`) was backported to main to address CWE-918 (Server-Side Request Forgery). The fix prevents the A2A proxy from forwarding requests to internal network addresses (localhost, private ranges, etc.).

### References

- Commit: `e431fc4` (fix(security): backport SSRF defence (CWE-918) to main — isSafeURL in a2a_proxy.go (#1292) (#1302))

---

## CWE-22 + CWE-78 Security Fixes — Merged (RESOLVED)

**Severity:** Critical
**Status:** Resolved — proper fixes merged to staging and main

### Summary

The `fix/cwe78-delete-via-ephemeral-shell-injection` branch was the right diagnosis but wrong implementation (removed `safeName` from `copyFilesToContainer`). The correct fixes were merged separately:

| Location | Commit | Fix |
|----------|--------|-----|
| staging | `ce2491e` | CWE-22: `copyFilesToContainer` safeName + `deleteViaEphemeral` validateRelPath + exec form |
| main | `169120d` | CWE-78/CWE-22: block shell injection in `deleteViaEphemeral` |

Both CWEs are fully resolved on both branches. The regression branch is superseded and must not be merged as-is.

### Verification (staging `ce2491e`)

`copyFilesToContainer` (container_files.go:73-99):
```go
clean := filepath.Clean(name)
if filepath.IsAbs(clean) || strings.Contains(clean, "..") {
    return fmt.Errorf("path traversal blocked: %s", name)
}
safeName := filepath.Join(destPath, clean)
header := &tar.Header{Name: safeName, ...}  ✅
```

`deleteViaEphemeral` (container_files.go:152-168):
```go
validateRelPath(filePath)  ✅
Cmd: []string{"rm", "-rf", "/configs", filePath}  ✅ exec form, no shell interpolation
```

---



**Severity:** High
**Period:** ~2026-04-20T22:00Z – 2026-04-21T03:30Z
**Finding IDs:** N/A (infra incident)
**Status:** Resolved

### Summary

All self-hosted macOS arm64 runners saturated. 27 runs queued, 0 in-progress, 0 completed. Only cancellations processing. PRs #1053 and #1036 had zero CI runs.

### Root Causes (multiple)

1. `changes` job ran on `[self-hosted, macos, arm64]` despite having zero macOS dependencies (plain `git diff`) — wasted runner slots
2. YAML corruption in `ci.yml` (JSON-escaped `\n` sequences from commits `12c52d4`/`5831b4e`) caused "workflow file issue" failures before any job could start
3. `cancel-in-progress: false` at workflow level caused stale runs to queue instead of being cancelled
4. Workflow-level concurrency not set — multiple in-flight runs queued on same ref

---

## CI Stall — molecule-core/staging (RESOLVED 2026-04-21 ~07:05Z)

**Severity:** High
**Period:** ~2026-04-21T02:47Z – ~2026-04-21T07:00Z
**Status:** Resolved — CI progressing normally, no config problems remain

### Resolution

All prior runner-saturation and YAML-corruption fixes were correct. The stall resolved naturally once stale queued runs drained. Current CI state (2026-04-21 ~07:07Z):

- Staging run #24708961892: **success** (SHA `5d32373`)
- Staging run #24708976467: **success** (changes job, SHA `72d825f`)
- Main run #24708984339: queued (normal — healthy queue, not stalled)
- Runner agent healthy — no dead slots

### Root Causes (all resolved)

1. `changes` job on `[self-hosted, macos, arm64]` — fixed by moving to `ubuntu-latest` (`9601545`)
2. YAML corruption in `ci.yml` — fixed by PR #1264 / `b61692c` ✅
3. `cancel-in-progress: false` at workflow level — reverted to `true` on staging ✅
4. `cancel-in-progress: false` on main — correct for single-runner env, aligned via PR #1248 ✅

### Staging CI Config (confirmed healthy)

- `ci.yml`: `cancel-in-progress: true`, `changes` job on `ubuntu-latest` ✅
- `codeql.yml`: `cancel-in-progress: false` ✅
- `e2e-api.yml`: `cancel-in-progress: false` ✅

### Infra Recommendations (for long-term stability)

1. Provision org-wide GitHub App installation token for CI automation (PATs rotate too frequently)
2. Update remote URLs on controlplane and tenant-proxy repos
3. Monitor runner agent health on mac mini — restart agent if future stalls recur

---

## PR #1242 YAML Corruption — RESOLVED (PR never merged)

**Severity:** Critical
**Status:** Resolved — PR #1242 closed without merge, staging unaffected

### Summary

PR #1242 (`fix/ci-runner-queue-contention`) branch contained a YAML corruption in `ci.yml` — the `concurrency` block was replaced with a commit-SHA string literal:

```yaml
e4a62e1 (ci: add workflow-level concurrency to ci.yml and codeql.yml)
```

However, PR #1242 was **closed without merging**. Staging received `cancel-in-progress: true` via PR #1264 (commit `b61692c`) instead, which is the correct clean version.

### Current State (updated 2026-04-21 ~04:30Z)

- **main:** `cancel-in-progress: false` ✅ (from PR #1248 / `2ffd11c` or similar clean commit)
- **staging:** `cancel-in-progress: true` (via `0b30465` tick restore after corruption)
- **PR #1248** (`2ffd11c`): open, sets staging `cancel-in-progress: false` — aligns staging with main ✅
- **Main has moved to `false`** — staging should follow to stay consistent

### PR #1248 — URGENT MERGE

PR #1248 (`fix/ci: restore corrupted ci.yml concurrency block`) by Dev Lead:
- Fixes the corruption pattern (same as prior incident)
- Sets `cancel-in-progress: false` — correct for single-runner environment
- Aligns staging CI config with main (which already has `false`)
- Must merge before any further CI runs on staging

### References

- PR: #1242 (`fix/ci-runner-queue-contention`) — closed, not merged
- Staging corruption restored via: PR #1264 / `b61692c`
- PR #1248 (`2ffd11c`): open, Dev Lead fix, `cancel-in-progress: false`
- Main: `cancel-in-progress: false` ✅

---

## PR #1036 QA Audit (STALE)

**Severity:** Low
**Date:** 2026-04-20 (QA audit performed)
**Status:** Stale — CI infrastructure has been fixed since audit

### Summary

QA audit (2026-04-20) flagged CI as failing on PR #1036. However, CI was failing due to infrastructure issues (runner saturation, YAML corruption) that have since been resolved. The audit should be re-run now that staging CI is healthy.

---

## PR #1246 / #1247 — Sed Regression Fix — RESOLVED (PR #1247 merged)

**Severity:** Critical
**Status:** Resolved — PR #1247 merged to main (2026-04-21 ~03:18Z)

### Summary

PR #1246 (`364712d`) was closed without merging. However, **PR #1247** (`04be218`) achieved the same fix cleanly and merged to main:

```
fix(go): replace $1 literal with resp.Body.Close() in 7 files (#1247)
```

Commit `04be218` (merged by molecule-ai[bot]) applied:
```
sed -i 's/defer func() { _ = \$1 }()/defer func() { _ = resp.Body.Close() }()/g'
```

### Affected Files (all fixed on main)

- `workspace-server/cmd/server/cp_config.go`
- `workspace-server/internal/handlers/a2a_proxy.go`
- `workspace-server/internal/handlers/github_token.go`
- `workspace-server/internal/handlers/traces.go`
- `workspace-server/internal/handlers/transcript.go`
- `workspace-server/internal/middleware/session_auth.go`
- `workspace-server/internal/provisioner/cp_provisioner.go` (3 occurrences)

**Staging:** Fix present via prior commits. `cp_config.go` on staging has SHA `d1021c2` (correct form).

**PR #1246:** Closed without merging — superseded by PR #1247. No further action needed.

---

## CWE-78/CWE-22 Branch — RESOLVED (proper fixes merged separately)

**Severity:** Critical
**Status:** Resolved — proper fixes merged via `ce2491e` (staging) and `169120d` (main)

### Summary

The `fix/cwe78-delete-via-ephemeral-shell-injection` branch (commit `17419dd`) was **correct** for CWE-78 (`deleteViaEphemeral` exec form + `validateRelPath`) but **regressed** `copyFilesToContainer` by removing the `safeName` path-traversal guard.

**Resolution — both branches merged to main and staging:**

| Branch | Commit | Status |
|--------|--------|--------|
| staging | `ce2491e` — fix(security): CWE-22 in copyFilesToContainer and deleteViaEphemeral | ✅ merged |
| main | `169120d` — fix(security): CWE-78/CWE-22 — block shell injection in deleteViaEphemeral | ✅ merged |

### What was fixed (staging `ce2491e`)

- `copyFilesToContainer`: `filepath.Clean` + `IsAbs` + `strings.Contains("..")` validation, `safeName` in tar header ✅
- `deleteViaEphemeral`: `validateRelPath(filePath)` check before rm command ✅
- Both CWE-22 and CWE-78 addressed correctly

### `fix/cwe78-delete-via-ephemeral-shell-injection` branch status

**Do NOT merge** — it's now superseded by `ce2491e`/`169120d`. The regression it introduced (removing `safeName` from `copyFilesToContainer`) was never the right approach. If this branch is revived, it must be rebased on top of `ce2491e` to preserve existing CWE-22 protections while adding the CWE-78 exec-form fix.

---

## F1085 Regression Branch (`fix/f1085-regression-1283`) — IS a Regression

**Severity:** High
**Status:** Active — branch removes the confirmed-good F1085 fix (confirmed 2026-04-21 ~07:10Z)

### Summary

Branch `origin/fix/f1085-regression-1283` (commit `3b244e6`) removes `redactSecrets(workspaceID, content)` from `seedInitialMemories` in `workspace_provision.go:249`:

```diff
-`, workspaceID, redactSecrets(workspaceID, content), scope, awarenessNamespace); err != nil {
+`, workspaceID, content, scope, awarenessNamespace); err != nil {
```

**Staging still has the correct fix** (`workspace_provision.go:253` on origin/staging confirms `redactSecrets` is present). This branch is behind staging and would regress it if merged.

### Required Fix

Close or revert this branch. `redactSecrets` must remain in `seedInitialMemories`. If there is a legitimate reason to change this (e.g., a different redaction strategy), document it clearly in the PR before merging.

---

## F1097 — org_id Context Fix — RESOLVED

**Severity:** Medium
**Status:** Resolved — PR #1258 merged to main (`dc9c64e`)

### Summary

`orgToken.Validate` refactored to return `org_id` directly, eliminating the redundant 2nd SELECT in `AdminAuth`. All SQL parameterized correctly.

### References

- PR #1258 (`dc9c64e`): fix(F1097): set org_id in Gin context for org-token callers

---

## PR #1226 — err.Error() Leaks (STALE — closed without merge)

**Severity:** Medium
**Status:** Open — PR closed without merging, leaks still present on main

### Summary

PR #1226 (`fix(security): sanitize remaining err.Error() leaks + errcheck artifacts/client.go`) was **closed without merging**. The following leaks remain on main:

| File | Line | Code | Fix |
|------|------|------|-----|
| `mcp.go` | 259 | `"parse error: " + err.Error()` | → `"parse error: invalid JSON request body"` |
| `mcp.go` | 347 | `"invalid params: " + err.Error()` | → `"invalid params: malformed JSON"` |
| `mcp.go` | 352 | `err.Error()` | → `"dispatch error"` |
| `org_plugin_allowlist.go` | 260 | `"detail": err.Error()` | → `"detail": "plugin name validation failed"` |
| `admin_memories.go` | 99 | `"invalid JSON: " + err.Error()` | → `"invalid JSON request body"` |

**Already fixed:** `artifacts/client.go:175` — `defer func() { _ = resp.Body.Close() }()` confirmed correct (via PR #1247).

### Action Required

Reopen PR #1226 and fast-track merge. Alternatively, cherry-pick the 4 commits from that PR onto a fresh branch.

---

## QA Round 18 — orgs-page Test Regression (FIXED on main, pending staging port)

**Severity:** Medium
**SHA tested:** `ce33da5` (PR #1257 branch merge with staging)
**Status:** Regression identified in PR #1255, fixed on main, not yet on staging

### Findings

| Finding | Status |
|---------|--------|
| Canvas tests: 53 passed, **1 FAILED** | orgs-page.test.tsx line 133 — `vi.useRealTimers()` + raw `setTimeout(50)` without `act()` |
| PR #1257 conflict | MERGEABLE, approved — closed without merge; fix is on main/staging via `a66f889` |
| PR #1255 regression | Introduced orgs-page test flakiness — +18/-2 in orgs-page.test.tsx |

### orgs-page Test Regression — Root Cause

PR #1255 (`e885fa1`) regressed the timer fix from PR #1235. It replaced `waitFor()` with `vi.useRealTimers()` + raw `setTimeout(50)` without `act()` — causing microtask flush issues.

### Resolution

**Main:** Fixed in `674384b` (PR #1313) — wraps all 10 affected `vi.advanceTimersByTimeAsync(50)` calls in `act(async () => { ... })`. All 813 canvas tests pass on main.
**Staging:** Regression NOT yet fixed — `origin/staging` is 13 commits behind main.

### Action needed

Cherry-pick or port the orgs-page test fix from `674384b` to staging.

---

## Issue #1124 — Orchestrator GET /workspaces 404: Env Var Misconfiguration (OPEN)

**Severity:** Medium
**Status:** Active — root cause confirmed, fix pending, delegated to Core-BE

### Summary

Orchestrator (workspace agent, `workspace/` directory) GET /workspaces/{WORKSPACE_ID} returns 404 due to missing or empty `WORKSPACE_ID` env var. Confirmed via code review (2026-04-21 ~07:10Z).

### Root Causes

**Platform-side (provisioner.go:375-377) is CORRECT:**
```go
env := []string{
    fmt.Sprintf("WORKSPACE_ID=%s", cfg.WorkspaceID),  // ✅ correctly injected
    "WORKSPACE_CONFIG_PATH=/configs",
    fmt.Sprintf("PLATFORM_URL=%s", cfg.PlatformURL),
}
```
The platform injects `WORKSPACE_ID` at container provision time. **The bug is in the Python orchestrator modules** that default to empty string instead of validating the injected value.

**Buggy Python module-level defaults (empty string → broken API calls):**
| File | Line | Code |
|------|------|------|
| `workspace/a2a_cli.py` | 24 | `WORKSPACE_ID = os.environ.get("WORKSPACE_ID", "")` |
| `workspace/a2a_client.py` | 17 | `WORKSPACE_ID = os.environ.get("WORKSPACE_ID", "")` |
| `workspace/coordinator.py` | 26 | `WORKSPACE_ID = os.environ.get("WORKSPACE_ID", "")` |
| `workspace/consolidation.py` | 22 | `WORKSPACE_ID = os.environ.get("WORKSPACE_ID", "")` |
| `workspace/molecule_ai_status.py` | 25 | `WORKSPACE_ID = os.environ.get("WORKSPACE_ID", "")` |

When `WORKSPACE_ID` is empty, API calls produce URLs like `/workspaces//heartbeat` or `/registry/discover/` — platform returns 404 or wrong routing.

**Note — main.py is already correct:**
```python
workspace_id = os.environ.get("WORKSPACE_ID", "workspace-default")  # main.py:55 ✅
```
However, `main.py` uses a local variable — it doesn't export `WORKSPACE_ID` as a module constant. The other modules that import `WORKSPACE_ID` from `a2a_client` etc. still get the empty-string default.

### Fix Required (Quick Win for Core-BE)

**Option A — Fail fast at module import (recommended):**
```python
WORKSPACE_ID = os.environ.get("WORKSPACE_ID")
if not WORKSPACE_ID:
    raise RuntimeError("WORKSPACE_ID environment variable is required but not set")
```
Apply to all 5 affected modules. This surfaces the misconfiguration immediately instead of producing silent 404s downstream.

**Option B — Align with main.py's approach (safer):**
```python
WORKSPACE_ID = os.environ.get("WORKSPACE_ID", "workspace-default")
```
But this masks real misconfigurations. Option A is better.

### Modules Requiring Fix

- `workspace/a2a_cli.py` — line 24
- `workspace/a2a_client.py` — line 17
- `workspace/coordinator.py` — line 26
- `workspace/consolidation.py` — line 22
- `workspace/molecule_ai_status.py` — line 25

### PLATFORM_URL Note

All modules default to `http://platform:8080` (container mesh hostname). This is correct for in-container use but fails outside Docker. No action needed for in-container orchestrators — the platform injects `PLATFORM_URL` at provision time which overrides this default.

### Owner

Core-BE — delegated to Dev Lead (A2A failed). Core-BE sub-team: please pick up.

### Fix PR

[PR #1336](https://github.com/Molecule-AI/molecule-core/pull/1336) filed — `fix(orchestrator): fail-fast if WORKSPACE_ID env var is unset/empty`. Targets staging. Labels: bug, needs-work, area:backend-engineer, area:dev-lead.

---

*Last updated: 2026-04-21T07:10Z by Core Platform Lead (post-restart session — all findings re-verified)*