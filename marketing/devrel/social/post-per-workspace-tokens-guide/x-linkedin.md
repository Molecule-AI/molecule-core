# Social Copy — Post: Per-Workspace Bearer Tokens Explainer
> Guide: `docs/blog/2026-04-21-per-workspace-tokens/index.md` | Slug: `per-workspace-bearer-tokens`
> Platform: X (Twitter) thread + LinkedIn post | Status: READY

---

## X Thread — 5 posts

### Post 1 — Hook
> Your agent has its own credential.
> One 256-bit bearer token, scoped to one workspace. Revoke it, the agent shuts down cleanly.
> Here's why the scope matters.

### Post 2 — What workspace-scoped actually means
> A workspace token can reach:
> — The workspace's own routes
> — The workspace's secrets
> — The workspace's A2A dispatch endpoints
> It cannot reach other workspaces in the org.
> It cannot reach org-level routes (billing, org key management).

### Post 3 — The blast radius difference
> Shared org token compromised: every workspace in the org is exposed simultaneously.
> Per-workspace token compromised: one workspace. The others keep running.
> This is the same least-privilege principle from human IAM — applied to machine identities.

### Post 4 — Secrets never travel over the wire
> When the agent registers, the registration payload contains NO credentials.
> The agent authenticates, then pulls its secrets from /workspaces/:id/secrets/values.
> Rotate a key in the platform UI → every agent picks it up on next boot.
> No agent config files to touch.

### Post 5 — Revocation is clean
> Revoke the token → next API call returns 401.
> Agent detects 401 → finishes any in-flight task acknowledgment → exits.
> Max detection window: one polling cycle (~30-45 seconds).
> No zombie agents. No stale sessions.

---

## LinkedIn Post

**Every agent has its own credential. Why does that matter?**

When a remote agent registers with a Molecule AI org, it receives a workspace-scoped bearer token — a 256-bit cryptographic credential scoped to exactly one workspace. Not the org. Not every workspace. One workspace.

This is the same least-privilege principle that underpins human IAM best practices — applied to machine identities. The question worth asking: why does the blast radius of a compromised agent credential matter?

Because agents run unattended. The token lives in a config file or a secrets manager. If it's exfiltrated, the window of exposure depends on two things: how quickly you can revoke it, and how much damage it can do while it's live.

A workspace-scoped token limits the blast radius. If the researcher's token is compromised, the PM's workspace is unaffected. The agent detects the 401 on its next heartbeat (max 45 seconds), finishes its in-flight task acknowledgment, and exits cleanly. There's no zombie agent sitting with a live credential.

The secrets pull flow is the other design worth understanding: credentials never travel over the registration channel. The agent authenticates with its bearer token, then pulls its API keys from the platform over an authenticated connection. Rotate `OPENAI_API_KEY` in the platform UI, and every agent that uses it picks up the new key on its next boot. No config files to touch, no agent redeployments.

This is the model that makes fleet-wide secret rotation practical — and the model that makes per-agent revocation surgical rather than a blunt instrument.

→ [Per-Workspace Bearer Tokens explainer](https://docs.molecule.ai/blog/per-workspace-bearer-tokens)

---

## Platform Notes
- X thread: thread start at 9am PST
- LinkedIn: post same day at 10am PST
- Pairs well with the Fleet Visibility guide for a "security + operations" content pairing
- Good SEO anchor for: "AI agent authentication", "bearer token security", "agent credential management"
- CTA links: update before posting