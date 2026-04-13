"""Skill dependency security scanner — supply-chain risk management.

Scans a skill's ``requirements.txt`` for known CVEs before the skill is
loaded into the workspace.  Two scanners are supported:

  Snyk CLI   — ``snyk test --file=requirements.txt --json``
               Preferred; requires the ``snyk`` binary in PATH and
               a SNYK_TOKEN env var for authenticated scans.

  pip-audit  — ``pip-audit -r requirements.txt --json``
               Fallback; no authentication required.

The scanner is auto-selected: Snyk if available, pip-audit otherwise.
If neither is present in PATH the scan is silently skipped with a log line.

Scan mode (``security_scan.mode`` in config.yaml):

  block  — raise ``SkillSecurityError`` when critical/high CVEs are found;
            the skill is *not* loaded.
  warn   — log a WARNING + audit event; the skill is loaded anyway.
  off    — skip scanning entirely; useful in air-gapped CI.

Audit trail
-----------
Every scan (pass or fail) is recorded via ``tools.audit.log_event`` with
``event_type="security_scan"``, enabling compliance reports to prove that
all loaded skills were checked before activation.
"""

from __future__ import annotations

import json
import logging
import shutil
import subprocess
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

from builtin_tools.audit import log_event

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Public exception
# ---------------------------------------------------------------------------


class SkillSecurityError(RuntimeError):
    """Raised when a skill fails security scanning in ``block`` mode.

    The message contains the skill name, scanner used, and a summary of the
    critical/high findings so operators can act on it immediately.
    """


# ---------------------------------------------------------------------------
# Data models
# ---------------------------------------------------------------------------


@dataclass
class CVEFinding:
    """A single vulnerability finding from a security scanner."""

    vuln_id: str
    """CVE or advisory identifier, e.g. ``SNYK-PYTHON-REQUESTS-1234``."""
    package: str
    """Affected package name."""
    version: str
    """Installed version of the package."""
    severity: str
    """One of: critical | high | medium | low | unknown."""
    description: str
    """Short human-readable summary (≤ 200 chars)."""


@dataclass
class ScanResult:
    """Aggregated result of a single skill dependency scan."""

    skill_name: str
    scanner: str
    """Scanner used: ``"snyk"`` | ``"pip-audit"`` | ``"none"``."""
    requirements_file: Optional[str]
    """Absolute path to the scanned requirements.txt, or ``None``."""
    findings: list[CVEFinding] = field(default_factory=list)
    scan_error: Optional[str] = None
    """Non-fatal scanner error (e.g. timeout); findings may be incomplete."""

    @property
    def critical_or_high(self) -> list[CVEFinding]:
        return [f for f in self.findings if f.severity in ("critical", "high")]

    @property
    def has_critical_or_high(self) -> bool:
        return bool(self.critical_or_high)


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------


def _find_requirements(skill_path: Path) -> Optional[Path]:
    """Return the first ``requirements.txt`` found in the skill tree."""
    for candidate in (
        skill_path / "requirements.txt",
        skill_path / "tools" / "requirements.txt",
    ):
        if candidate.exists():
            return candidate
    return None


def _run_scanner(cmd: list[str], timeout: int = 120) -> tuple[str, Optional[str]]:
    """Run a scanner subprocess and return ``(stdout, error_or_None)``."""
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        # Both Snyk and pip-audit exit 1 when vulns are found — not an error.
        # Exit 2 from Snyk means a genuine scan failure.
        if result.returncode == 2 and not result.stdout.strip():
            return "", f"scanner exited 2: {result.stderr.strip()[:200]}"
        return result.stdout, None
    except subprocess.TimeoutExpired:
        return "", f"scanner timed out after {timeout}s"
    except FileNotFoundError as exc:
        return "", str(exc)
    except Exception as exc:  # pylint: disable=broad-except
        return "", str(exc)


def _parse_snyk(stdout: str) -> tuple[list[CVEFinding], Optional[str]]:
    """Parse ``snyk test --json`` output."""
    if not stdout.strip():
        return [], "empty snyk output"
    try:
        data = json.loads(stdout)
    except json.JSONDecodeError as exc:
        return [], f"snyk JSON parse error: {exc}"

    vulns = data.get("vulnerabilities", [])
    findings = [
        CVEFinding(
            vuln_id=v.get("id", "UNKNOWN"),
            package=v.get("packageName", "?"),
            version=v.get("version", "?"),
            severity=v.get("severity", "unknown").lower(),
            description=(v.get("title", "") or "")[:200],
        )
        for v in vulns
        if isinstance(v, dict)
    ]
    return findings, None


