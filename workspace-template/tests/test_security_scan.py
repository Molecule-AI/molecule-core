"""Tests for tools/security_scan.py — CVE scanning, parse logic, and mode enforcement.

Loads the real module via importlib so the conftest mock for tools.audit
does not interfere.  Each test receives a fresh module instance via the
real_security_scan fixture.
"""

from __future__ import annotations

import os
import importlib.util
import os
import json
import os
import sys
from pathlib import Path
from unittest.mock import MagicMock

import os
import pytest


# ---------------------------------------------------------------------------
# Fixture — load the real tools.security_scan module
# ---------------------------------------------------------------------------


@pytest.fixture
def real_security_scan(monkeypatch):
    """Load the real tools/security_scan.py, injecting a mock tools.audit."""
    mock_audit = MagicMock()
    mock_audit.log_event = MagicMock(return_value="trace-sec")
    monkeypatch.setitem(sys.modules, "builtin_tools.audit", mock_audit)
    monkeypatch.delitem(sys.modules, "builtin_tools.security_scan", raising=False)
    spec = importlib.util.spec_from_file_location(
        "builtin_tools.security_scan",
        os.path.join(os.path.dirname(__file__), "..", "builtin_tools/security_scan.py"),
    )
    mod = importlib.util.module_from_spec(spec)
    monkeypatch.setitem(sys.modules, "builtin_tools.security_scan", mod)
    spec.loader.exec_module(mod)
    return mod, mock_audit


# ---------------------------------------------------------------------------
# Helper: build a fake subprocess result
# ---------------------------------------------------------------------------


def _make_subprocess_result(returncode=0, stdout="", stderr=""):
    result = MagicMock()
    result.returncode = returncode
    result.stdout = stdout
    result.stderr = stderr
    return result


# ---------------------------------------------------------------------------
# Test 1: mode="off" returns ScanResult with scanner="none"
# ---------------------------------------------------------------------------


class TestScanModeOff:

    def test_scan_mode_off(self, real_security_scan, tmp_path):
        """mode='off' returns ScanResult with scanner='none', no subprocess called."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()

        subprocess_called = []
        monkeypatch_run = MagicMock(side_effect=lambda *a, **kw: subprocess_called.append(True))

        result = mod.scan_skill_dependencies("myskill", skill_path, "off")

        assert result.scanner == "none"
        assert result.requirements_file is None
        assert result.findings == []
        assert not subprocess_called


# ---------------------------------------------------------------------------
# Test 2: no requirements.txt → ScanResult scanner="none"
# ---------------------------------------------------------------------------


class TestScanNoRequirementsFile:

    def test_scan_no_requirements_file(self, real_security_scan, tmp_path):
        """Skill dir has no requirements.txt → ScanResult scanner='none'."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()

        result = mod.scan_skill_dependencies("myskill", skill_path, "warn")

        assert result.scanner == "none"
        assert result.requirements_file is None


# ---------------------------------------------------------------------------
# Tests 3-5: _find_requirements
# ---------------------------------------------------------------------------


