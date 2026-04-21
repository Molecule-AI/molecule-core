---
title: "Per-Workspace Bearer Tokens — Why Your Agent Authenticates to One Workspace"
date: 2026-04-21
slug: per-workspace-bearer-tokens
description: "A 256-bit bearer token scoped to one workspace. What that actually means, why it's different from a shared admin token, and what the revocation boundary looks like in practice."
tags: [security, auth, platform, tokens, remote-agents]
---

# Per-Workspace Bearer Tokens — Why Your Agent Authenticates to One Workspace

When a remote agent registers with a Molecule AI org, it receives a workspace-scoped bearer token — a 256-bit cryptographic credential that grants the agent access to exactly one workspace. Not the org. Not every workspace. One workspace.

This post explains what that means in practice, why it matters for security, and what the revocation boundary looks like.

---

## What a workspace-scoped token actually is

A bearer token is a secret string. Anyone who holds it can make authenticated requests on behalf of whoever it was issued to. The platform validates the token on every request — there's no session, no cookies, no refresh cycle.

The workspace scope means this token is bound to one workspace record in the database. When the token is presented, the platform looks up the workspace, verifies the token hash matches, and executes the request in the context of that workspace's org and permissions.

```bash
# This token can only act within ws-abc123
curl https://acme.moleculesai.app/workspaces/ws-abc123/state \
  -H "Authorization: Bearer mka_ab_c7xXz..."
# → 200 OK with workspace state

# Same token, different workspace
curl https://acme.moleculesai.app/workspaces/ws-other/state \
  -H "Authorization: Bearer mka_ab_c7xXz..."
# → 403 Forbidden
```

The token can reach the workspace's own routes, secrets, and A2A dispatch endpoints. It cannot reach other workspaces in the org. Org-level routes (like billing or org-wide key management) require an org-level token.

---

## Why this matters for agent security

Agent bearer tokens are a different trust model from human credentials. An agent runs unattended — the token lives in a config file, an environment variable, or a secrets manager. If that token is exfiltrated, the window of exposure depends on two things: how quickly you can revoke it, and how much damage it can do while it's live.

A workspace-scoped token limits the blast radius to one workspace. If the researcher's token is compromised, the PM's workspace is unaffected. If an org-level token is compromised, every workspace in the org is exposed simultaneously.

This is the same principle behind least-privilege access in human IAM — but applied to machine identities, where the revocation speed is often slower because the credential may be buried in a config file that isn't monitored.

---

## The secrets pull flow

Agents don't just use the bearer token for API calls — they also use it to pull their own secrets from the platform at boot. This is the `GET /workspaces/:id/secrets/values` endpoint:

```python
secrets = client.pull_secrets()
# Returns: {"OPENAI_API_KEY": "sk-...", "MODEL_NAME": "gpt-4o"}
# The token was used to authenticate the pull.
# The secrets never appear in the agent's registration payload.
```

The agent's API keys never travel over the registration channel. They stay in the platform's secrets manager and are retrieved over an authenticated connection. This means:

1. The registration payload (sent over the network once) contains no credentials
2. The secrets are retrieved on an authenticated channel every time the agent boots
3. If the token is revoked, the next pull attempt returns 401 — the agent knows immediately

The second point means secrets can be rotated without redeploying the agent. If you rotate `OPENAI_API_KEY` in the platform UI, every agent that pulls from `/secrets/values` picks up the new key on its next boot. No agent config files to touch.

---

## Revocation in practice

When you revoke a workspace token from the Canvas Settings panel:

1. The platform deletes the token record from its database
2. Any subsequent API call with that token returns `401 Unauthorized`
3. The agent's next heartbeat or secrets pull fails with `401`
4. The agent detects the failure and enters its shutdown sequence

The shutdown is clean — the agent finishes any in-flight task dispatch acknowledgment, then exits. It doesn't leave partial state in the platform.

The detection window is the polling interval: 30 seconds for state polling, 45 seconds for heartbeat. After at most one polling cycle, the agent is offline from the platform's perspective.

```python
# What the agent does when it gets a 401
def run_heartbeat_loop(self, task_supplier):
    while True:
        try:
            self.platform.post_state(task_supplier())
        except UnauthorizedError:
            logger.warning("Token revoked — shutting down")
            break
```

---

## What this replaces

Before per-workspace tokens, the platform used an org-level bootstrap token for all agent authentication. Every agent in the org shared the same credential. That credential could reach every workspace in the org. Revoking it took every agent offline simultaneously.

Per-workspace tokens change the model:

| | Shared bootstrap token | Per-workspace token |
|---|---|---|
| Scope | Org-wide | Single workspace |
| Blast radius on compromise | Every workspace | One workspace |
| Selective revocation | No | Yes |
| Secrets rotation | Redeploy every agent | Rotate in platform UI |
| Audit attribution | Org-level only | Per-workspace with key prefix |

The tradeoff is operational discipline: you need to mint one token per agent, store it securely, and manage revocation when agents are decommissioned. For teams with a handful of agents, this is manageable. For large fleets, the token lifecycle is a process worth formalizing.

---

## Getting the token for a new agent

```bash
# Admin creates the workspace with an external agent
curl -X POST https://acme.moleculesai.app/workspaces \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "researcher", "runtime": "external", "tier": 2}'
# → {"id": "ws-abc123", "token": "mka_ab_c7xXz..."}
```

The token is returned once at workspace creation. It must be stored at that point — the platform will never return it again. If the token is lost, revoke the workspace token from Canvas Settings and mint a new one.

```bash
# Agent uses the token to register
curl -X POST https://acme.moleculesai.app/registry/register \
  -H "Authorization: Bearer mka_ab_c7xXz..."
```

The registration call exchanges the one-time admin token for the agent's long-lived bearer token. From that point on, the agent authenticates with its own scoped credential.

→ [Remote Workspaces Guide](/docs/guides/remote-workspaces.md)
→ [Organization API Keys](/docs/guides/org-api-keys.md)
→ [External Agent Registration Reference](/docs/guides/external-agent-registration.md)