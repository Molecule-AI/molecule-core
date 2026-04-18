package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/scheduler"
)

// AdminSchedulesHealthHandler serves GET /admin/schedules/health — a cross-workspace
// schedule monitoring view gated behind AdminAuth. Unlike the per-workspace
// GET /workspaces/:id/schedules/health (which requires caller identity + CanCommunicate),
// this endpoint is intended for operators and automated audit agents that hold a
// global admin bearer token. Issue #618.
type AdminSchedulesHealthHandler struct{}

// NewAdminSchedulesHealthHandler returns an AdminSchedulesHealthHandler.
func NewAdminSchedulesHealthHandler() *AdminSchedulesHealthHandler {
	return &AdminSchedulesHealthHandler{}
}

// adminScheduleHealth is the per-schedule entry in the health response.
type adminScheduleHealth struct {
	WorkspaceID           string     `json:"workspace_id"`
	WorkspaceName         string     `json:"workspace_name"`
	ScheduleID            string     `json:"schedule_id"`
	ScheduleName          string     `json:"schedule_name"`
	CronExpr              string     `json:"cron_expr"`
	LastRunAt             *time.Time `json:"last_run_at"`
	ExpectedNextRun       *time.Time `json:"expected_next_run"`
	Status                string     `json:"status"` // "ok" | "stale" | "never_run"
	StaleThresholdSeconds int64      `json:"stale_threshold_seconds"`
}

// computeStaleThreshold returns 2× the cron interval for the given expression
// and timezone. The interval is approximated as the gap between two consecutive
// scheduled fire times computed from now.
//
// Exported as a package-level function so it can be unit-tested independently
// from the handler.
func computeStaleThreshold(cronExpr, tz string, now time.Time) (time.Duration, error) {
	t1, err := scheduler.ComputeNextRun(cronExpr, tz, now)
	if err != nil {
		return 0, err
	}
	t2, err := scheduler.ComputeNextRun(cronExpr, tz, t1)
	if err != nil {
		return 0, err
	}
	return 2 * t2.Sub(t1), nil
}

// Health handles GET /admin/schedules/health.
//
// It joins workspace_schedules with workspaces and, for each schedule, computes:
//   - status:                "never_run" (last_run_at IS NULL),
//     "stale" (now - last_run_at > 2 × cron interval), or
//     "ok" (recently run).
//   - stale_threshold_seconds: 2 × the cron interval derived from cron_expr.
//   - expected_next_run:     the next_run_at value stored by the scheduler.
//
// Returns 200 with a JSON array (empty if no schedules exist), 500 on DB error.
// Auth is enforced by the adminAuth() middleware registered in router.go.
func (h *AdminSchedulesHealthHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	now := time.Now()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT
			w.id          AS workspace_id,
			w.name        AS workspace_name,
			s.id          AS schedule_id,
			s.name        AS schedule_name,
			s.cron_expr,
			s.timezone,
			s.last_run_at,
			s.next_run_at
		FROM workspace_schedules s
		JOIN workspaces w ON w.id = s.workspace_id
		WHERE w.status != 'removed'
		ORDER BY w.name ASC, s.name ASC
	`)
	if err != nil {
		log.Printf("AdminSchedulesHealth: query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query schedules"})
		return
	}
	defer rows.Close()

	entries := make([]adminScheduleHealth, 0)
	for rows.Next() {
		var (
			workspaceID   string
			workspaceName string
			scheduleID    string
			scheduleName  string
			cronExpr      string
			timezone      string
			lastRunAt     *time.Time
			nextRunAt     *time.Time
		)
		if err := rows.Scan(
			&workspaceID, &workspaceName,
			&scheduleID, &scheduleName,
			&cronExpr, &timezone,
			&lastRunAt, &nextRunAt,
		); err != nil {
			log.Printf("AdminSchedulesHealth: scan error: %v", err)
			continue
		}

		// Compute stale threshold = 2 × cron interval.
		// On parse failure (malformed cron_expr in DB) we report 0 and still
		// classify the row — a bad cron_expr itself is worth surfacing in the
		// health view rather than silently skipping the row.
		staleThreshold, cronErr := computeStaleThreshold(cronExpr, timezone, now)
		var staleThresholdSeconds int64
		if cronErr == nil {
			staleThresholdSeconds = int64(staleThreshold.Seconds())
		} else {
			log.Printf("AdminSchedulesHealth: cron parse error for schedule %s (%q): %v",
				scheduleID, cronExpr, cronErr)
		}

		// Classify schedule status.
		status := classifyScheduleStatus(lastRunAt, staleThreshold, now)

		entries = append(entries, adminScheduleHealth{
			WorkspaceID:           workspaceID,
			WorkspaceName:         workspaceName,
			ScheduleID:            scheduleID,
			ScheduleName:          scheduleName,
			CronExpr:              cronExpr,
			LastRunAt:             lastRunAt,
			ExpectedNextRun:       nextRunAt,
			Status:                status,
			StaleThresholdSeconds: staleThresholdSeconds,
		})
	}
	if err := rows.Err(); err != nil {
		log.Printf("AdminSchedulesHealth: rows iteration error: %v", err)
	}

	c.JSON(http.StatusOK, entries)
}

// classifyScheduleStatus returns the health status string for a schedule.
//   - "never_run"  — last_run_at is NULL (schedule has never fired)
//   - "stale"      — now - last_run_at > staleThreshold (and threshold > 0)
//   - "ok"         — recently run within the expected window
func classifyScheduleStatus(lastRunAt *time.Time, staleThreshold time.Duration, now time.Time) string {
	if lastRunAt == nil {
		return "never_run"
	}
	if staleThreshold > 0 && now.Sub(*lastRunAt) > staleThreshold {
		return "stale"
	}
	return "ok"
}
