"""Built-in plugin adaptors — one per agent shape.

The adapter layer is our extensibility surface. Each agent "shape" (form
of installable capability) gets its own named sub-type adapter. A plugin
picks which sub-type to use by importing it as ``Adaptor`` in its
per-runtime file:

.. code-block:: python

    # plugins/<name>/adapters/claude_code.py
    from plugins_registry.builtins import AgentskillsAdaptor as Adaptor

Shape taxonomy (one class per shape; add more as the ecosystem evolves):

* :class:`AgentskillsAdaptor` — skills in the `agentskills.io
  <https://agentskills.io>`_ format (``SKILL.md`` + ``scripts/`` +
  ``references/`` + ``assets/``), plus Molecule AI's optional ``rules/`` and
  root-level prompt fragments at the plugin level. Works on every runtime
  we support (the spec's filesystem layout makes activation trivial on
  Claude Code, our adapter code does the equivalent on DeepAgents /
  LangGraph / etc.). **This is the default and covers the common case.**

Planned as the ecosystem matures (none are implemented yet — rule of
three: promote a class here only after 3+ plugins ship the same custom
shape via their own ``adapters/<runtime>.py``):

* ``MCPServerAdaptor`` — install a plugin as an MCP server *(TODO)*
* ``DeepAgentsSubagentAdaptor`` — register a DeepAgents sub-agent
  (runtime-locked to deepagents) *(TODO)*
* ``LangGraphSubgraphAdaptor`` — install a LangGraph sub-graph *(TODO)*
* ``RAGPipelineAdaptor`` — wire a retriever + index *(TODO)*
* ``SwarmAdaptor`` — bind an OpenAI-swarm / AutoGen-swarm *(TODO)*
* ``WebhookAdaptor`` — register an event handler *(TODO)*

Plugins whose shape doesn't match any built-in ship their own adapter
class in ``plugins/<name>/adapters/<runtime>.py`` — full Python, no
constraint. When 3+ plugins ship the same custom pattern, we promote
the class into this module.
"""

from __future__ import annotations

import asyncio
import json
import os
import re
import shutil
import subprocess
from pathlib import Path

import yaml

from .protocol import SKILLS_SUBDIR, InstallContext, InstallResult

# Files at the plugin root that are never treated as prompt fragments,
# even if they're markdown. Module-level so tests and other adapters can
# import the set rather than re-declaring it.
SKIP_ROOT_MD = frozenset({"readme.md", "changelog.md", "license.md", "contributing.md"})


def _read_md_files(directory: Path) -> list[tuple[str, str]]:
    """Return [(filename, content)] for all *.md files in directory, sorted."""
    if not directory.is_dir():
        return []
    out: list[tuple[str, str]] = []
    for p in sorted(directory.iterdir()):
        if p.is_file() and p.suffix == ".md":
            out.append((p.name, p.read_text().strip()))
    return out


