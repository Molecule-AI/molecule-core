"""Snapshot scrubbing — strip secrets and internal details from hibernation snapshots.

Issue #823 (sub of #799). Before the workspace runtime serializes a memory
snapshot for hibernation, every memory entry's content must pass through
this scrubber so an attacker who obtains a snapshot blob cannot recover:

- API keys (sk-ant-, sk-proj-, ghp_, etc.)
- Auth tokens (Bearer headers, OAuth tokens)
- Env-var assignments (ANTHROPIC_API_KEY=..., OPENAI_API_KEY=...)
- Arbitrary subprocess output from the sandbox tool (can be anything)

The scrubber is a pure function so it can be unit-tested independently.
"""
from __future__ import annotations

import re
from typing import Any


# Compiled once at import time — most-specific patterns first so that
# env-var assignments are caught before the generic sk-* or base64 sweeps
# swallow only part of the match.
_SECRET_PATTERNS: list[tuple[re.Pattern[str], str]] = [
    # Env-var assignments: ANTHROPIC_API_KEY=sk-ant-... GITHUB_TOKEN=ghp_...
    (re.compile(r"(?i)\b[A-Z][A-Z0-9_]*_API_KEY\s*=\s*\S+"), "API_KEY"),
    (re.compile(r"(?i)\b[A-Z][A-Z0-9_]*_TOKEN\s*=\s*\S+"), "TOKEN"),
    (re.compile(r"(?i)\b[A-Z][A-Z0-9_]*_SECRET\s*=\s*\S+"), "SECRET"),
    # HTTP Bearer header values.
    (re.compile(r"Bearer\s+\S+"), "BEARER_TOKEN"),
    # OpenAI / Anthropic sk-... / sk-ant-... / sk-proj-... key format.
    (re.compile(r"sk-[A-Za-z0-9\-_]{16,}"), "SK_TOKEN"),
    # GitHub personal access tokens and installation tokens.
    (re.compile(r"ghp_[A-Za-z0-9]{20,}"), "GITHUB_PAT"),
    (re.compile(r"ghs_[A-Za-z0-9]{20,}"), "GITHUB_SERVER_TOKEN"),
    (re.compile(r"github_pat_[A-Za-z0-9_]{60,}"), "GITHUB_PAT_V2"),
    # AWS access key IDs.
    (re.compile(r"\bAKIA[A-Z0-9]{16}\b"), "AWS_ACCESS_KEY"),
    # Cloudflare API tokens.
    (re.compile(r"\bcfut_[A-Za-z0-9]{32,}"), "CF_TOKEN"),
    # Molecule partner API keys (Phase 34).
    (re.compile(r"\bmol_pk_[A-Za-z0-9]{20,}"), "MOL_PK"),
    # context7 tokens.
    (re.compile(r"\bctx7_[A-Za-z0-9]+"), "CTX7_TOKEN"),
    # High-entropy base64 blobs 33+ chars. Catches long opaque tokens that
    # don't match any structured pattern above.
    (re.compile(r"[A-Za-z0-9+/]{33,}={0,2}"), "BASE64_BLOB"),
]


# Substring markers that identify content from the run_code sandbox tool.
# Any memory entry tagged with this source is excluded wholesale from the
# snapshot — the arbitrary subprocess output cannot be safely scrubbed by
# pattern alone (attacker could print `echo "innocent"` but have hidden
# secrets in stderr or file handles).
_SANDBOX_TOOL_MARKERS = (
    "source=sandbox",
    "tool=run_code",
    "[sandbox_output]",
)


def scrub_content(content: str) -> str:
    """Return `content` with secret patterns replaced by [REDACTED:LABEL] markers.

    Idempotent — running scrub_content on already-scrubbed output is a no-op
    because [REDACTED:...] doesn't match any of the patterns above.
    """
    if not content:
        return content
    out = content
    for pattern, label in _SECRET_PATTERNS:
        out = pattern.sub(f"[REDACTED:{label}]", out)
    return out


def is_sandbox_content(content: str) -> bool:
    """Return True if `content` originates from the run_code sandbox tool.

    Sandbox output can contain arbitrary subprocess stdout/stderr that may
    include secrets the scrubber wouldn't recognize (e.g. printed via a
    custom format). Entries matching this check should be excluded from
    the snapshot entirely rather than scrubbed.
    """
    if not content:
        return False
    lower = content.lower()
    return any(marker in lower for marker in _SANDBOX_TOOL_MARKERS)


def scrub_memory_entry(entry: dict[str, Any]) -> dict[str, Any] | None:
    """Scrub a single memory entry for snapshot inclusion.

    Returns a new dict with secrets redacted, or None if the entry must be
    excluded entirely (sandbox-sourced content).

    The input dict is treated as read-only — callers should use the returned
    value and not mutate the original.
    """
    content = entry.get("content", "")
    if is_sandbox_content(content):
        return None
    scrubbed = dict(entry)
    scrubbed["content"] = scrub_content(content)
    return scrubbed


def scrub_snapshot(snapshot: dict[str, Any]) -> dict[str, Any]:
    """Scrub a full snapshot payload before serialization.

    Walks the `memories` list, scrubs each entry's content, and drops
    sandbox-sourced entries. Other snapshot fields (workspace metadata,
    config, etc.) pass through unchanged — they are not expected to contain
    user-supplied secret-bearing content.

    Returns a new dict; the input is not mutated.
    """
    out = dict(snapshot)
    memories = snapshot.get("memories") or []
    scrubbed_list = []
    for entry in memories:
        cleaned = scrub_memory_entry(entry)
        if cleaned is not None:
            scrubbed_list.append(cleaned)
    out["memories"] = scrubbed_list
    return out
