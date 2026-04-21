package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ---------- Install: invalid JSON body → 400 ----------

func TestPluginInstall_InvalidJSON(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-1/plugins", strings.NewReader(`not json`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	h.Install(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if body["error"] != "invalid request body" {
		t.Errorf("expected 'invalid request body', got %q", body["error"])
	}
}

// ---------- Uninstall: empty plugin name → 400 ----------

func TestPluginUninstall_EmptyName(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-1/plugins/", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-1"},
		{Key: "name", Value: ""},
	}

	h.Uninstall(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- Uninstall: traversal in plugin name → 400 ----------

func TestPluginUninstall_TraversalName(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-1/plugins/../../../etc", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-1"},
		{Key: "name", Value: "../../../etc"},
	}

	h.Uninstall(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- Uninstall: valid name but no Docker → 503 ----------

func TestPluginUninstall_NoDocker(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-1/plugins/my-plugin", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-1"},
		{Key: "name", Value: "my-plugin"},
	}

	h.Uninstall(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- Download: invalid plugin name → 400 ----------

func TestPluginDownload_InvalidName(t *testing.T) {
	setupTestDB(t) // Download checks wsauth which needs db.DB
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/plugins/../bad/download", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-1"},
		{Key: "name", Value: "../bad"},
	}

	h.Download(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- Install: empty body → 400 ----------

func TestPluginInstall_EmptyBody(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-1/plugins", strings.NewReader(``))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	h.Install(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
