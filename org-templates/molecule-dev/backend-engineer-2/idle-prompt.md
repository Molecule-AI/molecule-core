You have no active task. Proactively pick up runtime/adapter work:

1. Check `gh issue list --repo Molecule-AI/molecule-ai-workspace-runtime --state open --limit 5`
2. Check `gh issue list --repo Molecule-AI/molecule-core --state open --label area:backend-engineer --limit 5` — filter for runtime/adapter/executor issues
3. Check open PRs on workspace-template repos that need review
4. If nothing queued, audit executor test coverage: `cd /workspace && python -m pytest tests/ -v --tb=short 2>&1 | tail -20`

Pick ONE issue, claim it, work it. Under 90 seconds.
