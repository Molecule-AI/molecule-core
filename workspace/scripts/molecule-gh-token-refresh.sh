#!/bin/bash
# molecule-gh-token-refresh.sh — background daemon that keeps GitHub
# credentials fresh inside Molecule AI workspace containers.
#
# Started by entrypoint.sh under a respawn wrapper. Every
# REFRESH_INTERVAL_SEC + jitter (default 45 min ± 2 min) it calls the
# credential helper's _refresh_gh action.
#
# # Jitter
# A 0..120s random offset prevents 39 containers from synchronizing
# their refresh requests against /workspaces/:id/github-installation-token.
#
# # Security
# - This daemon NEVER prints token values. Failures log the helper's
#   exit code only, not its stderr, so token bytes can't leak via the
#   docker log pipeline.
# - The helper script is responsible for chmod 600 on cache files.
#
set -uo pipefail

HELPER_SCRIPT="${TOKEN_HELPER_SCRIPT:-/app/scripts/molecule-git-token-helper.sh}"
REFRESH_INTERVAL_SEC="${TOKEN_REFRESH_INTERVAL_SEC:-2700}"  # 45 min
JITTER_MAX_SEC="${TOKEN_REFRESH_JITTER_SEC:-120}"
INITIAL_DELAY_SEC="${TOKEN_REFRESH_INITIAL_DELAY_SEC:-60}"

log() {
    echo "[molecule-gh-token-refresh] $(date -u '+%Y-%m-%dT%H:%M:%SZ') $*" >&2
}

jittered_sleep() {
    local base="$1"
    local jitter=$((RANDOM % (JITTER_MAX_SEC + 1)))
    sleep $((base + jitter))
}

log "starting (interval=${REFRESH_INTERVAL_SEC}s ± ${JITTER_MAX_SEC}s, initial_delay=${INITIAL_DELAY_SEC}s)"
sleep "${INITIAL_DELAY_SEC}"

# Initial refresh — prime the cache + gh auth immediately after boot.
# Discard helper output to /dev/null so token can't leak via docker logs.
log "initial token refresh"
if bash "${HELPER_SCRIPT}" _refresh_gh >/dev/null 2>&1; then
    log "initial refresh succeeded"
else
    log "initial refresh failed (rc=$?) — will retry in ~${REFRESH_INTERVAL_SEC}s"
fi

# Steady-state loop.
while true; do
    jittered_sleep "${REFRESH_INTERVAL_SEC}"
    log "periodic token refresh"
    if bash "${HELPER_SCRIPT}" _refresh_gh >/dev/null 2>&1; then
        log "refresh succeeded"
    else
        log "refresh failed (rc=$?) — will retry in ~${REFRESH_INTERVAL_SEC}s"
    fi
done
