package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/ws"
	"github.com/gin-gonic/gin"
)

// socket.go is WebSocket-heavy and requires a real WS handshake for full testing.
// These tests cover the CORS origin checker and request validation logic
// that can be exercised without a full WebSocket upgrade.

// ---------- upgrader.CheckOrigin: dev mode (no CORS_ORIGINS) → allow all ----------

func TestSocketUpgrader_DevModeAllowsAll(t *testing.T) {
	// Ensure no CORS_ORIGINS is set
	os.Unsetenv("CORS_ORIGINS")

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://evil.example.com")

	if !upgrader.CheckOrigin(req) {
		t.Error("expected CheckOrigin to return true in dev mode (no CORS_ORIGINS)")
	}
}

// ---------- upgrader.CheckOrigin: production mode → allowed origin ----------

func TestSocketUpgrader_AllowedOrigin(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "http://localhost:3000,https://app.molecule.ai")

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://app.molecule.ai")

	if !upgrader.CheckOrigin(req) {
		t.Error("expected CheckOrigin to return true for allowed origin")
	}
}

// ---------- upgrader.CheckOrigin: production mode → blocked origin ----------

func TestSocketUpgrader_BlockedOrigin(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "http://localhost:3000,https://app.molecule.ai")

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://evil.example.com")

	if upgrader.CheckOrigin(req) {
		t.Error("expected CheckOrigin to return false for blocked origin")
	}
}

// ---------- HandleConnect: non-WebSocket request → error (no upgrade) ----------

func TestSocketHandleConnect_NonWSRequest(t *testing.T) {
	setupTestDB(t) // HandleConnect calls wsauth which needs db.DB

	hub := ws.NewHub(func(callerID, targetID string) bool { return true })
	h := NewSocketHandler(hub)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/ws", nil)

	// Without proper WebSocket upgrade headers, the upgrader should fail.
	// The handler doesn't return an HTTP error in this case — the gorilla
	// upgrader writes the error directly. We just verify it doesn't panic.
	h.HandleConnect(c)

	// The response should not be 200 OK (successful upgrade) since this is a plain HTTP request
	if w.Code == http.StatusSwitchingProtocols {
		t.Error("did not expect successful WebSocket upgrade for plain HTTP request")
	}
}

// ---------- NewSocketHandler: constructs correctly ----------

func TestNewSocketHandler(t *testing.T) {
	hub := ws.NewHub(func(callerID, targetID string) bool { return true })
	h := NewSocketHandler(hub)

	if h == nil {
		t.Fatal("expected non-nil SocketHandler")
	}
	if h.hub == nil {
		t.Fatal("expected non-nil hub")
	}
}
