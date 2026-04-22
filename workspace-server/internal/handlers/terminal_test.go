package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// TestHandleConnect_RoutesToRemote asserts HandleConnect picks the CP path
// when the workspace row carries an instance_id. The WS upgrade fails in
// a unit test (plain HTTP request, no ws handshake), but that's after the
// DB lookup — so unmet sqlmock expectations is the routing assertion.
func TestHandleConnect_RoutesToRemote(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("ws-remote").
		WillReturnRows(sqlmock.NewRows([]string{"instance_id"}).AddRow("i-abc123"))

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-remote"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-remote/terminal", nil)

	h.HandleConnect(c)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations (router didn't hit CP branch): %v", err)
	}
}

// TestHandleConnect_RoutesToLocal asserts HandleConnect stays on the local
// Docker path when instance_id is empty.
func TestHandleConnect_RoutesToLocal(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	// DB: workspace row with NULL instance_id → COALESCE returns "".
	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("ws-local").
		WillReturnRows(sqlmock.NewRows([]string{"instance_id"}).AddRow(""))

	// nil docker client: local path errors early with 503 rather than
	// trying to inspect containers. Confirms we took the local branch.
	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-local"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-local/terminal", nil)

	h.HandleConnect(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("local branch should 503 when Docker is unavailable; got %d", w.Code)
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

	// Token lookup: ValidateToken queries workspace_auth_tokens for token hash,
	// JOINs with workspaces, and checks revoked_at + status. Return id + workspace_id.
	// ValidateToken then verifies workspace_id == callerID (ws-caller) → passes.
	rows := sqlmock.NewRows([]string{"id", "workspace_id"}).AddRow("token-123", "ws-caller")
	mock.ExpectQuery(`SELECT t\.id, t\.workspace_id.*FROM workspace_auth_tokens t.*JOIN workspaces`).
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

// TestOpenTunnelCmd_BuildsArgv guards against silent drift in the EIC
// tunnel invocation (e.g. someone flipping --local-port to --port).
func TestOpenTunnelCmd_BuildsArgv(t *testing.T) {
	cmd := openTunnelCmd(eicSSHOptions{
		InstanceID: "i-0abc",
		Region:     "us-east-2",
		LocalPort:  2222,
	})
	want := []string{
		"aws", "--region", "us-east-2",
		"ec2-instance-connect", "open-tunnel",
		"--instance-id", "i-0abc",
		"--local-port", "2222",
	}
	if len(cmd.Args) != len(want) {
		t.Fatalf("argv length: got %v want %v", cmd.Args, want)
	}
	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Errorf("argv[%d] = %q, want %q", i, cmd.Args[i], want[i])
		}
	}
}

// TestSSHCommandCmd_BuildsArgv guards against drift in the ssh-client
// invocation — specifically the user@host shape and the inline options
// that defeat host-key + known_hosts friction.
func TestSSHCommandCmd_BuildsArgv(t *testing.T) {
	cmd := sshCommandCmd(eicSSHOptions{
		OSUser:         "ubuntu",
		LocalPort:      2222,
		PrivateKeyPath: "/tmp/k",
	})
	want := []string{
		"ssh",
		"-i", "/tmp/k",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-p", "2222",
		"ubuntu@127.0.0.1",
	}
	if len(cmd.Args) != len(want) {
		t.Fatalf("argv length: got %v want %v", cmd.Args, want)
	}
	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Errorf("argv[%d] = %q, want %q", i, cmd.Args[i], want[i])
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
	mock := setupTestDB(t)
	canCommunicateCalled := false
	prev := canCommunicateCheck
	canCommunicateCheck = func(callerID, targetID string) bool {
		canCommunicateCalled = true
		return true
	}
	defer func() { canCommunicateCheck = prev }()

	// ValidateToken queries workspace_auth_tokens with the token hash.
	// No rows returned = ErrInvalidToken → 401.
	mock.ExpectQuery(`SELECT t\.id, t\.workspace_id.*FROM workspace_auth_tokens t.*JOIN workspaces`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}))

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
	// #1609 fix: ValidateToken rejects the invalid token → 401.
	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid token: got %d, want 401 (%s)", w.Code, w.Body.String())
	}
}

// TestTerminalConnect_KI005_AllowsSiblingWorkspace tests the sibling path:
// two workspaces with the same parent ID should be allowed to communicate.
func TestTerminalConnect_KI005_AllowsSiblingWorkspace(t *testing.T) {
	mock := setupTestDB(t)
	prev := canCommunicateCheck
	canCommunicateCheck = func(callerID, targetID string) bool {
		// Simulate sibling: same parent
		return callerID == "ws-pm" && targetID == "ws-dev"
	}
	defer func() { canCommunicateCheck = prev }()

	// ValidateToken must succeed for ws-pm's token.
	rows := sqlmock.NewRows([]string{"id", "workspace_id"}).AddRow("token-456", "ws-pm")
	mock.ExpectQuery(`SELECT t\.id, t\.workspace_id.*FROM workspace_auth_tokens t.*JOIN workspaces`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-dev"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-dev/terminal", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-pm")
	c.Request.Header.Set("Authorization", "Bearer valid-token")

	h.HandleConnect(c)

	// ValidateToken passed (ws-pm token) + CanCommunicate returned true → reached Docker path → 503 nil-docker
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("sibling access: got %d, want 503 nil-docker (%s)", w.Code, w.Body.String())
	}
}

// TestTerminalConnect_KI005_TokenMustMatchClaimedWorkspace is the regression test for
// GH#756 / #1609: a caller with a valid token for workspace A must NOT be able to
// forge X-Workspace-ID: B and access B's terminal. ValidateAnyToken accepted any
// valid org token; the fix uses ValidateToken(callerID, tok) which verifies the
// token belongs to the claimed X-Workspace-ID.
func TestTerminalConnect_KI005_TokenMustMatchClaimedWorkspace(t *testing.T) {
	mock := setupTestDB(t)
	prev := canCommunicateCheck
	canCommunicateCheck = func(callerID, targetID string) bool {
		return true // Simulate sibling/parent relationship
	}
	defer func() { canCommunicateCheck = prev }()

	// The attacker (ws-attacker) has a valid token, but claims X-Workspace-ID: ws-victim.
	// ValidateToken(ws-victim, attacker-token) looks up the token hash and finds it belongs
	// to ws-attacker (not ws-victim) → ErrInvalidToken → 401.
	mock.ExpectQuery(`SELECT t\.id, t\.workspace_id.*FROM workspace_auth_tokens t.*JOIN workspaces`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}).AddRow("token-attack", "ws-attacker"))

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-victim"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-victim/terminal", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-victim")
	c.Request.Header.Set("Authorization", "Bearer attacker-valid-token")

	h.HandleConnect(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("token-workspace mismatch: got %d, want 401 (%s)", w.Code, w.Body.String())
	}
}

