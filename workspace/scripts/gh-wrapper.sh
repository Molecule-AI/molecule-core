#!/usr/bin/env bash
# gh wrapper — auto-prefixes PR + issue titles with the agent role and
# appends an "Opened by: Molecule AI <Role>" footer to bodies. Shadows
# the real `gh` binary (installed at /usr/bin/gh) because /usr/local/bin
# is earlier in PATH in the workspace image.
#
# Why: every agent in the molecule-dev template shares one GitHub token
# (the CEO's PAT), so `gh pr list` shows every PR as authored by the
# same human user. This wrapper preserves the real gh behaviour while
# injecting the agent's identity into the PR/issue metadata so the
# list + body reveal WHICH agent opened each item. Commit authors are
# already per-agent via GIT_AUTHOR_NAME (shipped in the provisioner);
# this handles the PR/issue surface layer the commit layer can't reach.
#
# Role is derived from GIT_AUTHOR_NAME which the platform sets to
# "Molecule AI <Role Name>" at container provision time. If GIT_AUTHOR_NAME
# is missing or doesn't follow the expected prefix, the wrapper passes
# through unmodified — fail-open so no call is ever BLOCKED by this
# script.
#
# Behaviour table:
#
#   gh pr create --title "fix: foo" ...
#     → title becomes "[Frontend Engineer] fix: foo"
#     → body gets "\n\n---\n_Opened by: Molecule AI Frontend Engineer_\n" appended
#
#   gh issue create --title "..." ...
#     → same title + body transforms
#
#   gh <anything else>
#     → passes through untouched
#
# Idempotence: if the title already starts with "[" + any characters + "]",
# the wrapper does NOT re-prefix. Rerunning `gh pr edit` won't layer
# multiple "[Role] [Role] ..." prefixes. Same for body footer — we check
# for the exact "Opened by: Molecule AI" marker and skip if present.

set -euo pipefail

REAL_GH=/usr/bin/gh
if [[ ! -x "$REAL_GH" ]]; then
    # Fallback: find the real gh wherever it landed.
    REAL_GH=$(command -v /usr/bin/gh /opt/gh/bin/gh /usr/local/bin/gh-original 2>/dev/null | head -1)
    if [[ -z "$REAL_GH" ]]; then
        echo "gh-wrapper: real gh binary not found" >&2
        exit 127
    fi
fi

# Extract the agent role from GIT_AUTHOR_NAME ("Molecule AI <Role>").
# If missing or malformed, skip all transforms.
role=""
if [[ -n "${GIT_AUTHOR_NAME:-}" && "${GIT_AUTHOR_NAME}" == "Molecule AI "* ]]; then
    role="${GIT_AUTHOR_NAME#Molecule AI }"
fi

# Subcommand must be pr or issue, followed by `create`, to trigger the
# transform. Everything else is a passthrough.
if [[ $# -lt 2 || ( "$1" != "pr" && "$1" != "issue" ) || "$2" != "create" ]]; then
    exec "$REAL_GH" "$@"
fi

if [[ -z "$role" ]]; then
    # No role detected — behave exactly like real gh. Don't eat arguments
    # trying to be clever.
    exec "$REAL_GH" "$@"
fi

# Walk the args, rewriting --title / --body in place. Preserve every
# other flag untouched. Accept both "--title X" and "--title=X" forms.
new_args=()
i=1
while (( i <= $# )); do
    arg="${!i}"
    case "$arg" in
        --title)
            next_i=$((i + 1))
            val="${!next_i:-}"
            if [[ "$val" == \[*\]* ]]; then
                # Already prefixed — leave alone.
                new_args+=("$arg" "$val")
            else
                new_args+=("$arg" "[$role] $val")
            fi
            i=$((i + 2))
            continue
            ;;
        --title=*)
            val="${arg#--title=}"
            if [[ "$val" == \[*\]* ]]; then
                new_args+=("$arg")
            else
                new_args+=("--title=[$role] $val")
            fi
            i=$((i + 1))
            continue
            ;;
        --body)
            next_i=$((i + 1))
            val="${!next_i:-}"
            if [[ "$val" == *"Opened by: Molecule AI"* ]]; then
                new_args+=("$arg" "$val")
            else
                new_args+=("$arg" "${val}

---
_Opened by: Molecule AI ${role}_")
            fi
            i=$((i + 2))
            continue
            ;;
        --body=*)
            val="${arg#--body=}"
            if [[ "$val" == *"Opened by: Molecule AI"* ]]; then
                new_args+=("$arg")
            else
                new_args+=("--body=${val}

---
_Opened by: Molecule AI ${role}_")
            fi
            i=$((i + 1))
            continue
            ;;
        # Identity translation (#1957). All agents share one PAT, so
        # `gh ... --assignee @me` resolves to the CEO and lands every
        # agent-filed issue/PR on the human's plate. Translate to a
        # role-tagged label instead — labels are the right abstraction
        # for "this team owns it" in a multi-agent fleet.
        #
        # Reviewer requests are dropped: the review-bot scans by label,
        # not by direct request, so --reviewer @me is just noise.
        --assignee)
            next_i=$((i + 1))
            val="${!next_i:-}"
            if [[ "$val" == "@me" ]]; then
                # Translate: drop --assignee, add --label team:<role-slug>
                slug=$(echo "$role" | tr '[:upper:] ' '[:lower:]-')
                new_args+=(--label "team:${slug}")
            else
                new_args+=("$arg" "$val")
            fi
            i=$((i + 2))
            continue
            ;;
        --assignee=@me)
            slug=$(echo "$role" | tr '[:upper:] ' '[:lower:]-')
            new_args+=(--label "team:${slug}")
            i=$((i + 1))
            continue
            ;;
        --reviewer)
            next_i=$((i + 1))
            val="${!next_i:-}"
            if [[ "$val" == "@me" ]]; then
                # Drop entirely — review-bot picks up via label scan
                : # no-op
            else
                new_args+=("$arg" "$val")
            fi
            i=$((i + 2))
            continue
            ;;
        --reviewer=@me)
            # Drop entirely
            i=$((i + 1))
            continue
            ;;
        *)
            new_args+=("$arg")
            i=$((i + 1))
            ;;
    esac
done

exec "$REAL_GH" "${new_args[@]}"
