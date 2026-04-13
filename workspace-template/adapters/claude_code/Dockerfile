FROM workspace-template:base
USER root
RUN npm install -g @anthropic-ai/claude-code 2>/dev/null || true
# Pre-install adapter Python deps (single source of truth: requirements.txt)
# so first-boot doesn't pay a pip install penalty. entrypoint.sh skips
# reinstall when the first listed package is already importable.
RUN pip install --no-cache-dir -r /app/adapters/claude_code/requirements.txt
# Do NOT set USER agent — entrypoint starts as root, chowns volumes, drops to agent via gosu
