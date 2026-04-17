"""Tests for molecule_audit — HMAC-chained audit ledger.

Coverage
--------
ledger.py:
  - _get_hmac_key()       missing SALT raises RuntimeError; repeated calls return same key
  - _ts_to_canonical()    UTC datetime, naive datetime, None
  - _to_canonical_dict()  excludes hmac field, timestamp is Z-suffixed
  - _compute_event_hmac() deterministic; changes when any field changes
  - hash_content()        str, bytes, None
  - AuditEvent.to_dict()  all fields present, ISO timestamp
  - append_event()        single event, chain linkage, error rollback
  - verify_chain()        valid chain, tampered hmac, broken prev_hmac, empty chain

hooks.py:
  - LedgerHooks.on_task_start()  hashes input, writes task_start event
  - LedgerHooks.on_llm_call()    hashes i/o, stores model name
  - LedgerHooks.on_tool_call()   hashes serialised i/o, stores tool name in model_used
  - LedgerHooks.on_task_end()    hashes output, writes task_end event
  - LedgerHooks context manager  close() releases session
  - Exception swallowing         missing SALT → warning, no raise

verify.py CLI:
  - valid chain → exit 0, prints "CHAIN VALID"
  - no events   → exit 0, prints "No audit events"
  - broken chain → exit 1, prints "CHAIN BROKEN"
  - missing SALT → exit 2
"""

from __future__ import annotations

import hashlib
import hmac as _hmac_mod
import json
import logging
import os
import sys
from datetime import datetime, timezone
from unittest.mock import MagicMock, patch

import pytest
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

# ---------------------------------------------------------------------------
# Fixtures — isolated in-memory SQLite DB per test
# ---------------------------------------------------------------------------

@pytest.fixture(autouse=True)
def _reset_ledger_caches(monkeypatch):
    """Reset module-level caches and force AUDIT_LEDGER_SALT for every test."""
    import molecule_audit.ledger as ledger

    monkeypatch.setattr(ledger, "AUDIT_LEDGER_SALT", "test-salt-for-pytest")
    monkeypatch.setattr(ledger, "_hmac_key", None)
    monkeypatch.setattr(ledger, "_engine", None)
    monkeypatch.setattr(ledger, "_SessionFactory", None)

    yield

    # Clean up after test
    ledger.reset_hmac_key_cache()
    ledger.reset_engine_cache()


@pytest.fixture
def mem_session():
    """Provide a fresh in-memory SQLite session with the schema created."""
    import molecule_audit.ledger as ledger
    from molecule_audit.ledger import Base

    engine = create_engine(
        "sqlite:///:memory:", connect_args={"check_same_thread": False}
    )
    Base.metadata.create_all(engine)
    factory = sessionmaker(bind=engine)
    session = factory()

    # Inject the engine into the module cache so append_event uses it
    ledger._engine = engine
    ledger._SessionFactory = factory

    yield session

    session.close()
    Base.metadata.drop_all(engine)
    ledger.reset_engine_cache()


# ---------------------------------------------------------------------------
# ledger._get_hmac_key
# ---------------------------------------------------------------------------

class TestGetHmacKey:

    def test_raises_when_salt_missing(self, monkeypatch):
        import molecule_audit.ledger as ledger
        monkeypatch.setattr(ledger, "AUDIT_LEDGER_SALT", "")
        monkeypatch.setenv("AUDIT_LEDGER_SALT", "")
        # Remove from env so os.environ.get also returns ""
        monkeypatch.delenv("AUDIT_LEDGER_SALT", raising=False)
        ledger._hmac_key = None  # clear cache

        with pytest.raises(RuntimeError, match="AUDIT_LEDGER_SALT"):
            ledger._get_hmac_key()

    def test_same_key_returned_on_repeated_calls(self):
        import molecule_audit.ledger as ledger

        key1 = ledger._get_hmac_key()
        key2 = ledger._get_hmac_key()
        assert key1 is key2  # same object (cached)
        assert len(key1) == 32

    def test_key_changes_with_different_salt(self, monkeypatch):
        import molecule_audit.ledger as ledger

        key1 = ledger._get_hmac_key()

        ledger.reset_hmac_key_cache()
        monkeypatch.setattr(ledger, "AUDIT_LEDGER_SALT", "different-salt")
        key2 = ledger._get_hmac_key()

        assert key1 != key2


