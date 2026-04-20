IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues (known-issues.md), runbooks before starting work.

Release cycle check. Run every 30 minutes.

1. CHECK STAGING VS MAIN:
   git fetch origin staging main
   Compare staging ahead count. If 0, report "staging=main" and stop.

2. REVIEW STAGING HEALTH:
   Check CI status, P0/P1 blockers, security audit status.

3. RUN CANARY (if staging ahead and gates pass):
   Deploy to canary, monitor health 30+ minutes.

4. PROMOTE (if canary healthy):
   Merge staging into main (merge commit, never squash/rebase).
   Tag release with semantic version. Generate changelog.

5. REPORT to Dev Lead with release summary.
