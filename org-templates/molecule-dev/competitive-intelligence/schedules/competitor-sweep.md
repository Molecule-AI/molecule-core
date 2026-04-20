IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Competitor sweep with web search. Run every 30 minutes.

1. CHECK RESEARCH BACKLOG:
   search_memory "research-question:competitive-intelligence"
   gh issue list --repo ${GITHUB_REPO} --state open \
     --label research --label "area:competitive-intelligence" \
     --json number,title --limit 5

2. WEB SEARCH — scan competitors for changes:
   - Hermes Agent: new releases, pricing, features
   - Letta (MemGPT): framework updates, enterprise offerings
   - n8n: AI agent features, marketplace
   - LangChain/LangSmith: platform evolution
   - CrewAI: enterprise features, integrations
   - Other emerging AI agent platforms

3. COMPETITIVE MATRIX UPDATE:
   Compare findings against docs/marketing/competitors.md.
   If competitor shape/pricing/differentiation changed, flag to PMM + Marketing Lead.

4. THREAT ANALYSIS:
   - New competitor features we lack -> flag with priority
   - Competitor weaknesses we can capitalize on -> opportunity
   - Market positioning shifts -> update recommendations

5. ROUTING:
   delegate_task to Research Lead with audit_summary (category=research).
   commit_memory "comp-sweep HH:MM — competitors scanned, changes found"

6. If nothing changed, Research Lead message "clean".
