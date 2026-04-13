package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRootCmdIncludesDoctor(t *testing.T) {
	root := buildRootCmd()
	cmd, _, err := root.Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("expected doctor command to be registered: %v", err)
	}
	if cmd == nil || cmd.Name() != "doctor" {
		t.Fatalf("expected to find doctor command, got %#v", cmd)
	}
}

func TestFindConfigsDirPrefersDirectoryWithTemplateConfig(t *testing.T) {
	tmp := t.TempDir()

	stale := filepath.Join(tmp, "empty-templates")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatalf("mkdir stale dir: %v", err)
	}

	valid := filepath.Join(tmp, "workspace-configs-templates")
	if err := os.MkdirAll(filepath.Join(valid, "claude-code-default"), 0o755); err != nil {
		t.Fatalf("mkdir valid dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(valid, "claude-code-default", "config.yaml"), []byte("name: Claude\n"), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	got := findDoctorConfigsDir([]string{stale, valid})
	if got != valid {
		t.Fatalf("findDoctorConfigsDir() = %q, want %q", got, valid)
	}
}

func TestDoctorSummaryHasFailures(t *testing.T) {
	results := []DoctorResult{
		{Name: "Platform", Status: DoctorStatusPass},
		{Name: "Postgres", Status: DoctorStatusFail},
		{Name: "Redis", Status: DoctorStatusWarn},
	}

	summary := summarizeDoctorResults(results)
	if !summary.HasFailures {
		t.Fatal("expected failures to be reported")
	}
	if summary.FailCount != 1 {
		t.Fatalf("FailCount = %d, want 1", summary.FailCount)
	}
	if summary.WarnCount != 1 {
		t.Fatalf("WarnCount = %d, want 1", summary.WarnCount)
	}
	if summary.PassCount != 1 {
		t.Fatalf("PassCount = %d, want 1", summary.PassCount)
	}
}

func TestFindDoctorMigrationsDirPrefersExistingDirectory(t *testing.T) {
	tmp := t.TempDir()

	missing := filepath.Join(tmp, "missing")
	valid := filepath.Join(tmp, "platform", "migrations")
	if err := os.MkdirAll(valid, 0o755); err != nil {
		t.Fatalf("mkdir migrations dir: %v", err)
	}

	got := findDoctorMigrationsDir([]string{missing, valid})
	if got != valid {
		t.Fatalf("findDoctorMigrationsDir() = %q, want %q", got, valid)
	}
}

func TestDoctorNextStepGuidance(t *testing.T) {
	t.Run("has failures", func(t *testing.T) {
		report := DoctorReport{
			Summary: DoctorSummary{HasFailures: true},
		}
		got := doctorNextStep(report)
		if !strings.Contains(got, "Fix FAIL items first") {
			t.Fatalf("doctorNextStep() = %q, want failure guidance", got)
		}
	})

	t.Run("only warnings", func(t *testing.T) {
		report := DoctorReport{
			Summary: DoctorSummary{WarnCount: 1},
		}
		got := doctorNextStep(report)
		if !strings.Contains(got, "warnings before provisioning") {
			t.Fatalf("doctorNextStep() = %q, want warning guidance", got)
		}
	})

	t.Run("all clear", func(t *testing.T) {
		report := DoctorReport{
			Summary: DoctorSummary{PassCount: 6},
		}
		got := doctorNextStep(report)
		if !strings.Contains(got, "start the platform") {
			t.Fatalf("doctorNextStep() = %q, want success guidance", got)
		}
	})
}
