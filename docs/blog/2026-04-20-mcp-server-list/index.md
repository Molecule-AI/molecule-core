---
title: "MCP Servers: A Curated Guide to Model Context Protocol Integrations"
date: "2026-04-20"
slug: "mcp-server-list"
description: "The MCP ecosystem spans official servers, community servers, and platform-specific implementations. Here's what every developer evaluating MCP should know about the server landscape — and what to look for before choosing a platform."
tags: [MCP, server-list, AI-agents, tools, governance, integrations]
---

# MCP Servers: A Curated Guide to Model Context Protocol Integrations

The MCP (Model Context Protocol) ecosystem has grown rapidly since the specification went public. If you're evaluating AI agent platforms, you've likely run into the question: which MCP servers should I care about, and what separates a production-ready implementation from a prototype?

This guide answers both. It maps the official MCP server landscape, highlights notable community servers, and outlines the evaluation criteria that matter when you're choosing an MCP platform for your team.

**Companion post:** [Chrome DevTools MCP: AI Agent Browser Control with Governance](/blog/chrome-devtools-mcp)

---

## Official MCP Servers

The MCP project maintains a set of reference servers covering the most common integration patterns. These are the baseline — any MCP-compatible platform should be able to run these out of the box.

### Filesystem

The filesystem server exposes local file operations to the agent. Tools include reading, writing, and listing directories. It's the most universally needed MCP server — every agent that needs to work with code, configs, or documents needs filesystem access. Some platforms restrict which paths are accessible; production platforms should scope filesystem access to the workspace directory only.

### Git

The Git server exposes repository operations — log, diff, status, branch, checkout. Useful for agents that triage pull requests, review code changes, or maintain changelogs. The key capability to evaluate is whether the server runs in-process (requiring a git binary on the host) or runs over HTTP to a remote git host.

### Fetch

The Fetch server makes outbound HTTP requests. Every platform should implement SSRF (Server-Side Request Forgery) protection at the router level — rejecting `http://`, `git://`, and other non-`https://` schemes before the request is dispatched. Without SSRF protection, an agent can be redirected to internal services.

### Memory

The Memory server provides a persistent key-value store scoped to a session or workspace. It's useful for maintaining state across agent sessions without rebuilding context from scratch. Platforms that support hierarchical memory (LOCAL, TEAM, GLOBAL scopes) extend this further — allowing different visibility levels for different use cases.

### Brave Search

The Brave Search server gives agents the ability to search the web. Particularly useful for research agents that need to gather current information. Requires a Brave Search API key.

---

## Notable Community Servers

Beyond the official servers, the community has built MCP servers for a wide range of tools and platforms. These are the ones that appear most frequently in production deployments.

| Server | What it does |
|--------|-------------|
| **Chrome DevTools** | Full browser automation via Chrome's DevTools Protocol. Navigate, screenshot, evaluate JS, read cookies. [See the Chrome DevTools MCP integration →](/blog/chrome-devtools-mcp) |
| **Slack** | Post messages, search channels, list workspace contents. Useful for agents that report to team channels. |
| **GitHub** | Search repos, read and create issues, manage pull requests. The standard server for any agent doing code review or issue triage. |
| **Puppeteer** | Browser automation at a higher abstraction level than CDP. Good for common web interaction patterns. |
| **PostgreSQL** | Execute read queries against a database. Useful for data analysis and reporting agents. |
| **AWS KB Retrieval** | Query AWS knowledge bases for internal documentation retrieval. |
| **Google Workspace** | Read and write to Google Drive, Docs, Sheets. Useful for agents managing internal documentation. |
| **Sentry** | Read error events and issues from Sentry. Useful for agents doing production incident response. |

