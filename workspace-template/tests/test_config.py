"""Tests for config.py — workspace configuration loading."""

import os

import yaml

from config import (
    A2AConfig,
    DelegationConfig,
    SandboxConfig,
    WorkspaceConfig,
    load_config,
)


def test_load_config_basic(tmp_path):
    """load_config reads a YAML file and returns a WorkspaceConfig."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(
        yaml.dump(
            {
                "name": "Test Agent",
                "description": "A test workspace",
                "version": "2.0.0",
                "tier": 3,
                "model": "openai:gpt-4o",
                "skills": ["seo", "writing"],
                "tools": ["delegation", "sandbox"],
                "prompt_files": ["SOUL.md", "TOOLS.md"],
            }
        )
    )

    cfg = load_config(str(tmp_path))
    assert cfg.name == "Test Agent"
    assert cfg.description == "A test workspace"
    assert cfg.version == "2.0.0"
    assert cfg.tier == 3
    assert cfg.model == "openai:gpt-4o"
    assert cfg.skills == ["seo", "writing"]
    assert cfg.tools == ["delegation", "sandbox"]
    assert cfg.prompt_files == ["SOUL.md", "TOOLS.md"]


def test_load_config_defaults(tmp_path):
    """Missing fields fall back to WorkspaceConfig defaults."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({}))

    cfg = load_config(str(tmp_path))
    assert cfg.name == "Workspace"
    assert cfg.description == ""
    assert cfg.version == "1.0.0"
    assert cfg.tier == 1
    assert cfg.model == "anthropic:claude-sonnet-4-6"
    assert cfg.skills == []
    assert cfg.tools == []
    assert cfg.prompt_files == []
    assert cfg.sub_workspaces == []


def test_load_config_model_env_override(tmp_path, monkeypatch):
    """MODEL_PROVIDER env var overrides the model from YAML."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({"model": "openai:gpt-4o"}))

    monkeypatch.setenv("MODEL_PROVIDER", "google:gemini-2.0-flash")
    cfg = load_config(str(tmp_path))
    assert cfg.model == "google:gemini-2.0-flash"


def test_load_config_model_no_env(tmp_path, monkeypatch):
    """Without MODEL_PROVIDER, model comes from YAML."""
    monkeypatch.delenv("MODEL_PROVIDER", raising=False)
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({"model": "openai:gpt-4o"}))

    cfg = load_config(str(tmp_path))
    assert cfg.model == "openai:gpt-4o"


def test_delegation_config_defaults(tmp_path):
    """DelegationConfig nested defaults are applied."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({}))

    cfg = load_config(str(tmp_path))
    assert cfg.delegation.retry_attempts == 3
    assert cfg.delegation.retry_delay == 5.0
    assert cfg.delegation.timeout == 120.0
    assert cfg.delegation.escalate is True


def test_delegation_config_override(tmp_path):
    """Delegation values from YAML override defaults."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(
        yaml.dump(
            {"delegation": {"retry_attempts": 5, "timeout": 60.0, "escalate": False}}
        )
    )

    cfg = load_config(str(tmp_path))
    assert cfg.delegation.retry_attempts == 5
    assert cfg.delegation.timeout == 60.0
    assert cfg.delegation.escalate is False
    # retry_delay still default
    assert cfg.delegation.retry_delay == 5.0


def test_a2a_config_defaults(tmp_path):
    """A2AConfig nested defaults are applied."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({}))

    cfg = load_config(str(tmp_path))
    assert cfg.a2a.port == 8000
    assert cfg.a2a.streaming is True
    assert cfg.a2a.push_notifications is True


def test_a2a_config_override(tmp_path):
    """A2A values from YAML override defaults."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(
        yaml.dump({"a2a": {"port": 9000, "streaming": False}})
    )

    cfg = load_config(str(tmp_path))
    assert cfg.a2a.port == 9000
    assert cfg.a2a.streaming is False
    assert cfg.a2a.push_notifications is True


def test_sandbox_config_defaults(tmp_path):
    """SandboxConfig nested defaults are applied."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({}))

    cfg = load_config(str(tmp_path))
    assert cfg.sandbox.backend == "subprocess"
    assert cfg.sandbox.memory_limit == "256m"
    assert cfg.sandbox.timeout == 30


def test_sandbox_config_override(tmp_path):
    """Sandbox values from YAML override defaults."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(
        yaml.dump({"sandbox": {"backend": "docker", "memory_limit": "512m", "timeout": 60}})
    )

    cfg = load_config(str(tmp_path))
    assert cfg.sandbox.backend == "docker"
    assert cfg.sandbox.memory_limit == "512m"
    assert cfg.sandbox.timeout == 60


def test_load_config_file_not_found(tmp_path):
    """load_config raises FileNotFoundError when config.yaml is missing."""
    import pytest

    with pytest.raises(FileNotFoundError):
        load_config(str(tmp_path))


def test_load_config_env_path(tmp_path, monkeypatch):
    """load_config reads from WORKSPACE_CONFIG_PATH env var when no arg given."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({"name": "EnvAgent"}))

    monkeypatch.setenv("WORKSPACE_CONFIG_PATH", str(tmp_path))
    cfg = load_config()  # no argument
    assert cfg.name == "EnvAgent"


def test_initial_prompt_inline(tmp_path):
    """initial_prompt reads inline string from YAML."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({"initial_prompt": "Wake up and clone the repo"}))

    cfg = load_config(str(tmp_path))
    assert cfg.initial_prompt == "Wake up and clone the repo"


def test_initial_prompt_from_file(tmp_path):
    """initial_prompt_file reads prompt from a file."""
    prompt_file = tmp_path / "init.md"
    prompt_file.write_text("Clone repo and read CLAUDE.md")
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({"initial_prompt_file": "init.md"}))

    cfg = load_config(str(tmp_path))
    assert cfg.initial_prompt == "Clone repo and read CLAUDE.md"


def test_initial_prompt_inline_overrides_file(tmp_path):
    """Inline initial_prompt takes precedence over initial_prompt_file."""
    prompt_file = tmp_path / "init.md"
    prompt_file.write_text("From file")
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({
        "initial_prompt": "From inline",
        "initial_prompt_file": "init.md",
    }))

    cfg = load_config(str(tmp_path))
    assert cfg.initial_prompt == "From inline"


def test_initial_prompt_default_empty(tmp_path):
    """initial_prompt defaults to empty string when not specified."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({}))

    cfg = load_config(str(tmp_path))
    assert cfg.initial_prompt == ""


def test_initial_prompt_file_missing(tmp_path):
    """initial_prompt_file gracefully handles missing file."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({"initial_prompt_file": "nonexistent.md"}))

    cfg = load_config(str(tmp_path))
    assert cfg.initial_prompt == ""


def test_shared_context_default(tmp_path):
    """shared_context defaults to empty list when not specified in YAML."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(yaml.dump({}))

    cfg = load_config(str(tmp_path))
    assert cfg.shared_context == []


def test_shared_context_from_yaml(tmp_path):
    """shared_context reads file paths from YAML."""
    config_yaml = tmp_path / "config.yaml"
    config_yaml.write_text(
        yaml.dump({"shared_context": ["guidelines.md", "architecture.md"]})
    )

    cfg = load_config(str(tmp_path))
    assert cfg.shared_context == ["guidelines.md", "architecture.md"]
