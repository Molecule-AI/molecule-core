"""Platform client for registering and maintaining external workspaces."""

import logging
import time
import threading
from pathlib import Path

import httpx

logger = logging.getLogger("bridge.platform")


class PlatformClient:
    """Manages the external workspace lifecycle with the Molecule AI platform."""

    def __init__(self, platform_url: str, bridge_port: int, data_dir: Path):
        self.platform_url = platform_url
        self.bridge_port = bridge_port
        self.data_dir = data_dir
        self.ws_id_file = data_dir / "workspace_id"
        self.workspace_id = ""

    def register(self, name: str, role: str, tier: int = 3, parent_id: str = "") -> str:
        """Register as an external workspace. Returns workspace ID."""
        # Reuse existing if alive
        if self.ws_id_file.exists():
            ws_id = self.ws_id_file.read_text().strip()
            try:
                resp = httpx.get(f"{self.platform_url}/workspaces/{ws_id}", timeout=5)
                if resp.status_code == 200 and resp.json().get("status") not in ("removed",):
                    httpx.patch(f"{self.platform_url}/workspaces/{ws_id}",
                        json={"url": f"http://127.0.0.1:{self.bridge_port}"}, timeout=5)
                    self.workspace_id = ws_id
                    logger.info(f"Reusing workspace {ws_id}")
                    return ws_id
            except Exception:
                pass

        # Create new
        payload = {
            "name": name,
            "role": role,
            "tier": tier,
            "runtime": "external",
            "external": True,
            "url": f"http://127.0.0.1:{self.bridge_port}",
        }
        if parent_id:
            payload["parent_id"] = parent_id

        resp = httpx.post(f"{self.platform_url}/workspaces", json=payload, timeout=10)
        data = resp.json()
        ws_id = data.get("id", "")
        if not ws_id:
            raise RuntimeError(f"Failed to create workspace: {data}")

        self.ws_id_file.write_text(ws_id)
        self.workspace_id = ws_id
        logger.info(f"Created external workspace: {ws_id}")

        # Register agent card
        httpx.post(f"{self.platform_url}/registry/register", json={
            "workspace_id": ws_id,
            "agent_card": {
                "name": name,
                "description": role,
                "url": f"http://127.0.0.1:{self.bridge_port}",
                "version": "1.0.0",
                "skills": [],
                "capabilities": {"streaming": False, "pushNotifications": False},
            },
        }, timeout=10)

        return ws_id

    def start_heartbeat(self):
        """Send periodic heartbeats to keep the workspace online."""
        start = time.time()
        def loop():
            while True:
                try:
                    httpx.post(f"{self.platform_url}/registry/heartbeat", json={
                        "workspace_id": self.workspace_id,
                        "error_rate": 0, "sample_error": "",
                        "uptime_seconds": int(time.time() - start),
                        "active_tasks": 0, "current_task": "",
                    }, timeout=5)
                except Exception:
                    pass
                time.sleep(30)
        threading.Thread(target=loop, daemon=True).start()

    def resolve_name(self, workspace_id: str) -> str:
        """Look up workspace name."""
        try:
            resp = httpx.get(f"{self.platform_url}/workspaces/{workspace_id}", timeout=3)
            if resp.status_code == 200:
                return resp.json().get("name", workspace_id[:8])
        except Exception:
            pass
        return workspace_id[:8]
