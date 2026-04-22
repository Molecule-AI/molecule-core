"""Bridge between Molecule AI's RBAC + audit subsystem and the Microsoft Agent
Governance Toolkit (agent-os-kernel, released April 2, 2026).

Integration points
------------------
* ``check_permission`` → ``PolicyEvaluator.evaluate()``
  Molecule AI's RBAC gate runs first; if RBAC allows the action the toolkit
  evaluator is consulted according to ``policy_mode``.

* ``log_event`` → governance audit sink
  Every permission decision (allow or deny) is written via
  ``tools.audit.log_event`` with extra governance metadata so the full
  decision trail lands in Molecule AI's existing audit stream.

* OTEL traceparent flows through
  ``tools.telemetry.get_current_traceparent()`` is called inside ``emit()``
  and the W3C traceparent string is attached to every audit record, giving
  end-to-end distributed tracing across agent boundaries.

Graceful degradation
--------------------
If ``agent-os-kernel`` is not installed the module falls back to Molecule AI
RBAC alone.  No exception propagates to the agent — governance is a
best-effort overlay, never a hard dependency.

Install::

    pip install agent-os-kernel

Minimal config.yaml snippet::

    governance:
      enabled: true
      toolkit: microsoft
      policy_mode: strict          # strict | permissive | audit
      policy_endpoint: https://your-tenant.governance.azure.com
      policy_file: policies/workspace.rego
      blocked_patterns:
        - ".*\\.exec$"
        - "shell\\."
      max_tool_calls_per_task: 50

NOTE: The agent-os-kernel package was released April 2, 2026 and is in
community preview.  The API bindings in this module target v3.0.x of the
package (agent_os.policies.PolicyEvaluator).  If the package API changes,
update _init_evaluator() accordingly.
"""

import logging
import os
from typing import Any, Optional

logger = logging.getLogger(__name__)
_WS_ID_RAW = os.environ.get("WORKSPACE_ID")
if not _WS_ID_RAW:
    raise RuntimeError("WORKSPACE_ID environment variable is required but not set")
WORKSPACE_ID: str = _WS_ID_RAW

# Module-level singleton — set by initialize_governance() at startup
_adapter: Optional["GovernanceAdapter"] = None


