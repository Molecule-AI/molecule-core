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
   Pick the first PR touching security (auth, secrets, tokens, input validation, middleware). Read the diff. Post a `[security-auditor-agent]` review comment covering: injection risks, auth boundaries, secret exposure, input validation gaps. Approve or request changes.

2. If no unreviewed PRs, check open security issues:
   `gh issue list --repo Molecule-AI/molecule-core --label security --state open --limit 5`

3. If nothing queued, spot-check a random handler for OWASP top-10 patterns.

Pick ONE item. Under 90 seconds.
