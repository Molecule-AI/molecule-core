"""Workspace-scoped awareness backend wrapper.

The agent-facing memory tools keep their existing signatures and delegate
to this helper when workspace awareness is configured.
"""

from __future__ import annotations

import os
import sys
from types import SimpleNamespace
from typing import Any

from policies.namespaces import resolve_awareness_namespace

try:  # pragma: no cover - optional runtime dependency in lightweight test envs
    import httpx  # type: ignore
except ImportError:  # pragma: no cover
    httpx = SimpleNamespace(AsyncClient=None)


DEFAULT_AWARENESS_TIMEOUT = 10.0


def get_awareness_config() -> dict[str, str] | None:
    """Return awareness connection settings if the workspace is configured."""
    base_url = os.environ.get("AWARENESS_URL", "").rstrip("/")
    workspace_id = os.environ.get("WORKSPACE_ID", "")
    configured_namespace = os.environ.get("AWARENESS_NAMESPACE", "")
    if not base_url:
        return None
    if not workspace_id and not configured_namespace:
        return None
    namespace = resolve_awareness_namespace(workspace_id, configured_namespace)
    return {
        "base_url": base_url,
        "namespace": namespace,
    }


class AwarenessClient:
    """Small HTTP client for workspace-scoped awareness memory operations."""

    def __init__(self, base_url: str, namespace: str, timeout: float = DEFAULT_AWARENESS_TIMEOUT):
        self.base_url = base_url.rstrip("/")
        self.namespace = namespace
        self.timeout = timeout

    def _memories_url(self) -> str:
        # Keep the awareness path isolated in one helper so the contract can
        # be adjusted later without touching the agent-facing tools.
        return f"{self.base_url}/api/v1/namespaces/{self.namespace}/memories"

    async def commit(self, content: str, scope: str) -> dict[str, Any]:
        client_cls = _resolve_async_client()
        async with client_cls(timeout=self.timeout) as client:
            resp = await client.post(
                self._memories_url(),
                json={"content": content, "scope": scope},
            )
        return _parse_commit_response(resp, scope)

    async def search(self, query: str = "", scope: str = "") -> dict[str, Any]:
        params: dict[str, str] = {}
        if query:
            params["q"] = query
        if scope:
            params["scope"] = scope

        client_cls = _resolve_async_client()
        async with client_cls(timeout=self.timeout) as client:
            resp = await client.get(self._memories_url(), params=params)
        return _parse_search_response(resp)


def build_awareness_client() -> AwarenessClient | None:
    """Create an awareness client from the current workspace environment."""
    config = get_awareness_config()
    if not config:
        return None
    return AwarenessClient(config["base_url"], config["namespace"])


def _parse_commit_response(resp: httpx.Response, scope: str) -> dict[str, Any]:
    data = _safe_json(resp)
    if resp.status_code in (200, 201):
        return {"success": True, "id": data.get("id"), "scope": scope}
    return {"success": False, "error": data.get("error", resp.text)}


def _parse_search_response(resp: httpx.Response) -> dict[str, Any]:
    data = _safe_json(resp)
    if resp.status_code == 200:
        memories = data if isinstance(data, list) else data.get("memories", [])
        return {
            "success": True,
            "count": len(memories),
            "memories": memories,
        }
    return {"success": False, "error": data.get("error", resp.text)}


def _safe_json(resp: httpx.Response) -> dict[str, Any] | list[Any]:
    try:
        return resp.json()
    except ValueError:
        return {"error": resp.text}


def _resolve_async_client():
    client_cls = getattr(httpx, "AsyncClient", None)
    if client_cls is not None:
        return client_cls

    memory_module = sys.modules.get("builtin_tools.memory")
    if memory_module is not None:
        memory_httpx = getattr(memory_module, "httpx", None)
        client_cls = getattr(memory_httpx, "AsyncClient", None)
        if client_cls is not None:
            return client_cls

    raise RuntimeError("httpx.AsyncClient is unavailable")
