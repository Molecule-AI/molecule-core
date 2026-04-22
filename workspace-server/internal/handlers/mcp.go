package handlers

// Package handlers — MCP bridge for opencode integration (#800, #809, #810).
//
// Exposes the same 8 A2A tools as workspace/a2a_mcp_server.py but
// served directly from the platform over HTTP so CLI runtimes running
// OUTSIDE workspace containers (opencode, Claude Code on the developer's
// machine) can participate in the A2A mesh.
//
// Routes (registered under wsAuth — bearer token binds to :id):
//
//	GET  /workspaces/:id/mcp/stream  — SSE transport (MCP 2024-11-05 compat)
//	POST /workspaces/:id/mcp         — Streamable HTTP transport (primary)
//
// Security conditions satisfied:
//   C1: WorkspaceAuth middleware rejects requests without a valid bearer token.
//   C2: MCPRateLimiter (120 req/min/token) middleware applied in router.go.
//   C3: commit_memory / recall_memory with scope=GLOBAL return a permission
//       error; send_message_to_user is excluded from tools/list unless
//       MOLECULE_MCP_ALLOW_SEND_MESSAGE=true.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/events"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/registry"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// mcpProtocolVersion is the MCP spec version this server implements.
const mcpProtocolVersion = "2024-11-05"

// mcpCallTimeout is the maximum time delegate_task waits for a workspace response.
const mcpCallTimeout = 30 * time.Second

// mcpAsyncCallTimeout is the fire-and-forget A2A call timeout for delegate_task_async.
const mcpAsyncCallTimeout = 8 * time.Second

