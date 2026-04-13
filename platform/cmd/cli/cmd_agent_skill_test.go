package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const skillConfigFixture = `name: Demo
description: Demo agent
version: 1.0.0
tier: 1
model: anthropic:claude-sonnet-4-6

prompt_files:
  - system-prompt.md

skills:
  - brainstorming
  - memory-curation

tools: []
`

func TestParseSkillsFromConfig(t *testing.T) {
	skills, err := parseSkillsFromConfig(skillConfigFixture)
	if err != nil {
		t.Fatalf("parseSkillsFromConfig failed: %v", err)
	}
	want := []string{"brainstorming", "memory-curation"}
	if len(skills) != len(want) {
		t.Fatalf("expected %d skills, got %d", len(want), len(skills))
	}
	for i, s := range want {
		if skills[i] != s {
			t.Fatalf("skill %d = %q, want %q", i, skills[i], s)
		}
	}
}

func TestReplaceSkillsInConfig(t *testing.T) {
	updated, err := replaceSkillsInConfig(skillConfigFixture, []string{"brainstorming", "memory-curation", "skill-authoring"})
	if err != nil {
		t.Fatalf("replaceSkillsInConfig failed: %v", err)
	}
	if !strings.Contains(updated, "  - skill-authoring") {
		t.Fatalf("updated config missing new skill: %s", updated)
	}
}

func TestUpsertSkill(t *testing.T) {
	next, changed := upsertSkill([]string{"brainstorming"}, "skill-authoring", true)
	if !changed || len(next) != 2 {
		t.Fatalf("expected skill add to change list, got changed=%v next=%v", changed, next)
	}
	next, changed = upsertSkill(next, "brainstorming", true)
	if changed {
		t.Fatalf("expected duplicate add to be ignored, got next=%v", next)
	}
	next, changed = upsertSkill(next, "brainstorming", false)
	if !changed || len(next) != 1 || next[0] != "skill-authoring" {
		t.Fatalf("expected removal to leave one skill, got changed=%v next=%v", changed, next)
	}
}

func TestBuildAgentSkillCmdIncludesAudit(t *testing.T) {
	cmd := buildAgentSkillCmd()
	found := false
	for _, child := range cmd.Commands() {
		if child.Name() == "audit" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected skill command to include audit subcommand")
	}
}

func TestBuildAgentSkillCmdIncludesInstallAndPublish(t *testing.T) {
	cmd := buildAgentSkillCmd()
	want := map[string]bool{"install": false, "publish": false}
	for _, child := range cmd.Commands() {
		if _, ok := want[child.Name()]; ok {
			want[child.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("expected skill command to include %s subcommand", name)
		}
	}
}

func TestAuditWorkspaceSkills(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/workspaces/ws-1/files/config.yaml", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"path":    "config.yaml",
			"content": "name: Demo\nskills:\n  - good-skill\n  - missing-skill\n",
			"size":    52,
		})
	})
	mux.HandleFunc("/workspaces/ws-1/files/skills/good-skill/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"path": "skills/good-skill/SKILL.md",
			"content": "---\nname: Good Skill\ndescription: Works well\nversion: 1.0.0\n---\nUse this skill carefully.\n",
			"size": 92,
		})
	})
	mux.HandleFunc("/workspaces/ws-1/files/skills/missing-skill/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewPlatformClient(server.URL)
	results, err := auditWorkspaceSkills(client, "ws-1")
	if err != nil {
		t.Fatalf("auditWorkspaceSkills failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Skill != "good-skill" || results[0].Status != "PASS" {
		t.Fatalf("expected first skill to pass, got %+v", results[0])
	}
	if results[1].Skill != "missing-skill" || results[1].Status != "FAIL" {
		t.Fatalf("expected second skill to fail, got %+v", results[1])
	}
	if !strings.Contains(strings.Join(results[1].Issues, " "), "SKILL.md") {
		t.Fatalf("expected missing file issue, got %+v", results[1])
	}
}

func TestInstallWorkspaceSkillFromDir(t *testing.T) {
	mux := http.NewServeMux()
	var putPaths []string
	var putBodies = map[string]string{}

	mux.HandleFunc("/workspaces/ws-1/files/config.yaml", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"path":    "config.yaml",
				"content": "name: Demo\nskills:\n  - brainstorming\n",
				"size":    37,
			})
		case http.MethodPut:
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode put body: %v", err)
			}
			putPaths = append(putPaths, r.URL.Path)
			putBodies[r.URL.Path] = req["content"]
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	})
	mux.HandleFunc("/workspaces/ws-1/files/skills/authoring/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode put body: %v", err)
		}
		putPaths = append(putPaths, r.URL.Path)
		putBodies[r.URL.Path] = req["content"]
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/workspaces/ws-1/files/skills/authoring/tools/helper.py", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode put body: %v", err)
		}
		putPaths = append(putPaths, r.URL.Path)
		putBodies[r.URL.Path] = req["content"]
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	src := t.TempDir()
	skillDir := filepath.Join(src, "authoring")
	if err := os.MkdirAll(filepath.Join(skillDir, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: Authoring
description: Writes skills
version: 1.0.0
---
Do the thing.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "tools", "helper.py"), []byte(`print("hi")`), 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewPlatformClient(server.URL)
	result, err := installWorkspaceSkillFromDir(client, "ws-1", skillDir)
	if err != nil {
		t.Fatalf("installWorkspaceSkillFromDir failed: %v", err)
	}
	if result.Skill != "authoring" {
		t.Fatalf("unexpected skill: %+v", result)
	}
	if len(putPaths) != 3 {
		t.Fatalf("expected 3 PUTs, got %d (%v)", len(putPaths), putPaths)
	}
	if _, ok := putBodies["/workspaces/ws-1/files/skills/authoring/SKILL.md"]; !ok {
		t.Fatal("missing SKILL.md upload")
	}
	if !strings.Contains(putBodies["/workspaces/ws-1/files/config.yaml"], "authoring") {
		t.Fatalf("config.yaml missing installed skill: %s", putBodies["/workspaces/ws-1/files/config.yaml"])
	}
}

func TestPublishWorkspaceSkillToDir(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bundles/export/ws-1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema": "1.0",
			"id":     "ws-1",
			"skills": []map[string]any{
				{
					"id":          "authoring",
					"name":        "Authoring",
					"description": "Writes skills",
					"files": map[string]string{
						"SKILL.md":              "---\nname: Authoring\ndescription: Writes skills\nversion: 1.0.0\n---\nDo the thing.\n",
						"tools/helper.py":        `print("hi")`,
					},
				},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	dest := t.TempDir()
	client := NewPlatformClient(server.URL)
	result, err := publishWorkspaceSkillToDir(client, "ws-1", "authoring", dest)
	if err != nil {
		t.Fatalf("publishWorkspaceSkillToDir failed: %v", err)
	}
	if result.Destination != filepath.Join(dest, "authoring") {
		t.Fatalf("unexpected destination: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(dest, "authoring", "SKILL.md")); err != nil {
		t.Fatalf("expected SKILL.md to be written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "authoring", "tools", "helper.py")); err != nil {
		t.Fatalf("expected helper.py to be written: %v", err)
	}
}
