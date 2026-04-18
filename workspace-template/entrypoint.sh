#!/bin/sh
# Drop privileges to the agent user before exec'ing molecule-runtime.
# claude-code refuses --dangerously-skip-permissions when running as
# root/sudo for safety. Without this entrypoint, every cron tick fails
# with `ProcessError: Command failed with exit code 1` and the agent
# logs `--dangerously-skip-permissions cannot be used with root/sudo
# privileges for security reasons`.
#
# Pattern matches the legacy monorepo workspace-template/entrypoint.sh:
# fix volume ownership as root, then re-exec via gosu as agent (uid 1000).

if [ "$(id -u)" = "0" ]; then
    # Configs volume is created by Docker as root; agent needs write access
    # for plugin installs, memory writes, .auth_token rotation, etc.
    chown -R agent:agent /configs 2>/dev/null
    # Strip CRLF from hook scripts — Windows Docker Desktop copies host files
    # with CRLF line endings even when .gitattributes says eol=lf. The \r in
    # the shebang line makes python3 try to open 'script.py\r' → ENOENT →
    # claude-code swallows the hook error → "(no response generated)".
    # This is the permanent fix — runs at every container start.
    for f in /configs/.claude/hooks/*.sh /configs/.claude/hooks/*.py; do
        [ -f "$f" ] && sed -i 's/\r$//' "$f"
    done
    # /workspace handling — only chown when the contents are root-owned
    # (typical on Docker Desktop on Windows where host uid maps to 0).
    # On Linux Docker with matching uids the recursive chown is skipped
    # to keep startup fast.
    chown agent:agent /workspace 2>/dev/null || true
    if [ -d /workspace ]; then
        first_entry=$(find /workspace -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)
        if [ -n "$first_entry" ] && [ "$(stat -c '%u' "$first_entry" 2>/dev/null)" = "0" ]; then
            chown -R agent:agent /workspace 2>/dev/null
        fi
    fi
    # Claude Code session directory — mounted at /root/.claude/sessions by
    # the platform provisioner. Symlink it into agent's home so the SDK
    # finds it when running as agent. The provisioner's mount point is
    # hardcoded to /root/.claude/sessions; we don't want to change the
    # platform contract just for this template.
    mkdir -p /home/agent/.claude
    if [ -d /root/.claude/sessions ]; then
        chown -R agent:agent /root/.claude /home/agent/.claude 2>/dev/null
        ln -sfn /root/.claude/sessions /home/agent/.claude/sessions
    fi
    exec gosu agent "$0" "$@"
fi

# Now running as agent (uid 1000)
exec molecule-runtime "$@"
