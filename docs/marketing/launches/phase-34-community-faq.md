# Phase 34 — Community FAQ
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Use:** Pinned in Discord `#faq` on launch day. Also linked from announcement CTA.
**Date:** 2026-04-23

---

## Tool Trace

### How do I access tool_trace in my code?

Every A2A response carries `Message.metadata.tool_trace` — it's a JSON array. Here's the simplest way to read it:

```python
response = agent.send(task="deploy to staging")
for entry in response.metadata.tool_trace:
    print(f"{entry['tool']}: {entry['input']} → {entry['output_preview']}")
```

Each entry: `{tool, input, output_preview, run_id, started_at, completed_at}`.

If you're consuming A2A responses in a different language, the structure is the same — it's plain JSON in the response envelope. No SDK required.

---

### Is Tool Trace available on all plans?

Tool Trace is enabled by default for all Molecule AI workspaces. No tier restriction — it's a platform-native feature, not a paid add-on. Activity logging must be enabled on your workspace for traces to be stored in `activity_logs`; the response-level metadata is always present regardless.

---

### What's the 200-entry cap for?

The cap prevents runaway loops from generating unbounded trace data. A task that calls more than 200 tools will have entries dropped from the end of the trace — you get the first 200, which is almost always sufficient for debugging a production task.

If you hit the cap regularly, that's a signal your agent is doing too much in a single task — consider splitting into smaller, more focused subtasks.

---

### Does Tool Trace work with custom MCP tools?

Yes. Tool Trace records every tool invocation via the agent's runtime, regardless of whether the tool is a built-in Molecule tool or a custom MCP tool. The `tool` field shows the tool's registered name; `input` captures the parameters passed; `output_preview` is the first ~200 chars of the response.

Custom MCP tools show up identically to built-in tools in the trace.

---

## Platform Instructions

### How is this different from per-workspace system prompts?

`config.yaml` sets a workspace's system prompt — that's the workspace owner's config. Platform Instructions are org-level rules injected by your platform admin, before the agent reads its own config.

The key difference: a workspace user can't remove a Platform Instruction by editing their `config.yaml`. The org-level rule prepends to the agent's effective system prompt at startup, regardless of what the workspace has configured. It's governance — not configuration.

---

### Can workspace-level instructions override global ones?

Yes. Platform Instructions support two scopes:

- **Global** (`PUT /cp/platform-instructions`, scope: `global`) — applies to every workspace in the org
- **Workspace** (`PUT /cp/platform-instructions?workspace_id=xxx`) — applies to a specific workspace only

Workspace-level instructions are additive to global ones. If a global rule says "tag all outputs" and a workspace rule says "also confirm before prod writes," both apply. A workspace rule does not remove or override a global rule unless the workspace rule explicitly opts out a global rule by prefix (e.g. `!global-rule-name`).

---

### Is there an audit log for instruction changes?

Yes. Instruction CRUD events (create/update/delete) are logged in the platform audit trail with the admin identity who made the change and a timestamp. Audit logs are accessible via `GET /cp/platform-instructions/history` (org admin only). This is the same audit infrastructure used for org API key management.

---

## Partner API Keys

### What's the difference between `mol_pk_*` and `mol_ws_*` keys?

`mol_pk_*` — Partner API keys. Scoped to partner platform operations: org creation, management, and deletion. Cannot read workspace data or agent memory. Rate-limited (60 req/min default), revocable immediately.

`mol_ws_*` — Workspace-level keys (existing). Scoped to a single workspace. Used for workspace-level API access (agent API, activity logs, secrets). Lower privilege surface.

Partner keys and workspace keys are independent token families with no overlap. A `mol_pk_*` key cannot act as a `mol_ws_*` key and vice versa.

---

### How do ephemeral test orgs work for CI/CD?

The typical pattern:

```bash
# 1. Create a test org
curl -X POST https://api.moleculesai.app/cp/orgs \
  -H "Authorization: Bearer mol_pk_live_YOUR_KEY" \
  -d '{"slug": "ci-test-$(date +%s)", "plan": "starter"}'
# Returns: {status: "provisioning", id: "..."}

# 2. Poll until active
# GET /cp/orgs/{slug} until status == "active"

# 3. Run your test suite against the workspace

# 4. Delete the org — billing stops immediately
curl -X DELETE https://api.moleculesai.app/cp/orgs/{slug} \
  -H "Authorization: Bearer mol_pk_live_YOUR_KEY"
```

A key scoped to `orgs:create` + `orgs:delete` can run this pattern. The slug should be unique per run (use timestamp or pipeline run ID) to avoid collisions. Org creation is async — poll for `status: "active"` before running tests.

---

### What happens to billing when I DELETE a partner key?

Deleting a key (`DELETE /cp/admin/partner-keys/:id`) is immediate and irreversible. Any orgs created by that key remain active until you explicitly delete them. Deleting the key does **not** delete the orgs it created — those continue to run and accrue billing.

If you want to stop billing for a partner-created org, you must delete the org separately (`DELETE /cp/orgs/:slug`). Keys and orgs are separate resources.

---

## General

### Where's the migration guide?

For Tool Trace and Platform Instructions: these are additive platform features, not migrations. Existing agents and workspaces work unchanged — Tool Trace starts appearing in responses automatically when activity logging is enabled. No config changes required to existing agents.

For Partner API Keys: new feature, no migration needed. Existing org management continues via the browser UI and WorkOS session auth.

General migration docs will be at `docs.moleculesai.app/guides/phase-34-migration` on GA day (April 30).

---

### Do I need to update my SDK?

Tool Trace and Platform Instructions are **API-level changes** — they don't require SDK updates. The features are accessible via the existing A2A protocol (Tool Trace in `Message.metadata`) and the control plane REST API (Platform Instructions via `PUT /cp/platform-instructions`).

If you're using a Molecule SDK, you may already have helpers for these — check the changelog at `docs.moleculesai.app/changelog` on April 30.

---

## Quick reference — what to read for each feature

| Feature | Docs | Blog |
|---------|------|------|
| Tool Trace (deep-dive) | `docs.moleculesai.app/blog/ai-agent-observability-without-overhead` | |
| Platform Instructions | `docs.moleculesai.app/blog/platform-instructions-governance` | |
| Partner API Keys | `docs.moleculesai.app/blog/partner-api-keys` | GA April 30 |
| SaaS Fed v2 | `docs.moleculesai.app/guides/external-workspace-quickstart` | |

Questions not answered here? Open a GitHub Discussion: github.com/Molecule-AI/molecule-core/discussions