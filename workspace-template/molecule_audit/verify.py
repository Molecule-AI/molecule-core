"""molecule_audit.verify — CLI to verify an agent's HMAC chain integrity.

Usage
-----
    python -m molecule_audit.verify --agent-id <id> [--db <url>]

Options
-------
--agent-id   Agent ID whose chain to verify (required).
--db         SQLAlchemy DB URL override.
             Defaults to AUDIT_LEDGER_DB env var or /var/log/molecule/audit_ledger.db.

Exit codes
----------
0   Chain is valid (or no events found for this agent).
1   Chain is broken — tampered or corrupted row(s) detected.
2   Configuration error (e.g. AUDIT_LEDGER_SALT not set).
3   Database error (e.g. file not found, connection refused).

Example
-------
    export AUDIT_LEDGER_SALT=<your-secret>
    export AUDIT_LEDGER_DB=/var/log/molecule/audit_ledger.db
    python -m molecule_audit.verify --agent-id my-workspace-id
    # CHAIN VALID (42 events)
"""

from __future__ import annotations

import argparse
import hmac as _hmac_mod
import sys


def main(argv=None) -> None:
    parser = argparse.ArgumentParser(
        prog="python -m molecule_audit.verify",
        description=(
            "Verify the HMAC chain integrity for a given agent's audit log. "
            "Exit 0 = valid, 1 = broken, 2 = config error, 3 = DB error."
        ),
    )
    parser.add_argument(
        "--agent-id",
        required=True,
        metavar="AGENT_ID",
        help="Agent workspace ID to verify.",
    )
    parser.add_argument(
        "--db",
        default=None,
        metavar="URL",
        help=(
            "SQLAlchemy DB URL (e.g. sqlite:///path.db or "
            "postgresql://user:pass@host/db). "
            "Defaults to AUDIT_LEDGER_DB env var."
        ),
    )
    args = parser.parse_args(argv)

    # Defer imports so errors in configuration (missing SALT) produce clean output.
    try:
        from molecule_audit.ledger import (
            AuditEvent,
            _compute_event_hmac,
            get_session_factory,
            verify_chain,
        )
    except RuntimeError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(2)

    try:
        factory = get_session_factory(args.db)
        session = factory()
    except Exception as exc:
        print(f"ERROR: could not open database: {exc}", file=sys.stderr)
        sys.exit(3)

    try:
        from sqlalchemy import asc

        n_events = (
            session.query(AuditEvent)
            .filter(AuditEvent.agent_id == args.agent_id)
            .count()
        )

        if n_events == 0:
            print(f"No audit events found for agent_id={args.agent_id!r}")
            sys.exit(0)

        valid = verify_chain(args.agent_id, session)

        if valid:
            print(f"CHAIN VALID ({n_events} events)")
            sys.exit(0)
        else:
            # Walk the chain manually to report the exact broken event.
            events = (
                session.query(AuditEvent)
                .filter(AuditEvent.agent_id == args.agent_id)
                .order_by(asc(AuditEvent.timestamp), asc(AuditEvent.id))
                .all()
            )
            expected_prev = None
            for ev in events:
                expected_hmac = _compute_event_hmac(ev)
                if not _hmac_mod.compare_digest(ev.hmac, expected_hmac):
                    print(
                        f"CHAIN BROKEN at event {ev.id} "
                        f"(HMAC mismatch: stored={ev.hmac[:12]}... "
                        f"computed={expected_hmac[:12]}...)"
                    )
                    sys.exit(1)
                if not _hmac_mod.compare_digest(ev.prev_hmac or "", expected_prev or ""):
                    print(
                        f"CHAIN BROKEN at event {ev.id} "
                        f"(prev_hmac mismatch: stored={ev.prev_hmac} "
                        f"expected={expected_prev})"
                    )
                    sys.exit(1)
                expected_prev = ev.hmac
            # verify_chain said broken but we couldn't find the exact event
            print(f"CHAIN BROKEN (position unknown; run with DEBUG logging)")
            sys.exit(1)

    except Exception as exc:
        print(f"ERROR: verification failed: {exc}", file=sys.stderr)
        sys.exit(3)
    finally:
        session.close()


if __name__ == "__main__":
    main()
