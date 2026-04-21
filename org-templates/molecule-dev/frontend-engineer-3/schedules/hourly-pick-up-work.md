IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Independent work cycle for docs site. Find work, write content, push, open PR, return to main. FULL CYCLE REQUIRED.

STEP 1 — CHECK CURRENT STATE:
  cd /workspace/repo
  If NOT on main: push previous work first.
    git push origin $(git branch --show-current)
    gh pr create --base main --title "docs: description" --body "description" 2>/dev/null || true
    git checkout main && git pull origin main

STEP 2 — FIND WORK:
  gh issue list --repo Molecule-AI/docs --state open --json number,title,labels,assignees --jq '.[] | select(.assignees | length == 0) | "#\(.number) \(.title)"'
  Also check: recent merged PRs in molecule-core and molecule-controlplane that need docs updates.

STEP 3 — SELF-ASSIGN:
  gh issue edit <NUMBER> --repo Molecule-AI/docs --add-assignee @me

STEP 4 — WRITE CONTENT:
  git checkout -b docs/issue-N-description
  Write/update documentation. Build check:
    npm install && npm run build
  git add && git commit -m "docs: description (closes #N)"

STEP 5 — PUSH + OPEN PR:
  git push origin <branch>
  gh pr create --base main --title "docs: description" --body "Closes #N"

STEP 6 — RETURN TO MAIN:
  git checkout main && git pull origin main
  MANDATORY.

RULES: Build must pass. All links must resolve. Dark theme.
