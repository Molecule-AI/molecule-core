package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// ---------- ListRegistry: runtime filter ----------

func TestPluginListRegistry_RuntimeFilter(t *testing.T) {
	dir := t.TempDir()

	// Plugin A: supports claude_code
	pluginA := filepath.Join(dir, "plugin-a")
	os.Mkdir(pluginA, 0755)
	os.WriteFile(filepath.Join(pluginA, "plugin.yaml"), []byte(`
name: plugin-a
runtimes: [claude_code]
`), 0644)

	// Plugin B: supports deepagents only
	pluginB := filepath.Join(dir, "plugin-b")
	os.Mkdir(pluginB, 0755)
	os.WriteFile(filepath.Join(pluginB, "plugin.yaml"), []byte(`
name: plugin-b
runtimes: [deepagents]
`), 0644)

	// Plugin C: no runtimes declared (should be included in any filter)
	pluginC := filepath.Join(dir, "plugin-c")
	os.Mkdir(pluginC, 0755)
	os.WriteFile(filepath.Join(pluginC, "plugin.yaml"), []byte(`
name: plugin-c
`), 0644)

	h := NewPluginsHandler(dir, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/plugins?runtime=claude_code", nil)

	h.ListRegistry(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var plugins []pluginInfo
	if err := json.Unmarshal(w.Body.Bytes(), &plugins); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	// Should include plugin-a (matches) and plugin-c (no runtimes = universal)
	// Should exclude plugin-b (deepagents only)
	if len(plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d: %+v", len(plugins), plugins)
	}
}

// ---------- ListAvailableForWorkspace: with runtime lookup ----------

func TestPluginListAvailableForWorkspace_WithRuntimeLookup(t *testing.T) {
	dir := t.TempDir()

	pluginA := filepath.Join(dir, "plugin-a")
	os.Mkdir(pluginA, 0755)
	os.WriteFile(filepath.Join(pluginA, "plugin.yaml"), []byte(`
name: plugin-a
runtimes: [langgraph]
`), 0644)

	h := NewPluginsHandler(dir, nil, nil).
		WithRuntimeLookup(func(wsID string) (string, error) {
			return "langgraph", nil
		})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/plugins/available", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	h.ListAvailableForWorkspace(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var plugins []pluginInfo
	json.Unmarshal(w.Body.Bytes(), &plugins)
	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(plugins))
	}
}

// ---------- ListInstalled: no Docker → empty list ----------

func TestPluginListInstalled_NoDocker(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/plugins", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	h.ListInstalled(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var plugins []pluginInfo
	json.Unmarshal(w.Body.Bytes(), &plugins)
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

// ---------- CheckRuntimeCompatibility: missing runtime param → 400 ----------

func TestPluginCheckRuntimeCompatibility_MissingRuntime(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/plugins/compatibility", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	h.CheckRuntimeCompatibility(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- CheckRuntimeCompatibility: no Docker → all compatible ----------

func TestPluginCheckRuntimeCompatibility_NoDocker(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/plugins/compatibility?runtime=claude_code", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	h.CheckRuntimeCompatibility(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["all_compatible"] != true {
		t.Errorf("expected all_compatible=true, got %v", body["all_compatible"])
	}
}
