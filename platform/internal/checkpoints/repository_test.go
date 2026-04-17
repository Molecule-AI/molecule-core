package checkpoints_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/checkpoints"
)

// newMock is a test helper that creates a sqlmock DB + mock controller and
// returns a CheckpointRepository backed by it.
func newMock(t *testing.T) (checkpoints.CheckpointRepository, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })
	return checkpoints.NewRepository(mockDB), mock
}

// ─────────────────────────────────────────────────────────────────────────────
// UpsertCheckpoint
// ─────────────────────────────────────────────────────────────────────────────

// TestUpsertCheckpoint_Success verifies that a successful INSERT/UPSERT
// produces no error and executes exactly the expected parameterised query.
func TestUpsertCheckpoint_Success(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectExec("INSERT INTO workflow_checkpoints").
		WithArgs("ws-1", "wf-abc", "task_receive", 0, "null").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpsertCheckpoint(context.Background(), "ws-1", "wf-abc", "task_receive", 0, nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestUpsertCheckpoint_WithPayload verifies that a non-nil payload is sent to
// the DB as a stringified JSONB value (not as raw bytes / bytea).
func TestUpsertCheckpoint_WithPayload(t *testing.T) {
	repo, mock := newMock(t)

	payload := []byte(`{"result":"ok","tokens":42}`)
	mock.ExpectExec("INSERT INTO workflow_checkpoints").
		WithArgs("ws-2", "wf-xyz", "llm_call", 1, `{"result":"ok","tokens":42}`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpsertCheckpoint(context.Background(), "ws-2", "wf-xyz", "llm_call", 1, payload)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestUpsertCheckpoint_DBError verifies that a database error is propagated to
// the caller (not swallowed).
func TestUpsertCheckpoint_DBError(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectExec("INSERT INTO workflow_checkpoints").
		WithArgs("ws-1", "wf-abc", "task_receive", 0, "null").
		WillReturnError(errors.New("connection reset"))

	err := repo.UpsertCheckpoint(context.Background(), "ws-1", "wf-abc", "task_receive", 0, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GetLatestCheckpoint
// ─────────────────────────────────────────────────────────────────────────────

// TestGetLatestCheckpoint_Found verifies that a single-row result is correctly
// scanned into a *Checkpoint with all fields populated.
func TestGetLatestCheckpoint_Found(t *testing.T) {
	repo, mock := newMock(t)

	completedAt := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	cols := []string{"id", "workspace_id", "workflow_id", "step_name", "step_index", "completed_at", "payload"}
	mock.ExpectQuery("SELECT id, workspace_id, workflow_id, step_name, step_index").
		WithArgs("ws-1", "wf-abc").
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow("ckpt-001", "ws-1", "wf-abc", "llm_call", 1, completedAt, nil))

	cp, err := repo.GetLatestCheckpoint(context.Background(), "ws-1", "wf-abc")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if cp == nil {
		t.Fatal("expected non-nil checkpoint, got nil")
	}
	if cp.ID != "ckpt-001" {
		t.Errorf("ID: got %q, want %q", cp.ID, "ckpt-001")
	}
	if cp.WorkspaceID != "ws-1" {
		t.Errorf("WorkspaceID: got %q, want %q", cp.WorkspaceID, "ws-1")
	}
	if cp.StepName != "llm_call" {
		t.Errorf("StepName: got %q, want %q", cp.StepName, "llm_call")
	}
	if cp.StepIndex != 1 {
		t.Errorf("StepIndex: got %d, want 1", cp.StepIndex)
	}
	if !cp.CompletedAt.Equal(completedAt) {
		t.Errorf("CompletedAt: got %v, want %v", cp.CompletedAt, completedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetLatestCheckpoint_WithPayload verifies that a non-NULL payload column
// is returned as a json.RawMessage on the Checkpoint struct.
func TestGetLatestCheckpoint_WithPayload(t *testing.T) {
	repo, mock := newMock(t)

	completedAt := time.Now()
	rawPayload := []byte(`{"final_text":"done","success":true}`)
	cols := []string{"id", "workspace_id", "workflow_id", "step_name", "step_index", "completed_at", "payload"}
	mock.ExpectQuery("SELECT id, workspace_id, workflow_id, step_name, step_index").
		WithArgs("ws-1", "wf-abc").
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow("ckpt-002", "ws-1", "wf-abc", "task_complete", 2, completedAt, rawPayload))

	cp, err := repo.GetLatestCheckpoint(context.Background(), "ws-1", "wf-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cp == nil {
		t.Fatal("expected checkpoint, got nil")
	}
	if string(cp.Payload) != string(rawPayload) {
		t.Errorf("Payload: got %s, want %s", cp.Payload, rawPayload)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetLatestCheckpoint_NotFound verifies that (nil, nil) is returned when
// the workflow has no checkpoints — not an error condition.
func TestGetLatestCheckpoint_NotFound(t *testing.T) {
	repo, mock := newMock(t)

	cols := []string{"id", "workspace_id", "workflow_id", "step_name", "step_index", "completed_at", "payload"}
	mock.ExpectQuery("SELECT id, workspace_id, workflow_id, step_name, step_index").
		WithArgs("ws-1", "wf-missing").
		WillReturnRows(sqlmock.NewRows(cols)) // zero rows

	cp, err := repo.GetLatestCheckpoint(context.Background(), "ws-1", "wf-missing")
	if err != nil {
		t.Fatalf("expected nil error for not-found, got: %v", err)
	}
	if cp != nil {
		t.Errorf("expected nil checkpoint, got: %+v", cp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetLatestCheckpoint_QueryError verifies that a query-level DB error is
// propagated (not silenced) and nil is returned for the checkpoint.
func TestGetLatestCheckpoint_QueryError(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery("SELECT id, workspace_id, workflow_id, step_name, step_index").
		WithArgs("ws-1", "wf-err").
		WillReturnError(errors.New("db unavailable"))

	cp, err := repo.GetLatestCheckpoint(context.Background(), "ws-1", "wf-err")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if cp != nil {
		t.Errorf("expected nil checkpoint on error, got: %+v", cp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DeleteCheckpoints
// ─────────────────────────────────────────────────────────────────────────────

// TestDeleteCheckpoints_Success verifies that the DELETE query runs against the
// correct workspace and returns the number of affected rows.
func TestDeleteCheckpoints_Success(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectExec("DELETE FROM workflow_checkpoints").
		WithArgs("ws-1", "wf-abc").
		WillReturnResult(sqlmock.NewResult(0, 3))

	n, err := repo.DeleteCheckpoints(context.Background(), "ws-1", "wf-abc")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 deleted rows, got %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestDeleteCheckpoints_NoneExist verifies that (0, nil) is returned when the
// workflow has no checkpoints — deleting nothing is not an error.
func TestDeleteCheckpoints_NoneExist(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectExec("DELETE FROM workflow_checkpoints").
		WithArgs("ws-1", "wf-gone").
		WillReturnResult(sqlmock.NewResult(0, 0))

	n, err := repo.DeleteCheckpoints(context.Background(), "ws-1", "wf-gone")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 deleted rows, got %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestDeleteCheckpoints_DBError verifies that a database error is propagated.
func TestDeleteCheckpoints_DBError(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectExec("DELETE FROM workflow_checkpoints").
		WithArgs("ws-1", "wf-err").
		WillReturnError(errors.New("db timeout"))

	n, err := repo.DeleteCheckpoints(context.Background(), "ws-1", "wf-err")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if n != 0 {
		t.Errorf("expected 0 on error, got %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestGetLatestCheckpoint_RowsErr verifies that a rows.Err() set after
// iteration causes GetLatestCheckpoint to return an error rather than a
// partially-populated or empty result.
func TestGetLatestCheckpoint_RowsErr(t *testing.T) {
	repo, mock := newMock(t)

	completedAt := time.Now()
	cols := []string{"id", "workspace_id", "workflow_id", "step_name", "step_index", "completed_at", "payload"}
	// RowError(0, err): sqlmock fires the error when the driver tries to
	// advance past row 0, which causes rows.Next() to return false with
	// lasterr set.  rows.Err() then surfaces that error.
	mock.ExpectQuery("SELECT id, workspace_id, workflow_id, step_name, step_index").
		WithArgs("ws-1", "wf-roerr").
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow("ckpt-x", "ws-1", "wf-roerr", "task_receive", 0, completedAt, nil).
			RowError(0, errors.New("storage engine fault")))

	cp, err := repo.GetLatestCheckpoint(context.Background(), "ws-1", "wf-roerr")
	if err == nil {
		t.Fatal("expected rows.Err() to produce an error, got nil")
	}
	if cp != nil {
		t.Errorf("expected nil checkpoint on rows.Err(), got: %+v", cp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
