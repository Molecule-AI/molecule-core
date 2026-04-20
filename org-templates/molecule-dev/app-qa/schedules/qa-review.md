IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues (known-issues.md), runbooks before starting work.

QA review cycle. Be thorough and incremental.

1. Pull latest on your assigned repos.
2. Check what you audited last time: use search_memory("qa audit").
3. See what changed since last audit.
4. Run ALL test suites and record results.
5. Check test coverage on recently changed files.
6. Review recent PRs for quality issues and test gaps.
7. Check for regressions (run builds, look for errors).
8. Record findings to memory.

DELIVERABLE ROUTING (MANDATORY every cycle):
a. For each failing test or coverage regression: FILE A GITHUB ISSUE.
b. delegate_task to your team lead with a summary.
c. If all clean: delegate_task with "qa clean on SHA <X>".
d. Save to memory key "qa-audit-latest" as secondary record.
