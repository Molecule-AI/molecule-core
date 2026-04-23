# Phase 34 Launch — Discord Community Announcement

**Suggested announcement:**

---

👋 **Phase 34 is live — here's what's new**

Hey @devs — Phase 34 just shipped. Four features dropping today, April 30, 2026.

**🔍 Tool Trace** *(the one you'll use first)*
Every tool call now logs its reasoning in `Message.metadata.tool_trace`. Debug your agent's decisions, audit call chains, build observability on top of it. The data's there — it's on you to use it.

```python
trace = message.metadata.tool_trace
# [{tool, reasoning, input, output}, ...]
```

**🏛️ Platform Instructions** *(for org admins)*
Set org-wide governance rules that every agent in your org inherits at session start. Think: "tag resources with cost_center", "always confirm before prod deploys". Rules travel with the org.

**🔑 Partner API Keys** *(for builders)*
Scoped `mol_pk_*` tokens for programmatic org provisioning. Provision workspaces under your org with a token scoped to exactly what it needs — nothing more.

**🌐 SaaS Federation v2** *(for multi-tenant setups)*
Cross-org agent identity and delegation. Define trust policies that let agents in partner orgs act within explicit scopes.

**What to try first**
Tool Trace. Spin up a session, run a task, inspect `metadata.tool_trace`. You'll immediately see where your agent's reasoning matches — and where it surprises you.

Links: [Docs](https://docs.molecule.ai) · [Changelog](https://docs.molecule.ai/changelog) · [GitHub](https://github.com/Molecule-AI/molecule)

Questions? Drop them here — someone from the team will answer. 🙏
