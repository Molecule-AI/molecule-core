package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// resolveYAMLIncludes is the preprocessor Phase 3 uses to split org.yaml
// into per-team / per-role files. These tests cover the happy path,
// nested includes, path traversal defense, cycle detection, depth cap,
// and the inline-template (no baseDir) error.

func TestResolveYAMLIncludes_FlatInclude(t *testing.T) {
	tmp := t.TempDir()
	// Write a team file with a single workspace.
	team := filepath.Join(tmp, "team.yaml")
	if err := os.WriteFile(team, []byte("name: Role A\nrole: Worker\ntier: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := []byte(`name: Test Org
workspaces:
  - !include team.yaml
`)
	out, err := resolveYAMLIncludes(src, tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Parse result and verify workspace name landed in place.
	var tmpl struct {
		Name       string         `yaml:"name"`
		Workspaces []OrgWorkspace `yaml:"workspaces"`
	}
	if err := yaml.Unmarshal(out, &tmpl); err != nil {
		t.Fatalf("re-parse failed: %v\n---\n%s", err, out)
	}
	if len(tmpl.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(tmpl.Workspaces))
	}
	if tmpl.Workspaces[0].Name != "Role A" {
		t.Errorf("workspace name: got %q, want %q", tmpl.Workspaces[0].Name, "Role A")
	}
}

func TestResolveYAMLIncludes_Nested(t *testing.T) {
	// team.yaml includes leaf.yaml. Prove nested resolution works + that
	// relative paths inside the included file resolve against THAT file's
	// dir, not the top-level org dir.
	tmp := t.TempDir()
	subDir := filepath.Join(tmp, "teams")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	leaf := filepath.Join(subDir, "leaf.yaml")
	if err := os.WriteFile(leaf, []byte("name: Leaf\ntier: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	team := filepath.Join(subDir, "team.yaml")
	if err := os.WriteFile(team, []byte("name: Parent\ntier: 3\nchildren:\n  - !include leaf.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := []byte(`name: Test
workspaces:
  - !include teams/team.yaml
`)
	out, err := resolveYAMLIncludes(src, tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var tmpl OrgTemplate
	if err := yaml.Unmarshal(out, &tmpl); err != nil {
		t.Fatalf("re-parse failed: %v\n---\n%s", err, out)
	}
	if len(tmpl.Workspaces) != 1 || tmpl.Workspaces[0].Name != "Parent" {
		t.Fatalf("workspaces[0]: %+v", tmpl.Workspaces)
	}
	if len(tmpl.Workspaces[0].Children) != 1 || tmpl.Workspaces[0].Children[0].Name != "Leaf" {
		t.Fatalf("children: %+v", tmpl.Workspaces[0].Children)
	}
}

func TestResolveYAMLIncludes_RejectsTraversal(t *testing.T) {
	tmp := t.TempDir()
	// Write a file outside tmp that the include would exfiltrate.
	outside := filepath.Join(filepath.Dir(tmp), "secret.yaml")
	if err := os.WriteFile(outside, []byte("name: Leak\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outside)

	cases := []string{"../secret.yaml", "../../secret.yaml"}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			src := []byte("workspaces:\n  - !include " + tc + "\n")
			_, err := resolveYAMLIncludes(src, tmp)
			if err == nil {
				t.Errorf("expected error for traversal %q", tc)
			}
		})
	}
}

func TestResolveYAMLIncludes_CycleDetected(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.yaml")
	b := filepath.Join(tmp, "b.yaml")
	if err := os.WriteFile(a, []byte("name: A\nchildren:\n  - !include b.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("name: B\nchildren:\n  - !include a.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := []byte("workspaces:\n  - !include a.yaml\n")
	_, err := resolveYAMLIncludes(src, tmp)
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if !strings.Contains(err.Error(), "cycle") && !strings.Contains(err.Error(), "depth") {
		t.Errorf("error should mention cycle or depth; got: %v", err)
	}
}

func TestResolveYAMLIncludes_EmptyPathErrors(t *testing.T) {
	tmp := t.TempDir()
	src := []byte("workspaces:\n  - !include \"\"\n")
	_, err := resolveYAMLIncludes(src, tmp)
	if err == nil {
		t.Error("expected error for empty !include path")
	}
}

func TestResolveYAMLIncludes_MissingFileErrors(t *testing.T) {
	tmp := t.TempDir()
	src := []byte("workspaces:\n  - !include nonexistent.yaml\n")
	_, err := resolveYAMLIncludes(src, tmp)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestResolveYAMLIncludes_InlineTemplateErrors(t *testing.T) {
	src := []byte("workspaces:\n  - !include team.yaml\n")
	_, err := resolveYAMLIncludes(src, "")
	if err == nil {
		t.Error("expected error when baseDir empty and !include used")
	}
}

func TestResolveYAMLIncludes_SiblingDirAccess(t *testing.T) {
	// Phase 4 pattern: a team file at `teams/<x>.yaml` refers to a role
	// file at `<role>/workspace.yaml` via `../<role>/workspace.yaml`.
	// The ref escapes the team file's own dir but stays inside the org
	// root — this must be allowed.
	tmp := t.TempDir()
	teamsDir := filepath.Join(tmp, "teams")
	roleDir := filepath.Join(tmp, "my-role")
	if err := os.MkdirAll(teamsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(roleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(roleDir, "workspace.yaml"), []byte("name: Cousin\ntier: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(teamsDir, "parent.yaml"), []byte("name: Parent\nchildren:\n  - !include ../my-role/workspace.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := []byte("workspaces:\n  - !include teams/parent.yaml\n")
	out, err := resolveYAMLIncludes(src, tmp)
	if err != nil {
		t.Fatalf("sibling-dir !include should work: %v", err)
	}
	var tmpl OrgTemplate
	if err := yaml.Unmarshal(out, &tmpl); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(tmpl.Workspaces) != 1 {
		t.Fatalf("workspaces: %d", len(tmpl.Workspaces))
	}
	kids := tmpl.Workspaces[0].Children
	if len(kids) != 1 || kids[0].Name != "Cousin" {
		t.Fatalf("children: %+v", kids)
	}
}

// Integration check: after Phase 3 split, the real molecule-dev/org.yaml
// resolves cleanly via !include and unmarshal into OrgTemplate produces
// the full workspace tree. Guards against split regressions landing on
// main before they can be caught by a deploy.
func TestResolveYAMLIncludes_RealMoleculeDev(t *testing.T) {
	// Locate the monorepo root from the test file location.
	// Test runs in platform/internal/handlers/; org template is at
	// ../../../org-templates/molecule-dev/org.yaml.
	here, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	orgDir := filepath.Clean(filepath.Join(here, "..", "..", "..", "org-templates", "molecule-dev"))
	orgFile := filepath.Join(orgDir, "org.yaml")
	data, err := os.ReadFile(orgFile)
	if err != nil {
		t.Skipf("molecule-dev/org.yaml not found (skipping integration test): %v", err)
	}
	expanded, err := resolveYAMLIncludes(data, orgDir)
	if err != nil {
		t.Fatalf("resolveYAMLIncludes on real org.yaml: %v", err)
	}
	var tmpl OrgTemplate
	if err := yaml.Unmarshal(expanded, &tmpl); err != nil {
		t.Fatalf("unmarshal expanded yaml: %v", err)
	}
	// Sanity: should have PM + Marketing Lead at top, and PM should have
	// at least Research Lead, Dev Lead, Documentation Specialist, Triage
	// Operator as children (the Phase 3 split targets).
	if len(tmpl.Workspaces) < 2 {
		t.Fatalf("expected ≥2 top-level workspaces, got %d", len(tmpl.Workspaces))
	}
	names := map[string]bool{}
	for _, w := range tmpl.Workspaces {
		names[w.Name] = true
	}
	for _, want := range []string{"PM", "Marketing Lead"} {
		if !names[want] {
			t.Errorf("expected top-level workspace %q, not found", want)
		}
	}
	var pm *OrgWorkspace
	for i := range tmpl.Workspaces {
		if tmpl.Workspaces[i].Name == "PM" {
			pm = &tmpl.Workspaces[i]
			break
		}
	}
	if pm == nil || len(pm.Children) < 4 {
		t.Errorf("PM should have ≥4 children after include resolution, got %d", len(pm.Children))
	}
}

func TestResolveYAMLIncludes_NoIncludesIsNoop(t *testing.T) {
	// Ensure the preprocessor is a safe no-op for templates that don't
	// use !include — critical since we always run it on POST /org/import.
	tmp := t.TempDir()
	src := []byte(`name: Simple
workspaces:
  - name: Only
    tier: 2
`)
	out, err := resolveYAMLIncludes(src, tmp)
	if err != nil {
		t.Fatalf("no-op should not error, got %v", err)
	}
	var orig, expanded OrgTemplate
	_ = yaml.Unmarshal(src, &orig)
	_ = yaml.Unmarshal(out, &expanded)
	if orig.Name != expanded.Name || len(orig.Workspaces) != len(expanded.Workspaces) {
		t.Errorf("no-op changed semantics; orig=%+v expanded=%+v", orig, expanded)
	}
}
