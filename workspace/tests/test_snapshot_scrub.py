"""Tests for workspace.lib.snapshot_scrub — issue #823."""
from __future__ import annotations

import pytest

from lib.snapshot_scrub import (
    is_sandbox_content,
    scrub_content,
    scrub_memory_entry,
    scrub_snapshot,
)


# ---------- scrub_content ----------

def test_scrub_empty_returns_empty():
    assert scrub_content("") == ""
    assert scrub_content("no secrets here") == "no secrets here"


def test_scrub_anthropic_key():
    got = scrub_content("key: sk-ant-api03-aaaaaaaaaaaaaaaaaaaaaa")
    assert "sk-ant-api03" not in got
    assert "[REDACTED:SK_TOKEN]" in got


def test_scrub_openai_project_key():
    got = scrub_content("OPENAI_API_KEY=sk-proj-ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
    # Env-var pattern fires first and consumes the whole assignment.
    assert "sk-proj-" not in got
    assert "[REDACTED:API_KEY]" in got


def test_scrub_github_pat():
    got = scrub_content("token: ghp_ABCDEFGHIJKLMNOPQRSTUV1234567890")
    assert "ghp_" not in got
    assert "[REDACTED:GITHUB_PAT]" in got


def test_scrub_bearer_header():
    got = scrub_content("Authorization: Bearer abc123.def456.ghi789")
    assert "Bearer abc" not in got
    assert "[REDACTED:BEARER_TOKEN]" in got


def test_scrub_aws_access_key():
    got = scrub_content("AKIAIOSFODNN7EXAMPLE is embedded")
    assert "AKIAIOSFODNN7EXAMPLE" not in got
    assert "[REDACTED:AWS_ACCESS_KEY]" in got


def test_scrub_cloudflare_token():
    got = scrub_content("CF_TOKEN=cfut_abcdefghijklmnopqrstuvwxyz1234567890")
    assert "cfut_abc" not in got
    # Env-var pattern wins because it's more specific.
    assert "[REDACTED:TOKEN]" in got


def test_scrub_molecule_partner_key():
    got = scrub_content("mol_pk_abcdefghijklmnopqrstuvwxyz")
    assert "mol_pk_abc" not in got
    assert "[REDACTED:MOL_PK]" in got


def test_scrub_idempotent():
    # Running scrub twice produces the same output — [REDACTED:...] doesn't
    # itself match any pattern.
    first = scrub_content("sk-ant-api03-aaaaaaaaaaaaaaaaaaaaaa")
    second = scrub_content(first)
    assert first == second


def test_scrub_preserves_surrounding_text():
    got = scrub_content("prefix sk-ant-api03-abcdefghijklmnopqrst suffix")
    assert "prefix " in got
    assert " suffix" in got
    assert "sk-ant-" not in got


# ---------- is_sandbox_content ----------

def test_is_sandbox_content_detects_source_tag():
    assert is_sandbox_content("Some output, source=sandbox logged")
    assert is_sandbox_content("tool=run_code fired at 2026-01-01")


def test_is_sandbox_content_detects_output_marker():
    assert is_sandbox_content("[sandbox_output] ls -la\ntotal 0")


def test_is_sandbox_content_ignores_normal_memory():
    assert not is_sandbox_content("Remember to check the deploy on Monday")
    assert not is_sandbox_content("")


# ---------- scrub_memory_entry ----------

def test_scrub_memory_entry_redacts_content():
    entry = {"id": "mem-1", "content": "ANTHROPIC_API_KEY=sk-ant-api03-xxxxxxxxxxxxxxxxxxxx", "scope": "LOCAL"}
    got = scrub_memory_entry(entry)
    assert got is not None
    assert "sk-ant-" not in got["content"]
    assert got["id"] == "mem-1"
    assert got["scope"] == "LOCAL"


def test_scrub_memory_entry_drops_sandbox():
    entry = {"id": "mem-sandbox", "content": "source=sandbox cmd output"}
    got = scrub_memory_entry(entry)
    assert got is None


def test_scrub_memory_entry_preserves_original():
    entry = {"id": "mem-1", "content": "sk-ant-api03-xxxxxxxxxxxxxxxxxxxx"}
    _ = scrub_memory_entry(entry)
    # Original dict unchanged
    assert entry["content"] == "sk-ant-api03-xxxxxxxxxxxxxxxxxxxx"


# ---------- scrub_snapshot ----------

def test_scrub_snapshot_filters_and_redacts():
    snapshot = {
        "workspace_id": "ws-1",
        "memories": [
            {"id": "m1", "content": "Task completed successfully"},
            {"id": "m2", "content": "ANTHROPIC_API_KEY=sk-ant-api03-xxxxxxxxxxxxxxxxxxxx"},
            {"id": "m3", "content": "tool=run_code output: rm -rf /tmp"},
        ],
    }
    got = scrub_snapshot(snapshot)
    assert got["workspace_id"] == "ws-1"
    assert len(got["memories"]) == 2  # m3 dropped
    ids = [m["id"] for m in got["memories"]]
    assert "m1" in ids
    assert "m2" in ids
    assert "m3" not in ids
    # m2 content redacted
    m2 = next(m for m in got["memories"] if m["id"] == "m2")
    assert "sk-ant-" not in m2["content"]


def test_scrub_snapshot_empty_memories():
    snapshot = {"workspace_id": "ws-1", "memories": []}
    got = scrub_snapshot(snapshot)
    assert got["memories"] == []


def test_scrub_snapshot_missing_memories_key():
    snapshot = {"workspace_id": "ws-1"}
    got = scrub_snapshot(snapshot)
    assert got["memories"] == []


def test_scrub_snapshot_does_not_mutate_input():
    snapshot = {
        "workspace_id": "ws-1",
        "memories": [
            {"id": "m1", "content": "sk-ant-api03-xxxxxxxxxxxxxxxxxxxx"},
        ],
    }
    original_content = snapshot["memories"][0]["content"]
    _ = scrub_snapshot(snapshot)
    # Input memory content unchanged
    assert snapshot["memories"][0]["content"] == original_content


# ---------- regression: real-world combined patterns ----------

def test_scrub_combined_secrets_in_one_memory():
    """A memory that accumulated multiple secrets during a single session."""
    content = (
        "Called Anthropic with sk-ant-api03-abcdefghijklmnop "
        "and GitHub with ghp_ABCDEFGHIJKLMNOPQRST1234567890 "
        "and got Authorization: Bearer xyz.jwt.token"
    )
    got = scrub_content(content)
    assert "sk-ant-" not in got
    assert "ghp_" not in got
    assert "Bearer xyz" not in got
    assert got.count("[REDACTED:") == 3