def _parse_pip_audit(stdout: str) -> tuple[list[CVEFinding], Optional[str]]:
    """Parse ``pip-audit --json`` output.

    pip-audit does not always provide a CVSS severity level.  When absent we
    conservatively classify the finding as ``"high"`` so it is not silently
    ignored in ``warn`` mode.
    """
    if not stdout.strip():
        return [], "empty pip-audit output"
    try:
        data = json.loads(stdout)
    except json.JSONDecodeError as exc:
        return [], f"pip-audit JSON parse error: {exc}"

    # pip-audit ≥ 2.x wraps results in {"dependencies": [...]}
    if isinstance(data, dict):
        deps = data.get("dependencies", [])
    else:
        deps = data  # older versions return a bare list

    findings: list[CVEFinding] = []
    for dep in deps:
        if not isinstance(dep, dict):
            continue
        for vuln in dep.get("vulns", []):
            sev_raw = vuln.get("fix_versions") and "high"  # pip-audit lacks severity
            sev = (vuln.get("severity") or sev_raw or "high").lower()
            findings.append(
                CVEFinding(
                    vuln_id=vuln.get("id", "UNKNOWN"),
                    package=dep.get("name", "?"),
                    version=dep.get("version", "?"),
                    severity=sev,
                    description=(vuln.get("description", "") or "")[:200],
                )
            )
    return findings, None


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def scan_skill_dependencies(
    skill_name: str,
    skill_path: Path,
    mode: str,
) -> ScanResult:
    """Scan a skill's dependency file for known CVEs.

    Args:
        skill_name: Name of the skill (used in log messages and audit events).
        skill_path: Absolute path to the skill's root directory.
        mode:       ``"block"`` | ``"warn"`` | ``"off"``

    Returns:
        A :class:`ScanResult` describing what was found.

    Raises:
        :class:`SkillSecurityError`: Only when ``mode="block"`` and one or
            more critical/high severity CVEs are found.
    """
    if mode == "off":
        return ScanResult(skill_name=skill_name, scanner="none", requirements_file=None)

    req_file = _find_requirements(skill_path)
    if req_file is None:
        # No requirements file — nothing to scan; not a problem.
        return ScanResult(skill_name=skill_name, scanner="none", requirements_file=None)

    # ── Select scanner ────────────────────────────────────────────────────────
    scanner_name: str
    findings: list[CVEFinding]
    scan_error: Optional[str]

    if shutil.which("snyk"):
        scanner_name = "snyk"
        stdout, run_error = _run_scanner(
            ["snyk", "test", f"--file={req_file}", "--json"]
        )
        if run_error:
            findings, scan_error = [], run_error
        else:
            findings, scan_error = _parse_snyk(stdout)

    elif shutil.which("pip-audit"):
        scanner_name = "pip-audit"
        stdout, run_error = _run_scanner(
            ["pip-audit", "-r", str(req_file), "--json", "--progress-spinner=off"]
        )
        if run_error:
            findings, scan_error = [], run_error
        else:
            findings, scan_error = _parse_pip_audit(stdout)

    else:
        logger.info(
            "security_scan: no scanner (snyk, pip-audit) in PATH — skipping %s",
            skill_name,
        )
        log_event(
            event_type="security_scan",
            action="skill.security_scan",
            resource=skill_name,
            outcome="skipped",
            reason="no_scanner_in_path",
            requirements_file=str(req_file),
            mode=mode,
        )
        return ScanResult(
            skill_name=skill_name,
            scanner="none",
            requirements_file=str(req_file),
            scan_error="No scanner (snyk or pip-audit) found in PATH",
        )

    result = ScanResult(
        skill_name=skill_name,
        scanner=scanner_name,
        requirements_file=str(req_file),
        findings=findings,
        scan_error=scan_error,
    )

    # ── Log scan outcome to audit trail ──────────────────────────────────────
    audit_outcome = "clean" if not result.has_critical_or_high else "vulnerable"
    log_event(
        event_type="security_scan",
        action="skill.security_scan",
        resource=skill_name,
        outcome=audit_outcome,
        scanner=scanner_name,
        requirements_file=str(req_file),
        total_findings=len(findings),
        critical_or_high_count=len(result.critical_or_high),
        scan_error=scan_error,
    )

    if scan_error:
        logger.warning(
            "security_scan: scanner error for skill '%s': %s", skill_name, scan_error
        )

    # ── Enforce mode ─────────────────────────────────────────────────────────
    if result.has_critical_or_high:
        summary = ", ".join(
            f"{f.vuln_id}({f.severity}) in {f.package}@{f.version}"
            for f in result.critical_or_high[:5]
        )
        if len(result.critical_or_high) > 5:
            summary += f" … and {len(result.critical_or_high) - 5} more"

        msg = (
            f"Skill '{skill_name}' has {len(result.critical_or_high)} "
            f"critical/high CVE(s) [{scanner_name}]: {summary}"
        )

        if mode == "block":
            logger.error("Blocking skill load — %s", msg)
            raise SkillSecurityError(msg)

        # warn mode — continue loading, but make noise
        logger.warning("Security warning — %s", msg)

    return result
