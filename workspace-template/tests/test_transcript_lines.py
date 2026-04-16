"""Tests for the new BaseAdapter.transcript_lines() method + claude-code override."""

import asyncio
import json
import os
import tempfile
from pathlib import Path

import pytest


# ── Default (BaseAdapter) ───────────────────────────────────────────────────


def test_base_adapter_returns_unsupported():
    """Adapters that don't override return supported:False."""
    from adapters.langgraph.adapter import LangGraphAdapter
    a = LangGraphAdapter()
    r = asyncio.run(a.transcript_lines())
    assert r["supported"] is False
    assert r["lines"] == []
    assert r["cursor"] == 0
    assert r["runtime"] == "langgraph"
    assert r["more"] is False


# ── Claude Code override ────────────────────────────────────────────────────


def _write_jsonl(path: Path, entries: list[dict]) -> None:
    with path.open("w") as f:
        for e in entries:
            f.write(json.dumps(e) + "\n")


def test_claude_code_no_projects_dir():
    """Returns supported:True with empty lines when projects dir missing."""
    from adapters.claude_code.adapter import ClaudeCodeAdapter
    with tempfile.TemporaryDirectory() as tmp:
        os.environ["HOME"] = tmp
        os.environ["CLAUDE_PROJECT_CWD"] = "/configs"
        try:
            r = asyncio.run(ClaudeCodeAdapter().transcript_lines())
            assert r["supported"] is True
            assert r["lines"] == []
            assert r["cursor"] == 0
            assert "-configs" in r["source"]
        finally:
            del os.environ["CLAUDE_PROJECT_CWD"]


def test_claude_code_reads_jsonl_with_pagination():
    from adapters.claude_code.adapter import ClaudeCodeAdapter
    with tempfile.TemporaryDirectory() as tmp:
        os.environ["HOME"] = tmp
        os.environ["CLAUDE_PROJECT_CWD"] = "/configs"
        try:
            projdir = Path(tmp) / ".claude" / "projects" / "-configs"
            projdir.mkdir(parents=True)
            _write_jsonl(projdir / "abc.jsonl", [
                {"type": "user", "n": 1},
                {"type": "assistant", "n": 2},
                {"type": "user", "n": 3},
                {"type": "assistant", "n": 4},
                {"type": "user", "n": 5},
            ])
            a = ClaudeCodeAdapter()
            # First page (limit=2)
            r1 = asyncio.run(a.transcript_lines(since=0, limit=2))
            assert r1["supported"] is True
            assert [l["n"] for l in r1["lines"]] == [1, 2]
            assert r1["cursor"] == 2
            assert r1["more"] is True
            # Second page (since=2, limit=2)
            r2 = asyncio.run(a.transcript_lines(since=2, limit=2))
            assert [l["n"] for l in r2["lines"]] == [3, 4]
            assert r2["cursor"] == 4
            assert r2["more"] is True
            # Third page exhausts
            r3 = asyncio.run(a.transcript_lines(since=4, limit=2))
            assert [l["n"] for l in r3["lines"]] == [5]
            assert r3["cursor"] == 5
            assert r3["more"] is False
        finally:
            del os.environ["CLAUDE_PROJECT_CWD"]


def test_claude_code_picks_most_recent_jsonl():
    """When multiple .jsonl files exist, picks the most-recently-modified."""
    from adapters.claude_code.adapter import ClaudeCodeAdapter
    with tempfile.TemporaryDirectory() as tmp:
        os.environ["HOME"] = tmp
        os.environ["CLAUDE_PROJECT_CWD"] = "/configs"
        try:
            projdir = Path(tmp) / ".claude" / "projects" / "-configs"
            projdir.mkdir(parents=True)
            old = projdir / "old.jsonl"
            new = projdir / "new.jsonl"
            _write_jsonl(old, [{"src": "old"}])
            _write_jsonl(new, [{"src": "new"}])
            # Force new to be more recent
            os.utime(old, (1000, 1000))
            os.utime(new, (2000, 2000))
            r = asyncio.run(ClaudeCodeAdapter().transcript_lines())
            assert r["lines"] == [{"src": "new"}]
            assert r["source"].endswith("new.jsonl")
        finally:
            del os.environ["CLAUDE_PROJECT_CWD"]


def test_claude_code_skips_malformed_lines():
    """Bad JSON lines surface as ``_parse_error: True`` rather than 500'ing."""
    from adapters.claude_code.adapter import ClaudeCodeAdapter
    with tempfile.TemporaryDirectory() as tmp:
        os.environ["HOME"] = tmp
        os.environ["CLAUDE_PROJECT_CWD"] = "/configs"
        try:
            projdir = Path(tmp) / ".claude" / "projects" / "-configs"
            projdir.mkdir(parents=True)
            with (projdir / "x.jsonl").open("w") as f:
                f.write('{"good": 1}\n')
                f.write("not-json garbage\n")
                f.write('{"good": 2}\n')
            r = asyncio.run(ClaudeCodeAdapter().transcript_lines())
            assert r["lines"][0] == {"good": 1}
            assert r["lines"][1].get("_parse_error") is True
            assert r["lines"][2] == {"good": 2}
        finally:
            del os.environ["CLAUDE_PROJECT_CWD"]


def test_claude_code_caps_limit():
    """Limit is capped at 1000 to prevent OOM via paranoid client."""
    from adapters.claude_code.adapter import ClaudeCodeAdapter
    with tempfile.TemporaryDirectory() as tmp:
        os.environ["HOME"] = tmp
        os.environ["CLAUDE_PROJECT_CWD"] = "/configs"
        try:
            projdir = Path(tmp) / ".claude" / "projects" / "-configs"
            projdir.mkdir(parents=True)
            _write_jsonl(projdir / "x.jsonl", [{"i": i} for i in range(1500)])
            r = asyncio.run(ClaudeCodeAdapter().transcript_lines(limit=999999))
            assert len(r["lines"]) == 1000  # capped
            assert r["more"] is True
            assert r["cursor"] == 1000
        finally:
            del os.environ["CLAUDE_PROJECT_CWD"]
