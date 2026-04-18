"""molecule_audit — HMAC-SHA256-chained immutable agent event log.

EU AI Act Annex III compliance (Art. 12/13 record-keeping, Art. 17 quality
management) for high-risk AI systems.

Quick start
-----------
    from molecule_audit.hooks import LedgerHooks

    with LedgerHooks(session_id=task_id) as hooks:
        hooks.on_task_start(input_text=user_prompt)
        # ... call LLM / tools ...
        hooks.on_llm_call(model="hermes-3", output_text=reply)
        hooks.on_task_end(output_text=result)

Verify a chain
--------------
    python -m molecule_audit.verify --agent-id <id>
"""

from .ledger import AuditEvent, append_event, get_engine, verify_chain
from .hooks import LedgerHooks

__all__ = ["AuditEvent", "append_event", "get_engine", "verify_chain", "LedgerHooks"]