class AgentskillsAdaptor:
    """Sub-type adaptor for `agentskills.io <https://agentskills.io>`_-format skills.

    This is the default adapter for the "skills + rules" shape — the most
    common pattern. A plugin using this adapter ships:

    * ``skills/<name>/SKILL.md`` (+ optional ``scripts/``, ``references/``,
      ``assets/``) — each skill is a spec-compliant agentskills unit,
      portable to Claude Code, Cursor, Codex, and ~35 other skill-compatible
      tools without modification.
    * ``rules/*.md`` (optional, Molecule AI extension) — always-on prose that
      gets appended to the runtime's memory file (CLAUDE.md).
    * Root-level ``*.md`` (optional) — prompt fragments, also appended to
      memory.

    On ``install()``:
      1. Rules → append to ``/configs/<memory_filename>``, wrapped in a
         ``# Plugin: <name>`` marker for idempotent re-install.
      2. Prompt fragments (``*.md`` at plugin root, excl. README/CHANGELOG/etc.)
         → same treatment.
      3. Skills (``skills/<skill_name>/``) → copied to
         ``/configs/skills/<skill_name>/``. Runtimes with native agentskills
         activation (Claude Code) pick them up automatically; other runtimes'
         loaders scan the same path.

    Uninstall reverses the file copies and strips the rule/fragment block by
    marker (best-effort — if the user edited CLAUDE.md manually, only the
    marker line itself is removed).

    For shapes other than agentskills (MCP server, DeepAgents sub-agent,
    LangGraph sub-graph, RAG pipeline, swarm, webhook handler, etc.), see
    the module docstring for the planned sibling adapters, or ship a custom
    adapter class in the plugin's ``adapters/<runtime>.py``.
    """

    def __init__(self, plugin_name: str, runtime: str) -> None:
        self.plugin_name = plugin_name
        self.runtime = runtime

    # ------------------------------------------------------------------
    # install
    # ------------------------------------------------------------------

    async def install(self, ctx: InstallContext) -> InstallResult:
        result = InstallResult(
            plugin_name=self.plugin_name,
            runtime=self.runtime,
            source="plugin",  # overridden by registry caller if source==registry
        )

        # 1. Rules — append to memory file.
        rules = _read_md_files(ctx.plugin_root / "rules")
        # 2. Prompt fragments — any *.md at plugin root except skip list.
        root_fragments: list[tuple[str, str]] = []
        if ctx.plugin_root.is_dir():
            for p in sorted(ctx.plugin_root.iterdir()):
                if p.is_file() and p.suffix == ".md" and p.name.lower() not in SKIP_ROOT_MD:
                    content = p.read_text().strip()
                    if content:
                        root_fragments.append((p.name, content))

        memory_blocks: list[str] = []
        for filename, content in rules:
            memory_blocks.append(f"# Plugin: {self.plugin_name} / rule: {filename}\n\n{content}")
        for filename, content in root_fragments:
            memory_blocks.append(f"# Plugin: {self.plugin_name} / fragment: {filename}\n\n{content}")

        if memory_blocks:
            joined = "\n\n".join(memory_blocks)
            ctx.append_to_memory(ctx.memory_filename, joined)
            ctx.logger.info(
                "%s: injected %d rule+fragment block(s) into %s",
                self.plugin_name, len(memory_blocks), ctx.memory_filename,
            )

        # 3. Skills — copy each skill dir to /configs/skills/.
        src_skills_dir = ctx.plugin_root / "skills"
        if src_skills_dir.is_dir():
            dst_skills_root = ctx.configs_dir / SKILLS_SUBDIR
            dst_skills_root.mkdir(parents=True, exist_ok=True)
            copied = 0
            for entry in sorted(src_skills_dir.iterdir()):
                if not entry.is_dir():
                    continue
                dst = dst_skills_root / entry.name
                if dst.exists():
                    ctx.logger.debug("%s: skill %s already present, skipping", self.plugin_name, entry.name)
                    continue
                shutil.copytree(entry, dst)
                copied += 1
                for p in dst.rglob("*"):
                    if p.is_file():
                        result.files_written.append(str(p.relative_to(ctx.configs_dir)))
            if copied:
                ctx.logger.info("%s: copied %d skill dir(s) to %s", self.plugin_name, copied, dst_skills_root)

        # 4. Setup script — run setup.sh if present (for npm/pip dependencies).
        # Mirrors sdk/python/molecule_plugin/builtins.py — must stay in sync
        # (drift guard: tests/test_plugins_builtins_drift.py).
        setup_script = ctx.plugin_root / "setup.sh"
        if setup_script.is_file():
            ctx.logger.info("%s: running setup.sh", self.plugin_name)
            try:
                proc = subprocess.run(
                    ["bash", str(setup_script)],
                    capture_output=True, text=True, timeout=120,
                    cwd=str(ctx.plugin_root),
                    env={**os.environ, "CONFIGS_DIR": str(ctx.configs_dir)},
                )
                if proc.returncode == 0:
                    ctx.logger.info("%s: setup.sh completed successfully", self.plugin_name)
                else:
                    result.warnings.append(f"setup.sh exited {proc.returncode}: {proc.stderr[:200]}")
                    ctx.logger.warning("%s: setup.sh failed: %s", self.plugin_name, proc.stderr[:200])
            except subprocess.TimeoutExpired:
                result.warnings.append("setup.sh timed out (120s)")
                ctx.logger.warning("%s: setup.sh timed out", self.plugin_name)

        # 5. Hooks — copy hooks/* into <configs>/.claude/hooks/ (Claude Code-
        #    style harness hooks). No-op when the plugin doesn't ship any.
        # 6. Commands — copy commands/*.md into <configs>/.claude/commands/.
        # 7. settings-fragment.json — merge into <configs>/.claude/settings.json,
        #    rewriting ${CLAUDE_DIR} to the absolute install path. Existing
        #    user hooks are preserved (deep-merge by event).
        _install_claude_layer(ctx, result, self.plugin_name)

        return result

    # ------------------------------------------------------------------
    # uninstall
    # ------------------------------------------------------------------

    async def uninstall(self, ctx: InstallContext) -> None:
        # Remove copied skill dirs.
        src_skills_dir = ctx.plugin_root / "skills"
        if src_skills_dir.is_dir():
            for entry in src_skills_dir.iterdir():
                dst = ctx.configs_dir / SKILLS_SUBDIR / entry.name
                if dst.exists() and dst.is_dir():
                    shutil.rmtree(dst)
                    ctx.logger.info("%s: removed %s", self.plugin_name, dst)

        # Best-effort strip of our markers from CLAUDE.md. Users can always
        # edit manually; we only guarantee the injected block's first line
        # is removed so re-install re-adds cleanly.
        memory_path = ctx.configs_dir / ctx.memory_filename
        if not memory_path.exists():
            return
        text = memory_path.read_text()
        prefix = f"# Plugin: {self.plugin_name} / "
        lines = text.splitlines(keepends=True)
        kept = [line for line in lines if not line.startswith(prefix)]
        if len(kept) != len(lines):
            memory_path.write_text("".join(kept))
            ctx.logger.info("%s: stripped markers from %s", self.plugin_name, ctx.memory_filename)




