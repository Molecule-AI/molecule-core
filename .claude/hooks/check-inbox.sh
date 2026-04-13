#!/bin/bash
# Check for unread agent messages in the bridge inbox
INBOX="/Users/hongming/Documents/GitHub/molecule-monorepo/.claude-bridge/inbox.jsonl"
if [ -f "$INBOX" ]; then
  UNREAD=$(grep -c '"responded": false' "$INBOX" 2>/dev/null || echo 0)
  if [ "$UNREAD" -gt 0 ]; then
    echo "[INBOX] You have $UNREAD unread message(s) from agents. Run: cat .claude-bridge/inbox.jsonl"
  fi
fi
