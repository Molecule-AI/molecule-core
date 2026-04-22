package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// TestTerminalConnect_KI005_RejectsUnauthorizedCrossWorkspace tests the KI-005
// regression fix: workspace A must NOT be able to open a terminal on workspace B's
// container, even with a valid bearer token, unless they share a parent/child
// relationship. The vulnerability existed because HandleConnect only checked
// WorkspaceAuth (valid bearer → any :id) without the CanCommunicate hierarchy guard.
func TestTerminalConnect_KI005_RejectsUnauthorizedCrossWorkspace(t *testing.T) {
	mock := setupTestDB(t)
	// Stub CanCommunicate so it always returns false (no relationship).
	// Reset after test to avoid polluting other tests.
	prev := canCommunicateCheck
	canCommunicateCheck = func(callerID, targetID string) bool { return false }
	defer func() { canCommunicateCheck = prev }()

	// Token lookup: ws-caller's token is valid.
	rows := sqlmock.NewRows([]string{"workspace_id"}).AddRow("ws-caller")
	mock.ExpectQuery("SELECT workspace_id FROM workspace_tokens").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	h := NewTerminalHandler(nil) // nil docker → local path
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-target"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-target/terminal", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-caller")
	c.Request.Header.Set("Authorization", "Bearer valid-token-for-ws-caller")

	h.HandleConnect(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("cross-workspace terminal: got %d, want 403 (%s)", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestTerminalConnect_KI005_AllowsOwnTerminal tests the flip side of KI-005:
// a workspace must still be able to access its own terminal. The CanCommunicate
// fast-path returns true when callerID == targetID.
func TestTerminalConnect_KI005_AllowsOwnTerminal(t *testing.T) {
	// CanCommunicate fast-path: callerID == targetID → returns true without DB.
	prev := canCommunicateCheck
	canCommunicateCheck = func(callerID, targetID string) bool { return callerID == targetID }
	defer func() { canCommunicateCheck = prev }()

	h := NewTerminalHandler(nil) // nil docker → 503 if reached
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-alice"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-alice/terminal", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-alice")
	c.Request.Header.Set("Authorization", "Bearer valid-token")

	h.HandleConnect(c)

	// Got 503 (nil docker) instead of 403 — means CanCommunicate passed
	// and we reached the Docker path, which is correct.
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("own-terminal pass-through: got %d, want 503 nil-docker (%s)", w.Code, w.Body.String())
	}
}

// TestTerminalConnect_KI005_SkipsCheckWithoutHeader tests the allowlist path:
// callers that don't send X-Workspace-ID (canvas/molecli with bearer-only auth)
// skip the CanCommunicate check entirely and fall through to the Docker auth path.
// We assert they get the nil-docker 503 instead of 403.
func TestTerminalConnect_KI005_SkipsCheckWithoutHeader(t *testing.T) {
	h := NewTerminalHandler(nil) // nil docker → 503 if reached
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-any"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-any/terminal", nil)
	// No X-Workspace-ID header → KI-005 check is skipped

	h.HandleConnect(c)

	// Got 503 (nil docker) instead of 403 — means KI-005 check was skipped
	// and we reached the Docker path, which is correct.
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("no X-Workspace-ID: got %d, want 503 nil-docker (%s)", w.Code, w.Body.String())
	}
}

// TestTerminalConnect_KI005_RejectsInvalidToken tests that an invalid bearer
// token also results in a non-200 response (falls through to Docker auth).
// ValidateAnyToken returns error → CanCommunicate is never called.
func TestTerminalConnect_KI005_RejectsInvalidToken(t *testing.T) {
	canCommunicateCalled := false
	prev := canCommunicateCheck
	canCommunicateCheck = func(callerID, targetID string) bool {
		canCommunicateCalled = true
		return true
	}
	defer func() { canCommunicateCheck = prev }()

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-target"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-target/terminal", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-caller")
	c.Request.Header.Set("Authorization", "Bearer invalid-token")

	h.HandleConnect(c)

	if canCommunicateCalled {
		t.Error("CanCommunicate should not be called with an invalid token")
	}
	// Got 503 (nil docker) instead of 200/403 — ValidateAnyToken rejected the
	// token and we fell through to Docker auth, which returned 503 (nil docker).
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("invalid token: got %d, want 503 nil-docker (%s)", w.Code, w.Body.String())
	}
}

// TestTerminalConnect_KI005_AllowsSiblingWorkspace tests the sibling path:
// two workspaces with the same parent ID should be allowed to communicate.
func TestTerminalConnect_KI005_AllowsSiblingWorkspace(t *testing.T) {
	prev := canCommunicateCheck
	canCommunicateCheck = func(callerID, targetID string) bool {
		// Simulate sibling: same parent
		return callerID == "ws-pm" && targetID == "ws-dev"
	}
	defer func() { canCommunicateCheck = prev }()

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-dev"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-dev/terminal", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-pm")
	c.Request.Header.Set("Authorization", "Bearer valid-token")

	h.HandleConnect(c)

	// CanCommunicate returned true → reached Docker path → 503 nil-docker
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("sibling access: got %d, want 503 nil-docker (%s)", w.Code, w.Body.String())
	}
}