# ----------------------------------------------------------------------
# MCPServerAdaptor — install a plugin as one or more MCP servers (#573).
#
# Wears two hats:
#   1. PluginAdaptor (install/uninstall): reads mcp_servers from plugin.yaml,
#      validates env keys, resolves ${VAR} templates, writes mcp-servers.json
#      under the workspace's configs dir so executors can launch them.
#   2. Subprocess manager (start/stop/call_tool/list_tools): manages a single
#      MCP server process, communicating via JSON-RPC 2.0 over stdio.
#
# Security constraints (reviewed by Security Auditor):
#   * shell=False always — asyncio.create_subprocess_exec, never shell=True
#   * Blocked env keys: PATH, LD_PRELOAD, PYTHONPATH, and others that could
#     hijack the process; validated at construction time.
#   * Command validation: bare names always allowed; absolute paths must be
#     under known-safe prefixes (/usr/local/bin, ~/.local/bin, etc.)
#   * Stderr from the MCP server process is captured and NEVER forwarded
#     verbatim; only first 200 bytes appear in RuntimeError messages.
#   * call_tool has a configurable timeout (default 30 s).
# ----------------------------------------------------------------------

# Env keys that plugin-supplied env must never override.
_BLOCKED_ENV_KEYS: frozenset[str] = frozenset({
    "PATH",
    "LD_PRELOAD",
    "LD_LIBRARY_PATH",
    "PYTHONPATH",
    "PYTHONHOME",
    "HOME",
    "USER",
    "SHELL",
    "DYLD_LIBRARY_PATH",
    "DYLD_INSERT_LIBRARIES",
})

# Absolute-path prefixes that are safe for MCP server commands.
_ALLOWED_COMMAND_PREFIXES: tuple[str, ...] = (
    "/usr/local/bin/",
    "/usr/bin/",
    "/bin/",
    "/opt/homebrew/bin/",
    str(Path.home() / ".local" / "bin") + "/",
    str(Path.home() / ".nvm") + "/",
    "/nix/",
    "/snap/",
)

# Filename written under configs_dir for all plugin-contributed MCP servers.
MCP_SERVERS_CONFIG = "mcp-servers.json"

# Pattern for ${VAR_NAME} template substitution in env values.
_ENV_TEMPLATE_RE = re.compile(r"\$\{([^}]+)\}")


def _resolve_env_template(value: str) -> str:
    """Expand ``${VAR_NAME}`` patterns using ``os.environ`` (unknown → empty string)."""
    return _ENV_TEMPLATE_RE.sub(lambda m: os.environ.get(m.group(1), ""), value)


