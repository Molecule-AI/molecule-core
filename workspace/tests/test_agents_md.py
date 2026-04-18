"""TDD specification for agents_md.py — AGENTS.md auto-generation (#733).

This file defines the REQUIRED behaviour that the Backend Engineer must
implement. All tests are RED until agents_md.py exists and is correct.

Contract
--------
The generator exposes a single public function::

    from agents_md import generate_agents_md

    generate_agents_md(config_dir: str, output_path: str) -> None

``config_dir``  — directory that contains config.yaml (same convention as
                  ``load_config`` in config.py).
``output_path`` — absolute path where AGENTS.md will be written. The
                  parent directory is guaranteed to exist.

AGENTS.md format (AAIF / Linux Foundation standard)
----------------------------------------------------
The generated file must be valid Markdown with at least these sections::

    # <agent name>

    **Role:** <role field from config.yaml>

    ## Description
    <description from config.yaml>

    ## A2A Endpoint
    <endpoint URL>

    ## MCP Tools
    <tool list or "None">

Any ordering of sections is acceptable; the tests check for presence, not
order.

Environment variables
---------------------
``AGENT_URL`` — when set, overrides the derived endpoint URL
               (``http://localhost:{a2a.port}/a2a`` by default).
"""

import os

import pytest
import yaml

# ---------------------------------------------------------------------------
# The module under test. This import will fail (ModuleNotFoundError) until
# the implementation is written — that is the expected RED state.
# ---------------------------------------------------------------------------
from agents_md import generate_agents_md  # noqa: E402  (module doesn't exist yet)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _write_config(tmp_path, **fields):
    """Write a config.yaml into tmp_path and return the directory path."""
    cfg = tmp_path / "config.yaml"
    cfg.write_text(yaml.dump(fields), encoding="utf-8")
    return str(tmp_path)


def _output_path(tmp_path):
    """Return the canonical output path for AGENTS.md in tests."""
    return str(tmp_path / "AGENTS.md")


# ---------------------------------------------------------------------------
# 1. File existence
# ---------------------------------------------------------------------------

def test_agents_md_exists_after_startup(tmp_path):
    """generate_agents_md() must create AGENTS.md at the given output path.

    This is the most fundamental contract: calling the function must produce
    a file. If this test fails, nothing else matters.
    """
    config_dir = _write_config(
        tmp_path,
        name="Existence Bot",
        description="Tests that the file is created.",
        role="tester",
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)

    assert os.path.isfile(out), (
        f"AGENTS.md was not created at {out}. "
        "generate_agents_md() must write the file before returning."
    )


# ---------------------------------------------------------------------------
# 2. Agent name
# ---------------------------------------------------------------------------

