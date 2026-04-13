"""Startup preflight checks for workspace runtime configs."""

import os
from dataclasses import dataclass, field
from pathlib import Path

from config import WorkspaceConfig

SUPPORTED_RUNTIMES = {
    "langgraph",
    "claude-code",
    "codex",
    "ollama",
    "custom",
    "crewai",
    "autogen",
    "deepagents",
    "openclaw",
}


@dataclass
class PreflightIssue:
    severity: str
    title: str
    detail: str
    fix: str = ""


@dataclass
class PreflightReport:
    warnings: list[PreflightIssue] = field(default_factory=list)
    failures: list[PreflightIssue] = field(default_factory=list)

    @property
    def ok(self) -> bool:
        return not self.failures


def run_preflight(config: WorkspaceConfig, config_path: str) -> PreflightReport:
    """Check the workspace config for obvious startup blockers."""
    report = PreflightReport()
    config_dir = Path(config_path)

    if config.runtime not in SUPPORTED_RUNTIMES:
        report.failures.append(
            PreflightIssue(
                severity="fail",
                title="Runtime",
                detail=f"Unsupported runtime '{config.runtime}'",
                fix="Choose one of the supported runtimes or install the matching adapter.",
            )
        )

    if not 1 <= int(config.a2a.port) <= 65535:
        report.failures.append(
            PreflightIssue(
                severity="fail",
                title="A2A port",
                detail=f"Invalid A2A port: {config.a2a.port}",
                fix="Set a2a.port to a value between 1 and 65535.",
            )
        )

    # Check required environment variables (e.g. CLAUDE_CODE_OAUTH_TOKEN, OPENAI_API_KEY).
    # These are declared per-runtime in config.yaml and injected via the secrets API.
    required_env = getattr(config.runtime_config, "required_env", []) or []
    for env_var in required_env:
        if not os.environ.get(env_var):
            report.failures.append(
                PreflightIssue(
                    severity="fail",
                    title="Required env",
                    detail=f"Missing required environment variable: {env_var}",
                    fix=f"Set {env_var} via the secrets API (global or workspace-level).",
                )
            )

    # Backward compat: if legacy auth_token_file is set, warn but don't block
    # if the token is available via required_env or auth_token_env.
    token_file = getattr(config.runtime_config, "auth_token_file", "")
    if token_file:
        token_path = config_dir / token_file
        if not token_path.exists():
            token_env = getattr(config.runtime_config, "auth_token_env", "")
            env_has_token = bool(token_env and os.environ.get(token_env))
            # Also check if any required_env is set (covers the new path)
            if not env_has_token and required_env:
                env_has_token = all(os.environ.get(e) for e in required_env)

            if not env_has_token:
                report.failures.append(
                    PreflightIssue(
                        severity="fail",
                        title="Auth token",
                        detail=f"Missing auth token file: {token_file}",
                        fix="Remove auth_token_file and use required_env + secrets API instead.",
                    )
                )

    prompt_files = config.prompt_files or ["system-prompt.md"]
    for prompt_file in prompt_files:
        prompt_path = config_dir / prompt_file
        if not prompt_path.exists():
            report.warnings.append(
                PreflightIssue(
                    severity="warn",
                    title="Prompt file",
                    detail=f"Missing prompt file: {prompt_file}",
                    fix="Add the file or remove it from prompt_files.",
                )
            )

    skills_dir = config_dir / "skills"
    for skill_name in config.skills:
        skill_path = skills_dir / skill_name / "SKILL.md"
        if not skill_path.exists():
            report.warnings.append(
                PreflightIssue(
                    severity="warn",
                    title="Skill",
                    detail=f"Missing skill package: {skill_name}",
                    fix="Restore the skill folder or remove it from config.yaml.",
                )
            )

    return report


def render_preflight_report(report: PreflightReport) -> None:
    """Print a concise startup report."""
    if not report.warnings and not report.failures:
        return

    print("Preflight checks:")
    for issue in report.failures:
        print(f"[FAIL] {issue.title}: {issue.detail}")
        if issue.fix:
            print(f"  Fix: {issue.fix}")
    for issue in report.warnings:
        print(f"[WARN] {issue.title}: {issue.detail}")
        if issue.fix:
            print(f"  Fix: {issue.fix}")
