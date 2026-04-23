# Screencast narration — Cloudflare Artifacts demo
# TTS generation: pending (TTS tool not available in this workspace)
# Run this script through any TTS service (ElevenLabs, Azure TTS, AWS Polly, etc.)

NARRATION_SCRIPT="""
[0-10s]
Every Molecule AI workspace can now have its own Git repository on Cloudflare's edge — no SSH keys, no credential rotation, no self-hosted server. The agent works. The commits get written. The history stays auditable.

[10-25s]
Step one — attach a repo to a workspace. One API call. The platform talks to Cloudflare's Artifacts API, creates the repository, and returns a git remote URL. The credential is scoped to that workspace alone.

[25-40s]
Step two — mint a short-lived credential. One API call, one hour TTL by default. The agent clones the repo, writes its work as a Git commit — a snapshot of the agent's state at this moment — and pushes. Every agent run is versioned.

[40-50s]
Step three — before a risky change, the agent forks. One more API call, an isolated branch, no pollution of the main history. The main branch stays clean.

[50-60s]
All of this is visible from Canvas. No terminal required. Your team sees the commit history, the active branches, and the artifact repository — right alongside your agent. Cloudflare stores the git data on the edge, close to the platform. Short-lived credentials mean no long-lived secrets to manage. This is what a Git workflow for AI agents looks like when it's designed for agents, not humans.
"""

# Save to file for use with external TTS
echo "$NARRATION_SCRIPT" > /workspace/repos/molecule-core/docs/marketing/devrel/cloudflare-artifacts-narration.txt
echo "Narration script saved. Pipe into your TTS service to generate audio."