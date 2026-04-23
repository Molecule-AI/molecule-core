#!/bin/bash
# rollback-latest.sh — moves the :latest tag on ghcr.io/molecule-ai/platform
# (and the matching tenant image) back to a prior :staging-<sha> digest
# without rebuilding anything. Prod tenants auto-pull :latest every 5
# min, so this is the fast path when a canary-verified image turns out
# to have a runtime regression that canary didn't catch.
#
# Usage:
#   scripts/rollback-latest.sh <sha>
#   scripts/rollback-latest.sh 4c1d56e
#
# Prereqs:
#   - crane on $PATH (brew install crane OR download from
#     https://github.com/google/go-containerregistry/releases)
#   - GHCR token exported as GITHUB_TOKEN with write:packages scope
#
# What it does (per image — platform + tenant):
#   crane digest ghcr.io/…:<sha>         # verify the target sha exists
#   crane tag    ghcr.io/…:<sha> latest  # retag remotely, single API call
#   crane digest ghcr.io/…:latest        # confirm the move
#
# Exit codes: 0 = both retagged, 1 = tag missing / crane error, 2 = bad args.

set -euo pipefail

if [ "${1:-}" = "" ]; then
  echo "usage: $0 <staging-sha>" >&2
  echo "  e.g. $0 4c1d56e — retags :latest to :staging-4c1d56e" >&2
  exit 2
fi

TARGET_SHA="$1"
PLATFORM=ghcr.io/molecule-ai/platform
TENANT=ghcr.io/molecule-ai/platform-tenant

if ! command -v crane >/dev/null; then
  echo "ERROR: crane not installed. brew install crane" >&2
  exit 1
fi
if [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "ERROR: GITHUB_TOKEN unset. export it with write:packages scope." >&2
  exit 1
fi

# Log in once. crane stores creds in a config file keyed by registry;
# re-running is cheap.
printf '%s\n' "$GITHUB_TOKEN" | crane auth login ghcr.io -u "${GITHUB_ACTOR:-$(whoami)}" --password-stdin >/dev/null

roll() {
  local image="$1"
  local src="$image:staging-$TARGET_SHA"
  local dst="$image:latest"

  echo "→ $image"
  # Abort rollout if the target tag doesn't exist in the registry.
  # Otherwise crane tag would error anyway, but a pre-check gives a
  # clearer message for ops.
  if ! crane digest "$src" >/dev/null 2>&1; then
    echo "  FAIL: $src not found in registry. Did you type the wrong sha?" >&2
    return 1
  fi
  local src_digest=$(crane digest "$src")

  crane tag "$src" latest
  local new_digest=$(crane digest "$dst")

  if [ "$new_digest" != "$src_digest" ]; then
    echo "  FAIL: $dst digest $new_digest does not match expected $src_digest" >&2
    return 1
  fi
  echo "  OK   $dst → $new_digest"
}

roll "$PLATFORM"
roll "$TENANT"

echo
echo "=== ROLLBACK COMPLETE ==="
echo "Both images now point :latest at staging-$TARGET_SHA."
echo "Prod tenants will pick up the rollback within their 5-min auto-update cycle."
