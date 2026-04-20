IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Research cycle with web search. Run every 30 minutes.

1. CHECK RESEARCH BACKLOG:
   search_memory "research-question:technical-researcher"
   gh issue list --repo ${GITHUB_REPO} --state open \
     --label research --label "area:technical-researcher" \
     --json number,title --limit 5

2. WEB SEARCH — for active research questions, use web_search to gather current info:
   - AI agent framework releases (LangChain, CrewAI, AutoGen, Swarm, etc.)
   - MCP server ecosystem updates (new servers, protocol changes)
   - Claude/Anthropic SDK updates, OpenAI API changes
   - Relevant GitHub trending repos in ai-agents topic
   - Conference talks, blog posts, technical papers

3. PLUGIN CURATION (from hourly-plugin-curation):
   - Survey plugins/ and workspace-template/builtin_tools/ for gaps
   - External survey via web_search for new tools worth wrapping
   - File GH issue for 1-3 highest-value plugin proposals

4. SYNTHESIZE findings:
   - What changed since last cycle
   - Impact on Molecule AI platform
   - Recommended actions with priority

5. ROUTING:
   delegate_task to Research Lead with audit_summary (category=plugins).
   commit_memory "tech-research HH:MM — topics researched, findings count"

6. If nothing notable, Research Lead message "clean".
