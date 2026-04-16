Weekly survey of `plugins/` and `workspace-template/builtin_tools/` for
evolution opportunities. The team should keep gaining capabilities.

1. Inventory:
   - ls plugins/ — every plugin and its plugin.yaml description
   - ls workspace-template/builtin_tools/*.py — every builtin tool
   - cat org-templates/molecule-dev/org.yaml — see how plugins are wired
2. Gap analysis:
   - Any builtin_tool not exposed via a plugin?
   - Any role with no plugins beyond defaults that *should* have extras?
   - Any plugin that's installed everywhere via defaults but is rarely used?
3. External survey (use browser-automation):
   - github.com/topics/ai-agents (last week)
   - github.com/topics/mcp-server (last week)
   - claude.ai/cookbook, openai/swarm releases
   - anthropic blog, openai blog, langchain blog (last week)
4. For 1-3 highest-value findings, file a GH issue with concrete proposal:
   - "Plugin proposal: <name> — wraps <upstream tool> for <role(s)>"
   - body: what it does, which roles benefit, integration sketch (~30 lines),
     upstream link, license check.
5. Routing: delegate_task to PM with audit_summary metadata
   (category=plugins, issues=[…], top_recommendation=…).
6. If nothing notable this week, PM-message a one-line "clean".
