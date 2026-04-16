#!/usr/bin/env bash
# clone-manifest.sh — clone all repos listed in manifest.json into their
# target directories. Replaces 33 hardcoded git-clone lines in Dockerfiles.
#
# Usage:
#   ./scripts/clone-manifest.sh <manifest.json> <ws-templates-dir> <org-templates-dir> <plugins-dir>
#
# Example (Docker build stage):
#   /scripts/clone-manifest.sh /manifest.json /workspace-configs-templates /org-templates /plugins

set -euo pipefail

MANIFEST="${1:?Usage: clone-manifest.sh <manifest.json> <ws-dir> <org-dir> <plugins-dir>}"
WS_DIR="${2:?Missing workspace-templates dir}"
ORG_DIR="${3:?Missing org-templates dir}"
PLUGINS_DIR="${4:?Missing plugins dir}"

clone_category() {
    local category="$1"
    local target_dir="$2"

    mkdir -p "$target_dir"

    # Use python3 to parse JSON (jq may not be available in Docker)
    python3 -c "
import json, sys
with open('$MANIFEST') as f:
    m = json.load(f)
for entry in m.get('$category', []):
    print(entry['name'], entry['repo'], entry.get('ref', 'main'))
" | while read -r name repo ref; do
        echo "  cloning $repo -> $target_dir/$name (ref=$ref)"
        if [ "$ref" = "main" ]; then
            git clone --depth=1 -q "https://github.com/${repo}.git" "$target_dir/$name"
        else
            git clone --depth=1 -q --branch "$ref" "https://github.com/${repo}.git" "$target_dir/$name"
        fi
    done

    # Strip .git dirs to save space
    find "$target_dir" -name '.git' -type d -exec rm -rf {} + 2>/dev/null || true
}

echo "==> Cloning workspace templates..."
clone_category "workspace_templates" "$WS_DIR"

echo "==> Cloning org templates..."
clone_category "org_templates" "$ORG_DIR"

echo "==> Cloning plugins..."
clone_category "plugins" "$PLUGINS_DIR"

echo "==> Done. $(find "$WS_DIR" "$ORG_DIR" "$PLUGINS_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ') repos cloned."
