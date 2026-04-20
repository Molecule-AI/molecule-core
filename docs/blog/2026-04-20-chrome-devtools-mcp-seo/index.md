---
title: "How to Add Browser Automation to AI Agents with MCP"
date: 2026-04-20
slug: browser-automation-ai-agents-mcp
description: "Connect Chrome DevTools Protocol to your AI agent via the Model Context Protocol. Full Python code examples — no Puppeteer, no Playwright, just CDP over MCP."
tags: [MCP, browser-automation, AI-agents, Chrome, CDP, tutorial]
---

# How to Add Browser Automation to AI Agents with MCP

AI agents are only as useful as the tools they can wield. Right now, the most-requested tool that most agent frameworks get wrong is browser automation. Developers want their AI agents to navigate websites, extract structured data, fill forms, and take screenshots — but the integration code to make that work is either missing, brittle, or locked behind a SaaS paywall.

The Model Context Protocol (MCP) changes this. MCP gives AI models a standardized interface for calling external tools — the same interface that Molecule AI workspaces use natively. And Chrome DevTools Protocol (CDP) gives you programmatic control of a real browser. Combine the two, and you have browser automation that an AI agent can invoke like any other tool: with typed inputs, structured outputs, and session persistence.

This post shows exactly how to wire Chrome DevTools into an AI agent via MCP — with working Python code and a complete end-to-end example.

## Why MCP for Browser Automation

Before MCP, connecting an AI agent to a browser meant one of two paths:

**Path 1: Custom wrapper scripts.** You write Python functions that call Puppeteer or Playwright, expose them via a prompt, and hope the model routes tool calls correctly. It works in demos. It breaks in production when the prompt drifts or the tool schema is ambiguous.

**Path 2: SaaS browser APIs.** Services like Browserbase or Steel provide managed browser infrastructure, but they add a dependency, a pricing tier, and a network hop between your agent and the browser. For teams already self-hosting or using Molecule AI, it's the wrong direction.

MCP solves both problems. It gives you:

- **Typed tool definitions** — your agent sees `browser_navigate`, `dom_query`, `page_screenshot` with JSON Schema inputs, not raw Python function names buried in a system prompt.
- **Streaming tool calls** — long-running browser operations (page loads, form submissions) stream progress back without blocking the agent's reasoning loop.
- **Session persistence** — CDP sessions maintain browser state (cookies, localStorage, scroll position) across tool calls, so your agent isn't starting from a blank page every turn.

Molecule AI workspaces ship MCP support out of the box. If you're already running Molecule AI, browser automation via MCP is a configuration change, not a rewrite.

## The Chrome DevTools Protocol + MCP Bridge

Chrome ships with a built-in remote debugging interface: the Chrome DevTools Protocol (CDP). It's the same protocol that Chrome DevTools, Puppeteer, and Playwright are built on. CDP exposes browser functionality over a WebSocket connection as JSON-RPC 2.0 commands across a set of domains:

| Domain | What it does |
|---|---|
| `Page` | Navigate, reload, capture screenshots |
| `DOM` | Query and traverse the DOM tree |
| `Runtime` | Execute JavaScript in the page context |
| `Network` | Inspect and intercept network requests |
| `Input` | Dispatch mouse and keyboard events |

An MCP server that bridges to CDP maps these domains onto MCP tool definitions. The result: your AI agent calls `browser_navigate` and the MCP server translates it to a `Page.navigate` CDP command over WebSocket.

The tool schema looks like this:

```json
{
  "name": "browser_navigate",
  "description": "Navigate to a URL in the headless Chrome session",
  "inputSchema": {
    "type": "object",
    "properties": {
      "url": { "type": "string", "description": "The URL to navigate to" }
    },
    "required": ["url"]
  }
}
```

```json
{
  "name": "dom_query",
  "description": "Query the DOM using a CSS selector",
  "inputSchema": {
    "type": "object",
    "properties": {
      "selector": { "type": "string", "description": "CSS selector" }
    }
  }
}
```

```json
{
  "name": "page_screenshot",
  "description": "Capture a screenshot of the current page",
  "inputSchema": {
    "type": "object",
    "properties": {
      "fullPage": { "type": "boolean", "description": "Capture the full scrollable page", "default": false }
    }
  }
}
```

The MCP server handles the WebSocket lifecycle, CDP command dispatch, and response parsing. Your agent code stays clean.

## Full Code Example: AI Agent That Researches Competitors

Here's a complete example using Molecule AI's Python SDK. The agent's task: go to a competitor's pricing page, extract the plan names and prices, and save a screenshot.

```python
from molecule_ai import Agent, MCPToolset
from browser_mcp import ChromeDevToolsMCP  # your MCP server

# Start the CDP session — connects to Chrome's remote debugging port
browser = ChromeDevToolsMCP(debugging_port=9222)

# Attach browser tools as MCP tools on the agent
agent = Agent(
    system_prompt="You are a competitive research assistant. "
                  "Use the browser tools to gather data.",
    mcp_tools=browser.tools(),   # fetches tools via MCP manifest
)

# Run the task
result = agent.run(
    "Go to https://example-competitor.com/pricing, extract all plan "
    "names and monthly prices, then save a screenshot of the page."
)

print(result.final_output)
```

Behind the scenes, the tool call cycle looks like this:

