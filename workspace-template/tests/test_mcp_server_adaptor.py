"""Tests for MCPServerAdaptor (#573).

Coverage targets:
  - Subprocess management: start, stop (clean + SIGKILL fallback), call_tool,
    list_tools, timeouts, error responses
  - Security: PATH/LD_PRELOAD/PYTHONPATH override rejected, shell=False enforced,
    absolute path outside allowed prefixes rejected
  - PluginAdaptor interface: install writes mcp-servers.json, blocked env keys
    surface as warnings (never hard-fail), uninstall cleans up entries
  - Helpers: _resolve_env_template, _validate_env_keys, _validate_command
  - plugins.py PluginManifest: mcp_servers field round-trips through YAML load
"""

from __future__ import annotations

import asyncio
import inspect
import json
import logging
import sys
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Path setup — ensure workspace-template root is importable
# ---------------------------------------------------------------------------
import importlib.util

_WS_TEMPLATE = Path(__file__).resolve().parents[1]
if str(_WS_TEMPLATE) not in sys.path:
    sys.path.insert(0, str(_WS_TEMPLATE))

from plugins_registry import InstallContext  # noqa: E402
from plugins_registry.builtins import (  # noqa: E402
    MCPServerAdaptor,
    MCP_SERVERS_CONFIG,
    _resolve_env_template,
    _validate_env_keys,
    _validate_command,
)

# Load the real plugins.py by absolute path, bypassing the conftest mock that
# replaces sys.modules["plugins"] with a MagicMock stub.
_plugins_spec = importlib.util.spec_from_file_location(
    "_real_plugins", str(_WS_TEMPLATE / "plugins.py")
)
_real_plugins = importlib.util.module_from_spec(_plugins_spec)
_plugins_spec.loader.exec_module(_real_plugins)  # type: ignore[union-attr]
_load_plugin_manifest = _real_plugins.load_plugin_manifest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_ctx(configs_dir: Path, plugin_root: Path) -> InstallContext:
    return InstallContext(
        configs_dir=configs_dir,
        workspace_id="ws-mcp-test",
        runtime="claude_code",
        plugin_root=plugin_root,
        logger=logging.getLogger("test"),
    )


def _make_adaptor(**kwargs) -> MCPServerAdaptor:
    """Build a minimal MCPServerAdaptor with sensible defaults."""
    defaults = dict(
        name="test-server",
        command="npx",
        args=["-y", "some-mcp-server"],
        env={},
    )
    defaults.update(kwargs)
    return MCPServerAdaptor(**defaults)


def _fake_process(returncode=None):
    """Return a mock asyncio.subprocess.Process."""
    proc = MagicMock()
    proc.returncode = returncode
    proc.stdin = MagicMock()
    proc.stdin.write = MagicMock()
    proc.stdin.drain = AsyncMock()
    proc.stdout = MagicMock()
    proc.stdout.readline = AsyncMock()
    proc.terminate = MagicMock()
    proc.kill = MagicMock()
    proc.wait = AsyncMock(return_value=0)
    return proc


# ---------------------------------------------------------------------------
# _resolve_env_template
# ---------------------------------------------------------------------------

def test_resolve_env_template_known_var(monkeypatch):
    monkeypatch.setenv("GITHUB_TOKEN", "ghp_abc123")
    assert _resolve_env_template("${GITHUB_TOKEN}") == "ghp_abc123"


def test_resolve_env_template_unknown_var():
    """Unknown variable resolves to empty string (not the raw template)."""
    result = _resolve_env_template("${DEFINITELY_NOT_SET_XYZ_12345}")
    assert result == ""


def test_resolve_env_template_mixed(monkeypatch):
    monkeypatch.setenv("MY_TOKEN", "tok")
    result = _resolve_env_template("Bearer ${MY_TOKEN}")
    assert result == "Bearer tok"


def test_resolve_env_template_no_vars():
    assert _resolve_env_template("plain-value") == "plain-value"


# ---------------------------------------------------------------------------
# _validate_env_keys
# ---------------------------------------------------------------------------

