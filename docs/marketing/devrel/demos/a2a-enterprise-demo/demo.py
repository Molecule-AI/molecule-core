#!/usr/bin/env python3
"""
demo.py — A2A Enterprise Deep-Dive Demo
=========================================
PR #1686 (molecule-core) | Issue: a2a-enterprise-deep-dive campaign

Demonstrates the A2A Protocol enterprise multi-agent orchestration:
  1. Register two sub-agents with the platform registry
  2. Direct peer-to-peer A2A call between sub-agents (no platform relay)
  3. PM agent delegates a task via platform proxy, receives result
  4. Parallel tool calls with run_id correlation
  5. SSE streaming response back to parent

Requirements: pip install requests sseclient-py

Usage:
  export PLATFORM_URL=https://your-deployment.moleculesai.app
  export WORKSPACE_TOKEN=your-workspace-token
  python demo.py

────────────────────────────────────────────────────────────────────────────
"""

from __future__ import annotations

import json, os, textwrap, time
from dataclasses import dataclass
from typing import Optional

try:
    import requests
except ImportError:
    raise SystemExit("pip install requests  # HTTP client for Molecule AI API")


PLATFORM_URL    = os.environ.get("PLATFORM_URL",    "https://your-deployment.moleculesai.app")
WORKSPACE_TOKEN = os.environ.get("WORKSPACE_TOKEN", "your-workspace-token")


# ─────────────────────────────────────────────────────────────────────────────
# Utilities
# ─────────────────────────────────────────────────────────────────────────────

def is_live_platform() -> bool:
    """Return True only when credentials point to a real deployment."""
    if "your-deployment" in PLATFORM_URL:
        return False
    if PLATFORM_URL.startswith("http://") and "localhost" not in PLATFORM_URL:
        return False
    if WORKSPACE_TOKEN in ("", "your-workspace-token", "demo-token"):
        return False
    return True


def divider(title: str) -> None:
    d = "═" * 68
    print(f"\n  {d}")
    print(f"  {title}")
    print(f"  {d}\n")


def step(num: int, title: str) -> None:
    print(f"  ┌{'─'*66}┐")
    print(f"  │  STEP {num}: {title:<60}│")
    print(f"  └{'─'*66}┘")
    print()


# ─────────────────────────────────────────────────────────────────────────────
# API client
# ─────────────────────────────────────────────────────────────────────────────

class A2AClient:
    """Minimal client for A2A protocol operations."""

    def __init__(self, platform_url: str, workspace_token: str):
        self.base = platform_url.rstrip("/")
        self.hdrs = {"Authorization": f"Bearer {workspace_token}", "Content-Type": "application/json"}

    # POST /workspaces/:id/a2a — send A2A task to workspace
    def send_task(self, workspace_id: str, task: str, run_id: str = "") -> dict:
        body = {"task": task}
        if run_id:
            body["run_id"] = run_id
        r = requests.post(
            f"{self.base}/workspaces/{workspace_id}/a2a",
            headers=self.hdrs, json=body, timeout=60,
        )
        r.raise_for_status()
        return r.json()

    # GET /registry/:id/peers — discover sibling workspaces
    def list_peers(self, parent_id: str) -> dict:
        r = requests.get(f"{self.base}/registry/{parent_id}/peers", headers=self.hdrs, timeout=15)
        r.raise_for_status()
        return r.json()

    # GET /workspaces/:id/state — poll workspace state
    def get_workspace_state(self, workspace_id: str) -> dict:
        r = requests.get(f"{self.base}/workspaces/{workspace_id}/state", headers=self.hdrs, timeout=15)
        r.raise_for_status()
        return r.json()


# ─────────────────────────────────────────────────────────────────────────────
# Simulated responses (no live platform needed)
# ─────────────────────────────────────────────────────────────────────────────

def simulate_peers() -> list[dict]:
    return [
        {
            "workspace_id": "ws-researcher-001",
            "name": "Research Lead",
            "role": "research",
            "status": "online",
            "remote_url": "https://ws-researcher-001.moleculesai.app",
        },
        {
            "workspace_id": "ws-codereview-001",
            "name": "Code Review Agent",
            "role": "code-review",
            "status": "online",
            "remote_url": "https://ws-codereview-001.moleculesai.app",
        },
        {
            "workspace_id": "ws-qa-001",
            "name": "QA Agent",
            "role": "qa",
            "status": "online",
            "remote_url": "https://ws-qa-001.moleculesai.app",
        },
    ]

def simulate_send_task(task: str) -> dict:
    return {
        "result": f"[Simulated result for: {task[:50]}...]",
        "status": "completed",
        "agent": "research-lead",
        "metadata": {
            "tool_trace": [
                {"tool": "web_search", "input": '{"query": "..."}', "output_preview": "[8 results]", "run_id": "run-001"},
                {"tool": "summarize_text", "input": '{"text": "..."}', "output_preview": "3 bullet summary", "run_id": "run-002"},
            ],
            "duration_ms": 4200,
            "tokens_used": 1820,
        },
    }

