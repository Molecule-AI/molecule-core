"""A2A protocol client — peer discovery, messaging, and workspace info.

Shared constants (WORKSPACE_ID, PLATFORM_URL) live here so that
a2a_tools and a2a_mcp_server can import them from a single place.
"""

import logging
import os
import uuid

import httpx

from platform_auth import auth_headers, self_source_headers

logger = logging.getLogger(__name__)

_WORKSPACE_ID_raw = os.environ.get("WORKSPACE_ID")
if not _WORKSPACE_ID_raw:
    raise RuntimeError("WORKSPACE_ID environment variable is required but not set")
WORKSPACE_ID = _WORKSPACE_ID_raw
if os.path.exists("/.dockerenv") or os.environ.get("DOCKER_VERSION"):
    PLATFORM_URL = os.environ.get("PLATFORM_URL", "http://host.docker.internal:8080")
else:
    PLATFORM_URL = os.environ.get("PLATFORM_URL", "http://localhost:8080")

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
                headers={"X-Workspace-ID": WORKSPACE_ID, **auth_headers()},
            )
            if resp.status_code == 200:
                return resp.json()
            return None
        except Exception as e:
            logger.error(f"Discovery failed for {target_id}: {e}")
            return None


async def send_a2a_message(target_url: str, message: str) -> str:
    """Send an A2A message/send to a target workspace."""
    # Fix F (Cycle 5 / H2 — flagged 5 consecutive audits): timeout=None allowed
    # a hung upstream to block the agent indefinitely. Use a generous but bounded
    # timeout: 30s connect + 300s read (long enough for slow LLM responses).
    async with httpx.AsyncClient(
        timeout=httpx.Timeout(connect=30.0, read=300.0, write=30.0, pool=30.0)
    ) as client:
        try:
            # self_source_headers() includes X-Workspace-ID so the
            # platform's a2a_receive logger records source_id =
            # WORKSPACE_ID. Otherwise peer-A2A messages — including
            # the case where target_url resolves to this workspace's
            # own /a2a — get logged with source_id=NULL and surface
            # in the recipient's My Chat tab as user-typed input.
            resp = await client.post(
                target_url,
                headers=self_source_headers(WORKSPACE_ID),
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
            resp = await client.get(
                f"{PLATFORM_URL}/registry/{WORKSPACE_ID}/peers",
                headers={"X-Workspace-ID": WORKSPACE_ID, **auth_headers()},
            )
            if resp.status_code == 200:
                return resp.json()
            return []
        except Exception:
            return []


async def get_workspace_info() -> dict:
    """Get this workspace's info from the platform."""
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.get(
                f"{PLATFORM_URL}/workspaces/{WORKSPACE_ID}",
                headers=auth_headers(),
            )
            if resp.status_code == 200:
                return resp.json()
            return {"error": "not found"}
        except Exception as e:
            return {"error": str(e)}