def _validate_env_keys(env: dict[str, str]) -> None:
    """Raise ValueError if any key would override a security-sensitive variable."""
    bad = {k for k in env if k.upper() in _BLOCKED_ENV_KEYS}
    if bad:
        raise ValueError(
            f"MCPServerAdaptor: env must not override security-sensitive keys: "
            f"{', '.join(sorted(bad))}"
        )


def _validate_command(command: str) -> None:
    """Raise ValueError for absolute paths outside known-safe locations.

    Bare command names (e.g. ``npx``, ``uvx``) are always allowed because
    they resolve via the subprocess PATH at runtime.  Absolute paths outside
    the known-safe prefix list are rejected to prevent a malicious plugin.yaml
    from running arbitrary binaries.
    """
    p = Path(command)
    if not p.is_absolute():
        return  # bare name — fine
    if not any(str(p).startswith(prefix) for prefix in _ALLOWED_COMMAND_PREFIXES):
        raise ValueError(
            f"MCPServerAdaptor: command '{command}' is an absolute path outside "
            f"allowed locations.  Use a bare command name (e.g. 'npx', 'python3') "
            f"or a path under /usr/local/bin, /usr/bin, or ~/.local/bin."
        )


class MCPServerAdaptor:
    """Adaptor that installs and communicates with an MCP server subprocess.

    **As a PluginAdaptor** (installed by the plugin registry):
    ``install(ctx)`` reads the ``mcp_servers:`` list from the plugin's
    ``plugin.yaml``, validates each server's ``env`` keys, resolves
    ``${VAR}`` templates, and writes / merges entries into
    ``<configs_dir>/mcp-servers.json`` — the file executors read at
    startup to populate their ``mcp_servers`` dict.

    **As a subprocess manager** (used directly by agents or test harnesses):
    Construct with a single server's ``(name, command, args, env)`` and
    call ``start()`` / ``stop()`` / ``call_tool()`` / ``list_tools()``.
    Communication is JSON-RPC 2.0 over stdio (line-delimited).

    Usage as subprocess manager::

        adaptor = MCPServerAdaptor(
            name="github",
            command="npx",
            args=["-y", "@modelcontextprotocol/server-github"],
            env={"GITHUB_TOKEN": os.environ["GITHUB_TOKEN"]},
        )
        await adaptor.start()
        tools = await adaptor.list_tools()
        result = await adaptor.call_tool("create_issue", {"title": "Bug"})
        await adaptor.stop()
    """

    def __init__(
        self,
        name: str,
        command: str,
        args: list[str],
        env: dict[str, str] | None = None,
        *,
        plugin_name: str = "",
        runtime: str = "",
        call_timeout: float = 30.0,
    ) -> None:
        _validate_command(command)
        raw_env = dict(env or {})
        _validate_env_keys(raw_env)

        self.name = name
        self.command = command
        self.args = list(args)
        self.env = raw_env
        # PluginAdaptor protocol attributes
        self.plugin_name = plugin_name or name
        self.runtime = runtime
        self.call_timeout = call_timeout

        self._process: asyncio.subprocess.Process | None = None
        self._request_id: int = 0

    # ------------------------------------------------------------------
    # Subprocess management
    # ------------------------------------------------------------------

    async def start(self) -> None:
        """Launch the MCP server subprocess (shell=False always)."""
        if self._process is not None and self._process.returncode is None:
            return  # already running
        merged_env = {**os.environ, **self.env}
        # asyncio.create_subprocess_exec never uses shell=True
        self._process = await asyncio.create_subprocess_exec(
            self.command, *self.args,
            env=merged_env,
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )

    async def stop(self) -> None:
        """Terminate the subprocess cleanly; SIGKILL after 5 s if it hangs."""
        if self._process is None or self._process.returncode is not None:
            return
        self._process.terminate()
        try:
            await asyncio.wait_for(self._process.wait(), timeout=5.0)
        except asyncio.TimeoutError:
            self._process.kill()
            await self._process.wait()

    async def list_tools(self) -> list[dict]:
        """Return the tool manifest from the MCP server (``tools/list`` RPC)."""
        self._assert_running()
        req = self._build_request("tools/list", {})
        resp = await self._send_request(req)
        return resp.get("result", {}).get("tools", [])

    async def call_tool(self, tool_name: str, arguments: dict) -> dict:
        """Call a tool on the MCP server and return the result dict.

        Raises ``TimeoutError`` if no response arrives within ``call_timeout``
        seconds.  Raises ``RuntimeError`` if the server returns a JSON-RPC
        error object.
        """
        self._assert_running()
        req = self._build_request(
            "tools/call", {"name": tool_name, "arguments": arguments}
        )
        resp = await self._send_request(req)
        if "error" in resp:
            err = resp["error"]
            raise RuntimeError(
                f"MCP tool '{tool_name}' returned error "
                f"{err.get('code')}: {err.get('message')}"
            )
        return resp.get("result", {})

    # ------------------------------------------------------------------
    # PluginAdaptor interface
    # ------------------------------------------------------------------

    async def install(self, ctx: InstallContext) -> InstallResult:
        """Read ``mcp_servers`` from ``plugin.yaml``, validate, and write
        ``mcp-servers.json`` under ``ctx.configs_dir``.

        Env validation errors (e.g. PATH override) are surfaced as
        ``InstallResult.warnings`` so install never hard-fails.
        """
        result = InstallResult(
            plugin_name=self.plugin_name,
            runtime=self.runtime,
            source="plugin",
        )
        manifest_path = ctx.plugin_root / "plugin.yaml"
        if not manifest_path.is_file():
            ctx.logger.info(
                "%s: no plugin.yaml found — skipping MCP server install", self.plugin_name
            )
            return result

        try:
            raw = yaml.safe_load(manifest_path.read_text()) or {}
        except Exception as exc:
            result.warnings.append(f"plugin.yaml parse error: {exc}")
            return result

        mcp_servers_raw: list[dict] = raw.get("mcp_servers", [])
        if not mcp_servers_raw:
            ctx.logger.info(
                "%s: no mcp_servers declared in plugin.yaml", self.plugin_name
            )
            return result

        validated: list[dict] = []
        for srv in mcp_servers_raw:
            srv_name = srv.get("name", "<unnamed>")
            srv_command = srv.get("command", "")
            srv_env = dict(srv.get("env") or {})
            try:
                _validate_command(srv_command)
                _validate_env_keys(srv_env)
            except ValueError as exc:
                result.warnings.append(f"server '{srv_name}': {exc}")
                continue
            validated.append({
                "name": srv_name,
                "command": srv_command,
                "args": list(srv.get("args") or []),
                "env": {k: _resolve_env_template(str(v)) for k, v in srv_env.items()},
                "plugin": self.plugin_name,
            })

        if not validated:
            return result

        config_path = ctx.configs_dir / MCP_SERVERS_CONFIG
        existing: dict[str, dict] = {}
        if config_path.is_file():
            try:
                existing = json.loads(config_path.read_text())
            except Exception:
                existing = {}

        for srv in validated:
            existing[srv["name"]] = srv

        config_path.write_text(json.dumps(existing, indent=2) + "\n")
        result.files_written.append(MCP_SERVERS_CONFIG)
        ctx.logger.info(
            "%s: registered %d MCP server(s) in %s",
            self.plugin_name, len(validated), MCP_SERVERS_CONFIG,
        )
        return result

    async def uninstall(self, ctx: InstallContext) -> None:
        """Remove this plugin's MCP server entries from ``mcp-servers.json``."""
        config_path = ctx.configs_dir / MCP_SERVERS_CONFIG
        if not config_path.is_file():
            return
        try:
            existing: dict[str, dict] = json.loads(config_path.read_text())
        except Exception:
            return
        updated = {
            k: v for k, v in existing.items()
            if v.get("plugin") != self.plugin_name
        }
        if len(updated) != len(existing):
            config_path.write_text(json.dumps(updated, indent=2) + "\n")
            ctx.logger.info(
                "%s: removed MCP server entries from %s",
                self.plugin_name, MCP_SERVERS_CONFIG,
            )

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _assert_running(self) -> None:
        if self._process is None or self._process.returncode is not None:
            raise RuntimeError(
                f"MCPServerAdaptor '{self.name}' is not running; call start() first"
            )

    def _build_request(self, method: str, params: dict) -> bytes:
        self._request_id += 1
        payload = {
            "jsonrpc": "2.0",
            "id": self._request_id,
            "method": method,
            "params": params,
        }
        return (json.dumps(payload) + "\n").encode()

    async def _send_request(self, request_bytes: bytes) -> dict:
        """Write request to stdin, read response from stdout with timeout.

        Stderr is intentionally NOT forwarded — only sanitised substrings
        appear in RuntimeError messages to prevent information leakage.
        """
        assert self._process is not None
        assert self._process.stdin is not None
        assert self._process.stdout is not None

        try:
            async with asyncio.timeout(self.call_timeout):
                self._process.stdin.write(request_bytes)
                await self._process.stdin.drain()
                line = await self._process.stdout.readline()
        except asyncio.TimeoutError:
            raise TimeoutError(
                f"MCPServerAdaptor '{self.name}': call timed out "
                f"after {self.call_timeout}s"
            )

        if not line:
            raise RuntimeError(
                f"MCPServerAdaptor '{self.name}': server closed stdout unexpectedly"
            )
        return json.loads(line.decode())


