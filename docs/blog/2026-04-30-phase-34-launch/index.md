---
title: "Three Platform Capabilities That Close the Gap Between AI Agents and Production Infrastructure"
date: 2026-04-30
slug: phase-34-launch
description: "Phase 34 ships Tool Trace, Platform Instructions, and Partner API Keys — three capabilities that make Molecule AI usable at the infrastructure layer, not just the agent layer."
tags: [observability, governance, security, platform-engineering, enterprise]
---

# Three Platform Capabilities That Close the Gap Between AI Agents and Production Infrastructure

Every AI agent platform starts with the same promise: an agent that runs tasks, communicates results, and gets work done. That's the easy part.

The hard part — the part that determines whether you can actually use agents in production — is everything underneath: *What did the agent actually do? Can you enforce rules across every agent in the org? Can a partner or CI pipeline provision a new tenant without a browser session?*

Phase 34 ships three capabilities that close those gaps.

---

## Tool Trace: Know What Your Agent Actually Did

LLM output tells you what an agent said it did. Activity logs should tell you what it actually did.

Tool Trace records every tool call an agent makes — the tool name, the input arguments (sanitized), and a 200-character output preview — and stores it in your org's `activity_logs` table. Admins query it via the `/workspaces/:id/activity` endpoint.

```
GET /workspaces/:id/activity?limit=5

{
  "id": "log-abc123",
  "activity_type": "a2a_call",
  "created_at": "2026-04-30T12:01:00Z",
  "tool_trace": [
    {
      "tool": "mcp__files__read",
      "input": {"path": "config.yaml"},
      "output_preview": "api_version: v2, region: us-east-1, ..."
    },
    {
      "tool": "mcp__httpx__get",
      "input": {"url": "https://api.example.com/status"},
      "output_preview": "{\"status\": \"ok\", \"latency_ms\": 42}"
    }
  ]
}
```

**Why it matters:** Before Tool Trace, verifying that an agent "checked the config file before writing" meant replaying the entire conversation and hoping the agent mentioned it in its output. With Tool Trace, you query the activity log and see the exact sequence of tool calls.

For compliance and audit scenarios, Tool Trace serves as the ground-truth record: not what the agent said it did, but what it actually called.

**Who it's for:** Platform engineers debugging production issues. DevOps teams verifying agent behavior in CI. Compliance teams that need audit trails. AI/ML teams running agent evaluations and regression tests.

→ [Tool Trace demo →](/docs/devrel/demos/tool-trace-platform-instructions)
→ [Activity Logs API →](/docs/api-reference)

---

## Platform Instructions: Governance Enforced at the Infrastructure Layer

System prompt engineering works until it doesn't — until a developer forgets to include the rule, or an agent overrides it, or a new agent gets spun up without the right context.

Platform Instructions solve this by storing governance rules in the platform database, not in your agent's prompt file. An admin defines a rule once. Every agent in the org inherits it. Agents can't override it.

Create a global instruction:
```bash
curl -X POST "$PLATFORM_URL/instructions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "global",
    "title": "No shell commands in user-facing agents",
    "content": "Agents must NOT execute shell commands for users. Use file read/write tools or MCP tools only. Shell commands are only permitted in internal provisioning scripts.",
    "priority": 10
  }'
```

Create a workspace-scoped instruction (for a specific team or project):
```bash
curl -X POST "$PLATFORM_URL/instructions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"scope\": \"workspace\",
    \"scope_target\": \"$WORKSPACE_ID\",
    \"title\": \"Use dark theme by default\",
    \"content\": \"When generating UI components, default to the dark theme unless the user explicitly requests light mode.\",
    \"priority\": 5
  }"
```

When a workspace boots, it calls `GET /workspaces/:id/instructions/resolve` and receives the merged instruction text. This string is prepended as the first section of the agent's system prompt — ahead of all other content, highest precedence.

The 8KB content cap is enforced by a database CHECK constraint, not a code check. You can't accidentally prepend a 50KB instruction that blows out your token budget.

