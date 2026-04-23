# Coverage Floor

CI enforces three coverage gates on `workspace-server` (Go). All defined in
`.github/workflows/ci.yml` → `platform-build` job.

## Current floors (2026-04-23)

| Gate | Threshold | What fails |
|---|---|---|
| **Total floor** | `25%` | `go tool cover -func` reports total below floor |
| **Critical-path per-file floor** | `10%` | Any non-test source file in a security-critical path with coverage ≤10% |
| **Per-file report** | advisory | Printed in CI log, sorted worst-first, does not fail |

Total floor starts at 25% (unchanged from pre-#1823 to keep this PR strictly
additive). The new protection is the critical-path per-file floor, which
directly closes the gap that prompted the issue. Ratchet plan below begins
the month after to let the team first observe the gate in action.

## Security-critical paths (Gate 2)

Changes to these paths have historically introduced security issues (CWE-22,
CWE-78, KI-005, SSRF) or billing/auth risk. Coverage must not drop to zero.

- `internal/handlers/tokens*`
- `internal/handlers/workspace_provision*`
- `internal/handlers/a2a_proxy*`
- `internal/handlers/registry*`
- `internal/handlers/secrets*`
- `internal/middleware/wsauth*`
- `internal/crypto*`

## Ratchet plan

Floor ratchets upward on a fixed cadence. Any ratchet is a PR — reviewable,
reversible, and creates history. The table below is the intended schedule.

| Date | Total floor | Critical-path floor | Notes |
|---|---|---|---|
| 2026-04-23 | 25% | 10% | Initial gate (this file). |
| 2026-05-23 | 30% | 20% | First ratchet |
| 2026-06-23 | 40% | 30% | |
| 2026-07-23 | 50% | 40% | |
| 2026-08-23 | 55% | 50% | |
| 2026-09-23 | 60% | 60% | |
| 2026-10-23 | 70% | 70% | Target steady-state |

The target end-state matches the per-role QA prompts which specify
"coverage >80% on changed files". CI enforces the floor; reviewers still
enforce the per-PR bar.

## Exceptions

If a critical-path file genuinely cannot have coverage above the floor (e.g.
thin wrapper around a third-party SDK with no branches to test), add an entry
here with:

1. **File**: `internal/handlers/example.go`
2. **Reason**: Why coverage can't hit the floor
3. **Tracking issue**: GitHub issue for the real fix
4. **Expiry**: 14 days from entry date; after expiry either coverage is fixed
   or the issue is closed as "accepted technical debt"

### Active exceptions

*(none — add here if you need to land code that legitimately can't clear the floor)*

## Why this gate exists

Issue #1823: an external audit found critical files at 0% coverage despite
test files existing with hundreds of lines. The existing CI step measured
coverage but didn't enforce a meaningful threshold. Any file could go from
80% → 0% and CI stayed green, because the single gate (total ≥25%) ignored
per-file distribution.

This gate makes "no untested critical paths merged" a mechanical property of
the CI, not a behavioural property of QA agents or individual reviewers —
which is the only way to make it survive fleet outages, agent rotations, or
QA process changes.
