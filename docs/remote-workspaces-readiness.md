# Remote Workspaces — Readiness Audit

**Status:** Phase 30.1 shipped (auth tokens + token management API). Phases 30.2–30.7 in progress.
**Last reviewed:** 2026-04-16
**Scope:** what it takes to let a Python agent on a different machine / different
network / behind NAT join the same Molecule AI organization as a first-class workspace.

This doc backs the [Phase 30 plan](../PLAN.md). Its purpose is to make sure we
are not building a parallel subsystem — the existing `runtime='external'` path
already handles ~80% of what remote workspaces need; the remaining 20% is four
bounded additions plus per-workspace authentication.

---

## 1. Today's local-only assumptions

Each bullet names the function and why remote would break it. Line numbers
drift — grep for the function name.

- **A2A proxy URL rewrite** — `platform/internal/handlers/a2a_proxy.go::detectPlatformInDocker()`
  and URL rewrite at request time. Rewrites `http://127.0.0.1:<port>` to
  `http://ws-<id>:8000` (Docker DNS) when platform runs inside Docker. Remote
  agent URL is `http://203.0.113.x:8080` or similar — no Docker DNS, no
  rewrite should happen. Already guarded by the ephemeral-localhost check,
  but untested for WAN URLs.

- **Health sweep** — `platform/internal/registry.StartHealthSweep`. Polls
  Docker daemon every 15s via `ContainerChecker.IsRunning(id)`. Already
  filters `WHERE runtime != 'external'`, so remote agents are skipped.
  Good — liveness for remote has to come from heartbeat TTL instead.

- **Auto-restart** — `platform/internal/handlers/workspace_restart.go::RestartByID`.
  Early-returns if `runtime == 'external'`. Good — no Docker restart for
  remote. Means remote agents must run their own supervisor.

- **Container file ops** — `container_files.go::findContainer` +
  `execInContainer`. Resolves container by `ws-<id>` name, runs
  `docker exec`. No remote equivalent. Uses: plugin install, uninstall,
  terminal tab, config writes post-provision.

- **Secrets delivery** — `workspace_provision.go`. Secrets are decrypted
  from DB and passed as env vars at `ContainerCreate` time. Remote agent
  was never provisioned by us — it needs a pull endpoint.

- **Bind mounts & config volume** — `provisioner.Start`. Creates
  `ws-<id>-configs` volume, mounts it at `/configs`, writes template
  files into it. Remote agent owns its own filesystem.

- **Liveness monitor** — Redis 60s TTL keyed by workspace. Works
  identically for remote agents that call `POST /registry/heartbeat`.
  No change needed beyond slightly longer TTL to tolerate WAN jitter.

- **Canvas push (WebSocket)** — `ws.Hub` pushes `WORKSPACE_PAUSED`,
  `WORKSPACE_OFFLINE`, etc. to connected clients. Local agents do not
  listen to this. Remote agents can't reach the WS port inbound.
  Need: polled `GET /workspaces/:id/state` with event tail.

- **Access control** — `registry/access.go::CanCommunicate`. Pure DB
  query (same parent / parent-child / both-root). Works for remote
  with no change — the proxy already uses it for every A2A call.

## 2. Existing seams we can build on

- **`runtime='external'` escape hatch** — `platform/internal/models/workspace.go`
  + migration 011 + every Docker-touching handler already gates on this.
  Reuse. Do not add a parallel "remote" flag.

- **Registry endpoints** — `POST /registry/register`, `POST /registry/heartbeat`,
  `POST /registry/update-card`, `GET /registry/:id/peers`. All already
  accept any HTTP caller and persist the returned URL. These ARE the remote
  registration contract today — we just haven't authenticated them.

- **Discovery URL rewrite** — `discovery.go::Discover` already rewrites
  `127.0.0.1` to `host.docker.internal` when the caller is a Docker
  workspace looking up an external workspace. The infrastructure for
  "URLs that point outside the host" exists.

- **`PLATFORM_URL` env-var pattern** — provisioner injects
  `PLATFORM_URL` + `MOLECULE_URL` into every container.
  `workspace-template/main.py` reads it. Remote agent just reads the
  same env var — no new plumbing.

- **Bundle export/import** — `platform/internal/bundle/`. The lingua
  franca for "move a workspace's config + prompts + skills." Can mark
  `external=true` on import. Useful for "I have a template I want to
  run on my own machine."

