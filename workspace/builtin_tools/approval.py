"""Approval tool for human-in-the-loop workflows.

When an agent encounters a destructive, expensive, or unauthorized action,
it calls request_approval() which creates a request and waits for a decision.

## Notification strategy

By default this module uses a **WebSocket subscription** (APPROVAL_USE_WEBSOCKET=true
or when the ``websockets`` package is installed).  The platform pushes an
``APPROVAL_DECIDED`` event to the workspace WebSocket as soon as a human
clicks Approve / Deny on the canvas — no polling required, instant delivery.

If WebSocket is unavailable (env var opt-out or import error) the module
falls back to a **polling loop** so existing deployments without WebSocket
support continue to work without any config change.

RBAC enforcement
----------------
The calling workspace must hold a role that grants the ``"approve"`` action.
Roles are read from ``config.yaml`` under ``rbac.roles`` (default: operator).

Audit trail
-----------
Every approval lifecycle emits structured JSON Lines records:

  1. ``approval / approve / requested``  — request submitted to platform
  2. ``approval / approve / granted``    — human approved  (actor = decided_by)
  3. ``approval / approve / denied``     — human denied    (actor = decided_by)
  4. ``approval / approve / timeout``    — no decision within APPROVAL_TIMEOUT

RBAC denials emit an ``rbac / rbac.deny / denied`` event instead.

Environment variables
---------------------
PLATFORM_URL            Platform base URL            (default: http://platform:8080)
WORKSPACE_ID            This workspace's ID          (default: "")
APPROVAL_TIMEOUT        Max wait in seconds          (default: 300)
APPROVAL_POLL_INTERVAL  Polling interval in seconds  (default: 5, polling path only)
APPROVAL_USE_WEBSOCKET  "true" to force WS, "false"
                        to force polling             (default: auto-detect)
AUDIT_LOG_PATH          Path for JSON Lines audit log (default: /var/log/molecule/audit.jsonl)
"""

import asyncio
import json
import logging
import os
import uuid

import httpx
from langchain_core.tools import tool

from builtin_tools.audit import check_permission, get_workspace_roles, log_event

logger = logging.getLogger(__name__)

PLATFORM_URL = os.environ.get("PLATFORM_URL", "http://host.docker.internal:8080")
WORKSPACE_ID = os.environ.get("WORKSPACE_ID", "")
APPROVAL_POLL_INTERVAL = float(os.environ.get("APPROVAL_POLL_INTERVAL", "5"))
APPROVAL_TIMEOUT = float(os.environ.get("APPROVAL_TIMEOUT", "300"))

# Auto-detect WebSocket support; can be overridden with env var
_ws_env = os.environ.get("APPROVAL_USE_WEBSOCKET", "").lower()
if _ws_env == "false":
    _USE_WEBSOCKET_DEFAULT = False
elif _ws_env == "true":
    _USE_WEBSOCKET_DEFAULT = True
else:
    try:
        import websockets as _ws_probe  # noqa: F401
        _USE_WEBSOCKET_DEFAULT = True
    except ImportError:
        _USE_WEBSOCKET_DEFAULT = False

# Module-level reference so tests can monkeypatch it
try:
    import websockets
except ImportError:
    websockets = None  # type: ignore[assignment]

# Expose for test introspection
APPROVAL_USE_WEBSOCKET = _USE_WEBSOCKET_DEFAULT


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------

async def _create_approval_request(action: str, reason: str) -> dict:
    """POST to the platform to create an approval request.

    Returns {"approval_id": str} on success or {"error": str} on failure.
    """
    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.post(
                f"{PLATFORM_URL}/workspaces/{WORKSPACE_ID}/approvals",
                json={"action": action, "reason": reason},
            )
            if resp.status_code != 201:
                return {"error": f"Failed to create request: {resp.status_code}"}
            try:
                approval_id = resp.json().get("approval_id")
            except (ValueError, Exception):
                return {"error": f"Platform returned invalid JSON (status {resp.status_code})"}
            logger.info("Approval requested: %s (id=%s)", action, approval_id)
            return {"approval_id": approval_id}
        except Exception as e:
            return {"error": f"Failed to request approval: {e}"}


async def _wait_websocket(approval_id: str, timeout: float) -> dict:
    """Subscribe to the platform WebSocket and wait for APPROVAL_DECIDED event.

    Returns the decision dict or raises asyncio.TimeoutError on expiry.
    """
    ws_url = (
        PLATFORM_URL.replace("http://", "ws://").replace("https://", "wss://")
        + "/ws"
    )
    headers = {"X-Workspace-ID": WORKSPACE_ID}

    logger.debug("Approval %s: waiting via WebSocket %s", approval_id, ws_url)

    async with websockets.connect(ws_url, additional_headers=headers) as ws:
        async for raw_message in ws:
            try:
                event = json.loads(raw_message)
            except json.JSONDecodeError:
                continue

            if event.get("event") != "APPROVAL_DECIDED":
                continue
            if event.get("approval_id") != approval_id:
                continue

            status = event.get("status")
            decided_by = event.get("decided_by", "")
            logger.info("Approval %s decided via WebSocket: %s by %s",
                        approval_id, status, decided_by)

            if status == "approved":
                return {
                    "approved": True,
                    "approval_id": approval_id,
                    "decided_by": decided_by,
                }
            else:
                return {
                    "approved": False,
                    "approval_id": approval_id,
                    "decided_by": decided_by,
                    "message": "Denied by human",
                }


