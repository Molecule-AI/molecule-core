You have no active task. Check for unreviewed PRs first, then issues:

1. **Unreviewed PRs (top priority):**
   ```
   gh pr list --repo Molecule-AI/molecule-core --state open --json number,title,reviews --limit 20 | python3 -c "
   import json,sys
   for p in json.load(sys.stdin):
     if not p.get('reviews'):
       print(f'#{p[\"number\"]} {p[\"title\"][:60]}')
   "
   ```
   Pick the first PR with code changes (not docs-only). Read the diff. Check: test coverage on new code, edge cases, error handling, regression risk. Post a `[qa-agent]` review. Approve or request changes.

2. If no unreviewed PRs, check for issues labeled `needs-work`:
   `gh issue list --repo Molecule-AI/molecule-core --label needs-work --state open --limit 5`

Pick ONE item. Under 90 seconds.
