"""Immutable append-only audit log for EU AI Act compliance.

Fulfils Article 12 (record-keeping), Article 13 (transparency), and
Article 17 (quality-management system) requirements for high-risk AI systems.

Log format: JSON Lines (one UTF-8 JSON object per line), suitable for direct
ingestion by any SIEM (Splunk, Elastic, Datadog, etc.).

Required event fields
---------------------
timestamp       ISO 8601 UTC datetime with timezone offset
event_type      Coarse category: "delegation", "approval", "memory", "rbac"
workspace_id    Workspace that generated this event
actor           Entity that triggered the action; defaults to workspace_id for
                automated events, or the human identity for approval decisions
action          Verb describing what was attempted:
                  delegate | approve | memory.read | memory.write | rbac.deny
resource        Object of the action: target workspace ID, memory scope,
                approval action string, etc.
outcome         One of: allowed | denied | success | failure | timeout |
                requested | granted
trace_id        UUID v4 correlating related events across workspaces

The log file is opened in append mode ("a") on every write — it is NEVER
truncated, rewritten, or deleted by this module.  Rotate externally using
logrotate (with ``copytruncate`` disabled) or ship to a SIEM before rotating.

Configuration
-------------
AUDIT_LOG_PATH  env var — full path to the JSONL file
                default: /var/log/molecule/audit.jsonl
"""

from __future__ import annotations

import functools
import json
import logging
import os
import threading
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    pass  # avoid circular import at runtime

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

AUDIT_LOG_PATH: str = os.environ.get(
    "AUDIT_LOG_PATH", "/var/log/molecule/audit.jsonl"
)
_WS_ID_RAW = os.environ.get("WORKSPACE_ID")
if not _WS_ID_RAW:
    raise RuntimeError("WORKSPACE_ID environment variable is required but not set")
WORKSPACE_ID: str = _WS_ID_RAW

# Protects the open() + write() sequence; prevents interleaved JSON lines
# when multiple async tasks run in the same event-loop thread.
_write_lock = threading.Lock()


# ---------------------------------------------------------------------------
# Built-in role → permitted-action mappings
# ---------------------------------------------------------------------------

#: Maps each built-in role name to the set of actions it grants.
#: Custom roles can be added in config.yaml under ``rbac.allowed_actions``.
ROLE_PERMISSIONS: dict[str, set[str]] = {
    # Full access — shortcircuits all other checks
    "admin": {"delegate", "approve", "memory.read", "memory.write"},
    # Standard agent role
    "operator": {"delegate", "approve", "memory.read", "memory.write"},
    # Read-only observer — no writes, no delegation, no approvals
    "read-only": {"memory.read"},
    # Can approve and write memory, but cannot delegate
    "no-delegation": {"approve", "memory.read", "memory.write"},
    # Can delegate and write memory, but cannot invoke approval gate
    "no-approval": {"delegate", "memory.read", "memory.write"},
    # Memory reads only (useful for analytic sidecars)
    "memory-readonly": {"memory.read"},
}


# ---------------------------------------------------------------------------
# Config loader (lazy, cached per process)
# ---------------------------------------------------------------------------

@functools.lru_cache(maxsize=1)
def _load_workspace_config():
    """Return the WorkspaceConfig or None if it cannot be loaded."""
    try:
        from config import load_config  # local import avoids circular deps
        return load_config()
    except Exception as exc:
        logger.warning("audit: could not load workspace config for RBAC: %s", exc)
        return None


def get_workspace_roles() -> tuple[list[str], dict[str, list[str]]]:
    """Return ``(roles, custom_permissions)`` from the workspace config.

    Falls back to ``["operator"]`` / ``{}`` when the config is unavailable so
    that agents remain functional in degraded environments.
    """
    cfg = _load_workspace_config()
    if cfg is None:
        return ["operator"], {}
    return list(cfg.rbac.roles), dict(cfg.rbac.allowed_actions)


# ---------------------------------------------------------------------------
# RBAC helpers
# ---------------------------------------------------------------------------

