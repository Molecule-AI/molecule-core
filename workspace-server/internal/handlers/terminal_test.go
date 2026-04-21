package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

// terminal.go is WebSocket + Docker heavy. These tests cover the parts
// that can be exercised without a Docker daemon or real WebSocket upgrade.

// ---------- HandleConnect: nil Docker → 503 ----------

func TestTerminalHandleConnect_NilDocker(t *testing.T) {
	h := NewTerminalHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/terminal", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	h.HandleConnect(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if body["error"] != "Docker not available" {
		t.Errorf("expected 'Docker not available', got %q", body["error"])
	}
}

// ---------- NewTerminalHandler: constructs correctly ----------

func TestNewTerminalHandler(t *testing.T) {
	h := NewTerminalHandler(nil)
	if h == nil {
		t.Fatal("expected non-nil TerminalHandler")
	}
	if h.docker != nil {
		t.Error("expected nil docker client")
	}
}

// ---------- termUpgrader.CheckOrigin: localhost allowed ----------

func TestTermUpgrader_LocalhostAllowed(t *testing.T) {
	os.Unsetenv("CORS_ORIGINS")

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	if !termUpgrader.CheckOrigin(req) {
		t.Error("expected localhost origin to be allowed")
	}
}

// ---------- termUpgrader.CheckOrigin: empty origin allowed ----------

func TestTermUpgrader_EmptyOriginAllowed(t *testing.T) {
	os.Unsetenv("CORS_ORIGINS")

	req := httptest.NewRequest("GET", "/ws", nil)
	// No Origin header

	if !termUpgrader.CheckOrigin(req) {
		t.Error("expected empty origin to be allowed")
	}
}

// ---------- termUpgrader.CheckOrigin: non-localhost blocked without CORS_ORIGINS ----------

func TestTermUpgrader_NonLocalhostBlocked(t *testing.T) {
	os.Unsetenv("CORS_ORIGINS")

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://evil.example.com")

	if termUpgrader.CheckOrigin(req) {
		t.Error("expected non-localhost origin to be blocked when no CORS_ORIGINS set")
	}
}

// ---------- termUpgrader.CheckOrigin: CORS_ORIGINS allows listed origin ----------

func TestTermUpgrader_CORSOriginsAllowed(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "https://app.molecule.ai,https://staging.molecule.ai")

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://app.molecule.ai")

	if !termUpgrader.CheckOrigin(req) {
		t.Error("expected CORS_ORIGINS-listed origin to be allowed")
	}
}
