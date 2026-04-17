#!/usr/bin/env python3
"""
Spike #745 — Anthropic Managed Agents as a Molecule workspace executor.

This script validates the managed-agents-2026-04-01 beta API against the
criteria in issue #742:
  - Authentication & agent provisioning
  - Session start (cold-start latency)
  - Round-trip prompt/response (per-turn latency)
  - State persistence across turns (session continuity)
  - Clean shutdown

Usage:
    ANTHROPIC_API_KEY=sk-ant-... python demo.py

Optional env vars:
    MA_SKIP_CLEANUP=1   keep the agent/session alive after the run
    MA_VERBOSE=1        print every SSE event type (not just agent messages)
"""

import os
import sys
import time
import json

try:
    import anthropic
except ImportError:
    sys.exit("anthropic SDK not installed — run: pip install anthropic")

# ── helpers ──────────────────────────────────────────────────────────────────

VERBOSE = os.getenv("MA_VERBOSE") == "1"
SKIP_CLEANUP = os.getenv("MA_SKIP_CLEANUP") == "1"


def ts() -> float:
    return time.monotonic()


def elapsed(start: float) -> float:
    return round(time.monotonic() - start, 3)


def collect_turn(client: anthropic.Anthropic, session_id: str, message: str) -> tuple[str, float]:
    """
    Stream-first turn: open the SSE stream, send the user message inside the
    context manager, then drain events until session.status_idle or
    session.status_terminated.

    Returns (agent_reply_text, round_trip_seconds).
    Raises RuntimeError if the session terminates unexpectedly mid-turn.
    """
    reply_parts: list[str] = []
    turn_start = ts()

    with client.beta.sessions.stream(session_id=session_id) as stream:
        # Send inside the stream so we never miss early events
        client.beta.sessions.events.send(
            session_id=session_id,
            events=[
                {
                    "type": "user.message",
                    "content": [{"type": "text", "text": message}],
                }
            ],
        )

        for event in stream:
            if VERBOSE:
                print(f"  [evt] {event.type}", flush=True)

            if event.type == "agent.message":
                for block in event.content:
                    if block.type == "text":
                        reply_parts.append(block.text)

            elif event.type == "session.status_idle":
                break  # normal turn completion

            elif event.type == "session.status_terminated":
                # session ended — surface whatever text arrived
                if reply_parts:
                    break
                raise RuntimeError("Session terminated unexpectedly during turn")

    return "".join(reply_parts), elapsed(turn_start)


# ── main ─────────────────────────────────────────────────────────────────────

def main() -> None:
    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        sys.exit("ANTHROPIC_API_KEY not set")

    client = anthropic.Anthropic(api_key=api_key)

    # ── 1. Create environment ─────────────────────────────────────────────────
    print("=== Managed Agents Spike #745 ===\n")
    print("Step 1: Creating cloud environment…")
    t0 = ts()
    environment = client.beta.environments.create(
        name="molecule-spike-742",
        config={
            "type": "cloud",
            "networking": {"type": "unrestricted"},
        },
    )
    env_time = elapsed(t0)
    print(f"  environment_id : {environment.id}")
    print(f"  env create time: {env_time}s\n")

    # ── 2. Create agent ───────────────────────────────────────────────────────
    print("Step 2: Creating agent…")
    t0 = ts()
    agent = client.beta.agents.create(
        name="molecule-spike-agent",
        model="claude-opus-4-7",
        system=(
            "You are a stateful test agent for the Molecule AI spike. "
            "When asked to remember something, confirm you will. "
            "On subsequent turns, recall it accurately."
        ),
        tools=[
            {"type": "agent_toolset_20260401", "default_config": {"enabled": True}}
        ],
    )
    agent_time = elapsed(t0)
    print(f"  agent_id  : {agent.id}")
    print(f"  version   : {agent.version}")
    print(f"  agent create time: {agent_time}s\n")

    # ── 3. Create session (cold start) ────────────────────────────────────────
    print("Step 3: Creating session (cold start)…")
    cold_start = ts()
    session = client.beta.sessions.create(
        agent={"type": "agent", "id": agent.id, "version": agent.version},
        environment_id=environment.id,
        title="molecule-spike-742-session",
    )
    cold_time = elapsed(cold_start)
    print(f"  session_id : {session.id}")
    print(f"  status     : {session.status}")
    print(f"  cold-start : {cold_time}s\n")

    # ── 4. Turn 1 — establish a fact the agent should remember ────────────────
    turn1_prompt = (
        "Please remember this token for the rest of our conversation: "
        "MOLECULE_SPIKE_7a3f. "
        "What is today's task? Reply in one sentence."
    )
    print(f"Turn 1 prompt:\n  {turn1_prompt!r}\n")
    turn1_reply, turn1_time = collect_turn(client, session.id, turn1_prompt)
    print(f"Turn 1 reply ({turn1_time}s):\n  {turn1_reply!r}\n")

    # ── 5. Turn 2 — verify state persistence ─────────────────────────────────
    turn2_prompt = "What was the token I asked you to remember?"
    print(f"Turn 2 prompt:\n  {turn2_prompt!r}\n")
    turn2_reply, turn2_time = collect_turn(client, session.id, turn2_prompt)
    print(f"Turn 2 reply ({turn2_time}s):\n  {turn2_reply!r}\n")

    # ── 6. State continuity check ─────────────────────────────────────────────
    token_recalled = "MOLECULE_SPIKE_7a3f" in turn2_reply
    print("=== Results ===")
    print(f"  environment create : {env_time}s")
    print(f"  agent create       : {agent_time}s")
    print(f"  cold-start (session create → ready) : {cold_time}s")
    print(f"  turn 1 round-trip  : {turn1_time}s")
    print(f"  turn 2 round-trip  : {turn2_time}s")
    print(f"  state continuity   : {'PASS — token recalled' if token_recalled else 'FAIL — token not found in turn 2'}")

    # Emit JSON summary for easy parsing in CI / PR bots
    summary = {
        "environment_id": environment.id,
        "agent_id": agent.id,
        "session_id": session.id,
        "timings": {
            "environment_create_s": env_time,
            "agent_create_s": agent_time,
            "cold_start_s": cold_time,
            "turn1_rtt_s": turn1_time,
            "turn2_rtt_s": turn2_time,
        },
        "state_continuity_pass": token_recalled,
    }
    print("\nJSON summary:")
    print(json.dumps(summary, indent=2))

    # ── 7. Cleanup ────────────────────────────────────────────────────────────
    if not SKIP_CLEANUP:
        print("\nCleaning up…")
        try:
            client.beta.sessions.delete(session_id=session.id)
            print(f"  session {session.id} deleted")
        except Exception as exc:
            print(f"  session delete warning: {exc}")
        # Agents are persistent/shared — don't delete unless explicitly asked.
        # Set MA_SKIP_CLEANUP=1 and clean up manually with:
        #   client.beta.agents.delete(agent.id)
        print(f"  agent {agent.id} kept (persistent object; delete manually if needed)")
    else:
        print(f"\nSKIP_CLEANUP=1 — session and agent left alive.")
        print(f"  Session: {session.id}")
        print(f"  Agent:   {agent.id}")

    sys.exit(0 if token_recalled else 1)


if __name__ == "__main__":
    main()
