#!/usr/bin/env bash
# rebuild-runtime-images.sh — Rebuild all 6 workspace runtime Docker images.
#
# Run this script from the repo root (or from workspace-template/) after any
# change to workspace-template/Dockerfile, entrypoint.sh, or the git credential
# helper scripts. Also run after PR #640 merged.
#
# What this does:
#   1. Builds workspace-template:base from the monorepo Dockerfile (includes
#      the fixed entrypoint.sh + molecule-git-token-helper.sh)
#   2. For each runtime adapter, clones its standalone repo to a temp dir,
#      patches its Dockerfile to:
#        a. COPY the git credential helper into the image
#        b. Set git config --system to register the helper globally
#      Then builds and tags workspace-template:<runtime>.
#
# Why the patch is needed:
#   Standalone adapter images (molecule-ai-workspace-template-*) use
#   ENTRYPOINT ["molecule-runtime"] — they do not run entrypoint.sh, so the
#   git config registration from entrypoint.sh never fires for them. Baking
#   it into the image via git config --system at Docker build time is the
#   correct permanent fix (issue #613 / PR #640).
#
# Prerequisites: docker, git, gh (authenticated)
#
# Usage (from repo root):
#   bash workspace-template/rebuild-runtime-images.sh
#
# To rebuild a single runtime:
#   bash workspace-template/rebuild-runtime-images.sh claude-code
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
HELPER_SCRIPT="${SCRIPT_DIR}/scripts/molecule-git-token-helper.sh"
VALID_RUNTIMES=(langgraph claude-code openclaw crewai autogen deepagents)

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'
log()  { echo -e "${GREEN}[rebuild]${NC} $1"; }
warn() { echo -e "${YELLOW}[rebuild]${NC} $1"; }
err()  { echo -e "${RED}[rebuild]${NC} $1"; }

# ─────────────────────────────────────────────────────
# Argument: optional single runtime to rebuild
# Allowlist-validated: $1 must be one of VALID_RUNTIMES.
# Prevents path traversal and unexpected Docker tag injection.
# ─────────────────────────────────────────────────────
if [ -n "${1:-}" ]; then
  valid=0
  for v in "${VALID_RUNTIMES[@]}"; do
    [ "$1" = "$v" ] && valid=1 && break
  done
  if [ "${valid}" -eq 0 ]; then
    err "Unknown runtime '${1}'. Valid: ${VALID_RUNTIMES[*]}"
    exit 1
  fi
  RUNTIMES=("$1")
else
  RUNTIMES=("${VALID_RUNTIMES[@]}")
fi

# ─────────────────────────────────────────────────────
# Preflight checks
# ─────────────────────────────────────────────────────
if ! command -v docker >/dev/null 2>&1; then
  err "docker not found — run this on the host machine, not inside a workspace container"
  exit 1
fi

if [ ! -f "${HELPER_SCRIPT}" ]; then
  err "molecule-git-token-helper.sh not found at ${HELPER_SCRIPT}"
  err "Run: git pull origin main (PR #640 adds this file)"
  exit 1
fi

log "Building workspace-template:base from monorepo Dockerfile..."
docker build \
  --no-cache \
  -t workspace-template:base \
  -f "${SCRIPT_DIR}/Dockerfile" \
  "${SCRIPT_DIR}"
log "✓ workspace-template:base built"

# ─────────────────────────────────────────────────────
# Build each runtime adapter image
# ─────────────────────────────────────────────────────
TMPBASE=$(mktemp -d)
trap 'rm -rf "${TMPBASE}"' EXIT

SUCCESS=()
FAILED=()

for runtime in "${RUNTIMES[@]}"; do
  log "──────────────────────────────────────────"
  log "Building workspace-template:${runtime} ..."

  RUNTIME_DIR="${TMPBASE}/${runtime}"
  mkdir -p "${RUNTIME_DIR}"

  # Clone the standalone template repo
  REPO="Molecule-AI/molecule-ai-workspace-template-${runtime}"
  log "  Cloning ${REPO} ..."
  if ! git clone --depth 1 "https://github.com/${REPO}.git" "${RUNTIME_DIR}" 2>&1; then
    err "  Failed to clone ${REPO} — skipping ${runtime}"
    FAILED+=("${runtime}")
    continue
  fi

  # Verify a Dockerfile exists
  if [ ! -f "${RUNTIME_DIR}/Dockerfile" ]; then
    err "  No Dockerfile in ${REPO} — skipping ${runtime}"
    FAILED+=("${runtime}")
    continue
  fi

  # Copy the credential helper into the build context so the Dockerfile can COPY it.
  cp "${HELPER_SCRIPT}" "${RUNTIME_DIR}/molecule-git-token-helper.sh"

  # Patch the Dockerfile:
  #   1. COPY the helper script into the image at a predictable path
  #   2. git config --system registers it globally (applies to all users in the
  #      container, survives the root→agent gosu handoff)
  #   3. Re-declare ENTRYPOINT last (safe — molecule-runtime entrypoint is
  #      unchanged, just ensuring it's after our additions)
  #
  # We do NOT replace the ENTRYPOINT or CMD — molecule-runtime remains the
  # entry point. The git config --system baked into the image layer means
  # git will call the helper on every push/fetch without any startup script.
  cat >> "${RUNTIME_DIR}/Dockerfile" << 'PATCH'

# ─── git credential helper (issue #613 / PR #640) ───────────────────────────
# Bake the credential helper into the image so git always has a fresh
# GitHub App token. git config --system writes to /etc/gitconfig which is
# inherited by all users (root → agent gosu handoff). No startup script change
# needed — git invokes this helper automatically on push/fetch.
COPY molecule-git-token-helper.sh /usr/local/bin/molecule-git-credential-helper
RUN chmod +x /usr/local/bin/molecule-git-credential-helper && \
    git config --system credential.https://github.com.helper \
      '!molecule-git-credential-helper' && \
    echo "git credential helper registered (molecule-git-credential-helper)"
# ─────────────────────────────────────────────────────────────────────────────
PATCH

  # Build and tag
  # Capture docker's exit code via PIPESTATUS[0] before grep's exit code
  # overwrites $?. Without this, set -o pipefail causes grep's exit (0 = match
  # found, 1 = no match) to determine success — not docker's exit code.
  log "  Running docker build ..."
  docker build \
      --no-cache \
      -t "workspace-template:${runtime}" \
      "${RUNTIME_DIR}" 2>&1 | grep -E "^(Step|#|---|\[|✓|ERROR|error)"
  docker_exit=${PIPESTATUS[0]}
  if [ "${docker_exit}" -eq 0 ]; then
    log "  ✓ workspace-template:${runtime} built"
    SUCCESS+=("${runtime}")
  else
    err "  Build failed for ${runtime} (docker exit ${docker_exit})"
    FAILED+=("${runtime}")
  fi
done

# ─────────────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────────────
echo ""
log "══════════════════════════════════════════"
log "Rebuild complete"
log "══════════════════════════════════════════"
if [ "${#SUCCESS[@]}" -gt 0 ]; then
  log "✓ Succeeded: ${SUCCESS[*]}"
fi
if [ "${#FAILED[@]}" -gt 0 ]; then
  err "✗ Failed:    ${FAILED[*]}"
fi

echo ""
log "Verify images:"
docker images | grep "workspace-template" | sort

echo ""
log "To restart all running workspaces and pick up new images:"
log "  docker ps --filter name=molecule --format '{{.Names}}' | xargs -r docker rm -f"
log "  # Then restart workspaces via Canvas or API"

if [ "${#FAILED[@]}" -gt 0 ]; then
  exit 1
fi