def simulate_direct_a2a(from_agent: str, to_agent: str, task: str) -> dict:
    return {
        "from": from_agent,
        "to": to_agent,
        "task": task,
        "result": f"Direct peer result from {to_agent}",
        "status": "completed",
        "transport": "direct",  # no platform relay
    }

def simulate_parallel_calls() -> dict:
    return {
        "status": "completed",
        "parallel_calls": [
            {"tool": "web_search", "run_id": "run-parallel-a", "output_preview": "[6 results in 89ms]"},
            {"tool": "read_file",  "run_id": "run-parallel-b", "output_preview": "config: postgres://... model: claude-sonnet-4"},
        ],
        "run_id_correlation": "Both calls completed within same agent turn; run_id pairs start/end events.",
    }


# ─────────────────────────────────────────────────────────────────────────────
# Visualization helpers
# ─────────────────────────────────────────────────────────────────────────────

def print_a2a_topology():
    """Print ASCII topology diagram."""
    print(textwrap.dedent("""\
      ┌─────────────────────────────────────────────────────────────────────┐
      │                    A2A Enterprise Topology                       │
      └─────────────────────────────────────────────────────────────────────┘

                         ┌──────────────────────┐
                         │  Molecule AI Platform │  ← A2A Registry + Proxy
                         │  (control plane)      │    - Workspace registry
                         └──────────┬───────────┘    - Auth validation
                                    │               - SSE streaming
           ┌────────────────────────┼────────────────────────┐
           │                        │                        │
           ▼                        ▼                        ▼
    ┌────────────┐          ┌────────────┐          ┌────────────┐
    │  PM Agent  │──────────▶│  Research  │◀────────▶│    QA     │
    │(orchestra- │          │   Lead     │  A2A     │   Agent   │
    │   tor)     │  A2A     │            │  direct  │           │
    └────────────┘          └────────────┘          └────────────┘
           │                        │
           │ A2A via proxy           │ A2A via proxy
           ▼                        ▼
    ┌────────────┐          ┌────────────┐
    │   Code     │◀────────▶│   PM      │
    │  Review    │  A2A     │           │
    └────────────┘          └────────────┘
    """))


def print_peer_table(peers: list[dict]):
    """Print peer discovery table."""
    print("  ┌─────────────────┬──────────────────────┬────────┬──────────┐")
    print("  │ Workspace ID    │ Name                 │ Role   │ Status   │")
    print("  ├─────────────────┼──────────────────────┼────────┼──────────┤")
    for p in peers:
        print(f"  │ {p['workspace_id']:15} │ {p['name']:20} │ {p['role']:6} │ {p['status']:8} │")
    print("  └─────────────────┴──────────────────────┴────────┴──────────┘")


# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