def test_validate_env_keys_clean_env():
    _validate_env_keys({"GITHUB_TOKEN": "ghp_x", "MY_VAR": "val"})  # must not raise


def test_validate_env_keys_rejects_path():
    with pytest.raises(ValueError, match="PATH"):
        _validate_env_keys({"PATH": "/evil/bin:$PATH"})


def test_validate_env_keys_rejects_ld_preload():
    with pytest.raises(ValueError, match="LD_PRELOAD"):
        _validate_env_keys({"LD_PRELOAD": "/evil.so"})


def test_validate_env_keys_rejects_pythonpath():
    with pytest.raises(ValueError, match="PYTHONPATH"):
        _validate_env_keys({"PYTHONPATH": "/evil"})


def test_validate_env_keys_rejects_multiple():
    with pytest.raises(ValueError):
        _validate_env_keys({"PATH": "/evil", "LD_PRELOAD": "/evil.so"})


# ---------------------------------------------------------------------------
# _validate_command
# ---------------------------------------------------------------------------

def test_validate_command_bare_name_allowed():
    _validate_command("npx")          # must not raise
    _validate_command("python3")
    _validate_command("uvx")


def test_validate_command_absolute_allowed_prefix():
    _validate_command("/usr/local/bin/npx")   # known-safe prefix
    _validate_command("/usr/bin/python3")


def test_validate_command_absolute_rejected():
    with pytest.raises(ValueError, match="absolute path outside allowed"):
        _validate_command("/tmp/evil-binary")


def test_validate_command_absolute_slash_bin_rejected():
    """Bare /malicious is not under any known-safe prefix."""
    with pytest.raises(ValueError):
        _validate_command("/malicious")


# ---------------------------------------------------------------------------
# MCPServerAdaptor constructor
# ---------------------------------------------------------------------------

def test_constructor_rejects_path_override():
    with pytest.raises(ValueError, match="PATH"):
        MCPServerAdaptor("s", "npx", [], env={"PATH": "/evil"})


def test_constructor_rejects_bad_command():
    with pytest.raises(ValueError, match="absolute path outside allowed"):
        MCPServerAdaptor("s", "/tmp/evil", [])


def test_constructor_stores_fields():
    a = MCPServerAdaptor(
        "github", "npx", ["-y", "server-github"],
        env={"GITHUB_TOKEN": "tok"},
        plugin_name="molecule-github-mcp",
        runtime="claude_code",
        call_timeout=15.0,
    )
    assert a.name == "github"
    assert a.command == "npx"
    assert a.args == ["-y", "server-github"]
    assert a.env == {"GITHUB_TOKEN": "tok"}
    assert a.plugin_name == "molecule-github-mcp"
    assert a.runtime == "claude_code"
    assert a.call_timeout == 15.0


def test_constructor_plugin_name_defaults_to_name():
    a = MCPServerAdaptor("github", "npx", [])
    assert a.plugin_name == "github"


# ---------------------------------------------------------------------------
# test_shell_false_enforced — verify shell=True is NEVER used in live code
# ---------------------------------------------------------------------------

def test_shell_false_enforced():
    """Executable lines in MCPServerAdaptor must never pass shell=True.

    We strip comment lines and string literals before scanning so that
    security-documenting comments (e.g. '# never shell=True') don't
    cause a false positive.
    """
    import ast
    import plugins_registry.builtins as mod
    source = inspect.getsource(mod.MCPServerAdaptor)
    # Parse to AST and stringify only real keyword calls — no string/comment noise
    tree = ast.parse(source)
    for node in ast.walk(tree):
        if isinstance(node, ast.Call):
            for kw in node.keywords:
                if kw.arg == "shell" and isinstance(kw.value, ast.Constant) and kw.value.value is True:
                    raise AssertionError(
                        f"shell=True found in a function call at line {node.lineno} — "
                        "use asyncio.create_subprocess_exec instead"
                    )


# ---------------------------------------------------------------------------
# start() — test_start_launches_subprocess
# ---------------------------------------------------------------------------

