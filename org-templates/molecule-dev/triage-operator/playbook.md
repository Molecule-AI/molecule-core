# Triage Operator — Playbook

The step-by-step flow for a single triage tick. Cron fires, you wake, you run this exact sequence.

Expected wall-clock: **5–15 minutes** per tick when the backlog is small; up to 30 minutes when clearing a large stack. If you're going past 30 minutes, you're doing engineer work — stop, leave a triage comment, escalate.

---

## Step 0 — Guard activation + learnings replay

1. Invoke the `careful-mode` skill → loads REFUSE / WARN / ALLOW lists into your working context.
2. Read the last 20 lines of `~/.claude/projects/-Users-hongming-Documents-GitHub-molecule-monorepo/memory/cron-learnings.jsonl`. This tells you:
   - What the previous tick did
   - What the previous tick's `next_action` is expecting from you or from the CEO
   - Any open scope calls

Never skip Step 0. The cron-learnings file is your primary "what did past-me already figure out" signal.

---

## Step 1 — List state

```bash
gh pr list --repo Molecule-AI/molecule-monorepo --state open \
  --json number,title,author,isDraft,mergeable,statusCheckRollup,files

gh pr list --repo Molecule-AI/molecule-controlplane --state open \
  --json number,title,author,isDraft,mergeable

gh issue list --repo Molecule-AI/molecule-monorepo --state open \
  --json number,title,assignees,labels
```

