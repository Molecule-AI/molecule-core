package registry

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// fakeEmitter records every RecordAndBroadcast call so tests can assert
// payload shape + emission count. Safe for concurrent use since the sweeper
// itself is single-goroutine but keeping the lock lets the suite fan out.
type fakeEmitter struct {
	mu     sync.Mutex
	events []emittedEvent
	fail   bool
}

type emittedEvent struct {
	Type        string
	WorkspaceID string
	Payload     interface{}
}

func (f *fakeEmitter) RecordAndBroadcast(_ context.Context, eventType string, workspaceID string, payload interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, emittedEvent{eventType, workspaceID, payload})
	if f.fail {
		return errors.New("broadcast boom")
	}
	return nil
}

func (f *fakeEmitter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

// TestSweepStuckProvisioning_FlipsOverdue verifies the happy path: a stuck
// provisioning workspace gets flipped to failed AND an event is broadcast.
func TestSweepStuckProvisioning_FlipsOverdue(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT id FROM workspaces`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-stuck"))

	mock.ExpectExec(`UPDATE workspaces`).
		WithArgs("ws-stuck", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	emit := &fakeEmitter{}
	sweepStuckProvisioning(context.Background(), emit)

	if emit.count() != 1 {
		t.Fatalf("expected 1 event, got %d", emit.count())
	}
	if emit.events[0].Type != "WORKSPACE_PROVISION_FAILED" {
		t.Errorf("event type = %q, want WORKSPACE_PROVISION_FAILED", emit.events[0].Type)
	}
	if emit.events[0].WorkspaceID != "ws-stuck" {
		t.Errorf("workspace id = %q, want ws-stuck", emit.events[0].WorkspaceID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestSweepStuckProvisioning_RaceSafe covers the case where UPDATE affects
// 0 rows because the workspace flipped to online (or got restarted) between
// the SELECT and the UPDATE. We should skip the event, not emit a false
// WORKSPACE_PROVISION_FAILED.
func TestSweepStuckProvisioning_RaceSafe(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT id FROM workspaces`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-raced"))

	mock.ExpectExec(`UPDATE workspaces`).
		WithArgs("ws-raced", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows — raced

	emit := &fakeEmitter{}
	sweepStuckProvisioning(context.Background(), emit)

	if emit.count() != 0 {
		t.Errorf("expected 0 events on race, got %d", emit.count())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestSweepStuckProvisioning_NoStuck verifies that an empty candidate list
// produces no events and no update queries.
func TestSweepStuckProvisioning_NoStuck(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT id FROM workspaces`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	emit := &fakeEmitter{}
	sweepStuckProvisioning(context.Background(), emit)

	if emit.count() != 0 {
		t.Errorf("expected 0 events when nothing stuck, got %d", emit.count())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestSweepStuckProvisioning_MultipleStuck covers the realistic case where
// both agents (claude-code + hermes) are stuck — both should get flipped
// and both should get events.
func TestSweepStuckProvisioning_MultipleStuck(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT id FROM workspaces`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow("ws-claude-code").
			AddRow("ws-hermes"))

	mock.ExpectExec(`UPDATE workspaces`).
		WithArgs("ws-claude-code", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE workspaces`).
		WithArgs("ws-hermes", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	emit := &fakeEmitter{}
	sweepStuckProvisioning(context.Background(), emit)

	if emit.count() != 2 {
		t.Fatalf("expected 2 events, got %d", emit.count())
	}
}

// TestSweepStuckProvisioning_BroadcastFailureDoesNotCrash ensures the
// sweeper tolerates a broadcast error (Redis hiccup) — the DB row is
// already flipped so the state stays coherent.
func TestSweepStuckProvisioning_BroadcastFailureDoesNotCrash(t *testing.T) {
	mock := setupTestDB(t)

	mock.ExpectQuery(`SELECT id FROM workspaces`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-stuck"))
	mock.ExpectExec(`UPDATE workspaces`).
		WithArgs("ws-stuck", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	emit := &fakeEmitter{fail: true}
	// Must not panic.
	sweepStuckProvisioning(context.Background(), emit)
}

// TestProvisioningTimeout_EnvOverride verifies PROVISION_TIMEOUT_SECONDS
// env var takes effect when set to a positive integer, and falls back to
// default otherwise.
func TestProvisioningTimeout_EnvOverride(t *testing.T) {
	t.Setenv("PROVISION_TIMEOUT_SECONDS", "60")
	if got := provisioningTimeout(); got.Seconds() != 60 {
		t.Errorf("override: got %v, want 60s", got)
	}
	t.Setenv("PROVISION_TIMEOUT_SECONDS", "")
	if got := provisioningTimeout(); got != DefaultProvisioningTimeout {
		t.Errorf("default: got %v, want %v", got, DefaultProvisioningTimeout)
	}
	t.Setenv("PROVISION_TIMEOUT_SECONDS", "not-a-number")
	if got := provisioningTimeout(); got != DefaultProvisioningTimeout {
		t.Errorf("bad override: got %v, want default %v", got, DefaultProvisioningTimeout)
	}
}
