"""Plugin + skill manifest schema and validators.

Two layers:

1. **Plugin-level** (`plugin.yaml`) — Molecule AI's superset: name, version,
   description, declared `runtimes:`, skill list, rule list. The spec has
   no concept of bundling; this is our own.
2. **Skill-level** (`skills/<skill>/SKILL.md`) — follows the
   `agentskills.io` open standard (name, description, optional license,
   compatibility, metadata, allowed-tools). Validated against the spec
   so our skills are installable in Claude Code, Cursor, Codex, and
   every other skills-compatible agent product.

A plugin that validates locally will also load cleanly in the Molecule AI
platform AND be installable as-is into any agentskills-compatible tool.
"""

from __future__ import annotations

import re
from pathlib import Path
from typing import Any

import yaml

PLUGIN_YAML_SCHEMA: dict[str, Any] = {
    "type": "object",
    "required": ["name"],
    "properties": {
        "name": {"type": "string"},
        "version": {"type": "string"},
        "description": {"type": "string"},
        "author": {"type": "string"},
        "tags": {"type": "array", "items": {"type": "string"}},
        "skills": {"type": "array", "items": {"type": "string"}},
        "rules": {"type": "array", "items": {"type": "string"}},
        "prompt_fragments": {"type": "array", "items": {"type": "string"}},
        "runtimes": {
            "type": "array",
            "items": {"type": "string"},
            "description": "Declared supported runtimes (e.g. claude_code, deepagents).",
        },
    },
}


def validate_manifest(path: str | Path) -> list[str]:
    """Return a list of validation error messages. Empty list = valid.

    Deliberately simple — no jsonschema dependency so SDK consumers don't
    pick up an extra transitive dep just to lint their plugin.
    """
    path = Path(path)
    if not path.is_file():
        return [f"manifest not found: {path}"]

    try:
        raw = yaml.safe_load(path.read_text())
    except yaml.YAMLError as exc:
        return [f"yaml parse error: {exc}"]

    errors: list[str] = []
    if not isinstance(raw, dict):
        return ["manifest root must be a mapping"]

    if "name" not in raw or not isinstance(raw.get("name"), str) or not raw["name"].strip():
        errors.append("`name` is required and must be a non-empty string")

    for field_name in ("tags", "skills", "rules", "prompt_fragments", "runtimes"):
        if field_name in raw and not isinstance(raw[field_name], list):
            errors.append(f"`{field_name}` must be a list")

    if "runtimes" in raw and isinstance(raw["runtimes"], list):
        known = {"claude_code", "deepagents", "langgraph", "crewai", "autogen", "openclaw"}
        for r in raw["runtimes"]:
            if not isinstance(r, str):
                errors.append(f"`runtimes` entry must be string, got {type(r).__name__}")
            elif r.replace("-", "_") not in known:
                errors.append(
                    f"unknown runtime '{r}' — supported: {sorted(known)} "
                    f"(use underscore form, e.g. 'claude_code')"
                )

    return errors


# ---------------------------------------------------------------------------
# agentskills.io spec — SKILL.md validation
# ---------------------------------------------------------------------------

# Spec limits — public so tooling/tests/docs can import them rather than
# duplicate magic numbers. Source: https://agentskills.io/specification
SKILL_NAME_RE = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")
SKILL_NAME_MAX = 64
SKILL_DESC_MAX = 1024
SKILL_COMPAT_MAX = 500


def parse_skill_md(path: str | Path) -> tuple[dict[str, Any], str, list[str]]:
    """Parse a SKILL.md into (frontmatter, body, errors).

    Returns ``({}, "", [error])`` if the file can't be read or doesn't have
    valid frontmatter. Never raises.
    """
    path = Path(path)
    if not path.is_file():
        return {}, "", [f"SKILL.md not found: {path}"]

    text = path.read_text()
    if not text.startswith("---"):
        return {}, text, ["SKILL.md must start with YAML frontmatter (---)"]

    parts = text.split("---", 2)
    if len(parts) < 3:
        return {}, text, ["malformed frontmatter — expected opening and closing '---'"]

    try:
        fm = yaml.safe_load(parts[1]) or {}
    except yaml.YAMLError as exc:
        return {}, parts[2], [f"frontmatter yaml parse error: {exc}"]

    if not isinstance(fm, dict):
        return {}, parts[2], ["frontmatter must be a YAML mapping"]

    return fm, parts[2].strip(), []