def check_permission(
    action: str,
    roles: list[str],
    custom_permissions: dict[str, list[str]] | None = None,
) -> bool:
    """Return True if *any* of ``roles`` grants ``action``.

    Evaluation order
    ~~~~~~~~~~~~~~~~
    1. ``"admin"`` shortcircuits — always grants everything.
    2. Custom role definitions (from ``rbac.allowed_actions`` in config.yaml).
    3. Built-in :data:`ROLE_PERMISSIONS` table.

    When a role appears in *custom_permissions* its built-in definition is
    **ignored** — the custom list is the complete permission set for that role.

    Args:
        action:             Action to authorise, e.g. ``"delegate"``.
        roles:              Roles assigned to the calling workspace.
        custom_permissions: Optional ``{role: [action, ...]}`` mapping loaded
                            from ``WorkspaceConfig.rbac.allowed_actions``.

    Returns:
        ``True`` if the action is permitted, ``False`` otherwise.

    Examples::

        >>> check_permission("delegate", ["operator"])
        True
        >>> check_permission("delegate", ["read-only"])
        False
        >>> check_permission("deploy", ["developer"], {"developer": ["deploy"]})
        True
    """
    for role in roles:
        if role == "admin":
            return True
        if custom_permissions and role in custom_permissions:
            # Custom entry is definitive for this role
            if action in custom_permissions[role]:
                return True
            continue  # Don't fall through to built-ins for custom roles
        if role in ROLE_PERMISSIONS and action in ROLE_PERMISSIONS[role]:
            return True
    return False


# ---------------------------------------------------------------------------
# Public audit API
# ---------------------------------------------------------------------------

def log_event(
    event_type: str,
    action: str,
    resource: str,
    outcome: str,
    actor: str | None = None,
    trace_id: str | None = None,
    **extra: Any,
) -> str:
    """Append one audit event to the immutable JSON Lines log.

    Args:
        event_type: Coarse category — ``"delegation"``, ``"approval"``,
                    ``"memory"``, or ``"rbac"``.
        action:     Verb — ``"delegate"``, ``"approve"``, ``"memory.write"``,
                    ``"memory.read"``, ``"rbac.deny"``.
        resource:   Object of the action — target workspace ID, memory scope,
                    approval action string, etc.
        outcome:    Terminal state — one of ``"allowed"``, ``"denied"``,
                    ``"success"``, ``"failure"``, ``"timeout"``,
                    ``"requested"``, ``"granted"``.
        actor:      Identity that triggered the event.  Defaults to
                    ``WORKSPACE_ID`` (the running workspace) for automated
                    events.  Pass ``decided_by`` for human approval decisions.
        trace_id:   Caller-supplied UUID v4 for cross-event correlation.
                    A fresh UUID is generated when omitted.
        **extra:    Additional key-value pairs appended verbatim to the JSON
                    object (e.g. ``target_workspace_id``, ``memory_scope``,
                    ``attempt``).  Built-in keys cannot be overridden.

    Returns:
        The ``trace_id`` used for this event, enabling callers to chain
        related events under a single correlation identifier.

    Example::

        trace = log_event(
            event_type="delegation",
            action="delegate",
            resource="billing-agent",
            outcome="success",
            target_workspace_id="billing-agent",
            attempt=1,
        )
    """
    if trace_id is None:
        trace_id = str(uuid.uuid4())

    event: dict[str, Any] = {
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "event_type": event_type,
        "workspace_id": WORKSPACE_ID,
        "actor": actor if actor is not None else WORKSPACE_ID,
        "action": action,
        "resource": resource,
        "outcome": outcome,
        "trace_id": trace_id,
    }

    # Merge extra fields — built-in keys are not overridable
    for key, value in extra.items():
        if key not in event:
            event[key] = value

    _write_event(event)
    return trace_id


# ---------------------------------------------------------------------------
# Internal writer
# ---------------------------------------------------------------------------

def _ensure_log_dir(path: str) -> None:
    """Create the parent directory for *path* if it does not already exist."""
    Path(path).parent.mkdir(parents=True, exist_ok=True)


def _write_event(event: dict[str, Any]) -> None:
    """Serialise *event* as a JSON line and fsync-append it to the log file.

    The write is atomic with respect to other threads in this process: the
    lock ensures that no two JSON objects are interleaved on the same line.

    Failures are emitted to the standard Python logger at WARNING level but
    are **never** re-raised — the application must not crash because audit
    logging is temporarily unavailable (e.g. disk full, permission error).
    In production, consider wiring an alert on WARNING messages from this
    module so that missing audit records are detected quickly.
    """
    try:
        log_path = AUDIT_LOG_PATH
        _ensure_log_dir(log_path)
        line = json.dumps(event, default=str, ensure_ascii=False) + "\n"
        with _write_lock:
            with open(log_path, "a", encoding="utf-8") as fh:
                fh.write(line)
                fh.flush()
                os.fsync(fh.fileno())
    except Exception as exc:  # pylint: disable=broad-except
        logger.warning(
            "Audit log write failed — event NOT persisted "
            "(trace_id=%s, action=%s): %s",
            event.get("trace_id", "?"),
            event.get("action", "?"),
            exc,
        )