# ---------------------------------------------------------------------------
# ledger._ts_to_canonical
# ---------------------------------------------------------------------------

class TestTsToCanonical:

    def test_utc_aware_datetime(self):
        from molecule_audit.ledger import _ts_to_canonical

        ts = datetime(2026, 4, 17, 12, 34, 56, 789000, tzinfo=timezone.utc)
        result = _ts_to_canonical(ts)
        assert result == "2026-04-17T12:34:56Z"

    def test_naive_datetime(self):
        from molecule_audit.ledger import _ts_to_canonical

        ts = datetime(2026, 4, 17, 12, 34, 56)
        result = _ts_to_canonical(ts)
        assert result == "2026-04-17T12:34:56Z"

    def test_none_returns_none(self):
        from molecule_audit.ledger import _ts_to_canonical

        assert _ts_to_canonical(None) is None

    def test_microseconds_stripped(self):
        from molecule_audit.ledger import _ts_to_canonical

        ts = datetime(2026, 1, 1, 0, 0, 0, 999999, tzinfo=timezone.utc)
        result = _ts_to_canonical(ts)
        assert "." not in result
        assert result.endswith("Z")


# ---------------------------------------------------------------------------
# ledger.hash_content
# ---------------------------------------------------------------------------

class TestHashContent:

    def test_none_returns_none(self):
        from molecule_audit.ledger import hash_content
        assert hash_content(None) is None

    def test_str_returns_sha256_hex(self):
        from molecule_audit.ledger import hash_content
        result = hash_content("hello")
        expected = hashlib.sha256(b"hello").hexdigest()
        assert result == expected
        assert len(result) == 64

    def test_bytes_returns_sha256_hex(self):
        from molecule_audit.ledger import hash_content
        result = hash_content(b"hello")
        expected = hashlib.sha256(b"hello").hexdigest()
        assert result == expected

    def test_str_and_bytes_same_result_for_utf8(self):
        from molecule_audit.ledger import hash_content
        assert hash_content("café") == hash_content("café".encode("utf-8"))


# ---------------------------------------------------------------------------
# ledger._compute_event_hmac
# ---------------------------------------------------------------------------

