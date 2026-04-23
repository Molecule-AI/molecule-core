# Phase 34 — Community FAQ
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Use:** Pinned in Discord `#faq` on launch day. Linked from announcement CTA.
**Date:** 2026-04-23

---

## Tool Trace

### What is Tool Trace?

Tool Trace is a structured execution record that gets added to every A2A response your agent produces. For each tool your agent calls, you get: the tool name, the input parameters, and a short output preview — in the order they happened, with `run_id` pairing for parallel calls.

Think of it as a print statement for your entire agent pipeline — you can finally see exactly what your agent did, not just what it returned.

---

### Where is the `tool_trace` field?

In `Message.metadata.tool_trace`. It's a JSON array in the response envelope — no SDK, no sidecar, no extra infrastructure. If you're consuming A2A responses in any language, just read `response.metadata.tool_trace` after a task completes.

```python
response = agent.send(task="deploy to staging")
for entry in response.metadata.tool_trace:
    print(f"{entry['tool']}: {entry['input']} → {entry['output_preview']}")
```

---

### Is Tool Trace on by default?

Yes. Tool Trace is active by default for all Molecule AI workspaces — no configuration required, no feature flag to enable. Activity logging must be enabled on your workspace for traces to be persisted to `activity_logs` (searchable history), but the response-level metadata is always present regardless.

---

### Does it cost extra?

No. Tool Trace is a platform-native feature included with every Molecule AI plan. There is no additional cost and no tier restriction — it's in the A2A protocol, not a paid add-on.

---

### What is the 200-entry cap?

The cap prevents runaway loops from generating unbounded trace data. A task that calls more than 200 tools will have entries dropped from the end of the trace — you get the first 200, which is sufficient for debugging virtually any production task.

If you're hitting the cap regularly, that's usually a signal your agent is doing too much in a single task — consider breaking it into smaller, more focused subtasks.

---

### Does Tool Trace work with custom MCP tools?

Yes. Tool Trace records every tool invocation via the agent runtime, regardless of whether the tool is a built-in Molecule tool or a custom MCP tool. The `tool` field shows the tool's registered name, `input` captures the parameters, and `output_preview` is the first ~200 chars of the response. Custom MCP tools appear identically to built-in tools in the trace.

---

## Platform Instructions

### What is Platform Instructions?

Platform Instructions lets org admins set system-level rules that apply to every agent in the organization — at startup, before the agent reads its own config. Think of it as a system prompt for your whole org.

Example rules:
- "Never commit directly to main — always open a PR first."
- "Tag all security-sensitive operations with an audit tag."
- "Confirm before running destructive commands in production."

Rules are injected by the platform, not the workspace. A workspace user can't override or remove them by editing their `config.yaml`.

---

### How do I set a Platform Instruction?

Org admins use the control plane API:

```
PUT /cp/platform-instructions
```

Two scopes:
- **Global** — applies to every workspace in the org
- **Workspace** — applies to a specific workspace only (additive to global rules)

For examples and full API documentation, see `docs.moleculesai.app/blog/platform-instructions-governance`.

---

### Can workspace-level instructions override global ones?

Workspace-level rules are additive to global ones, not overriding. A global rule and a workspace rule can both apply simultaneously. A workspace rule can explicitly opt out a global rule by prefix (e.g., `!global-rule-name`), but by default, both apply.

---

### Is Platform Instructions on all plans?

Yes. Platform Instructions is a platform-native feature included with every Molecule AI plan. No additional subscription required.

---

## Partner API Keys (`mol_pk_*`)

### What is it?

Partner API Keys (`mol_pk_*`) let marketplaces, CI/CD pipelines, and platform builders programmatically create and manage Molecule AI orgs via API — no browser session, no manual handoff. It's a scoped, revocable token that grants partner-level access to org management endpoints.

---

### When is it available?

GA: **April 30, 2026**. The feature ships on April 30. Until then, the API is not live.

---

### What can I build with it?

Common patterns:
- **Marketplace integrations** — let customers provision Molecule AI orgs from your platform's admin dashboard
- **CI/CD test orgs** — spin up an ephemeral org per pipeline run, run your test suite, tear it down. Billing stops when you DELETE.
- **Internal tooling** — provision orgs for internal teams or products without browser auth

Key constraints: keys are scoped to the orgs they're authorized for, rate-limited (60 req/min default, configurable), and fully revocable. Revocation is immediate — no grace period.

---

### How do I get access?

If you want to test Partner API Keys before GA, reach out via GitHub Discussions (`github.com/Molecule-AI/molecule-core/discussions`) or DM the Community Manager. Early access is available for partners with a concrete integration use case.

Full docs on GA day: `docs.moleculesai.app/blog/partner-api-keys`

---

## SaaS Fed v2

### What changed in SaaS Fed v2?

SaaS Federation v2 is the multi-tenant control plane architecture that underlies all SaaS deployments. The v2 update brings: improved cross-tenant isolation, cleaner org lifecycle management, and tighter alignment with the Partner API Keys infrastructure.

If you're running multiple orgs — for partners, internal teams, or separate products — the federation improvements make multi-org setups more robust and easier to operate at scale. External workspaces (agents running on laptops or other networks) also benefit from improved discovery and heartbeat reliability.

For self-hosted users: nothing changes. Federation v2 is SaaS-only.

---

## General

### Why are you shipping all four features at once?

Because they're built to work together. Partner API Keys provisions the orgs. Platform Instructions governs what happens inside them. Tool Trace shows you what your agents actually did. SaaS Fed v2 keeps it all isolated and reliable. Shipping them together means you get a coherent system from day one — not four unrelated features with interdependency debt.

---

### Where do I report issues?

For bugs and unexpected behavior:
- **Tool Trace / Platform Instructions** → `#bug-reports` in Discord, or open a GitHub issue and tag `@devrel`
- **Partner API Keys** → `#bug-reports` or GitHub issue

For how-to questions and setup help:
- **All features** → `#general` in Discord, or GitHub Discussions

---

### Where is the Discord for partner questions?

There's no separate partner Discord — partner questions are handled in the main Molecule AI Discord, same as all community questions. Use `#general` for broad questions, `#announcements` for launch updates, and `#feedback` for product feedback.

For direct partner program questions (early access, integration support, enterprise deals), DM the Community Manager or post in `#announcements` and tag `@community`.

---

### Do I need to update my SDK?

No. Tool Trace and Platform Instructions are API-level changes — they don't require SDK updates. Tool Trace is accessible via the existing A2A protocol (`Message.metadata.tool_trace`). Platform Instructions are managed via the control plane REST API (`PUT /cp/platform-instructions`). If you're using a Molecule SDK, you may already have helpers for these — check `docs.moleculesai.app/changelog` on April 30 for details.

---

## Quick reference

| Feature | Docs | Blog |
|---------|------|------|
| Tool Trace | `docs.moleculesai.app/blog/ai-agent-observability-without-overhead` | |
| Platform Instructions | `docs.moleculesai.app/blog/platform-instructions-governance` | |
| Partner API Keys | `docs.moleculesai.app/blog/partner-api-keys` | GA April 30 |
| SaaS Fed v2 | `docs.moleculesai.app/guides/external-workspace-quickstart` | |

Questions not answered here? GitHub Discussions: `github.com/Molecule-AI/molecule-core/discussions`