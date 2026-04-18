"""molecule_audit.ledger — HMAC-SHA256-chained SQLAlchemy audit event log.

EU AI Act Annex III compliance (Art. 12/13 record-keeping, Art. 17 quality
management system) for high-risk AI systems.

HMAC chain design (EDDI pattern, PBKDF2 + SHA-256)
----------------------------------------------------
Key derivation:
    key = PBKDF2HMAC(
        algorithm=SHA-256,
        password=AUDIT_LEDGER_SALT,      # from env — the shared secret
        salt=b"molecule-audit-ledger-v1", # fixed domain separator
        iterations=210_000,
        length=32,
    )

Canonical JSON (for HMAC input):
    json.dumps(row_dict_without_hmac_field, sort_keys=True, separators=(",", ":"))
    Timestamp is serialised as RFC-3339 seconds-precision with Z suffix
    (e.g. "2026-04-17T12:34:56Z") so the format matches Go's time.Time.UTC().

Per-row HMAC:
    hmac_hex = HMAC-SHA256(key, canonical_json.encode()).hexdigest()

Chain linkage:
    prev_hmac = hmac field of the immediately prior row for this agent_id
                (None / NULL for the first row of each agent)

Tamper-evidence: any row modification breaks all subsequent HMACs for that
agent_id.

Environment variables
---------------------
AUDIT_LEDGER_SALT   REQUIRED. Secret salt used as PBKDF2 password.
                    Raises RuntimeError at first key-derivation call if unset.
AUDIT_LEDGER_DB     Path to SQLite file.
                    Default: /var/log/molecule/audit_ledger.db
                    Override with a full SQLAlchemy URL (sqlite:///..., postgresql://...)
                    for non-SQLite backends.
"""

from __future__ import annotations

import hashlib
import hmac as _hmac_mod
import json
import logging
import os
from datetime import datetime, timezone
from typing import Optional
from uuid import uuid4

from sqlalchemy import Boolean, Column, DateTime, String, create_engine
from sqlalchemy.orm import DeclarativeBase, Session, sessionmaker

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

AUDIT_LEDGER_DB: str = os.environ.get(
    "AUDIT_LEDGER_DB", "/var/log/molecule/audit_ledger.db"
)

# PBKDF2 parameters (must never change once events are written — all existing
# HMACs become unverifiable if parameters change).
_PBKDF2_SALT: bytes = b"molecule-audit-ledger-v1"  # fixed domain separator
_PBKDF2_ITERATIONS: int = 210_000
_PBKDF2_DKLEN: int = 32

# Cached derived key (reset to None in tests when AUDIT_LEDGER_SALT changes).
_hmac_key: Optional[bytes] = None


# ---------------------------------------------------------------------------
# PBKDF2 key derivation
# ---------------------------------------------------------------------------

def _get_hmac_key() -> bytes:
    """Return (and cache) the 32-byte HMAC key derived from AUDIT_LEDGER_SALT.

    Reads AUDIT_LEDGER_SALT exclusively from the environment — never from a
    module-level attribute — so the secret is not exposed in the module
    namespace.  Raises RuntimeError if the env var is not set.
    """
    global _hmac_key
    if _hmac_key is None:
        salt = os.environ.get("AUDIT_LEDGER_SALT", "")
        if not salt:
            raise RuntimeError(
                "AUDIT_LEDGER_SALT environment variable is required but not set. "
                "Generate a random 32-byte hex string and export it before "
                "starting the agent: "
                "export AUDIT_LEDGER_SALT=$(python3 -c "
                "\"import secrets; print(secrets.token_hex(32))\")"
            )
        _hmac_key = hashlib.pbkdf2_hmac(
            "sha256",
            password=salt.encode("utf-8"),
            salt=_PBKDF2_SALT,
            iterations=_PBKDF2_ITERATIONS,
            dklen=_PBKDF2_DKLEN,
        )
    return _hmac_key


def reset_hmac_key_cache() -> None:
    """Reset the cached HMAC key — call after changing AUDIT_LEDGER_SALT env var in tests."""
    global _hmac_key
    _hmac_key = None


# ---------------------------------------------------------------------------
# Canonical JSON helpers
# ---------------------------------------------------------------------------

def _ts_to_canonical(ts: datetime | None) -> str | None:
    """Format a datetime as RFC-3339 seconds-precision Z-suffixed string.

    Strips microseconds and converts to UTC so the format is identical to
    Go's ``time.Time.UTC().Format("2006-01-02T15:04:05Z")``.
    """
    if ts is None:
        return None
    if ts.tzinfo is not None:
        ts = ts.astimezone(timezone.utc)
    return ts.strftime("%Y-%m-%dT%H:%M:%SZ")


