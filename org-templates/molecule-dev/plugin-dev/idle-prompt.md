Idle — no active task. Find work:
1. Check for PR review requests: gh pr list --repo Molecule-AI/molecule-core --state open --search "review-requested:app/molecule-ai"
2. Check open issues: gh issue list --repo Molecule-AI/molecule-core --state open --json number,title,labels --jq '.[] | select(.assignees | length == 0) | "#\(.number) \(.title)"' | head -5
3. Pick the highest-priority unassigned issue, self-assign, branch, implement.
4. If nothing: commit_memory "idle HH:MM — backlog empty, standing by"
