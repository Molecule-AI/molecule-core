IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Independent work cycle for molecule-app (Next.js SaaS). Find work, write code, push, open PR, return to staging. FULL CYCLE REQUIRED.

STEP 1 — CHECK CURRENT STATE:
  cd /workspace/repo
  If NOT on staging: push previous work first.
    git fetch origin staging && git rebase origin/staging
    git push origin $(git branch --show-current)
    gh pr create --base staging --title "fix: description" --body "description" 2>/dev/null || true
    git checkout staging && git pull origin staging

STEP 2 — FIND WORK:
  gh issue list --repo Molecule-AI/molecule-app --state open --json number,title,labels,assignees --jq '.[] | select(.assignees | length == 0) | "#\(.number) \(.title)"'

STEP 3 — SELF-ASSIGN:
  gh issue edit <NUMBER> --repo Molecule-AI/molecule-app --add-assignee @me

STEP 4 — WRITE CODE:
  git checkout -b fix/issue-N-description
  Write code. Run self-check:
    for f in $(grep -rl "useState\|useEffect\|useCallback\|useMemo\|useRef" src/ --include="*.tsx"); do
      head -3 "$f" | grep -q "use client" || echo "MISSING 'use client': $f"
    done
  npm test && npm run build
  git add && git commit -m "fix(app): description (closes #N)"

STEP 5 — PUSH + OPEN PR:
  git fetch origin staging && git rebase origin/staging
  git push origin <branch>
  gh pr create --base staging --title "fix(app): description" --body "Closes #N"

STEP 6 — RETURN TO STAGING:
  git checkout staging && git pull origin staging
  MANDATORY.

RULES: All PRs target staging. Rebase before push. Merge-commits only. Dark theme only.
