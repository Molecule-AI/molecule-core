package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/events"
	"github.com/gin-gonic/gin"
)

// newMCPHandler is a test helper that constructs an MCPHandler backed by the
// sqlmock DB set up by setupTestDB.
func newMCPHandler(t *testing.T) (*MCPHandler, sqlmock.Sqlmock) {
	t.Helper()
	mock := setupTestDB(t)
	h := NewMCPHandler(db.DB, events.NewBroadcaster(nil))
	return h, mock
}

// errNotFound is sql.ErrNoRows, used to simulate missing-row DB errors.
var errNotFound = sql.ErrNoRows

// contextForTest returns a cancellable context pre-cancelled so that
// streaming handlers (Stream) return immediately in tests.
func contextForTest() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	return ctx, cancel
}

// mcpPost builds a POST /workspaces/:id/mcp request with the given JSON body.
func mcpPost(t *testing.T, h *MCPHandler, workspaceID string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: workspaceID}}
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer(b))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Call(c)
	return w
}

// ─────────────────────────────────────────────────────────────────────────────
// initialize
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_Initialize_ReturnsCapabilities(t *testing.T) {
	h, _ := newMCPHandler(t)

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp mcpResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %T", resp.Result)
	}
	if result["protocolVersion"] != mcpProtocolVersion {
		t.Errorf("protocolVersion: got %v, want %s", result["protocolVersion"], mcpProtocolVersion)
	}
	caps, _ := result["capabilities"].(map[string]interface{})
	if _, ok := caps["tools"]; !ok {
		t.Error("capabilities.tools missing")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// tools/list
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_ToolsList_ExcludesSendMessageByDefault(t *testing.T) {
	_ = os.Unsetenv("MOLECULE_MCP_ALLOW_SEND_MESSAGE")
	h, _ := newMCPHandler(t)

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	result, _ := resp.Result.(map[string]interface{})
	toolsRaw, _ := result["tools"].([]interface{})

	for _, ti := range toolsRaw {
		tool, _ := ti.(map[string]interface{})
		if tool["name"] == "send_message_to_user" {
			t.Error("send_message_to_user should be excluded when MOLECULE_MCP_ALLOW_SEND_MESSAGE is unset")
		}
	}
	if len(toolsRaw) == 0 {
		t.Error("tool list should not be empty")
	}
}

func TestMCPHandler_ToolsList_IncludesSendMessageWhenEnvSet(t *testing.T) {
	t.Setenv("MOLECULE_MCP_ALLOW_SEND_MESSAGE", "true")
	h, _ := newMCPHandler(t)

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/list",
	})

	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	result, _ := resp.Result.(map[string]interface{})
	toolsRaw, _ := result["tools"].([]interface{})

	found := false
	for _, ti := range toolsRaw {
		tool, _ := ti.(map[string]interface{})
		if tool["name"] == "send_message_to_user" {
			found = true
		}
	}
	if !found {
		t.Error("send_message_to_user should be included when MOLECULE_MCP_ALLOW_SEND_MESSAGE=true")
	}
}

