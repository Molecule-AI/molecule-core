Plugin ecosystem audit. Run this EVERY cycle — you own every molecule-ai-plugin-* repo.

## Step 1: Discover all plugin repos (NEVER use a hardcoded list)
```bash
gh repo list Molecule-AI --limit 100 --json name,updatedAt \
  | jq -r '.[] | select(.name | startswith("molecule-ai-plugin-")) | "\(.name) \(.updatedAt)"' \
  | sort
```
Save the count. If it changed since last cycle, investigate new repos.

## Step 2: Health check each repo
For each plugin repo discovered above:
```bash
REPO="Molecule-AI/<name>"
# CI status
gh run list --repo $REPO --limit 1 --json conclusion,createdAt
# Open issues
gh issue list --repo $REPO --state open --json number,title --limit 5
# Open PRs
gh pr list --repo $REPO --state open --json number,title --limit 5
# Last commit age
gh api repos/$REPO/commits?per_page=1 --jq '.[0].commit.committer.date'
```

## Step 3: Triage and act
- **CI red**: fix it NOW — clone, diagnose, push fix
- **Open issues > 0**: self-assign the highest-priority one, start working
- **Stale PR**: review it, approve or request changes
- **Last commit > 7 days**: check if the plugin is feature-complete or abandoned. If abandoned, file an issue.
- **No README or empty README**: write one
- **No tests**: add basic tests

## Step 4: Core pipeline check
```bash
cd /workspace/repos/molecule-core
git pull
# Check for plugin pipeline changes
git log --oneline --since="24 hours ago" -- workspace/plugins_registry/
```
If pipeline changed, verify all plugins still install correctly.

## Step 5: Report
```
commit_memory "plugin-audit HH:MM — N repos, CI: X green / Y red, issues: Z open, acted on: <list>"
```

RULE: Do NOT just report numbers. If something is broken, FIX IT in this cycle.
