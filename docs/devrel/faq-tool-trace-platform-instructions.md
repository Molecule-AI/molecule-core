# Tool Trace + Platform Instructions: Community FAQ

> **Routing note:** Bug reports → tag [Dev Lead](https://github.com/orgs/molecule-ai/teams/dev-leads) |
> Enterprise or pricing questions → tag [PM](https://github.com/orgs/molecule-ai/teams/product) |
> Feature requests → open a [GitHub Discussion](https://github.com/molecule-ai/molecule/discussions)
>
> **Docs:** [Tool Trace blog post](https://moleculesai.app/blog/tool-trace-observability) · [Platform Instructions blog post](https://moleculesai.app/blog/platform-instructions-governance) · [molecule-docs (internal)](https://github.com/molecule-ai/molecule-docs)

---

## Tool Trace

### Is Tool Trace always on, or do I need to enable it?

Tool Trace is **enabled by default** for all runs in GA organizations. No configuration is required to start capturing tool call data. If you are on a Legacy or Beta plan, contact your account manager to enable the feature flag `phase_34_tool_trace`.

### What is `run_id`, and how does it work with parallel tool calls?

Each agentic **run** receives a unique `run_id`. Within a run, individual tool calls are scoped to the same `run_id` but carry their own `tool_call_id` for correlation. When launching parallel tool calls (e.g., via async or concurrent execution), use the shared `run_id` to group all calls back to the originating run:

```
GET /v1/activity_logs?run_id=<run_id>
```

Tool calls from parallel branches within the same run will share the `run_id` but have distinct `tool_call_id` values. This allows full reconstruction of concurrent execution trees.

### How long is tool call data retained in activity logs?

Tool call records in activity logs are retained for **30 days** for Standard tier organizations and **90 days** for Enterprise tier organizations, after which they are automatically purged. There is no manual purge option. Export any data you need before the retention window closes.

### How do I filter activity logs by tool name?

Use the `tool_name` query parameter on the activity log API:

```
GET /v1/activity_logs?tool_name=<exact-tool-name>
```

Tool names are stored exactly as they appear in the tool manifest (e.g., `github_create_issue`, `web_search`). Partial-match filtering is not currently supported.

### What happens when I hit the 200-entry cap per run?

Each run stores up to **200 tool call entries**. Once the cap is reached, the oldest entries are evicted in FIFO order (oldest first). The `has_more` flag in the API response will be `true` when evictions have occurred. If you consistently hit the cap, consider splitting your task into shorter sub-runs, or contact support about raising the limit for your organization.

### How do I query tool trace data via the activity log API?

See the [Tool Trace API reference](https://docs.molecule.ai/api-reference/activity-logs) for the full schema. Quick reference:

```
GET /v1/activity_logs
  ?run_id=<run_id>          # required
  &tool_name=<tool-name>    # optional filter
  &limit=<1-200>            # optional, default 50
  &cursor=<opaque-token>    # optional pagination cursor
```

Response includes `tool_call_id`, `tool_name`, `arguments`, `output`, `started_at`, and `duration_ms`.

---

## Platform Instructions

### Can a workspace override a global instruction set?

**Yes.** Workspace-scoped instructions take precedence over global organization-level instructions. The resolution order is:

1. **Run-level** (highest priority — set at invocation time)
2. **Workspace-level**
3. **Organization-level** (default fallback)

This lets teams specialize general org policies for their own workflows without editing the global baseline.

### What happens when an instruction set exceeds the 8KB cap?

Platform Instructions enforces a **hard 8KB limit** per instruction document. Saves that would exceed this limit are rejected with a `422 Unprocessable Entity` response and the error message: `"instruction_payload_exceeds_8kb_limit"`. To work around the cap, split your instructions into multiple named documents and reference them by key in your agent configuration, or trim verbose preamble from existing documents.

### Who can create and delete Platform Instructions?

All authenticated users with **member or owner roles** in the organization can create, edit, and delete Platform Instructions at the org and workspace level. Guests and observers do **not** have write access. Audit log entries (see below) track all write operations with actor identity.

### Is a Canvas UI for managing Platform Instructions in the roadmap?

**Yes — targeted for Q3 2026.** The Canvas UI will provide a visual editor for instruction documents, version history, diff view, and team-level permission controls. Until then, all management is done via the REST API or CLI. Track progress at [issue #XXXX](#) (link to be added when filed).

### Can an agent modify its own Platform Instructions at runtime?

Agents can **read** their active Platform Instructions but cannot **write** to them at runtime. Runtime modification would require an explicit `UPDATE_INSTRUCTIONS` permission on the calling identity's API key. This is disabled by default. If you need runtime instruction drift detection, use the audit log endpoint (see below) to compare expected vs. actual instruction snapshots.

### How do I audit changes to Platform Instructions?

All write operations (create, update, delete) on instruction documents are recorded in the **organization audit log**. Query recent changes:

```
GET /v1/orgs/{org}/audit_log
  ?event_type=platform_instruction.*
  &after=<ISO-8601 timestamp>
```

Each audit entry includes the actor's user ID, the instruction document ID, the operation type, and the diff (for updates).

---

## Community Signal

> _Last updated: 2026-04-24_

No public Reddit r/MachineLearning, HN Show HN, or X/Twitter mentions of "Molecule AI Tool Trace" or "Molecule AI Platform Instructions" detected in open forums as of this write-up. This section will be updated as community posts surface around the April 26, 2026 GA launch.

If you see a community post that needs a response, ping the Community Manager workspace with the link.
