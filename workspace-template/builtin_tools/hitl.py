"""Human-In-The-Loop (HITL) workflow primitives.

Generalizes the approval tool into reusable HITL building blocks that work
across all Molecule AI adapters.

Features
--------
@requires_approval
    Decorator that gates *any* async callable (tool, method, standalone fn)
    behind a human approval request.  The decorated function only runs if
    the request is granted.  Roles in ``hitl.bypass_roles`` skip the gate.

pause_task / resume_task
    LangChain tools for explicit pause/resume of in-flight tasks.  An agent
    calls ``pause_task(task_id, reason)`` to suspend itself; an external
    signal (webhook, dashboard click, another agent) calls ``resume_task``
    with the same task_id to wake it up.

Notification channels
---------------------
Configured under ``hitl:`` in ``config.yaml``:

    hitl:
      channels:
        - type: dashboard        # always active; uses platform approval API
        - type: slack
          webhook_url: https://hooks.slack.com/services/…
        - type: email
          smtp_host: smtp.example.com
          smtp_port: 587
          from: alerts@example.com
          to: ops@example.com
          username: alerts@example.com   # optional; password from SMTP_PASSWORD env
      default_timeout: 300          # seconds before an unanswered request times out
      bypass_roles: [admin]         # roles that skip the approval gate entirely

Environment variables
---------------------
SMTP_PASSWORD   Password for SMTP authentication (preferred over config file)
"""

from __future__ import annotations

import asyncio
import functools
import logging
import os
import smtplib
from dataclasses import dataclass, field
from email.mime.text import MIMEText
from typing import Any, Callable

import httpx
from langchain_core.tools import tool

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

@dataclass
class HITLConfig:
    """HITL settings loaded from the ``hitl:`` block in config.yaml."""
    channels: list[dict] = field(default_factory=lambda: [{"type": "dashboard"}])
    default_timeout: float = 300.0
    bypass_roles: list[str] = field(default_factory=list)


def _load_hitl_config() -> HITLConfig:
    """Load HITL config from workspace config; fall back to safe defaults."""
    try:
        from config import load_config
        cfg = load_config()
        raw = getattr(cfg, "hitl", None)
        if raw is None:
            return HITLConfig()
        return HITLConfig(
            channels=raw.channels if hasattr(raw, "channels") else [{"type": "dashboard"}],
            default_timeout=float(raw.default_timeout if hasattr(raw, "default_timeout") else 300),
            bypass_roles=list(raw.bypass_roles if hasattr(raw, "bypass_roles") else []),
        )
    except Exception:
        return HITLConfig()


# ---------------------------------------------------------------------------
# Pause / Resume registry
# ---------------------------------------------------------------------------

class _TaskPauseRegistry:
    """In-process registry mapping task_id → asyncio.Event + optional result.

    Multiple coroutines awaiting the same task_id are all unblocked when
    ``resume()`` is called.  Results survive until the awaiting coroutine
    calls ``pop_result()``.
    """

    def __init__(self) -> None:
        self._events: dict[str, asyncio.Event] = {}
        self._results: dict[str, dict] = {}
        # #265: owner map — workspace_id that created each task.
        # Empty string means "no owner / legacy" (bypasses ownership check).
        self._owners: dict[str, str] = {}

    def register(self, task_id: str, owner: str = "") -> asyncio.Event:
        """Create and store an Event for *task_id*.  Returns the event.

        Args:
            task_id: Unique task identifier.
            owner:   Workspace ID that owns this task.  When set, ``resume``
                     will reject callers from a different workspace.
        """
        ev = asyncio.Event()
        self._events[task_id] = ev
        self._owners[task_id] = owner
        return ev

    def resume(self, task_id: str, result: dict | None = None, owner: str = "") -> bool:
        """Signal the Event for *task_id*.  Returns False if not registered.

        Args:
            task_id: The identifier used in ``register``.
            result:  Optional result payload forwarded to the waiting coroutine.
            owner:   Caller's workspace ID.  When both the stored owner and
                     *owner* are non-empty and they differ, the call is rejected
                     (returns False) — prevents cross-workspace prompt injection
                     (#265).  Passing ``owner=""`` bypasses the check (used in
                     direct registry calls from tests and platform code).
        """
        # #265 ownership check
        stored_owner = self._owners.get(task_id, "")
        if owner and stored_owner and owner != stored_owner:
            logger.warning(
                "HITL: resume rejected for task %s — caller workspace %r != owner %r",
                task_id, owner, stored_owner,
            )
            return False
        ev = self._events.get(task_id)
        if ev is None:
            return False
        self._results[task_id] = result or {}
        ev.set()
        return True

    def pop_result(self, task_id: str) -> dict:
        """Return and remove the stored result for *task_id*."""
        return self._results.pop(task_id, {})

    def cleanup(self, task_id: str) -> None:
        """Remove *task_id* from all dicts."""
        self._events.pop(task_id, None)
        self._results.pop(task_id, None)
        self._owners.pop(task_id, None)

    def list_paused(self) -> list[str]:
        """Return IDs of tasks whose events have not yet been set."""
        return [tid for tid, ev in self._events.items() if not ev.is_set()]


