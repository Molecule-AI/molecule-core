# MCP Server List Launch — Social Copy
Campaign: mcp-server-list | Blog: `docs/blog/2026-04-25-mcp-server-list/index.md`
Publish day: TBD — coordinate with Marketing Lead
Assets: visual assets at `docs/marketing/campaigns/mcp-server-list/assets/`

---
**NOTE:** This copy is ready for human social media execution if Social Media Brand workspace remains FAILED.

---

## X (Twitter) — Primary thread (5 posts)

### Post 1 — Hook
Molecule AI agents can use 20+ MCP tools out of the box.

Browser automation. GitHub access. Slack notifications. Vector DB queries. AWS ops. And that's just the start.

MCP (Model Context Protocol) is how you connect your agent to anything you already use — without writing a custom integration.

→ https://docs.molecule.ai/blog/mcp-server-list-2026

---

### Post 2 — The MCP problem it solves
The problem with AI agent integrations:

→ Every agent decides its own way to call external tools
→ Your Slack integration only works with that one agent
→ Switching agents means re-building everything

MCP (Model Context Protocol) standardizes the interface. One MCP server works with any MCP-compatible agent.

Molecule AI ships with MCP built in — no wrapper code required.

→ https://docs.molecule.ai/blog/mcp-server-list-2026

---

### Post 3 — What's available today
Molecule AI's MCP ecosystem today:

→ @modelcontextprotocol/server-browser — headless Chrome automation
→ @modelcontextprotocol/server-github — repos, issues, PRs
→ @modelcontextprotocol/server-slack — send messages, list channels
→ @modelcontextprotocol/server-filesystem — read/write, scoped to workspace
→ + AWS, Datadog, Postgres, Pinecone, and more

Every server is org-scoped. Your agents access what your org grants them.

→ https://docs.molecule.ai/blog/mcp-server-list-2026

---

### Post 4 — Governance angle
Most MCP tutorials show you how to connect tools.

Molecule AI adds the part that's easy to skip: governance.

→ Per-workspace MCP tool access control
→ Org API key attribution on every MCP call
→ Instant revocation — no redeploy
→ Audit trail exportable for compliance

Your security team will appreciate this.

→ https://docs.molecule.ai/blog/mcp-server-list-2026

---

### Post 5 — CTA
The MCP ecosystem is growing fast.

Molecule AI's MCP layer gives you: one interface, any compatible agent, org-level governance on top.

20+ servers available now. More added each release.

Browse the full list with runnable examples:

→ https://docs.molecule.ai/blog/mcp-server-list-2026

---

## LinkedIn — Single post

**Title:** The MCP ecosystem is exploding. Here's how Molecule AI fits in.

**Body:**

If you're building with AI agents, you've probably noticed: every agent runtime has its own way to talk to external tools. The Slack integration you built for one agent doesn't port to the next. Adding a new tool means writing custom adapter code from scratch.

The Model Context Protocol (MCP) changes this. MCP standardizes the interface between agents and the tools they use — so your GitHub integration works the same way whether your agent is built on LangGraph, Claude Code, CrewAI, or anything else.

Molecule AI ships with MCP built in. We maintain a curated ecosystem of MCP servers that work with any MCP-compatible agent running in a Molecule AI workspace:

→ Browser automation (headless Chrome via CDP)
→ GitHub (repos, issues, PRs, actions)
→ Slack (send messages, list channels)
→ Filesystem (workspace-scoped read/write)
→ AWS, Datadog, Postgres, Pinecone, and more

What makes this different from a standard MCP setup: governance is included.

Every MCP call is routed through Molecule AI's org API key layer. You know which agent called which tool, what arguments it passed, and what came back. Revocation is instant — one API call, the access is gone.

This is the difference between "our agents use MCP" and "our MCP usage is auditable, attributable, and controlled."

Browse the full list with runnable code examples:

→ https://docs.molecule.ai/blog/mcp-server-list-2026

---

## Campaign notes

**Audience:** Developers (X), Platform engineers / DevRel (LinkedIn)
**Tone:** Practical, inventory-focused. Lead with the breadth and variety of the ecosystem. Governance is the differentiator — mention it in post 4 and LinkedIn.
**Differentiation:** MCP is open, but MCP + org-level governance + workspace-scoped access is Molecule AI-specific. Don't just list servers — emphasize the org security layer.
**Suggested image:** `docs/marketing/campaigns/mcp-server-list/assets/mcp-server-list-hero.png` (1200x630)
Alt-text: "Molecule AI MCP server ecosystem showing 20+ integrations (browser, GitHub, Slack, AWS, filesystem) connected to a Molecule AI workspace with org-level governance layer"
Social card: `docs/marketing/campaigns/mcp-server-list/assets/mcp-server-list-social-card.png` (1080x1080)
**Hashtags:** #MCP #AIAgents #AgenticAI #MoleculeAI #PlatformEngineering
**Coordination:** Publish after blog goes live. Coordinate with Marketing Lead on timing. Suggested spacing: Day 2 of MCP launch week (after Chrome DevTools MCP Day 1).
**Social Media Brand status:** Copy ready for manual execution by a human with X/LinkedIn access.