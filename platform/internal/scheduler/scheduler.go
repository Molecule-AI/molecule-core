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

	fireCtx, cancel := context.WithTimeout(ctx, fireTimeout)
	defer cancel()

	idPrefix := sched.ID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}
	msgID := fmt.Sprintf("cron-%s-%s", idPrefix, uuid.New().String()[:8])

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

	log.Printf("Scheduler: firing '%s' → workspace %s", sched.Name, sched.WorkspaceID[:12])

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
	}

	_, err := db.DB.ExecContext(ctx, `
		UPDATE workspace_schedules
		SET last_run_at = now(),
		    next_run_at = $2,
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
	_, _ = db.DB.ExecContext(ctx, `
		INSERT INTO activity_logs (workspace_id, activity_type, source_id, method, summary, request_body, status, created_at)
		VALUES ($1, 'cron_run', NULL, 'cron', $2, $3::jsonb, $4, now())
	`, sched.WorkspaceID, "Cron: "+sched.Name, string(cronMeta), lastStatus)

	if s.broadcaster != nil {
		s.broadcaster.RecordAndBroadcast(ctx, "CRON_EXECUTED", sched.WorkspaceID, map[string]interface{}{
			"schedule_id":   sched.ID,
			"schedule_name": sched.Name,
			"status":        lastStatus,
		})
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
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
