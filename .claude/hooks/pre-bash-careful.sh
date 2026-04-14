#!/usr/bin/env bash
# PreToolUse hook for Bash. Enforces careful-mode at the harness level
# rather than relying on the agent to remember. Exit 2 / JSON deny blocks.
exec python3 "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/pre-bash-careful.py"
