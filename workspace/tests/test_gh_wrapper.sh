#!/usr/bin/env bash
# Smoke-test the gh-wrapper behaviour with a fake gh binary that echoes
# back its argv. Runs entirely in-process (no Docker), so it's cheap to
# run per-CI-job. Tests the behaviour table in scripts/gh-wrapper.sh.
#
# Invoked by CI's Python Lint & Test job via a subprocess shell-out, or
# locally via `bash tests/test_gh_wrapper.sh`.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WRAPPER="$HERE/../scripts/gh-wrapper.sh"

if [[ ! -x "$WRAPPER" ]]; then
    echo "FAIL: wrapper not executable: $WRAPPER" >&2
    exit 1
fi

# Fake gh: prints every arg on its own line, prefixed by "ARG:". Lets
# tests introspect what the wrapper passed through.
FAKE_GH_DIR=$(mktemp -d)
trap 'rm -rf "$FAKE_GH_DIR"' EXIT
cat > "$FAKE_GH_DIR/gh" <<'EOF'
#!/usr/bin/env bash
for a in "$@"; do
    printf 'ARG:%s\n' "$a"
done
EOF
chmod +x "$FAKE_GH_DIR/gh"

# Make the wrapper use the fake gh by overriding the hardcoded path via
# a temporary symlink trick: copy the wrapper to a temp location and
# sed-replace the REAL_GH default with our fake.
WRAPPER_UNDER_TEST=$(mktemp)
trap 'rm -f "$WRAPPER_UNDER_TEST"' EXIT
sed "s|REAL_GH=/usr/bin/gh|REAL_GH=$FAKE_GH_DIR/gh|" "$WRAPPER" > "$WRAPPER_UNDER_TEST"
chmod +x "$WRAPPER_UNDER_TEST"

pass=0
fail=0

assert_contains() {
    local name="$1" haystack="$2" needle="$3"
    if [[ "$haystack" == *"$needle"* ]]; then
        pass=$((pass + 1))
        echo "  PASS: $name"
    else
        fail=$((fail + 1))
        echo "  FAIL: $name" >&2
        echo "    expected to contain: $needle" >&2
        echo "    got: $haystack" >&2
    fi
}

assert_not_contains() {
    local name="$1" haystack="$2" needle="$3"
    if [[ "$haystack" == *"$needle"* ]]; then
        fail=$((fail + 1))
        echo "  FAIL: $name — should not contain: $needle" >&2
        echo "    got: $haystack" >&2
    else
        pass=$((pass + 1))
        echo "  PASS: $name"
    fi
}

echo "--- passthrough (no subcommand transform) ---"
out=$(GIT_AUTHOR_NAME="Molecule AI Frontend Engineer" "$WRAPPER_UNDER_TEST" pr list --state open)
assert_contains "pr list passthrough" "$out" "ARG:list"
assert_not_contains "pr list no prefix" "$out" "[Frontend"

echo "--- pr create with role ---"
out=$(GIT_AUTHOR_NAME="Molecule AI Backend Engineer" "$WRAPPER_UNDER_TEST" pr create --title "fix: auth" --body "Short description")
assert_contains "pr create title prefix" "$out" "ARG:[Backend Engineer] fix: auth"
assert_contains "pr create body footer" "$out" "_Opened by: Molecule AI Backend Engineer_"

echo "--- issue create with = form ---"
out=$(GIT_AUTHOR_NAME="Molecule AI PM" "$WRAPPER_UNDER_TEST" issue create --title="bug: foo" --body="details")
assert_contains "issue create --title= prefix" "$out" "ARG:--title=[PM] bug: foo"
assert_contains "issue create --body= footer" "$out" "_Opened by: Molecule AI PM_"

echo "--- idempotent title re-prefix ---"
out=$(GIT_AUTHOR_NAME="Molecule AI DevRel Engineer" "$WRAPPER_UNDER_TEST" pr create --title "[DevRel Engineer] already prefixed")
assert_not_contains "no double prefix" "$out" "[DevRel Engineer] [DevRel Engineer]"

echo "--- idempotent body footer ---"
already="original body

---
_Opened by: Molecule AI UIUX Designer_"
out=$(GIT_AUTHOR_NAME="Molecule AI UIUX Designer" "$WRAPPER_UNDER_TEST" pr create --title "x" --body "$already")
# Count how many times the footer marker appears — should be exactly 1.
count=$(echo "$out" | grep -c "_Opened by: Molecule AI UIUX Designer_" || true)
if [[ "$count" -eq 1 ]]; then
    pass=$((pass + 1)); echo "  PASS: footer not double-appended"
else
    fail=$((fail + 1)); echo "  FAIL: footer count=$count (want 1)" >&2
fi

echo "--- missing GIT_AUTHOR_NAME — passes through ---"
out=$(unset GIT_AUTHOR_NAME; "$WRAPPER_UNDER_TEST" pr create --title "fix: foo")
assert_not_contains "no role means no prefix" "$out" "[M"
assert_contains "raw title survives" "$out" "ARG:fix: foo"

echo "--- wrong prefix in GIT_AUTHOR_NAME — passes through ---"
out=$(GIT_AUTHOR_NAME="Some Random Human" "$WRAPPER_UNDER_TEST" pr create --title "fix: foo")
assert_not_contains "non-Molecule author means no prefix" "$out" "[S"
assert_contains "raw title survives (wrong prefix)" "$out" "ARG:fix: foo"

echo
echo "================================"
echo "gh-wrapper: $pass passed, $fail failed"
echo "================================"
[[ $fail -eq 0 ]]
