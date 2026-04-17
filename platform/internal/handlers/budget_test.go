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

// ==================== GET /workspaces/:id/budget ====================

// TestBudgetGet_NotFound verifies that GET /budget returns 404 for an unknown
// workspace ID (ErrNoRows from the budget query).
func TestBudgetGet_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery(`SELECT budget_limit, COALESCE\(monthly_spend, 0\)`).
		WithArgs("ws-not-there").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-not-there"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-not-there/budget", nil)

	h := NewBudgetHandler()
	h.GetBudget(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestBudgetGet_DBError verifies that a non-ErrNoRows DB error returns 500.
func TestBudgetGet_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery(`SELECT budget_limit, COALESCE\(monthly_spend, 0\)`).
		WithArgs("ws-db-err").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-db-err"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-db-err/budget", nil)

	h := NewBudgetHandler()
	h.GetBudget(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestBudgetGet_NoLimit verifies that budget_limit and budget_remaining are
// null when the workspace has no budget ceiling configured.
func TestBudgetGet_NoLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery(`SELECT budget_limit, COALESCE\(monthly_spend, 0\)`).
		WithArgs("ws-free").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(nil, int64(42)))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-free"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-free/budget", nil)

	h := NewBudgetHandler()
	h.GetBudget(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["budget_limit"] != nil {
		t.Errorf("expected budget_limit=null, got %v", resp["budget_limit"])
	}
	if resp["budget_remaining"] != nil {
		t.Errorf("expected budget_remaining=null, got %v", resp["budget_remaining"])
	}
	if resp["monthly_spend"] != float64(42) {
		t.Errorf("expected monthly_spend=42, got %v", resp["monthly_spend"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestBudgetGet_WithLimit verifies that budget_limit, monthly_spend, and
// budget_remaining are all returned correctly when a ceiling is set.
func TestBudgetGet_WithLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery(`SELECT budget_limit, COALESCE\(monthly_spend, 0\)`).
		WithArgs("ws-capped").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(int64(500), int64(123)))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-capped"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-capped/budget", nil)

	h := NewBudgetHandler()
	h.GetBudget(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["budget_limit"] != float64(500) {
		t.Errorf("expected budget_limit=500, got %v", resp["budget_limit"])
	}
	if resp["monthly_spend"] != float64(123) {
		t.Errorf("expected monthly_spend=123, got %v", resp["monthly_spend"])
	}
	// budget_remaining = 500 - 123 = 377
	if resp["budget_remaining"] != float64(377) {
		t.Errorf("expected budget_remaining=377, got %v", resp["budget_remaining"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestBudgetGet_OverBudget verifies that budget_remaining can be negative
// when monthly_spend has already exceeded budget_limit.
func TestBudgetGet_OverBudget(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery(`SELECT budget_limit, COALESCE\(monthly_spend, 0\)`).
		WithArgs("ws-over").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(int64(100), int64(150)))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-over"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-over/budget", nil)

	h := NewBudgetHandler()
	h.GetBudget(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	// budget_remaining = 100 - 150 = -50 (negative, but we store actual value)
	if resp["budget_remaining"] != float64(-50) {
		t.Errorf("expected budget_remaining=-50, got %v", resp["budget_remaining"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// ==================== PATCH /workspaces/:id/budget ====================

// TestBudgetPatch_MissingField verifies that PATCH /budget with no budget_limit
// field in the body returns 400.
func TestBudgetPatch_MissingField(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-patch-missing"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-patch-missing/budget",
		bytes.NewBufferString(`{"other_field":123}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewBudgetHandler()
	h.PatchBudget(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestBudgetPatch_InvalidBody verifies that a malformed JSON body returns 400.
func TestBudgetPatch_InvalidBody(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-patch-bad"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-patch-bad/budget",
		bytes.NewBufferString(`not json`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewBudgetHandler()
	h.PatchBudget(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestBudgetPatch_NegativeValue verifies that a negative budget_limit is rejected.
func TestBudgetPatch_NegativeValue(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-negative"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-negative/budget",
		bytes.NewBufferString(`{"budget_limit":-1}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewBudgetHandler()
	h.PatchBudget(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative budget_limit, got %d: %s", w.Code, w.Body.String())
	}
}

// TestBudgetPatch_InvalidType verifies that a non-numeric budget_limit returns 400.
func TestBudgetPatch_InvalidType(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-badtype"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-badtype/budget",
		bytes.NewBufferString(`{"budget_limit":"not-a-number"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewBudgetHandler()
	h.PatchBudget(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for string budget_limit, got %d: %s", w.Code, w.Body.String())
	}
}

// TestBudgetPatch_WorkspaceNotFound verifies that PATCH /budget returns 404
// when the workspace doesn't exist.
func TestBudgetPatch_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery(`SELECT EXISTS.*status != 'removed'`).
		WithArgs("ws-no-exist").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-no-exist"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-no-exist/budget",
		bytes.NewBufferString(`{"budget_limit":500}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewBudgetHandler()
	h.PatchBudget(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestBudgetPatch_SetLimit verifies that PATCH /budget with a positive value
// updates the DB and returns the new budget state.
func TestBudgetPatch_SetLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	// Existence probe
	mock.ExpectQuery(`SELECT EXISTS.*status != 'removed'`).
		WithArgs("ws-set-limit").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// UPDATE
	mock.ExpectExec(`UPDATE workspaces SET budget_limit`).
		WithArgs("ws-set-limit", int64(500)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Re-read for response
	mock.ExpectQuery(`SELECT budget_limit, COALESCE\(monthly_spend, 0\) FROM workspaces WHERE id`).
		WithArgs("ws-set-limit").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(int64(500), int64(200)))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-set-limit"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-set-limit/budget",
		bytes.NewBufferString(`{"budget_limit":500}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewBudgetHandler()
	h.PatchBudget(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["budget_limit"] != float64(500) {
		t.Errorf("expected budget_limit=500, got %v", resp["budget_limit"])
	}
	if resp["monthly_spend"] != float64(200) {
		t.Errorf("expected monthly_spend=200, got %v", resp["monthly_spend"])
	}
	// budget_remaining = 500 - 200 = 300
	if resp["budget_remaining"] != float64(300) {
		t.Errorf("expected budget_remaining=300, got %v", resp["budget_remaining"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestBudgetPatch_ClearLimit verifies that PATCH /budget with budget_limit=null
// clears the ceiling, making budget_limit and budget_remaining null in the response.
func TestBudgetPatch_ClearLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery(`SELECT EXISTS.*status != 'removed'`).
		WithArgs("ws-clear-limit").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// UPDATE with NULL
	mock.ExpectExec(`UPDATE workspaces SET budget_limit`).
		WithArgs("ws-clear-limit", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Re-read — budget_limit is now NULL
	mock.ExpectQuery(`SELECT budget_limit, COALESCE\(monthly_spend, 0\) FROM workspaces WHERE id`).
		WithArgs("ws-clear-limit").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(nil, int64(50)))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-clear-limit"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-clear-limit/budget",
		bytes.NewBufferString(`{"budget_limit":null}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewBudgetHandler()
	h.PatchBudget(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["budget_limit"] != nil {
		t.Errorf("expected budget_limit=null after clear, got %v", resp["budget_limit"])
	}
	if resp["budget_remaining"] != nil {
		t.Errorf("expected budget_remaining=null after clear, got %v", resp["budget_remaining"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestBudgetPatch_UpdateDBError verifies that a DB error during the UPDATE
// returns 500.
func TestBudgetPatch_UpdateDBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery(`SELECT EXISTS.*status != 'removed'`).
		WithArgs("ws-patch-dberr").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`UPDATE workspaces SET budget_limit`).
		WithArgs("ws-patch-dberr", int64(500)).
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-patch-dberr"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-patch-dberr/budget",
		bytes.NewBufferString(`{"budget_limit":500}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewBudgetHandler()
	h.PatchBudget(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on UPDATE error, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestBudgetPatch_ZeroLimit verifies that budget_limit=0 is accepted (it means
// every A2A call is blocked — useful to pause a workspace's LLM spend entirely).
func TestBudgetPatch_ZeroLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	mock.ExpectQuery(`SELECT EXISTS.*status != 'removed'`).
		WithArgs("ws-zero-limit").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`UPDATE workspaces SET budget_limit`).
		WithArgs("ws-zero-limit", int64(0)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT budget_limit, COALESCE\(monthly_spend, 0\) FROM workspaces WHERE id`).
		WithArgs("ws-zero-limit").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(int64(0), int64(0)))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-zero-limit"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-zero-limit/budget",
		bytes.NewBufferString(`{"budget_limit":0}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewBudgetHandler()
	h.PatchBudget(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for zero budget_limit, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}
