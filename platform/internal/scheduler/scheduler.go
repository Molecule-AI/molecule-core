package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	cronlib "github.com/robfig/cron/v3"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/supervised"
)

const (
	pollInterval   = 30 * time.Second
	maxConcurrent  = 10
	batchLimit     = 50
	fireTimeout    = 5 * time.Minute
)

// A2AProxy is the interface the scheduler needs to send messages to workspaces.
// WorkspaceHandler.ProxyA2ARequest satisfies this.
type A2AProxy interface {
	ProxyA2ARequest(ctx context.Context, workspaceID string, body []byte, callerID string, logActivity bool) (int, []byte, error)
}

// Broadcaster records events and pushes them to WebSocket clients.
type Broadcaster interface {
	RecordAndBroadcast(ctx context.Context, eventType, workspaceID string, data interface{}) error
}

type scheduleRow struct {
	ID          string
	WorkspaceID string
	Name        string
	CronExpr    string
	Timezone    string
	Prompt      string
}

// Scheduler polls the workspace_schedules table and fires A2A messages
// when a schedule's next_run_at has passed. Follows the same goroutine
// pattern as registry.StartHealthSweep.
type Scheduler struct {
	proxy       A2AProxy
	broadcaster Broadcaster

	// lastTickAt records the wall-clock time of the most recent tick
	// (whether it fired schedules or not). Read by Healthy() and the
	// /admin/scheduler/health endpoint to detect stuck-tick conditions.
	// Atomic-ish via the mutex; tick rate is 30s so contention is trivial.
	mu           sync.RWMutex
	lastTickAt   time.Time
	tickInterval time.Duration // defaults to pollInterval; overridable in tests
}

func New(proxy A2AProxy, broadcaster Broadcaster) *Scheduler {
	return &Scheduler{
		proxy:        proxy,
		broadcaster:  broadcaster,
		tickInterval: pollInterval,
	}
}

// LastTickAt returns the wall-clock time of the most recently completed tick.
// Returns a zero time.Time if the scheduler has never completed a tick.
func (s *Scheduler) LastTickAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastTickAt
}

// Healthy returns true if the scheduler has completed a tick within the last
// 2×pollInterval window. Returns false before the first tick or if the
// scheduler is stalled.
func (s *Scheduler) Healthy() bool {
	s.mu.RLock()
	t := s.lastTickAt
	s.mu.RUnlock()
	if t.IsZero() {
		return false
	}
	return time.Since(t) < 2*pollInterval
}