async def test_start_launches_subprocess():
    """start() must call create_subprocess_exec with correct command and args."""
    adaptor = _make_adaptor(
        name="gh",
        command="npx",
        args=["-y", "@modelcontextprotocol/server-github"],
        env={"GITHUB_TOKEN": "tok"},
    )
    proc = _fake_process()

    with patch(
        "plugins_registry.builtins.asyncio.create_subprocess_exec",
        new=AsyncMock(return_value=proc),
    ) as mock_exec:
        await adaptor.start()

    mock_exec.assert_called_once()
    call_args = mock_exec.call_args
    # First positional arg is the command; remaining are *args
    assert call_args.args[0] == "npx"
    assert "-y" in call_args.args
    assert "@modelcontextprotocol/server-github" in call_args.args
    # shell keyword must not be True (create_subprocess_exec has no shell kwarg
    # but we verify it wasn't somehow passed)
    assert call_args.kwargs.get("shell") is not True
    # stdin/stdout/stderr must be PIPE
    assert call_args.kwargs["stdin"] == asyncio.subprocess.PIPE
    assert call_args.kwargs["stdout"] == asyncio.subprocess.PIPE
    assert call_args.kwargs["stderr"] == asyncio.subprocess.PIPE


async def test_start_is_idempotent_when_already_running():
    """Calling start() twice does not launch a second process."""
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=None)  # still running

    with patch(
        "plugins_registry.builtins.asyncio.create_subprocess_exec",
        new=AsyncMock(return_value=proc),
    ) as mock_exec:
        await adaptor.start()
        await adaptor.start()  # second call

    mock_exec.assert_called_once()  # only one subprocess


async def test_start_merges_env_over_os_environ(monkeypatch):
    """Plugin env is merged on top of os.environ, not replacing it."""
    monkeypatch.setenv("EXISTING_VAR", "existing")
    adaptor = _make_adaptor(env={"MY_TOKEN": "secret"})
    proc = _fake_process()

    with patch(
        "plugins_registry.builtins.asyncio.create_subprocess_exec",
        new=AsyncMock(return_value=proc),
    ) as mock_exec:
        await adaptor.start()

    passed_env = mock_exec.call_args.kwargs["env"]
    assert passed_env["EXISTING_VAR"] == "existing"
    assert passed_env["MY_TOKEN"] == "secret"


# ---------------------------------------------------------------------------
# stop() — test_stop_terminates_cleanly + SIGKILL fallback
# ---------------------------------------------------------------------------

async def test_stop_terminates_cleanly():
    """stop() calls terminate() and waits; no kill() when process exits cleanly."""
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=None)
    adaptor._process = proc

    await adaptor.stop()

    proc.terminate.assert_called_once()
    proc.kill.assert_not_called()
    proc.wait.assert_called_once()


async def test_stop_kills_on_timeout():
    """stop() falls back to kill() when terminate() doesn't work within 5 s."""
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=None)
    # After kill(), wait() returns normally (process is dead).
    proc.wait = AsyncMock(return_value=0)
    adaptor._process = proc

    async def _wait_for_raises_timeout(coro, timeout):
        # Cancel the coroutine to avoid "never awaited" ResourceWarning.
        coro.close()
        raise asyncio.TimeoutError

    with patch("plugins_registry.builtins.asyncio.wait_for", new=_wait_for_raises_timeout):
        await adaptor.stop()

    proc.kill.assert_called_once()
    # wait() is called twice: once to build the coroutine for wait_for,
    # once more after kill() to reap the process.
    assert proc.wait.call_count >= 1


async def test_stop_noop_when_not_started():
    """stop() on an adaptor that was never started must not raise."""
    adaptor = _make_adaptor()
    await adaptor.stop()  # _process is None — must be silent


async def test_stop_noop_when_already_exited():
    """stop() when process has already exited must not call terminate."""
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=0)  # already exited
    adaptor._process = proc

    await adaptor.stop()

    proc.terminate.assert_not_called()


