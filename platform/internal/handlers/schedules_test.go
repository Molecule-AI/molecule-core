package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// Issue #24 — DB is the source of truth; org/import is additive on
// template-source rows only. Runtime-added schedules survive re-imports.

// TestRuntimeSchedule_HasSourceRuntime asserts that POST /workspaces/:id/schedules
// writes source='runtime' so that re-imports of the org template never touch
// these user-created rows (preserved across re-imports).
func TestRuntimeSchedule_HasSourceRuntime(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewScheduleHandler()

	// Match the literal 'runtime' source baked into the INSERT and capture
	// the workspace id arg. The inserted row id is returned via RETURNING.
	mock.ExpectQuery("INSERT INTO workspace_schedules .* VALUES .* 'runtime'").
		WithArgs("550e8400-e29b-41d4-a716-446655440000", "test", "*/5 * * * *", "UTC", "do thing", true, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("11111111-1111-1111-1111-111111111111"))

	body := []byte(`{"name":"test","cron_expr":"*/5 * * * *","prompt":"do thing"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "550e8400-e29b-41d4-a716-446655440000"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/550e8400-e29b-41d4-a716-446655440000/schedules", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestImport_OrgScheduleSQLShape verifies the SQL emitted by the org/import
// path for schedules. It MUST be an INSERT ... ON CONFLICT (workspace_id, name)
// DO UPDATE ... WHERE source='template' with VALUES ... 'template'. Together
// these guarantee that re-import is:
//   - additive (new template rows are inserted),
//   - idempotent (existing template rows are refreshed),
//   - non-destructive of runtime rows (the WHERE filter skips them),
//   - never DELETE-based (additive only).
//
// This is a structural assertion against the source — cheap and catches a
// regression that would silently break user-created schedules across
// re-imports without needing a full provisioner harness.
func TestImport_OrgScheduleSQLShape(t *testing.T) {
	got := orgImportScheduleSQL(t)

	// Single test covers four CEO requirements at once: additive seed
	// (template marker), idempotent refresh (ON CONFLICT DO UPDATE),
	// runtime-row preservation (WHERE source='template'), and never-DELETE.
	mustContain := []string{
		"INSERT INTO workspace_schedules",
		"source",
		"'template'",
		"ON CONFLICT (workspace_id, name) DO UPDATE",
		"WHERE workspace_schedules.source = 'template'",
	}
	for _, s := range mustContain {
		if !strings.Contains(got, s) {
			t.Errorf("org/import schedule SQL missing fragment %q\n--- SQL ---\n%s", s, got)
		}
	}
	if regexp.MustCompile(`(?i)\bDELETE\b\s+FROM\s+workspace_schedules`).MatchString(got) {
		t.Error("org/import schedule SQL must never DELETE — additive only")
	}
}

// orgImportScheduleSQL pulls the relevant SQL fragment from org.go for shape
// assertions. Keeps this test independent of the provisioner.
func orgImportScheduleSQL(t *testing.T) string {
	t.Helper()
	// Test runs from package dir (platform/internal/handlers) so org.go is local.
	data, err := os.ReadFile("org.go")
	if err != nil {
		t.Fatalf("read org.go: %v", err)
	}
	src := string(data)
	idx := strings.Index(src, "INSERT INTO workspace_schedules")
	if idx < 0 {
		t.Fatalf("could not find schedules INSERT in org.go")
	}
	end := strings.Index(src[idx:], "`")
	if end < 0 {
		t.Fatalf("could not find closing backtick after schedules INSERT")
	}
	return src[idx : idx+end]
}

// TestList_IncludesSourceColumn asserts GET /workspaces/:id/schedules
// returns the source field so Canvas can render template/runtime badges.
func TestList_IncludesSourceColumn(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewScheduleHandler()

	cols := []string{
		"id", "workspace_id", "name", "cron_expr", "timezone", "prompt", "enabled",
		"last_run_at", "next_run_at", "run_count", "last_status", "last_error",
		"source", "created_at", "updated_at",
	}
	now := time.Now()
	mock.ExpectQuery("SELECT .* source, created_at, updated_at\\s+FROM workspace_schedules").
		WithArgs("550e8400-e29b-41d4-a716-446655440000").
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow("id1", "550e8400-e29b-41d4-a716-446655440000", "tmpl-sched", "0 * * * *", "UTC", "p", true,
				nil, nil, 0, "", "", "template", now, now).
			AddRow("id2", "550e8400-e29b-41d4-a716-446655440000", "user-sched", "*/5 * * * *", "UTC", "p2", true,
				nil, nil, 0, "", "", "runtime", now, now))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "550e8400-e29b-41d4-a716-446655440000"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/550e8400-e29b-41d4-a716-446655440000/schedules", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"source":"template"`) {
		t.Errorf(`response missing "source":"template": %s`, body)
	}
	if !strings.Contains(body, `"source":"runtime"`) {
		t.Errorf(`response missing "source":"runtime": %s`, body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
