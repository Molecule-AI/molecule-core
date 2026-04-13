"""MeDo builtin tools — Baidu MeDo no-code AI platform integration.

MeDo (摩搭, moda.baidu.com) is Baidu's no-code AI application builder used in
the Molecule AI hackathon integration (May 2026).  Three core operations:
  create_medo_app  — scaffold a new application from a template
  update_medo_app  — push content / config changes to an existing app
  publish_medo_app — publish a draft app to a target environment

Authentication: set MEDO_API_KEY as a workspace secret.
Override base URL via MEDO_BASE_URL (default: https://api.moda.baidu.com/v1).

Mock backend: when MEDO_API_KEY is absent the tools return a predictable stub
response — safe for unit tests and local development.
TODO: swap _mock_http_post for a real httpx.AsyncClient call once keys are live.
"""

import logging
import os

from langchain_core.tools import tool

logger = logging.getLogger(__name__)

MEDO_BASE_URL = os.environ.get("MEDO_BASE_URL", "https://api.moda.baidu.com/v1")
MEDO_API_KEY = os.environ.get("MEDO_API_KEY", "")

_VALID_TEMPLATES = ("blank", "chatbot", "form", "dashboard")
_VALID_ENVS = ("production", "staging")


async def _mock_http_post(path: str, payload: dict) -> dict:
    """Stub HTTP call.  TODO: replace with real httpx.AsyncClient once MEDO_API_KEY is live."""
    return {"status": "ok", "mock": True, "path": path, "payload_keys": list(payload.keys())}


@tool
async def create_medo_app(name: str, template: str = "blank", description: str = "") -> dict:
    """Create a new MeDo application.

    Args:
        name: Application name (required).
        template: Starting template — blank | chatbot | form | dashboard (default: blank).
        description: Short description of the application.

    Returns:
        dict with 'app_id' and 'status' on success, 'error' key on failure.
    """
    if not name:
        return {"error": "name is required"}
    if template not in _VALID_TEMPLATES:
        return {"error": f"template must be one of: {', '.join(_VALID_TEMPLATES)}"}
    try:
        result = await _mock_http_post("/apps", {"name": name, "template": template, "description": description})
        logger.info("MeDo create_app: name=%s template=%s → %s", name, template, result)
        return result
    except Exception as exc:
        logger.exception("MeDo create_app failed")
        return {"error": str(exc)}


@tool
async def update_medo_app(app_id: str, content: dict) -> dict:
    """Push content or configuration changes to an existing MeDo application.

    Args:
        app_id: The MeDo application ID returned by create_medo_app.
        content: Dict of fields to update (e.g. {"title": "...", "nodes": [...]}).

    Returns:
        dict with 'status' on success, 'error' key on failure.
    """
    if not app_id:
        return {"error": "app_id is required"}
    if not content:
        return {"error": "content must be a non-empty dict"}
    try:
        result = await _mock_http_post(f"/apps/{app_id}", content)
        logger.info("MeDo update_app: app_id=%s keys=%s → %s", app_id, list(content.keys()), result)
        return result
    except Exception as exc:
        logger.exception("MeDo update_app failed")
        return {"error": str(exc)}


@tool
async def publish_medo_app(app_id: str, environment: str = "production") -> dict:
    """Publish a MeDo application to a target environment.

    Args:
        app_id: The MeDo application ID to publish.
        environment: Target — production | staging (default: production).

    Returns:
        dict with 'status' on success, 'error' key on failure.
    """
    if not app_id:
        return {"error": "app_id is required"}
    if environment not in _VALID_ENVS:
        return {"error": f"environment must be one of: {', '.join(_VALID_ENVS)}"}
    try:
        result = await _mock_http_post(f"/apps/{app_id}/publish", {"environment": environment})
        logger.info("MeDo publish_app: app_id=%s env=%s → %s", app_id, environment, result)
        return result
    except Exception as exc:
        logger.exception("MeDo publish_app failed")
        return {"error": str(exc)}
