#!/bin/bash
# molecule-git-token-helper.sh — git credential helper for GitHub App tokens
#
# Fetches a fresh GitHub App installation token from the Molecule AI
# platform endpoint GET /admin/github-installation-token on every git
# push/fetch, so workspace containers never use an expired GH_TOKEN after
# the ~60 min GitHub App token TTL.
#
# # Setup (called once at provision time or initial_prompt)
#
#   git config --global \
#     "credential.https://github.com.helper" \
#     "!/workspace/scripts/molecule-git-token-helper.sh"
#
# # How git calls this helper
#
# git passes the action as the first positional arg.  The protocol is:
#   get   → output credentials on stdout (we handle this)
#   store → persist credentials (no-op — we never cache)
#   erase → revoke credentials (no-op — platform manages lifecycle)
#
# On `get`, git reads key=value pairs terminated by an empty line.
# We must emit at minimum:
#   username=x-access-token
#   password=<token>
#   (blank line)
#
# # Auth
#
# The platform endpoint requires a valid workspace bearer token.  The
# token is stored at ${CONFIGS_DIR}/.auth_token (written by platform_auth.py
# on first /registry/register).  Workspace env var PLATFORM_URL defaults
# to http://platform:8080.
#
# # Fallback
#
# If the platform endpoint is unreachable (e.g. network partition) or
# returns non-200, the script exits 1 without printing credentials so git
# will fall through to the next helper in the chain (if any).  This
# preserves the operator's fallback PAT from .env if present.
#
# # gh CLI re-auth (30-min cron)
#
# To also fix `gh` CLI auth, run this from a workspace cron prompt:
#
#   token=$(bash /workspace/scripts/molecule-git-token-helper.sh _fetch_token)
#   echo "$token" | gh auth login --with-token
#
# (The _fetch_token private action returns only the raw token string.)
#
set -euo pipefail

PLATFORM_URL="${PLATFORM_URL:-http://platform:8080}"
CONFIGS_DIR="${CONFIGS_DIR:-/configs}"
TOKEN_FILE="${CONFIGS_DIR}/.auth_token"
# #1068: use workspace-scoped path (WorkspaceAuth) instead of admin path
# (AdminAuth rejects workspace bearer tokens since PR #729).
WORKSPACE_ID="${WORKSPACE_ID:-}"
if [ -n "$WORKSPACE_ID" ]; then
    ENDPOINT="${PLATFORM_URL}/workspaces/${WORKSPACE_ID}/github-installation-token"
else
    ENDPOINT="${PLATFORM_URL}/admin/github-installation-token"
fi

# _fetch_token — internal helper; also callable directly from cron.
# Outputs the raw token string on success; exits non-zero on failure.
_fetch_token() {
    if [ ! -f "${TOKEN_FILE}" ]; then
        echo "[molecule-git-token-helper] .auth_token not found at ${TOKEN_FILE}" >&2
        exit 1
    fi

    bearer=$(cat "${TOKEN_FILE}" | tr -d '[:space:]')
    if [ -z "${bearer}" ]; then
        echo "[molecule-git-token-helper] .auth_token is empty" >&2
        exit 1
    fi

    response=$(curl -sf \
        -H "Authorization: Bearer ${bearer}" \
        -H "Accept: application/json" \
        --max-time 10 \
        "${ENDPOINT}" 2>&1) || {
        echo "[molecule-git-token-helper] platform request failed: ${response}" >&2
        exit 1
    }

    # Parse {"token":"ghs_...","expires_at":"..."} with sed (no jq dependency).
    token=$(echo "${response}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
    if [ -z "${token}" ]; then
        echo "[molecule-git-token-helper] empty token in platform response: ${response}" >&2
        exit 1
    fi

    echo "${token}"
}

ACTION="${1:-get}"

case "${ACTION}" in
    get)
        token=$(_fetch_token) || exit 1
        # Emit git credential protocol response.
        printf 'username=x-access-token\n'
        printf 'password=%s\n' "${token}"
        printf '\n'
        ;;
    store|erase)
        # No-op — the platform manages token lifecycle.
        ;;
    _fetch_token)
        # Private action for cron-based gh auth login --with-token.
        _fetch_token
        ;;
    *)
        echo "[molecule-git-token-helper] unknown action: ${ACTION}" >&2
        exit 1
        ;;
esac