// repairNullNextRunAt patches enabled schedules whose next_run_at is NULL.
// This can happen when a previous ComputeNextRun call failed at fire-time or
// import-time and the protective COALESCE wasn't in place (issue #722 Bug 3).
// Called once at startup, before the first tick, so affected schedules are
// never permanently silenced — the poll loop would skip them forever because
// tick() filters WHERE next_run_at IS NOT NULL.
func (s *Scheduler) repairNullNextRunAt(ctx context.Context) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, cron_expr, timezone
		FROM workspace_schedules
		WHERE enabled = true AND next_run_at IS NULL
	`)
	if err != nil {
		log.Printf("Scheduler: repairNullNextRunAt: query error: %v", err)
		return
	}
	defer rows.Close()

	repaired := 0
	for rows.Next() {
		var id, cronExpr, tz string
		if err := rows.Scan(&id, &cronExpr, &tz); err != nil {
			log.Printf("Scheduler: repairNullNextRunAt: scan error: %v", err)
			continue
		}
		nextRun, nextErr := ComputeNextRun(cronExpr, tz, time.Now())
		if nextErr != nil {
			log.Printf("Scheduler: repairNullNextRunAt: ComputeNextRun failed for %s (expr=%q tz=%q): %v — leaving NULL", short(id, 12), cronExpr, tz, nextErr)
			continue
		}
		if _, err := db.DB.ExecContext(ctx,
			`UPDATE workspace_schedules SET next_run_at = $2, updated_at = now() WHERE id = $1`,
			id, nextRun,
		); err != nil {
			log.Printf("Scheduler: repairNullNextRunAt: update error for %s: %v", short(id, 12), err)
			continue
		}
		repaired++
		log.Printf("Scheduler: repairNullNextRunAt: repaired %s → next_run_at=%s", short(id, 12), nextRun.Format(time.RFC3339))
	}
	// rows.Err() surfaces any error that cut the iteration short (network blip,
	// context cancel, etc.). Without this check a partial repair would be
	// silently treated as complete — some NULL-next_run_at schedules would stay
	// silenced until the next startup.
	if err := rows.Err(); err != nil {
		log.Printf("Scheduler: repairNullNextRunAt: row iteration error: %v", err)
		return
	}
	if repaired > 0 {
		log.Printf("Scheduler: repairNullNextRunAt: repaired %d schedule(s)", repaired)
	}
}

// Start runs the scheduler poll loop. Blocks until ctx is cancelled.
//
// Defends against panics inside tick() so a single bad row / bad cron
// expression / DB blip can't permanently kill the scheduler. Without
// this recover the goroutine dies and the only signal to the operator
// is "no crons firing" — which we observed as a 12+ hour silent outage
// on 2026-04-14 (issue #85).
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	// Repair any schedules silenced by a prior ComputeNextRun failure (#722).
	s.repairNullNextRunAt(ctx)

	log.Printf("Scheduler: started (poll interval=%s)", s.tickInterval)

	tickWithRecover := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Scheduler: PANIC in tick — recovered: %v (next tick in %s)", r, pollInterval)
			}
		}()
		s.tick(ctx)
		s.mu.Lock()
		s.lastTickAt = time.Now()
		s.mu.Unlock()
	}

	// Heartbeat + initial lastTickAt so /admin/liveness and Healthy() both
	// pass during the first 30s interval after startup.
	supervised.Heartbeat("scheduler")
	s.mu.Lock()
	s.lastTickAt = time.Now()
	s.mu.Unlock()

	// Independent heartbeat pulse (#140). Decoupled from tick completion so
	// a single long fire (UIUX audits routinely take 60-120s; max fireTimeout
	// is 5min) can't make /admin/liveness look stale for the whole fire window.
	// tick() also calls Heartbeat at its top + each fire goroutine calls it
	// entry/exit — those are kept as redundant signals but this pulse is the
	// one that guarantees liveness freshness regardless of tick state.
	go func() {
		pulseTicker := time.NewTicker(10 * time.Second)
		defer pulseTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-pulseTicker.C:
				supervised.Heartbeat("scheduler")
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler: stopped")
			return
		case <-ticker.C:
			tickWithRecover()
			supervised.Heartbeat("scheduler")
		}
	}
}

// tick queries all due schedules and fires each in a goroutine.
// Waits for all goroutines to finish before returning so the next tick
// doesn't re-fire schedules whose next_run_at hasn't been updated yet.
//
// Heartbeat is called at three points to keep /admin/liveness fresh during
// long-running fires (some prompts take minutes — without these heartbeats
// the scheduler looks "stale" the whole time it's working):
//   - immediately on entering tick (proves we're past the ticker.C wait)
//   - inside each per-fire goroutine (every fire bumps the heartbeat)
//   - implicitly via the post-tick heartbeat in Start()
func (s *Scheduler) tick(ctx context.Context) {
	supervised.Heartbeat("scheduler")

	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, workspace_id, name, cron_expr, timezone, prompt
		FROM workspace_schedules
		WHERE enabled = true AND next_run_at IS NOT NULL AND next_run_at <= now()
		ORDER BY next_run_at ASC
		LIMIT $1
	`, batchLimit)
	if err != nil {
		log.Printf("Scheduler: tick query error: %v", err)
		return
	}
	defer rows.Close()

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)
	for rows.Next() {
		var sched scheduleRow
		if err := rows.Scan(&sched.ID, &sched.WorkspaceID, &sched.Name, &sched.CronExpr, &sched.Timezone, &sched.Prompt); err != nil {
			log.Printf("Scheduler: scan error: %v", err)
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(s2 scheduleRow) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Scheduler: PANIC firing '%s' on workspace %s — recovered: %v",
						s2.Name, s2.WorkspaceID, r)
				}
			}()
			supervised.Heartbeat("scheduler")
			s.fireSchedule(ctx, s2)
			supervised.Heartbeat("scheduler")
		}(sched)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Scheduler: rows error: %v", err)
	}
	wg.Wait()

	// Record tick completion time for health monitoring.
	s.mu.Lock()
	s.lastTickAt = time.Now()
	s.mu.Unlock()
}

