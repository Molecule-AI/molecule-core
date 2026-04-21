IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Market analysis with web search. Run every 30 minutes.

1. CHECK RESEARCH BACKLOG:
   search_memory "research-question:market-analyst"
   gh issue list --repo ${GITHUB_REPO} --state open \
     --label research --label "area:market-analyst" \
     --json number,title --limit 5

2. WEB SEARCH — gather market intelligence:
   - AI agent market sizing (analyst reports, funding rounds)
   - Enterprise AI adoption trends
   - Developer tooling market shifts
   - Pricing model evolution across AI platforms
   - Regulatory developments (EU AI Act, etc.)
   - User research signals (HN, Reddit, Discord)

3. TREND ANALYSIS:
   - Compare current signals against last cycle's snapshot
   - Identify emerging patterns (new use cases, shifting budgets)
   - Track funding rounds in AI agent space

4. ACTIONABLE INSIGHTS:
   For each finding:
   - What it means for Molecule AI
   - Recommended response (product, positioning, pricing)
   - Time sensitivity (act now vs. monitor)

5. ROUTING:
   delegate_task to Research Lead with audit_summary (category=research).
   commit_memory "market-analysis HH:MM — topics analyzed, key findings"

6. If nothing notable, Research Lead message "clean".
