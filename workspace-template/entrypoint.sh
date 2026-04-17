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
    # /workspace handling:
    #   - Always fix the top-level dir so agent can create files in it.
    #   - If the contents are root-owned (common on Docker Desktop / Windows
    #     bind mounts where host uid maps to 0 inside the container), do a
    #     full recursive chown — otherwise git clone, pip install, and file
    #     writes under /workspace fail with EACCES (issue #13). On normal
    #     Linux Docker with matching uids this branch is skipped, so we keep
    #     the fast startup for the common case.
    chown agent:agent /workspace 2>/dev/null
    if [ -d /workspace ]; then
        # Sample the first entry inside /workspace; if it's root-owned assume
        # the whole tree is a root-owned bind mount and recursively chown.
        first_entry=$(find /workspace -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)
        if [ -n "$first_entry" ] && [ "$(stat -c '%u' "$first_entry" 2>/dev/null)" = "0" ]; then
            echo "[entrypoint] /workspace contents are root-owned — chowning recursively to agent (uid 1000)"
            chown -R agent:agent /workspace 2>/dev/null
        fi
    fi
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

echo "=== Molecule AI Workspace ==="
echo "Runtime: $RUNTIME"

# ──────────────────────────────────────────────────────────
# GitHub credential helper — issue #547
# ──────────────────────────────────────────────────────────
# GitHub App installation tokens expire after ~60 min.  The platform
# exposes GET /admin/github-installation-token (backed by the plugin's
# in-process refreshing cache) so workspaces can always get a valid
# token without restarting.
#
# Register molecule-git-token-helper.sh as the git credential helper for
# github.com.  git calls it on every push/fetch; it hits the platform
# endpoint and emits a fresh token.  Falls through to any existing
# credential helper (e.g. operator .env PAT) if the platform is
# unreachable.
#
# Idempotent — safe to re-run on restart.
HELPER_SCRIPT="/workspace-template/scripts/molecule-git-token-helper.sh"
if [ -f "${HELPER_SCRIPT}" ]; then
    git config --global \
        "credential.https://github.com.helper" \
        "!${HELPER_SCRIPT}" 2>/dev/null || true
    echo "[entrypoint] git credential helper registered (molecule-git-token-helper)"
else
    echo "[entrypoint] WARNING: molecule-git-token-helper.sh not found at ${HELPER_SCRIPT} — GitHub tokens may expire after 60 min"
fi

# NOTE: Adapter-specific deps are now pre-installed in each adapter's Docker image
# (standalone template repos). Each image installs molecule-ai-workspace-runtime
# from PyPI plus the adapter-specific requirements. No per-runtime pip install needed here.

exec python3 main.py
