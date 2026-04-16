FROM workspace-template:base
USER root
RUN npm install -g @google/gemini-cli 2>/dev/null || true
# gemini-cli has no extra Python adapter deps — uses CLIAgentExecutor from base
# Do NOT set USER agent — entrypoint starts as root, chowns volumes, drops to agent via gosu
