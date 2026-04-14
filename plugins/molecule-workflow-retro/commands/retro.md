---
name: retro
description: Generate a weekly retrospective digest — PRs merged, gate failures, code-review severity trend, time-to-merge, issues picked up. Posts as a GitHub issue.
---

# /retro

Weekly retrospective on cron + agent activity. Default cadence: Sundays
23:00 local. Manual invocation on demand.

## Steps

1. Compute over the prior 7 days:
   - Merged PR count (total + by category)
   - Issues closed (with PR-link for each)
   - Time-to-merge: median, p90, max — exclude docs PRs
   - Gate failure breakdown (which gates, how often)
   - Code-review findings: total 🔴/🟡/🔵, trend vs prior week
   - Mechanical fixes pushed (count)
   - Skips by reason: design-judgment / CI-down / scope-too-open / noteworthy-CEO-needed
   - Code volume: net LOC added/removed
   - Test count delta (Go + Python + Vitest + Jest)
   - New runtime / library / tool added or removed

2. Format per the `cron-retro` skill template.

3. Post as a new GitHub issue titled
   `Cron retro: <start> → <end> (week N)` with labels `meta`, `retro`.

4. If trends are bad (gate failure rate up, 🔴 findings appearing,
   time-to-merge >50% increase), flag prominently in the body and
   @-mention the workspace owner.

5. Skip new-issue creation if the prior 7 days had < 3 merged PRs;
   post a one-liner in the latest weekly retro issue's comments instead.

## Standing rules
- careful-mode applies — don't mass-close stale issues, don't delete
  prior retros
- The retro is observational, not actionable — propose 2-3 follow-ups
  for the user but never auto-create them
