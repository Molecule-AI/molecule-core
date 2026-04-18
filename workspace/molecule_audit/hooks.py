"""molecule_audit.hooks — Pipeline hook registrations for the audit ledger.

Registers audit events at four EU AI Act Art. 12 pipeline checkpoints:
  task_start  — an A2A task begins execution
  llm_call    — a model inference call is made (records model name)
  tool_call   — a tool/function is invoked (records tool name in model_used)
  task_end    — a task completes (success or failure)

Usage
-----
The recommended pattern is to create a LedgerHooks instance at the start of
each task and use it as a context manager:

    from molecule_audit.hooks import LedgerHooks

    with LedgerHooks(session_id=task_id, agent_id=agent_id) as hooks:
        hooks.on_task_start(input_text=user_prompt)
        response = call_llm(model="hermes-4", prompt=user_prompt)
        hooks.on_llm_call(model="hermes-4", input_text=user_prompt,
                          output_text=response)
        result = run_tool("search", query=user_prompt)
        hooks.on_tool_call("search", input_data=user_prompt, output_data=result)
        hooks.on_task_end(output_text=result)

All hook methods swallow exceptions so that audit failures never block the
agent pipeline.  Failures are emitted at WARNING level.

Privacy note
------------
Raw input/output text is never persisted.  All on_* methods take plaintext
for convenience and immediately hash it with SHA-256 via hash_content().
Only the hex digest is stored in the ledger.
"""

from __future__ import annotations

import json
import logging
import os
from typing import Any

from .ledger import append_event, get_session_factory, hash_content

logger = logging.getLogger(__name__)

# Default agent identity — set by the platform when launching a workspace container.
_DEFAULT_AGENT_ID: str = os.environ.get("WORKSPACE_ID", "unknown-agent")


class LedgerHooks:
    """Lifecycle hooks that write signed events to the audit ledger.

    Parameters
    ----------
    session_id:            Task / conversation ID (gen_ai.conversation.id).
                           Required — must be unique per agent session.
    agent_id:              Identity of this agent.
                           Defaults to the WORKSPACE_ID env var.
    db_url:                SQLAlchemy URL override — useful in tests to point at
                           an in-memory SQLite DB (``"sqlite:///:memory:"``).
    human_oversight_flag:  Default oversight flag written on task_start / task_end.
                           Can be overridden per call.
    """

    def __init__(
        self,
        session_id: str,
        agent_id: str | None = None,
        db_url: str | None = None,
        human_oversight_flag: bool = False,
    ) -> None:
        self.agent_id: str = agent_id or _DEFAULT_AGENT_ID
        self.session_id: str = session_id
        self._db_url: str | None = db_url
        self._default_human_oversight: bool = human_oversight_flag
        self._session = None

    # ------------------------------------------------------------------
    # Session management
    # ------------------------------------------------------------------

    def _open_session(self):
        """Return a lazily-opened SQLAlchemy session (cached for this instance)."""
        if self._session is None:
            factory = get_session_factory(self._db_url)
            self._session = factory()
        return self._session

    def close(self) -> None:
        """Release the underlying SQLAlchemy session."""
        if self._session is not None:
            self._session.close()
            self._session = None

    def __enter__(self) -> "LedgerHooks":
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        self.close()

    # ------------------------------------------------------------------
    # Four pipeline hook points (EU AI Act Art. 12)
    # ------------------------------------------------------------------

    def on_task_start(
        self,
        input_text: str | None = None,
        human_oversight_flag: bool | None = None,
        risk_flag: bool = False,
    ) -> None:
        """Log ``operation=task_start`` when an agent task begins.

        Parameters
        ----------
        input_text:            Raw user / caller input (hashed before storage).
        human_oversight_flag:  Override the instance-level default.
        risk_flag:             Set True when the input triggers a risk condition.
        """
        self._safe_append(
            operation="task_start",
            input_hash=hash_content(input_text),
            human_oversight_flag=(
                human_oversight_flag
                if human_oversight_flag is not None
                else self._default_human_oversight
            ),
            risk_flag=risk_flag,
        )

    def on_llm_call(
        self,
        model: str,
        input_text: str | None = None,
        output_text: str | None = None,
        risk_flag: bool = False,
    ) -> None:
        """Log ``operation=llm_call`` when a model inference call is made.

        Parameters
        ----------
        model:       Model identifier (e.g. ``"hermes-4-405b"``).
        input_text:  Prompt / messages sent to the model (hashed).
        output_text: Model response text (hashed).
        risk_flag:   Set True when the response triggers a risk condition.
        """
        self._safe_append(
            operation="llm_call",
            input_hash=hash_content(input_text),
            output_hash=hash_content(output_text),
            model_used=model,
            risk_flag=risk_flag,
        )

    def on_tool_call(
        self,
        tool_name: str,
        input_data: Any = None,
        output_data: Any = None,
        risk_flag: bool = False,
    ) -> None:
        """Log ``operation=tool_call`` when a tool/function is invoked.

        Parameters
        ----------
        tool_name:   Name of the tool or function (stored in ``model_used``).
        input_data:  Tool input — str, bytes, or JSON-serializable object (hashed).
        output_data: Tool output — same type options (hashed).
        risk_flag:   Set True when the tool result triggers a risk condition.
        """
        self._safe_append(
            operation="tool_call",
            input_hash=hash_content(_to_bytes(input_data)),
            output_hash=hash_content(_to_bytes(output_data)),
            model_used=tool_name,
            risk_flag=risk_flag,
        )

    def on_task_end(
        self,
        output_text: str | None = None,
        human_oversight_flag: bool | None = None,
        risk_flag: bool = False,
    ) -> None:
        """Log ``operation=task_end`` when a task completes.

        Parameters
        ----------
        output_text:           Final task output / result (hashed before storage).
        human_oversight_flag:  Override the instance-level default.
        risk_flag:             Set True when the final result triggers a risk condition.
        """
        self._safe_append(
            operation="task_end",
            output_hash=hash_content(output_text),
            human_oversight_flag=(
                human_oversight_flag
                if human_oversight_flag is not None
                else self._default_human_oversight
            ),
            risk_flag=risk_flag,
        )

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _safe_append(self, **kwargs) -> None:
        """Append an audit event, swallowing all exceptions.

        Audit failures must never block the agent pipeline.  All errors are
        logged at WARNING level so operators can detect gaps in the log.
        """
        try:
            append_event(
                agent_id=self.agent_id,
                session_id=self.session_id,
                db_session=self._open_session(),
                **kwargs,
            )
        except Exception as exc:
            logger.warning(
                "audit: failed to append event "
                "(agent=%s session=%s op=%s): %s",
                self.agent_id,
                self.session_id,
                kwargs.get("operation", "?"),
                exc,
            )


# ---------------------------------------------------------------------------
# Private helpers
# ---------------------------------------------------------------------------

def _to_bytes(value: Any) -> bytes | None:
    """Convert a value to bytes for hashing; returns None for None."""
    if value is None:
        return None
    if isinstance(value, bytes):
        return value
    if isinstance(value, str):
        return value.encode("utf-8")
    # JSON-serializable objects (dicts, lists, etc.)
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode("utf-8")