class GovernanceAdapter:
    """Bridges Molecule AI RBAC + audit trail to the Microsoft Agent Governance Toolkit."""

    def __init__(self, config: Any) -> None:
        self._config = config
        self._evaluator = None
        self._toolkit_available: bool = False

    async def initialize(self) -> None:
        """Async entry point: initialise evaluator and log outcome."""
        self._init_evaluator()
        if self._toolkit_available:
            logger.info(
                "GovernanceAdapter initialised — toolkit=%s mode=%s",
                self._config.toolkit,
                self._config.policy_mode,
            )
        else:
            logger.warning(
                "GovernanceAdapter initialised in RBAC-only mode "
                "(agent-os-kernel not available or failed to load)."
            )

    def _init_evaluator(self) -> None:
        """Lazy-import and configure the PolicyEvaluator from agent-os-kernel.

        All failures are caught and logged; the adapter simply runs without
        the toolkit rather than crashing the workspace.
        """
        try:
            try:
                from agent_os.policies import PolicyEvaluator  # type: ignore[import]
            except ImportError:
                logger.warning(
                    "agent-os-kernel is not installed — graceful degradation active. "
                    "Governance will use Molecule AI RBAC only. "
                    "To enable the Microsoft Agent Governance Toolkit run: "
                    "pip install agent-os-kernel"
                )
                return

            kwargs: dict[str, Any] = {
                "policy_mode": self._config.policy_mode,
                "max_tool_calls_per_task": self._config.max_tool_calls_per_task,
                "blocked_patterns": self._config.blocked_patterns,
            }
            if self._config.policy_endpoint:
                kwargs["endpoint"] = self._config.policy_endpoint

            self._evaluator = PolicyEvaluator(**kwargs)

            # Load a policy file if one is configured and exists on disk.
            if self._config.policy_file:
                policy_file = self._config.policy_file
                if os.path.exists(policy_file):
                    ext = os.path.splitext(policy_file)[1].lower()
                    if ext == ".rego":
                        self._evaluator.load_rego(path=policy_file)
                        logger.info("Loaded Rego policy file: %s", policy_file)
                    elif ext in (".yaml", ".yml"):
                        self._evaluator.load_yaml(path=policy_file)
                        logger.info("Loaded YAML policy file: %s", policy_file)
                    elif ext == ".cedar":
                        self._evaluator.load_cedar(path=policy_file)
                        logger.info("Loaded Cedar policy file: %s", policy_file)
                    else:
                        logger.warning(
                            "Unrecognised policy file extension '%s' — skipping load.",
                            ext,
                        )
                else:
                    logger.warning(
                        "policy_file '%s' does not exist — skipping load.",
                        policy_file,
                    )

            self._toolkit_available = True
            logger.info(
                "agent-os-kernel PolicyEvaluator ready — policy_mode=%s",
                self._config.policy_mode,
            )

        except Exception as exc:  # noqa: BLE001
            logger.warning(
                "Failed to initialise agent-os-kernel PolicyEvaluator: %s — "
                "graceful degradation active (RBAC only).",
                exc,
            )

    def check_permission(
        self,
        action: str,
        roles: list[str],
        custom_permissions: dict | None = None,
        context: dict | None = None,
    ) -> tuple[bool, str]:
        """Evaluate an action against Molecule AI RBAC and (optionally) the toolkit.

        Returns
        -------
        tuple[bool, str]
            ``(allowed, reason)`` — reason is a short human-readable string
            explaining the decision.
        """
        from builtin_tools import audit  # inline import to avoid circular dependencies

        context = context or {}

        # --- Step 1: Molecule AI RBAC gate (always runs) ---
        rbac_allowed: bool = audit.check_permission(action, roles, custom_permissions)

        if not rbac_allowed:
            self.emit(
                event_type="permission_check",
                action=action,
                resource=context.get("resource", ""),
                outcome="denied",
                actor=context.get("actor"),
                policy_decision="rbac_deny",
                roles=roles,
            )
            return False, f"RBAC denied action '{action}' for roles {roles}"

        # --- Step 2: If toolkit unavailable or audit-only mode, return RBAC result ---
        if not self._toolkit_available or self._config.policy_mode == "audit":
            self.emit(
                event_type="permission_check",
                action=action,
                resource=context.get("resource", ""),
                outcome="allowed",
                actor=context.get("actor"),
                policy_decision="rbac_allowed",
                roles=roles,
                toolkit_mode=self._config.policy_mode,
            )
            return rbac_allowed, "rbac_allowed"

        # --- Step 3: Toolkit evaluation ---
        eval_context: dict[str, Any] = {
            "action": action,
            "resource": context.get("resource", ""),
            "roles": roles,
            "workspace_id": WORKSPACE_ID,
        }
        # Merge any extra context keys the caller supplied.
        for key, value in context.items():
            if key not in eval_context:
                eval_context[key] = value

        toolkit_allowed: bool = True
        reason: str = ""
        evaluator_name: str = "agent-os-kernel"

        try:
            decision = self._evaluator.evaluate(eval_context)
            toolkit_allowed = getattr(decision, "allowed", True)
            reason = getattr(decision, "reason", "")
            evaluator_name = getattr(decision, "evaluator_name", "agent-os-kernel")
        except Exception as exc:  # noqa: BLE001
            logger.warning(
                "agent-os-kernel evaluation raised an exception: %s — "
                "falling back to RBAC result to avoid blocking the agent.",
                exc,
            )
            self.emit(
                event_type="permission_check",
                action=action,
                resource=context.get("resource", ""),
                outcome="allowed",
                actor=context.get("actor"),
                policy_decision="toolkit_evaluation_error",
                toolkit_mode=self._config.policy_mode,
                roles=roles,
            )
            return rbac_allowed, "toolkit_evaluation_error"

        # --- Step 4: Combine results according to policy_mode ---
        if self._config.policy_mode == "permissive":
            # Toolkit denial is advisory only in permissive mode.
            if not toolkit_allowed:
                logger.warning(
                    "Governance toolkit denied action '%s' (reason=%s) but policy_mode "
                    "is 'permissive' — allowing and logging advisory denial.",
                    action,
                    reason,
                )
            final_allowed = rbac_allowed
        else:
            # strict: both gates must allow.
            final_allowed = rbac_allowed and toolkit_allowed

        outcome = "allowed" if final_allowed else "denied"
        self.emit(
            event_type="permission_check",
            action=action,
            resource=context.get("resource", ""),
            outcome=outcome,
            actor=context.get("actor"),
            policy_decision=reason or outcome,
            evaluator=evaluator_name,
            toolkit_mode=self._config.policy_mode,
            roles=roles,
        )
        return final_allowed, reason or "allowed"

    def emit(
        self,
        event_type: str,
        action: str,
        resource: str,
        outcome: str,
        actor: str | None = None,
        trace_id: str | None = None,
        **extra: Any,
    ) -> str:
        """Write a governance-annotated audit event.

        Pulls the current W3C traceparent from the active OTEL span so that
        governance decisions are traceable across service boundaries.

        Returns
        -------
        str
            The ``trace_id`` produced by ``audit.log_event``.
        """
        from builtin_tools import audit  # inline import to avoid circular dependencies
        from builtin_tools.telemetry import get_current_traceparent  # inline import

        traceparent: str | None = get_current_traceparent()

        recorded_trace_id: str = audit.log_event(
            event_type,
            action,
            resource,
            outcome,
            actor=actor,
            trace_id=trace_id,
            governance_toolkit=(
                self._config.toolkit if self._toolkit_available else "disabled"
            ),
            traceparent=traceparent or "",
            **extra,
        )
        return recorded_trace_id


