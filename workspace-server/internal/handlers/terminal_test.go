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
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestKI005_SelfAccess_AlwaysAllowed — when callerID equals the target workspace
// ID the request always passes (self-access: workspace's own token reaches its
// own terminal without needing the hierarchy check).
func TestKI005_SelfAccess_AlwaysAllowed(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("ws-self").
		WillReturnRows(sqlmock.NewRows([]string{"instance_id"}).AddRow(""))

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-self"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-self/terminal", nil)
	// Self-access: X-Workspace-ID matches the route param, no auth needed.
	c.Request.Header.Set("X-Workspace-ID", "ws-self")

	h.HandleConnect(c)

	// Self-access passes without any token check or CanCommunicate query.
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("self-access: expected 503 (Docker unavailable), got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestKI005_CanCommunicatePeer_Allowed — when the caller and target are siblings
// (share a parent), CanCommunicate returns true and the terminal access is granted.
func TestKI005_CanCommunicatePeer_Allowed(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	// DB: caller workspace row for token validation.
	mock.ExpectQuery("SELECT t.id, t.workspace_id").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}).
			AddRow("tok-caller", "ws-peer-a"))

	// DB: caller and target are siblings → CanCommunicate queries both.
	mock.ExpectQuery("SELECT id, parent_id FROM workspaces WHERE id").
		WithArgs("ws-peer-a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id"}).
			AddRow("ws-peer-a", "org-lead"))
	mock.ExpectQuery("SELECT id, parent_id FROM workspaces WHERE id").
		WithArgs("ws-peer-b").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id"}).
			AddRow("ws-peer-b", "org-lead"))

	// DB: target workspace has no instance_id → local Docker path.
	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("ws-peer-b").
		WillReturnRows(sqlmock.NewRows([]string{"instance_id"}).AddRow(""))

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-peer-b"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-peer-b/terminal", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-peer-a")
	c.Request.Header.Set("Authorization", "Bearer peer-token")

	h.HandleConnect(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("peer access: expected 503 (Docker unavailable), got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestKI005_CanCommunicateNonPeer_Forbidden — when caller and target have
// different parents (not siblings, not root-level), CanCommunicate returns
// false and the terminal access is blocked with 403.
func TestKI005_CanCommunicateNonPeer_Forbidden(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	// DB: caller workspace row for token validation.
	mock.ExpectQuery("SELECT t.id, t.workspace_id").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}).
			AddRow("tok-attacker", "ws-attacker"))

	// DB: caller and target have different parents → CanCommunicate denies.
	mock.ExpectQuery("SELECT id, parent_id FROM workspaces WHERE id").
		WithArgs("ws-attacker").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id"}).
			AddRow("ws-attacker", "org-a"))
	mock.ExpectQuery("SELECT id, parent_id FROM workspaces WHERE id").
		WithArgs("ws-victim").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id"}).
			AddRow("ws-victim", "org-b"))

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-victim"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-victim/terminal", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-attacker")
	c.Request.Header.Set("Authorization", "Bearer attacker-token")

	h.HandleConnect(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("cross-workspace: expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestKI005_TokenMismatch_Unauthorized — when the bearer token belongs to a
// different workspace than the claimed X-Workspace-ID, ValidateToken fails
// and the request is rejected with 401 before CanCommunicate is checked.
func TestKI005_TokenMismatch_Unauthorized(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	// DB: token belongs to a different workspace than claimed — ValidateToken
	// returns ErrInvalidToken (workspaceID mismatch).
	mock.ExpectQuery("SELECT t.id, t.workspace_id").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}))

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-target"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-target/terminal", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-claimed")
	c.Request.Header.Set("Authorization", "Bearer wrong-workspace-token")

	h.HandleConnect(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("token mismatch: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestKI005_NoXWorkspaceIDHeader_LegacyAllowed — when no X-Workspace-ID header
// is present (legacy canvas, direct browser access), the hierarchy check is
// skipped and the request proceeds to the container (standard WorkspaceAuth
// gates apply upstream).
func TestKI005_NoXWorkspaceIDHeader_LegacyAllowed(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	// DB: no instance_id → local Docker path.
	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("ws-legacy").
		WillReturnRows(sqlmock.NewRows([]string{"instance_id"}).AddRow(""))

	h := NewTerminalHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-legacy"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-legacy/terminal", nil)
	// No X-Workspace-ID header: legacy access, no hierarchy check.

	h.HandleConnect(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("legacy access: expected 503 (Docker unavailable), got %d: %s", w.Code, w.Body.String())
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
	}
}