# ---------------------------------------------------------------------------
# call_tool() — test_call_tool_sends_jsonrpc
# ---------------------------------------------------------------------------

async def test_call_tool_sends_jsonrpc():
    """call_tool() must write a valid JSON-RPC 2.0 request to the server stdin."""
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=None)
    response_payload = json.dumps({
        "jsonrpc": "2.0", "id": 1, "result": {"content": "ok"}
    }).encode() + b"\n"
    proc.stdout.readline = AsyncMock(return_value=response_payload)
    adaptor._process = proc

    result = await adaptor.call_tool("create_issue", {"title": "Bug fix"})

    # Verify the bytes written to stdin are valid JSON-RPC
    written_bytes: bytes = proc.stdin.write.call_args.args[0]
    request = json.loads(written_bytes.decode())
    assert request["jsonrpc"] == "2.0"
    assert request["method"] == "tools/call"
    assert request["params"]["name"] == "create_issue"
    assert request["params"]["arguments"] == {"title": "Bug fix"}
    assert "id" in request

    # And the result is the inner result dict
    assert result == {"content": "ok"}


async def test_call_tool_raises_on_error_response():
    """A JSON-RPC error from the server must raise RuntimeError."""
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=None)
    error_payload = json.dumps({
        "jsonrpc": "2.0", "id": 1,
        "error": {"code": -32000, "message": "tool not found"},
    }).encode() + b"\n"
    proc.stdout.readline = AsyncMock(return_value=error_payload)
    adaptor._process = proc

    with pytest.raises(RuntimeError, match="tool not found"):
        await adaptor.call_tool("missing_tool", {})


async def test_call_tool_raises_when_not_running():
    """call_tool() before start() must raise RuntimeError immediately."""
    adaptor = _make_adaptor()
    with pytest.raises(RuntimeError, match="not running"):
        await adaptor.call_tool("any_tool", {})


async def test_call_tool_raises_when_process_exited():
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=1)  # exited
    adaptor._process = proc

    with pytest.raises(RuntimeError, match="not running"):
        await adaptor.call_tool("any_tool", {})


async def test_call_tool_raises_on_closed_stdout():
    """If stdout returns empty bytes the server closed unexpectedly."""
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=None)
    proc.stdout.readline = AsyncMock(return_value=b"")  # EOF
    adaptor._process = proc

    with pytest.raises(RuntimeError, match="closed stdout"):
        await adaptor.call_tool("any_tool", {})


# ---------------------------------------------------------------------------
# call_tool() — test_call_tool_timeout
# ---------------------------------------------------------------------------

async def test_call_tool_timeout():
    """call_tool() must raise TimeoutError after the configured timeout."""
    adaptor = _make_adaptor(call_timeout=0.01)
    proc = _fake_process(returncode=None)
    # readline hangs indefinitely
    async def _hang():
        await asyncio.sleep(999)
    proc.stdout.readline = AsyncMock(side_effect=_hang)
    adaptor._process = proc

    with pytest.raises(TimeoutError, match="timed out"):
        await adaptor.call_tool("slow_tool", {})


# ---------------------------------------------------------------------------
# list_tools()
# ---------------------------------------------------------------------------

async def test_list_tools_sends_correct_rpc():
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=None)
    response_payload = json.dumps({
        "jsonrpc": "2.0", "id": 1,
        "result": {"tools": [{"name": "create_issue"}, {"name": "list_prs"}]},
    }).encode() + b"\n"
    proc.stdout.readline = AsyncMock(return_value=response_payload)
    adaptor._process = proc

    tools = await adaptor.list_tools()

    written_bytes: bytes = proc.stdin.write.call_args.args[0]
    request = json.loads(written_bytes.decode())
    assert request["method"] == "tools/list"
    assert tools == [{"name": "create_issue"}, {"name": "list_prs"}]


async def test_list_tools_raises_when_not_running():
    adaptor = _make_adaptor()
    with pytest.raises(RuntimeError, match="not running"):
        await adaptor.list_tools()