```
Agent → MCP invoke: browser_navigate { url: "https://example-competitor.com/pricing" }
MCP Server → CDP command: Page.navigate { url: "https://example-competitor.com/pricing" }
CDP → Page.loadEventFired event (streamed back)
Agent → MCP invoke: dom_query { selector: ".pricing-plan, [data-plan]" }
Agent → MCP invoke: page_screenshot { fullPage: false }
Agent → MCP invoke: browser_navigate { url: "about:blank" }  # cleanup
```

Each step is a structured tool call with typed inputs. The agent's prompt never mentions `websocket`, `JSON-RPC`, or `port 9222`. The MCP abstraction hides the infrastructure.

### Setting Up Chrome for Remote Debugging

To use CDP, start Chrome with the remote debugging port open:

```bash
# macOS
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome \
  --remote-debugging-port=9222 \
  --user-data-dir=/tmp/chrome-debug

# Linux
google-chrome --remote-debugging-port=9222 --user-data-dir=/tmp/chrome-debug

# Windows
chrome.exe --remote-debugging-port=9222 --user-data-dir="C:\tmp\chrome-debug"
```

Or launch a headless instance:

```bash
google-chrome \
  --headless \
  --remote-debugging-port=9222 \
  --user-data-dir=/tmp/chrome-headless
```

Make sure no other Chrome instance is already using port 9222 on your machine.

### The MCP Server: Minimal Implementation

If you want to roll your own MCP-to-CDP bridge (or understand what `browser_mcp` is doing above), here's the core of it:

```python
import json
import asyncio
import websockets

class ChromeDevToolsMCP:
    def __init__(self, debugging_port: int = 9222):
        self.ws_url = f"ws://localhost:{debugging_port}/devtools/browser"
        self._session_id: str | None = None
        self._ws: websockets.WebSocketClientProtocol | None = None

    async def __aenter__(self):
        self._ws = await websockets.connect(self.ws_url)
        # Create a new browser session
        resp = await self._send("Target.createBrowserContext")
        self._session_id = resp["browserContextId"]
        return self

    async def __aexit__(self, *args):
        if self._ws:
            await self._ws.close()

    async def _send(self, method: str, params: dict = None) -> dict:
        """Send a CDP command and wait for the response."""
        await self._ws.send(json.dumps({
            "id": 1,
            "method": method,
            "params": params or {},
        }))
        raw = await self._ws.recv()
        return json.loads(raw)

    def tools(self) -> list[dict]:
        """Return MCP tool definitions for this server."""
        return [
            {
                "name": "browser_navigate",
                "description": "Navigate to a URL",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "url": {"type": "string", "format": "uri"}
                    },
                    "required": ["url"]
                },
                "handler": self._navigate,
            },
            {
                "name": "page_screenshot",
                "description": "Capture a screenshot",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "fullPage": {"type": "boolean", "default": False}
                    }
                },
                "handler": self._screenshot,
            },
        ]

    async def _navigate(self, url: str) -> str:
        resp = await self._send("Page.navigate", {"url": url})
        return f"Navigated. FrameId: {resp.get('frameId')}"

    async def _screenshot(self, fullPage: bool = False) -> str:
        # Enable screenshot domain first
        await self._send("Page.enable")
        resp = await self._send("Page.captureScreenshot", {
            "format": "png",
            "fullPage": fullPage,
        })
        return f"screenshot:{resp['data']}"  # base64-encoded PNG
```

This is deliberately minimal — it shows the shape of the bridge without error handling, tab management, or the full CDP command surface. Production MCP servers (including Molecule AI's built-in browser tools) handle all of that.

## Real-World Use Cases

Browser automation via MCP isn't just a demo trick. Here are the production use cases teams are already running:

**Competitive intelligence pipelines.** An agent that visits a competitor's site weekly, extracts pricing and feature data, and writes a diff summary to a Notion page. No Puppeteer scripts to maintain — the agent updates the extraction logic itself when the competitor redesigns.

**AI-assisted data entry.** An agent that receives a spreadsheet row, navigates to a web form, fills it in, and submits. Particularly useful for legacy systems that only have a web UI and no API.

**Automated UI regression testing.** Instead of writing Playwright test scripts that break on every CSS change, describe the expected state in natural language. The agent uses `dom_query` and `page_screenshot` to verify the UI matches your specification.

**Real-time price and availability monitoring.** An agent that polls a retail or ticketing site, captures a screenshot on price change, and sends a Slack alert. Runs on a schedule or triggers from a webhook.

All four of these work with the same MCP toolset — the agent's reasoning layer is identical; only the task description changes.

## Getting Started with Molecule AI

Molecule AI workspaces have MCP support built in. The browser automation tools described in this post are available as a first-class MCP toolset — no custom server to deploy, no CDP WebSocket management required.

→ [MCP Server Setup Guide](/docs/guides/mcp-server-setup)
→ [Quickstart: Deploy your first AI agent](/docs/quickstart)

**Try it free** — Molecule AI is open source and self-hostable. Get a workspace running in under 5 minutes.

→ [Get started on GitHub →](https://github.com/Molecule-AI/molecule-core)

---

*Have a browser automation use case you want to see covered? Open a discussion on [GitHub Discussions](https://github.com/Molecule-AI/molecule-core/discussions) — or file an issue with the `enhancement` label.*
