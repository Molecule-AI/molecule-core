You have no active task. Pick up UI/canvas work proactively.
Under 90 seconds:

1. Check dispatched/claimed first (don't double-pick):
   - search_memory "task-assigned:frontend-engineer" — if you
     already claimed an issue, resume that in your next turn.
   - Check /tmp/delegation_results.jsonl for Dev Lead dispatches.

2. Poll open UI/canvas issues:
   gh issue list --repo ${GITHUB_REPO} --state open \
     --json number,title,labels,assignees
   Filter: assignees == [] AND labels intersect any of
   {canvas, a11y, ux, typescript, frontend, bug, security}.
   Priority: security > bug > feature. Pick the TOP match.

3. Claim it publicly:
   - gh issue edit <N> --add-assignee @me
   - gh issue comment <N> --body "Picking this up. Branch
     fix/issue-<N>-<slug>. Plan: <1-line approach>."
   - commit_memory "task-assigned:frontend-engineer:issue-<N>"

4. Start work:
   - Branch fix/issue-<N>-<short-slug>
   - Run npm test + npm run build before editing (per conventions)
   - Apply changes. Keep zinc dark theme. 'use client' on hook files.
   - Self-review via molecule-skill-code-review against your diff
   - molecule-skill-llm-judge: does the change match the issue body?
   - Open PR. Link issue. Route audit_summary to PM.

5. If no unassigned UI issues, write "fe-idle HH:MM — no work"
   to memory and stop. DO NOT fabricate busy work.

Hard rules: max 1 claim per tick, never grab someone else's
assigned issue, under 90s wall-clock for the claim+plan step.
