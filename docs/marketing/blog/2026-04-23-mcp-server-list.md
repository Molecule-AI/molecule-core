---
title: "The Complete MCP Server List for Molecule AI (2026)"
slug: mcp-server-list
date: 2026-04-23
authors: [molecule-ai]
tags: [mcp, tools, integrations, agent-platform]
description: "Browse every MCP server available in Molecule AI — browser automation, code execution, file access, and more. Updated for 2026."
og_image: /assets/blog/2026-04-23-mcp-server-list/og.png
---

# Every MCP Server Available in Molecule AI (2026)

Model Context Protocol (MCP) is the open standard that lets AI agents discover and call external tools at runtime — without hard-coded integrations. Instead of shipping a new SDK version every time you add a capability, you expose a server that speaks MCP, and every compliant agent can use it immediately.

For agent builders, the server list is the practical question: *what can my agent actually do?* This page is the authoritative catalogue of every MCP server available in Molecule AI today, grouped by category, with notes on what each one does and its current availability status.

---

## Full MCP Server Catalogue

### Browser and Web Automation

| Server | Category | What it does | Status |
|--------|----------|--------------|--------|
| Chrome DevTools MCP | Browser/Web | Drives a real Chrome browser via the DevTools Protocol — navigate pages, click elements, fill forms, capture screenshots, read the live DOM | GA |
| Playwright MCP | Browser/Web | Headless browser automation via Playwright — end-to-end web flows, cross-browser testing, structured data extraction from rendered pages | GA |

The Chrome DevTools MCP integration ships as a first-class platform feature. For a full walkthrough of what it enables — live DOM inspection, agentic form fills, screenshot capture mid-task — see the [Chrome DevTools MCP blog post](/docs/blog/2026-04-20-chrome-devtools-mcp/).

---

### Cloud Infrastructure

| Server | Category | What it does | Status |
|--------|----------|--------------|--------|
| Cloudflare Artifacts | Cloud | Git-backed versioned workspace snapshots — agents can fork, commit, and roll back their own working state on Cloudflare's edge | GA |
| EC2 Instance Connect | Cloud | Establishes short-lived SSH sessions to AWS EC2 instances using IAM-scoped tokens — no long-lived credentials stored | GA |

Cloudflare Artifacts deserves a callout: it treats every workspace snapshot as a real Git commit, giving agents branching, rollback, and multi-agent collaboration over a shared working tree. Full details in the [Cloudflare Artifacts integration post](/docs/blog/2026-04-21-cloudflare-artifacts/).

---

### Code Execution

| Server | Category | What it does | Status |
|--------|----------|--------------|--------|
| Sandbox (Node.js) | Code execution | Runs JavaScript in an isolated, network-sandboxed V8 environment — safe for untrusted agent-generated code | GA |
| Sandbox (Python) | Code execution | Executes Python in a resource-limited container — supports pip-installed packages via a pre-warmed layer | GA |
| Bash | Code execution | Executes shell commands in an ephemeral workspace shell — scoped to the workspace directory, no persistent state between invocations | GA |

Sandbox backends enforce hard resource limits: CPU time, memory, and network egress caps are set at the org level and enforced at the kernel layer. Agents cannot escalate past what the org policy permits.

---

### File and Storage

| Server | Category | What it does | Status |
|--------|----------|--------------|--------|
| WriteFile | File/Storage | Writes or overwrites a file at a given workspace path — supports atomic writes and optional base64 encoding for binary content | GA |
| ReadFile | File/Storage | Reads a file from the workspace by path — returns content with optional line-range limits to avoid context overflow | GA |
| Glob | File/Storage | Pattern-matches files across the workspace tree — returns matching paths sorted by modification time | GA |
| Grep | File/Storage | Full-text and regex search across workspace files — returns matching lines with configurable context, file-type filters, and match counts | GA |

These four tools form the agent filesystem toolkit. They're intentionally primitive — each does one thing well — and compose cleanly for workflows like "find all TODO comments, read the surrounding context, rewrite and write back."

---

### Communication

| Server | Category | What it does | Status |
|--------|----------|--------------|--------|
| Slack adapter | Communication | Posts messages and threads to Slack channels, reads channel history, reacts to messages — authenticated via bot OAuth token | GA |
| Discord adapter | Communication | Sends messages to Discord channels, manages threads, reads message history — scoped to bot permissions set at install time | GA |

Both adapters support structured message blocks, not just plain text. Agents can format Slack messages with Block Kit components or Discord messages with embeds — useful for surfacing structured summaries or pipeline results to a human-readable channel.

---

### Custom and Community Servers

| Server | Category | What it does | Status |
|--------|----------|--------------|--------|
| Custom MCP server | Custom/Community | Bring-your-own server exposing any tools you define — registered via the workspace config API | Available |
| Community registry | Custom/Community | Third-party servers submitted via the partner program — reviewed for security and listed here when approved | Rolling |

Adding a custom MCP server takes three steps: implement the MCP spec (open source reference implementations exist for TypeScript, Python, and Go), expose it over HTTPS, and register it in your workspace config:

```bash
curl -X POST https://api.molecule.ai/workspaces/:id/mcp-servers \
  -H "Authorization: Bearer $MOL_WS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-custom-server",
    "url": "https://mcp.example.com",
    "description": "Internal data warehouse query tool"
  }'
```

Any agent in that workspace can call the server's tools immediately — no redeployment, no SDK update.

---

## Governance: Controlling Which Servers Your Agents Can Use

Listing available MCP servers is only half the picture. In production, the question is: *which servers should this agent be allowed to reach?*

Molecule AI's answer is a two-layer governance model.

**Org-scoped API keys** (`mol_pk_*`) constrain which workspaces a credential can touch, and each workspace config specifies which MCP servers are active for agents running in that workspace. A workspace configured for browser automation cannot reach the EC2 Instance Connect server unless it is explicitly registered — there is no ambient access.

**Platform Instructions** let workspace admins set behavioral rules that apply before every agent turn — things like "do not execute destructive shell commands without explicit confirmation" or "only call external APIs on this allowlist." These rules are in the system prompt, not a post-hoc filter, which means the agent reasons under the constraint rather than being blocked after the fact.

Together, these two controls give platform and compliance teams a practical answer to MCP governance at org scale: capability access is scoped by workspace config, and behavior within those capabilities is shaped by Platform Instructions. For a deeper look at how Tool Trace lets you verify agents are calling only the MCP tools they should, see the [observability and governance post](https://docs.molecule.ai/blog/agent-observability-tool-trace-platform-instructions).

---

## Get Started

To enable MCP servers for your workspace, see the [MCP configuration docs](https://docs.molecule.ai/platform/mcp-servers). To register a custom server or submit a community integration, open a request via the [Molecule AI partner program](https://docs.molecule.ai/docs/guides/partner-onboarding).

→ [MCP server configuration reference](https://docs.molecule.ai/platform/mcp-servers)  
→ [Platform Instructions governance docs](https://docs.molecule.ai/platform/platform-instructions)  
→ [Partner program and custom server submission](https://docs.molecule.ai/docs/guides/partner-onboarding)

---

*This catalogue reflects MCP servers available as of 2026-04-23. New servers are added on a rolling basis. Check the [changelog](https://docs.molecule.ai/changelog) for additions.*