The full list grows regularly. The [MCP GitHub repository](https://github.com/modelcontextprotocol/servers) maintains an index of community servers.

---

## How to Evaluate an MCP Server Implementation

A server existing is not the same as a server being production-ready. Here are the evaluation criteria that separate an experiment from a deployment:

**1. Scope isolation.** Can you scope access per integration? A filesystem MCP server that can read `/etc/` is not production-ready. Workspace-scoped filesystem access is the minimum baseline.

**2. SSRF protection.** Does the HTTP fetch server reject internal IP ranges, `localhost`, and non-HTTPS schemes? This is the most common vulnerability in HTTP tool integrations.

**3. Audit attribution.** Can you trace every MCP tool call back to the integration that made it? Org-level API key attribution on every call is the production baseline.

**4. Credential isolation.** Are credentials scoped per integration, or shared? A shared credential model means revoking one integration revokes all of them.

**5. Revocation latency.** If you revoke an integration's access, how long until the agent can no longer use it? Instant revocation (no restart required) is the production target.

---

## The MCP Governance Layer: Why It Matters

MCP is a connectivity protocol. By itself, it answers "how does the agent talk to this tool?" It doesn't answer "who is accountable for what the agent did with that tool?"

That second question is the governance layer — and it's the difference between an MCP integration you can put in front of a security review and one you can't.

Molecule AI's MCP governance layer adds:

- **Org API key attribution on every MCP call** — which integration made which call, with what parameters, at what time
- **Token-scoped session isolation** — each integration gets its own session context, isolated from other integrations
- **Instant revocation** — revoke a key, the agent loses access immediately, no redeploy

This is what makes MCP integrations enterprise-ready. The protocol gets your agent connected. The governance layer gets your security team to sign off.

---

## Get Started

- [Chrome DevTools MCP: AI Agent Browser Control with Governance](/blog/chrome-devtools-mcp) — browser automation via MCP with full governance
- [Org API Keys: Audit Attribution Setup](/blog/org-scoped-api-keys) — set up org-level API keys for MCP attribution
- [MCP Server Setup Guide](/docs/guides/mcp-server-setup) — configure MCP servers in your Molecule AI workspace
- [Official MCP Server Repository](https://github.com/modelcontextprotocol/servers) — full index of community MCP servers

---

*Molecule AI is open source. MCP server support ships in `workspace-server/internal/handlers/mcp.go` on `main`.*

---

## The MCP Server List

Molecule AI's MCP server support covers four categories: platform services, external data sources, execution environments, and developer tooling.

### Platform Services

These MCP servers connect agents to Molecule AI's own platform capabilities:

- **`molecule-platform`** — access workspace state, secrets, schedules, channels, and delegation. Every agent uses this by default. Tools include `list_workspaces`, `get_workspace`, `create_secret`, `list_schedules`, `delegate_task`, `send_message_to_user`.
- **`molecule-memory`** — read and write the agent's persistent memory store (LOCAL, TEAM, GLOBAL scopes). Tools include `commit_memory`, `recall_memory`, `search_memory`. See [Hierarchical Memory Architecture](/docs/architecture/memory).
- **`molecule-files`** — read and write files in the workspace filesystem. Tools include `read_file`, `write_file`, `list_directory`, `search_files`. Operates on the workspace's bind-mounted directory.

### External Data Sources

These MCP servers give agents read access to external data and services:

- **`mcp-code`** — search and read source code across the workspace. Uses tree-sitter based code search for finding functions, classes, and patterns. Tools include `code_search`, `code_read`, `code_grep`.
- **`mcp-git`** — interact with git repositories. Tools include `git_log`, `git_diff`, `git_status`, `git_branch`, `git_checkout`. Useful for agents that review PRs, track changes, or manage branches.
- **`mcp-http`** — make outbound HTTP requests. Tools include `http_request` with method, URL, headers, and body parameters. SSRF protection is enforced at the router level — `http://` and `git://` schemes are rejected before the request is made.

### Execution Environments

These MCP servers give agents the ability to execute code and commands:

- **`mcp-sandbox`** — run code in an isolated execution environment. Supports multiple backends: `subprocess` (default, asyncio subprocess with hard timeout), `docker` (throwaway container with resource limits), and `e2b` (cloud microVMs via E2B API). Tools include `run_code` with language, code, and timeout parameters.
- **`mcp-bash`** — run shell commands on the host. Subject to guardrails configured per workspace. Tools include `bash` with command and timeout parameters. Access is gated by the workspace's configured permission level.

### Developer Tooling

These MCP servers connect agents to common developer workflows:

- **`mcp-puppeteer`** — browser automation via Puppeteer. A higher-level alternative to Chrome DevTools CDP for common web interaction patterns. Tools include `puppeteer_navigate`, `puppeteer_screenshot`, `puppeteer_click`, `puppeteer_evaluate`.
- **`mcp-github`** — GitHub API integration. Tools include `github_search_repos`, `github_get_issue`, `github_create_issue`, `github_list_prs`, `github_merge_pr`. Useful for agents that triage issues, review PRs, or manage repos.
- **`mcp-slack`** — Slack API integration. Tools include `slack_post_message`, `slack_search_messages`, `slack_list_channels`. Useful for agents that report to team Slack channels.

---

## How the MCP Governance Layer Applies Across All Servers

The governance story isn't specific to Chrome DevTools MCP — it applies uniformly across every MCP server:

### Every call is audit-attributed

Every MCP tool call generates an audit log entry with:
- The org API key prefix that made the call
- The tool name and parameters
- The result or error
- Timestamp and workspace ID

This means whether your agent calls `mcp-github` to merge a PR or `mcp-bash` to run a test suite, the audit trail exists.

### Sessions are scoped per org API key

Each org API key has its own isolated session context for stateful MCP servers (browser sessions, bash environments). Agent A's session is never Agent B's. No cross-contamination of credentials or state.

### Instant revocation

Revoke an org API key and every MCP server that key had access to immediately rejects future calls. No redeploy, no agent restart. The audit trail shows exactly what was done before revocation.

---

## Adding a New MCP Server

Molecule AI's MCP server architecture is pluggable. To add a new MCP server:

1. **Define the server manifest** in your workspace `config.yaml`:
   ```yaml
   mcpServers:
     - name: my-custom-server
       url: "stdio:/path/to/server"
       transport: stdio  # or cdp, http
   ```

2. **Configure the server binary** — for stdio transport, the server binary must be available in the workspace's PATH or specified as an absolute path.

3. **Tools become available immediately** — once the server is configured, all tools it exposes appear in the agent's tool list and are subject to the same governance layer.

---

## Choosing the Right MCP Servers for Your Agent

Not every agent needs every MCP server. Here's a quick guide:

| Agent Role | Recommended MCP Servers |
|---|---|
| PM / Orchestrator | `molecule-platform`, `molecule-memory`, `mcp-http` |
| Research Agent | `mcp-code`, `mcp-http`, `mcp-github`, `mcp-slack` |
| Browser Automation Agent | `chrome-devtools-mcp` (CDP), `mcp-puppeteer` |
| CI / Testing Agent | `mcp-bash`, `mcp-sandbox`, `mcp-git`, `mcp-github` |
| Security Auditor | `molecule-platform`, `molecule-memory` (read-only), `mcp-git` |

Configure MCP servers per workspace via `config.yaml`. Use the org template's `defaults.plugins` field to set a baseline for every new workspace in your org.

---

## Get Started

- [Chrome DevTools MCP: AI Agent Browser Control with Governance](/blog/chrome-devtools-mcp) — the browser automation companion post
- [MCP Server Setup Guide](/docs/guides/mcp-server-setup) — configure MCP tools in your workspace
- [Org API Keys: Audit Attribution Setup](/blog/org-scoped-api-keys) — set up org API keys with attribution
- [Plugin System Documentation](/docs/plugins/sources) — MCP server installation via the plugin API

---

*Molecule AI is open source. MCP server support ships in `workspace-server/internal/handlers/mcp.go` on `main`.*