// fireSchedule sends the A2A message and updates the schedule row.
// A deferred recover guards against panics in the A2A proxy so that a single
// misbehaving workspace cannot crash the scheduler goroutine pool.
func (s *Scheduler) fireSchedule(ctx context.Context, sched scheduleRow) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Scheduler: panic recovered in fireSchedule for '%s' (%s): %v",
				sched.Name, sched.ID, r)
		}
	}()

	// #115 concurrency-aware skip — before firing check if the target
	// workspace is already executing a task. If so, skip this tick instead
	// of colliding (which used to surface as "workspace agent busy" errors
	// and register as a hard fail). advance next_run_at so the next cron
	// slot gets a fresh chance; log a skipped cron_run row so history shows
	// the gap instead of a silent miss. COALESCE guards against NULL.
	var activeTasks int
	if err := db.DB.QueryRowContext(ctx,
		`SELECT COALESCE(active_tasks, 0) FROM workspaces WHERE id = $1`,
		sched.WorkspaceID,
	).Scan(&activeTasks); err == nil && activeTasks > 0 {
		log.Printf("Scheduler: skipping '%s' on busy workspace %s (active_tasks=%d)",
			sched.Name, short(sched.WorkspaceID, 12), activeTasks)
		s.recordSkipped(ctx, sched, activeTasks)
		return
	}

	fireCtx, cancel := context.WithTimeout(ctx, fireTimeout)
	defer cancel()

	msgID := fmt.Sprintf("cron-%s-%s", short(sched.ID, 8), uuid.New().String()[:8])

	a2aBody, _ := json.Marshal(map[string]interface{}{
		"method": "message/send",
		"params": map[string]interface{}{
			"message": map[string]interface{}{
				"role":      "user",
				"messageId": msgID,
				"parts":     []map[string]interface{}{{"kind": "text", "text": sched.Prompt}},
			},
		},
	})

	log.Printf("Scheduler: firing '%s' → workspace %s", sched.Name, short(sched.WorkspaceID, 12))

	// Empty callerID = canvas-style request (bypasses access control, source_id=NULL in activity log).
	// "system:scheduler" was invalid — source_id column is UUID and rejects non-UUID strings.
	statusCode, _, proxyErr := s.proxy.ProxyA2ARequest(fireCtx, sched.WorkspaceID, a2aBody, "", true)

	lastStatus := "ok"
	lastError := ""
	if proxyErr != nil {
		lastStatus = "error"
		lastError = fmt.Sprintf("%v", proxyErr)
		log.Printf("Scheduler: '%s' error: %v", sched.Name, proxyErr)
	} else if statusCode < 200 || statusCode >= 300 {
		lastStatus = "error"
		lastError = fmt.Sprintf("HTTP %d", statusCode)
		log.Printf("Scheduler: '%s' non-2xx: %d", sched.Name, statusCode)
	} else {
		log.Printf("Scheduler: '%s' completed (HTTP %d)", sched.Name, statusCode)
	}

	nextRun, nextErr := ComputeNextRun(sched.CronExpr, sched.Timezone, time.Now())
	var nextRunPtr *time.Time
	if nextErr == nil {
		nextRunPtr = &nextRun
	} else {
		// #722 Bug 1: log the failure so it's not silent; COALESCE below
		// preserves the existing next_run_at rather than writing NULL.
		log.Printf("Scheduler: ComputeNextRun failed for '%s' (expr=%q tz=%q): %v — preserving existing next_run_at",
			sched.Name, sched.CronExpr, sched.Timezone, nextErr)
	}

	_, err := db.DB.ExecContext(ctx, `
		UPDATE workspace_schedules
		SET last_run_at = now(),
		    next_run_at = COALESCE($2, next_run_at),
		    run_count = run_count + 1,
		    last_status = $3,
		    last_error = $4,
		    updated_at = now()
		WHERE id = $1
	`, sched.ID, nextRunPtr, lastStatus, lastError)
	if err != nil {
		log.Printf("Scheduler: update error for %s: %v", sched.ID, err)
	}

	// Log a dedicated cron_run activity entry with schedule metadata so the
	// history endpoint can query by schedule_id.
	cronMeta, _ := json.Marshal(map[string]interface{}{
		"schedule_id":   sched.ID,
		"schedule_name": sched.Name,
		"cron_expr":     sched.CronExpr,
		"prompt":        truncate(sched.Prompt, 200),
	})
	// #152: persist lastError into error_detail on the activity_logs row
	// so GET /workspaces/:id/schedules/:id/history can surface why a run
	// failed (previously dropped — history returned status without any
	// error context, making root-cause debugging impossible).
	_, _ = db.DB.ExecContext(ctx, `
		INSERT INTO activity_logs (workspace_id, activity_type, source_id, method, summary, request_body, status, error_detail, created_at)
		VALUES ($1, 'cron_run', NULL, 'cron', $2, $3::jsonb, $4, $5, now())
	`, sched.WorkspaceID, "Cron: "+sched.Name, string(cronMeta), lastStatus, lastError)

	if s.broadcaster != nil {
		s.broadcaster.RecordAndBroadcast(ctx, "CRON_EXECUTED", sched.WorkspaceID, map[string]interface{}{
			"schedule_id":   sched.ID,
			"schedule_name": sched.Name,
			"status":        lastStatus,
		})
	}
}

