"""Tests for the SDK's workspace/org/channel validators + CLI dispatch."""
from __future__ import annotations

import json
from pathlib import Path

import pytest
import yaml

from molecule_plugin import (
    SUPPORTED_CHANNEL_TYPES,
    SUPPORTED_RUNTIMES,
    validate_channel_config,
    validate_channel_file,
    validate_org_template,
    validate_workspace_template,
)
from molecule_plugin.__main__ import main as cli_main


# ---------- workspace ----------

def _write_yaml(path: Path, data) -> None:
    path.write_text(yaml.safe_dump(data))


def test_workspace_happy(tmp_path: Path):
    _write_yaml(
        tmp_path / "config.yaml",
        {"name": "x", "runtime": "claude-code", "tier": 2,
         "runtime_config": {"required_env": ["FOO"], "timeout": 30}},
    )
    assert validate_workspace_template(tmp_path) == []


def test_workspace_missing_file(tmp_path: Path):
    errs = validate_workspace_template(tmp_path)
    assert len(errs) == 1 and "missing config.yaml" in errs[0].message


def test_workspace_bad_yaml(tmp_path: Path):
    (tmp_path / "config.yaml").write_text("foo: [bar\n")
    errs = validate_workspace_template(tmp_path)
    assert any("invalid YAML" in e.message for e in errs)


def test_workspace_not_object(tmp_path: Path):
    (tmp_path / "config.yaml").write_text("- a\n- b\n")
    errs = validate_workspace_template(tmp_path)
    assert any("must be a YAML object" in e.message for e in errs)


def test_workspace_validation_errors(tmp_path: Path):
    _write_yaml(
        tmp_path / "config.yaml",
        {"name": "", "runtime": "wat", "tier": 9,
         "runtime_config": {"required_env": "nope", "timeout": "soon"}},
    )
    msgs = [e.message for e in validate_workspace_template(tmp_path)]
    assert any("missing required field: name" in m for m in msgs)
    assert any("runtime=" in m for m in msgs)
    assert any("tier must be 1, 2, or 3" in m for m in msgs)
    assert any("required_env" in m for m in msgs)
    assert any("timeout" in m for m in msgs)


def test_workspace_runtime_config_not_dict(tmp_path: Path):
    _write_yaml(
        tmp_path / "config.yaml",
        {"name": "x", "runtime": "langgraph", "runtime_config": "nope"},
    )
    msgs = [e.message for e in validate_workspace_template(tmp_path)]
    assert any("runtime_config must be an object" in m for m in msgs)


def test_workspace_runtime_config_none_ok(tmp_path: Path):
    _write_yaml(tmp_path / "config.yaml", {"name": "x", "runtime": "langgraph", "runtime_config": None})
    assert validate_workspace_template(tmp_path) == []


def test_org_defaults_none_ok(tmp_path: Path):
    _write_yaml(tmp_path / "org.yaml", {"name": "T", "defaults": None, "workspaces": [{"name": "a"}]})
    assert validate_org_template(tmp_path) == []


def test_supported_runtimes_contains_known():
    assert "claude-code" in SUPPORTED_RUNTIMES
    assert "deepagents" in SUPPORTED_RUNTIMES


# ---------- org ----------

def test_org_happy(tmp_path: Path):
    _write_yaml(
        tmp_path / "org.yaml",
        {
            "name": "T",
            "defaults": {"runtime": "claude-code"},
            "workspaces": [
                {
                    "name": "PM",
                    "tier": 3,
                    "runtime": "claude-code",
                    "workspace_access": "read_only",
                    "workspace_dir": "/repo",
                    "channels": [{"type": "telegram", "config": {"bot_token": "x"}}],
                    "schedules": [{"cron_expr": "* * * * *", "prompt": "hi"}],
                    "plugins": ["molecule-dev"],
                    "children": [{"name": "Dev"}],
                }
            ],
        },
    )
    assert validate_org_template(tmp_path) == []


def test_org_missing_file(tmp_path: Path):
    errs = validate_org_template(tmp_path)
    assert any("missing org.yaml" in e.message for e in errs)


def test_org_bad_yaml(tmp_path: Path):
    (tmp_path / "org.yaml").write_text("foo: [bar\n")
    errs = validate_org_template(tmp_path)
    assert any("invalid YAML" in e.message for e in errs)


def test_org_not_object(tmp_path: Path):
    (tmp_path / "org.yaml").write_text("- a\n")
    errs = validate_org_template(tmp_path)
    assert any("must be a YAML object" in e.message for e in errs)


def test_org_various_errors(tmp_path: Path):
    _write_yaml(
        tmp_path / "org.yaml",
        {
            "defaults": "nope",
            "workspaces": [
                "notadict",
                {
                    "name": "",
                    "tier": 8,
                    "runtime": "wat",
                    "workspace_access": "invalid",
                    "channels": "nope",
                    "schedules": "nope",
                    "plugins": [1, 2],
                    "external": True,
                },
                {
                    "name": "y",
                    "workspace_access": "read_write",  # but no workspace_dir
                    "channels": ["bad", {"config": "nope"}],
                    "schedules": ["bad", {}],
                    "children": "nope",
                },
                {
                    "name": "z",
                    "children": [{"name": "c"}, "bad"],
                },
            ],
        },
    )
    msgs = [e.message for e in validate_org_template(tmp_path)]
    joined = "\n".join(msgs)
    assert "missing required field: name" in joined
    assert "defaults must be an object" in joined
    assert "tier must be 1, 2, or 3" in joined
    assert "runtime=" in joined
    assert "workspace_access=" in joined
    assert "requires workspace_dir" in joined
    assert ".channels: must be a list" in joined
    assert ".schedules: must be a list" in joined
    assert "plugins: must be a list of strings" in joined
    assert "external=true requires url" in joined
    assert "missing required 'type'" in joined or "must be an object" in joined
    assert "missing 'cron_expr'" in joined
    assert "missing 'prompt'" in joined
    assert ".children: must be a list" in joined
    assert "must be an object" in joined


