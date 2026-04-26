package registry

// orphan_sweeper.go — periodic reconcile pass that cleans up Docker
// containers whose corresponding workspace row in Postgres has
// status='removed'. Defence in depth on top of the inline cleanup
// in handlers/workspace_crud.go.
//
// Why this exists: the inline cleanup is one-shot — if Docker hiccups
// (daemon restart, host load, transient API error), the container
// silently stays alive while the DB row is already 'removed'. Without
// a reconcile pass those leaks accumulate forever. With one, every
// missed cleanup heals on the next sweep.
//
// Cost: O(running containers) per cycle, not O(historical removed
// rows). The Docker name filter trims the candidate set to ws-* only
// (typically the same handful as ContainerList without filter on a
// dev host); the DB lookup is one indexed query against the
// idx_workspaces_status btree.

import (
	"context"
	"log"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/lib/pq"
)

// OrphanReaper is the dependency the sweeper takes from provisioner.
// Extracted as an interface so the sweeper is unit-testable without
// a real Docker daemon — matches the ContainerChecker pattern in
// healthsweep.go. *provisioner.Provisioner satisfies this naturally.
type OrphanReaper interface {
	ListWorkspaceContainerIDPrefixes(ctx context.Context) ([]string, error)
	Stop(ctx context.Context, workspaceID string) error
	RemoveVolume(ctx context.Context, workspaceID string) error
}

// isLikelyWorkspaceID accepts strings shaped like a UUID prefix —
// hex chars and `-` only. Workspace IDs are full UUIDs and the
// container-name truncation keeps the hex prefix intact, so any
// container name that doesn't match this is by definition not one
// of ours and should be skipped. Also doubles as a SQL LIKE
// wildcard guard (rejects `_` and `%`).
func isLikelyWorkspaceID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

// OrphanSweepInterval is the cadence of the reconcile loop. 60s
// matches the heartbeat cadence (30s) × 2 — a single missed cleanup
// surfaces within ~90s end-to-end (canvas delete → next sweep tick →
// container gone). Faster cycles would just pay Docker API cost for
// no UX win; slower would let leaks linger long enough to compound
// CPU pressure on dev hosts.
const OrphanSweepInterval = 60 * time.Second

// orphanSweepDeadline bounds a single sweep cycle. A daemon at the
// edge of timing out shouldn't accumulate goroutines. 30s is generous
// for a dev host with dozens of containers and a busy daemon.
const orphanSweepDeadline = 30 * time.Second

// StartOrphanSweeper runs the reconcile loop until ctx is cancelled.
// nil reaper makes the loop a no-op (matches handlers'
// nil-provisioner-tolerant pattern — some test harnesses run without
// Docker available).
func StartOrphanSweeper(ctx context.Context, reaper OrphanReaper) {
	if reaper == nil {
		log.Println("Orphan sweeper: reaper is nil — sweeper disabled")
		return
	}
	log.Printf("Orphan sweeper started — reconciling every %s", OrphanSweepInterval)
	ticker := time.NewTicker(OrphanSweepInterval)
	defer ticker.Stop()
	// Run once immediately so a platform restart cleans up any
	// containers leaked while we were down — don't make the user
	// wait 60s for the first reconcile.
	sweepOnce(ctx, reaper)
	for {
		select {
		case <-ctx.Done():
			log.Println("Orphan sweeper: shutdown")
			return
		case <-ticker.C:
			sweepOnce(ctx, reaper)
		}
	}
}

func sweepOnce(parent context.Context, reaper OrphanReaper) {
	ctx, cancel := context.WithTimeout(parent, orphanSweepDeadline)
	defer cancel()

	prefixes, err := reaper.ListWorkspaceContainerIDPrefixes(ctx)
	if err != nil {
		log.Printf("Orphan sweeper: ListWorkspaceContainerIDPrefixes failed: %v — skipping cycle", err)
		return
	}
	if len(prefixes) == 0 {
		return
	}

	// Resolve each prefix to a full workspace_id whose status is
	// 'removed'. The platform's workspace IDs are full UUIDs but
	// container names are truncated to 12 chars — an UPPER BOUND
	// of one match per prefix is guaranteed by the DB (UUID v4
	// collisions in the first 12 chars across active rows are
	// statistically negligible). Use a single IN-style query so
	// the cost is one round-trip regardless of leak count.
	//
	// Defence: drop any prefix whose contents fall outside the
	// hex-and-dash UUID alphabet. Workspace IDs are UUIDs, so
	// container names follow ws-<12 hex chars>. Anything else is
	// either a non-workspace container that slipped past the
	// substring-match Docker filter (workspace-runner, etc.) or a
	// malformed entry — neither should be turned into a LIKE
	// pattern. Also blocks SQL LIKE wildcards (`_` and `%`) from
	// reaching the query, even though Docker's container-name
	// validation would already have rejected them upstream.
	likes := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		if !isLikelyWorkspaceID(p) {
			continue
		}
		likes = append(likes, p+"%")
	}
	if len(likes) == 0 {
		return
	}
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id::text
		  FROM workspaces
		 WHERE status = 'removed'
		   AND id::text LIKE ANY($1::text[])
	`, pq.Array(likes))
	if err != nil {
		log.Printf("Orphan sweeper: DB query failed: %v — skipping cycle", err)
		return
	}
	defer rows.Close()

	var orphanIDs []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			log.Printf("Orphan sweeper: row scan failed: %v", scanErr)
			continue
		}
		orphanIDs = append(orphanIDs, id)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Orphan sweeper: rows iteration failed: %v", err)
		return
	}

	for _, id := range orphanIDs {
		log.Printf("Orphan sweeper: stopping leaked container for removed workspace %s", id)
		if stopErr := reaper.Stop(ctx, id); stopErr != nil {
			// Stop returns the wrapped Docker error (treating
			// "container not found" as nil-success via
			// isContainerNotFound), so a non-nil here means the
			// container is genuinely still alive — daemon timeout,
			// ctx cancellation, or a transient socket EOF.
			// Skip RemoveVolume so we don't fall into the same
			// Stop-failed-then-volume-in-use trap that motivated
			// this sweeper. The next cycle (60s out) retries Stop.
			log.Printf("Orphan sweeper: Stop failed for %s: %v — leaving volume for next cycle", id, stopErr)
			continue
		}
		if rmErr := reaper.RemoveVolume(ctx, id); rmErr != nil {
			log.Printf("Orphan sweeper: RemoveVolume warning for %s: %v", id, rmErr)
		}
	}
}
