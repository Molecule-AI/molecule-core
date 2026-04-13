"""Molecule AI remote-agent SDK — build agents that run outside the platform
network and register as first-class workspaces.

This is the Phase 30.8 companion to ``molecule_plugin`` (for plugin authors).
Where ``molecule_plugin`` helps you ship installable behavior for workspaces
that already exist, ``molecule_agent`` helps you *be* a workspace from the
other side of the wire: register, authenticate, pull secrets, heartbeat,
and detect pause/resume/delete — all via the Phase 30.1–30.5 HTTP contract.

Intended usage::

    from molecule_agent import RemoteAgentClient

    client = RemoteAgentClient(
        workspace_id="550e8400-e29b-41d4-a716-446655440000",
        platform_url="https://your-platform.example.com",
        agent_card={"name": "my-remote-agent", "skills": []},
    )
    client.register()                # mints + persists the auth token
    env = client.pull_secrets()      # decrypted secrets dict
    client.run_heartbeat_loop()      # background heartbeat + state-poll

See ``examples/remote-agent/`` for a runnable demo.

Design notes:
* **No async.** The SDK uses blocking ``requests`` so a remote agent author
  can embed it in any event loop / thread / script without forcing anyio.
* **Token cached on disk** at ``~/.molecule/<workspace_id>/.auth_token``
  with 0600 permissions, so a restart of the agent doesn't re-issue a
  token (the platform refuses to issue a second token when one is on file).
* **Pause/delete detection is polling-based** because remote agents usually
  can't expose an inbound WebSocket reachable from the platform.
"""

from __future__ import annotations

from .client import PeerInfo, RemoteAgentClient, WorkspaceState

__all__ = ["RemoteAgentClient", "WorkspaceState", "PeerInfo", "__version__"]
__version__ = "0.1.0"
