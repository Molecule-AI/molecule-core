IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Multi-repo security audit. Rotate across org repos every cycle.

1. SETUP — pick 2-3 repos to audit this cycle:
   REPOS=(molecule-controlplane molecule-app molecule-tenant-proxy
          molecule-ai-workspace-runtime docs landingpage molecule-ci)
   # Rotate: read last-audited from memory, pick repos not audited last cycle
   LAST=$(cat /tmp/last-security-repos 2>/dev/null || echo "")
   Pick 2-3 repos not in $LAST. Save selection to /tmp/last-security-repos.

2. FOR EACH REPO:
   Clone/pull the repo under /workspace/repos/.

   a. STATIC ANALYSIS on changed files (last 48h):
      - Go: gosec -quiet <files>
      - Python: bandit -ll <files>
      - JS/TS: check for eval(), dangerouslySetInnerHTML, unescaped user input

   b. SECRETS SCAN: last 20 commits grepped for token patterns
      (sk-ant, sk-or, api_key=, GITHUB_TOKEN=) excluding test files.

   c. DEPENDENCY AUDIT:
      - npm audit (if package.json)
      - go mod tidy + check for CVEs (if go.mod)

   d. OPEN PR REVIEW:
      gh pr list --repo Molecule-AI/${repo} --state open --json number
      For each: gh pr diff | grep '^+' for injection/exec/unsafe patterns.

3. FILE ISSUES for every HIGH+ finding:
   Dedupe: gh issue list --repo Molecule-AI/<repo> --search "<category>" --state open
   gh issue create with severity, file:line, repro, proposed fix.

4. ROUTING:
   delegate_task to PM with summary: repos audited, severity counts, issue numbers.

5. MEMORY:
   commit_memory key='multi-repo-security-audit-latest'.

6. If clean: delegate_task to PM with "clean, audited <repos>, no new findings."

Coordinate with Security Auditor (molecule-core primary) to avoid duplicate coverage.
