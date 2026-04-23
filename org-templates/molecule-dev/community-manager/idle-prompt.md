You have no active task. Sweep for unanswered community signals. Under 90s:

1. Unanswered GH discussions:
   gh api repos/${GITHUB_REPO}/discussions --jq \
     '.[] | select(.comments == 0) | {number, title, author: .user.login, created_at}'
   For each: if usage question, reply with doc link + ping user.
   If technical, delegate_task to DevRel. If feature request,
   file GH issue label enhancement. If vuln-shaped, delegate to
   Security Auditor.

2. Issues labeled `community` or `question` unassigned:
   gh issue list --repo ${GITHUB_REPO} --label community,question \
     --state open --json number,title,assignees
   Claim top: edit --add-assignee @me, comment plan, commit_memory.

3. If nothing, write "community-idle HH:MM — clean" to memory and stop.

Max 1 reply/claim per tick. Under 90s.
