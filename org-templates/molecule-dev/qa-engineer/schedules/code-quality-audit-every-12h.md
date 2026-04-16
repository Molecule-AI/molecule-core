Recurring code quality audit. Be thorough and incremental.

1. Pull latest: cd /workspace/repo && git pull
2. Check what you audited last time: use search_memory("qa audit") to recall prior findings
3. See what changed since last audit: git log --oneline --since="12 hours ago"
4. Run ALL test suites and record results:
   cd /workspace/repo/platform && go test -race ./... 2>&1 | tail -20
   cd /workspace/repo/canvas && npm test 2>&1 | tail -10
   cd /workspace/repo/workspace-template && python -m pytest --tb=short -q 2>&1 | tail -10
5. Check test coverage on recently changed files:
   - For each changed Python file, check if it has corresponding tests
   - For each changed Go handler, check if it has test coverage
   - For each changed .tsx component, check if it has a .test.tsx
6. Review recent PRs for quality issues:
   cd /workspace/repo && gh pr list --state merged --limit 5
   For each: check if tests were added, if docs were updated, if 'use client' is present on hook-using .tsx
7. Check for regressions:
   cd /workspace/repo/canvas && npm run build 2>&1 | tail -5
   Look for TypeScript errors, missing exports, build warnings
8. Record your findings to memory:
   Use commit_memory with key "qa-audit-latest" and value containing:
   - Date and commit hash audited up to
   - Test counts (Go, Python, Canvas) and pass/fail status
   - Files with missing test coverage
   - Quality issues found
   - Areas to investigate deeper next time
=== FINAL STEP — DELIVERABLE ROUTING (MANDATORY every cycle) ===

a. For each failing test, build break, or coverage regression: FILE A GITHUB ISSUE:
   - Dedupe: gh issue list --repo Molecule-AI/molecule-monorepo --search "<suite>" --state open
   - If new: gh issue create --title "qa: <suite> — <short>" --body with failure log, commit SHA,
     reproducer command, suspected file:line, proposed approach
   - Capture issue numbers for the PM summary.

b. delegate_task to PM with a summary: audit SHA, test counts (Go/Python/Canvas),
   pass/fail, new issue numbers, top 3 risks. PM routes to dev.

c. If all clean: delegate_task to PM with "qa clean on SHA <X>" so the audit is observable.

d. Save to memory key 'qa-audit-latest' as a secondary record only.