def _to_canonical_dict(ev: "AuditEvent") -> dict:
    """Return the dict used as HMAC input — excludes the hmac field itself."""
    return {
        "agent_id": ev.agent_id,
        "human_oversight_flag": ev.human_oversight_flag,
        "id": ev.id,
        "input_hash": ev.input_hash,
        "model_used": ev.model_used,
        "operation": ev.operation,
        "output_hash": ev.output_hash,
        "prev_hmac": ev.prev_hmac,
        "risk_flag": ev.risk_flag,
        "session_id": ev.session_id,
        "timestamp": _ts_to_canonical(ev.timestamp),
    }


def _compute_event_hmac(ev: "AuditEvent") -> str:
    """Compute HMAC-SHA256 hex digest of ev's canonical JSON.

    Keys are sorted alphabetically (matching Python json.dumps sort_keys=True
    and Go encoding/json.Marshal on a map).  Separators are compact (no spaces)
    so the output matches Go's json.Marshal.
    """
    canonical = _to_canonical_dict(ev)
    payload = json.dumps(canonical, sort_keys=True, separators=(",", ":")).encode("utf-8")
    key = _get_hmac_key()
    return _hmac_mod.new(key, payload, "sha256").hexdigest()


# ---------------------------------------------------------------------------
# Content hashing helper (privacy-preserving)
# ---------------------------------------------------------------------------

def hash_content(content: str | bytes | None) -> str | None:
    """Return SHA-256 hex digest of content, or None if content is falsy.

    Use this to record *that* specific content was processed without persisting
    the raw content itself (satisfies EU AI Act data-minimisation principles).
    """
    if content is None:
        return None
    if isinstance(content, str):
        content = content.encode("utf-8")
    return hashlib.sha256(content).hexdigest()


# ---------------------------------------------------------------------------
# SQLAlchemy model
# ---------------------------------------------------------------------------

class Base(DeclarativeBase):
    pass


class AuditEvent(Base):
    """Append-only HMAC-chained audit event.

    12 fields: 6 legally mandatory under EU AI Act Art. 12/13, plus 4 strongly
    recommended, plus the 2-field HMAC chain (prev_hmac, hmac).
    """

    __tablename__ = "audit_events"

    # Identity
    id = Column(String, primary_key=True, default=lambda: str(uuid4()))
    timestamp = Column(
        DateTime(timezone=True),
        nullable=False,
        default=lambda: datetime.now(timezone.utc),
    )

    # EU AI Act Art. 12 mandatory fields
    agent_id = Column(String, nullable=False)
    session_id = Column(String, nullable=False)   # gen_ai.conversation.id
    operation = Column(String, nullable=False)    # task_start|llm_call|tool_call|task_end

    # Privacy-preserving content fingerprints
    input_hash = Column(String, nullable=True)    # SHA-256 of input text
    output_hash = Column(String, nullable=True)   # SHA-256 of output text

    # EU AI Act Art. 13 transparency fields
    model_used = Column(String, nullable=True)    # gen_ai.request.model (or tool name)

    # Oversight flags (Art. 14 human oversight)
    human_oversight_flag = Column(Boolean, nullable=False, default=False)
    risk_flag = Column(Boolean, nullable=False, default=False)

    # HMAC chain
    prev_hmac = Column(String, nullable=True)  # hmac of previous row for this agent_id
    hmac = Column(String, nullable=False)      # HMAC of this row's canonical JSON

    def to_dict(self) -> dict:
        """Return a full dict suitable for API responses (ISO 8601 timestamp)."""
        return {
            "id": self.id,
            "timestamp": self.timestamp.isoformat() if self.timestamp else None,
            "agent_id": self.agent_id,
            "session_id": self.session_id,
            "operation": self.operation,
            "input_hash": self.input_hash,
            "output_hash": self.output_hash,
            "model_used": self.model_used,
            "human_oversight_flag": self.human_oversight_flag,
            "risk_flag": self.risk_flag,
            "prev_hmac": self.prev_hmac,
            "hmac": self.hmac,
        }

    def __repr__(self) -> str:
        return (
            f"<AuditEvent id={self.id!r} agent_id={self.agent_id!r} "
            f"op={self.operation!r} ts={self.timestamp!r}>"
        )


# ---------------------------------------------------------------------------
# Engine / session factory
# ---------------------------------------------------------------------------

_engine = None
_SessionFactory = None


def get_engine(db_url: str | None = None):
    """Return (and cache) the SQLAlchemy engine.

    Creates the ``audit_events`` table if it does not already exist.
    """
    global _engine
    if _engine is None:
        url = db_url or _db_url_from_env()
        if url.startswith("sqlite:///"):
            _ensure_sqlite_parent(url)
        connect_args = {"check_same_thread": False} if "sqlite" in url else {}
        _engine = create_engine(url, connect_args=connect_args)
        Base.metadata.create_all(_engine)
    return _engine


def _db_url_from_env() -> str:
    """Build the DB URL from environment variables."""
    db = AUDIT_LEDGER_DB
    if db.startswith(("sqlite://", "postgresql://", "postgres://")):
        return db
    return f"sqlite:///{db}"


