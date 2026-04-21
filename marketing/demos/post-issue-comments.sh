#!/usr/bin/env bash
# post-issue-comments.sh
# Run once GH_TOKEN is refreshed. Posts completion comments to #1172 and #1173.
# Usage: bash post-issue-comments.sh

set -e

TOKEN="${GH_TOKEN:-${GITHUB_TOKEN}}"
if [[ -z "$TOKEN" ]]; then
  echo "ERROR: GH_TOKEN or GITHUB_TOKEN not set" >&2
  exit 1
fi

BASE="https://api.github.com/repos/Molecule-AI/internal"

comment_1172() {
  curl -s -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    -H "Content-Type: application/json" \
    "$BASE/issues/1172/comments" \
    -d @comment-1172.json
}

comment_1173() {
  curl -s -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    -H "Content-Type: application/json" \
    "$BASE/issues/1173/comments" \
    -d @comment-1173.json
}

echo "Posting comment to #1172..."
comment_1172

echo "Posting comment to #1173..."
comment_1173

echo "Done."
