package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ─── helpers ───────────────────────────────────────────────────────────────

func newAllowlistGET(orgID string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: orgID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/orgs/"+orgID+"/plugins/allowlist", nil)
	return w, c
}

func newAllowlistPUT(orgID string, body interface{}) (*httptest.ResponseRecorder, *gin.Context) {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: orgID}}
	c.Request = httptest.NewRequest(http.MethodPut, "/orgs/"+orgID+"/plugins/allowlist",
		bytes.NewReader(b))
	c.Request.Header.Set("Content-Type", "application/json")
	return w, c
}

// ─── GetAllowlist ──────────────────────────────────────────────────────────

func TestGetAllowlist_OrgNotFound(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-missing").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistGET("org-missing")
	h.GetAllowlist(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAllowlist_DBErrorOnOrgCheck(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-1").
		WillReturnError(sql.ErrConnDone)

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistGET("org-1")
	h.GetAllowlist(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAllowlist_Empty(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`SELECT plugin_name, enabled_by, enabled_at`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"plugin_name", "enabled_by", "enabled_at"}))

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistGET("org-1")
	h.GetAllowlist(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		OrgID    string           `json:"org_id"`
		Plugins  []allowlistEntry `json:"plugins"`
		AllowAll bool             `json:"allow_all"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if resp.OrgID != "org-1" {
		t.Errorf("expected org_id=org-1, got %q", resp.OrgID)
	}
	if len(resp.Plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(resp.Plugins))
	}
	if !resp.AllowAll {
		t.Error("expected allow_all=true for empty list")
	}
}

func TestGetAllowlist_WithEntries(t *testing.T) {
	mock := setupTestDB(t)
	ts := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`SELECT plugin_name, enabled_by, enabled_at`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"plugin_name", "enabled_by", "enabled_at"}).
			AddRow("browser-automation", "admin-ws", ts).
			AddRow("superpowers", "admin-ws", ts))

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistGET("org-1")
	h.GetAllowlist(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		OrgID    string           `json:"org_id"`
		Plugins  []allowlistEntry `json:"plugins"`
		AllowAll bool             `json:"allow_all"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if len(resp.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(resp.Plugins))
	}
	if resp.Plugins[0].PluginName != "browser-automation" {
		t.Errorf("expected first plugin=browser-automation, got %q", resp.Plugins[0].PluginName)
	}
	if resp.AllowAll {
		t.Error("expected allow_all=false when list is non-empty")
	}
}

func TestGetAllowlist_DBErrorOnQuery(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`SELECT plugin_name, enabled_by, enabled_at`).
		WithArgs("org-1").
		WillReturnError(sql.ErrConnDone)

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistGET("org-1")
	h.GetAllowlist(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── PutAllowlist ──────────────────────────────────────────────────────────

func TestPutAllowlist_MissingEnabledBy(t *testing.T) {
	setupTestDB(t)

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistPUT("org-1", map[string]interface{}{
		"plugins": []string{"my-plugin"},
		// enabled_by intentionally omitted
	})
	h.PutAllowlist(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPutAllowlist_InvalidPluginName(t *testing.T) {
	setupTestDB(t)

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistPUT("org-1", map[string]interface{}{
		"plugins":    []string{"../../evil"},
		"enabled_by": "admin-ws",
	})
	h.PutAllowlist(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid plugin name, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPutAllowlist_OrgNotFound(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-missing").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistPUT("org-missing", map[string]interface{}{
		"plugins":    []string{"my-plugin"},
		"enabled_by": "admin-ws",
	})
	h.PutAllowlist(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPutAllowlist_AddPlugins(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM org_plugin_allowlist`).
		WithArgs("org-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO org_plugin_allowlist`).
		WithArgs("org-1", "my-plugin", "admin-ws").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistPUT("org-1", map[string]interface{}{
		"plugins":    []string{"my-plugin"},
		"enabled_by": "admin-ws",
	})
	h.PutAllowlist(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		OrgID    string   `json:"org_id"`
		Plugins  []string `json:"plugins"`
		AllowAll bool     `json:"allow_all"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if len(resp.Plugins) != 1 || resp.Plugins[0] != "my-plugin" {
		t.Errorf("unexpected plugins: %v", resp.Plugins)
	}
	if resp.AllowAll {
		t.Error("expected allow_all=false for non-empty plugins list")
	}
}

func TestPutAllowlist_ClearAllowlist(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM org_plugin_allowlist`).
		WithArgs("org-1").
		WillReturnResult(sqlmock.NewResult(0, 3))
	// No INSERT expected — empty plugins slice.
	mock.ExpectCommit()

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistPUT("org-1", map[string]interface{}{
		"plugins":    []string{},
		"enabled_by": "admin-ws",
	})
	h.PutAllowlist(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		AllowAll bool `json:"allow_all"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if !resp.AllowAll {
		t.Error("expected allow_all=true after clearing all plugins")
	}
}

func TestPutAllowlist_MultiplePlugins(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM org_plugin_allowlist`).
		WithArgs("org-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO org_plugin_allowlist`).
		WithArgs("org-1", "browser-automation", "admin-ws").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO org_plugin_allowlist`).
		WithArgs("org-1", "superpowers", "admin-ws").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistPUT("org-1", map[string]interface{}{
		"plugins":    []string{"browser-automation", "superpowers"},
		"enabled_by": "admin-ws",
	})
	h.PutAllowlist(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPutAllowlist_InsertFails(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("org-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM org_plugin_allowlist`).
		WithArgs("org-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO org_plugin_allowlist`).
		WithArgs("org-1", "my-plugin", "admin-ws").
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	h := NewOrgPluginAllowlistHandler()
	w, c := newAllowlistPUT("org-1", map[string]interface{}{
		"plugins":    []string{"my-plugin"},
		"enabled_by": "admin-ws",
	})
	h.PutAllowlist(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on insert failure, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── resolveOrgID ──────────────────────────────────────────────────────────

func TestResolveOrgID_OrgRoot(t *testing.T) {
	mock := setupTestDB(t)

	// workspace has no parent → it IS the org root
	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-root").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	got, err := resolveOrgID(context.Background(), "ws-root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ws-root" {
		t.Errorf("expected ws-root, got %q", got)
	}
}

func TestResolveOrgID_WithParent(t *testing.T) {
	mock := setupTestDB(t)

	// workspace has a parent → parent is the org root
	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-child").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow("ws-parent"))

	got, err := resolveOrgID(context.Background(), "ws-child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ws-parent" {
		t.Errorf("expected ws-parent, got %q", got)
	}
}

func TestResolveOrgID_NotFound(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-ghost").
		WillReturnError(sql.ErrNoRows)

	got, err := resolveOrgID(context.Background(), "ws-ghost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for not-found workspace, got %q", got)
	}
}

// ─── checkOrgPluginAllowlist ───────────────────────────────────────────────

func TestCheckOrgPluginAllowlist_AllowAll_EmptyList(t *testing.T) {
	mock := setupTestDB(t)

	// resolveOrgID: no parent → ws-1 is org root
	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// plugin NOT in list
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("ws-1", "my-plugin").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// count = 0 → allow-all
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM org_plugin_allowlist`).
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	blocked, reason := checkOrgPluginAllowlist(context.Background(), "ws-1", "my-plugin")
	if blocked {
		t.Errorf("expected not blocked (allow-all), got blocked: %s", reason)
	}
}

func TestCheckOrgPluginAllowlist_Allowed_OnList(t *testing.T) {
	mock := setupTestDB(t)

	// resolveOrgID: no parent
	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// plugin IS in the allowlist
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("ws-1", "my-plugin").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	blocked, reason := checkOrgPluginAllowlist(context.Background(), "ws-1", "my-plugin")
	if blocked {
		t.Errorf("expected not blocked (on list), got blocked: %s", reason)
	}
}

func TestCheckOrgPluginAllowlist_Blocked_NotOnList(t *testing.T) {
	mock := setupTestDB(t)

	// resolveOrgID: no parent
	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// plugin NOT in the list
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("ws-1", "evil-plugin").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// count > 0 → allowlist is active
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM org_plugin_allowlist`).
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	blocked, reason := checkOrgPluginAllowlist(context.Background(), "ws-1", "evil-plugin")
	if !blocked {
		t.Error("expected plugin to be blocked (not on non-empty allowlist)")
	}
	if reason == "" {
		t.Error("expected non-empty reason when blocked")
	}
}

func TestCheckOrgPluginAllowlist_ChildWorkspace_UsesParentOrg(t *testing.T) {
	mock := setupTestDB(t)

	// resolveOrgID: ws-child has parent ws-parent
	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-child").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow("ws-parent"))

	// allowlist check uses parent org ID (ws-parent)
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("ws-parent", "my-plugin").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	blocked, reason := checkOrgPluginAllowlist(context.Background(), "ws-child", "my-plugin")
	if blocked {
		t.Errorf("expected not blocked (on parent's allowlist), got blocked: %s", reason)
	}
}

func TestCheckOrgPluginAllowlist_FailOpen_OnResolveError(t *testing.T) {
	mock := setupTestDB(t)

	// DB error during resolveOrgID → fail-open
	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-1").
		WillReturnError(sql.ErrConnDone)

	blocked, _ := checkOrgPluginAllowlist(context.Background(), "ws-1", "any-plugin")
	if blocked {
		t.Error("expected fail-open (not blocked) on DB error during resolveOrgID")
	}
}

func TestCheckOrgPluginAllowlist_FailOpen_OnExistsError(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// DB error on EXISTS check → fail-open
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("ws-1", "any-plugin").
		WillReturnError(sql.ErrConnDone)

	blocked, _ := checkOrgPluginAllowlist(context.Background(), "ws-1", "any-plugin")
	if blocked {
		t.Error("expected fail-open (not blocked) on DB error during EXISTS check")
	}
}

func TestCheckOrgPluginAllowlist_FailOpen_OnCountError(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT parent_id FROM workspaces WHERE id`).
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("ws-1", "any-plugin").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// DB error on COUNT check → fail-open
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM org_plugin_allowlist`).
		WithArgs("ws-1").
		WillReturnError(sql.ErrConnDone)

	blocked, _ := checkOrgPluginAllowlist(context.Background(), "ws-1", "any-plugin")
	if blocked {
		t.Error("expected fail-open (not blocked) on DB error during COUNT check")
	}
}