def validate_skill(path: str | Path) -> list[str]:
    """Validate a single skill directory against agentskills.io/specification.

    `path` should be the skill directory (its parent of `SKILL.md`). Returns
    an empty list when the skill is spec-compliant.
    """
    path = Path(path)
    if not path.is_dir():
        return [f"skill path is not a directory: {path}"]

    fm, _body, errors = parse_skill_md(path / "SKILL.md")
    if errors:
        return errors

    # name — required
    name = fm.get("name")
    if not name:
        errors.append("`name` is required in SKILL.md frontmatter")
    elif not isinstance(name, str):
        errors.append(f"`name` must be a string, got {type(name).__name__}")
    else:
        if len(name) > SKILL_NAME_MAX:
            errors.append(f"`name` length must be ≤{SKILL_NAME_MAX}, got {len(name)}")
        if not SKILL_NAME_RE.match(name):
            errors.append(
                f"`name` '{name}' must be lowercase alphanumeric with single hyphens, "
                f"no leading/trailing/consecutive hyphens"
            )
        if name != path.name:
            errors.append(
                f"`name` '{name}' must match directory name '{path.name}' "
                f"(agentskills.io spec)"
            )

    # description — required
    desc = fm.get("description")
    if not desc:
        errors.append("`description` is required in SKILL.md frontmatter")
    elif not isinstance(desc, str):
        errors.append(f"`description` must be a string, got {type(desc).__name__}")
    elif len(desc) > SKILL_DESC_MAX:
        errors.append(f"`description` length must be ≤{SKILL_DESC_MAX}, got {len(desc)}")

    # compatibility — optional, ≤500 chars
    compat = fm.get("compatibility")
    if compat is not None:
        if not isinstance(compat, str):
            errors.append(f"`compatibility` must be a string, got {type(compat).__name__}")
        elif len(compat) > SKILL_COMPAT_MAX:
            errors.append(
                f"`compatibility` length must be ≤{SKILL_COMPAT_MAX}, got {len(compat)}"
            )

    # metadata — optional, string→string map
    meta = fm.get("metadata")
    if meta is not None:
        if not isinstance(meta, dict):
            errors.append(f"`metadata` must be a mapping, got {type(meta).__name__}")
        else:
            for k, v in meta.items():
                if not isinstance(k, str):
                    errors.append(f"`metadata` keys must be strings, got {type(k).__name__}")
                # values may be stringified — spec says "string-to-string" but is lenient

    # allowed-tools — optional, space-separated string (experimental in spec)
    allowed = fm.get("allowed-tools")
    if allowed is not None and not isinstance(allowed, str):
        errors.append(f"`allowed-tools` must be a space-separated string, got {type(allowed).__name__}")

    # license — optional, free-form string
    lic = fm.get("license")
    if lic is not None and not isinstance(lic, str):
        errors.append(f"`license` must be a string, got {type(lic).__name__}")

    return errors


def validate_plugin(path: str | Path) -> dict[str, list[str]]:
    """Validate an entire Molecule AI plugin: plugin.yaml + all skills.

    Returns a dict mapping source (``"plugin.yaml"`` or ``"skills/<name>"``)
    to a list of error messages. Empty dict means fully valid.
    """
    path = Path(path)
    results: dict[str, list[str]] = {}

    manifest_errs = validate_manifest(path / "plugin.yaml")
    if manifest_errs:
        results["plugin.yaml"] = manifest_errs

    skills_dir = path / "skills"
    if skills_dir.is_dir():
        for entry in sorted(skills_dir.iterdir()):
            if not entry.is_dir():
                continue
            skill_errs = validate_skill(entry)
            if skill_errs:
                results[f"skills/{entry.name}"] = skill_errs

    return results