def main():
    is_live = is_live_platform()
    client  = A2AClient(PLATFORM_URL, WORKSPACE_TOKEN) if is_live else None

    print("""
    ╔══════════════════════════════════════════════════════════════════════╗
    ║      A2A Protocol — Enterprise Multi-Agent Orchestration             ║
    ║      PR #1686 (molecule-core)                                        ║
    ║                                                                      ║
    ║  The A2A Protocol enables peer-to-peer, JSON-RPC communication       ║
    ║  between Molecule AI agents — no proprietary SDK, no platform      ║
    ║  relay for direct calls.                                            ║
    ╚══════════════════════════════════════════════════════════════════════╝
    """)

    print_a2a_topology()

    # ── Step 1 ────────────────────────────────────────────────────────────
    step(1, "Discover sibling agents via registry")
    print("  GET /registry/:parent_id/peers")
    print()
    print("  Sub-agents register with the platform on boot. The PM agent")
    print("  calls GET /registry/:parent_id/peers once, then caches the")
    print("  peer URLs for direct A2A calls.\n")
    if is_live:
        try:
            parent_id = "ws-pm-orchestrator"
            peers = client.list_peers(parent_id)
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        peers = simulate_peers()
    print_peer_table(peers)
    print("  ✓ PM agent now knows about Research Lead, Code Review, and QA Agent")
    print("  ✓ Peer URLs cached — direct A2A calls bypass the platform proxy")

    # ── Step 2 ────────────────────────────────────────────────────────────
    step(2, "Direct peer-to-peer A2A call (no platform relay)")
    print("  Once peers are cached, sub-agents call each other DIRECTLY:")
    print()
    print("    Research Lead → QA Agent (A2A, direct)")
    print("    curl -X POST <qa-remote-url>/a2a \\")
    print("      -H 'X-Workspace-ID: ws-researcher-001' \\")
    print("      -H 'Authorization: Bearer <token>' \\")
    print("      -d '{\"jsonrpc\": \"2.0\", \"method\": \"verify_facts\", ...}'")
    print()
    print("  The platform is in the DISCOVERY path, not the DATA path.")
    print("  Once peers are known, agents talk directly to each other.\n")
    if not is_live:
        result = simulate_direct_a2a("research-lead", "qa-agent",
                                      "verify_facts against source docs")
        print(f"  ✓ Direct A2A result: {result['result']}")
        print(f"    transport: {result['transport']} (no platform relay)")

    # ── Step 3 ────────────────────────────────────────────────────────────
    step(3, "PM agent delegates via platform A2A proxy")
    print("  POST /workspaces/:id/a2a  (platform acts as A2A proxy)")
    print("  Body: {\"task\": \"Research AI observability standards...\"}")
    print()
    print("  The PM agent delegates complex tasks through the platform proxy.")
    print("  The platform validates the caller's auth, routes to the target")
    print("  workspace, and streams back via SSE.\n")
    if is_live:
        try:
            result = client.send_task("ws-researcher-001", "Summarize A2A protocol observability best practices")
            print(f"  ✓ Response: {result.get('result', '')[:80]}...")
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        result = simulate_send_task("Summarize A2A protocol observability best practices")
        print(f"  ✓ Agent completed task in {result['metadata']['duration_ms']}ms")
        print(f"    tools called: {[t['tool'] for t in result['metadata']['tool_trace']]}")
        print(f"    tokens used: {result['metadata']['tokens_used']}")

    # ── Step 4 ────────────────────────────────────────────────────────────
    step(4, "Parallel tool calls with run_id correlation")
    print("  LangGraph agents can call multiple tools concurrently.")
    print("  The platform uses run_id to pair on_tool_start → on_tool_end events.\n")
    if not is_live:
        result = simulate_parallel_calls()
        print("  ┌─────────────────────────────────────────────────────────────┐")
        print("  │  Parallel tool calls (same agent turn):                    │")
        print("  ├─────────────────────────────────────────────────────────────┤")
        for call in result["parallel_calls"]:
            print(f"  │  run_id={call['run_id']:16}  tool={call['tool']:12}  │")
            print(f"  │    output_preview: {call['output_preview'][:38]:38} │")
        print("  └─────────────────────────────────────────────────────────────┘")
        print(f"\n  ✓ run_id correlation: {result['run_id_correlation']}")

    # ── Architecture ────────────────────────────────────────────────────────
    divider("Architecture Summary")
    print(textwrap.dedent("""\
      Workspace boot:
        → Agent calls POST /registry/register with capabilities
        → Platform stores workspace_id + remote_url + status in registry
        → Agent receives workspace token (Bearer auth)

      Peer discovery:
        → GET /registry/:parent_id/peers  (called once by PM/orchestrator)
        → Returns list of sibling workspaces: {workspace_id, remote_url, status}
        → PM caches peer URLs in memory

      Direct A2A (peer-to-peer):
        → Agent POSTs JSON-RPC 2.0 to peer's remote_url directly
        → Includes X-Workspace-ID header for auth
        → Bearer token in Authorization header
        → Platform NOT in data path — agents talk directly
        → Resilient to brief platform outages (PEER CACHING)

      A2A via proxy (cross-workspace delegation):
        → PM calls POST /workspaces/:id/a2a (platform proxy)
        → Platform validates auth, routes to target workspace
        → SSE streaming response back to caller
        → Used when target workspace URL is unknown or external

      JSON-RPC 2.0 message format:
        → {"jsonrpc": "2.0", "id": 1, "method": "task", "params": {...}}
        → Response: {"jsonrpc": "2.0", "id": 1, "result": {...}}

      Security:
      • CanCommunicate() enforces org hierarchy — agents can only A2A
        with siblings in the same org (parent/child chain)
      • Auth tokens validated at every hop
      • SSRF protection in a2a_proxy.go (CWE-918)
    """))

    divider("Reference")
    print("  JSON-RPC spec : https://www.jsonrpc.org/specification")
    print("  A2A protocol : docs/api-protocol/a2a-protocol.md")
    print("  Demo path    : docs/marketing/devrel/demos/a2a-enterprise-demo/")
    print()
    print("  Set PLATFORM_URL + WORKSPACE_TOKEN to run against a live platform.")
    print("  Screencast storyboard: marketing/devrel/demos/screencasts/storyboard-a2a-enterprise.md")


if __name__ == "__main__":
    main()


# ─────────────────────────────────────────────────────────────────────────────
# Key design notes
# ─────────────────────────────────────────────────────────────────────────────
"""
A2A Protocol key design decisions:

1. Peer discovery vs data path separation:
   The platform is in the discovery path (GET /registry/peers) but NOT in
   the data path for direct peer-to-peer calls. This is intentional:
   - Reduces latency (direct network hop vs proxy hop)
   - Provides resilience (agents keep working if platform is briefly down)
   - Removes platform as a bottleneck for high-throughput agent teams

2. JSON-RPC 2.0 over SSE:
   A2A uses JSON-RPC 2.0 for messages, wrapped in SSE for streaming.
   This gives: structured messages, error objects per spec, event stream
   compatibility with standard HTTP infrastructure.

3. run_id for parallel tool call pairing:
   When LangGraph calls two tools at once, both start events fire before
   either end event. Without run_id pairing, output_previews would collide
   in a simple list. run_id ensures each output_preview pairs with its
   matching tool entry.

4. CanCommunicate() org hierarchy:
   Access control is enforced via CanCommunicate() in the registry.
   Agents in org A cannot communicate with agents in org B — only the
   parent/child chain within the same org hierarchy.
"""