**Why it matters:** "Define once, inherit everywhere." One admin sets the rule. Every agent in the org — current and future — runs with it. There's no per-developer discipline required, no version-controlled prompt files that drift out of sync.

**Who it's for:** VP Engineering / CTO who needs AI governance without engineering-wide process. Platform admins managing multi-tenant deployments. Compliance teams requiring auditable, centrally-defined rules. Security engineers enforcing "no shell commands for users" at the platform level.

→ [Platform Instructions demo →](/docs/devrel/demos/tool-trace-platform-instructions)
→ [Architecture →](/docs/architecture/platform-instructions)

---

## Partner API Keys: Programmatic Org Management Without a Browser

Modern infrastructure is API-first. CI pipelines, partner platforms, and automation tools provision resources via API keys — not browser sessions. For AI agent platforms, that means being able to spin up a test org, run it through its paces, and tear it down — without a human in the loop.

Partner API Keys (GA in Phase 34) enable exactly this. Mint a key from the Canvas Settings → Org API Keys panel, hand it to your partner integration or CI pipeline, and that integration can:

- `POST /cp/orgs` — create a new org
- `GET /cp/orgs/:slug` — poll for provisioning status
- `GET /cp/orgs` — list all orgs on the platform

Keys are SHA-256 hashed in the database — never stored in plaintext. They're scoped to the org that created them. Revocation is immediate (401 on next request). Rate limiting prevents abuse (per-key and per-IP).

For CI/CD: a test pipeline creates a throwaway org, runs integration tests, and deletes the org — fully automated, fully audited.

For partners: a marketplace reseller provisions a white-labeled org for each new customer, linked to their billing system.

For internal tooling: a secrets automation tool manages org provisioning without needing a human admin session.

```bash
# CI/CD test org lifecycle
ORG_KEY="mol_pk_..."

# Create test org
ORG_RESP=$(curl -s -X POST "$CP_URL/cp/orgs" \
  -H "Authorization: Bearer $ORG_KEY" \
  -H "Content-Type: application/json" \
  -d '{"slug": "ci-test-001", "name": "CI Test Org"}')
ORG_ID=$(echo "$ORG_RESP" | jq -r '.id')

# Poll until ready
sleep 5 && curl -s "$CP_URL/cp/orgs/$ORG_ID" \
  -H "Authorization: Bearer $ORG_KEY" | jq '.status'

# Run tests, then delete
curl -s -X DELETE "$CP_URL/cp/orgs/$ORG_ID" \
  -H "Authorization: Bearer $ORG_KEY"
```

→ [Partner API Keys User Guide →](/docs/guides/partner-api-keys)
→ [Org API Keys (for your own org) →](/docs/blog/org-api-keys)

---

## How They Work Together

These three capabilities aren't isolated features. They're layers of a production-ready platform:

- **Partner API Keys** let partners and automation tools provision orgs programmatically — the entry point
- **Platform Instructions** define the governance rules every agent in that org inherits — the control plane
- **Tool Trace** records what every agent actually did — the observability layer

Together, they answer the questions production teams ask:
- *Can I provision a new org without a human?* → Partner API Keys
- *Can I enforce AI governance rules centrally?* → Platform Instructions
- *Can I verify what my agents actually did?* → Tool Trace

---

## What's Next

Role-scoped API keys (read-only, per-workspace) are on the roadmap. Platform Instructions will gain template support and a Canvas UI for non-admin teams. Tool Trace will be queryable from the Canvas Agent Comms panel.

Phase 34 is available today. → [moleculesai.app](https://www.moleculesai.app)

→ [Tool Trace + Platform Instructions Demo](/docs/devrel/demos/tool-trace-platform-instructions)
→ [Partner API Keys Guide](/docs/guides/partner-api-keys)
→ [Org API Keys](/docs/guides/org-api-keys)
→ [Phase 32 SaaS Launch →](/docs/blog/phase-32-saas-launch)

---

*Phase 34 — April 30, 2026. Three capabilities that make Molecule AI production-ready.*