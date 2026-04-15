---
name: ai-act-audit-log
description: "Emit immutable audit events for EU AI Act compliance. Use when a workspace performs any action that needs to be legally reconstructable: delegations, approvals, RBAC decisions, memory read/write. JSON Lines, append-only, SIEM-friendly."
---

# EU AI Act Audit Log

Opt-in plugin that activates `builtin_tools/audit.py` — an append-only
JSON Lines log satisfying the record-keeping and transparency obligations
of the EU AI Act (Articles 12, 13, 17) for high-risk AI systems.

## When to install

Install on any workspace that:
- Must satisfy EU AI Act conformity assessment
- Needs a tamper-evident trail of agent decisions for a legal discovery
- Pairs with `molecule-compliance` to record OWASP OA-01 detections and
  OA-03 terminations

Skip on disposable dev workspaces — the log fills disk over time and
isn't useful for throwaway agents.

## Event schema

Every line is one JSON object:

```json
{
  "timestamp":    "2026-04-15T21:30:00.123Z",
  "event_type":   "delegation",
  "workspace_id": "ws-acme-pm-a1b2c3d4",
  "actor":        "ws-acme-pm-a1b2c3d4",
  "action":       "delegate",
  "resource":     "ws-acme-dev-lead-e5f6g7h8",
  "outcome":      "allowed",
  "trace_id":     "5e8b2f3c-9a1d-4e7b-8c6f-1234567890ab"
}
```

Required fields:

| Field | Meaning |
|---|---|
| `timestamp` | ISO-8601 UTC with offset — sort key + freshness indicator |
| `event_type` | `delegation` / `approval` / `memory` / `rbac` |
| `workspace_id` | Who generated the event |
| `actor` | Who triggered the action (defaults to workspace_id for automated events; human identity for approval decisions) |
| `action` | Verb: `delegate`, `approve`, `memory.read`, `memory.write`, `rbac.deny` |
| `resource` | Target of the action: another workspace id, memory scope, approval action string |
| `outcome` | `allowed` / `denied` / `success` / `failure` / `timeout` / `requested` / `granted` |
| `trace_id` | UUID v4 correlating related events across workspaces |

## Usage

Call `audit.log_event` from any tool or handler:

```python
from builtin_tools.audit import log_event

log_event(
    event_type="delegation",
    workspace_id=self.workspace_id,
    actor=self.workspace_id,
    action="delegate",
    resource=target_workspace_id,
    outcome="allowed",
    trace_id=ctx.trace_id,
)
```

The function is synchronous and fire-and-forget — it opens the log file
in append mode, writes one line, closes. No buffering, no retry. If the
disk is full the call raises `IOError`; the caller decides whether to
surface that (usually yes — an audit gap is a compliance event itself).

## Configuration

Add to `config.yaml`:

```yaml
audit:
  enabled: true
  log_path: /var/log/molecule/audit.jsonl
  max_size_mb: 100      # informational only; rotation is EXTERNAL
  retention_days: 365   # informational only; the module never deletes
```

## Rotation (external)

This module is **write-only by design**. It does not rotate, compress,
or delete log lines. Use the host's `logrotate` (Linux) or equivalent:

```
/var/log/molecule/audit.jsonl {
    daily
    rotate 365
    compress
    copytruncate   # NOT truncate — copytruncate leaves the file open
    missingok
    notifempty
}
```

`copytruncate` is load-bearing — the Python side holds the file
descriptor open for append, so a rename-based rotation would orphan the
new file and writes would continue to the rotated-away path.

## SIEM ingestion

The JSON Lines format is directly consumable by:
- Splunk (ingest via Universal Forwarder)
- Elastic (Filebeat + JSON decoder)
- Datadog (Agent in JSON mode)
- Self-hosted Loki

One ingestion pipeline per workspace volume. No post-processing needed.

## Anti-patterns

- **Don't** write to the same log path from multiple workspaces on the
  same host — races corrupt the JSONL newlines. Use per-workspace paths.
- **Don't** truncate or edit the log. Tamper-evidence is the whole point.
- **Don't** log raw PII or secrets in the `resource` or `outcome` fields.
  Use IDs or hashes; the audit story and the GDPR story have to coexist.
- **Don't** skip this on OA-01/OA-03 detections — they're exactly the
  events an auditor wants to see.

## Related

- `builtin_tools/audit.py` — the implementation
- `molecule-compliance` — emits OWASP OA-01 / OA-03 events into this log
- `molecule-security-scan` — emits CVE-scan results into this log
- Issue #256 — the proposal that led to this plugin split
