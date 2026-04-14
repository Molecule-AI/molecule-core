---
name: cron-retro
description: Weekly retrospective digest of cron activity — PRs merged, gates failed, issues picked, code-review findings by severity, time-to-merge, regression trend. Posts to a dedicated GitHub issue. Inspired by gstack's /retro.
---

# cron-retro

The cron runs hourly and ships a lot. Without a periodic summary, drift happens silently — Gate 4 starts failing more often, code-review noise climbs, time-to-merge balloons, and nobody notices for weeks.

## When to run

- Every Sunday at 23:00 local (`0 23 * * 0` cron expression)
- On-demand by the CEO

## What to compute (over the prior 7 days)

From `gh pr list --state merged --search "merged:>=YYYY-MM-DD"` and our local `cron-learnings.jsonl`:

1. **Merged PR count** — total + by category (auth/security, refactor, feat, fix, docs, infra)
2. **Issues closed** — count, with PR-link for each
3. **Time-to-merge distribution** — median, p90, max. Excluding docs PRs (they merge instantly).
4. **Gate failure breakdown** — which gates failed how often. Patterns?
5. **Code-review findings** — total 🔴 / 🟡 / 🔵 across all PRs. Trend vs prior week.
6. **Mechanical fixes pushed** — how often did the cron fix a gate failure on-branch?
7. **Skips by reason** — categorize: design-judgment, CI-down, scope-too-open, noteworthy-CEO-needed
8. **Code volume** — net LOC added/removed (Garry Tan publishes these in his retros — keep us honest)
9. **Test count delta** — Go + Python + Vitest + Jest from start to end of week
10. **New runtime / library / tool added or removed** — anything strategic

## Format

Post a new GitHub issue titled `Cron retro: 2026-04-14 → 2026-04-21 (week N)` with body:

```markdown
# Week summary
- Merged: X PRs (Y closed issues)
- Median TTM: 3h12m (excluding docs)
- Code-review findings: 0 🔴 / 4 🟡 / 18 🔵 (vs last week: 0 / 6 / 24)
- Mechanical fixes pushed: 5
- Skips: 2 design-judgment, 1 CI-down

# Trend signals
- ↑ Frontend test coverage (+12 vitest, +1 file)
- ↓ Time-to-merge for auth PRs (down from 8h median to 3h — likely
   because Gate-4 doc-sync subagent now catches missing .env entries)
- ⚠ Gate 7 (Playwright) failed 3 times this week vs 0 last week —
   probably the canvas dev-server stale-chunk issue. Action item.

# Code volume
- 12,847 lines added, 8,213 removed across 23 commits

# Notes
- Closed #6, #13, #17, #23 — 4 issues from the launch backlog
- 2 issues remain in the SaaS-launch Tier 1 list (multi-tenancy, Fly Machines)
- New skills added this week: cross-vendor-review, careful-mode, cron-learnings, cron-retro

# Action items for next week
- [ ] Investigate Gate 7 flakes (likely fix: persistent canvas dev daemon)
- [ ] Pick up issue #19 (workspace restart context)
- [ ] PR #58 needs CEO review (configurable tier limits — behavior change)
```

## Why this exists

What gets measured improves. gstack publishes weekly retros and credits them with knowing where to invest. We have no analog. This is the smallest viable analog: one issue per week, generated automatically, costs nothing to ignore, valuable when the metrics start drifting.

## Implementation note

This skill should be invoked from a separate cron job (not the hourly triage cron). Suggested cron expression: `7 23 * * 0` — Sunday 23:07 local.