// ─────────────────────────────────────────────────────────────────────────────
// JSON-RPC 2.0 types
// ─────────────────────────────────────────────────────────────────────────────

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      interface{}  `json:"id"`
	Result  interface{}  `json:"result,omitempty"`
	Error   *mcpRPCError `json:"error,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// mcpTool is a tool descriptor returned in tools/list responses.
type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Handler
// ─────────────────────────────────────────────────────────────────────────────

// MCPHandler serves the MCP bridge endpoints for the workspace identified by :id.
type MCPHandler struct {
	database    *sql.DB
	broadcaster *events.Broadcaster
}

// NewMCPHandler wires the handler to db and broadcaster.
// Pass db.DB and the platform broadcaster at router-setup time.
func NewMCPHandler(database *sql.DB, broadcaster *events.Broadcaster) *MCPHandler {
	return &MCPHandler{database: database, broadcaster: broadcaster}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tool definitions (mirrors workspace/a2a_mcp_server.py TOOLS list)
// ─────────────────────────────────────────────────────────────────────────────

var mcpAllTools = []mcpTool{
	{
		Name:        "delegate_task",
		Description: "Delegate a task to another workspace via A2A protocol and WAIT for the response. Use for quick tasks. The target must be a peer (sibling or parent/child). Use list_peers to find available targets.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id": map[string]interface{}{
					"type":        "string",
					"description": "Target workspace ID (from list_peers)",
				},
				"task": map[string]interface{}{
					"type":        "string",
					"description": "The task description to send to the target workspace",
				},
			},
			"required": []string{"workspace_id", "task"},
		},
	},
	{
		Name:        "delegate_task_async",
		Description: "Send a task to another workspace with a short timeout (fire-and-forget). Returns immediately with a task_id — use check_task_status to poll for results.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id": map[string]interface{}{
					"type":        "string",
					"description": "Target workspace ID (from list_peers)",
				},
				"task": map[string]interface{}{
					"type":        "string",
					"description": "The task description to send to the target workspace",
				},
			},
			"required": []string{"workspace_id", "task"},
		},
	},
	{
		Name:        "check_task_status",
		Description: "Check the status of a previously submitted async task. Returns status (dispatched/success/failed) and result when available.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"workspace_id": map[string]interface{}{
					"type":        "string",
					"description": "The workspace ID the task was sent to",
				},
				"task_id": map[string]interface{}{
					"type":        "string",
					"description": "The task_id returned by delegate_task_async",
				},
			},
			"required": []string{"workspace_id", "task_id"},
		},
	},
	{
		Name:        "list_peers",
		Description: "List all workspaces this agent can communicate with (siblings and parent/children). Returns name, ID, status, and role for each peer.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "get_workspace_info",
		Description: "Get this workspace's own info — ID, name, role, tier, parent, status.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "send_message_to_user",
		Description: "Send a message directly to the user's canvas chat — pushed instantly via WebSocket. Use this to acknowledge tasks, send progress updates, or deliver follow-up results.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "The message to send to the user",
				},
			},
			"required": []string{"message"},
		},
	},
	{
		Name:        "commit_memory",
		Description: "Save important information to persistent memory. Scope LOCAL (this workspace only) and TEAM (parent + siblings) are supported. GLOBAL scope is not available via the MCP bridge.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The information to remember",
				},
				"scope": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"LOCAL", "TEAM"},
					"description": "Memory scope (LOCAL or TEAM — GLOBAL is blocked on the MCP bridge)",
				},
			},
			"required": []string{"content"},
		},
	},
	{
		Name:        "recall_memory",
		Description: "Search persistent memory for previously saved information. Returns all matching memories. GLOBAL scope is not available via the MCP bridge.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (empty returns all memories)",
				},
				"scope": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"LOCAL", "TEAM", ""},
					"description": "Filter by scope (empty returns LOCAL + TEAM; GLOBAL is blocked)",
				},
			},
		},
	},
}

// mcpToolList returns the filtered tool list for this MCP bridge.
// C3: send_message_to_user is excluded unless MOLECULE_MCP_ALLOW_SEND_MESSAGE=true.
func mcpToolList() []mcpTool {
	allowSend := os.Getenv("MOLECULE_MCP_ALLOW_SEND_MESSAGE") == "true"
	var out []mcpTool
	for _, t := range mcpAllTools {
		if t.Name == "send_message_to_user" && !allowSend {
			continue
		}
		out = append(out, t)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// HTTP handlers
// ─────────────────────────────────────────────────────────────────────────────

// Call handles POST /workspaces/:id/mcp — Streamable HTTP transport.
//
// Accepts a JSON-RPC 2.0 request and returns a JSON-RPC 2.0 response.
// WorkspaceAuth on the wsAuth group ensures the bearer token is valid for :id
// before this handler runs.
func (h *MCPHandler) Call(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	var req mcpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, mcpResponse{
			JSONRPC: "2.0",
			Error:   &mcpRPCError{Code: -32700, Message: "parse error: " + err.Error()},
		})
		return
	}

	resp := h.dispatchRPC(ctx, workspaceID, req)
	c.JSON(http.StatusOK, resp)
}

// Stream handles GET /workspaces/:id/mcp/stream — SSE transport (backwards compat).
//
// Implements the MCP 2024-11-05 SSE transport:
//  1. Sends an `endpoint` event pointing to the POST endpoint.
//  2. Keeps the connection alive with periodic ping comments.
//
// Clients should POST JSON-RPC requests to the endpoint URL returned in the
// event. The Streamable HTTP POST endpoint is the primary transport for new
// integrations.
func (h *MCPHandler) Stream(c *gin.Context) {
	workspaceID := c.Param("id")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// MCP 2024-11-05 SSE transport: the first event must be "endpoint" with
	// the URL clients should use for JSON-RPC POSTs.
	endpointURL := "/workspaces/" + workspaceID + "/mcp"
	fmt.Fprintf(c.Writer, "event: endpoint\ndata: %s\n\n", endpointURL)
	flusher.Flush()

	ctx := c.Request.Context()
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			fmt.Fprintf(c.Writer, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// JSON-RPC dispatch
// ─────────────────────────────────────────────────────────────────────────────

func (h *MCPHandler) dispatchRPC(ctx context.Context, workspaceID string, req mcpRequest) mcpResponse {
	base := mcpResponse{JSONRPC: "2.0", ID: req.ID}

	switch req.Method {
	case "initialize":
		base.Result = map[string]interface{}{
			"protocolVersion": mcpProtocolVersion,
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{"listChanged": false},
			},
			"serverInfo": map[string]string{
				"name":    "molecule-a2a",
				"version": "1.0.0",
			},
		}

	case "notifications/initialized":
		// No response required for notifications — return empty result.
		base.Result = nil

	case "tools/list":
		base.Result = map[string]interface{}{
			"tools": mcpToolList(),
		}

	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			base.Error = &mcpRPCError{Code: -32602, Message: "invalid params: " + err.Error()}
			return base
		}
		text, err := h.dispatch(ctx, workspaceID, params.Name, params.Arguments)
		if err != nil {
			base.Error = &mcpRPCError{Code: -32000, Message: err.Error()}
			return base
		}
		base.Result = map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": text},
			},
		}

	default:
		base.Error = &mcpRPCError{Code: -32601, Message: "method not found: " + req.Method}
	}

	return base
}

// ─────────────────────────────────────────────────────────────────────────────
// Tool dispatch
// ─────────────────────────────────────────────────────────────────────────────

func (h *MCPHandler) dispatch(ctx context.Context, workspaceID, toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "list_peers":
		return h.toolListPeers(ctx, workspaceID)
	case "get_workspace_info":
		return h.toolGetWorkspaceInfo(ctx, workspaceID)
	case "delegate_task":
		return h.toolDelegateTask(ctx, workspaceID, args, mcpCallTimeout)
	case "delegate_task_async":
		return h.toolDelegateTaskAsync(ctx, workspaceID, args)
	case "check_task_status":
		return h.toolCheckTaskStatus(ctx, workspaceID, args)
	case "send_message_to_user":
		return h.toolSendMessageToUser(ctx, workspaceID, args)
	case "commit_memory":
		return h.toolCommitMemory(ctx, workspaceID, args)
	case "recall_memory":
		return h.toolRecallMemory(ctx, workspaceID, args)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tool implementations
// ─────────────────────────────────────────────────────────────────────────────

func (h *MCPHandler) toolListPeers(ctx context.Context, workspaceID string) (string, error) {
	var parentID sql.NullString
	err := h.database.QueryRowContext(ctx,
		`SELECT parent_id FROM workspaces WHERE id = $1`, workspaceID,
	).Scan(&parentID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("workspace not found")
	}
	if err != nil {
		return "", fmt.Errorf("lookup failed: %w", err)
	}

	type peer struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Role   string `json:"role"`
		Status string `json:"status"`
		Tier   int    `json:"tier"`
	}

	var peers []peer

	scanPeers := func(rows *sql.Rows) error {
		defer rows.Close()
		for rows.Next() {
			var p peer
			if err := rows.Scan(&p.ID, &p.Name, &p.Role, &p.Status, &p.Tier); err != nil {
				return err
			}
			peers = append(peers, p)
		}
		return rows.Err()
	}

	const cols = `SELECT w.id, w.name, COALESCE(w.role,''), w.status, w.tier`

	// Siblings
	if parentID.Valid {
		rows, err := h.database.QueryContext(ctx,
			cols+` FROM workspaces w WHERE w.parent_id = $1 AND w.id != $2 AND w.status != 'removed'`,
			parentID.String, workspaceID)
		if err == nil {
			_ = scanPeers(rows)
		}
	} else {
		rows, err := h.database.QueryContext(ctx,
			cols+` FROM workspaces w WHERE w.parent_id IS NULL AND w.id != $1 AND w.status != 'removed'`,
			workspaceID)
		if err == nil {
			_ = scanPeers(rows)
		}
	}

	// Children
	{
		rows, err := h.database.QueryContext(ctx,
			cols+` FROM workspaces w WHERE w.parent_id = $1 AND w.status != 'removed'`,
			workspaceID)
		if err == nil {
			_ = scanPeers(rows)
		}
	}

	// Parent
	if parentID.Valid {
		rows, err := h.database.QueryContext(ctx,
			cols+` FROM workspaces w WHERE w.id = $1 AND w.status != 'removed'`,
			parentID.String)
		if err == nil {
			_ = scanPeers(rows)
		}
	}

	if len(peers) == 0 {
		return "No peers found.", nil
	}

	b, _ := json.MarshalIndent(peers, "", "  ")
	return string(b), nil
}

func (h *MCPHandler) toolGetWorkspaceInfo(ctx context.Context, workspaceID string) (string, error) {
	var id, name, role, status string
	var tier int
	var parentID sql.NullString

	err := h.database.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(role,''), tier, status, parent_id
		FROM workspaces WHERE id = $1
	`, workspaceID).Scan(&id, &name, &role, &tier, &status, &parentID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("workspace not found")
	}
	if err != nil {
		return "", fmt.Errorf("lookup failed: %w", err)
	}

	info := map[string]interface{}{
		"id":     id,
		"name":   name,
		"role":   role,
		"tier":   tier,
		"status": status,
	}
	if parentID.Valid {
		info["parent_id"] = parentID.String
	}
	b, _ := json.MarshalIndent(info, "", "  ")
	return string(b), nil
}