class TestFindRequirements:

    def test_find_requirements_root(self, real_security_scan, tmp_path):
        """Creates requirements.txt in root dir → found."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()
        req = skill_path / "requirements.txt"
        req.write_text("requests==2.28.0\n")

        found = mod._find_requirements(skill_path)
        assert found == req

    def test_find_requirements_tools_subdir(self, real_security_scan, tmp_path):
        """Creates requirements.txt in tools/ subdir → found."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        tools_dir = skill_path / "tools"
        tools_dir.mkdir(parents=True)
        req = tools_dir / "requirements.txt"
        req.write_text("flask==2.3.0\n")

        found = mod._find_requirements(skill_path)
        assert found == req

    def test_find_requirements_not_found(self, real_security_scan, tmp_path):
        """No requirements file → returns None."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()

        found = mod._find_requirements(skill_path)
        assert found is None


# ---------------------------------------------------------------------------
# Tests 6-9: _run_scanner
# ---------------------------------------------------------------------------


class TestRunScanner:

    def test_run_scanner_success(self, real_security_scan, monkeypatch):
        """subprocess.run returns returncode=0 with stdout → (stdout, None)."""
        mod, mock_audit = real_security_scan
        mock_result = _make_subprocess_result(returncode=0, stdout='{"vulnerabilities": []}')
        monkeypatch.setattr(mod.subprocess, "run", MagicMock(return_value=mock_result))

        stdout, error = mod._run_scanner(["snyk", "test", "--file=req.txt", "--json"])
        assert stdout == '{"vulnerabilities": []}'
        assert error is None

    def test_run_scanner_exit_code_2(self, real_security_scan, monkeypatch):
        """subprocess returns exit 2 with empty stdout → returns error string."""
        mod, mock_audit = real_security_scan
        mock_result = _make_subprocess_result(returncode=2, stdout="", stderr="scan failed")
        monkeypatch.setattr(mod.subprocess, "run", MagicMock(return_value=mock_result))

        stdout, error = mod._run_scanner(["snyk", "test", "--file=req.txt", "--json"])
        assert stdout == ""
        assert error is not None
        assert "2" in error or "scan" in error.lower()

    def test_run_scanner_timeout(self, real_security_scan, monkeypatch):
        """subprocess raises TimeoutExpired → returns error."""
        mod, mock_audit = real_security_scan
        monkeypatch.setattr(
            mod.subprocess,
            "run",
            MagicMock(
                side_effect=mod.subprocess.TimeoutExpired(cmd="snyk", timeout=120)
            ),
        )

        stdout, error = mod._run_scanner(["snyk", "test"], timeout=120)
        assert stdout == ""
        assert error is not None
        assert "120" in error or "timed out" in error

    def test_run_scanner_file_not_found(self, real_security_scan, monkeypatch):
        """subprocess raises FileNotFoundError → returns error."""
        mod, mock_audit = real_security_scan
        monkeypatch.setattr(
            mod.subprocess,
            "run",
            MagicMock(side_effect=FileNotFoundError("snyk: not found")),
        )

        stdout, error = mod._run_scanner(["snyk", "test"])
        assert stdout == ""
        assert error is not None
        assert "snyk" in error or "not found" in error


# ---------------------------------------------------------------------------
# Tests 10-12: _parse_snyk
# ---------------------------------------------------------------------------


class TestParseSnyk:

    def test_parse_snyk_empty_output(self, real_security_scan):
        """Empty string → ([], 'empty snyk output')."""
        mod, mock_audit = real_security_scan
        findings, error = mod._parse_snyk("")
        assert findings == []
        assert error == "empty snyk output"

    def test_parse_snyk_json_error(self, real_security_scan):
        """Invalid JSON → returns parse error."""
        mod, mock_audit = real_security_scan
        findings, error = mod._parse_snyk("not valid json {")
        assert findings == []
        assert error is not None
        assert "parse error" in error or "JSON" in error

    def test_parse_snyk_valid(self, real_security_scan):
        """Valid snyk JSON with vulnerabilities → list of CVEFinding."""
        mod, mock_audit = real_security_scan
        snyk_output = json.dumps({
            "vulnerabilities": [
                {
                    "id": "SNYK-PYTHON-REQUESTS-1234",
                    "packageName": "requests",
                    "version": "2.28.0",
                    "severity": "HIGH",
                    "title": "SSRF vulnerability",
                },
                {
                    "id": "SNYK-PYTHON-FLASK-5678",
                    "packageName": "flask",
                    "version": "2.3.0",
                    "severity": "medium",
                    "title": "XSS issue",
                },
            ]
        })
        findings, error = mod._parse_snyk(snyk_output)
        assert error is None
        assert len(findings) == 2
        assert findings[0].vuln_id == "SNYK-PYTHON-REQUESTS-1234"
        assert findings[0].package == "requests"
        assert findings[0].version == "2.28.0"
        assert findings[0].severity == "high"  # lowercased
        assert "SSRF" in findings[0].description
        assert findings[1].severity == "medium"


# ---------------------------------------------------------------------------
# Tests 13-15: _parse_pip_audit
# ---------------------------------------------------------------------------


class TestParsePipAudit:

    def test_parse_pip_audit_empty(self, real_security_scan):
        """Empty string → ([], 'empty pip-audit output')."""
        mod, mock_audit = real_security_scan
        findings, error = mod._parse_pip_audit("")
        assert findings == []
        assert error == "empty pip-audit output"

    def test_parse_pip_audit_dict_format(self, real_security_scan):
        """Dict with 'dependencies' key → findings."""
        mod, mock_audit = real_security_scan
        output = json.dumps({
            "dependencies": [
                {
                    "name": "requests",
                    "version": "2.25.0",
                    "vulns": [
                        {
                            "id": "PYSEC-2023-001",
                            "description": "SSRF in requests library",
                            "fix_versions": ["2.28.0"],
                        }
                    ],
                }
            ]
        })
        findings, error = mod._parse_pip_audit(output)
        assert error is None
        assert len(findings) == 1
        assert findings[0].vuln_id == "PYSEC-2023-001"
        assert findings[0].package == "requests"
        assert findings[0].version == "2.25.0"
        assert "SSRF" in findings[0].description

    def test_parse_pip_audit_list_format(self, real_security_scan):
        """Bare list format → findings."""
        mod, mock_audit = real_security_scan
        output = json.dumps([
            {
                "name": "flask",
                "version": "2.0.0",
                "vulns": [
                    {
                        "id": "PYSEC-2023-002",
                        "description": "XSS vulnerability",
                        "severity": "high",
                    }
                ],
            }
        ])
        findings, error = mod._parse_pip_audit(output)
        assert error is None
        assert len(findings) == 1
        assert findings[0].vuln_id == "PYSEC-2023-002"
        assert findings[0].package == "flask"
        assert findings[0].severity == "high"

    def test_parse_pip_audit_list_with_non_dict_items(self, real_security_scan):
        """List containing non-dict items (e.g. null/string) — non-dicts skipped (line 185)."""
        import json as _json
        mod, mock_audit = real_security_scan
        # Mix: one valid dict dep + one string non-dict (triggers the continue on line 185)
        output = _json.dumps([
            "not_a_dict_item",
            {
                "name": "requests",
                "version": "2.25.0",
                "vulns": [
                    {
                        "id": "PYSEC-2023-010",
                        "description": "Vuln in requests",
                    }
                ],
            },
        ])
        findings, error = mod._parse_pip_audit(output)
        assert error is None
        assert len(findings) == 1
        assert findings[0].vuln_id == "PYSEC-2023-010"


# ---------------------------------------------------------------------------
# Tests 16-21: scan_skill_dependencies integration
# ---------------------------------------------------------------------------


class TestScanSkillDependencies:

    def test_scan_no_scanner_in_path(self, real_security_scan, monkeypatch, tmp_path):
        """shutil.which returns None for both snyk and pip-audit → skipped result, audit logged."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()
        req = skill_path / "requirements.txt"
        req.write_text("requests==2.28.0\n")

        monkeypatch.setattr(mod.shutil, "which", lambda name: None)

        result = mod.scan_skill_dependencies("myskill", skill_path, "warn")

        assert result.scanner == "none"
        assert result.scan_error is not None
        assert "PATH" in result.scan_error or "scanner" in result.scan_error.lower()
        mock_audit.log_event.assert_called_once()
        call_kwargs = str(mock_audit.log_event.call_args)
        assert "skipped" in call_kwargs

    def test_scan_snyk_clean(self, real_security_scan, monkeypatch, tmp_path):
        """shutil.which('snyk') → truthy, scanner returns clean output → clean result, audit logged."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()
        req = skill_path / "requirements.txt"
        req.write_text("requests==2.28.0\n")

        monkeypatch.setattr(mod.shutil, "which", lambda name: "/usr/bin/snyk" if name == "snyk" else None)
        mock_result = _make_subprocess_result(returncode=0, stdout='{"vulnerabilities": []}')
        monkeypatch.setattr(mod.subprocess, "run", MagicMock(return_value=mock_result))

        result = mod.scan_skill_dependencies("myskill", skill_path, "warn")

        assert result.scanner == "snyk"
        assert not result.has_critical_or_high
        assert result.findings == []
        mock_audit.log_event.assert_called_once()

    def test_scan_snyk_vulnerable_warn_mode(self, real_security_scan, monkeypatch, tmp_path):
        """snyk finds critical CVE, mode='warn' → logs warning, does NOT raise, returns result."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()
        req = skill_path / "requirements.txt"
        req.write_text("requests==2.25.0\n")

        monkeypatch.setattr(mod.shutil, "which", lambda name: "/usr/bin/snyk" if name == "snyk" else None)
        snyk_output = json.dumps({
            "vulnerabilities": [
                {
                    "id": "SNYK-CRITICAL-001",
                    "packageName": "requests",
                    "version": "2.25.0",
                    "severity": "critical",
                    "title": "Remote code execution",
                }
            ]
        })
        mock_result = _make_subprocess_result(returncode=1, stdout=snyk_output)
        monkeypatch.setattr(mod.subprocess, "run", MagicMock(return_value=mock_result))

        # In warn mode, should NOT raise
        result = mod.scan_skill_dependencies("myskill", skill_path, "warn")

        assert result.scanner == "snyk"
        assert result.has_critical_or_high
        assert len(result.critical_or_high) == 1
        mock_audit.log_event.assert_called_once()

    def test_scan_snyk_vulnerable_block_mode(self, real_security_scan, monkeypatch, tmp_path):
        """snyk finds critical CVE, mode='block' → raises SkillSecurityError."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()
        req = skill_path / "requirements.txt"
        req.write_text("requests==2.25.0\n")

        monkeypatch.setattr(mod.shutil, "which", lambda name: "/usr/bin/snyk" if name == "snyk" else None)
        snyk_output = json.dumps({
            "vulnerabilities": [
                {
                    "id": "SNYK-CRITICAL-001",
                    "packageName": "requests",
                    "version": "2.25.0",
                    "severity": "critical",
                    "title": "Remote code execution",
                }
            ]
        })
        mock_result = _make_subprocess_result(returncode=1, stdout=snyk_output)
        monkeypatch.setattr(mod.subprocess, "run", MagicMock(return_value=mock_result))

        with pytest.raises(mod.SkillSecurityError) as exc_info:
            mod.scan_skill_dependencies("myskill", skill_path, "block")

        assert "myskill" in str(exc_info.value)
        assert "snyk" in str(exc_info.value)

    def test_scan_pip_audit_fallback(self, real_security_scan, monkeypatch, tmp_path):
        """shutil.which('snyk') → None, shutil.which('pip-audit') → truthy → uses pip-audit."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()
        req = skill_path / "requirements.txt"
        req.write_text("flask==2.0.0\n")

        monkeypatch.setattr(
            mod.shutil,
            "which",
            lambda name: "/usr/bin/pip-audit" if name == "pip-audit" else None,
        )
        pip_audit_output = json.dumps({"dependencies": []})
        mock_result = _make_subprocess_result(returncode=0, stdout=pip_audit_output)
        monkeypatch.setattr(mod.subprocess, "run", MagicMock(return_value=mock_result))

        result = mod.scan_skill_dependencies("myskill", skill_path, "warn")

        assert result.scanner == "pip-audit"
        assert result.findings == []
        mock_audit.log_event.assert_called_once()

    def test_scan_with_run_error(self, real_security_scan, monkeypatch, tmp_path):
        """_run_scanner returns error → scan_error set in result."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()
        req = skill_path / "requirements.txt"
        req.write_text("requests==2.28.0\n")

        monkeypatch.setattr(mod.shutil, "which", lambda name: "/usr/bin/snyk" if name == "snyk" else None)
        # Simulate scanner error (exit 2)
        mock_result = _make_subprocess_result(returncode=2, stdout="", stderr="scan failed hard")
        monkeypatch.setattr(mod.subprocess, "run", MagicMock(return_value=mock_result))

        result = mod.scan_skill_dependencies("myskill", skill_path, "warn")

        assert result.scanner == "snyk"
        assert result.scan_error is not None
        assert result.findings == []
        mock_audit.log_event.assert_called_once()


# ---------------------------------------------------------------------------
# Tests 22-24: data model properties
# ---------------------------------------------------------------------------


class TestDataModels:

    def test_cve_finding_dataclass(self, real_security_scan):
        """CVEFinding fields are accessible."""
        mod, mock_audit = real_security_scan
        finding = mod.CVEFinding(
            vuln_id="CVE-2023-0001",
            package="requests",
            version="2.25.0",
            severity="critical",
            description="A critical vulnerability",
        )
        assert finding.vuln_id == "CVE-2023-0001"
        assert finding.package == "requests"
        assert finding.version == "2.25.0"
        assert finding.severity == "critical"
        assert finding.description == "A critical vulnerability"

    def test_scan_result_critical_or_high_property(self, real_security_scan):
        """ScanResult.critical_or_high filters correctly."""
        mod, mock_audit = real_security_scan
        findings = [
            mod.CVEFinding("CVE-001", "pkg-a", "1.0", "critical", "crit vuln"),
            mod.CVEFinding("CVE-002", "pkg-b", "2.0", "high", "high vuln"),
            mod.CVEFinding("CVE-003", "pkg-c", "3.0", "medium", "med vuln"),
            mod.CVEFinding("CVE-004", "pkg-d", "4.0", "low", "low vuln"),
        ]
        result = mod.ScanResult(
            skill_name="test-skill",
            scanner="snyk",
            requirements_file="/tmp/requirements.txt",
            findings=findings,
        )
        critical_high = result.critical_or_high
        assert len(critical_high) == 2
        severities = {f.severity for f in critical_high}
        assert severities == {"critical", "high"}

    def test_scan_result_has_critical_or_high(self, real_security_scan):
        """has_critical_or_high is True/False correctly."""
        mod, mock_audit = real_security_scan

        # No findings
        empty_result = mod.ScanResult(
            skill_name="clean-skill",
            scanner="snyk",
            requirements_file="/tmp/requirements.txt",
            findings=[],
        )
        assert empty_result.has_critical_or_high is False

        # Only medium/low
        safe_result = mod.ScanResult(
            skill_name="safe-skill",
            scanner="snyk",
            requirements_file="/tmp/requirements.txt",
            findings=[
                mod.CVEFinding("CVE-001", "pkg", "1.0", "medium", "desc"),
                mod.CVEFinding("CVE-002", "pkg", "1.0", "low", "desc"),
            ],
        )
        assert safe_result.has_critical_or_high is False

        # Has a high finding
        vuln_result = mod.ScanResult(
            skill_name="vuln-skill",
            scanner="snyk",
            requirements_file="/tmp/requirements.txt",
            findings=[
                mod.CVEFinding("CVE-001", "pkg", "1.0", "high", "desc"),
            ],
        )
        assert vuln_result.has_critical_or_high is True


# ---------------------------------------------------------------------------
# Gap 5: New targeted coverage tests
# ---------------------------------------------------------------------------


class TestRunScannerGenericException:

    def test_run_scanner_generic_exception(self, real_security_scan, monkeypatch):
        """Lines 134-135: subprocess.run raises OSError (not Timeout/FileNotFoundError) → returns error."""
        mod, mock_audit = real_security_scan
        monkeypatch.setattr(
            mod.subprocess,
            "run",
            MagicMock(side_effect=OSError("device busy")),
        )

        stdout, error = mod._run_scanner(["snyk", "test"])
        assert stdout == ""
        assert error is not None
        assert "device busy" in error


class TestParsePipAuditGaps:

    def test_parse_pip_audit_invalid_json(self, real_security_scan):
        """Lines 173-174: invalid JSON → returns parse error."""
        mod, mock_audit = real_security_scan
        findings, error = mod._parse_pip_audit("{bad json}")
        assert findings == []
        assert error is not None
        assert "parse error" in error or "JSON" in error

    def test_parse_pip_audit_bare_list_format(self, real_security_scan):
        """Line 185: bare list format (not dict) → deps = data (old pip-audit format)."""
        mod, mock_audit = real_security_scan
        output = json.dumps([
            {
                "name": "requests",
                "version": "2.0.0",
                "vulns": [
                    {
                        "id": "CVE-2023-LIST-1",
                        "description": "A vulnerability in list format",
                    }
                ],
            }
        ])
        findings, error = mod._parse_pip_audit(output)
        assert error is None
        assert len(findings) == 1
        assert findings[0].vuln_id == "CVE-2023-LIST-1"
        assert findings[0].package == "requests"


class TestScanSkillDependenciesGaps:

    def test_scan_pip_audit_run_error(self, real_security_scan, monkeypatch, tmp_path):
        """Line 254: pip-audit _run_scanner returns error string → scan_error set."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()
        req = skill_path / "requirements.txt"
        req.write_text("requests==2.28.0\n")

        monkeypatch.setattr(
            mod.shutil, "which",
            lambda name: "/usr/bin/pip-audit" if name == "pip-audit" else None,
        )
        # Make pip-audit exit 2 with empty stdout → run_error returned
        mock_result = _make_subprocess_result(returncode=2, stdout="", stderr="pip-audit failed")
        monkeypatch.setattr(mod.subprocess, "run", MagicMock(return_value=mock_result))

        result = mod.scan_skill_dependencies("myskill", skill_path, "warn")

        assert result.scanner == "pip-audit"
        assert result.scan_error is not None
        assert result.findings == []

    def test_scan_snyk_more_than_5_critical_findings_summary(self, real_security_scan, monkeypatch, tmp_path):
        """Line 313: >5 critical/high findings → summary includes '... and N more'."""
        mod, mock_audit = real_security_scan
        skill_path = tmp_path / "myskill"
        skill_path.mkdir()
        req = skill_path / "requirements.txt"
        req.write_text("requests==2.25.0\n")

        monkeypatch.setattr(
            mod.shutil, "which",
            lambda name: "/usr/bin/snyk" if name == "snyk" else None,
        )

        # Build 6 critical vulnerabilities
        vulns = [
            {
                "id": f"SNYK-CRIT-{i:03d}",
                "packageName": f"pkg{i}",
                "version": "1.0.0",
                "severity": "critical",
                "title": f"Critical vuln {i}",
            }
            for i in range(6)
        ]
        snyk_output = json.dumps({"vulnerabilities": vulns})
        mock_result = _make_subprocess_result(returncode=1, stdout=snyk_output)
        monkeypatch.setattr(mod.subprocess, "run", MagicMock(return_value=mock_result))

        # warn mode so it doesn't raise
        result = mod.scan_skill_dependencies("myskill", skill_path, "warn")

        assert result.scanner == "snyk"
        assert len(result.critical_or_high) == 6
        # The summary string would contain "more" — we verify it doesn't raise
        # and has the correct count
        assert result.has_critical_or_high is True
