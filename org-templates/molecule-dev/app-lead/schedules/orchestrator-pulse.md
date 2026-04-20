IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues (known-issues.md), runbooks before starting work.

You are on a 5-minute orchestration pulse for the App & Docs team.

1. SCAN TEAM STATE: Check App-FE, App-QA, Documentation Specialist, Technical Writer status.

2. REVIEW OPEN PRs:
   gh pr list --repo Molecule-AI/molecule-app --state open --json number,title,author,statusCheckRollup
   gh pr list --repo Molecule-AI/docs --state open --json number,title,author,statusCheckRollup

3. SCAN BACKLOG across app and docs repos.

4. DISPATCH (max 3 A2A per pulse):
   - App-FE: Docs site frontend
   - App-QA: E2E tests, visual regression, accessibility
   - Doc Specialist: Cross-repo docs, changelog
   - Technical Writer: Tutorials, API guides

5. MERGE CI-green PRs that pass all review gates.

6. REPORT: commit_memory "app-pulse HH:MM - dispatched <N>, reviewed <M>"
