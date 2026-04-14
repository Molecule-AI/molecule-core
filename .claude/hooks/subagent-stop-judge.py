#!/usr/bin/env python3
"""SubagentStop — optional self-check prompt before accepting subagent output.

Disabled by default. Enable per-tick with: touch .claude/judge-subagents

When on, asks the orchestrator to verify the subagent's output addresses
the original task. Cost-free MVP — does NOT call an LLM. Future versions
can plug in an actual llm-judge call gated by a separate toggle.
"""
import json
import os
import sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from _lib import read_input, emit, warn_to_stderr  # noqa

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
TOGGLE = os.path.join(REPO, ".claude", "judge-subagents")


def main() -> None:
    if not os.path.isfile(TOGGLE):
        return

    data = read_input()
    last = data.get("last_assistant_message", "")
    agent = data.get("agent_type", "unknown")
    if not last or len(last) < 100:
        return

    snippet = last[:400].replace("\n", " ")
    emit({
        "decision": "block",
        "reason": (
            f"subagent-judge: {agent} returned. Before proceeding, re-read its last message "
            f"(snippet: {snippet}...) and confirm: did it actually address the original task? "
            f"If unsure, re-spawn with a tighter prompt."
        ),
    })


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        warn_to_stderr(f"[subagent-stop hook error] {e}")
        sys.exit(0)
