package registry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// fakeReaper is a hand-rolled OrphanReaper for the sweeper tests.
// Records every Stop / RemoveVolume call so tests can assert which
// workspace IDs got reconciled.
type fakeReaper struct {
	mu             sync.Mutex
	listResponse   []string
	listErr        error
	stopErr        map[string]error
	removeVolErr   map[string]error
	stopCalls      []string
	removeVolCalls []string
}

func (f *fakeReaper) ListWorkspaceContainerIDPrefixes(_ context.Context) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResponse, nil
}

func (f *fakeReaper) Stop(_ context.Context, wsID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls = append(f.stopCalls, wsID)
	return f.stopErr[wsID]
}

func (f *fakeReaper) RemoveVolume(_ context.Context, wsID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removeVolCalls = append(f.removeVolCalls, wsID)
	return f.removeVolErr[wsID]
}

// TestSweepOnce_ReconcilesRunningRemovedRows — the core reconcile
// behavior: a container running for a workspace whose DB row is
// 'removed' gets stopped + volume removed.
func TestSweepOnce_ReconcilesRunningRemovedRows(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	// Docker reports two ws-* containers; one's row is 'removed'
	// (the leak), the other's is 'online' (the DB rightly excludes
	// it from the WHERE clause and we should NOT reap it).
	reaper := &fakeReaper{
		listResponse: []string{"abc123def456", "xyz789ghi012"},
	}

	// The query asks for status='removed' rows whose id matches the
	// LIKE patterns built from the running container prefixes. Mock
	// returns only the leaked one as a UUID-shaped full id.
	mock.ExpectQuery(`SELECT id::text\s+FROM workspaces`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow("abc123def456-0000-0000-0000-000000000000"))

	sweepOnce(context.Background(), reaper)

	if len(reaper.stopCalls) != 1 || reaper.stopCalls[0] != "abc123def456-0000-0000-0000-000000000000" {
		t.Errorf("Stop calls = %v, want exactly the leaked id", reaper.stopCalls)
	}
	if len(reaper.removeVolCalls) != 1 || reaper.removeVolCalls[0] != "abc123def456-0000-0000-0000-000000000000" {
		t.Errorf("RemoveVolume calls = %v, want exactly the leaked id", reaper.removeVolCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestSweepOnce_NoRunningContainers — Docker returns nothing, sweeper
// short-circuits without a DB query (no leak possible if no
// containers exist).
func TestSweepOnce_NoRunningContainers(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	reaper := &fakeReaper{listResponse: nil}

	// No DB query expected — if sweepOnce makes one anyway the
	// sqlmock will fail "unexpected query".
	sweepOnce(context.Background(), reaper)

	if len(reaper.stopCalls) != 0 {
		t.Errorf("Stop should not fire when no containers exist; got %v", reaper.stopCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestSweepOnce_DockerListErrorSkipsCycle — a Docker daemon hiccup
// must not cascade into a DB query (otherwise we'd reap based on
// stale information). Skip the cycle, retry next tick.
func TestSweepOnce_DockerListErrorSkipsCycle(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	reaper := &fakeReaper{listErr: errors.New("daemon unreachable")}
	sweepOnce(context.Background(), reaper)

	if len(reaper.stopCalls) != 0 {
		t.Errorf("Stop must not fire when Docker list failed; got %v", reaper.stopCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestSweepOnce_StopFailureLeavesVolume — if Stop fails, RemoveVolume
// MUST NOT fire. This is the same trap that motivated the sweeper:
// removing a volume held by a still-running container always errors
// with "volume in use", and we'd accumulate noise in the log without
// actually fixing anything. Leave the volume for the next sweep
// (which will retry Stop).
func TestSweepOnce_StopFailureLeavesVolume(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	reaper := &fakeReaper{
		listResponse: []string{"abc123def456"},
		stopErr: map[string]error{
			"abc123def456-0000-0000-0000-000000000000": errors.New("docker daemon timeout"),
		},
	}
	mock.ExpectQuery(`SELECT id::text\s+FROM workspaces`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow("abc123def456-0000-0000-0000-000000000000"))

	sweepOnce(context.Background(), reaper)

	if len(reaper.stopCalls) != 1 {
		t.Errorf("Stop should have been attempted exactly once, got %v", reaper.stopCalls)
	}
	if len(reaper.removeVolCalls) != 0 {
		t.Errorf("RemoveVolume must not fire when Stop failed; got %v", reaper.removeVolCalls)
	}
}

// TestSweepOnce_VolumeRemoveErrorIsNonFatal — RemoveVolume failures
// are logged but don't prevent processing other orphans in the same
// cycle. Belt + braces against a transient daemon issue mid-loop.
func TestSweepOnce_VolumeRemoveErrorIsNonFatal(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	reaper := &fakeReaper{
		listResponse: []string{"aaa111bbb222", "ccc333ddd444"},
		removeVolErr: map[string]error{
			"aaa111bbb222-0000-0000-0000-000000000000": errors.New("volume not found"),
		},
	}
	mock.ExpectQuery(`SELECT id::text\s+FROM workspaces`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow("aaa111bbb222-0000-0000-0000-000000000000").
			AddRow("ccc333ddd444-0000-0000-0000-000000000000"))

	sweepOnce(context.Background(), reaper)

	if len(reaper.stopCalls) != 2 {
		t.Errorf("both orphans should have been Stopped; got %v", reaper.stopCalls)
	}
	if len(reaper.removeVolCalls) != 2 {
		t.Errorf("both orphans should have had RemoveVolume attempted; got %v", reaper.removeVolCalls)
	}
}

// TestSweepOnce_FiltersNonWorkspacePrefixes — the Docker name filter
// is a SUBSTRING match so containers like "my-ws-thing" can slip
// through. The HasPrefix check in the provisioner trims those, but
// the in-sweeper isLikelyWorkspaceID guard is the second line of
// defence: anything outside the UUID alphabet (hex + dashes) is
// rejected before being turned into a SQL LIKE pattern. Locks in
// that no DB query fires when every prefix is filtered out.
func TestSweepOnce_FiltersNonWorkspacePrefixes(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	reaper := &fakeReaper{
		listResponse: []string{
			"not_a_uuid_at_all",            // underscore not in UUID alphabet
			"contains%wildcard",            // SQL LIKE wildcard — must not reach the query
			"contains_wildcard",            // SQL LIKE single-char wildcard
			"",                             // empty
			"valid-but-non-workspace-name", // dash + lowercase letters that aren't hex
		},
	}

	// No DB query expected — every prefix is rejected before the
	// query builds, so we short-circuit. sqlmock fails on any
	// unexpected query.
	sweepOnce(context.Background(), reaper)

	if len(reaper.stopCalls) != 0 {
		t.Errorf("Stop must not fire when all prefixes filtered; got %v", reaper.stopCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestIsLikelyWorkspaceID — pin the alphabet directly. This is the
// guard that prevents SQL LIKE wildcards (`%`, `_`) from reaching
// the sweeper's query.
func TestIsLikelyWorkspaceID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"abc123def456", true},
		{"abcdef-1234-5678-90ab-cdef00112233", true},
		{"ABC123DEF456", true}, // uppercase hex still allowed
		{"", false},
		{"abc_123", false},      // underscore (SQL LIKE single-char wildcard)
		{"abc%123", false},      // percent (SQL LIKE multi-char wildcard)
		{"hello world", false},  // space, non-hex letters
		{"valid-but-not", false}, // 'l', 't', 'n' aren't hex
		{"abc 123", false},
		{".../escape", false},
	}
	for _, tc := range cases {
		got := isLikelyWorkspaceID(tc.in)
		if got != tc.want {
			t.Errorf("isLikelyWorkspaceID(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestStartOrphanSweeper_NilReaperIsNoOp — tolerance for the
// nil-provisioner path used by some test harnesses.
func TestStartOrphanSweeper_NilReaperIsNoOp(t *testing.T) {
	// Should return immediately without panicking. Wrap in a goroutine
	// + done-channel so we can assert it didn't block.
	done := make(chan struct{})
	go func() {
		StartOrphanSweeper(context.Background(), nil)
		close(done)
	}()
	select {
	case <-done:
		// expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("StartOrphanSweeper(nil) blocked instead of returning immediately")
	}
}
