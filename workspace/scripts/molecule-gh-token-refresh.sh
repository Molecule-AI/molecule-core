#!/bin/bash
# molecule-gh-token-refresh.sh — background daemon that keeps GitHub
# credentials fresh inside Molecule AI workspace containers.
#
# Runs as a background process started by entrypoint.sh. Every
# REFRESH_INTERVAL_SEC (default 45 min = 2700s) it calls the credential
# helper's _refresh_gh action which:
#   1. Fetches a fresh installation token from the platform API
#   2. Updates the local cache (used by git credential helper)
#   3. Runs `gh auth login --with-token` so `gh` CLI stays authenticated
#   4. Writes ~/.gh_token for any scripts that read it
#
# The daemon logs to stderr (captured by Docker) and is designed to be
# fire-and-forget — if a single refresh fails, it logs the error and
# retries on the next interval. The credential helper itself has a
# fallback chain (cache > API > env var) so a missed refresh is not
# immediately fatal.
#
# Usage (from entrypoint.sh):
#   nohup /app/scripts/molecule-gh-token-refresh.sh &
#
set -uo pipefail

HELPER_SCRIPT="/app/scripts/molecule-git-token-helper.sh"
REFRESH_INTERVAL_SEC="${TOKEN_REFRESH_INTERVAL_SEC:-2700}"  # 45 min

log() {
    echo "[molecule-gh-token-refresh] $(date -u '+%Y-%m-%dT%H:%M:%SZ') $*" >&2
}

# Wait a short time before the first refresh to let the container finish
# booting and .auth_token to be written by the runtime's register call.
INITIAL_DELAY_SEC="${TOKEN_REFRESH_INITIAL_DELAY_SEC:-60}"
log "starting (interval=${REFRESH_INTERVAL_SEC}s, initial_delay=${INITIAL_DELAY_SEC}s)"
sleep "${INITIAL_DELAY_SEC}"

# Initial refresh — prime the cache + gh auth immediately after boot.
log "initial token refresh"
if bash "${HELPER_SCRIPT}" _refresh_gh 2>&1; then
    log "initial refresh succeeded"
else
    log "initial refresh failed (will retry in ${REFRESH_INTERVAL_SEC}s)"
fi

# Steady-state loop.
while true; do
    sleep "${REFRESH_INTERVAL_SEC}"
    log "periodic token refresh"
    if bash "${HELPER_SCRIPT}" _refresh_gh 2>&1; then
        log "refresh succeeded"
    else
        log "refresh failed (will retry in ${REFRESH_INTERVAL_SEC}s)"
    fi
done
