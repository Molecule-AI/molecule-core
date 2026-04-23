# Phase 34 Community Announcement

**Channel:** Discord `#announcements` + GitHub Discussions  
**Date:** 2026-04-23  
**Author:** Marketing Lead (drafted on behalf of Community Manager)

---

🚀 **Phase 34 shipped — and it's a big one.**

This phase is all about giving platform builders the visibility, control, and provisioning primitives they've been asking for. Four features landed today:

---

## What's new

### 🔍 Tool Trace — see exactly what your agents did
Every A2A response now includes a `tool_trace` in `Message.metadata`. It's a list of every tool your agent called, with the input it sent and a preview of the output it got back:

```json
"tool_trace": [
  { "tool_name": "web_search", "input": {"query": "molecule ai agent platform"}, "output_preview": "Molecule AI is a multi-agent..." },
  { "tool_name": "write_file",  "input": {"path": "summary.md"}, "output_preview": "File written (412 bytes)" }
]
```

Parallel tool calls are handled — start/end events are paired via `run_id`, so concurrent calls don't get mixed up. Capped at 200 entries to prevent runaway loops from bloating your logs.

The full trace is stored in `activity_logs.tool_trace` — queryable, auditable, and there when you need to debug why an agent did what it did.

**Why it matters:** No more guessing. When something goes wrong in a multi-agent workflow, you now have the full tool call history to diagnose it.

---

### ⚙️ Platform Instructions — system prompt for your whole org
Org admins can now configure system-level instructions that apply across every agent in the org:

```http
PUT /cp/platform-instructions
{ "instructions": "Always respond in English. Tag every response with the agent workspace ID." }
```

Set it once, and every agent in your org inherits it — without touching individual workspace configs. Great for compliance requirements, house-style rules, or shared context that all your agents need.

---

### 🔑 Partner API Keys (`mol_pk_*`) — GA April 30
The partner provisioning API is entering GA on **April 30**. If you're building a platform on top of Molecule AI — a marketplace, a CI/CD integration, or a multi-tenant product — you can now programmatically create and manage Molecule AI orgs via API:

```http
POST /cp/admin/partner-keys
DELETE /cp/admin/partner-keys/:id
```

Ephemeral test orgs per PR. Org-scoped keys that can't escape their boundary. Automated teardown that stops billing on `DELETE`. No browser session required.

We believe this makes Molecule AI the first agent platform with a first-class partner provisioning API. If you're a platform builder and want early access ahead of April 30, drop a message in `#partner-program`.

---

### 🌐 SaaS Fed v2 — improved multi-org federation
Federation for multi-org deployments got a round of improvements in this phase. If you're running federated Molecule AI across multiple orgs, check the changelog for the specifics — the headline is better reliability and cleaner org boundary enforcement.

---

## Quick start

- **Tool Trace**: no config needed — it's in every A2A response today. Check `message.metadata.tool_trace`.
- **Platform Instructions**: `PUT /cp/platform-instructions` with your org admin token.
- **Partner API Keys**: docs at `/docs/api/partner-keys` — GA April 30.

---

## Try it & tell us what you think

All four features are live now (Partner API Keys GA on April 30). Spin them up and let us know:

- 🐛 Bugs or unexpected behavior → `#bug-reports`
- 💡 Feature requests → `#feedback`  
- 🤝 Partner program interest → `#partner-program`
- ❓ Questions → reply here or open a GitHub Discussion

We read everything. — The Molecule AI team
