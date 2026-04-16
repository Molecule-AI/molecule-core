# GitHub App setup — per-agent PR / issue authorship

## Goal

Stop every agent in the `molecule-dev` template from posting as `HongmingWang-Rabbit`. Replace the shared personal PAT with a GitHub App whose installation tokens authenticate workspace containers as `molecule-ai[bot]`, clearly distinct from human activity.

This is the second half of the agent-separation rollout. The first half (per-agent `git author`) ships in platform PR #402 — commits already attribute correctly; this runbook covers the remaining PR/issue authorship split.

## Prerequisites

- Admin access to `Molecule-AI` GitHub organization
- Platform build from `main` at or after the GitHub App support PR
- Ability to set platform env vars via `/admin/secrets` or Fly `secrets set`

## Part 1 — Create the App (one-time, ~5 minutes in the GitHub UI)

1. Go to https://github.com/organizations/Molecule-AI/settings/apps/new
   (or your user-level equivalent if the App should be user-scoped).

2. Fill in:
   - **GitHub App name:** `Molecule AI`
   - **Homepage URL:** `https://moleculesai.app`
   - **Webhook:** uncheck **Active** for now (we don't consume webhooks yet; enable later if we want PR-event-driven pipelines).
   - **Repository permissions** (set these exactly — over-permissioning breaks least-privilege):
     | Permission | Access |
     |---|---|
     | Contents | Read & write |
     | Pull requests | Read & write |
     | Issues | Read & write |
     | Discussions | Read & write |
     | Metadata | Read (mandatory) |
   - **Organization permissions:** none needed for now.
   - **Where can this GitHub App be installed?:** **Only on this account** (we don't distribute it).

3. Click **Create GitHub App**.

4. On the next page:
   - Note the **App ID** (numeric, top of the page).
   - Scroll to **Private keys**, click **Generate a private key**. Download the `.pem` file. Store it somewhere sensible — Fly secrets, 1Password, etc. Lose it and you re-issue.

5. Click **Install App** in the left sidebar → **Install** next to `Molecule-AI` (or your account) → select **All repositories** (or specific ones: monorepo, docs, controlplane).

6. After install, the URL will be like `https://github.com/organizations/Molecule-AI/settings/installations/<ID>`. Note the **Installation ID** (numeric).

You now have three values:
- `APP_ID` (numeric)
- `PRIVATE_KEY` (PEM file contents, including `-----BEGIN ... END-----` lines)
- `INSTALLATION_ID` (numeric)

## Part 2 — Configure the platform (~2 minutes)

The platform reads these from process env. Set them via `/admin/secrets` (which plumbs them through to the platform container):

```bash
# From a shell with admin bearer:
TOK=$(MSYS_NO_PATHCONV=1 docker exec ws-<any-alive-workspace-id> cat /configs/.auth_token)
curl -s -X POST http://localhost:8080/admin/secrets \
  -H "Authorization: Bearer $TOK" \
  -H "Content-Type: application/json" \
  -d "$(jq -n --arg id "123456" \
               --arg install "987654" \
               --arg key "$(cat /path/to/downloaded.pem)" \
               '{GITHUB_APP_ID: $id, GITHUB_APP_INSTALLATION_ID: $install, GITHUB_APP_PRIVATE_KEY: $key}')"
```

Or for Fly-deployed platform (molecule-cp, when that migrates):

```bash
fly secrets set -a <app> \
  GITHUB_APP_ID=123456 \
  GITHUB_APP_INSTALLATION_ID=987654 \
  GITHUB_APP_PRIVATE_KEY="$(cat /path/to/downloaded.pem)"
```

## Part 3 — Restart the platform + roll workspaces

Platform reads the three vars on boot. Restart to pick them up:

```bash
docker compose up -d --force-recreate platform
# or fly deploy for the Fly-hosted variant
```

Verify the log shows:
```
GitHubApp: installation token minting enabled (AppID=123456)
```

Then roll-restart each workspace so they pick up App-minted tokens:

```bash
# Each workspace needs its own bearer
for WS_ID in $(curl -s -H "Authorization: Bearer $ADMIN_TOK" http://localhost:8080/workspaces | jq -r '.[].id'); do
  TOK=$(MSYS_NO_PATHCONV=1 docker exec ws-<short>$WS_ID cat /configs/.auth_token)
  curl -s -X POST "http://localhost:8080/workspaces/$WS_ID/restart" \
    -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" -d '{}'
  sleep 3
done
```

## Part 4 — Verify

After restart, confirm an agent's next PR is authored by `molecule-ai[bot]`:

```bash
# From inside a workspace container
docker exec ws-<id> sh -c 'cd /workspace/repo && echo "# test" >> test.md && \
  git add test.md && git commit -m "test: verify App auth" && \
  gh pr create --draft --title "auth-test" --body "checking bot authorship"'

# Then:
gh pr list --repo Molecule-AI/molecule-monorepo --state open --json author --limit 1
# Expected: {"author":{"login":"molecule-ai[bot]"}}
```

Clean up the test PR afterward.

## Fallback behaviour (if the App ever misbehaves)

The platform is built to fall back gracefully:

- **Missing env vars** → logs `GitHubApp: not configured`, uses the legacy `GITHUB_TOKEN` workspace secret (the CEO's PAT). Functional but you lose the authorship split.
- **Bad PEM / non-RSA key** → logs the parse error on boot, same fallback.
- **Transient mint failure** (GitHub 5xx, network drop) → logs the error per provision attempt, uses whatever GITHUB_TOKEN was set; reattempts on next restart.

To deliberately disable the App (e.g. if we ever need to pause the bot identity):

```bash
# unset any of the three env vars; platform treats it as not configured
fly secrets unset -a <app> GITHUB_APP_ID
# restart platform
```

## Security notes

- **Private key is the only sensitive credential.** The numeric IDs are fine in logs. Treat the PEM like a password: store encrypted, rotate if it leaks.
- **Installation tokens expire in ~60 minutes.** If a workspace container is compromised, the leaked token is useful to an attacker for at most an hour. Compare to the current PAT which never expires and grants the CEO's full access.
- **App permissions are the cap.** A compromised installation token can only do what the App's permissions allow — Contents/PRs/Issues/Discussions write. It can't touch other orgs, it can't read secrets, it can't modify the account.
- **Revocation is one click:** go to the App's installation page and click Suspend.

## Open questions for follow-up

- Should we also create an App for the `docs` repo specifically (vs letting the org install cover it)? Org install is simpler if we always want the same permissions across all three repos.
- Webhook integration — if we ever want the platform to react to PR events (e.g. auto-rebase stacked PRs, auto-respond to comments), enable webhooks on the App with `/webhook/github` as the target. Out of scope for this initial install.