# Global singleton — safe within one asyncio event loop / process
pause_registry = _TaskPauseRegistry()


# ---------------------------------------------------------------------------
# Notification channels
# ---------------------------------------------------------------------------

async def _notify_channels(
    action: str,
    reason: str,
    approval_id: str,
    cfg: HITLConfig,
) -> None:
    """Fire-and-forget notifications to all configured channels.

    Errors in individual channels are logged but never re-raised so that a
    misconfigured Slack webhook cannot block the approval flow.
    """
    platform_url = os.environ.get("PLATFORM_URL", "http://platform:8080")
    workspace_id = os.environ.get("WORKSPACE_ID", "")

    for channel in cfg.channels:
        ch_type = channel.get("type", "dashboard")
        try:
            if ch_type == "slack":
                await _notify_slack(channel, action, reason, approval_id,
                                    platform_url, workspace_id)
            elif ch_type == "email":
                await _notify_email(channel, action, reason, approval_id,
                                    platform_url, workspace_id)
            # "dashboard" is handled by the platform via the approval POST
        except Exception as exc:
            logger.warning("HITL: channel '%s' notification failed: %s", ch_type, exc)


async def _notify_slack(
    cfg: dict,
    action: str,
    reason: str,
    approval_id: str,
    platform_url: str,
    workspace_id: str,
) -> None:
    webhook_url = cfg.get("webhook_url", "")
    if not webhook_url:
        return

    approve_url = f"{platform_url}/workspaces/{workspace_id}/approvals/{approval_id}/approve"
    deny_url    = f"{platform_url}/workspaces/{workspace_id}/approvals/{approval_id}/deny"

    payload = {
        "text": f":warning: Approval required from workspace `{workspace_id}`",
        "blocks": [
            {
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": (
                        f"*Action:* {action}\n"
                        f"*Reason:* {reason}\n"
                        f"*Approval ID:* `{approval_id}`"
                    ),
                },
            },
            {
                "type": "actions",
                "elements": [
                    {
                        "type": "button",
                        "text": {"type": "plain_text", "text": "Approve"},
                        "style": "primary",
                        "url": approve_url,
                    },
                    {
                        "type": "button",
                        "text": {"type": "plain_text", "text": "Deny"},
                        "style": "danger",
                        "url": deny_url,
                    },
                ],
            },
        ],
    }
    async with httpx.AsyncClient(timeout=10.0) as client:
        await client.post(webhook_url, json=payload)
    logger.info("HITL: Slack notification sent for approval %s", approval_id)


async def _notify_email(
    cfg: dict,
    action: str,
    reason: str,
    approval_id: str,
    platform_url: str,
    workspace_id: str,
) -> None:
    smtp_host = cfg.get("smtp_host", "")
    smtp_port = int(cfg.get("smtp_port", 587))
    from_addr = cfg.get("from", "")
    to_addr   = cfg.get("to", "")

    if not all([smtp_host, from_addr, to_addr]):
        logger.warning("HITL: email channel missing smtp_host/from/to — skipping")
        return

    approve_url = f"{platform_url}/workspaces/{workspace_id}/approvals/{approval_id}/approve"
    deny_url    = f"{platform_url}/workspaces/{workspace_id}/approvals/{approval_id}/deny"

    body = (
        f"Approval required from workspace {workspace_id}\n\n"
        f"Action : {action}\n"
        f"Reason : {reason}\n"
        f"ID     : {approval_id}\n\n"
        f"Approve: {approve_url}\n"
        f"Deny   : {deny_url}\n"
    )

    msg = MIMEText(body, "plain", "utf-8")
    msg["Subject"] = f"[Molecule AI] Approval required: {action}"
    msg["From"]    = from_addr
    msg["To"]      = to_addr

    username = cfg.get("username", "")
    password = cfg.get("password", os.environ.get("SMTP_PASSWORD", ""))

    def _send() -> None:
        with smtplib.SMTP(smtp_host, smtp_port) as srv:
            srv.ehlo()
            srv.starttls()
            if username and password:
                srv.login(username, password)
            srv.send_message(msg)

    await asyncio.to_thread(_send)
    logger.info("HITL: email notification sent for approval %s", approval_id)