# ---------------------------------------------------------------------------
# Module-level functions
# ---------------------------------------------------------------------------


async def initialize_governance(config: Any) -> Optional[GovernanceAdapter]:
    """Initialize the module-level GovernanceAdapter singleton.

    Called once at startup by main.py when governance.enabled is True.
    Returns the adapter, or None if initialization fails.
    """
    global _adapter

    try:
        adapter = GovernanceAdapter(config)
        await adapter.initialize()
        _adapter = adapter
        logger.info(
            "Governance singleton initialised — toolkit=%s mode=%s",
            config.toolkit,
            config.policy_mode,
        )
        return adapter
    except Exception as exc:  # noqa: BLE001
        logger.warning(
            "initialize_governance() failed: %s — governance disabled for this session.",
            exc,
        )
        return None


def get_governance_adapter() -> Optional[GovernanceAdapter]:
    """Return the module-level GovernanceAdapter singleton (may be None)."""
    return _adapter


def check_permission_with_governance(
    action: str,
    roles: list[str],
    custom_permissions: dict | None = None,
    context: dict | None = None,
) -> tuple[bool, str]:
    """Convenience wrapper: use GovernanceAdapter when available, else RBAC only.

    Parameters
    ----------
    action:
        The action name to evaluate (e.g. ``"memory.write"``).
    roles:
        The list of role names held by the requesting actor.
    custom_permissions:
        Optional custom role→action mapping to overlay on built-in roles.
    context:
        Optional extra context forwarded to the PolicyEvaluator.

    Returns
    -------
    tuple[bool, str]
        ``(allowed, reason)``
    """
    if _adapter is None:
        from builtin_tools import audit  # inline import to avoid circular dependencies

        result: bool = audit.check_permission(action, roles, custom_permissions)
        return result, "rbac_only"

    return _adapter.check_permission(action, roles, custom_permissions, context)


# ---------------------------------------------------------------------------
# Private helper
# ---------------------------------------------------------------------------


def _emit_governance_event(
    event_type: str,
    action: str,
    resource: str,
    outcome: str,
    actor: str | None = None,
    trace_id: str | None = None,
    **extra: Any,
) -> Optional[str]:
    """Emit a governance audit event via the singleton adapter if one is set.

    Returns the trace_id produced by log_event, or None if no adapter is set.
    """
    if _adapter is None:
        return None
    return _adapter.emit(
        event_type,
        action,
        resource,
        outcome,
        actor=actor,
        trace_id=trace_id,
        **extra,
    )