def test_org_missing_workspaces(tmp_path: Path):
    _write_yaml(tmp_path / "org.yaml", {"name": "T"})
    msgs = [e.message for e in validate_org_template(tmp_path)]
    assert any("missing required field: workspaces" in m for m in msgs)


def test_org_workspaces_not_list(tmp_path: Path):
    _write_yaml(tmp_path / "org.yaml", {"name": "T", "workspaces": "nope"})
    msgs = [e.message for e in validate_org_template(tmp_path)]
    assert any("workspaces must be a list" in m for m in msgs)


# ---------- channel ----------

def test_channel_config_happy():
    assert validate_channel_config({
        "type": "telegram",
        "config": {"bot_token": "x"},
        "enabled": True,
    }) == []


def test_channel_config_missing_type():
    errs = validate_channel_config({})
    assert any("missing required field: type" in e.message for e in errs)


def test_channel_config_unsupported_type():
    errs = validate_channel_config({"type": "fax"})
    assert any("must be one of" in e.message for e in errs)


def test_channel_config_bad_config_type():
    errs = validate_channel_config({"type": "telegram", "config": "nope"})
    assert any("config must be an object" in e.message for e in errs)


def test_channel_config_missing_required_key():
    errs = validate_channel_config({"type": "telegram", "config": {}})
    assert any("bot_token is required" in e.message for e in errs)


def test_channel_config_bad_enabled():
    errs = validate_channel_config({"type": "telegram", "config": {"bot_token": "x"}, "enabled": "yes"})
    assert any("enabled must be a boolean" in e.message for e in errs)


def test_channel_file_list(tmp_path: Path):
    p = tmp_path / "channels.yaml"
    p.write_text(yaml.safe_dump([
        {"type": "telegram", "config": {"bot_token": "x"}},
        "notadict",
    ]))
    errs = validate_channel_file(p)
    assert any("must be an object" in e.message for e in errs)


def test_channel_file_single_dict(tmp_path: Path):
    p = tmp_path / "channel.yaml"
    p.write_text(yaml.safe_dump({"type": "telegram", "config": {"bot_token": "x"}}))
    assert validate_channel_file(p) == []


def test_channel_file_missing():
    errs = validate_channel_file(Path("/nonexistent/channel.yaml"))
    assert any("file does not exist" in e.message for e in errs)


def test_channel_file_empty(tmp_path: Path):
    p = tmp_path / "c.yaml"
    p.write_text("")
    errs = validate_channel_file(p)
    assert any("empty" in e.message for e in errs)


def test_channel_file_bad_yaml(tmp_path: Path):
    p = tmp_path / "c.yaml"
    p.write_text("foo: [bar\n")
    errs = validate_channel_file(p)
    assert any("invalid YAML" in e.message for e in errs)


def test_channel_file_wrong_toplevel(tmp_path: Path):
    p = tmp_path / "c.yaml"
    p.write_text("5\n")
    errs = validate_channel_file(p)
    assert any("top-level must be" in e.message for e in errs)


def test_channel_types_exports():
    assert "telegram" in SUPPORTED_CHANNEL_TYPES


# ---------- CLI ----------

def test_cli_workspace_valid(tmp_path, capsys):
    _write_yaml(tmp_path / "config.yaml", {"name": "x", "runtime": "langgraph"})
    assert cli_main(["validate", "workspace", str(tmp_path)]) == 0


def test_cli_workspace_invalid(tmp_path, capsys):
    _write_yaml(tmp_path / "config.yaml", {"name": "", "runtime": ""})
    assert cli_main(["validate", "workspace", str(tmp_path)]) == 1


def test_cli_org_quiet(tmp_path, capsys):
    _write_yaml(tmp_path / "org.yaml", {"name": "T", "workspaces": [{"name": "a"}]})
    assert cli_main(["validate", "org", str(tmp_path), "-q"]) == 0
    out = capsys.readouterr().out
    assert out == ""


def test_cli_channel_valid(tmp_path):
    p = tmp_path / "c.yaml"
    p.write_text(yaml.safe_dump({"type": "telegram", "config": {"bot_token": "x"}}))
    assert cli_main(["validate", "channel", str(p)]) == 0


def test_cli_channel_missing(tmp_path):
    assert cli_main(["validate", "channel", str(tmp_path / "missing.yaml")]) == 1


def test_cli_missing_path(tmp_path):
    assert cli_main(["validate", "workspace", str(tmp_path / "nope")]) == 1


def test_cli_path_not_dir(tmp_path):
    p = tmp_path / "file.txt"
    p.write_text("hi")
    assert cli_main(["validate", "workspace", str(p)]) == 1


def test_cli_plugin_dispatch(tmp_path):
    # Plugin dir missing plugin.yaml -> validator returns errors -> exit 1
    assert cli_main(["validate", "plugin", str(tmp_path)]) == 1
