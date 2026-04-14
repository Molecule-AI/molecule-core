---
name: triage
description: Run the hourly PR-triage + issue-pickup + code-review + docs-sync loop. Equivalent to one tick of the c5074cd5 cron, on demand.
---

# /triage

Manual invocation of the same prompt the hourly cron runs at :17 past each hour. Use when:
- You want to clear backlog faster than the hourly cadence
- You're testing a change to the cron prompt itself
- The cron is session-only and the session has ended

## Steps

Run the full c5074cd5 cron flow:

### Step 0 — Activate guards + replay learnings
1. Invoke `Skill careful-mode` — load REFUSE/WARN/ALLOW lists.
2. Read last 20 lines of `~/.claude/projects/-Users-hongming-Documents-GitHub-molecule-monorepo/memory/cron-learnings.jsonl`.

### Step 1 — List
```
gh pr list --repo Molecule-AI/molecule-monorepo --state open --json number,title,author,isDraft,mergeable,statusCheckRollup,files
gh issue list --repo Molecule-AI/molecule-monorepo --state open --json number,title,assignees,labels,body
```

### Step 2 — 7-gate verification per PR
- Gate 1 CI · Gate 2 build · Gate 3 tests · Gate 4 security · Gate 5 design · Gate 6 line review · Gate 7 Playwright if canvas
- Supplement A: `Skill code-review` on every PR
- Supplement B: `Skill cross-vendor-review` on noteworthy PRs (auth/billing/data-deletion/migration/large-blast-radius)

### Step 2a — Mechanical fixes only
Fix on-branch + commit `fix(gate-N): ...` + push + poll CI. NEVER fix logic / design / auth issues.

### Step 2b — Merge
All gates pass + 0 🔴 from code-review + cross-vendor agreement → `gh pr merge N --merge --delete-branch`. Merge-commit only.

### Step 3 — Docs sync after any merge
`Skill update-docs` — measure test counts, don't guess. Open `docs/sync-YYYY-MM-DD-tick-N` PR, don't merge.

### Step 4 — Issue pickup (cap 2 per tick)
For each candidate issue: gates I-1..I-6, self-assign, branch, implement, draft PR, run `Skill llm-judge` against issue body + PR diff, mark ready only if score >= 4.

### Step 5 — Status report + cron-learnings
Report includes every subsection (use "none" if empty):
- Merged: #A, #B
- Fixed + merged: #C (gate-N fix)
- Fixed + awaiting CI: #D
- Skipped-design: #E (🔴 finding)
- Picked up issue #F → draft PR #G (llm-judge: N/5)
- Skipped issue #H (gate I-2)
- Code-review summary: total 🔴/🟡/🔵
- Cross-vendor pass/escalation
- Docs PR: #K
- Idle reason if nothing to do

THEN: append 1-3 lines to cron-learnings.jsonl. Terse. Concrete next_action only.

## Standing rules (inviolable)
- Never push to main · Merge-commits only · Dark theme only · No native browser dialogs · Delegate through PM · Only PM mounts the repo
- careful-mode REFUSE list ALWAYS blocks
- code-review 🔴 ALWAYS blocks merge
- cross-vendor disagreement on noteworthy PR escalates to CEO
- llm-judge ≤ 2 blocks marking a draft PR ready