def test_agents_md_contains_name(tmp_path):
    """The generated file must include the agent name from config.yaml.

    The name should appear as a top-level Markdown heading so discovery
    tools can parse it without understanding the full document structure.
    """
    config_dir = _write_config(
        tmp_path,
        name="Research Analyst",
        description="Conducts market research.",
        role="analyst",
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    content = open(out, encoding="utf-8").read()

    assert "Research Analyst" in content, (
        "AGENTS.md must contain the agent name 'Research Analyst' from config.yaml. "
        f"Got:\n{content}"
    )
    # Name should appear in a top-level heading for AAIF compliance.
    assert "# Research Analyst" in content, (
        "Agent name must appear as a top-level Markdown heading (# Research Analyst). "
        f"Got:\n{content}"
    )


# ---------------------------------------------------------------------------
# 3. Role
# ---------------------------------------------------------------------------

def test_agents_md_contains_role(tmp_path):
    """The generated file must include the agent's role from config.yaml.

    The ``role`` field describes what the agent is responsible for in the
    multi-agent organisation. It must appear in the output so peer agents
    and orchestration tools can understand the agent's purpose without
    reading the full system prompt.
    """
    config_dir = _write_config(
        tmp_path,
        name="Code Reviewer",
        description="Reviews pull requests for quality and security.",
        role="Senior Code Reviewer",
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    content = open(out, encoding="utf-8").read()

    assert "Senior Code Reviewer" in content, (
        "AGENTS.md must contain the role 'Senior Code Reviewer' from config.yaml. "
        f"Got:\n{content}"
    )


# ---------------------------------------------------------------------------
# 4. A2A endpoint URL
# ---------------------------------------------------------------------------

def test_agents_md_contains_a2a_endpoint_default(tmp_path):
    """Without AGENT_URL set, the endpoint must default to http://localhost:{port}/a2a.

    The A2A port comes from the ``a2a.port`` field in config.yaml (default 8000).
    This URL is what peer agents use to send tasks to this workspace.
    """
    config_dir = _write_config(
        tmp_path,
        name="Default Port Bot",
        description="Uses default port.",
        role="worker",
        a2a={"port": 8000},
    )
    out = _output_path(tmp_path)

    # Ensure AGENT_URL is not set so we exercise the default derivation.
    env = os.environ.copy()
    env.pop("AGENT_URL", None)

    # Call without AGENT_URL in environment — use monkeypatch-safe approach
    orig = os.environ.pop("AGENT_URL", None)
    try:
        generate_agents_md(config_dir, out)
    finally:
        if orig is not None:
            os.environ["AGENT_URL"] = orig

    content = open(out, encoding="utf-8").read()
    assert "http://localhost:8000/a2a" in content, (
        "AGENTS.md must contain 'http://localhost:8000/a2a' when a2a.port=8000 "
        f"and AGENT_URL is not set. Got:\n{content}"
    )


def test_agents_md_contains_a2a_endpoint_custom_port(tmp_path):
    """When a2a.port is set to a non-default value, the endpoint must reflect it."""
    config_dir = _write_config(
        tmp_path,
        name="Custom Port Bot",
        description="Uses a custom port.",
        role="worker",
        a2a={"port": 9090},
    )
    out = _output_path(tmp_path)

    orig = os.environ.pop("AGENT_URL", None)
    try:
        generate_agents_md(config_dir, out)
    finally:
        if orig is not None:
            os.environ["AGENT_URL"] = orig

    content = open(out, encoding="utf-8").read()
    assert "http://localhost:9090/a2a" in content, (
        "AGENTS.md must derive endpoint from a2a.port — expected "
        f"'http://localhost:9090/a2a'. Got:\n{content}"
    )


def test_agents_md_contains_a2a_endpoint_from_env(tmp_path, monkeypatch):
    """When AGENT_URL env var is set, it must override the derived endpoint.

    This supports production deployments where the agent is behind a proxy
    or load balancer and the internal port is not the public-facing URL.
    """
    monkeypatch.setenv("AGENT_URL", "https://agent.prod.example.com/a2a")

    config_dir = _write_config(
        tmp_path,
        name="Prod Agent",
        description="Production deployment.",
        role="operator",
        a2a={"port": 8000},
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    content = open(out, encoding="utf-8").read()

    assert "https://agent.prod.example.com/a2a" in content, (
        "AGENTS.md must use AGENT_URL env var when set. "
        f"Got:\n{content}"
    )
    # The internal localhost URL must NOT appear when AGENT_URL overrides it.
    assert "localhost:8000" not in content, (
        "AGENTS.md must not contain the internal localhost URL when "
        f"AGENT_URL is set. Got:\n{content}"
    )


# ---------------------------------------------------------------------------
# 5. MCP Tools section
# ---------------------------------------------------------------------------

def test_agents_md_contains_mcp_tools_section(tmp_path):
    """The file must have a dedicated tools section.

    Peer agents need to know what capabilities this agent exposes.
    The section heading must be '## MCP Tools' or '## Tools' (case-insensitive
    match is acceptable, but the heading level must be ##).
    """
    config_dir = _write_config(
        tmp_path,
        name="Tool Agent",
        description="Has some tools.",
        role="specialist",
        tools=["web_search", "code_runner"],
        plugins=["github", "slack"],
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    content = open(out, encoding="utf-8").read()

    has_tools_section = (
        "## MCP Tools" in content
        or "## Tools" in content
        or "## mcp tools" in content.lower()
        or "## tools" in content.lower()
    )
    assert has_tools_section, (
        "AGENTS.md must contain a '## MCP Tools' or '## Tools' section. "
        f"Got:\n{content}"
    )


def test_agents_md_tools_section_lists_configured_tools(tmp_path):
    """Tools from config.yaml must appear in the tools section of AGENTS.md.

    When tools and plugins are configured, their names must be enumerated
    so peer agents know what they can request this agent to do.
    """
    config_dir = _write_config(
        tmp_path,
        name="Multi-Tool Agent",
        description="Has multiple tools.",
        role="specialist",
        tools=["web_search", "code_runner"],
        plugins=["github"],
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    content = open(out, encoding="utf-8").read()

    for tool in ("web_search", "code_runner", "github"):
        assert tool in content, (
            f"AGENTS.md must list tool/plugin '{tool}' from config.yaml. "
            f"Got:\n{content}"
        )


def test_agents_md_tools_section_no_tools_shows_none(tmp_path):
    """When no tools or plugins are configured, the section must say 'None'.

    An empty tools section with no content would be ambiguous — the
    implementation must explicitly indicate no tools are available.
    """
    config_dir = _write_config(
        tmp_path,
        name="Bare Agent",
        description="No tools at all.",
        role="basic",
        tools=[],
        plugins=[],
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    content = open(out, encoding="utf-8").read()

    # "None" (case-insensitive) should appear near/in the tools section
    assert "none" in content.lower() or "no tools" in content.lower(), (
        "AGENTS.md must indicate no tools (e.g. 'None') when tools and plugins "
        f"are empty. Got:\n{content}"
    )


# ---------------------------------------------------------------------------
# 6. Regeneration on config change
# ---------------------------------------------------------------------------

def test_agents_md_regenerates_on_config_change(tmp_path):
    """Calling generate_agents_md() again after updating config.yaml must
    overwrite AGENTS.md with the new values.

    This is critical for the hot-reload use case: when an admin updates
    config.yaml (e.g., changes the agent's role), the next call to
    generate_agents_md() must reflect the change without any manual cleanup.
    """
    config_dir = _write_config(
        tmp_path,
        name="Mutable Agent",
        description="First generation.",
        role="junior analyst",
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    content_v1 = open(out, encoding="utf-8").read()
    assert "junior analyst" in content_v1, "First generation must contain initial role."

    # Update config.yaml with a new role.
    _write_config(
        tmp_path,
        name="Mutable Agent",
        description="Second generation.",
        role="senior analyst",
    )

    generate_agents_md(config_dir, out)
    content_v2 = open(out, encoding="utf-8").read()

    assert "senior analyst" in content_v2, (
        "AGENTS.md must reflect the updated role after re-generation. "
        f"Got:\n{content_v2}"
    )
    assert "junior analyst" not in content_v2, (
        "AGENTS.md must not contain the old role after re-generation. "
        f"Got:\n{content_v2}"
    )


# ---------------------------------------------------------------------------
# 7. Valid Markdown
# ---------------------------------------------------------------------------

def test_agents_md_valid_markdown(tmp_path):
    """The generated file must be valid Markdown by a structural heuristic.

    Full Markdown parsing is out of scope for unit tests. We apply three
    structural checks that catch the most common generation bugs:

    1. The file is non-empty.
    2. The first non-blank line starts with ``#`` (top-level heading).
    3. The file has at least 3 lines of content (not just a heading).

    These rules match the minimum AAIF AGENTS.md structure.
    """
    config_dir = _write_config(
        tmp_path,
        name="Markdown Agent",
        description="Tests Markdown validity.",
        role="validator",
        tools=["linter"],
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    raw = open(out, encoding="utf-8").read()

    # Rule 1: non-empty
    assert raw.strip(), "AGENTS.md must not be empty."

    # Rule 2: first non-blank line is a top-level heading
    lines = [ln for ln in raw.splitlines() if ln.strip()]
    assert lines[0].startswith("#"), (
        f"AGENTS.md must start with a Markdown heading (#). "
        f"First non-blank line: {lines[0]!r}"
    )

    # Rule 3: at least 3 non-blank lines (heading + at least 2 content lines)
    assert len(lines) >= 3, (
        f"AGENTS.md must have at least 3 non-blank lines (heading + content). "
        f"Got {len(lines)} line(s):\n{raw}"
    )


def test_agents_md_has_multiple_sections(tmp_path):
    """The generated file must contain multiple ## sections.

    A single-section document would not satisfy the AAIF standard which
    requires separate sections for at least description, endpoint, and tools.
    """
    config_dir = _write_config(
        tmp_path,
        name="Sectioned Agent",
        description="Has multiple sections.",
        role="organiser",
        tools=["planner"],
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    content = open(out, encoding="utf-8").read()

    section_headings = [
        ln for ln in content.splitlines() if ln.startswith("## ")
    ]
    assert len(section_headings) >= 2, (
        f"AGENTS.md must have at least 2 '## ' section headings. "
        f"Found {len(section_headings)}: {section_headings}\nFull content:\n{content}"
    )


# ---------------------------------------------------------------------------
# 8. Edge cases
# ---------------------------------------------------------------------------

def test_agents_md_missing_role_uses_description(tmp_path):
    """When ``role`` is absent from config.yaml, fall back to description.

    Not all existing config.yaml files will have a ``role`` field. The
    generator must degrade gracefully and use ``description`` as the
    capability summary rather than writing an empty role field.
    """
    config_dir = _write_config(
        tmp_path,
        name="Legacy Agent",
        description="Does legacy things.",
        # no 'role' key
    )
    out = _output_path(tmp_path)

    generate_agents_md(config_dir, out)
    content = open(out, encoding="utf-8").read()

    # Either the description or some non-empty capability summary must appear.
    assert "Does legacy things." in content or "Legacy Agent" in content, (
        "AGENTS.md must still contain meaningful content when 'role' is absent. "
        f"Got:\n{content}"
    )


def test_agents_md_special_characters_in_name(tmp_path):
    """Agent names with special Markdown characters must not break the file.

    Names like 'R&D Agent' or 'Agent [Alpha]' contain characters that have
    special meaning in Markdown. The generator must handle them safely.
    """
    config_dir = _write_config(
        tmp_path,
        name="R&D Agent [Alpha]",
        description="Research and development.",
        role="researcher",
    )
    out = _output_path(tmp_path)

    # Must not raise an exception.
    generate_agents_md(config_dir, out)
    content = open(out, encoding="utf-8").read()

    # The name text must appear (exact escaping strategy is implementation's choice).
    assert "R&D Agent" in content or "R&#" in content, (
        "Agent name with special characters must appear in AGENTS.md. "
        f"Got:\n{content}"
    )

    # File must still start with a heading.
    first_nonempty = next(ln for ln in content.splitlines() if ln.strip())
    assert first_nonempty.startswith("#"), (
        "AGENTS.md must still start with a heading when name has special chars. "
        f"First line: {first_nonempty!r}"
    )
