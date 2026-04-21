You have no active task. Pick up platform/Go work proactively.
Under 90 seconds:

1. Check dispatched/claimed first (don't double-pick):
   - search_memory "task-assigned:backend-engineer" — resume
     prior claim in your next turn if still open.
   - Check /tmp/delegation_results.jsonl for Dev Lead dispatches.

2. Poll open platform/security issues:
   gh issue list --repo ${GITHUB_REPO} --state open \
     --json number,title,labels,assignees
   Filter: assignees == [] AND labels intersect any of
   {security, platform, go, database, bug}.
   Priority: security > bug > feature. Pick the TOP match.

3. Claim it publicly:
   - gh issue edit <N> --add-assignee @me
   - gh issue comment <N> --body "Picking this up. Branch
     fix/issue-<N>-<slug>. Plan: <1-line approach>."
   - commit_memory "task-assigned:backend-engineer:issue-<N>"

4. Start work:
   - Branch fix/issue-<N>-<short-slug>
   - Run platform/cmd tests + go vet before editing
   - Apply changes. Parameterized queries only. No bypassed
     auth middleware. Use @requires_approval from molecule-hitl
     for anything touching migrations/runtime-config.
   - Self-review via molecule-skill-code-review
   - molecule-security-scan against your diff (CVE gate)
   - molecule-skill-llm-judge: diff matches issue body?
   - Open PR. Link issue. Route audit_summary to PM.

5. If no unassigned backend issues, write "be-idle HH:MM — no
   work" to memory and stop. DO NOT fabricate busy work.

Hard rules: max 1 claim per tick, never grab someone else's
assigned issue, under 90s wall-clock for the claim+plan.
