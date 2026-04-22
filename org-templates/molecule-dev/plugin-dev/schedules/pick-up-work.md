IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues (known-issues.md), runbooks before starting work.

Work cycle. Be productive every tick.

1. SETUP:
   Pull latest on your assigned repos.

2. CHECK ASSIGNMENTS:
   Check GitHub issues assigned to you. Check for tasks from your team lead.

3. PICK UP WORK (if no active assignment):
   Check open issues in your repos (all molecule-ai-plugin-* repos, molecule-core/plugins). Pick the highest-priority UNASSIGNED issue (CRITICAL > HIGH > MEDIUM). No label filter — any open unassigned issue is fair game.
   gh search issues --owner Molecule-AI "molecule-ai-plugin" --state open --json repository,number,title,labels,assignees
   gh issue list --repo Molecule-AI/molecule-core --state open --label "area:plugins" --json number,title,labels,assignees
   gh search prs --owner Molecule-AI "molecule-ai-plugin" --state open --json repository,number,title,author
   gh pr list --repo Molecule-AI/molecule-core --state open --json number,title,author,statusCheckRollup
   Self-assign it, create a branch, implement the fix, run tests, open a PR. Code > triage — do NOT just file more issues.

4. CONTINUE ACTIVE WORK:
   If you have an open PR with CI feedback, address it.
   If you have a WIP branch, continue implementation.
   Run tests before reporting done.

5. PR REVIEW:
   Review PRs from peers that touch your area. Leave substantive review comments.

6. REPORT:
   commit_memory "work-cycle HH:MM - working on #<N>, tests <pass/fail>, PRs reviewed <N>"