func TestMCPHandler_ToolsList_ContainsExpectedTools(t *testing.T) {
	_ = os.Unsetenv("MOLECULE_MCP_ALLOW_SEND_MESSAGE")
	h, _ := newMCPHandler(t)

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/list",
	})

	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	result, _ := resp.Result.(map[string]interface{})
	toolsRaw, _ := result["tools"].([]interface{})

	names := make(map[string]bool)
	for _, ti := range toolsRaw {
		tool, _ := ti.(map[string]interface{})
		names[tool["name"].(string)] = true
	}
	required := []string{"list_peers", "get_workspace_info", "delegate_task", "delegate_task_async", "check_task_status", "commit_memory", "recall_memory"}
	for _, name := range required {
		if !names[name] {
			t.Errorf("tool %q missing from tools/list", name)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// notifications/initialized
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_NotificationsInitialized_Returns200(t *testing.T) {
	h, _ := newMCPHandler(t)

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      nil,
		"method":  "notifications/initialized",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != nil {
		t.Errorf("unexpected error: %+v", resp.Error)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Unknown method
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_UnknownMethod_Returns32601(t *testing.T) {
	h, _ := newMCPHandler(t)

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "not/a/real/method",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with error body, got %d", w.Code)
	}
	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil {
		t.Fatal("expected JSON-RPC error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", resp.Error.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// tools/call — get_workspace_info
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_GetWorkspaceInfo_Success(t *testing.T) {
	h, mock := newMCPHandler(t)

	mock.ExpectQuery("SELECT id, name").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role", "tier", "status", "parent_id"}).
			AddRow("ws-1", "Dev Lead", "developer", 2, "online", nil))

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "get_workspace_info",
			"arguments": map[string]interface{}{},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	result, _ := resp.Result.(map[string]interface{})
	content, _ := result["content"].([]interface{})
	if len(content) == 0 {
		t.Fatal("content is empty")
	}
	item, _ := content[0].(map[string]interface{})
	text, _ := item["text"].(string)
	if text == "" {
		t.Error("tool result text is empty")
	}
	// Verify the JSON contains expected fields.
	var info map[string]interface{}
	if err := json.Unmarshal([]byte(text), &info); err != nil {
		t.Fatalf("tool result is not valid JSON: %v", err)
	}
	if info["id"] != "ws-1" {
		t.Errorf("id: got %v, want ws-1", info["id"])
	}
	if info["name"] != "Dev Lead" {
		t.Errorf("name: got %v, want Dev Lead", info["name"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestMCPHandler_GetWorkspaceInfo_NotFound(t *testing.T) {
	h, mock := newMCPHandler(t)

	mock.ExpectQuery("SELECT id, name").
		WithArgs("ws-missing").
		WillReturnError(errNotFound)

	w := mcpPost(t, h, "ws-missing", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      7,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "get_workspace_info",
			"arguments": map[string]interface{}{},
		},
	})

	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil {
		t.Error("expected JSON-RPC error for missing workspace")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// tools/call — list_peers
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_ListPeers_ReturnsSiblings(t *testing.T) {
	h, mock := newMCPHandler(t)

	// Parent lookup
	mock.ExpectQuery("SELECT parent_id FROM workspaces").
		WithArgs("ws-child").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow("ws-parent"))

	// Siblings query
	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("ws-parent", "ws-child").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role", "status", "tier"}).
			AddRow("ws-sibling", "Research", "researcher", "online", 1))

	// Children query
	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("ws-child").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role", "status", "tier"}))

	// Parent query
	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("ws-parent").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role", "status", "tier"}).
			AddRow("ws-parent", "PM", "manager", "online", 3))

	w := mcpPost(t, h, "ws-child", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      8,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "list_peers",
			"arguments": map[string]interface{}{},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	result, _ := resp.Result.(map[string]interface{})
	content, _ := result["content"].([]interface{})
	item, _ := content[0].(map[string]interface{})
	text, _ := item["text"].(string)
	if !bytes.Contains([]byte(text), []byte("ws-sibling")) {
		t.Errorf("expected sibling ws-sibling in response, got: %s", text)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// tools/call — commit_memory
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_CommitMemory_LocalScope_Success(t *testing.T) {
	h, mock := newMCPHandler(t)

	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs(sqlmock.AnyArg(), "ws-1", "important fact", "LOCAL", "ws-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      9,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "commit_memory",
			"arguments": map[string]interface{}{
				"content": "important fact",
				"scope":   "LOCAL",
			},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestMCPHandler_CommitMemory_GlobalScope_Blocked verifies that C3 is enforced:
// GLOBAL scope is not permitted on the MCP bridge.
func TestMCPHandler_CommitMemory_GlobalScope_Blocked(t *testing.T) {
	h, mock := newMCPHandler(t)
	// No DB expectations — handler must abort before touching the DB.

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      10,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "commit_memory",
			"arguments": map[string]interface{}{
				"content": "secret global memory",
				"scope":   "GLOBAL",
			},
		},
	})

	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil {
		t.Error("expected JSON-RPC error for GLOBAL scope, got nil")
	}
	if resp.Error != nil && !bytes.Contains([]byte(resp.Error.Message), []byte("GLOBAL")) {
		t.Errorf("error message should mention GLOBAL, got: %s", resp.Error.Message)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected DB calls on GLOBAL scope block: %v", err)
	}
}

// TestMCPHandler_CommitMemory_SecretInContent_IsRedactedBeforeInsert verifies
// the SAFE-T1201 (#838) fix on the MCP bridge path. PR #881 closed the HTTP
// handler but missed this one — an agent tool-call carrying plain-text
// credentials must have them scrubbed before the INSERT reaches the DB.
//
// The test asserts via the sqlmock `WithArgs` matcher that the content column
// binds the REDACTED form, not the raw input. sqlmock verifies the exact arg
// values, so a regression (removing the redactSecrets call) would fail with
// "argument mismatch" rather than silently persisting the secret.
func TestMCPHandler_CommitMemory_SecretInContent_IsRedactedBeforeInsert(t *testing.T) {
	h, mock := newMCPHandler(t)

	// Content with three distinct secret patterns covered by redactSecrets:
	//   - env-var assignment (ANTHROPIC_API_KEY=)
	//   - Bearer token
	//   - sk-… prefixed key
	rawContent := "key=ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxxxxxxxx auth=Bearer ghp_yyyyyyyyyyyyy note=sk-proj-zzzzzzzzzzzzzzzzzzzz"

	// Derive what redactSecrets will produce so the sqlmock arg match is
	// exact. This keeps the test brittle-on-purpose: if redactSecrets's
	// output shape changes, this test must be re-derived, which surfaces
	// the change during review.
	expected, changed := redactSecrets("ws-1", rawContent)
	if !changed {
		t.Fatalf("precondition failed — redactSecrets must change the test content; got unchanged %q", expected)
	}
	if bytes.Contains([]byte(expected), []byte("sk-ant-xxxxxxxxxxxxxxxx")) {
		t.Fatalf("precondition failed — redacted content still contains raw secret: %s", expected)
	}

	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs(sqlmock.AnyArg(), "ws-1", expected, "LOCAL", "ws-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      99,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "commit_memory",
			"arguments": map[string]interface{}{
				"content": rawContent,
				"scope":   "LOCAL",
			},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %+v", resp.Error)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock mismatch — content was NOT redacted before insert: %v", err)
	}
}

// TestMCPHandler_CommitMemory_CleanContent_PassesThrough confirms that the
// redactor is a no-op on content with no credentials — a regression where
// redactSecrets corrupted benign content would be a user-visible bug.
func TestMCPHandler_CommitMemory_CleanContent_PassesThrough(t *testing.T) {
	h, mock := newMCPHandler(t)

	cleanContent := "the quick brown fox jumps over the lazy dog — no secrets here"

	// Bind the exact string — no wildcards — so that any transformation
	// (whitespace, case, truncation) would fail the arg match.
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs(sqlmock.AnyArg(), "ws-1", cleanContent, "TEAM", "ws-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      100,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "commit_memory",
			"arguments": map[string]interface{}{
				"content": cleanContent,
				"scope":   "TEAM",
			},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("clean content should pass through unchanged: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// tools/call — recall_memory
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_RecallMemory_GlobalScope_Blocked(t *testing.T) {
	h, mock := newMCPHandler(t)
	// No DB expectations — handler must abort before touching the DB.

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      11,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "recall_memory",
			"arguments": map[string]interface{}{
				"query": "secret",
				"scope": "GLOBAL",
			},
		},
	})

	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil {
		t.Error("expected JSON-RPC error for GLOBAL scope recall, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected DB calls on GLOBAL scope block: %v", err)
	}
}

func TestMCPHandler_RecallMemory_LocalScope_Empty(t *testing.T) {
	h, mock := newMCPHandler(t)

	mock.ExpectQuery("SELECT id, content, scope, created_at").
		WithArgs("ws-1", "").
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "scope", "created_at"}))

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      12,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "recall_memory",
			"arguments": map[string]interface{}{
				"query": "",
				"scope": "LOCAL",
			},
		},
	})

	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	result, _ := resp.Result.(map[string]interface{})
	content, _ := result["content"].([]interface{})
	item, _ := content[0].(map[string]interface{})
	text, _ := item["text"].(string)
	if text != "No memories found." {
		t.Errorf("expected 'No memories found.', got %q", text)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// tools/call — send_message_to_user
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_SendMessageToUser_Blocked_WhenEnvNotSet(t *testing.T) {
	_ = os.Unsetenv("MOLECULE_MCP_ALLOW_SEND_MESSAGE")
	h, mock := newMCPHandler(t)
	// No DB expectations — handler must abort before touching DB.

	w := mcpPost(t, h, "ws-1", map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      13,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "send_message_to_user",
			"arguments": map[string]interface{}{
				"message": "hello",
			},
		},
	})

	var resp mcpResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil {
		t.Error("expected JSON-RPC error when MOLECULE_MCP_ALLOW_SEND_MESSAGE is unset")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected DB calls: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Parse error
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_Call_InvalidJSON_Returns400(t *testing.T) {
	h, _ := newMCPHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Call(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SSE Stream
// ─────────────────────────────────────────────────────────────────────────────

func TestMCPHandler_Stream_SendsEndpointEvent(t *testing.T) {
	h, _ := newMCPHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-stream"}}

	// Use a context that is immediately cancelled so Stream returns quickly.
	ctx, cancel := contextForTest()
	defer cancel()

	c.Request = httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	cancel() // cancel before calling so Stream exits after the first write

	h.Stream(c)

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("event: endpoint")) {
		t.Errorf("SSE stream should contain 'event: endpoint', got: %q", body)
	}
	if !bytes.Contains([]byte(body), []byte("/workspaces/ws-stream/mcp")) {
		t.Errorf("SSE endpoint data should contain the POST URL, got: %q", body)
	}
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type: got %q, want text/event-stream", w.Header().Get("Content-Type"))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// extractA2AText helper
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractA2AText_ArtifactsFormat(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":"x","result":{"artifacts":[{"parts":[{"type":"text","text":"hello from agent"}]}]}}`)
	got := extractA2AText(body)
	if got != "hello from agent" {
		t.Errorf("extractA2AText: got %q, want %q", got, "hello from agent")
	}
}

func TestExtractA2AText_MessageFormat(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":"x","result":{"message":{"role":"assistant","parts":[{"type":"text","text":"agent reply"}]}}}`)
	got := extractA2AText(body)
	if got != "agent reply" {
		t.Errorf("extractA2AText: got %q, want %q", got, "agent reply")
	}
}

func TestExtractA2AText_ErrorFormat(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":"x","error":{"code":-32000,"message":"something went wrong"}}`)
	got := extractA2AText(body)
	if !bytes.Contains([]byte(got), []byte("something went wrong")) {
		t.Errorf("extractA2AText: error message not propagated, got %q", got)
	}
}

func TestExtractA2AText_InvalidJSON_ReturnRaw(t *testing.T) {
	body := []byte(`not json`)
	got := extractA2AText(body)
	if got != "not json" {
		t.Errorf("extractA2AText: expected raw fallback, got %q", got)
	}
}