def _ensure_sqlite_parent(url: str) -> None:
    """Create the parent directory for a sqlite:///path URL if needed."""
    path = url[len("sqlite:///"):]
    if path and path != ":memory:":
        os.makedirs(os.path.dirname(os.path.abspath(path)), exist_ok=True)


def get_session_factory(db_url: str | None = None):
    """Return (and cache) a SQLAlchemy sessionmaker bound to the engine."""
    global _SessionFactory
    if _SessionFactory is None:
        _SessionFactory = sessionmaker(bind=get_engine(db_url))
    return _SessionFactory


def reset_engine_cache() -> None:
    """Reset the cached engine and session factory — for tests only."""
    global _engine, _SessionFactory
    _engine = None
    _SessionFactory = None


# ---------------------------------------------------------------------------
# Core write API
# ---------------------------------------------------------------------------

def _prev_hmac_for_agent(agent_id: str, session: Session) -> str | None:
    """Return the hmac of the most recent event for agent_id (None if none)."""
    last = (
        session.query(AuditEvent)
        .filter(AuditEvent.agent_id == agent_id)
        .order_by(AuditEvent.timestamp.desc(), AuditEvent.id.desc())
        .first()
    )
    return last.hmac if last else None


def append_event(
    agent_id: str,
    session_id: str,
    operation: str,
    *,
    input_hash: str | None = None,
    output_hash: str | None = None,
    model_used: str | None = None,
    human_oversight_flag: bool = False,
    risk_flag: bool = False,
    db_session: Session | None = None,
    db_url: str | None = None,
) -> AuditEvent:
    """Append one signed, chained event to the ledger and return it.

    Derives the HMAC key from AUDIT_LEDGER_SALT (raises RuntimeError if unset),
    looks up the previous row's HMAC to form the chain link, signs the new row,
    and writes it to the database.

    Parameters
    ----------
    agent_id:              Identity of the agent (typically WORKSPACE_ID).
    session_id:            Task / conversation ID (gen_ai.conversation.id).
    operation:             One of: task_start, llm_call, tool_call, task_end.
    input_hash:            SHA-256 of the input (use hash_content()).
    output_hash:           SHA-256 of the output.
    model_used:            Model name (for llm_call) or tool name (for tool_call).
    human_oversight_flag:  True if human review was required / triggered.
    risk_flag:             True if a risk condition was detected.
    db_session:            Pre-opened Session (created + closed internally if None).
    db_url:                SQLAlchemy URL override (used if session is None).
    """
    own_session = db_session is None
    if own_session:
        factory = get_session_factory(db_url)
        db_session = factory()

    try:
        prev_hmac = _prev_hmac_for_agent(agent_id, db_session)

        event = AuditEvent(
            id=str(uuid4()),
            timestamp=datetime.now(timezone.utc),
            agent_id=agent_id,
            session_id=session_id,
            operation=operation,
            input_hash=input_hash,
            output_hash=output_hash,
            model_used=model_used,
            human_oversight_flag=human_oversight_flag,
            risk_flag=risk_flag,
            prev_hmac=prev_hmac,
            hmac="",  # placeholder — replaced below after ID/timestamp are set
        )

        # Compute the real HMAC now that all fields are populated.
        event.hmac = _compute_event_hmac(event)

        db_session.add(event)
        db_session.commit()
        db_session.refresh(event)
        return event

    except Exception:
        if own_session:
            db_session.rollback()
        raise
    finally:
        if own_session:
            db_session.close()


# ---------------------------------------------------------------------------
# Verification
# ---------------------------------------------------------------------------

def verify_chain(agent_id: str, db_session: Session) -> bool:
    """Return True if the entire HMAC chain for agent_id is intact.

    Iterates all events for agent_id in chronological order and checks:
    1. Each row's stored hmac matches the freshly-computed HMAC.
    2. Each row's prev_hmac equals the prior row's hmac (None for first row).

    Returns False (and logs a warning) at the first broken link.
    Returns True vacuously when there are no events.
    """
    events = (
        db_session.query(AuditEvent)
        .filter(AuditEvent.agent_id == agent_id)
        .order_by(AuditEvent.timestamp.asc(), AuditEvent.id.asc())
        .all()
    )

    expected_prev: str | None = None
    for ev in events:
        expected_hmac = _compute_event_hmac(ev)
        if not _hmac_mod.compare_digest(ev.hmac, expected_hmac):
            logger.warning(
                "audit: HMAC mismatch at event %s (agent=%s): "
                "stored=%r computed=%r",
                ev.id,
                agent_id,
                ev.hmac,
                expected_hmac,
            )
            return False
        if not _hmac_mod.compare_digest(ev.prev_hmac or "", expected_prev or ""):
            logger.warning(
                "audit: chain break at event %s (agent=%s): "
                "stored prev_hmac=%r expected=%r",
                ev.id,
                agent_id,
                ev.prev_hmac,
                expected_prev,
            )
            return False
        expected_prev = ev.hmac

    return True
