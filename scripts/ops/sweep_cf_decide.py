"""Decision logic extracted from sweep-cf-orphans.sh for unit testing (#2027).

The bash script embeds the same logic inline as a python heredoc — this
module is a verbatim copy used by test_sweep_cf_decide.py. The parity
test (TestParityWithBashScript) reads the bash script and asserts the
canonical block in this file is present byte-for-byte, so the two
cannot drift apart silently.

If you change the rules: edit BOTH this file AND the inline block in
``scripts/ops/sweep-cf-orphans.sh`` (the canonical block runs from
``# CANONICAL DECIDE BEGIN`` to ``# CANONICAL DECIDE END`` markers in
both files; the parity check compares those slices).

Inputs to ``decide(record, prod_slugs, staging_slugs, ec2_names)``:
  record       Cloudflare DNS record dict {name, id, type}
  prod_slugs   set of CP-prod org slugs (live tenants)
  staging_slugs set of CP-staging org slugs
  ec2_names    set of live EC2 Name tags (e.g. ``ws-d3605ef2-f7d``)

Returns ``(action, reason, id, name, type)`` matching the bash heredoc.
"""
from __future__ import annotations

import re
from typing import Iterable


# CANONICAL DECIDE BEGIN
def decide(r, prod_slugs, staging_slugs, ec2_names):
    n = r["name"]
    rid = r["id"]
    typ = r["type"]
    all_slugs = prod_slugs | staging_slugs

    # Rule 1: platform core — leave alone
    if n == "moleculesai.app":
        return ("keep", "apex", rid, n, typ)
    if n.startswith("_") or n.endswith("._domainkey.moleculesai.app"):
        return ("keep", "verification/key", rid, n, typ)
    if n in {"api.moleculesai.app","app.moleculesai.app","doc.moleculesai.app",
            "send.moleculesai.app","status.moleculesai.app","www.moleculesai.app",
            "staging-api.moleculesai.app"}:
        return ("keep", "platform-core", rid, n, typ)

    # Rule 3: ws-<hex8>-<rest>.(staging.)moleculesai.app
    m = re.match(r"^(ws-[a-f0-9]{8}-[a-f0-9]+)(?:\.staging)?\.moleculesai\.app$", n)
    if m:
        prefix = m.group(1)
        # Live EC2 names are like "ws-d3605ef2-f7d" — same shape as DNS subdomain.
        for ename in ec2_names:
            if ename.startswith(prefix):
                return ("keep", "live-ec2", rid, n, typ)
        return ("delete", "orphan-ws", rid, n, typ)

    # Rule 4: e2e-* tenants (includes canary, canvas variants)
    m = re.match(r"^(e2e-[^.]+)(?:\.staging)?\.moleculesai\.app$", n)
    if m:
        slug = m.group(1)
        if slug in all_slugs:
            return ("keep", "live-e2e-tenant", rid, n, typ)
        return ("delete", "orphan-e2e-tenant", rid, n, typ)

    # Rule 2: any other tenant subdomain (slug.moleculesai.app or slug.staging.moleculesai.app)
    m = re.match(r"^([a-z0-9][a-z0-9-]*)(?:\.staging)?\.moleculesai\.app$", n)
    if m:
        slug = m.group(1)
        if slug in all_slugs:
            return ("keep", "live-tenant", rid, n, typ)
        # Only flag as orphan if name looks like a tenant (not a one-off like "hermes-final-*")
        # To avoid false-positive nukes on ad-hoc records, we KEEP anything that
        # does not match a known pattern. Orphan only for explicit tenant-shaped names.
        return ("keep", "unknown-subdomain-kept-for-safety", rid, n, typ)

    return ("keep", "not-a-pattern-we-sweep", rid, n, typ)
# CANONICAL DECIDE END


def safety_gate(total: int, delete_count: int, max_delete_pct: int = 50) -> bool:
    """Return True iff the sweep is safe to execute.

    Mirrors the shell-side gate in sweep-cf-orphans.sh: if the deletion
    fraction exceeds ``max_delete_pct`` the sweep refuses to run. The
    bash script computes the integer percentage as ``DELETE_COUNT*100/TOTAL``
    — keeping the same arithmetic here so a future "raise to 75%" tweak
    needs to be made in only one place semantically.
    """
    if total <= 0:
        return True  # nothing to delete; gate is trivially satisfied
    pct = delete_count * 100 // total
    return pct <= max_delete_pct
