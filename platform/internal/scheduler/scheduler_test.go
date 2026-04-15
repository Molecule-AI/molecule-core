package scheduler

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
)

// setupTestDB replaces the global db.DB with a sqlmock and returns the mock
// handle. The real DB is restored (by closing the mock conn) via t.Cleanup.
func setupTestDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	db.DB = mockDB
	t.Cleanup(func() { mockDB.Close() })
	return mock
}

// panicProxy is a test double whose ProxyA2ARequest always panics.
// Used by TestPanicRecovery to verify the scheduler's goroutine-level
// panic recovery in fireSchedule.
type panicProxy struct{}

func (p *panicProxy) ProxyA2ARequest(
	_ context.Context, _ string, _ []byte, _ string, _ bool,
) (int, []byte, error) {
	panic("simulated A2A proxy panic")
}

// ── TestLastTickAt_zero ───────────────────────────────────────────────────────

// TestLastTickAt_zero confirms that LastTickAt returns a zero time.Time on a
// freshly-created scheduler that has never been started or ticked.
func TestLastTickAt_zero(t *testing.T) {
	s := New(nil, nil)
	if got := s.LastTickAt(); !got.IsZero() {
		t.Errorf("LastTickAt() before any tick = %v, want zero time.Time", got)
	}
}

// ── TestHealthy_beforeStart ───────────────────────────────────────────────────

// TestHealthy_beforeStart confirms that Healthy returns false when lastTickAt
// is zero (scheduler created but never ticked).
func TestHealthy_beforeStart(t *testing.T) {
	s := New(nil, nil)
	if s.Healthy() {
		t.Error("Healthy() = true on a scheduler that has never ticked, want false")
	}
}

// ── TestHealthy_freshTick ─────────────────────────────────────────────────────

// TestHealthy_freshTick sets lastTickAt to the current time — mirroring what
// tick() does after a completed poll cycle — and confirms Healthy returns true.
func TestHealthy_freshTick(t *testing.T) {
	s := New(nil, nil)

	// Simulate what tick() writes after wg.Wait() returns.
	s.mu.Lock()
	s.lastTickAt = time.Now()
	s.mu.Unlock()

	if !s.Healthy() {
		t.Error("Healthy() = false immediately after a fresh tick timestamp, want true")
	}
}

// ── TestHealthy_stale ─────────────────────────────────────────────────────────

// TestHealthy_stale backdates lastTickAt by 3×pollInterval (well beyond the
// 2×pollInterval liveness window) and confirms Healthy returns false.
func TestHealthy_stale(t *testing.T) {
	s := New(nil, nil)

	s.mu.Lock()
	s.lastTickAt = time.Now().Add(-3 * pollInterval) // 90 s ago; threshold is 60 s
	s.mu.Unlock()

	if s.Healthy() {
		t.Errorf("Healthy() = true when lastTickAt is 3×pollInterval (%s) ago, want false",
			3*pollInterval)
	}
}

// ── TestComputeNextRun_valid ──────────────────────────────────────────────────

// TestComputeNextRun_valid checks that "0 * * * *" (top-of-hour) returns a
// future time whose Minute() == 0 when the reference is mid-hour.
func TestComputeNextRun_valid(t *testing.T) {
	// 2025-01-01 12:30 UTC — clearly mid-hour so "next" top-of-hour is 13:00.
	ref := time.Date(2025, 1, 1, 12, 30, 0, 0, time.UTC)

	next, err := ComputeNextRun("0 * * * *", "UTC", ref)
	if err != nil {
		t.Fatalf("ComputeNextRun(valid expr) returned unexpected error: %v", err)
	}
	if !next.After(ref) {
		t.Errorf("ComputeNextRun() = %v, want a time strictly after ref %v", next, ref)
	}
	if next.Minute() != 0 {
		t.Errorf("ComputeNextRun() minute = %d, want 0 (top of hour)", next.Minute())
	}
}

// ── TestComputeNextRun_invalid ────────────────────────────────────────────────

// TestComputeNextRun_invalid confirms that an unparseable cron expression
// returns a non-nil error.
func TestComputeNextRun_invalid(t *testing.T) {
	_, err := ComputeNextRun("not-a-cron", "UTC", time.Now())
	if err == nil {
		t.Error("ComputeNextRun(invalid cron expr) = nil, want non-nil error")
	}
}

// ── TestComputeNextRun_invalidTimezone ────────────────────────────────────────

// TestComputeNextRun_invalidTimezone confirms that an unrecognised IANA
// timezone name returns a non-nil error (rather than silently falling back
// to UTC, which could mask misconfigured schedules).
func TestComputeNextRun_invalidTimezone(t *testing.T) {
	_, err := ComputeNextRun("0 * * * *", "Not/AZone", time.Now())
	if err == nil {
		t.Error("ComputeNextRun(invalid tz) = nil, want non-nil error")
	}
}