func (h *MCPHandler) toolDelegateTask(ctx context.Context, callerID string, args map[string]interface{}, timeout time.Duration) (string, error) {
	targetID, _ := args["workspace_id"].(string)
	task, _ := args["task"].(string)
	if targetID == "" {
		return "", fmt.Errorf("workspace_id is required")
	}
	if task == "" {
		return "", fmt.Errorf("task is required")
	}

	if !registry.CanCommunicate(callerID, targetID) {
		return "", fmt.Errorf("workspace %s is not authorised to communicate with %s", callerID, targetID)
	}

	agentURL, err := mcpResolveURL(ctx, h.database, targetID)
	if err != nil {
		return "", err
	}
	// SSRF defence: reject private/metadata URLs before making outbound call.
	if err := isSafeURL(agentURL); err != nil {
		return "", fmt.Errorf("invalid workspace URL: %w", err)
	}

	a2aBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      uuid.New().String(),
		"method":  "message/send",
		"params": map[string]interface{}{
			"message": map[string]interface{}{
				"role":      "user",
				"parts":     []map[string]interface{}{{"type": "text", "text": task}},
				"messageId": uuid.New().String(),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to build A2A request: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, "POST", agentURL+"/a2a", bytes.NewReader(a2aBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// X-Workspace-ID identifies this caller to the A2A proxy. The /workspaces/:id/a2a
	// endpoint is intentionally outside WorkspaceAuth (agents do not hold bearer tokens
	// to peer workspaces). Access control is enforced by CanCommunicate above, which
	// already validated callerID → targetID before this request is constructed.
	// callerID was authenticated by WorkspaceAuth on the MCP bridge entry point,
	// so this header reflects a verified caller identity, not a spoofable value.
	httpReq.Header.Set("X-Workspace-ID", callerID)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("A2A call failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return extractA2AText(body), nil
}

func (h *MCPHandler) toolDelegateTaskAsync(ctx context.Context, callerID string, args map[string]interface{}) (string, error) {
	targetID, _ := args["workspace_id"].(string)
	task, _ := args["task"].(string)
	if targetID == "" {
		return "", fmt.Errorf("workspace_id is required")
	}
	if task == "" {
		return "", fmt.Errorf("task is required")
	}

	if !registry.CanCommunicate(callerID, targetID) {
		return "", fmt.Errorf("workspace %s is not authorised to communicate with %s", callerID, targetID)
	}

	taskID := uuid.New().String()

	// Fire and forget in a detached goroutine. Use a background context so
	// the call is not cancelled when the HTTP request completes.
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), mcpAsyncCallTimeout)
		defer cancel()

		agentURL, err := mcpResolveURL(bgCtx, h.database, targetID)
		if err != nil {
			log.Printf("MCPHandler.delegate_task_async: resolve URL for %s: %v", targetID, err)
			return
		}
		// SSRF defence: reject private/metadata URLs before making outbound call.
		if err := isSafeURL(agentURL); err != nil {
			log.Printf("MCPHandler.delegate_task_async: unsafe URL for %s: %v", targetID, err)
			return
		}

		a2aBody, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      taskID,
			"method":  "message/send",
			"params": map[string]interface{}{
				"message": map[string]interface{}{
					"role":      "user",
					"parts":     []map[string]interface{}{{"type": "text", "text": task}},
					"messageId": uuid.New().String(),
				},
			},
		})

		httpReq, err := http.NewRequestWithContext(bgCtx, "POST", agentURL+"/a2a", bytes.NewReader(a2aBody))
		if err != nil {
			log.Printf("MCPHandler.delegate_task_async: create request: %v", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Workspace-ID", callerID)

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			log.Printf("MCPHandler.delegate_task_async: A2A call to %s: %v", targetID, err)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		// Drain response so the connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
	}()

	return fmt.Sprintf(`{"task_id":%q,"status":"dispatched","target_id":%q}`, taskID, targetID), nil
}

func (h *MCPHandler) toolCheckTaskStatus(ctx context.Context, callerID string, args map[string]interface{}) (string, error) {
	targetID, _ := args["workspace_id"].(string)
	taskID, _ := args["task_id"].(string)
	if targetID == "" {
		return "", fmt.Errorf("workspace_id is required")
	}
	if taskID == "" {
		return "", fmt.Errorf("task_id is required")
	}

	var status, errorDetail sql.NullString
	var responseBody []byte

	err := h.database.QueryRowContext(ctx, `
		SELECT status, error_detail, response_body
		FROM activity_logs
		WHERE workspace_id = $1
		  AND target_id = $2
		  AND request_body->>'delegation_id' = $3
		ORDER BY created_at DESC
		LIMIT 1
	`, callerID, targetID, taskID).Scan(&status, &errorDetail, &responseBody)
	if err == sql.ErrNoRows {
		return fmt.Sprintf(`{"task_id":%q,"status":"not_found","note":"task not tracked or not yet dispatched"}`, taskID), nil
	}
	if err != nil {
		return "", fmt.Errorf("status lookup failed: %w", err)
	}

	result := map[string]interface{}{
		"task_id":   taskID,
		"status":    status.String,
		"target_id": targetID,
	}
	if errorDetail.Valid && errorDetail.String != "" {
		result["error"] = errorDetail.String
	}
	if len(responseBody) > 0 {
		result["result"] = extractA2AText(responseBody)
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func (h *MCPHandler) toolSendMessageToUser(ctx context.Context, workspaceID string, args map[string]interface{}) (string, error) {
	message, _ := args["message"].(string)
	if message == "" {
		return "", fmt.Errorf("message is required")
	}

	// Check send_message_to_user is enabled (C3).
	if os.Getenv("MOLECULE_MCP_ALLOW_SEND_MESSAGE") != "true" {
		return "", fmt.Errorf("send_message_to_user is not enabled on this MCP bridge (set MOLECULE_MCP_ALLOW_SEND_MESSAGE=true)")
	}

	var wsName string
	err := h.database.QueryRowContext(ctx,
		`SELECT name FROM workspaces WHERE id = $1 AND status != 'removed'`, workspaceID,
	).Scan(&wsName)
	if err != nil {
		return "", fmt.Errorf("workspace not found")
	}

	h.broadcaster.BroadcastOnly(workspaceID, "AGENT_MESSAGE", map[string]interface{}{
		"message":      message,
		"workspace_id": workspaceID,
		"name":         wsName,
	})

	return "Message sent.", nil
}


func (h *MCPHandler) toolCommitMemory(ctx context.Context, workspaceID string, args map[string]interface{}) (string, error) {
	content, _ := args["content"].(string)
	scope, _ := args["scope"].(string)
	if content == "" {
		return "", fmt.Errorf("content is required")
	}
	if scope == "" {
		scope = "LOCAL"
	}

	// C3: GLOBAL scope is blocked on the MCP bridge.
	if scope == "GLOBAL" {
		return "", fmt.Errorf("GLOBAL scope is not permitted via the MCP bridge — use LOCAL or TEAM")
	}
	if scope != "LOCAL" && scope != "TEAM" {
		return "", fmt.Errorf("scope must be LOCAL or TEAM")
	}

	memoryID := uuid.New().String()
	// SAFE-T1201 (#838): scrub known credential patterns before persistence so
	// plain-text API keys pulled in via tool responses can't land in the
	// memories table (and leak into shared TEAM scope). Reuses redactSecrets
	// already shipped for the HTTP path in PR #881 — this was the MCP-bridge
	// sibling the original fix missed. Runs on every write regardless of scope.
	content, _ = redactSecrets(workspaceID, content)
	_, err := h.database.ExecContext(ctx, `
		INSERT INTO agent_memories (id, workspace_id, content, scope, namespace)
		VALUES ($1, $2, $3, $4, $5)
	`, memoryID, workspaceID, content, scope, workspaceID)
	if err != nil {
		log.Printf("MCPHandler.commit_memory workspace=%s: %v", workspaceID, err)
		return "", fmt.Errorf("failed to save memory")
	}

	// GH#1490: surface commit_memory MCP calls in Canvas Agent Comms tab.
	// LogActivity is in the same handlers package — no extra import needed.
	LogActivity(ctx, h.broadcaster, ActivityParams{
		WorkspaceID:  workspaceID,
		ActivityType: "memory_write",
		Summary:      nilIfEmpty(fmt.Sprintf("Memory committed [%s] id=%s", scope, memoryID[:8])),
		Status:       "ok",
	})

	return fmt.Sprintf(`{"id":%q,"scope":%q}`, memoryID, scope), nil
}

func (h *MCPHandler) toolRecallMemory(ctx context.Context, workspaceID string, args map[string]interface{}) (string, error) {
	query, _ := args["query"].(string)
	scope, _ := args["scope"].(string)

	// C3: GLOBAL scope is blocked on the MCP bridge.
	if scope == "GLOBAL" {
		return "", fmt.Errorf("GLOBAL scope is not permitted via the MCP bridge — use LOCAL, TEAM, or empty")
	}

	var rows *sql.Rows
	var err error

	switch scope {
	case "LOCAL":
		rows, err = h.database.QueryContext(ctx, `
			SELECT id, content, scope, created_at
			FROM agent_memories
			WHERE workspace_id = $1 AND scope = 'LOCAL'
			  AND ($2 = '' OR content ILIKE '%' || $2 || '%')
			ORDER BY created_at DESC LIMIT 50
		`, workspaceID, query)
	case "TEAM":
		// Team scope: parent + all siblings.
		rows, err = h.database.QueryContext(ctx, `
			SELECT m.id, m.content, m.scope, m.created_at
			FROM agent_memories m
			JOIN workspaces w ON w.id = m.workspace_id
			WHERE m.scope = 'TEAM'
			  AND w.status != 'removed'
			  AND (w.id = $1 OR w.parent_id = (SELECT parent_id FROM workspaces WHERE id = $1 AND parent_id IS NOT NULL))
			  AND ($2 = '' OR m.content ILIKE '%' || $2 || '%')
			ORDER BY m.created_at DESC LIMIT 50
		`, workspaceID, query)
	default:
		// Empty scope → LOCAL only for the MCP bridge (GLOBAL excluded per C3).
		rows, err = h.database.QueryContext(ctx, `
			SELECT id, content, scope, created_at
			FROM agent_memories
			WHERE workspace_id = $1 AND scope IN ('LOCAL', 'TEAM')
			  AND ($2 = '' OR content ILIKE '%' || $2 || '%')
			ORDER BY created_at DESC LIMIT 50
		`, workspaceID, query)
	}
	if err != nil {
		return "", fmt.Errorf("memory search failed: %w", err)
	}
	defer rows.Close()

	type memEntry struct {
		ID        string `json:"id"`
		Content   string `json:"content"`
		Scope     string `json:"scope"`
		CreatedAt string `json:"created_at"`
	}
	var results []memEntry
	for rows.Next() {
		var e memEntry
		if err := rows.Scan(&e.ID, &e.Content, &e.Scope, &e.CreatedAt); err != nil {
			continue
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("memory scan error: %w", err)
	}

	if len(results) == 0 {
		return "No memories found.", nil
	}
	b, _ := json.MarshalIndent(results, "", "  ")
	return string(b), nil
}

// isSafeURL and isPrivateOrMetadataIP live in a2a_proxy.go -- same package,
// shared across MCP + A2A proxy call sites. Keeping a single copy avoids
// drift between the two SSRF gates when one is tightened and the other
// isn't.

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