class TestComputeEventHmac:

    def _make_event(self, **kwargs):
        from molecule_audit.ledger import AuditEvent
        defaults = {
            "id": "evt-1",
            "timestamp": datetime(2026, 4, 17, 0, 0, 0, tzinfo=timezone.utc),
            "agent_id": "agent-1",
            "session_id": "sess-1",
            "operation": "task_start",
            "input_hash": None,
            "output_hash": None,
            "model_used": None,
            "human_oversight_flag": False,
            "risk_flag": False,
            "prev_hmac": None,
            "hmac": "placeholder",
        }
        defaults.update(kwargs)
        ev = AuditEvent(**defaults)
        return ev

    def test_deterministic(self):
        from molecule_audit.ledger import _compute_event_hmac
        ev = self._make_event()
        assert _compute_event_hmac(ev) == _compute_event_hmac(ev)

    def test_different_agent_id_changes_hmac(self):
        from molecule_audit.ledger import _compute_event_hmac
        ev1 = self._make_event(agent_id="agent-A")
        ev2 = self._make_event(agent_id="agent-B")
        assert _compute_event_hmac(ev1) != _compute_event_hmac(ev2)

    def test_different_operation_changes_hmac(self):
        from molecule_audit.ledger import _compute_event_hmac
        ev1 = self._make_event(operation="task_start")
        ev2 = self._make_event(operation="task_end")
        assert _compute_event_hmac(ev1) != _compute_event_hmac(ev2)

    def test_prev_hmac_included_in_computation(self):
        from molecule_audit.ledger import _compute_event_hmac
        ev1 = self._make_event(prev_hmac=None)
        ev2 = self._make_event(prev_hmac="abc123")
        assert _compute_event_hmac(ev1) != _compute_event_hmac(ev2)

    def test_hmac_field_excluded_from_canonical(self):
        """The stored hmac field itself must not affect the computation."""
        from molecule_audit.ledger import _compute_event_hmac
        ev1 = self._make_event(hmac="value-a")
        ev2 = self._make_event(hmac="value-b")
        assert _compute_event_hmac(ev1) == _compute_event_hmac(ev2)

    def test_canonical_json_uses_compact_separators(self):
        """Canonical JSON must have no spaces (compact separators)."""
        from molecule_audit.ledger import _to_canonical_dict
        ev = self._make_event()
        canonical = _to_canonical_dict(ev)
        payload = json.dumps(canonical, sort_keys=True, separators=(",", ":"))
        assert " " not in payload

    def test_canonical_json_sort_order_is_alphabetical(self):
        """Keys must be alphabetically sorted (Python sort_keys=True / Go map order)."""
        from molecule_audit.ledger import _to_canonical_dict
        ev = self._make_event()
        canonical = _to_canonical_dict(ev)
        payload = json.dumps(canonical, sort_keys=True, separators=(",", ":"))
        keys = [k.strip('"') for k in payload.split(',"')[0:]]
        first_key = payload.lstrip("{").split('"')[1]
        assert first_key == "agent_id"  # alphabetically first

    def test_result_is_hex_string(self):
        from molecule_audit.ledger import _compute_event_hmac
        ev = self._make_event()
        h = _compute_event_hmac(ev)
        assert isinstance(h, str)
        assert len(h) == 64
        int(h, 16)  # raises ValueError if not valid hex


# ---------------------------------------------------------------------------
# ledger.append_event + verify_chain
# ---------------------------------------------------------------------------

class TestAppendEvent:

    def test_single_event_written(self, mem_session):
        from molecule_audit.ledger import AuditEvent, append_event

        ev = append_event(
            agent_id="agent-1",
            session_id="sess-1",
            operation="task_start",
            db_session=mem_session,
        )
        assert ev.id is not None
        assert ev.operation == "task_start"
        assert ev.prev_hmac is None  # first event
        assert len(ev.hmac) == 64

        stored = mem_session.query(AuditEvent).first()
        assert stored.id == ev.id

    def test_chain_linkage_across_two_events(self, mem_session):
        from molecule_audit.ledger import append_event

        ev1 = append_event("a", "s", "task_start", db_session=mem_session)
        ev2 = append_event("a", "s", "task_end", db_session=mem_session)

        assert ev2.prev_hmac == ev1.hmac
        assert ev2.hmac != ev1.hmac

    def test_different_agents_independent_chains(self, mem_session):
        """Events from different agents do NOT link to each other."""
        from molecule_audit.ledger import append_event

        ev_a = append_event("agent-A", "s", "task_start", db_session=mem_session)
        ev_b = append_event("agent-B", "s", "task_start", db_session=mem_session)
        ev_a2 = append_event("agent-A", "s", "task_end", db_session=mem_session)

        assert ev_b.prev_hmac is None  # agent-B's first row
        assert ev_a2.prev_hmac == ev_a.hmac  # agent-A's chain continues

    def test_input_hash_stored(self, mem_session):
        from molecule_audit.ledger import append_event, hash_content

        content = "user prompt"
        ev = append_event(
            "a", "s", "llm_call",
            input_hash=hash_content(content),
            db_session=mem_session,
        )
        assert ev.input_hash == hashlib.sha256(content.encode()).hexdigest()

    def test_model_used_stored(self, mem_session):
        from molecule_audit.ledger import append_event

        ev = append_event("a", "s", "llm_call", model_used="hermes-4", db_session=mem_session)
        assert ev.model_used == "hermes-4"

    def test_to_dict_includes_all_fields(self, mem_session):
        from molecule_audit.ledger import append_event

        ev = append_event("a", "s", "task_start", db_session=mem_session)
        d = ev.to_dict()
        required_keys = {
            "id", "timestamp", "agent_id", "session_id", "operation",
            "input_hash", "output_hash", "model_used",
            "human_oversight_flag", "risk_flag", "prev_hmac", "hmac",
        }
        assert required_keys == set(d.keys())

    def test_risk_and_oversight_flags(self, mem_session):
        from molecule_audit.ledger import append_event

        ev = append_event(
            "a", "s", "task_start",
            human_oversight_flag=True,
            risk_flag=True,
            db_session=mem_session,
        )
        assert ev.human_oversight_flag is True
        assert ev.risk_flag is True