// recordSkipped advances next_run_at and logs a cron_run activity entry
// with status='skipped' when the target workspace was already busy.
// Issue #115 — replaces the previous "busy → fire → fail → retry next
// tick" loop with "busy → skip → advance → try next slot". Keeps the
// history surface honest (a skip is not an error) and stops filling
// last_error with noise.
func (s *Scheduler) recordSkipped(ctx context.Context, sched scheduleRow, activeTasks int) {
	reason := fmt.Sprintf("skipped: workspace busy (active_tasks=%d)", activeTasks)

	nextRun, nextErr := ComputeNextRun(sched.CronExpr, sched.Timezone, time.Now())
	var nextRunPtr *time.Time
	if nextErr == nil {
		nextRunPtr = &nextRun
	} else {
		// #722 Bug 2: same guard as fireSchedule — log and preserve existing
		// next_run_at via COALESCE rather than silencing the schedule with NULL.
		log.Printf("Scheduler: ComputeNextRun failed in recordSkipped for '%s' (expr=%q tz=%q): %v — preserving existing next_run_at",
			sched.Name, sched.CronExpr, sched.Timezone, nextErr)
	}

	// Advance next_run_at + bump run_count so the liveness view reflects
	// that we're still ticking. last_status='skipped', last_error carries
	// the reason for operators debugging via the schedule history API.
	// COALESCE($2, next_run_at): if ComputeNextRun failed, preserve the
	// existing next_run_at rather than writing NULL (#722).
	_, _ = db.DB.ExecContext(ctx, `
		UPDATE workspace_schedules
		SET last_run_at = now(),
		    next_run_at = COALESCE($2, next_run_at),
		    run_count = run_count + 1,
		    last_status = 'skipped',
		    last_error = $3,
		    updated_at = now()
		WHERE id = $1
	`, sched.ID, nextRunPtr, reason)

	cronMeta, _ := json.Marshal(map[string]interface{}{
		"schedule_id":   sched.ID,
		"schedule_name": sched.Name,
		"cron_expr":     sched.CronExpr,
		"skipped":       true,
		"active_tasks":  activeTasks,
	})
	_, _ = db.DB.ExecContext(ctx, `
		INSERT INTO activity_logs (workspace_id, activity_type, source_id, method, summary, request_body, status, error_detail, created_at)
		VALUES ($1, 'cron_run', NULL, 'cron', $2, $3::jsonb, 'skipped', $4, now())
	`, sched.WorkspaceID, "Cron skipped: "+sched.Name, string(cronMeta), reason)

	if s.broadcaster != nil {
		_ = s.broadcaster.RecordAndBroadcast(ctx, "CRON_SKIPPED", sched.WorkspaceID, map[string]interface{}{
			"schedule_id":   sched.ID,
			"schedule_name": sched.Name,
			"reason":        reason,
		})
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// short returns up to n leading characters of s without panicking when s is
// shorter than n. Used to safely display UUID prefixes in log lines where
// the full ID would be noisy but the full-length bounds check is repetitive.
func short(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ComputeNextRun parses a cron expression and returns the next fire time
// after the given time, in the specified timezone.
func ComputeNextRun(cronExpr, tz string, after time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}

	parser := cronlib.NewParser(cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow)
	sched, err := parser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}

	return sched.Next(after.In(loc)).UTC(), nil
}
