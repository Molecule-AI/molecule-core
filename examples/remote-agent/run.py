#!/usr/bin/env python3
"""Minimal remote-agent demo — Phase 30.1–30.5 end-to-end from outside the
platform's Docker network.

What this does:
1. Registers the workspace with the platform (mints + saves a bearer token).
2. Pulls the merged decrypted secrets via the token-gated 30.2 endpoint.
3. Runs a heartbeat + state-poll loop; exits cleanly when the platform
   reports the workspace paused or deleted.

What it doesn't do (future 30.8b work):
- Host an inbound A2A server. Platform-initiated calls to this agent
  won't reach it unless you expose one yourself.

Usage:
    # One-time setup on the platform side:
    #   1) Create the workspace row (any template is fine — external runtime
    #      is cleanest if you don't want Docker to try to start a container):
    curl -s -X POST http://localhost:8080/workspaces \\
        -H 'Content-Type: application/json' \\
        -d '{"name":"remote-demo","tier":2,"runtime":"external"}'
    #   2) Grab the returned workspace id.
    #   3) Optional — seed a secret:
    curl -s -X POST http://localhost:8080/workspaces/<id>/secrets \\
        -H 'Content-Type: application/json' \\
        -d '{"key":"REMOTE_DEMO_KEY","value":"hello-from-remote"}'

    # Now run this script from any machine that can reach the platform:
    WORKSPACE_ID=<id> PLATFORM_URL=http://localhost:8080 python3 run.py

Environment variables:
    WORKSPACE_ID    (required)
    PLATFORM_URL    (required)
    AGENT_NAME      (optional; default derived from workspace id)
    MAX_ITERATIONS  (optional; caps loop length for demos)
"""
from __future__ import annotations

import logging
import os
import sys

# Local-dev import path — when installed via pip the molecule_agent package
# resolves normally; when running from the repo checkout we add sdk/python/
# to sys.path so you can run `python3 run.py` without a pip install.
_here = os.path.dirname(os.path.abspath(__file__))
_sdk = os.path.join(_here, "..", "..", "sdk", "python")
if os.path.isdir(_sdk) and _sdk not in sys.path:
    sys.path.insert(0, _sdk)

from molecule_agent import RemoteAgentClient  # noqa: E402


def main() -> int:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(message)s",
    )
    log = logging.getLogger("remote-agent-demo")

    workspace_id = os.environ.get("WORKSPACE_ID", "").strip()
    platform_url = os.environ.get("PLATFORM_URL", "").strip()
    if not workspace_id or not platform_url:
        log.error("set WORKSPACE_ID and PLATFORM_URL and re-run")
        return 2

    agent_name = os.environ.get("AGENT_NAME", f"remote-demo-{workspace_id[:8]}")
    max_iter_env = os.environ.get("MAX_ITERATIONS", "").strip()
    max_iter = int(max_iter_env) if max_iter_env else None

    client = RemoteAgentClient(
        workspace_id=workspace_id,
        platform_url=platform_url,
        agent_card={"name": agent_name, "skills": []},
        # Shorter intervals for demo visibility; production would leave defaults.
        heartbeat_interval=5.0,
    )

    log.info("phase 1 — registering workspace %s with %s", workspace_id, platform_url)
    client.register()

    log.info("phase 2 — pulling secrets via 30.2 token-gated endpoint")
    try:
        secrets = client.pull_secrets()
    except Exception as exc:
        log.error("pull_secrets failed: %s", exc)
        return 1
    log.info("received %d secret(s): keys=%s", len(secrets), sorted(secrets.keys()))

    log.info("phase 3 — heartbeat + state-poll loop (will exit on pause/delete)")
    terminal = client.run_heartbeat_loop(
        max_iterations=max_iter,
        task_supplier=lambda: {"current_task": "remote-agent demo idle", "active_tasks": 0},
    )
    log.info("loop exited: terminal_status=%s", terminal)
    return 0


if __name__ == "__main__":
    sys.exit(main())
