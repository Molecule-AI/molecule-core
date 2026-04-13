package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Workspace struct {
	ID                 string          `json:"id" db:"id"`
	Name               string          `json:"name" db:"name"`
	Role               sql.NullString  `json:"role" db:"role"`
	Tier               int             `json:"tier" db:"tier"`
	AwarenessNamespace sql.NullString  `json:"awareness_namespace" db:"awareness_namespace"`
	Status             string          `json:"status" db:"status"`
	SourceBundleID     sql.NullString  `json:"source_bundle_id" db:"source_bundle_id"`
	AgentCard          json.RawMessage `json:"agent_card" db:"agent_card"`
	URL                sql.NullString  `json:"url" db:"url"`
	ParentID           *string         `json:"parent_id" db:"parent_id"`
	ForwardedTo        *string         `json:"forwarded_to" db:"forwarded_to"`
	LastHeartbeatAt    *time.Time      `json:"last_heartbeat_at" db:"last_heartbeat_at"`
	LastErrorRate      float64         `json:"last_error_rate" db:"last_error_rate"`
	LastSampleError    sql.NullString  `json:"last_sample_error" db:"last_sample_error"`
	ActiveTasks        int             `json:"active_tasks" db:"active_tasks"`
	UptimeSeconds      int             `json:"uptime_seconds" db:"uptime_seconds"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
	// Canvas layout fields (from JOIN)
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Collapsed bool    `json:"collapsed"`
}

type RegisterPayload struct {
	ID        string          `json:"id" binding:"required"`
	URL       string          `json:"url" binding:"required"`
	AgentCard json.RawMessage `json:"agent_card" binding:"required"`
}

type HeartbeatPayload struct {
	WorkspaceID   string  `json:"workspace_id" binding:"required"`
	ErrorRate     float64 `json:"error_rate"`
	SampleError   string  `json:"sample_error"`
	ActiveTasks   int     `json:"active_tasks"`
	UptimeSeconds int     `json:"uptime_seconds"`
	CurrentTask   string  `json:"current_task"`
}

type UpdateCardPayload struct {
	WorkspaceID string          `json:"workspace_id" binding:"required"`
	AgentCard   json.RawMessage `json:"agent_card" binding:"required"`
}

type CreateWorkspacePayload struct {
	Name     string  `json:"name" binding:"required"`
	Role     string  `json:"role"`
	Template string  `json:"template"` // workspace-configs-templates folder name
	Tier     int     `json:"tier"`
	Model    string  `json:"model"`
	Runtime      string  `json:"runtime"`       // "langgraph" (default), "claude-code", etc.
	External     bool    `json:"external"`      // true = no Docker container, just a registered URL
	URL          string  `json:"url"`           // for external workspaces: the A2A endpoint URL
	WorkspaceDir    string  `json:"workspace_dir"`    // host path to mount as /workspace (empty = isolated volume)
	WorkspaceAccess string  `json:"workspace_access"` // "none" (default), "read_only", or "read_write" — see #65
	ParentID        *string `json:"parent_id"`
	Canvas   struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"canvas"`
}

type CheckAccessPayload struct {
	CallerID string `json:"caller_id" binding:"required"`
	TargetID string `json:"target_id" binding:"required"`
}