For each new PR and issue (compared to the previous tick's cron-learning), decide: PR-gate flow (Step 2) or issue-triage flow (Step 4).

---

## Step 2 — Seven-gate PR verification

For each open PR:

### Gate 1 — CI

`gh pr checks <N>`. All green? Proceed. Any fail or cancel? Investigate.

- **Cancelled** = superseded by a newer push; rerun via `gh run rerun` if needed.
- **Failed** = read the log (`gh run view <runId> --log-failed`). If the failure is mechanical (lint, import order, flaky fixture), go to Step 2a. If it caught a real bug, go to Step 2d.

### Gate 2 — Build

Usually covered by Gate 1 CI, but confirm the build step specifically passed. On controlplane, that's the `build` job. On monorepo, that's `Platform (Go)` + `Canvas (Next.js)` + `MCP Server (Node.js)`.

### Gate 3 — Tests

- Unit tests in the changed packages (CI covers).
- New regression tests for any bug-fix PR — if the PR claims to fix a bug but has no test proving the bug is fixed, that's a 🟡 in code-review. Trust but verify.

### Gate 4 — Security

- Does the diff touch `handlers/` / `middleware/` / `auth*`? → Gate 4 is HIGH. Run `cross-vendor-review` skill.
- Any `fmt.Sprintf` in SQL? Path traversal risk? YAML injection? Secret-comparison using `!=` instead of `ConstantTimeCompare`? These are the repo's recurring classes — see `security-auditor/system-prompt.md` for the checklist.

### Gate 5 — Design

Does the change fit the system, or is it a local optimum? A PR that adds an env var to work around a structural problem is a 🟡. A PR that replicates a pattern already shipped elsewhere is a 🔵 — ask the author to share / reuse.

### Gate 6 — Line-level review

Invoke the `code-review` skill. 16 criteria. Any 🔴 blocks merge.

### Gate 7 — Playwright if canvas

If the PR touches `canvas/src/**/*.tsx`, run `cd canvas && npm test` locally (or trust the Canvas CI job). For large visual changes, do a manual browser check — the project has a pattern of visual regressions that pass unit tests (dark-theme breaks, hook-rule violations, SSR mismatches).

---

### Step 2a — Mechanical fix on the author's branch

If the fix is truly mechanical:

```bash
gh pr checkout <N>
# make the fix
git add <files>
git commit -m "fix(gate-N): <what you fixed>"
git push
gh run watch
```

Wait for CI. If green, proceed to Step 2b. If still red, you misdiagnosed — back out your change, leave a comment explaining what's wrong, let the author fix it.

### Step 2b — Merge (if approved)

All 7 gates pass + 0 🔴 from code-review + (for noteworthy PRs) cross-vendor-review agreement + (if auth/billing/schema/data-deletion) explicit CEO approval in the chat:

```bash
gh pr merge <N> --merge --delete-branch
```

Never `--squash`, never `--rebase`, never `--admin` bypassing checks.

### Step 2c — Hold for CEO

If the PR touches auth/billing/schema/data-deletion, or if cross-vendor-review disagrees with code-review, or if the PR claims an unverified authority:

1. Leave a comment summarising the gates passed + the concern.
2. Name the exact decision you need from the CEO.
3. Do NOT merge. The tick's cron-learnings `next_action` should read: "CEO to decide X on #N".

### Step 2d — Reject (🔴 finding)

Code-review turned up a red finding, or Gate 4 flagged a security concern:

1. Leave a comment with the exact file:line and the proposed fix.
2. Mark the PR status `changes requested` if you have review permission, otherwise just comment.
3. Do NOT attempt to fix logic yourself. Design-level 🔴 fixes are engineer work.

---

## Step 3 — Docs sync after any merge

If you merged anything this tick that changed behaviour:

1. Invoke `update-docs` skill.
2. The skill opens a `docs/sync-YYYY-MM-DD-tick-N` PR against main.
3. You do NOT merge the docs PR in the same tick — let the next tick (or CEO) review it.

Docs sync measures: test counts (`go test ./... -count=1 -run nothing 2>&1 | grep -c "^=== RUN"` etc.), API route counts, migration counts. NEVER guess — always measure.

---

## Step 4 — Issue pickup (cap 2 per tick)

For each unassigned issue, run gates I-1..I-6:

### I-1 — Is this a real ticket?

Spam, duplicates, "ping" issues. Close as duplicate / not planned with a brief comment.

### I-2 — Does this need a design decision?

If the fix requires choosing between approaches, NOT pickable. Leave a triage comment:
- Summary of the problem as you understand it
- 2–3 option menu
- Your recommendation
- The specific question the CEO needs to answer

### I-3 — Does it touch auth/billing/schema/data-deletion/large-blast-radius?

Noteworthy = explicit CEO approval before pickup. Leave a triage comment asking.

### I-4 — Can you implement alone in < 1 hour?

If the issue needs coordination with another engineer (FE + BE change together, DevOps + migration), delegate through PM instead. You are the triage operator, not the team.

### I-5 — Is there a test path?

If the fix can't be covered by a test you write alongside it, the PR will be un-verifiable. Escalate to Dev Lead.

### I-6 — Does any precondition exist?

Plugin needs to exist before you can wire it. Migration needs to exist before you can query it. Verify preconditions BEFORE self-assigning.

If all 6 pass:

```bash
gh issue edit <N> --add-assignee @me
git checkout -b fix/issue-<N>-<short-slug>
# implement + test
git commit -m "fix: <what>\n\nCloses #<N>"
git push -u origin fix/issue-<N>-<short-slug>
gh pr create --draft
```

Then run `llm-judge` skill against the issue body + PR diff. Score ≥ 4 → mark ready for review. Score ≤ 2 → stay draft, leave a note for yourself in the PR body.

---

## Step 5 — Status report + cron-learnings

Close the tick with a report (posted in chat if user-visible, logged if not). Format:

```
- Merged: #A, #B                            (use "none" if empty)
- Fixed + merged: #C (gate-N fix)
- Fixed + awaiting CI: #D
- Skipped-design: #E (🔴 finding)
- Picked up issue #F → draft PR #G (llm-judge: N/5)
- Skipped issue #H (gate I-2)
- Code-review summary: total 🔴/🟡/🔵
- Cross-vendor pass/escalation
- Docs PR: #K
- Idle reason (if nothing to do)
```

Then append ONE LINE to `cron-learnings.jsonl`:

```json
{"ts":"<ISO-8601>","tick_id":"manual-<N>","category":"workflow","summary":"<terse>","next_action":"<concrete>"}
```

And ONE LINE to `.claude/per-tick-reflections.md`:

```
<ISO-8601> — <what surprised me | what I'd do differently next tick>
```

---

## Cadence discipline

- Cron fires at `:07` and `:37` in manual mode (dev) or hourly at `:17` in full mode.
- If a user types `/triage`, run the full flow on-demand — same steps, same output.
- If the backlog is clean 3 ticks in a row, append a one-line "idle" entry and stop. Don't invent work.

---

## When NOT to triage

- The CEO is mid-conversation on a design decision → don't trigger a concurrent tick mid-thread.
- The Mac mini runner is queued for 2+ hours → CI signals are unreliable; skip Gate 1 merges until runner recovers.
- An incident is live (production down, cert expired, billing broken) → STOP triage, work the incident with the CEO directly.

---

## Escape hatches

If the tick is taking too long:

- Drop the issue-pickup step entirely. Just do PR gates + report.
- Skip the cross-vendor-review for borderline cases; note the skip in cron-learnings.
- Merge only the single-file docs-only PRs if you're in a hurry; leave multi-file PRs for the next tick.

Skipping a gate is always a cron-learning entry. "Skipped cross-vendor on #N due to session pressure — revisit next tick" is a valid line.
