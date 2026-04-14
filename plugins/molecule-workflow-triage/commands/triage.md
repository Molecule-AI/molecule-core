---
name: triage
description: Run a full PR-triage cycle (gates 1-7 + code-review + merge if green). Equivalent to one cron tick, on demand.
---

# /triage

Manual invocation of the hourly PR-triage flow. Use when:
- You want to clear backlog faster than the hourly cadence
- You're testing a change to the triage prompt itself
- A scheduled cron has died and the queue is backing up

## Steps

### Step 0 — Activate guards + replay learnings
1. `Skill careful-mode` — load REFUSE/WARN/ALLOW lists.
2. Read last 20 lines of cron-learnings JSONL (workspace memory dir).

### Step 1 — List
```
gh pr list --state open --json number,title,author,isDraft,mergeable,statusCheckRollup
gh issue list --state open --json number,title,assignees,labels
```

### Step 2 — 7-gate verification per PR
- Gate 1 CI · Gate 2 build · Gate 3 tests · Gate 4 security · Gate 5 design · Gate 6 line review · Gate 7 Playwright if UI
- Supplement A: `Skill code-review`
- Supplement B: `Skill cross-vendor-review` on noteworthy PRs (auth/billing/data-deletion/migration/large-blast-radius)

### Step 2a — Mechanical fixes only
Fix on-branch + commit `fix(gate-N): ...` + push + poll CI. NEVER fix logic / design / auth issues.

### Step 2b — Merge
All gates pass + 0 🔴 from code-review + cross-vendor agreement → `gh pr merge N --merge --delete-branch`. Merge-commit only.

### Step 3 — Docs sync after any merge
`Skill update-docs` — measure test counts, don't guess.

### Step 4 — Issue pickup (cap 2)
For each candidate: gates I-1..I-6, self-assign, branch, implement, draft PR, run `Skill llm-judge` against issue body + PR diff. Mark ready only if score >= 4.

### Step 5 — Status report + cron-learnings
Report includes every subsection ("none" if empty). Then append 1-3 lines to cron-learnings JSONL.

## Standing rules (inviolable)
- Never push to main · Merge-commits only
- careful-mode REFUSE list ALWAYS blocks
- code-review 🔴 ALWAYS blocks merge
- cross-vendor disagreement on noteworthy PR escalates to user
- llm-judge ≤ 2 blocks marking a draft PR ready