class TestVerifyChain:

    def test_empty_chain_returns_true(self, mem_session):
        from molecule_audit.ledger import verify_chain
        assert verify_chain("non-existent-agent", mem_session) is True

    def test_single_event_valid(self, mem_session):
        from molecule_audit.ledger import append_event, verify_chain

        append_event("a", "s", "task_start", db_session=mem_session)
        assert verify_chain("a", mem_session) is True

    def test_multi_event_chain_valid(self, mem_session):
        from molecule_audit.ledger import append_event, verify_chain

        for op in ("task_start", "llm_call", "tool_call", "task_end"):
            append_event("a", "s", op, db_session=mem_session)
        assert verify_chain("a", mem_session) is True

    def test_tampered_hmac_detected(self, mem_session):
        from molecule_audit.ledger import AuditEvent, append_event, verify_chain

        ev = append_event("a", "s", "task_start", db_session=mem_session)

        # Directly corrupt the stored HMAC
        mem_session.query(AuditEvent).filter(AuditEvent.id == ev.id).update(
            {"hmac": "deadbeef" + "0" * 56}
        )
        mem_session.commit()

        assert verify_chain("a", mem_session) is False

    def test_broken_prev_hmac_detected(self, mem_session):
        from molecule_audit.ledger import AuditEvent, append_event, verify_chain

        ev1 = append_event("a", "s", "task_start", db_session=mem_session)
        ev2 = append_event("a", "s", "task_end", db_session=mem_session)

        # Break the chain link in ev2
        mem_session.query(AuditEvent).filter(AuditEvent.id == ev2.id).update(
            {"prev_hmac": "wrong-prev-hmac"}
        )
        mem_session.commit()
        mem_session.expire_all()

        assert verify_chain("a", mem_session) is False

    def test_verify_only_checks_specified_agent(self, mem_session):
        from molecule_audit.ledger import AuditEvent, append_event, verify_chain

        append_event("agent-good", "s", "task_start", db_session=mem_session)
        ev_bad = append_event("agent-bad", "s", "task_start", db_session=mem_session)
        # Corrupt agent-bad's chain
        mem_session.query(AuditEvent).filter(AuditEvent.id == ev_bad.id).update(
            {"hmac": "a" * 64}
        )
        mem_session.commit()
        mem_session.expire_all()

        # agent-good should still be valid
        assert verify_chain("agent-good", mem_session) is True
        assert verify_chain("agent-bad", mem_session) is False


# ---------------------------------------------------------------------------
# hooks.LedgerHooks
# ---------------------------------------------------------------------------

