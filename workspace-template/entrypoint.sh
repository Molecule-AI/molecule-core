#!/bin/bash
# No set -e — individual commands handle their own errors gracefully

# ──────────────────────────────────────────────────────────
# Volume ownership fix (runs as root)
# ──────────────────────────────────────────────────────────
# Docker creates volume contents as root. The agent process runs as UID 1000
# and needs to write to /configs (CLAUDE.md, skills, plugins) and /workspace
# (cloned repos, scratch files). Fix ownership once at startup so every
# future file operation works without per-file chown hacks.
if [ "$(id -u)" = "0" ]; then
    # Fix /configs recursively (plugins, CLAUDE.md, skills — small directory)
    chown -R agent:agent /configs 2>/dev/null
    # Fix /workspace top-level only — it may be a bind-mounted host repo with
    # thousands of files. Recursive chown would take minutes and change the
    # host filesystem's ownership. The agent only needs to write at the top level.
    chown agent:agent /workspace 2>/dev/null
    # Re-exec this script as the agent user via gosu (clean PID 1 handoff)
    exec gosu agent "$0" "$@"
fi

# ──────────────────────────────────────────────────────────
# Everything below runs as the agent user (UID 1000)
# ──────────────────────────────────────────────────────────

# Ensure user-installed packages are in PATH
export PATH="$HOME/.local/bin:$PATH"

# Determine runtime from config.yaml
RUNTIME=$(python3 -c "
import yaml
from pathlib import Path
cfg_path = Path('/configs/config.yaml')
if cfg_path.exists():
    cfg = yaml.safe_load(cfg_path.read_text()) or {}
    print(cfg.get('runtime', 'langgraph'))
else:
    print('langgraph')
" 2>/dev/null || echo "langgraph")

# Normalize runtime name for directory lookup (claude-code -> claude_code)
ADAPTER_DIR=$(echo "$RUNTIME" | tr '-' '_')

echo "=== Molecule AI Workspace ==="
echo "Runtime: $RUNTIME"

# Install adapter-specific Python requirements (skip if already pre-installed in image)
REQ_FILE="/app/adapters/${ADAPTER_DIR}/requirements.txt"
if [ -f "$REQ_FILE" ]; then
    if grep -q '^[^#]' "$REQ_FILE" 2>/dev/null; then
        # Check if first package is already installed
        FIRST_PKG=$(grep '^[^#]' "$REQ_FILE" | head -1 | sed 's/[>=<].*//')
        if python3 -c "import importlib; importlib.import_module('${FIRST_PKG//-/_}')" 2>/dev/null; then
            echo "Adapter deps already installed (${FIRST_PKG})"
        else
            echo "Installing Python adapter dependencies..."
            pip install --no-cache-dir --user -q -r "$REQ_FILE" 2>&1 | tail -3
        fi
    fi
fi

# Install adapter-specific npm packages (for Node.js-based runtimes like OpenClaw)
NPM_FILE="/app/adapters/${ADAPTER_DIR}/package.json"
if [ -f "$NPM_FILE" ]; then
    echo "Installing npm adapter dependencies..."
    cd "/app/adapters/${ADAPTER_DIR}" && npm install --production 2>&1 | tail -3
    cd /app
fi

exec python3 main.py
