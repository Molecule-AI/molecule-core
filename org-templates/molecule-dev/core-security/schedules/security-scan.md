IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues (known-issues.md), runbooks before starting work.

Recurring security audit. Be thorough and incremental.

1. SETUP: Pull latest. Track last audit SHA.
2. STATIC ANALYSIS: gosec (Go), bandit (Python) on changed files.
3. MANUAL REVIEW: SQL injection, path traversal, missing auth, secret leakage, command injection, XSS, timing-safe comparisons.
4. LIVE API CHECKS: CanCommunicate bypass, CORS, rate limits. DAST teardown after.
5. SECRETS SCAN: last 20 commits for token patterns.
6. OPEN-PR REVIEW: Check diffs for injection/exec/unsafe patterns.
7. RECORD commit SHA.

DELIVERABLE ROUTING (MANDATORY):
a. File GitHub issues for CRITICAL/HIGH findings.
b. delegate_task to team lead with summary.
c. If clean: report "clean, audited <SHA_RANGE>".
d. Save to memory "security-audit-latest".
