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
}

func New(proxy A2AProxy, broadcaster Broadcaster) *Scheduler {
	return &Scheduler{proxy: proxy, broadcaster: broadcaster}
}

// Start runs the scheduler poll loop. Blocks until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Printf("Scheduler: started (poll interval=%s)", pollInterval)

	// Heartbeat before the first tick so /admin/liveness doesn't flag stale
	// during the initial 30s interval after startup.
	supervised.Heartbeat("scheduler")

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler: stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
			supervised.Heartbeat("scheduler")
		}
	}
}

// tick queries all due schedules and fires each in a goroutine.
// Waits for all goroutines to finish before returning so the next tick
// doesn't re-fire schedules whose next_run_at hasn't been updated yet.
func (s *Scheduler) tick(ctx context.Context) {
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
			s.fireSchedule(ctx, s2)
		}(sched)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Scheduler: rows error: %v", err)
	}
	wg.Wait()
}

// fireSchedule sends the A2A message and updates the schedule row.
func (s *Scheduler) fireSchedule(ctx context.Context, sched scheduleRow) {
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
		loc = time.UTC
	}

	parser := cronlib.NewParser(cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow)
	sched, err := parser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}

	return sched.Next(after.In(loc)).UTC(), nil
}
