#!/usr/bin/env python3
"""External Workspace Bridge — plug any AI agent into Molecule AI via A2A.

Registers as an external workspace (no Docker container) and processes
incoming A2A messages using a configurable backend processor.

Usage:
  # Claude Code backend (default)
  python3 scripts/claude-code-bridge.py

  # OpenAI API backend
  python3 scripts/claude-code-bridge.py --processor openai --model gpt-4.1-mini

  # Anthropic API backend
  python3 scripts/claude-code-bridge.py --processor anthropic --model claude-sonnet-4-6

  # Forward to any HTTP endpoint
  python3 scripts/claude-code-bridge.py --processor http --url http://my-agent:8000/chat

  # Echo (testing)
  python3 scripts/claude-code-bridge.py --processor echo

  # Management
  python3 scripts/claude-code-bridge.py --inbox   # Show messages
  python3 scripts/claude-code-bridge.py --clear   # Clear inbox
  python3 scripts/claude-code-bridge.py --stop    # Stop bridge

Environment variables:
  PLATFORM_URL        Platform API (default: http://localhost:8080)
  BRIDGE_PORT         Listen port (default: 9999)
  BRIDGE_NAME         Workspace name (default: Claude Code Advisor)
  BRIDGE_PARENT_ID    Parent workspace ID for hierarchy
  OPENAI_API_KEY      For --processor openai
  ANTHROPIC_API_KEY   For --processor anthropic
  BRIDGE_FORWARD_URL  For --processor http
"""

import argparse
import json
import logging
import os
import signal
import sys
from pathlib import Path

logging.basicConfig(level=logging.INFO, format="%(asctime)s [bridge] %(message)s")
logger = logging.getLogger("bridge")

BRIDGE_DIR = Path(__file__).parent.parent / ".claude-bridge"
INBOX = BRIDGE_DIR / "inbox.jsonl"
PID_FILE = BRIDGE_DIR / "bridge.pid"

BRIDGE_DIR.mkdir(exist_ok=True)


def show_inbox():
    if not INBOX.exists():
        print("No messages.")
        return
    unread = 0
    for line in INBOX.read_text().splitlines():
        try:
            e = json.loads(line)
            unread += 1
            print(f"  [{e.get('sender_name','?')}] {e.get('text','')[:120]}")
        except json.JSONDecodeError:
            continue
    print(f"\n{unread} message(s)" if unread else "No messages.")


def main():
    parser = argparse.ArgumentParser(description="External Workspace Bridge")
    parser.add_argument("--processor", default="claude-code",
        help="Backend processor: claude-code, openai, anthropic, http, echo")
    parser.add_argument("--model", default="", help="Model name for the processor")
    parser.add_argument("--url", default="", help="URL for http processor")
    parser.add_argument("--name", default=os.environ.get("BRIDGE_NAME", "Claude Code Advisor"))
    parser.add_argument("--role", default="CEO technical advisor — code review, architecture, debugging")
    parser.add_argument("--port", type=int, default=int(os.environ.get("BRIDGE_PORT", "9999")))
    parser.add_argument("--parent-id", default=os.environ.get("BRIDGE_PARENT_ID", ""))
    parser.add_argument("--inbox", action="store_true", help="Show inbox")
    parser.add_argument("--clear", action="store_true", help="Clear inbox")
    parser.add_argument("--stop", action="store_true", help="Stop bridge")
    args = parser.parse_args()

    if args.inbox:
        show_inbox()
        return
    if args.clear:
        INBOX.unlink(missing_ok=True)
        print("Inbox cleared")
        return
    if args.stop:
        if PID_FILE.exists():
            try:
                os.kill(int(PID_FILE.read_text().strip()), signal.SIGTERM)
                print("Bridge stopped")
            except ProcessLookupError:
                print("Bridge was not running")
            PID_FILE.unlink(missing_ok=True)
        return

    # Import here to keep --inbox/--stop fast
    from bridge.processor import create_processor
    from bridge.platform import PlatformClient
    from bridge.server import create_server

    PID_FILE.write_text(str(os.getpid()))

    # Create processor
    kwargs = {}
    if args.model:
        kwargs["model"] = args.model
    if args.url:
        kwargs["url"] = args.url
    processor = create_processor(args.processor, **kwargs)
    logger.info(f"Processor: {args.processor} ({type(processor).__name__})")

    # Register with platform
    platform_url = os.environ.get("PLATFORM_URL", "http://localhost:8080")
    client = PlatformClient(platform_url, args.port, BRIDGE_DIR)
    ws_id = client.register(args.name, args.role, parent_id=args.parent_id)
    client.start_heartbeat()

    # Start A2A server
    server = create_server(args.port, processor, INBOX, client.resolve_name)
    logger.info(f"Listening on :{args.port} | Workspace: {ws_id}")
    logger.info(f"Agents see '{args.name}' in list_peers → delegate_task to interact")

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        server.shutdown()
        PID_FILE.unlink(missing_ok=True)


if __name__ == "__main__":
    main()