// ── TestPanicRecovery ─────────────────────────────────────────────────────────

// TestPanicRecovery verifies that a panic inside a fireSchedule goroutine does
// NOT crash the scheduler.
//
// The test calls tick() directly with a sqlmock that surfaces one due schedule.
// panicProxy causes ProxyA2ARequest to panic; the deferred recover() in
// fireSchedule catches it.  After tick() returns, lastTickAt must be set and
// Healthy() must return true — proving the scheduler survived.
//
// Without panic recovery an unrecovered goroutine panic terminates the entire
// test binary, so the test completing is itself evidence that recovery worked.
func TestPanicRecovery(t *testing.T) {
	mock := setupTestDB(t)

	// WorkspaceID must be ≥12 chars: fireSchedule slices it with [:12] for logging.
	schedRows := sqlmock.NewRows(
		[]string{"id", "workspace_id", "name", "cron_expr", "timezone", "prompt"},
	).AddRow(
		"sched-panic-01",       // id
		"ws-panic-workspace-1", // workspace_id (21 chars > 12)
		"panic-job",            // name
		"* * * * *",            // cron_expr
		"UTC",                  // timezone
		"fire and panic",       // prompt
	)
	mock.ExpectQuery(`SELECT id, workspace_id`).WillReturnRows(schedRows)

	s := New(&panicProxy{}, nil)

	// tick() launches fireSchedule in a goroutine that will panic.
	// If there is no recovery, the goroutine crash terminates the process here.
	s.tick(context.Background())

	// tick() returned normally → the panic was caught, wg.Wait() completed,
	// and lastTickAt was updated.
	if !s.Healthy() {
		t.Error("Healthy() = false after panic-recovery tick, want true " +
			"— scheduler must survive a panicking A2A proxy")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet DB expectations: %v", err)
	}
}

// ── TestShort_helper ──────────────────────────────────────────────────────────
// Regression guard for the short() helper that replaced unsafe [:N] slices
// after code review. Panicked when IDs were shorter than the slice bound.

func TestShort_helper(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"abcdef1234567890", 8, "abcdef12"},
		{"abc", 8, "abc"}, // shorter than n — no panic, no truncation
		{"", 8, ""},
		{"12345678", 8, "12345678"}, // exactly n
	}
	for _, tc := range cases {
		if got := short(tc.in, tc.n); got != tc.want {
			t.Errorf("short(%q, %d) = %q, want %q", tc.in, tc.n, got, tc.want)
		}
	}
}

// ── TestRecordSkipped_writesSkippedStatus ────────────────────────────────────
// #115 coverage gap: the recordSkipped path wasn't tested at all when it
// first landed. Exercises the UPDATE workspace_schedules + INSERT into
// activity_logs via sqlmock. Broadcaster is nil so we don't need to stub
// RecordAndBroadcast (the nil-check in recordSkipped handles that).

func TestRecordSkipped_writesSkippedStatus(t *testing.T) {
	mock := setupTestDB(t)
	s := New(nil, nil)

	sched := scheduleRow{
		ID:          "11111111-1111-1111-1111-111111111111",
		WorkspaceID: "22222222-2222-2222-2222-222222222222",
		Name:        "Hourly security audit",
		CronExpr:    "17 * * * *",
		Timezone:    "UTC",
		Prompt:      "audit",
	}

	// Expect the schedule-row UPDATE with last_status='skipped' and the
	// cron_run activity_logs INSERT with status='skipped' + error_detail
	// carrying the active_tasks reason.
	mock.ExpectExec(`UPDATE workspace_schedules`).
		WithArgs(sched.ID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO activity_logs`).
		WithArgs(sched.WorkspaceID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	s.recordSkipped(context.Background(), sched, 3)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ── TestRecordSkipped_shortWorkspaceIDNoPanic ─────────────────────────────────
// Guards against the short() regression: recordSkipped must not panic if
// WorkspaceID is unexpectedly shorter than the 12-char prefix used in logs.

func TestRecordSkipped_shortWorkspaceIDNoPanic(t *testing.T) {
	mock := setupTestDB(t)
	s := New(nil, nil)

	// 4-char workspace id — shorter than any substring bound in the code.
	sched := scheduleRow{
		ID:          "11111111-1111-1111-1111-111111111111",
		WorkspaceID: "ws-x",
		Name:        "test",
		CronExpr:    "0 * * * *",
		Timezone:    "UTC",
	}
	mock.ExpectExec(`UPDATE workspace_schedules`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO activity_logs`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("recordSkipped panicked on short WorkspaceID: %v", r)
		}
	}()
	s.recordSkipped(context.Background(), sched, 1)
}