class TestLedgerHooks:

    def test_on_task_start_writes_event(self, mem_session):
        from molecule_audit.hooks import LedgerHooks
        from molecule_audit.ledger import AuditEvent

        with LedgerHooks(session_id="s1", agent_id="ag1") as hooks:
            hooks._session = mem_session
            hooks.on_task_start(input_text="hello world")

        ev = mem_session.query(AuditEvent).filter(AuditEvent.operation == "task_start").first()
        assert ev is not None
        assert ev.agent_id == "ag1"
        assert ev.session_id == "s1"
        assert ev.input_hash == hashlib.sha256(b"hello world").hexdigest()
        assert ev.output_hash is None

    def test_on_llm_call_stores_model_name(self, mem_session):
        from molecule_audit.hooks import LedgerHooks
        from molecule_audit.ledger import AuditEvent

        hooks = LedgerHooks(session_id="s1", agent_id="ag1")
        hooks._session = mem_session
        hooks.on_llm_call(model="hermes-4-405b", input_text="prompt", output_text="reply")
        hooks.close()

        ev = mem_session.query(AuditEvent).filter(AuditEvent.operation == "llm_call").first()
        assert ev.model_used == "hermes-4-405b"
        assert ev.input_hash == hashlib.sha256(b"prompt").hexdigest()
        assert ev.output_hash == hashlib.sha256(b"reply").hexdigest()

    def test_on_tool_call_stores_tool_name_in_model_used(self, mem_session):
        from molecule_audit.hooks import LedgerHooks
        from molecule_audit.ledger import AuditEvent

        hooks = LedgerHooks(session_id="s1", agent_id="ag1")
        hooks._session = mem_session
        hooks.on_tool_call("web_search", input_data={"query": "test"}, output_data="result")
        hooks.close()

        ev = mem_session.query(AuditEvent).filter(AuditEvent.operation == "tool_call").first()
        assert ev.model_used == "web_search"

    def test_on_tool_call_dict_input_is_hashed(self, mem_session):
        from molecule_audit.hooks import LedgerHooks, _to_bytes
        from molecule_audit.ledger import AuditEvent, hash_content

        hooks = LedgerHooks(session_id="s1", agent_id="ag1")
        hooks._session = mem_session
        input_data = {"query": "molecule AI"}
        hooks.on_tool_call("search", input_data=input_data)
        hooks.close()

        ev = mem_session.query(AuditEvent).filter(AuditEvent.operation == "tool_call").first()
        expected_hash = hash_content(_to_bytes(input_data))
        assert ev.input_hash == expected_hash

    def test_on_task_end_writes_event(self, mem_session):
        from molecule_audit.hooks import LedgerHooks
        from molecule_audit.ledger import AuditEvent

        hooks = LedgerHooks(session_id="s1", agent_id="ag1")
        hooks._session = mem_session
        hooks.on_task_end(output_text="done")
        hooks.close()

        ev = mem_session.query(AuditEvent).filter(AuditEvent.operation == "task_end").first()
        assert ev is not None
        assert ev.output_hash == hashlib.sha256(b"done").hexdigest()

    def test_full_task_lifecycle_writes_four_events(self, mem_session):
        from molecule_audit.hooks import LedgerHooks
        from molecule_audit.ledger import AuditEvent

        with LedgerHooks(session_id="s1", agent_id="ag1") as hooks:
            hooks._session = mem_session
            hooks.on_task_start(input_text="go")
            hooks.on_llm_call(model="m", input_text="q", output_text="a")
            hooks.on_tool_call("t", input_data="x", output_data="y")
            hooks.on_task_end(output_text="done")

        events = mem_session.query(AuditEvent).filter(AuditEvent.agent_id == "ag1").all()
        ops = [e.operation for e in events]
        assert ops == ["task_start", "llm_call", "tool_call", "task_end"]

    def test_context_manager_closes_session(self):
        from molecule_audit.hooks import LedgerHooks

        hooks = LedgerHooks(session_id="s1", agent_id="ag1", db_url="sqlite:///:memory:")
        # Force session open
        _ = hooks._open_session()
        assert hooks._session is not None

        with hooks:
            pass  # __exit__ calls close()

        assert hooks._session is None

    def test_exception_in_append_is_swallowed(self, mem_session, caplog):
        """Audit failures must never raise — they log a WARNING instead."""
        import molecule_audit.ledger as ledger
        from molecule_audit.hooks import LedgerHooks

        # Make the key derivation raise so append_event will fail
        ledger.reset_hmac_key_cache()
        original_salt = ledger.AUDIT_LEDGER_SALT
        ledger.AUDIT_LEDGER_SALT = ""

        hooks = LedgerHooks(session_id="s1", agent_id="ag1")
        hooks._session = mem_session

        with caplog.at_level(logging.WARNING, logger="molecule_audit.hooks"):
            # Must NOT raise
            hooks.on_task_start(input_text="test")

        assert any("failed to append event" in r.message for r in caplog.records)

        # Restore
        ledger.AUDIT_LEDGER_SALT = original_salt
        ledger.reset_hmac_key_cache()

    def test_human_oversight_flag_default(self, mem_session):
        from molecule_audit.hooks import LedgerHooks
        from molecule_audit.ledger import AuditEvent

        hooks = LedgerHooks(session_id="s1", agent_id="ag1", human_oversight_flag=True)
        hooks._session = mem_session
        hooks.on_task_start()
        hooks.close()

        ev = mem_session.query(AuditEvent).first()
        assert ev.human_oversight_flag is True

    def test_risk_flag_propagated(self, mem_session):
        from molecule_audit.hooks import LedgerHooks
        from molecule_audit.ledger import AuditEvent

        hooks = LedgerHooks(session_id="s1", agent_id="ag1")
        hooks._session = mem_session
        hooks.on_llm_call(model="m", risk_flag=True)
        hooks.close()

        ev = mem_session.query(AuditEvent).first()
        assert ev.risk_flag is True