# ---------------------------------------------------------------------------
# @requires_approval decorator
# ---------------------------------------------------------------------------

def requires_approval(
    action_description: str = "",
    reason_template: str = "",
    bypass_roles: list[str] | None = None,
) -> Callable[[Callable], Callable]:
    """Decorator that gates an async callable behind a human approval request.

    The wrapped function executes only when a human approves.  Use this on
    any tool or async helper that performs destructive or high-impact work.

    Args:
        action_description: Short label for the action shown to the approver.
                            Defaults to the function's ``name`` attribute or
                            ``__name__``.
        reason_template:    f-string template for the reason line.  Keyword
                            arguments of the decorated function are available,
                            e.g. ``"Delete table {table_name}"``).
        bypass_roles:       Roles that skip the gate entirely.  Overrides
                            ``hitl.bypass_roles`` in config.yaml when given.

    Returns:
        A decorator; applying it to a function returns an async wrapper.

    Usage::

        @tool
        @requires_approval("Wipe production DB", bypass_roles=["admin"])
        async def drop_table(table_name: str) -> dict:
            ...

        # Works with plain async functions too:
        @requires_approval("Send customer email")
        async def send_email(to: str, body: str) -> dict:
            ...
    """
    def decorator(fn: Callable) -> Callable:
        action = action_description or getattr(fn, "name", None) or fn.__name__

        @functools.wraps(fn)
        async def wrapper(*args: Any, **kwargs: Any) -> Any:
            hitl_cfg = _load_hitl_config()

            # --- Check bypass roles -----------------------------------------
            active_bypass = bypass_roles if bypass_roles is not None else hitl_cfg.bypass_roles
            if active_bypass:
                try:
                    from builtin_tools.audit import get_workspace_roles
                    roles, _ = get_workspace_roles()
                    if any(r in active_bypass for r in roles):
                        logger.info(
                            "@requires_approval bypassed (role %s) for '%s'", roles, action
                        )
                        return await fn(*args, **kwargs)
                except Exception:
                    pass  # If RBAC check fails, proceed to approval gate

            # --- Build reason string -----------------------------------------
            if reason_template:
                try:
                    reason = reason_template.format(**kwargs)
                except (KeyError, IndexError):
                    reason = reason_template
            else:
                arg_parts = [f"{k}={str(v)[:60]}" for k, v in list(kwargs.items())[:3]]
                reason = f"Args: {', '.join(arg_parts)}" if arg_parts else "Automated action"

            # --- Fire non-dashboard notifications (async, non-blocking) ------
            asyncio.create_task(
                _notify_channels(action, reason, "pending", hitl_cfg)
            )

            # --- Request approval via approval tool --------------------------
            try:
                from builtin_tools.approval import request_approval
                approval_result = await request_approval.ainvoke(
                    {"action": action, "reason": reason}
                )
            except Exception as exc:
                logger.error("@requires_approval: approval call failed: %s", exc)
                return {
                    "success": False,
                    "error": f"Approval gate error: {exc}",
                }

            if not approval_result.get("approved"):
                # Art. 14 audit: log the denial outcome so the activity log
                # contains evidence that the human oversight gate was exercised.
                try:
                    from builtin_tools.audit import log_event
                    log_event(
                        event_type="hitl",
                        action="approve",
                        resource=action,
                        outcome="denied",
                        actor=approval_result.get("decided_by"),
                        approval_id=approval_result.get("approval_id"),
                        reason=reason,
                    )
                except Exception:
                    pass
                return {
                    "success": False,
                    "error": (
                        f"Action '{action}' not approved: "
                        f"{approval_result.get('message', approval_result.get('error', 'denied'))}"
                    ),
                    "approval_id": approval_result.get("approval_id"),
                }

            # Art. 14 audit: log the approval grant before running the function.
            try:
                from builtin_tools.audit import log_event
                log_event(
                    event_type="hitl",
                    action="approve",
                    resource=action,
                    outcome="granted",
                    actor=approval_result.get("decided_by"),
                    approval_id=approval_result.get("approval_id"),
                    reason=reason,
                )
            except Exception:
                pass

            # --- Approved — run the original function ------------------------
            return await fn(*args, **kwargs)

        return wrapper

    return decorator


