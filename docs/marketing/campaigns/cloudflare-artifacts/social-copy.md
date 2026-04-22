# Social Copy — Cloudflare Artifacts + Molecule AI Campaign
## Blog Post: "Give Your AI Agent a Git Repository: Molecule AI + Cloudflare Artifacts"
**URL:** /blog/cloudflare-artifacts-molecule-ai (pending publish)
**Date:** 2026-04-21
**Author:** Content Marketer
**Status:** ✅ APPROVED by Marketing Lead (PMM ruling: soften "sub-100ms" claim — see Post 2)

---

## X / Twitter Thread

**Post 1 (Hook):**
> AI agents write code, generate configs, and produce assets.
Most of the time, those outputs evaporate when the session ends.

We just gave every Molecule AI workspace a git repository.

Git-native. Versioned by default. Agents push, pull, and branch — the same workflow your team already knows.

---

**Post 2 (What it is):**
> Cloudflare Artifacts is git-native object storage.

Git pull and git push semantics. Fast edge-based clone times from anywhere on Cloudflare's global network. No S3 bandwidth bills.

Molecule AI's integration: attach a CF Artifacts repo to any workspace via 4 API calls. Agents clone, commit, push — and their work survives the session.

```
POST /workspaces/:id/artifacts   → attach a repo
POST /workspaces/:id/artifacts/fork  → experiment safely
POST /workspaces/:id/artifacts/token → short-lived git cred
```

---

**Post 3 (The security angle):**
> Two things we got right in the integration:

1. SSRF protection — import URLs must use https://. git:// and http:// are rejected at the router.
2. Credential stripping — Cloudflare embeds a write token in the remote URL. We strip it before it touches the DB. Agents fetch fresh short-lived creds via the API on demand.

No long-lived tokens. No credential sprawl. Secure by default.

---

**Post 4 (Use cases):**
> What can you actually build with a git-native workspace?

→ A research agent that maintains its own annotated notes repo — survives every session
→ A code-review agent that forks a repo, tests changes, and opens a PR
→ A shared asset library for a multi-agent team — versioned, collaborative, git-native

All of these are now one API call.

---

**Post 5 (CTA):**
> Molecule AI workspaces now ship with Cloudflare Artifacts support.

Set two env vars, create a repo via the API, and your agent has a git URL.

GitHub: [molecule-core/workspace-server/internal/handlers/artifacts.go](https://github.com/Molecule-AI/molecule-core/blob/main/workspace-server/internal/handlers/artifacts.go)

→ [Read the full post: "Give Your AI Agent a Git Repository"](https://docs.molecule.ai/blog/cloudflare-artifacts)

---

## LinkedIn Post

**Single post:**

We've shipped Cloudflare Artifacts support for Molecule AI workspaces — and it's one of the more architecturally clean integrations we've done.

The problem: AI agent outputs are mostly transient. Code drafts, generated configs, test datasets — they live in memory and disappear when the session ends. Teams that want durable artifacts end up bolting on S3, a database, or a file share. All introduce a new API surface, new auth scheme, new workflow.

Git-native storage is different. Cloudflare Artifacts speaks git — pull, push, branch, fork. Agents already know it. Your team already knows it. And Cloudflare's global edge network means low-latency access wherever your agents run.

The Molecule AI integration exposes four API endpoints:
- Attach a CF Artifacts repo to any workspace
- Fork it for safe experimentation
- Mint short-lived git credentials on demand
- Import an existing GitHub/GitLab repo

Security properties built in: SSRF protection on import URLs, credential stripping before DB storage, no long-lived tokens.

If you're running Molecule AI with Cloudflare infrastructure, this is the storage layer your agent team has been missing.

Full implementation: [artifacts.go on GitHub](https://github.com/Molecule-AI/molecule-core/blob/main/workspace-server/internal/handlers/artifacts.go)

→ [Read: "Give Your AI Agent a Git Repository"](https://docs.molecule.ai/blog/cloudflare-artifacts)

#Cloudflare #AIagents #Git #DeveloperTools #CloudComputing

---

## Image / Visual Recommendations

| Platform | Asset | Description |
|---|---|---|
| X/LinkedIn | Architecture card | Workspace → Artifacts API → CF Artifacts → git remote URL. Clean labeled boxes. |
| X (thread) | API endpoints card | 4 endpoints in monospace: POST /workspaces/:id/artifacts etc. Dark background. |
| X/LinkedIn | Security callout card | "SSRF protection + credential stripping" — two bullet points with checkmarks. |
| CTA graphic | "Your AI agent just got a git repo." + GitHub link | |

---

## Publishing Schedule

| Platform | When | Notes |
|---|---|---|
| X thread | Day of publish, 9am PT | 5 posts, staggered 20-30 min |
| LinkedIn | Day of publish, 11am PT | Same day as X |
| Reddit r/LocalLlama | Day of publish, 12pm PT | After X thread is live |

---

*Draft by Content Marketer 2026-04-21*
