"""A2A protocol client — peer discovery, messaging, and workspace info.

Shared constants (WORKSPACE_ID, PLATFORM_URL) live here so that
a2a_tools and a2a_mcp_server can import them from a single place.
"""

import logging
import os
import uuid

import httpx

logger = logging.getLogger(__name__)

WORKSPACE_ID = os.environ.get("WORKSPACE_ID", "")
PLATFORM_URL = os.environ.get("PLATFORM_URL", "http://platform:8080")

# Cache workspace ID → name mappings (populated by list_peers calls)
_peer_names: dict[str, str] = {}

# Sentinel prefix for errors originating from send_a2a_message / child agents.
# Used by delegate_task to distinguish real errors from normal response text.
_A2A_ERROR_PREFIX = "[A2A_ERROR] "


async def discover_peer(target_id: str) -> dict | None:
    """Discover a peer workspace's URL via the platform registry."""
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.get(
                f"{PLATFORM_URL}/registry/discover/{target_id}",
                headers={"X-Workspace-ID": WORKSPACE_ID},
            )
            if resp.status_code == 200:
                return resp.json()
            return None
        except Exception as e:
            logger.error(f"Discovery failed for {target_id}: {e}")
            return None


async def send_a2a_message(target_url: str, message: str) -> str:
    """Send an A2A message/send to a target workspace."""
    async with httpx.AsyncClient(timeout=None) as client:
        try:
            resp = await client.post(
                target_url,
                json={
                    "jsonrpc": "2.0",
                    "id": str(uuid.uuid4()),
                    "method": "message/send",
                    "params": {
                        "message": {
                            "role": "user",
                            "messageId": str(uuid.uuid4()),
                            "parts": [{"kind": "text", "text": message}],
                        }
                    },
                },
            )
            data = resp.json()
            if "result" in data:
                parts = data["result"].get("parts", [])
                text = parts[0].get("text", "") if parts else "(no response)"
                # Tag child-reported errors so the caller can detect them reliably
                if text.startswith("Agent error:"):
                    return f"{_A2A_ERROR_PREFIX}{text}"
                return text
            elif "error" in data:
                return f"{_A2A_ERROR_PREFIX}{data['error'].get('message', 'unknown')}"
            return str(data)
        except Exception as e:
            return f"{_A2A_ERROR_PREFIX}{e}"


async def get_peers() -> list[dict]:
    """Get this workspace's peers from the platform registry."""
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.get(f"{PLATFORM_URL}/registry/{WORKSPACE_ID}/peers")
            if resp.status_code == 200:
                return resp.json()
            return []
        except Exception:
            return []


async def get_workspace_info() -> dict:
    """Get this workspace's info from the platform."""
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.get(f"{PLATFORM_URL}/workspaces/{WORKSPACE_ID}")
            if resp.status_code == 200:
                return resp.json()
            return {"error": "not found"}
        except Exception as e:
            return {"error": str(e)}
