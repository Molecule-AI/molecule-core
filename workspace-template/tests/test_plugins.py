"""Tests for plugins.py — plugin loading system."""

import importlib
import os
import sys

# conftest.py installs a mock 'plugins' module; reload the real one
_ws_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_real_spec = importlib.util.spec_from_file_location(
    "plugins", os.path.join(_ws_root, "plugins.py")
)
_real_plugins = importlib.util.module_from_spec(_real_spec)
_real_spec.loader.exec_module(_real_plugins)

load_plugins = _real_plugins.load_plugins
LoadedPlugins = _real_plugins.LoadedPlugins


def test_load_plugins_empty_dir(tmp_path):
    """No plugins in directory returns empty LoadedPlugins."""
    result = load_plugins(str(tmp_path))
    assert isinstance(result, LoadedPlugins)
    assert result.rules == []
    assert result.prompt_fragments == []
    assert result.skill_dirs == []
    assert result.plugin_names == []


def test_load_plugins_nonexistent_dir():
    """Non-existent directory returns empty LoadedPlugins."""
    result = load_plugins("/nonexistent/path/to/plugins")
    assert isinstance(result, LoadedPlugins)
    assert result.rules == []
    assert result.plugin_names == []


def test_load_plugins_with_rules(tmp_path):
    """Plugin with rules/*.md files loads rule content."""
    plugin_dir = tmp_path / "my-plugin"
    rules_dir = plugin_dir / "rules"
    rules_dir.mkdir(parents=True)

    (rules_dir / "rule1.md").write_text("Always be concise.")
    (rules_dir / "rule2.md").write_text("Never use jargon.")
    # Non-md file should be ignored
    (rules_dir / "notes.txt").write_text("This should be ignored.")

    result = load_plugins(str(tmp_path))

    assert "my-plugin" in result.plugin_names
    assert len(result.rules) == 2
    assert "Always be concise." in result.rules
    assert "Never use jargon." in result.rules


def test_load_plugins_with_rules_empty_content(tmp_path):
    """Empty rule files are skipped."""
    plugin_dir = tmp_path / "empty-rules-plugin"
    rules_dir = plugin_dir / "rules"
    rules_dir.mkdir(parents=True)

    (rules_dir / "empty.md").write_text("")
    (rules_dir / "whitespace.md").write_text("   \n\n  ")

    result = load_plugins(str(tmp_path))

    assert "empty-rules-plugin" in result.plugin_names
    assert len(result.rules) == 0


def test_load_plugins_with_skills(tmp_path):
    """Plugin with skills/ directory registers the skills dir."""
    plugin_dir = tmp_path / "skill-plugin"
    skills_dir = plugin_dir / "skills"
    skill_a = skills_dir / "skill-a"
    skill_b = skills_dir / "skill-b"
    skill_a.mkdir(parents=True)
    skill_b.mkdir(parents=True)

    # Add a file in skills dir (not a subdir — should not count as skill)
    (skills_dir / "readme.txt").write_text("info")

    result = load_plugins(str(tmp_path))

    assert "skill-plugin" in result.plugin_names
    assert len(result.skill_dirs) == 1
    assert result.skill_dirs[0] == str(skills_dir)


def test_load_plugins_with_prompt_fragments(tmp_path):
    """Plugin with .md files in root loads them as prompt fragments."""
    plugin_dir = tmp_path / "prompt-plugin"
    plugin_dir.mkdir()

    (plugin_dir / "prompt.md").write_text("You are a coding assistant.")
    (plugin_dir / "extra.md").write_text("Always explain your reasoning.")

    # These should be skipped
    (plugin_dir / "README.md").write_text("This is a readme.")
    (plugin_dir / "CHANGELOG.md").write_text("v1.0 release")
    (plugin_dir / "LICENSE.md").write_text("MIT License")
    (plugin_dir / "CONTRIBUTING.md").write_text("How to contribute")

    result = load_plugins(str(tmp_path))

    assert "prompt-plugin" in result.plugin_names
    assert len(result.prompt_fragments) == 2
    assert "You are a coding assistant." in result.prompt_fragments
    assert "Always explain your reasoning." in result.prompt_fragments
    # Verify skipped files are not included
    for frag in result.prompt_fragments:
        assert "readme" not in frag.lower()
        assert "changelog" not in frag.lower()


def test_load_plugins_multiple(tmp_path):
    """Multiple plugins are loaded and sorted by name."""
    for name in ["beta-plugin", "alpha-plugin"]:
        plugin_dir = tmp_path / name
        rules_dir = plugin_dir / "rules"
        rules_dir.mkdir(parents=True)
        (rules_dir / "rule.md").write_text(f"Rule from {name}")

    result = load_plugins(str(tmp_path))

    assert result.plugin_names == ["alpha-plugin", "beta-plugin"]
    assert len(result.rules) == 2


def test_load_plugins_skips_files_in_root(tmp_path):
    """Regular files in the plugins dir (not subdirs) are ignored."""
    (tmp_path / "stray-file.txt").write_text("not a plugin")

    result = load_plugins(str(tmp_path))

    assert result.plugin_names == []


def test_load_plugins_combined(tmp_path):
    """Plugin with rules, skills, and prompt fragments loads everything."""
    plugin_dir = tmp_path / "full-plugin"
    rules_dir = plugin_dir / "rules"
    skills_dir = plugin_dir / "skills" / "my-skill"
    rules_dir.mkdir(parents=True)
    skills_dir.mkdir(parents=True)

    (rules_dir / "guideline.md").write_text("Be thorough.")
    (plugin_dir / "prompt.md").write_text("System instructions here.")

    result = load_plugins(str(tmp_path))

    assert "full-plugin" in result.plugin_names
    assert len(result.rules) == 1
    assert len(result.prompt_fragments) == 1
    assert len(result.skill_dirs) == 1
