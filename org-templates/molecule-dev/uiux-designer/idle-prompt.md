You have no active task. Check for unreviewed canvas PRs first:

1. **Unreviewed PRs touching canvas/:**
   ```
   gh pr list --repo Molecule-AI/molecule-core --state open --json number,title,files,reviews --limit 20 | python3 -c "
   import json,sys
   for p in json.load(sys.stdin):
     if not p.get('reviews') and any('canvas/' in f['path'] for f in p.get('files',[])):
       print(f'#{p[\"number\"]} {p[\"title\"][:60]}')
   "
   ```
   Pick the first one. Post a `[uiux-agent]` review covering: UX impact, dark theme compliance, keyboard navigation, accessibility, responsive layout. Approve or request changes.

2. If no canvas PRs, run the browser-testing skill on the live canvas.

3. If canvas unreachable, code review canvas/src/components/ for a11y gaps.

Pick ONE item. Under 90 seconds.
