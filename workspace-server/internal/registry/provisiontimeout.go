package registry

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
)

// ProvisionTimeoutEmitter is the narrow broadcaster dependency the sweeper
// needs. Defined locally so the registry package stays event-bus agnostic
// (same pattern as OfflineHandler in healthsweep.go).
type ProvisionTimeoutEmitter interface {
	RecordAndBroadcast(ctx context.Context, eventType string, workspaceID string, payload interface{}) error
}

// DefaultProvisioningTimeout is how long a workspace may sit in
// status='provisioning' before the sweeper flips it to 'failed'. The
// container-launch path has its own 3-minute context timeout
// (provisioner.ProvisionTimeout) but that only bounds the docker API call —
// a container that started but crashes before /registry/register never
// triggers that path and would sit in provisioning forever. 10 minutes
// covers pathological image-pull + user-data execution on a cold EC2 worker
// while still getting well ahead of the "15+ minute" stuck state users see
// in production.
const DefaultProvisioningTimeout = 10 * time.Minute

// DefaultProvisionSweepInterval is how often the sweeper polls. Same cadence
// as the hibernation monitor — cheap and bounded by the provisioning-state
// query which hits the primary key / status partial index.
const DefaultProvisionSweepInterval = 30 * time.Second

// provisioningTimeout reads the override from env, falling back to the
// default. Env var expressed in seconds so operators can tune via a normal
// container restart without a code change.
func provisioningTimeout() time.Duration {
	if v := os.Getenv("PROVISION_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return DefaultProvisioningTimeout
}

// StartProvisioningTimeoutSweep periodically scans for workspaces stuck in
// `status='provisioning'` past the timeout window, flips them to `failed`,
// and broadcasts a WORKSPACE_PROVISION_TIMEOUT event so the canvas can
// render a fail-state instead of the indefinite cosmetic "Provisioning
// Timeout" banner.
//
// The sweep is idempotent: the UPDATE's WHERE clause re-checks both status
// and age under the same row lock, so a workspace that raced to `online` or
// was restarted while the sweep was scanning will not get flipped.
func StartProvisioningTimeoutSweep(ctx context.Context, emitter ProvisionTimeoutEmitter, interval time.Duration) {
	if emitter == nil {
		log.Println("Provision-timeout sweep: emitter is nil — skipping (no one to broadcast to)")
		return
	}
	if interval <= 0 {
		interval = DefaultProvisionSweepInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Provision-timeout sweep: started (interval=%s, timeout=%s)", interval, provisioningTimeout())

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepStuckProvisioning(ctx, emitter)
		}
	}
}

// sweepStuckProvisioning is one tick of the sweeper. Exported-for-test via
// the package boundary: keep all time.Now reads inside so tests can drive it
// deterministically by seeding updated_at rather than manipulating time.
func sweepStuckProvisioning(ctx context.Context, emitter ProvisionTimeoutEmitter) {
	timeout := provisioningTimeout()
	timeoutSec := int(timeout / time.Second)

	// Read candidates first so the event broadcast can include each id. The
	// subsequent UPDATE re-checks the predicate to stay race-safe against
	// concurrent restart / register paths that write updated_at.
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id FROM workspaces
		WHERE status = 'provisioning'
		  AND updated_at < now() - ($1 || ' seconds')::interval
	`, timeoutSec)
	if err != nil {
		log.Printf("Provision-timeout sweep: query error: %v", err)
		return
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}

	for _, id := range ids {
		msg := "provisioning timed out — container never reported online. Check the workspace's required env vars and retry."
		res, err := db.DB.ExecContext(ctx, `
			UPDATE workspaces
			   SET status = 'failed',
			       last_sample_error = $2,
			       updated_at = now()
			 WHERE id = $1
			   AND status = 'provisioning'
			   AND updated_at < now() - ($3 || ' seconds')::interval
		`, id, msg, timeoutSec)
		if err != nil {
			log.Printf("Provision-timeout sweep: failed to flip %s to failed: %v", id, err)
			continue
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			// Raced with restart / register — no harm, just skip.
			continue
		}
		log.Printf("Provision-timeout sweep: %s stuck in provisioning > %s — marked failed", id, timeout)
		if emitErr := emitter.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_TIMEOUT", id, map[string]interface{}{
			"error":         msg,
			"timeout_secs":  timeoutSec,
		}); emitErr != nil {
			log.Printf("Provision-timeout sweep: broadcast failed for %s: %v", id, emitErr)
		}
	}
}