# ---------------------------------------------------------------------------
# install() / uninstall()
# ---------------------------------------------------------------------------

async def test_install_writes_mcp_servers_json(tmp_path: Path):
    """install() reads plugin.yaml, validates, and writes mcp-servers.json."""
    plugin_root = tmp_path / "molecule-github-mcp"
    plugin_root.mkdir()
    (plugin_root / "plugin.yaml").write_text(
        "name: molecule-github-mcp\n"
        "mcp_servers:\n"
        "  - name: github\n"
        "    command: npx\n"
        "    args: [\"-y\", \"@modelcontextprotocol/server-github\"]\n"
        "    env:\n"
        "      GITHUB_TOKEN: \"${GITHUB_TOKEN}\"\n"
    )
    configs = tmp_path / "configs"
    configs.mkdir()

    adaptor = MCPServerAdaptor(
        "github", "npx", [], plugin_name="molecule-github-mcp", runtime="claude_code"
    )
    ctx = _make_ctx(configs, plugin_root)

    import os
    with patch.dict(os.environ, {"GITHUB_TOKEN": "ghp_test123"}):
        result = await adaptor.install(ctx)

    assert result.warnings == [], f"unexpected warnings: {result.warnings}"
    assert MCP_SERVERS_CONFIG in result.files_written

    config_path = configs / MCP_SERVERS_CONFIG
    assert config_path.is_file()
    data = json.loads(config_path.read_text())
    assert "github" in data
    assert data["github"]["command"] == "npx"
    assert data["github"]["env"]["GITHUB_TOKEN"] == "ghp_test123"
    assert data["github"]["plugin"] == "molecule-github-mcp"


async def test_install_rejects_blocked_env_keys(tmp_path: Path):
    """A server with PATH in env must surface a warning, not hard-fail install."""
    plugin_root = tmp_path / "bad-plugin"
    plugin_root.mkdir()
    (plugin_root / "plugin.yaml").write_text(
        "name: bad-plugin\n"
        "mcp_servers:\n"
        "  - name: evil\n"
        "    command: npx\n"
        "    args: []\n"
        "    env:\n"
        "      PATH: /evil/bin\n"
    )
    configs = tmp_path / "configs"
    configs.mkdir()

    adaptor = MCPServerAdaptor("evil", "npx", [], plugin_name="bad-plugin")
    result = await adaptor.install(_make_ctx(configs, plugin_root))

    assert any("PATH" in w for w in result.warnings), f"expected PATH warning, got: {result.warnings}"
    # mcp-servers.json must NOT be written (the only server was rejected)
    assert not (configs / MCP_SERVERS_CONFIG).is_file()


async def test_install_rejects_bad_command(tmp_path: Path):
    """A server with an unsafe absolute command path surfaces a warning."""
    plugin_root = tmp_path / "plugin"
    plugin_root.mkdir()
    (plugin_root / "plugin.yaml").write_text(
        "name: p\n"
        "mcp_servers:\n"
        "  - name: s\n"
        "    command: /tmp/evil\n"
        "    args: []\n"
    )
    configs = tmp_path / "configs"
    configs.mkdir()

    adaptor = MCPServerAdaptor("s", "npx", [], plugin_name="p")
    result = await adaptor.install(_make_ctx(configs, plugin_root))

    assert result.warnings
    assert not (configs / MCP_SERVERS_CONFIG).is_file()


async def test_install_no_mcp_servers_is_noop(tmp_path: Path):
    """install() on a plugin with no mcp_servers section is a clean no-op."""
    plugin_root = tmp_path / "agentskills-plugin"
    plugin_root.mkdir()
    (plugin_root / "plugin.yaml").write_text("name: agentskills-plugin\n")
    configs = tmp_path / "configs"
    configs.mkdir()

    adaptor = MCPServerAdaptor("s", "npx", [], plugin_name="agentskills-plugin")
    result = await adaptor.install(_make_ctx(configs, plugin_root))

    assert result.warnings == []
    assert not (configs / MCP_SERVERS_CONFIG).is_file()


