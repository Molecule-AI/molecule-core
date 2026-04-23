package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// testCanCommunicateCalls stub for tests — tracks last call.
var testCanCommunicateCalls [][2]string

func init() {
	gin.SetMode(gin.TestMode)
}

// stubCanCommunicate records calls and returns true (allows all) or false
// (denies all) based on the test scenario.
func stubCanCommunicate(callerID, targetID string) bool {
	testCanCommunicateCalls = append(testCanCommunicateCalls, [2]string{callerID, targetID})
	return len(testCanCommunicateCalls) > 0 && testCanCommunicateCalls[len(testCanCommunicateCalls)-1][0] != "ws-blocked"
}

// TestKI005_TerminalAuth_HierarchyGuard verifies that HandleConnect enforces
// CanCommunicate(callerID, workspaceID) before granting terminal access.
func TestKI005_TerminalAuth_HierarchyGuard(t *testing.T) {
	// Save and restore the real canCommunicateCheck.
	realCheck := canCommunicateCheck
	canCommunicateCheck = stubCanCommunicate
	testCanCommunicateCalls = nil
	defer func() { canCommunicateCheck = realCheck }()

	// No Docker client — we just check the auth rejection, not the connection.
	h := &TerminalHandler{docker: nil}

	tests := []struct {
		name           string
		callerID       string
		targetID       string
		authHeader     string
		wantStatus     int
		wantError      string
		canCommResult  bool // if set, override stub result
	}{
		{
			name:       "caller reaches own workspace — allowed",
			callerID:   "ws-1",
			targetID:   "ws-1",
			authHeader: "",
			wantStatus: http.StatusServiceUnavailable, // no docker, but auth passes
		},
		{
			name:       "different caller, CanCommunicate=true — allowed",
			callerID:   "ws-parent",
			targetID:   "ws-child",
			authHeader: "",
			wantStatus: http.StatusServiceUnavailable, // no docker, but auth passes
		},
		{
			name:       "caller explicitly blocked — forbidden",
			callerID:   "ws-blocked",
			targetID:   "ws-other",
			authHeader: "",
			wantStatus: http.StatusForbidden,
			wantError:  "not authorized to access this workspace's terminal",
		},
		{
			name:       "no X-Workspace-ID header — allowed (no identity claim)",
			callerID:   "",
			targetID:   "ws-1",
			authHeader: "",
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/workspaces/"+tt.targetID+"/terminal", nil)
			if tt.callerID != "" {
				c.Request.Header.Set("X-Workspace-ID", tt.callerID)
			}
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}

			// Re-stub to clear between subtests
			testCanCommunicateCalls = nil

			h.HandleConnect(c)

			if w.Code != tt.wantStatus {
				t.Errorf("HandleConnect: got %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantError != "" {
				body := w.Body.String()
				if !strContains(body, tt.wantError) {
					t.Errorf("HandleConnect: expected error containing %q, got %q", tt.wantError, body)
				}
			}
		})
	}
}

// TestKI005_TerminalAuth_NoHeaderNoCheck documents that when no X-Workspace-ID
// is provided, no hierarchy check is performed. This is intentional — canvas
// browser sessions without a workspace identity still need to reach terminals.
func TestKI005_TerminalAuth_NoHeaderNoCheck(t *testing.T) {
	realCheck := canCommunicateCheck
	canCommunicateCheck = func(callerID, targetID string) bool {
		t.Errorf("canCommunicateCheck called with no X-Workspace-ID header — should not happen")
		return false
	}
	defer func() { canCommunicateCheck = realCheck }()

	h := &TerminalHandler{docker: nil}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/terminal", nil)
	// No X-Workspace-ID header — no auth check

	h.HandleConnect(c)

	// Should proceed to Docker lookup (503 since no docker), not auth check
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Errorf("HandleConnect without X-Workspace-ID: expected auth to pass, got %d", w.Code)
	}
}

func strContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && strContainsHelper(s, substr))
}

func strContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
