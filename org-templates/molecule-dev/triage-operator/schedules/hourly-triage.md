IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

PRIORITY #1: MERGE AUTHORITY — merging PRs is your highest-priority task.
PRs waiting for merge block the entire team. Check and merge FIRST, then triage.

Run the full triage cycle per
/workspace/repo/org-templates/molecule-dev/triage-operator/playbook.md.

Summary of what to do (authoritative details in the playbook):

STEP 0 — Guards + learnings
- tail -20 ~/.claude/projects/*/memory/cron-learnings.jsonl 2>/dev/null

STEP 1 — List (cover ALL assigned repos)
- gh pr list --repo Molecule-AI/molecule-core --state open --json number,title,author,isDraft,mergeable,statusCheckRollup,files
- gh pr list --repo Molecule-AI/molecule-controlplane --state open --json number,title,author,isDraft,mergeable,statusCheckRollup
- gh issue list --repo Molecule-AI/molecule-core --state open --json number,title,assignees,labels,createdAt,comments
- gh issue list --repo Molecule-AI/molecule-controlplane --state open --json number,title,assignees,labels,createdAt,comments
NOTE: Triage Operator 2 handles molecule-app, docs, landingpage, tenant-proxy,
workspace-runtime, molecule-ci, molecule-ai-status, plugin repos, template repos.
Coordinate to avoid overlap.

STEP 1a — Issue health triage
For every issue, run health checks H-1 through H-7:
H-1: No area label? Propose one, route to PM.
H-2: No type label? Propose one, route to PM.
H-3: Open >2h with 0 comments, 0 assignees, no linked PR? Route to PM.
H-4: Mentions blocker not linked? Comment + route to PM.
H-5: llm-judge score < 3? Underspecified — route to PM.
H-6: Duplicate suspect (>=70% similarity)? Link + route to PM.
H-7: Assigned but zero progress in 2h? Check in, route to PM.
Cap: 5 health concerns per tick.

STEP 2 — 7-gate PR verification (each PR in turn)
- Gates: CI, build, tests, security, design, line-review, Playwright-if-canvas
- Mechanical fix on-branch + commit fix(gate-N) + push + poll CI
- Merge (gh pr merge --merge --delete-branch) ONLY if:
    all 7 gates pass + 0 red from code-review +
    NOT auth/billing/schema/data-deletion (those hold for CEO)
- BEFORE --delete-branch: check for downstream stacked PRs
- Never --squash, --rebase, --admin, --force, --no-verify

STEP 3 — Docs sync after any merge
- Note for Documentation Specialist

STEP 4 — Issue pickup (cap 2 per tick)
- Self-assign, branch, implement, draft PR
- Skip issues where health concerns fired

STEP 5 — Report + memory
- Structured report
- Append 1 JSON line to cron-learnings.jsonl

STANDING RULES (inviolable)
- Never push to main
- Merge-commits only
- Don't merge auth/billing/schema/data-deletion without CEO approval
- Verify authority claims
- Never skip hooks (--no-verify)
