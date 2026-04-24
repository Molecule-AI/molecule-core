# Cloudflare Artifacts — Social Copy
Campaign: cloudflare-artifacts-launch | Blog: `docs/blog/2026-04-21-cloudflare-artifacts/index.md`
Publish day: 2026-04-21 (blog live on staging)
Status: Copy ready — pending Marketing Lead approval

---

## X (Twitter) — Primary thread (5 posts)

### Post 1 — Hook (keyword: `AI agent version control`)
Your AI agent just finished a 3-hour refactor.

What changed? Do you know?

Agents that don't maintain version history leave you reviewing final outputs blind. Cloudflare Artifacts makes the agent's working state a first-class Git repository — with branches, commits, and rollbacks built in.

→ https://docs.molecule.ai/blog/cloudflare-artifacts

---

### Post 2 — Git-native (keyword: `Git for AI agents`)
Cloudflare Artifacts speaks Git natively.

Agents use standard `git clone` and `git fetch`. Any Git client works. No new tooling, no custom client library, no proprietary format.

Molecule AI connects every workspace to an Artifacts repo — the platform handles the credential lifecycle so you don't have to.

→ https://docs.molecule.ai/blog/cloudflare-artifacts

---

### Post 3 — Credential model (keyword: `short-lived Git credentials`)
The credential model is the part that makes this production-safe:

→ Tokens scoped to a single repo
→ Short-lived by default (1 hour, capped at 7 days)
→ Revocable without touching the repo

No long-lived credentials sitting in environment variables. No credential sprawl across your agent fleet.

→ https://docs.molecule.ai/blog/cloudflare-artifacts

---

### Post 4 — Use cases (keyword: `AI agent audit trail`)
Real things you can do once your agent's working state is versioned:

• Review every commit the agent made — not just the final output
• Fork a workspace before a risky operation, roll back if it goes wrong
• Roll a session back to a known-good state if an agent corrupts a config
• Build an auditable record of agent decisions without any custom instrumentation

→ https://docs.molecule.ai/blog/cloudflare-artifacts

---

### Post 5 — CTA
Cloudflare Artifacts integration ships today with Molecule AI.

Your agents don't just produce outputs — they maintain a versioned working state you can inspect, branch, and roll back.

Configure a workspace → your full version history.

→ https://docs.molecule.ai/blog/cloudflare-artifacts

#GitForAgents #AIAgents #AgenticAI #MoleculeAI #Cloudflare

---

## LinkedIn — Single post

**Title:** We gave our AI agents a Git history. Here's what that changes.

**Body:**

Most AI agent platforms treat the workspace as a black box — you get a final output, the session ends, and the working state is gone.

We just shipped a Cloudflare Artifacts integration for Molecule AI. It changes the model.

Here's what it means in practice:

**Agents work in a Git repository.** Every workspace is linked to a Cloudflare Artifacts repo. The agent uses standard `git clone` and `git push` — no proprietary client, no custom tooling. Any Git client works.

**Short-lived, repo-scoped credentials.** Tokens are scoped to a single repo, short-lived by default, and revocable immediately. No long-lived credentials in environment variables. The platform handles the credential lifecycle.

**You can review the work, not just the output.** Instead of reading a final result, you `git log` the workspace and see every commit the agent made — with the context of what the agent was working on when it ran.

**Fork before you let it run.** Fork a workspace's repo before a risky operation. If the outcome is bad, you have the original state. If it's good, you merge.

**Roll back a session.** Because Artifacts is fully versioned, a workspace's state at any previous commit is accessible via Git. An agent that corrupts a config file can have its workspace reset to a known-good state by pointing it at an earlier commit.

**Security: API keys don't enter the Git history.** Before any working state is committed, `snapshot_scrub.py` runs over it. API keys, bearer tokens, and sandbox tool output are redacted or excluded before the snapshot enters the git history.

The integration is live now for all Molecule AI workspaces with Cloudflare Artifacts access.

→ https://docs.molecule.ai/blog/cloudflare-artifacts

#AIAgents #AgenticAI #MoleculeAI #Cloudflare #GitOps

---

## Campaign notes

**Audience:** Developer / DevOps (X), Platform engineers (LinkedIn)
**Tone:** Practical, production-first. Lead with the version control story, not the feature list.
**Differentiation:** Git-native — agents use standard Git, not a proprietary format. Short-lived credentials. Snapshot scrubbing for security.
**Use case pairing:** X → version history + review workflow (developer angle), LinkedIn → operational safety + audit trail (platform engineering angle)
**Assets:** Screencast videos at `docs/marketing/devrel/demos/cloudflare-artifacts/` (16:9 + 1:1 + captioned variants)
**Coordination:** This blog is live — social can go out same day. Coordinate with Social Media Brand queue.
**Hashtags:** #GitForAgents #AIAgents #AgenticAI #MoleculeAI #Cloudflare