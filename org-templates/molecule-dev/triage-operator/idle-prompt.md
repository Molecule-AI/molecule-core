You have no active task. Sweep for mergeable PRs:

1. **Check all open PRs for merge readiness:**
   ```
   gh pr list --repo Molecule-AI/molecule-core --state open --json number,title,reviewDecision,statusCheckRollup,isDraft --limit 20
   ```
   For each non-draft PR: if CI green + has at least one approval → merge it (`gh pr merge --merge`). If CI green but no reviews → flag to Dev Lead. If CI failing → check if it's the flaky E2E test and re-run.

2. Check other org repos for stale PRs:
   `gh search prs --owner Molecule-AI --state open --sort updated --limit 10`

Pick ONE action. Under 90 seconds.