- **A2A proxy is URL-scheme agnostic** — `a2a_proxy.go::ProxyA2ARequest`
  doesn't care whether the URL is Docker-internal or WAN. It hits
  whatever is in the DB.

## 3. Hard problems (named explicitly)

| # | Problem | Impact | Solution zone |
|---|---------|--------|---------------|
| A | **Spoofing.** ~~`X-Workspace-ID` is a namespace header, not auth.~~ **SHIPPED (30.1).** Per-workspace bearer tokens now required on heartbeat, update-card, discover, peers, secrets, and all /workspaces/:id/* sub-routes. Token management API: `GET/POST/DELETE /workspaces/:id/tokens`. See [token-management.md](guides/token-management.md). | ~~Blocker~~ **Resolved.** | Per-workspace auth tokens (30.1) ✅ |
| B | **NAT / firewall asymmetry.** Agent→platform: fine (outbound). Platform→agent: blocked for most home/office agents. | Anything platform-initiated (config push, restart, plugin install, WS event) fails. | Pull-based APIs for the things that today are pushed (30.2, 30.3, 30.4). |
| C | **Secrets delivery.** Today: push at container-create. Remote agent was never provisioned. | Remote agent can't get API keys; any tool that needs them fails. | `GET /workspaces/:id/secrets` (30.2). |
| D | **Plugin install.** Today: `docker exec pip install` into the container. No Docker for remote. | Remote agent can't install plugins that require deps. | Plugin tarball download (30.3); agent runs its own install. |
| E | **Pause/resume/delete events.** Today: pushed via platform WebSocket to agents. Remote can't receive. | Remote agent unaware when user pauses it. | Agent polls `GET /workspaces/:id/state` (30.4). |
| F | **Liveness semantics.** Today: "Docker says running." Not applicable to remote. | Health sweep skips remote (good); nothing actively monitors heartbeat freshness. | Poll-liveness checker: no heartbeat in N seconds → offline (30.7). |
| G | **Agent-to-agent reachability across NATs.** Two behind-NAT agents can't reach each other directly. | Sibling A2A calls must route through the platform (works, but slow and adds a single point of failure). | Direct URL cache where possible (30.6); relay is out of scope for Phase 30. |

## 4. Minimum viable remote-workspace shape

Onboarding call sequence from the agent's point of view:

```
1. agent boots with env: WORKSPACE_ID, PLATFORM_URL
2. POST $PLATFORM_URL/registry/register  →  { token, ... }
3. GET  $PLATFORM_URL/workspaces/:id/secrets          Authorization: Bearer $TOKEN
4. GET  $PLATFORM_URL/plugins/:name/download  (if plugin needed)
5. heartbeat loop:
   POST $PLATFORM_URL/registry/heartbeat              Authorization: Bearer $TOKEN
   GET  $PLATFORM_URL/workspaces/:id/state            Authorization: Bearer $TOKEN
6. receives A2A from parent/siblings at its own HTTP port (or long-poll if
   behind NAT — Phase 30+ work).
```

Data-model diff from today:

```sql
CREATE TABLE workspace_auth_tokens (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  token_hash   BYTEA NOT NULL,            -- sha256(plaintext); never store plaintext
  prefix       TEXT  NOT NULL,            -- first 8 chars for display / debugging
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at TIMESTAMPTZ,
  revoked_at   TIMESTAMPTZ,
  UNIQUE (token_hash)
);
CREATE INDEX ON workspace_auth_tokens (workspace_id) WHERE revoked_at IS NULL;
```

No other schema changes. Remote agents already use the existing
`url` and `runtime='external'` fields.

The `external` flag already covers ~80% of the behavior we need.
The 20% gap: auth (30.1), secrets pull (30.2), plugin tarball (30.3),
state polling (30.4), live A2A proxy auth (30.5), sibling URL cache
(30.6), poll-liveness (30.7). No single step is large.

## 5. Ordered next-step list

See [PLAN.md Phase 30](../PLAN.md). Eight steps, ~2 weeks to GA.
Step 30.1 is shipped. Steps 30.2–30.8 can parallelize.

## 6. Related guides

- [External Agent Registration Guide](guides/external-agent-registration.md) — step-by-step for any agent to join, with Python + Node.js examples
- [Token Management API](guides/token-management.md) — create, list, revoke bearer tokens
- [MCP Server Setup](guides/mcp-server-setup.md) — 87 tools for managing workspaces via MCP
