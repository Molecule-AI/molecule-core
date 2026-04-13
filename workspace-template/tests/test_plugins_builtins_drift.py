"""Drift guard: the SDK's AgentskillsAdaptor must stay behaviourally in
sync with the runtime's copy.

The SDK vendors its own copy so plugin authors can unit-test without
depending on workspace-template, but a behavioural divergence would be
silent — a user fixes a rules-injection bug in one copy and the other
goes on emitting the wrong output. This test runs the same install
scenario through both copies and asserts the observable side effects
are identical (CLAUDE.md contents + skill files on disk + InstallResult
payload).
"""

from __future__ import annotations

import logging
import sys
from pathlib import Path


def _add_to_path(p: Path) -> None:
    if str(p) not in sys.path:
        sys.path.insert(0, str(p))


_REPO = Path(__file__).resolve().parents[2]
_add_to_path(_REPO / "workspace-template")
_add_to_path(_REPO / "sdk" / "python")

from plugins_registry.builtins import AgentskillsAdaptor as RuntimeAdaptor  # noqa: E402
from plugins_registry.protocol import InstallContext as RuntimeCtx  # noqa: E402
from molecule_plugin.builtins import AgentskillsAdaptor as SDKAdaptor  # noqa: E402
from molecule_plugin.protocol import InstallContext as SDKCtx  # noqa: E402


def _make_plugin(root: Path) -> Path:
    (root / "rules").mkdir(parents=True)
    (root / "rules" / "r1.md").write_text("- rule one")
    (root / "fragment.md").write_text("frag text")
    (root / "README.md").write_text("skip me")
    (root / "skills" / "s1").mkdir(parents=True)
    (root / "skills" / "s1" / "SKILL.md").write_text(
        "---\nname: s1\ndescription: d\n---\nbody"
    )
    # setup.sh proves both adaptors run the hook (drift guard for the
    # plugin-owned dependency-install step). The marker file lets the
    # test assert the script actually executed.
    setup = root / "setup.sh"
    setup.write_text(
        '#!/bin/bash\nset -e\ntouch "$CONFIGS_DIR/setup-ran-marker"\n'
    )
    setup.chmod(0o755)
    return root


def _memory_sink(store: dict):
    def _append(filename: str, content: str) -> None:
        store.setdefault(filename, "")
        store[filename] = (store[filename] + ("\n" if store[filename] else "") + content + "\n")
    return _append


async def _install(adaptor_cls, ctx_cls, plugin: Path, configs: Path) -> tuple[list[str], dict]:
    mem: dict = {}
    ctx = ctx_cls(
        configs_dir=configs,
        workspace_id="ws",
        runtime="claude_code",
        plugin_root=plugin,
        append_to_memory=_memory_sink(mem),
        logger=logging.getLogger("drift"),
    )
    result = await adaptor_cls("my-plugin", "claude_code").install(ctx)
    return sorted(result.files_written), mem


async def test_sdk_and_runtime_produce_identical_side_effects(tmp_path: Path):
    """SDK.install() and runtime.install() must yield byte-identical
    memory text and skill-file placement for the same input plugin."""
    plugin_runtime = _make_plugin(tmp_path / "plugin-a")
    plugin_sdk = _make_plugin(tmp_path / "plugin-b")
    configs_runtime = tmp_path / "configs-a"
    configs_runtime.mkdir()
    configs_sdk = tmp_path / "configs-b"
    configs_sdk.mkdir()

    rt_files, rt_mem = await _install(RuntimeAdaptor, RuntimeCtx, plugin_runtime, configs_runtime)
    sdk_files, sdk_mem = await _install(SDKAdaptor, SDKCtx, plugin_sdk, configs_sdk)

    assert rt_files == sdk_files, "copied-files lists diverge"
    assert rt_mem == sdk_mem, (
        "CLAUDE.md contents diverge between SDK and runtime AgentskillsAdaptor"
    )
    # Both adaptors must run setup.sh — the marker file proves it. If only
    # one side runs the hook, plugin authors get false-pass unit tests
    # against the SDK while production behaves differently.
    assert (configs_runtime / "setup-ran-marker").is_file(), (
        "runtime AgentskillsAdaptor did not execute setup.sh"
    )
    assert (configs_sdk / "setup-ran-marker").is_file(), (
        "SDK AgentskillsAdaptor did not execute setup.sh — drift from runtime"
    )
