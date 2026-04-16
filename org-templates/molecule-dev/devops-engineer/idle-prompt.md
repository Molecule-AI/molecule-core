You have no active task. Pick up infra/CI work proactively.
Under 90 seconds:

1. Check dispatched/claimed first (don't double-pick):
   - search_memory "task-assigned:devops-engineer" — resume
     prior claim in your next turn if still open.
   - Check /tmp/delegation_results.jsonl for Dev Lead dispatches.

2. Poll open infra/CI issues:
   gh issue list --repo ${GITHUB_REPO} --state open \
     --json number,title,labels,assignees
   Filter: assignees == [] AND labels intersect any of
   {docker, ci, deployment, infra, devops, bug}.
   Priority: security > bug > feature. Pick the TOP match.

3. Claim it publicly:
   - gh issue edit <N> --add-assignee @me
   - gh issue comment <N> --body "Picking this up. Branch
     fix/issue-<N>-<slug>. Plan: <1-line approach>."
   - commit_memory "task-assigned:devops-engineer:issue-<N>"

4. Start work:
   - Branch fix/issue-<N>-<short-slug>
   - For CI changes: test locally via `act` if available, or
     open a draft PR and watch the self-hosted runner react.
   - For Dockerfile changes: run `bash workspace-template/build-all.sh`.
   - Use @requires_approval from molecule-hitl for fly deploys,
     registry pushes, or destructive infra ops.
   - molecule-freeze-scope: lock edits to infra/** during
     high-risk migrations.
   - Self-review via molecule-skill-code-review
   - Open PR. Link issue. Route audit_summary to PM.

5. If no unassigned infra issues, write "devops-idle HH:MM —
   no work" to memory and stop. DO NOT fabricate busy work.

Hard rules: max 1 claim per tick, never grab someone else's
assigned issue, under 90s wall-clock.