# ----------------------------------------------------------------------
# Claude Code layer — hooks, slash commands, settings.json fragments.
# Promoted from the molecule-guardrails plugin so any plugin can ship
# these by dropping the right files; no custom adapter needed.
# ----------------------------------------------------------------------

def _install_claude_layer(ctx: InstallContext, result: InstallResult, plugin_name: str) -> None:
    claude_dir = ctx.configs_dir / ".claude"
    claude_dir.mkdir(parents=True, exist_ok=True)

    _copy_dir_files(
        ctx.plugin_root / "hooks",
        claude_dir / "hooks",
        result,
        executable_suffix=".sh",
    )
    _copy_dir_files(
        ctx.plugin_root / "commands",
        claude_dir / "commands",
        result,
        only_suffix=".md",
    )
    _merge_settings_fragment(ctx, claude_dir, result, plugin_name)


def _copy_dir_files(
    src: Path,
    dst: Path,
    result: InstallResult,
    executable_suffix: str | None = None,
    only_suffix: str | None = None,
) -> None:
    if not src.is_dir():
        return
    dst.mkdir(parents=True, exist_ok=True)
    for f in src.iterdir():
        if not f.is_file():
            continue
        if only_suffix and f.suffix != only_suffix:
            # When copying hooks, allow .py companion files alongside .sh
            if not (executable_suffix and f.suffix == ".py"):
                continue
        target = dst / f.name
        shutil.copy2(f, target)
        if executable_suffix and f.suffix == executable_suffix:
            target.chmod(0o755)
        result.files_written.append(str(target.relative_to(target.parents[2])))