async def test_install_no_plugin_yaml_is_noop(tmp_path: Path):
    """install() on a plugin directory without plugin.yaml is a clean no-op."""
    plugin_root = tmp_path / "no-manifest"
    plugin_root.mkdir()
    configs = tmp_path / "configs"
    configs.mkdir()

    adaptor = MCPServerAdaptor("s", "npx", [], plugin_name="no-manifest")
    result = await adaptor.install(_make_ctx(configs, plugin_root))

    assert result.warnings == []
    assert result.files_written == []


async def test_install_merges_with_existing_config(tmp_path: Path):
    """install() must merge new entries with existing mcp-servers.json content."""
    plugin_root = tmp_path / "plugin"
    plugin_root.mkdir()
    (plugin_root / "plugin.yaml").write_text(
        "name: my-plugin\n"
        "mcp_servers:\n"
        "  - name: new-server\n"
        "    command: node\n"
        "    args: [\"server.js\"]\n"
    )
    configs = tmp_path / "configs"
    configs.mkdir()
    # Pre-existing entry from a different plugin
    (configs / MCP_SERVERS_CONFIG).write_text(
        json.dumps({"existing": {"name": "existing", "command": "npx",
                                 "args": [], "env": {}, "plugin": "other-plugin"}})
    )

    adaptor = MCPServerAdaptor("s", "npx", [], plugin_name="my-plugin")
    await adaptor.install(_make_ctx(configs, plugin_root))

    data = json.loads((configs / MCP_SERVERS_CONFIG).read_text())
    assert "existing" in data     # preserved
    assert "new-server" in data   # newly added


async def test_install_recovers_from_corrupt_existing_config(tmp_path: Path):
    """install() treats a corrupt mcp-servers.json as an empty dict (not a crash)."""
    plugin_root = tmp_path / "plugin"
    plugin_root.mkdir()
    (plugin_root / "plugin.yaml").write_text(
        "name: p\nmcp_servers:\n  - name: s\n    command: npx\n    args: []\n"
    )
    configs = tmp_path / "configs"
    configs.mkdir()
    (configs / MCP_SERVERS_CONFIG).write_text("{ INVALID JSON }")

    adaptor = MCPServerAdaptor("s", "npx", [], plugin_name="p")
    result = await adaptor.install(_make_ctx(configs, plugin_root))

    assert result.warnings == []
    data = json.loads((configs / MCP_SERVERS_CONFIG).read_text())
    assert "s" in data  # new entry written successfully


async def test_install_invalid_yaml_surfaces_warning(tmp_path: Path):
    """A malformed plugin.yaml surfaces a warning and does not hard-fail."""
    plugin_root = tmp_path / "plugin"
    plugin_root.mkdir()
    (plugin_root / "plugin.yaml").write_text(": invalid: yaml: {{{")
    configs = tmp_path / "configs"
    configs.mkdir()

    adaptor = MCPServerAdaptor("s", "npx", [], plugin_name="p")
    result = await adaptor.install(_make_ctx(configs, plugin_root))

    assert result.warnings


async def test_uninstall_removes_plugin_entries(tmp_path: Path):
    """uninstall() removes only this plugin's servers, leaving others intact."""
    configs = tmp_path / "configs"
    configs.mkdir()
    (configs / MCP_SERVERS_CONFIG).write_text(json.dumps({
        "github": {"name": "github", "command": "npx", "args": [],
                   "env": {}, "plugin": "molecule-github-mcp"},
        "other": {"name": "other", "command": "npx", "args": [],
                  "env": {}, "plugin": "other-plugin"},
    }))

    adaptor = MCPServerAdaptor(
        "github", "npx", [], plugin_name="molecule-github-mcp"
    )
    plugin_root = tmp_path / "plugin"
    plugin_root.mkdir()
    await adaptor.uninstall(_make_ctx(configs, plugin_root))

    data = json.loads((configs / MCP_SERVERS_CONFIG).read_text())
    assert "github" not in data   # removed
    assert "other" in data        # preserved


