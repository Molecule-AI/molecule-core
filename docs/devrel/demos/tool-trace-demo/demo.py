#!/usr/bin/env python3
"""
demo.py — Tool Trace Demo
=========================
PR #1686 (molecule-core) | Tool Trace adds per-tool-call observability to
every A2A response: tool name, input args (sanitized), and a 300-char
output preview — stored in activity_logs.tool_trace (JSONB + GIN index).

This demo shows:
  1. How to read tool_trace from an A2A response metadata object
  2. How to query activity_logs for past tool traces
  3. How run_id pairs parallel tool calls (LangGraph supports concurrent
     tool invocations; run_id ensures start/end events pair correctly)
  4. A clean "Agent Activity Report" printed from the trace data

Requirements:
  pip install requests a2a

Usage:
  # 1. Run the demo against a live platform (replace with your credentials)
  export PLATFORM_URL=https://your-deployment.moleculesai.app
  export WORKSPACE_TOKEN=your-workspace-token

  python demo.py

  # 2. Or import the classes into your own agent code:
  from demo import AgentActivityReport, ToolTraceSession

────────────────────────────────────────────────────────────────────────────
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from typing import Any

try:
    import requests
except ImportError:
    raise SystemExit("pip install requests  # HTTP client for A2A API calls")


# ─────────────────────────────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────────────────────────────

PLATFORM_URL = "https://your-deployment.moleculesai.app"  # Override via env
WORKSPACE_TOKEN = "your-workspace-token"                   # Override via env


# ─────────────────────────────────────────────────────────────────────────────
# Utilities
# ─────────────────────────────────────────────────────────────────────────────

def truncate(value: Any, max_chars: int) -> str:
    """Truncate a value to at most max_chars characters."""
    s = str(value) if not isinstance(value, str) else value
    return s[:max_chars]


# ─────────────────────────────────────────────────────────────────────────────
# Data models
# ─────────────────────────────────────────────────────────────────────────────

@dataclass
class ToolCall:
    """A single tool invocation captured in the tool trace."""
    tool: str
    input_summary: str        # Sanitized input — first 500 chars
    output_preview: str       # Output preview — first 300 chars
    run_id: str = ""         # Correlates parallel start/end events

    @classmethod
    def from_dict(cls, d: dict) -> "ToolCall":
        return cls(
            tool=d.get("tool", "?"),
            input_summary=truncate(d.get("input", ""), 500),
            output_preview=truncate(d.get("output_preview", ""), 300),
            run_id=d.get("run_id", ""),
        )


@dataclass
class AgentActivityReport:
    """
    Parses a tool_trace list into a structured activity report.
    Suitable for printing, storing, or forwarding to a SIEM / audit pipeline.
    """
    workspace_id: str
    trace: list[ToolCall] = field(default_factory=list)
    session_id: str = ""
    timestamp: str = ""

    @classmethod
    def from_response(cls, workspace_id: str, a2a_response_metadata: dict) -> "AgentActivityReport":
        """
        Build a report from an A2A response metadata object.

        In production, the A2A response returned by
        POST /workspaces/:id/a2a has a metadata field:

          {
            "tool_trace": [
              {"tool": "bash", "input": "...", "output_preview": "..."},
              ...
            ]
          }

        Args:
            workspace_id: workspace that generated the trace
            a2a_response_metadata: the metadata dict from the A2A response

        Returns:
            AgentActivityReport populated from the tool_trace list
        """
        raw = a2a_response_metadata.get("tool_trace", [])
        trace = [ToolCall.from_dict(e) for e in raw]
        return cls(
            workspace_id=workspace_id,
            trace=trace,
            session_id=a2a_response_metadata.get("session_id", ""),
            timestamp=a2a_response_metadata.get("timestamp", ""),
        )

    @classmethod
    def from_activity_log(cls, workspace_id: str, activity_log_entry: dict) -> "AgentActivityReport":
        """
        Build a report from an activity_logs JSONB entry.

        After an agent run, the platform stores the tool trace in
        activity_logs.tool_trace (JSONB). Query it via:

          GET /workspaces/:id/activity?limit=5

        Each log entry has:
          - id, activity_type, created_at
          - tool_trace: list[...]  (may be null for non-agent entries)
        """
        raw = activity_log_entry.get("tool_trace") or []
        trace = [ToolCall.from_dict(e) for e in raw]
        return cls(
            workspace_id=workspace_id,
            trace=trace,
            session_id=activity_log_entry.get("id", ""),
            timestamp=activity_log_entry.get("created_at", ""),
        )

    def print_report(self, title: str = "Agent Activity Report") -> None:
        """Print a human-readable activity report to stdout."""
        divider = "═" * 68
        print(f"\n{' '}{divider}")
        print(f"  {title}")
        print(f"{' '}{divider}")
        print(f"  Workspace : {self.workspace_id}")
        if self.timestamp:
            print(f"  Timestamp : {self.timestamp}")
        if self.session_id:
            print(f"  Session   : {self.session_id}")
        print()

        if not self.trace:
            print("  (no tool calls recorded in this trace)")
        else:
            # Group by run_id to show parallel tool calls
            by_run: dict[str, list[ToolCall]] = {}
            for tc in self.trace:
                rid = tc.run_id or "(sequential)"
                by_run.setdefault(rid, []).append(tc)

            for run_id, calls in by_run.items():
                if len(calls) > 1:
                    print(f"  ┌─ Parallel call group [{run_id[:8]}...]")
                    for tc in calls:
                        self._print_tool_call(tc, prefix="  │ ", parallel=True)
                    print(f"  └─")
                else:
                    self._print_tool_call(calls[0], prefix="  ", parallel=False)

        print(f"\n{' '}{divider}\n")

    def _print_tool_call(self, tc: ToolCall, prefix: str, parallel: bool) -> None:
        print(f"{prefix}▸ {tc.tool}")
        print(f"{prefix}  Input   : {tc.input_summary}")
        if tc.output_preview:
            print(f"{prefix}  Output  : {tc.output_preview[:200]}")

    def as_dict(self) -> dict:
        """Return a JSON-serializable representation."""
        return {
            "workspace_id": self.workspace_id,
            "session_id": self.session_id,
            "timestamp": self.timestamp,
            "tool_call_count": len(self.trace),
            "tools_called": [tc.tool for tc in self.trace],
            "trace": [
                {"tool": tc.tool, "input": tc.input_summary, "output_preview": tc.output_preview}
                for tc in self.trace
            ],
        }


# ─────────────────────────────────────────────────────────────────────────────
# API helpers
# ─────────────────────────────────────────────────────────────────────────────

class MoleculeAIClient:
    """
    Minimal client for the Molecule AI A2A + activity API.
    Demonstrates how to call the platform and extract tool_trace.
    """

    def __init__(self, platform_url: str, workspace_token: str):
        self.platform_url = platform_url.rstrip("/")
        self.headers = {
            "Authorization": f"Bearer {workspace_token}",
            "Content-Type": "application/json",
        }

    def send_task(self, task: str, workspace_id: str) -> dict:
        """
        Send a task to a workspace and return the A2A response.
        The response.metadata contains the tool_trace.

        API: POST /workspaces/:id/a2a
        Body: {"task": task}
        """
        url = f"{self.platform_url}/workspaces/{workspace_id}/a2a"
        resp = requests.post(url, headers=self.headers, json={"task": task}, timeout=60)
        resp.raise_for_status()
        return resp.json()

    def get_activity_logs(self, workspace_id: str, limit: int = 5) -> list[dict]:
        """
        Query the activity log for a workspace.
        Each entry has .tool_trace (JSONB list) when recorded by an agent runtime.

        API: GET /workspaces/:id/activity?limit=N
        """
        url = f"{self.platform_url}/workspaces/{workspace_id}/activity"
        resp = requests.get(url, headers=self.headers, params={"limit": limit})
        resp.raise_for_status()
        return resp.json()


# ─────────────────────────────────────────────────────────────────────────────
# Demo: simulate tool_trace data (no live platform needed)
# ─────────────────────────────────────────────────────────────────────────────

def simulate_a2a_response_metadata() -> dict:
    """
    Returns a simulated A2A response metadata object as it would look
    after PR #1686. In production, this comes from:

      response = client.send_task(task="...", workspace_id="...")
      metadata = response.get("metadata", {})

    The agent called three tools in sequence:
      1. web_search → found relevant docs
      2. summarizer → condensed to 3 bullets
      3. write_to_file → saved result
    """
    return {
        "session_id": "sess-abc123",
        "timestamp": "2026-04-23T12:01:00Z",
        "tool_trace": [
            {
                "tool": "web_search",
                "input": json.dumps({"query": "Molecule AI agent platform observability", "top_k": 5}),
                "output_preview": "[{'title': 'A2A Protocol Spec', 'url': 'https://a2a.chat', 'snippet': 'The A2A protocol is...'}]",
                "run_id": "run-001",
            },
            {
                "tool": "summarize_text",
                "input": json.dumps({"text": "[full search results...", "max_bullets": 3}),
                "output_preview": "• A2A enables direct workspace-to-workspace communication...",
                "run_id": "run-002",
            },
            {
                "tool": "write_to_file",
                "input": json.dumps({"path": "/tmp/agent-report.md", "content": "# Agent Activity Report\n..."}),
                "output_preview": "File written: /tmp/agent-report.md (847 bytes)",
                "run_id": "run-003",
            },
        ],
    }


def simulate_parallel_tool_calls() -> dict:
    """
    Simulates a LangGraph agent making two tool calls concurrently
    (e.g., a parallel "web search" + "read config file" call).

    run_id links the on_tool_start → on_tool_end events so the
    output_preview gets paired with the right tool_name entry.
    """
    return {
        "session_id": "sess-xyz789",
        "timestamp": "2026-04-23T12:05:00Z",
        "tool_trace": [
            # Both start events recorded (order may vary)
            {"tool": "web_search",     "input": '{"query": "Molecule AI tool trace docs"}',  "run_id": "run-parallel-a"},
            {"tool": "read_file",      "input": '{"path": "/workspace/config.yaml"}',       "run_id": "run-parallel-b"},
            # Both end events update the same entries by run_id
            {"tool": "web_search",     "input": '{"query": "..."}',  "output_preview": "[found 8 results in 142ms]", "run_id": "run-parallel-a"},
            {"tool": "read_file",      "input": '{"path": "..."}',   "output_preview": "database_url: postgresql://...\nmodel: claude-sonnet-4", "run_id": "run-parallel-b"},
        ],
    }


def simulate_activity_log_entry() -> dict:
    """Simulates what you get back from GET /workspaces/:id/activity"""
    return {
        "id": "log-abc123",
        "activity_type": "a2a_call",
        "created_at": "2026-04-23T12:01:00Z",
        "workspace_id": "ws-abc123",
        "tool_trace": [
            {"tool": "bash",              "input": '{"command": "git status"}',       "output_preview": "On branch main\\nNothing to commit, working tree clean", "run_id": ""},
            {"tool": "mcp__httpx__get",  "input": '{"url": "https://api.github.com"}', "output_preview": '{"rate_limit": 5000, "remaining": 4998}', "run_id": ""},
            {"tool": "mcp__files__write", "input": '{"path": "audit.json", ...}',      "output_preview": "Written: audit.json (12.4 KB)", "run_id": ""},
        ],
    }


# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

def main():
    print("""
    ╔══════════════════════════════════════════════════════════════════════╗
    ║          Tool Trace Demo — PR #1686 (molecule-core)                  ║
    ║                                                                      ║
    ║  Every A2A response from a Molecule AI agent now carries a          ║
    ║  tool_trace list in metadata: what tools were called, with what     ║
    ║  inputs, and what came back — stored in activity_logs.tool_trace.    ║
    ╚══════════════════════════════════════════════════════════════════════╝
    """)

    # ── Demo 1: Parse from A2A response metadata ─────────────────────────
    print("\n[DEMO 1] Reading tool_trace from an A2A response metadata object")
    print("─" * 68)
    metadata = simulate_a2a_response_metadata()
    report = AgentActivityReport.from_response(
        workspace_id="ws-demo-001",
        a2a_response_metadata=metadata,
    )
    report.print_report(title="Demo 1 — A2A Response Tool Trace")

    # ── Demo 2: Parse from activity log entry ────────────────────────────
    print("\n[DEMO 2] Reading tool_trace from activity_logs JSONB entry")
    print("─" * 68)
    log_entry = simulate_activity_log_entry()
    report2 = AgentActivityReport.from_activity_log(
        workspace_id="ws-demo-001",
        activity_log_entry=log_entry,
    )
    report2.print_report(title="Demo 2 — Activity Log Tool Trace")

    # ── Demo 3: Parallel tool calls ───────────────────────────────────────
    print("\n[DEMO 3] Parallel tool calls — run_id pairs start/end events")
    print("─" * 68)
    print("  When a LangGraph agent calls two tools concurrently, the")
    print("  platform records both start events, then both end events.")
    print("  run_id ensures output_preview gets paired with the right tool.\n")
    parallel_metadata = simulate_parallel_tool_calls()
    report3 = AgentActivityReport.from_response(
        workspace_id="ws-demo-001",
        a2a_response_metadata=parallel_metadata,
    )
    report3.print_report(title="Demo 3 — Parallel Tool Calls")

    # ── Demo 4: Live platform (requires credentials) ─────────────────────
    print("\n[DEMO 4] Live platform — query activity logs via API")
    print("─" * 68)
    if PLATFORM_URL == "https://your-deployment.moleculesai.app":
        print("  ⚠ SKIPPED — set PLATFORM_URL and WORKSPACE_TOKEN env vars")
        print("  Code to use with a live platform:\n")
        print("    from demo import MoleculeAIClient, AgentActivityReport")
        print()
        print("    client = MoleculeAIClient(")
        print("        platform_url=os.environ['PLATFORM_URL'],")
        print("        workspace_token=os.environ['WORKSPACE_TOKEN'],")
        print("    )")
        print()
        print("    # Option A: from A2A response metadata")
        print("    resp = client.send_task('Summarize the last 5 commits', workspace_id='ws-abc')")
        print("    report = AgentActivityReport.from_response('ws-abc', resp.get('metadata', {}))")
        print("    report.print_report()")
        print()
        print("    # Option B: from activity log (query past runs)")
        print("    logs = client.get_activity_logs('ws-abc', limit=3)")
        print("    for entry in logs:")
        print("        report = AgentActivityReport.from_activity_log('ws-abc', entry)")
        print("        report.print_report()")
    else:
        print("  Connecting to live platform...")
        try:
            client = MoleculeAIClient(PLATFORM_URL, WORKSPACE_TOKEN)
            logs = client.get_activity_logs("demo-workspace", limit=3)
            for entry in logs:
                report = AgentActivityReport.from_activity_log("demo-workspace", entry)
                report.print_report(title="Live — Activity Log Entry")
        except Exception as e:
            print(f"  ⚠ Error: {e}")

    # ── Output: JSON export (for SIEM / audit pipelines) ──────────────────
    print("\n[DEMO 5] JSON export (for audit pipelines / SIEM ingestion)")
    print("─" * 68)
    print(json.dumps(report.as_dict(), indent=2))


# ─────────────────────────────────────────────────────────────────────────────
# Key insight
# ─────────────────────────────────────────────────────────────────────────────
"""
Tool Trace key design decisions (from a2a_executor.py):

1. Event-based collection:
   on_tool_start  → records {"tool", "input[:500]"}
   on_tool_end    → updates same entry: {"output_preview[:300]"}
   Pairing via run_id prevents parallel calls from clobbering each other.

2. 200-entry cap (MAX_TOOL_TRACE):
   Prevents runaway agent loops from generating unbounded JSONB payloads.
   Enforced at the write site — the DB has no constraint on the list size.

3. metadata.tool_trace in A2A response:
   Attached when the agent returns its response to the platform.
   Consumers: Canvas Agent Comms panel, audit pipelines, compliance tooling.

4. Stored in activity_logs.tool_trace (JSONB + GIN index):
   Queryable via GET /workspaces/:id/activity
   GIN index on tool_trace enables efficient JSONB containment queries.
"""


if __name__ == "__main__":
    main()