def _merge_settings_fragment(
    ctx: InstallContext,
    claude_dir: Path,
    result: InstallResult,
    plugin_name: str,
) -> None:
    fragment_path = ctx.plugin_root / "settings-fragment.json"
    if not fragment_path.is_file():
        return
    try:
        fragment = json.loads(fragment_path.read_text())
    except Exception as e:
        result.warnings.append(f"settings-fragment.json invalid: {e}")
        return

    settings_path = claude_dir / "settings.json"
    if settings_path.is_file():
        try:
            existing = json.loads(settings_path.read_text())
        except Exception:
            existing = {}
    else:
        existing = {}

    rewritten = _rewrite_hook_paths(fragment, claude_dir)
    merged = _deep_merge_hooks(existing, rewritten)
    settings_path.write_text(json.dumps(merged, indent=2) + "\n")
    result.files_written.append(str(settings_path.relative_to(ctx.configs_dir)))
    ctx.logger.info("%s: merged hook config into %s", plugin_name, settings_path)


def _rewrite_hook_paths(fragment: dict, claude_dir: Path) -> dict:
    out = json.loads(json.dumps(fragment))  # deep copy via roundtrip
    for handlers in out.get("hooks", {}).values():
        for handler in handlers:
            for h in handler.get("hooks", []):
                cmd = h.get("command", "")
                h["command"] = cmd.replace("${CLAUDE_DIR}", str(claude_dir))
    return out


def _deep_merge_hooks(existing: dict, fragment: dict) -> dict:
    out = dict(existing)
    out.setdefault("hooks", {})
    for event, handlers in fragment.get("hooks", {}).items():
        out["hooks"].setdefault(event, [])
        out["hooks"][event].extend(handlers)
    for key, val in fragment.items():
        if key == "hooks":
            continue
        out.setdefault(key, val)
    return out