async def _wait_polling(approval_id: str, timeout: float) -> dict:
    """Legacy polling loop — checks platform REST endpoint every APPROVAL_POLL_INTERVAL seconds."""
    elapsed = 0.0
    async with httpx.AsyncClient(timeout=10.0) as client:
        while elapsed < timeout:
            await asyncio.sleep(APPROVAL_POLL_INTERVAL)
            elapsed += APPROVAL_POLL_INTERVAL
            try:
                resp = await client.get(
                    f"{PLATFORM_URL}/workspaces/{WORKSPACE_ID}/approvals",
                )
                if resp.status_code == 200:
                    for a in resp.json():
                        if a.get("id") == approval_id:
                            status = a.get("status")
                            if status == "approved":
                                logger.info("Approval granted (poll): %s", approval_id)
                                return {
                                    "approved": True,
                                    "approval_id": approval_id,
                                    "decided_by": a.get("decided_by"),
                                }
                            elif status == "denied":
                                logger.info("Approval denied (poll): %s", approval_id)
                                return {
                                    "approved": False,
                                    "approval_id": approval_id,
                                    "decided_by": a.get("decided_by"),
                                    "message": "Denied by human",
                                }
            except Exception:
                pass  # transient error — keep retrying

    raise asyncio.TimeoutError()


# ---------------------------------------------------------------------------
# Public tool
# ---------------------------------------------------------------------------

@tool
async def request_approval(
    action: str,
    reason: str,
) -> dict:
    """Request human approval before proceeding with a sensitive action.

    Use this when you're about to do something destructive, expensive,
    or outside your normal authority. The request is sent to the canvas
    where a human can approve or deny it.

    Args:
        action: Short description of what you want to do
        reason: Why this action is necessary
    """
    # One trace_id links every audit event for this approval lifecycle.
    trace_id = str(uuid.uuid4())

    # --- RBAC check -----------------------------------------------------------
    roles, custom_perms = get_workspace_roles()
    if not check_permission("approve", roles, custom_perms):
        log_event(
            event_type="rbac",
            action="rbac.deny",
            resource=action,
            outcome="denied",
            trace_id=trace_id,
            attempted_action="approve",
            roles=roles,
        )
        return {
            "approved": False,
            "error": (
                "RBAC: this workspace does not have the 'approve' permission. "
                f"Current roles: {roles}"
            ),
        }

    # Step 1: Create the approval request
    creation = await _create_approval_request(action, reason)
    if "error" in creation:
        log_event(
            event_type="approval",
            action="approve",
            resource=action,
            outcome="failure",
            trace_id=trace_id,
            reason="submit_failed",
            error=creation["error"],
        )
        return {"approved": False, "error": creation["error"]}

    approval_id = creation["approval_id"]
    log_event(
        event_type="approval",
        action="approve",
        resource=action,
        outcome="requested",
        trace_id=trace_id,
        approval_id=approval_id,
        reason_text=reason,
    )

    timeout = float(os.environ.get("APPROVAL_TIMEOUT", str(APPROVAL_TIMEOUT)))

    # Step 2: Wait for decision — WebSocket preferred, polling as fallback
    use_ws = APPROVAL_USE_WEBSOCKET and websockets is not None

    try:
        if use_ws:
            try:
                result = await asyncio.wait_for(
                    _wait_websocket(approval_id, timeout),
                    timeout=timeout,
                )
            except Exception as ws_err:
                # WebSocket failed (connection error, etc.) — fall through to polling
                logger.warning(
                    "WebSocket approval wait failed (%s), falling back to polling",
                    ws_err,
                )
                result = await asyncio.wait_for(
                    _wait_polling(approval_id, timeout),
                    timeout=timeout + APPROVAL_POLL_INTERVAL,
                )
        else:
            # Polling path (primary when WS disabled)
            result = await asyncio.wait_for(
                _wait_polling(approval_id, timeout),
                timeout=timeout + APPROVAL_POLL_INTERVAL,  # slight grace period
            )

        # Log the human decision
        decided_by = result.get("decided_by")
        outcome = "granted" if result.get("approved") else "denied"
        log_event(
            event_type="approval",
            action="approve",
            resource=action,
            outcome=outcome,
            # Record the human identity as actor when available
            actor=decided_by or WORKSPACE_ID,
            trace_id=trace_id,
            approval_id=approval_id,
            decided_by=decided_by,
        )
        return result

    except asyncio.TimeoutError:
        logger.warning("Approval timed out after %.0fs: %s", timeout, approval_id)
        log_event(
            event_type="approval",
            action="approve",
            resource=action,
            outcome="timeout",
            trace_id=trace_id,
            approval_id=approval_id,
            timeout_seconds=timeout,
        )
        return {
            "approved": False,
            "approval_id": approval_id,
            "error": f"Timed out after {timeout}s waiting for human decision",
        }
