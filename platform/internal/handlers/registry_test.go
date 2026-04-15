package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ==================== Register — input validation ====================

func TestRegister_BadJSON(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/registry/register", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_MissingRequiredFields(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing url and agent_card
	body := `{"id":"ws-123"}`
	c.Request = httptest.NewRequest("POST", "/registry/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	// DB insert fails
	mock.ExpectExec("INSERT INTO workspaces").
		WithArgs("ws-fail", "ws-fail", "http://127.0.0.1:8000", `{"name":"test"}`).
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"id":"ws-fail","url":"http://127.0.0.1:8000","agent_card":{"name":"test"}}`
	c.Request = httptest.NewRequest("POST", "/registry/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== Heartbeat — offline → online recovery ====================

func TestHeartbeatHandler_OfflineToOnline(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	// Expect prevTask SELECT
	mock.ExpectQuery("SELECT COALESCE\\(current_task").
		WithArgs("ws-offline").
		WillReturnRows(sqlmock.NewRows([]string{"current_task"}).AddRow(""))

	// Expect heartbeat UPDATE
	mock.ExpectExec("UPDATE workspaces SET").
		WithArgs("ws-offline", 0.0, "", 1, 5000, "").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect evaluateStatus SELECT — currently offline
	mock.ExpectQuery("SELECT status FROM workspaces WHERE id =").
		WithArgs("ws-offline").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("offline"))

	// Expect status transition back to online
	mock.ExpectExec("UPDATE workspaces SET status = 'online'").
		WithArgs("ws-offline").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect RecordAndBroadcast INSERT for WORKSPACE_ONLINE
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"workspace_id":"ws-offline","error_rate":0.0,"sample_error":"","active_tasks":1,"uptime_seconds":5000}`
	c.Request = httptest.NewRequest("POST", "/registry/heartbeat", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Heartbeat(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestHeartbeatHandler_BadJSON(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/registry/heartbeat", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Heartbeat(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHeartbeatHandler_MissingWorkspaceID(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"error_rate":0.1}`
	c.Request = httptest.NewRequest("POST", "/registry/heartbeat", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Heartbeat(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHeartbeatHandler_DBUpdateError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	// Expect prevTask SELECT
	mock.ExpectQuery("SELECT COALESCE\\(current_task").
		WithArgs("ws-dberr").
		WillReturnRows(sqlmock.NewRows([]string{"current_task"}).AddRow(""))

	// Heartbeat UPDATE fails
	mock.ExpectExec("UPDATE workspaces SET").
		WithArgs("ws-dberr", 0.1, "", 0, 100, "").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"workspace_id":"ws-dberr","error_rate":0.1,"sample_error":"","active_tasks":0,"uptime_seconds":100}`
	c.Request = httptest.NewRequest("POST", "/registry/heartbeat", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Heartbeat(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== Heartbeat — stable (no transition) ====================

func TestHeartbeatHandler_OnlineStaysOnline(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	// Expect prevTask SELECT
	mock.ExpectQuery("SELECT COALESCE\\(current_task").
		WithArgs("ws-stable").
		WillReturnRows(sqlmock.NewRows([]string{"current_task"}).AddRow(""))

	// Expect heartbeat UPDATE
	mock.ExpectExec("UPDATE workspaces SET").
		WithArgs("ws-stable", 0.2, "", 3, 4000, "").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// evaluateStatus: online with error_rate 0.2 — below 0.5 threshold, stays online
	mock.ExpectQuery("SELECT status FROM workspaces WHERE id =").
		WithArgs("ws-stable").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("online"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"workspace_id":"ws-stable","error_rate":0.2,"sample_error":"","active_tasks":3,"uptime_seconds":4000}`
	c.Request = httptest.NewRequest("POST", "/registry/heartbeat", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Heartbeat(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== UpdateCard ====================

func TestUpdateCard_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	// Expect UPDATE query
	mock.ExpectExec("UPDATE workspaces SET agent_card").
		WithArgs("ws-card", `{"name":"Updated Agent","skills":["coding"]}`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect RecordAndBroadcast INSERT for AGENT_CARD_UPDATED
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"workspace_id":"ws-card","agent_card":{"name":"Updated Agent","skills":["coding"]}}`
	c.Request = httptest.NewRequest("POST", "/registry/update-card", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateCard(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "updated" {
		t.Errorf("expected status 'updated', got %v", resp["status"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestUpdateCard_BadJSON(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/registry/update-card", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateCard(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCard_MissingFields(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing agent_card
	body := `{"workspace_id":"ws-card"}`
	c.Request = httptest.NewRequest("POST", "/registry/update-card", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateCard(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCard_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	mock.ExpectExec("UPDATE workspaces SET agent_card").
		WithArgs("ws-card-err", `{"name":"fail"}`).
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"workspace_id":"ws-card-err","agent_card":{"name":"fail"}}`
	c.Request = httptest.NewRequest("POST", "/registry/update-card", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateCard(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestRegister_GuardAgainstResurrectingRemovedRow verifies the #73 fix:
// the ON CONFLICT UPSERT must carry a `WHERE status IS DISTINCT FROM 'removed'`
// clause so that a late heartbeat from a workspace that was just deleted
// does not resurrect the row to 'online'.
//
// sqlmock matches on a substring of the rendered SQL — we assert the WHERE
// clause is present in the statement issued by Register().
func TestRegister_GuardAgainstResurrectingRemovedRow(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	// This regex-ish match requires the guard. If the handler ever drops
	// the clause the test fails because the emitted SQL won't match.
	mock.ExpectExec("ON CONFLICT.*WHERE workspaces.status IS DISTINCT FROM 'removed'").
		WithArgs("ws-resurrect", "ws-resurrect", "http://127.0.0.1:8000", `{"name":"x"}`).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected = correctly guarded
	mock.ExpectQuery("SELECT url FROM workspaces WHERE id").
		WithArgs("ws-resurrect").
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("http://127.0.0.1:54321"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/registry/register",
		bytes.NewBufferString(`{"id":"ws-resurrect","url":"http://127.0.0.1:8000","agent_card":{"name":"x"}}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("#73 guard not present in UPSERT SQL: %v", err)
	}
}

// TestHeartbeat_SkipsRemovedRows verifies #73: heartbeat UPDATE carries
// `AND status != 'removed'` so a late heartbeat from a torn-down container
// doesn't refresh last_heartbeat_at on a tombstoned workspace.
func TestHeartbeat_SkipsRemovedRows(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	// prevTask lookup
	mock.ExpectQuery("SELECT COALESCE\\(current_task").
		WithArgs("ws-zombie").
		WillReturnRows(sqlmock.NewRows([]string{"current_task"}).AddRow(""))

	// UPDATE must include `AND status != 'removed'`. 0 rows affected is fine —
	// this is the tombstoned case the fix protects against.
	mock.ExpectExec("UPDATE workspaces SET.*WHERE id = .* AND status != 'removed'").
		WithArgs("ws-zombie", 0.0, "", 0, int64(0), "").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// evaluateStatus SELECT
	mock.ExpectQuery("SELECT status FROM workspaces WHERE id").
		WithArgs("ws-zombie").
		WillReturnError(sql.ErrNoRows) // row effectively removed from view

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/registry/heartbeat",
		bytes.NewBufferString(`{"workspace_id":"ws-zombie","error_rate":0,"sample_error":"","active_tasks":0,"uptime_seconds":0,"current_task":""}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Heartbeat(c)

	if w.Code != http.StatusOK {
		t.Errorf("heartbeat handler must still return 200 even on tombstoned row, got %d", w.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("#73 guard not present in heartbeat UPDATE SQL: %v", err)
	}
}

// ------------------------------------------------------------
// validateAgentURL (C6 SSRF fix)
// ------------------------------------------------------------

func TestValidateAgentURL(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// ── Valid URLs (public / localhost by name) ───────────────────────────
		{"valid https public hostname", "https://agent.example.com:443", false},
		// 127.0.0.1 loopback — blocked by separate PR fix/c6-loopback-ssrf; keep
		// this case at false until that branch merges so both PRs can coexist.
		{"valid localhost http (loopback handled by sibling PR)", "http://127.0.0.1:8000", false},
		// 172.32.x is outside the 172.16.0.0/12 range (which ends at 172.31.255.255).
		{"valid 172.32.x outside RFC-1918 /12 boundary", "http://172.32.0.1:8080", false},

		// ── Bad input / wrong scheme ──────────────────────────────────────────
		{"empty url", "", true},
		{"non-http scheme file", "file:///etc/passwd", true},
		{"non-http scheme ftp", "ftp://internal-server/secrets", true},
		{"malformed url", "://not-a-url", true},

		// ── 169.254.0.0/16 — link-local / cloud metadata ─────────────────────
		{"blocked link-local AWS IMDS", "http://169.254.169.254/latest/meta-data/", true},
		{"blocked link-local GCP metadata", "http://169.254.169.254/computeMetadata/v1/", true},
		{"blocked link-local 169.254.0.1", "http://169.254.0.1/anything", true},

		// ── 10.0.0.0/8 — RFC-1918 class A ────────────────────────────────────
		{"blocked RFC-1918 10.0.0.1 (range start)", "http://10.0.0.1:8080", true},
		{"blocked RFC-1918 10.0.0.5 (mid)", "http://10.0.0.5:8080", true},
		{"blocked RFC-1918 10.255.255.254 (range end)", "http://10.255.255.254:8080", true},

		// ── 172.16.0.0/12 — RFC-1918 class B (includes Docker bridge) ────────
		{"blocked RFC-1918 172.16.0.1 (range start)", "http://172.16.0.1:8080", true},
		{"blocked RFC-1918 172.18.0.5 (Docker bridge)", "http://172.18.0.5:8000", true},
		{"blocked RFC-1918 172.31.255.255 (range end)", "http://172.31.255.255:8080", true},

		// ── 192.168.0.0/16 — RFC-1918 class C ────────────────────────────────
		{"blocked RFC-1918 192.168.0.1 (range start)", "http://192.168.0.1:8080", true},
		{"blocked RFC-1918 192.168.1.100 (mid)", "http://192.168.1.100:8080", true},
		{"blocked RFC-1918 192.168.255.254 (range end)", "http://192.168.255.254:8080", true},

		// ── ::1/128 — IPv6 loopback ───────────────────────────────────────────
		{"blocked IPv6 loopback ::1", "http://[::1]:8080", true},

		// ── fc00::/7 — IPv6 unique-local ──────────────────────────────────────
		{"blocked IPv6 unique-local fc00::1", "http://[fc00::1]:8080", true},
		{"blocked IPv6 unique-local fd00::1 (common sub-range)", "http://[fd00::1]:8080", true},
		{"blocked IPv6 unique-local fdff::1 (range end)", "http://[fdff::1]:8080", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAgentURL(tc.url)
			if tc.wantErr && err == nil {
				t.Errorf("validateAgentURL(%q) = nil, want error", tc.url)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validateAgentURL(%q) = %v, want nil", tc.url, err)
			}
		})
	}
}

// ==================== C18 — Register ownership ====================

// TestRegister_C18_BootstrapAllowedNoTokens verifies that a workspace with NO
// live tokens (i.e. first-ever registration) is allowed through without a bearer
// token. This is the bootstrap path — the token is issued at the end of Register.
func TestRegister_C18_BootstrapAllowedNoTokens(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	// requireWorkspaceToken → HasAnyLiveToken → COUNT(*) returns 0 (no tokens).
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM workspace_auth_tokens").
		WithArgs("ws-new").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// Workspace upsert proceeds normally.
	mock.ExpectExec("INSERT INTO workspaces").
		WithArgs("ws-new", "ws-new", "http://127.0.0.1:9100", `{"name":"new-agent"}`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery("SELECT url FROM workspaces WHERE id").
		WithArgs("ws-new").
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow("http://127.0.0.1:9100"))

	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// HasAnyLiveToken check for token issuance at end of Register.
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM workspace_auth_tokens").
		WithArgs("ws-new").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// IssueToken INSERT.
	mock.ExpectExec("INSERT INTO workspace_auth_tokens").
		WithArgs("ws-new", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/registry/register",
		bytes.NewBufferString(`{"id":"ws-new","url":"http://127.0.0.1:9100","agent_card":{"name":"new-agent"}}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if w.Code != http.StatusOK {
		t.Errorf("C18 bootstrap: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Token should be present in response (first registration).
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["auth_token"] == nil {
		t.Errorf("C18 bootstrap: expected auth_token in first-registration response, got %v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("C18 bootstrap: unmet expectations: %v", err)
	}
}

// TestRegister_C18_HijackBlockedNoBearer verifies the C18 attack is blocked:
// when a workspace already has a live token, /register without a bearer → 401.
func TestRegister_C18_HijackBlockedNoBearer(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewRegistryHandler(broadcaster)

	// HasAnyLiveToken returns 1 — workspace already has an active token.
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM workspace_auth_tokens").
		WithArgs("ws-victim").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No Authorization header — simulates attacker with no credentials.
	c.Request = httptest.NewRequest("POST", "/registry/register",
		bytes.NewBufferString(`{"id":"ws-victim","url":"http://attacker.example.com:9999/steal","agent_card":{"name":"hijacked"}}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("C18 hijack: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	// The malicious URL must NOT have been persisted — no INSERT expectation was set.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("C18 hijack: unmet expectations: %v", err)
	}
}
