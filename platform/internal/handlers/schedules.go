package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/scheduler"
)

type ScheduleHandler struct{}

func NewScheduleHandler() *ScheduleHandler {
	return &ScheduleHandler{}
}

type scheduleResponse struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	Name        string     `json:"name"`
	CronExpr    string     `json:"cron_expr"`
	Timezone    string     `json:"timezone"`
	Prompt      string     `json:"prompt"`
	Enabled     bool       `json:"enabled"`
	LastRunAt   *time.Time `json:"last_run_at"`
	NextRunAt   *time.Time `json:"next_run_at"`
	RunCount    int        `json:"run_count"`
	LastStatus  string     `json:"last_status"`
	LastError   string     `json:"last_error"`
	Source      string     `json:"source,omitempty"` // 'template' (seeded by org/import) | 'runtime' (created via Canvas/API). Issue #24.
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// List returns all schedules for a workspace.
func (h *ScheduleHandler) List(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, workspace_id, name, cron_expr, timezone, prompt, enabled,
		       last_run_at, next_run_at, run_count, last_status, last_error,
		       source, created_at, updated_at
		FROM workspace_schedules
		WHERE workspace_id = $1
		ORDER BY created_at ASC
	`, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query schedules"})
		return
	}
	defer rows.Close()

	schedules := make([]scheduleResponse, 0)
	for rows.Next() {
		var s scheduleResponse
		if err := rows.Scan(
			&s.ID, &s.WorkspaceID, &s.Name, &s.CronExpr, &s.Timezone,
			&s.Prompt, &s.Enabled, &s.LastRunAt, &s.NextRunAt, &s.RunCount,
			&s.LastStatus, &s.LastError, &s.Source, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			log.Printf("Schedules.List: scan error: %v", err)
			continue
		}
		schedules = append(schedules, s)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Schedules.List: rows error: %v", err)
	}

	c.JSON(http.StatusOK, schedules)
}

type createScheduleRequest struct {
	Name     string `json:"name"`
	CronExpr string `json:"cron_expr" binding:"required"`
	Timezone string `json:"timezone"`
	Prompt   string `json:"prompt" binding:"required"`
	Enabled  *bool  `json:"enabled"`
}

// Create adds a new schedule for a workspace.
func (h *ScheduleHandler) Create(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	var body createScheduleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cron_expr and prompt are required"})
		return
	}

	if body.Timezone == "" {
		body.Timezone = "UTC"
	}

	// Validate timezone
	if _, err := time.LoadLocation(body.Timezone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timezone: " + body.Timezone})
		return
	}

	// Validate and compute next run
	nextRun, err := scheduler.ComputeNextRun(body.CronExpr, body.Timezone, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	var id string
	// source='runtime' marks this row as user-created (Canvas/API). The
	// org/import path inserts with source='template' and only refreshes
	// template-source rows on re-import (issue #24), so runtime rows survive.
	err = db.DB.QueryRowContext(ctx, `
		INSERT INTO workspace_schedules (workspace_id, name, cron_expr, timezone, prompt, enabled, next_run_at, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'runtime')
		RETURNING id
	`, workspaceID, body.Name, body.CronExpr, body.Timezone, body.Prompt, enabled, nextRun).Scan(&id)
	if err != nil {
		log.Printf("Schedules.Create: insert error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create schedule"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          id,
		"status":      "created",
		"next_run_at": nextRun,
	})
}

type updateScheduleRequest struct {
	Name     *string `json:"name"`
	CronExpr *string `json:"cron_expr"`
	Timezone *string `json:"timezone"`
	Prompt   *string `json:"prompt"`
	Enabled  *bool   `json:"enabled"`
}

// Update modifies a schedule. Uses a fixed UPDATE with COALESCE so only
// provided fields are changed — no dynamic SQL construction.
func (h *ScheduleHandler) Update(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	workspaceID := c.Param("id") // #113: bind to owning workspace to prevent IDOR
	ctx := c.Request.Context()

	var body updateScheduleRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// If cron_expr or timezone changed, revalidate and recompute next_run
	var nextRunAt *time.Time
	if body.CronExpr != nil || body.Timezone != nil {
		var currentCron, currentTZ string
		err := db.DB.QueryRowContext(ctx,
			`SELECT cron_expr, timezone FROM workspace_schedules WHERE id = $1 AND workspace_id = $2`,
			scheduleID, workspaceID,
		).Scan(&currentCron, &currentTZ)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}
		cronExpr := currentCron
		if body.CronExpr != nil {
			cronExpr = *body.CronExpr
		}
		tz := currentTZ
		if body.Timezone != nil {
			tz = *body.Timezone
		}
		if _, err := time.LoadLocation(tz); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timezone: " + tz})
			return
		}
		nextRun, err := scheduler.ComputeNextRun(cronExpr, tz, time.Now())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		nextRunAt = &nextRun
	}

	result, err := db.DB.ExecContext(ctx, `
		UPDATE workspace_schedules SET
			name      = COALESCE($2, name),
			cron_expr = COALESCE($3, cron_expr),
			timezone  = COALESCE($4, timezone),
			prompt    = COALESCE($5, prompt),
			enabled   = COALESCE($6, enabled),
			next_run_at = COALESCE($7, next_run_at),
			updated_at = now()
		WHERE id = $1 AND workspace_id = $8
	`, scheduleID, body.Name, body.CronExpr, body.Timezone, body.Prompt, body.Enabled, nextRunAt, workspaceID)
	if err != nil {
		log.Printf("Schedules.Update: error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update schedule"})
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// Delete removes a schedule.
func (h *ScheduleHandler) Delete(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	workspaceID := c.Param("id") // #113: bind to owning workspace to prevent IDOR
	ctx := c.Request.Context()

	result, err := db.DB.ExecContext(ctx,
		`DELETE FROM workspace_schedules WHERE id = $1 AND workspace_id = $2`,
		scheduleID, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule"})
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// RunNow manually fires a schedule immediately.
func (h *ScheduleHandler) RunNow(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	var prompt string
	err := db.DB.QueryRowContext(ctx,
		`SELECT prompt FROM workspace_schedules WHERE id = $1 AND workspace_id = $2`,
		scheduleID, workspaceID,
	).Scan(&prompt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read schedule"})
		return
	}

	// The actual A2A fire is done by the caller via the proxy — we just
	// return the prompt so the frontend can POST it to /workspaces/:id/a2a.
	// This keeps the handler stateless and avoids circular deps on WorkspaceHandler.
	c.JSON(http.StatusOK, gin.H{
		"status":       "fired",
		"workspace_id": workspaceID,
		"prompt":       prompt,
	})
}

// History returns recent runs for a schedule from activity_logs.
func (h *ScheduleHandler) History(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	// #152: include error_detail in history so UI can show why a run failed.
	// activity_logs.error_detail is populated by scheduler.fireSchedule when
	// the A2A proxy returns non-2xx or the update SQL reports an error.
	rows, err := db.DB.QueryContext(ctx, `
		SELECT created_at, duration_ms, status,
		       COALESCE(error_detail, '') as error_detail,
		       COALESCE(request_body::text, '{}') as request_body
		FROM activity_logs
		WHERE workspace_id = $1
		  AND activity_type = 'cron_run'
		  AND request_body->>'schedule_id' = $2
		ORDER BY created_at DESC
		LIMIT 20
	`, workspaceID, scheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query history"})
		return
	}
	defer rows.Close()

	type historyEntry struct {
		Timestamp   time.Time       `json:"timestamp"`
		DurationMs  *int            `json:"duration_ms"`
		Status      *string         `json:"status"`
		ErrorDetail string          `json:"error_detail"`
		Request     json.RawMessage `json:"request"`
	}

	entries := make([]historyEntry, 0)
	for rows.Next() {
		var e historyEntry
		var reqStr string
		if err := rows.Scan(&e.Timestamp, &e.DurationMs, &e.Status, &e.ErrorDetail, &reqStr); err != nil {
			continue
		}
		e.Request = json.RawMessage(reqStr)
		entries = append(entries, e)
	}

	c.JSON(http.StatusOK, entries)
}

