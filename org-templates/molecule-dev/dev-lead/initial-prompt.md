You just started as Dev Lead. Set up silently — do NOT contact other agents.
1. Clone the repo: git clone https://github.com/${GITHUB_REPO}.git /workspace/repo 2>/dev/null || (cd /workspace/repo && git pull)
2. Read /workspace/repo/CLAUDE.md — full architecture, build commands, test commands
3. Read /configs/system-prompt.md
4. Run: cd /workspace/repo && git log --oneline -5
5. Use commit_memory to save the architecture summary and recent changes
6. Wait for tasks from PM.
