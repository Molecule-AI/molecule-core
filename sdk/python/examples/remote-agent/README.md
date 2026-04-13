# Remote agent demo

A ~100-line Python script that registers with a Molecule AI platform from
outside its Docker network, pulls its secrets, and heartbeats — exercising
the Phase 30.1 / 30.2 / 30.4 endpoints end-to-end.

## Prerequisites

* A running Molecule AI platform (`./infra/scripts/setup.sh` + `go run
  ./cmd/server` from `platform/`)
* `pip install requests` in your Python environment

## Quick start

```bash
# 1. Create the workspace row on the platform. `external` runtime keeps
#    the provisioner from trying to start a Docker container:
curl -s -X POST http://localhost:8080/workspaces \
    -H 'Content-Type: application/json' \
    -d '{"name":"remote-demo","tier":2,"runtime":"external"}'
# → {"id":"<UUID>", ...}

# 2. (Optional) seed a secret so `pull_secrets` has something to return:
curl -s -X POST http://localhost:8080/workspaces/<UUID>/secrets \
    -H 'Content-Type: application/json' \
    -d '{"key":"REMOTE_DEMO_KEY","value":"hello-from-remote"}'

# 3. Run the demo from any machine that can reach the platform:
WORKSPACE_ID=<UUID> PLATFORM_URL=http://localhost:8080 \
    python3 sdk/python/examples/remote-agent/run.py
```

You should see log lines for each of the three phases, and then
heartbeat lines every 5s. The workspace should appear online on the
canvas. Pause or delete it from the canvas / via API, and the script
exits cleanly.

## What this demonstrates

| Phase | Endpoint | Shown in the demo |
|---|---|---|
| 30.1 | `POST /registry/register` | Token issuance + on-disk caching |
| 30.1 | `POST /registry/heartbeat` | Bearer-authenticated liveness report |
| 30.2 | `GET /workspaces/:id/secrets/values` | Token-gated decrypted-secrets pull |
| 30.4 | `GET /workspaces/:id/state` | Token-gated pause/delete detection |

## What it doesn't do yet

* **No inbound A2A server.** Other agents can't initiate calls back to
  this remote agent. Future 30.8b adds an optional HTTP server helper.
* **No sibling discovery.** Future 30.6 adds peer URL caching so this
  agent can call siblings directly instead of going through the proxy.

## Troubleshooting

* `401 missing workspace auth token` on the secrets/state calls — your
  cached token is stale (workspace was recreated). Delete
  `~/.molecule/<workspace_id>/.auth_token` and re-run.
* `connection refused` — double-check `PLATFORM_URL` and that the
  platform is actually listening.
* Workspace never appears as online on the canvas — confirm it was
  created with `runtime: external` (otherwise the provisioner will
  try to start a local container and fail).