# ---------------------------------------------------------------------------
# Pause / Resume LangChain tools
# ---------------------------------------------------------------------------

@tool
async def pause_task(task_id: str, reason: str = "") -> dict:
    """Suspend the current task and wait for a resume signal.

    The agent calls this to pause itself at a decision point.  Execution
    resumes when ``resume_task`` is called with the same task_id, or after
    the configured ``hitl.default_timeout`` seconds.

    Args:
        task_id: Unique identifier for this pause point (use the A2A task ID
                 or any stable string that the caller can reference later).
        reason:  Human-readable description of why the task is pausing.
    """
    # #265: record workspace ownership on registration so resume_task can
    # reject callers from a different workspace (cross-workspace prompt-injection
    # prevention).  External task_id is unchanged — only internal ownership
    # metadata is added, so no tests or callers need to update their task IDs.
    _ws = os.environ.get("WORKSPACE_ID", "")

    try:
        from builtin_tools.audit import log_event
        log_event(
            event_type="hitl",
            action="pause",
            resource=task_id,
            outcome="paused",
            trace_id=task_id,
            reason=reason,
        )
    except Exception:
        pass

    event = pause_registry.register(task_id, owner=_ws)
    timeout = _load_hitl_config().default_timeout
    logger.info("HITL: task %s paused — %s", task_id, reason or "(no reason given)")

    try:
        await asyncio.wait_for(event.wait(), timeout=timeout)
        result = pause_registry.pop_result(task_id)
        logger.info("HITL: task %s resumed", task_id)
        try:
            from builtin_tools.audit import log_event
            log_event(
                event_type="hitl",
                action="resume",
                resource=task_id,
                outcome="resumed",
                trace_id=task_id,
            )
        except Exception:
            pass
        return {"resumed": True, "task_id": task_id, **result}

    except asyncio.TimeoutError:
        logger.warning("HITL: task %s timed out after %.0fs", task_id, timeout)
        try:
            from builtin_tools.audit import log_event
            log_event(
                event_type="hitl",
                action="pause",
                resource=task_id,
                outcome="timeout",
                trace_id=task_id,
                timeout_seconds=timeout,
            )
        except Exception:
            pass
        return {
            "resumed": False,
            "task_id": task_id,
            "error": f"Timed out after {timeout:.0f}s waiting for resume signal",
        }
    finally:
        pause_registry.cleanup(task_id)


@tool
async def resume_task(task_id: str, message: str = "") -> dict:
    """Resume a previously paused task.

    Signals the ``pause_task`` coroutine waiting on *task_id* to continue.
    Safe to call even if the task has already resumed or timed out (returns
    success=False in that case).

    Args:
        task_id: The identifier passed to ``pause_task``.
        message: Optional message forwarded to the resumed task.
    """
    # #265: pass caller's workspace ID so the registry can reject a resume
    # from a different workspace (ownership check in _TaskPauseRegistry.resume).
    _ws = os.environ.get("WORKSPACE_ID", "")

    result_payload = {"message": message} if message else {}
    success = pause_registry.resume(task_id, result_payload, owner=_ws)

    if success:
        logger.info("HITL: resume signal sent for task %s", task_id)
        try:
            from builtin_tools.audit import log_event
            log_event(
                event_type="hitl",
                action="resume",
                resource=task_id,
                outcome="success",
                trace_id=task_id,
                message=message,
            )
        except Exception:
            pass
        return {"success": True, "task_id": task_id}

    return {
        "success": False,
        "task_id": task_id,
        "error": "Task not found or already resumed",
    }


@tool
async def list_paused_tasks() -> dict:
    """List all tasks currently suspended and waiting for a resume signal."""
    paused = pause_registry.list_paused()
    return {"paused_tasks": paused, "count": len(paused)}
