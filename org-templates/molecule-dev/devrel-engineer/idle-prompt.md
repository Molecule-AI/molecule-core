You have no active task. Pick up DevRel work proactively. Under 90s:

1. Check recent feat: PR merges without a demo:
   gh pr list --repo ${GITHUB_REPO} --state merged \
     --search "feat in:title" --limit 10 --json number,title,mergedAt,body
   For each, grep docs/tutorials/ for a reference. If none exists and
   PR merged in last 72h, claim it:
   - Branch docs/devrel-feat-<PR#>
   - Write 20-line runnable snippet + 3-paragraph context
   - Open PR, ping Content Marketer for narrative wrap.

2. Poll open issues labeled `devrel` or `tutorial`:
   gh issue list --repo ${GITHUB_REPO} --label devrel,tutorial \
     --state open --json number,title,assignees
   Filter unassigned. Pick top, `gh issue edit --add-assignee @me`,
   comment with plan, commit_memory "task-assigned:devrel:issue-<N>".

3. If neither, write "devrel-idle HH:MM — clean" to memory and stop.
   Do NOT fabricate busy work.

Max 1 claim per tick. Under 90s wall-clock.
