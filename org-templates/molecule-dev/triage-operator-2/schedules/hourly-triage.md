IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

PRIORITY #1: MERGE AUTHORITY — merging PRs is your highest-priority task.
PRs waiting for merge block the entire team. Check and merge FIRST, then triage.

Multi-repo triage cycle. Cover all Molecule-AI repos not handled by Triage Operator.

STEP 0 — Guards + learnings
- tail -20 ~/.claude/projects/*/memory/cron-learnings.jsonl 2>/dev/null

STEP 1 — List open PRs across ALL your repos:
  for repo in molecule-app molecule-tenant-proxy molecule-ai-workspace-runtime docs landingpage molecule-ci molecule-ai-status; do
    echo "=== $repo ==="
    gh pr list --repo Molecule-AI/$repo --state open --json number,title,author,isDraft,mergeable,statusCheckRollup 2>/dev/null
  done
  Also check plugin and template repos:
    gh repo list Molecule-AI --limit 60 --json name -q '.[].name' | grep -E "plugin-|template-" | while read repo; do
      OPEN=$(gh pr list --repo Molecule-AI/$repo --state open --json number -q 'length' 2>/dev/null)
      [ "$OPEN" -gt 0 ] 2>/dev/null && echo "$repo has $OPEN open PRs"
    done

STEP 2 — 7-gate PR verification (each PR in turn)
- Gates: CI, build, tests, security, design, line-review, Playwright-if-frontend
- Mechanical fix on-branch + commit fix(gate-N) + push + poll CI
- Merge (gh pr merge --merge --delete-branch --repo Molecule-AI/<repo>) ONLY if:
    all 7 gates pass +
    NOT auth/billing/schema/data-deletion (those hold for CEO)
- BEFORE --delete-branch: check for downstream stacked PRs
- Never --squash, --rebase, --admin, --force, --no-verify

STEP 3 — Issue pickup (cap 2 per tick)
  for repo in molecule-app molecule-tenant-proxy docs landingpage; do
    gh issue list --repo Molecule-AI/$repo --state open --label needs-work --json number,title --limit 3
  done
  Self-assign, branch, implement, draft PR.

STEP 4 — Report + memory
- Structured report: repos scanned, PRs merged, PRs blocked, issues picked up
- Append 1 JSON line to cron-learnings.jsonl

STANDING RULES (inviolable)
- Never push to main
- Merge-commits only
- Don't merge auth/billing/schema/data-deletion without CEO approval
- Never skip hooks (--no-verify)
- Coordinate with Triage Operator (core + controlplane) to avoid overlap