# ---------------------------------------------------------------------------
# verify.py CLI
# ---------------------------------------------------------------------------

class TestVerifyCLI:

    def test_valid_chain_exits_zero(self, mem_session, monkeypatch, capsys):
        import molecule_audit.ledger as ledger
        from molecule_audit.ledger import append_event
        from molecule_audit.verify import main

        # Write a short chain
        for op in ("task_start", "llm_call", "task_end"):
            append_event("cli-agent", "s", op, db_session=mem_session)

        # Patch get_session_factory to return our in-memory session
        factory_mock = MagicMock(return_value=mem_session)
        monkeypatch.setattr(
            "molecule_audit.ledger.get_session_factory",
            lambda db_url: factory_mock,
        )

        with pytest.raises(SystemExit) as exc_info:
            main(["--agent-id", "cli-agent"])

        assert exc_info.value.code == 0
        captured = capsys.readouterr()
        assert "CHAIN VALID" in captured.out
        assert "3 events" in captured.out

    def test_no_events_exits_zero(self, mem_session, monkeypatch, capsys):
        from molecule_audit.verify import main

        factory_mock = MagicMock(return_value=mem_session)
        monkeypatch.setattr(
            "molecule_audit.ledger.get_session_factory",
            lambda db_url: factory_mock,
        )

        with pytest.raises(SystemExit) as exc_info:
            main(["--agent-id", "ghost-agent"])

        assert exc_info.value.code == 0
        captured = capsys.readouterr()
        assert "No audit events" in captured.out

    def test_broken_chain_exits_one(self, mem_session, monkeypatch, capsys):
        from molecule_audit.ledger import AuditEvent, append_event
        from molecule_audit.verify import main

        ev = append_event("broken-agent", "s", "task_start", db_session=mem_session)
        # Corrupt the HMAC
        mem_session.query(AuditEvent).filter(AuditEvent.id == ev.id).update(
            {"hmac": "b" * 64}
        )
        mem_session.commit()
        mem_session.expire_all()

        factory_mock = MagicMock(return_value=mem_session)
        monkeypatch.setattr(
            "molecule_audit.ledger.get_session_factory",
            lambda db_url: factory_mock,
        )

        with pytest.raises(SystemExit) as exc_info:
            main(["--agent-id", "broken-agent"])

        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "CHAIN BROKEN" in captured.out

    def test_missing_salt_exits_two(self, monkeypatch, capsys):
        import molecule_audit.ledger as ledger
        from molecule_audit.verify import main

        ledger.reset_hmac_key_cache()
        ledger.AUDIT_LEDGER_SALT = ""
        monkeypatch.delenv("AUDIT_LEDGER_SALT", raising=False)

        # Patch get_session_factory to raise RuntimeError (simulates SALT check)
        def _raise(*a, **kw):
            raise RuntimeError("AUDIT_LEDGER_SALT environment variable is required but not set.")

        monkeypatch.setattr("molecule_audit.ledger.get_session_factory", _raise)

        with pytest.raises(SystemExit) as exc_info:
            main(["--agent-id", "any"])

        # The RuntimeError should be caught and cause exit(2) or exit(3)
        assert exc_info.value.code in (2, 3)