async def test_uninstall_noop_when_config_absent(tmp_path: Path):
    """uninstall() is safe when mcp-servers.json doesn't exist."""
    configs = tmp_path / "configs"
    configs.mkdir()
    plugin_root = tmp_path / "plugin"
    plugin_root.mkdir()

    adaptor = MCPServerAdaptor("s", "npx", [], plugin_name="p")
    await adaptor.uninstall(_make_ctx(configs, plugin_root))  # must not raise


async def test_uninstall_noop_when_config_corrupt(tmp_path: Path):
    """uninstall() is safe when mcp-servers.json contains invalid JSON."""
    configs = tmp_path / "configs"
    configs.mkdir()
    (configs / MCP_SERVERS_CONFIG).write_text("{ INVALID JSON }")
    plugin_root = tmp_path / "plugin"
    plugin_root.mkdir()

    adaptor = MCPServerAdaptor("s", "npx", [], plugin_name="p")
    await adaptor.uninstall(_make_ctx(configs, plugin_root))  # must not raise


# ---------------------------------------------------------------------------
# plugins.py PluginManifest — mcp_servers field
# ---------------------------------------------------------------------------

def test_plugin_manifest_mcp_servers_field_roundtrips(tmp_path: Path):
    """PluginManifest.mcp_servers survives plugin.yaml → load_plugin_manifest."""
    plugin_dir = tmp_path / "github-mcp"
    plugin_dir.mkdir()
    (plugin_dir / "plugin.yaml").write_text(
        "name: github-mcp\n"
        "mcp_servers:\n"
        "  - name: github\n"
        "    command: npx\n"
        "    args: [\"-y\", \"@modelcontextprotocol/server-github\"]\n"
        "    env:\n"
        "      GITHUB_TOKEN: \"${GITHUB_TOKEN}\"\n"
    )

    manifest = _load_plugin_manifest(str(plugin_dir))

    assert len(manifest.mcp_servers) == 1
    srv = manifest.mcp_servers[0]
    assert srv["name"] == "github"
    assert srv["command"] == "npx"
    assert srv["args"] == ["-y", "@modelcontextprotocol/server-github"]
    assert srv["env"]["GITHUB_TOKEN"] == "${GITHUB_TOKEN}"


def test_plugin_manifest_mcp_servers_defaults_empty(tmp_path: Path):
    """PluginManifest.mcp_servers defaults to [] when absent from plugin.yaml."""
    plugin_dir = tmp_path / "agentskills"
    plugin_dir.mkdir()
    (plugin_dir / "plugin.yaml").write_text("name: agentskills\n")

    manifest = _load_plugin_manifest(str(plugin_dir))
    assert manifest.mcp_servers == []


def test_plugin_manifest_mcp_servers_empty_when_no_manifest(tmp_path: Path):
    """No plugin.yaml → mcp_servers defaults to []."""
    plugin_dir = tmp_path / "no-manifest"
    plugin_dir.mkdir()

    manifest = _load_plugin_manifest(str(plugin_dir))
    assert manifest.mcp_servers == []


# ---------------------------------------------------------------------------
# Request ID increments — each call gets a unique ID
# ---------------------------------------------------------------------------

async def test_request_ids_are_unique():
    """Each call_tool call must use an incrementing request ID."""
    adaptor = _make_adaptor()
    proc = _fake_process(returncode=None)

    request_ids = []

    async def _capture_and_respond():
        written: bytes = proc.stdin.write.call_args.args[0]
        req = json.loads(written.decode())
        request_ids.append(req["id"])
        return (json.dumps({"jsonrpc": "2.0", "id": req["id"], "result": {}}) + "\n").encode()

    proc.stdout.readline = AsyncMock(side_effect=_capture_and_respond)
    adaptor._process = proc

    await adaptor.call_tool("tool_a", {})
    await adaptor.call_tool("tool_b", {})

    assert len(request_ids) == 2
    assert request_ids[0] != request_ids[1]
    assert request_ids[1] == request_ids[0] + 1
