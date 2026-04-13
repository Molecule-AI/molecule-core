package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/channels"
	"github.com/gin-gonic/gin"
)

type stubProxy struct {
	statusCode int
	respBody   []byte
	err        error
}

func (s *stubProxy) ProxyA2ARequest(ctx context.Context, workspaceID string, body []byte, callerID string, logActivity bool) (int, []byte, error) {
	return s.statusCode, s.respBody, s.err
}

type stubBroadcaster struct{}

func (s *stubBroadcaster) RecordAndBroadcast(ctx context.Context, eventType, workspaceID string, data interface{}) error {
	return nil
}

func newTestChannelManager() *channels.Manager {
	return channels.NewManager(&stubProxy{statusCode: 200}, &stubBroadcaster{})
}

// ==================== ListAdapters ====================

func TestChannelHandler_ListAdapters(t *testing.T) {
	handler := NewChannelHandler(newTestChannelManager())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/channels/adapters", nil)

	handler.ListAdapters(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result []map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) == 0 {
		t.Error("expected at least 1 adapter")
	}
	found := false
	for _, a := range result {
		if a["type"] == "telegram" {
			found = true
		}
	}
	if !found {
		t.Error("telegram not in adapter list")
	}
}

// ==================== List ====================

func TestChannelHandler_List(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewChannelHandler(newTestChannelManager())

	rows := sqlmock.NewRows([]string{
		"id", "workspace_id", "channel_type", "channel_config", "enabled",
		"allowed_users", "last_message_at", "message_count", "created_at", "updated_at",
	}).AddRow(
		"ch-1", "ws-1", "telegram",
		[]byte(`{"bot_token":"123:ABCDEFGHIJ","chat_id":"-100"}`),
		true, []byte(`["user-1"]`), nil, 5, nil, nil,
	)
	mock.ExpectQuery("SELECT .* FROM workspace_channels WHERE workspace_id").
		WithArgs("ws-1").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/workspaces/ws-1/channels", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	handler.List(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(result))
	}

	// Verify bot_token is masked
	config := result[0]["config"].(map[string]interface{})
	token := config["bot_token"].(string)
	if token == "123:ABCDEFGHIJ" {
		t.Error("bot_token should be masked in list response")
	}
	if token != "123:...GHIJ" {
		t.Errorf("expected masked token '123:...GHIJ', got %q", token)
	}
}

// ==================== Create ====================

func TestChannelHandler_Create_Success(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewChannelHandler(newTestChannelManager())

	mock.ExpectQuery("INSERT INTO workspace_channels").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("new-ch-id"))
	// Reload query
	mock.ExpectQuery("SELECT .* FROM workspace_channels").
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id", "channel_type", "channel_config", "enabled", "allowed_users"}))

	body, _ := json.Marshal(map[string]interface{}{
		"channel_type":  "telegram",
		"config":        map[string]interface{}{"bot_token": "123456789:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "chat_id": "-100"},
		"allowed_users": []string{"user-1"},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/workspaces/ws-1/channels", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	handler.Create(c)

	if w.Code != 201 {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["id"] != "new-ch-id" {
		t.Errorf("expected id 'new-ch-id', got %v", result["id"])
	}
}

func TestChannelHandler_Create_MissingType(t *testing.T) {
	handler := NewChannelHandler(newTestChannelManager())

	body, _ := json.Marshal(map[string]interface{}{
		"config": map[string]interface{}{"bot_token": "123"},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/workspaces/ws-1/channels", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	handler.Create(c)

	if w.Code != 400 {
		t.Errorf("expected 400 for missing channel_type, got %d", w.Code)
	}
}

func TestChannelHandler_Create_UnsupportedType(t *testing.T) {
	handler := NewChannelHandler(newTestChannelManager())

	body, _ := json.Marshal(map[string]interface{}{
		"channel_type": "whatsapp",
		"config":       map[string]interface{}{},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/workspaces/ws-1/channels", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	handler.Create(c)

	if w.Code != 400 {
		t.Errorf("expected 400 for unsupported type, got %d", w.Code)
	}
}

func TestChannelHandler_Create_InvalidConfig(t *testing.T) {
	handler := NewChannelHandler(newTestChannelManager())

	body, _ := json.Marshal(map[string]interface{}{
		"channel_type": "telegram",
		"config":       map[string]interface{}{}, // missing bot_token + chat_id
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/workspaces/ws-1/channels", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}

	handler.Create(c)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid config, got %d", w.Code)
	}
}

// ==================== Update ====================

func TestChannelHandler_Update_Success(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewChannelHandler(newTestChannelManager())

	mock.ExpectExec("UPDATE workspace_channels").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT .* FROM workspace_channels").
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id", "channel_type", "channel_config", "enabled", "allowed_users"}))

	enabled := false
	body, _ := json.Marshal(map[string]interface{}{
		"enabled": enabled,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PATCH", "/workspaces/ws-1/channels/ch-1", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "channelId", Value: "ch-1"}}

	handler.Update(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChannelHandler_Update_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewChannelHandler(newTestChannelManager())

	mock.ExpectExec("UPDATE workspace_channels").
		WillReturnResult(sqlmock.NewResult(0, 0))

	body, _ := json.Marshal(map[string]interface{}{"enabled": false})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PATCH", "/workspaces/ws-1/channels/ch-999", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "channelId", Value: "ch-999"}}

	handler.Update(c)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ==================== Delete ====================

func TestChannelHandler_Delete_Success(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewChannelHandler(newTestChannelManager())

	mock.ExpectExec("DELETE FROM workspace_channels").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT .* FROM workspace_channels").
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id", "channel_type", "channel_config", "enabled", "allowed_users"}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/workspaces/ws-1/channels/ch-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "channelId", Value: "ch-1"}}

	handler.Delete(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestChannelHandler_Delete_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewChannelHandler(newTestChannelManager())

	mock.ExpectExec("DELETE FROM workspace_channels").
		WillReturnResult(sqlmock.NewResult(0, 0))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/workspaces/ws-1/channels/ch-999", nil)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "channelId", Value: "ch-999"}}

	handler.Delete(c)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ==================== Send ====================

func TestChannelHandler_Send_EmptyText(t *testing.T) {
	handler := NewChannelHandler(newTestChannelManager())

	body, _ := json.Marshal(map[string]interface{}{"text": ""})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/workspaces/ws-1/channels/ch-1/send", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "channelId", Value: "ch-1"}}

	handler.Send(c)

	if w.Code != 400 {
		t.Errorf("expected 400 for empty text, got %d", w.Code)
	}
}

// ==================== Webhook ====================

func TestChannelHandler_Webhook_UnknownType(t *testing.T) {
	handler := NewChannelHandler(newTestChannelManager())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/webhooks/whatsapp", nil)
	c.Params = gin.Params{{Key: "type", Value: "whatsapp"}}

	handler.Webhook(c)

	if w.Code != 404 {
		t.Errorf("expected 404 for unknown type, got %d", w.Code)
	}
}

// ==================== Discover ====================

func TestChannelHandler_Discover_MissingToken(t *testing.T) {
	handler := NewChannelHandler(newTestChannelManager())

	body, _ := json.Marshal(map[string]interface{}{"channel_type": "telegram"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/channels/discover", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Discover(c)

	if w.Code != 400 {
		t.Errorf("expected 400 for missing token, got %d", w.Code)
	}
}

func TestChannelHandler_Discover_UnsupportedType(t *testing.T) {
	handler := NewChannelHandler(newTestChannelManager())

	body, _ := json.Marshal(map[string]interface{}{
		"channel_type": "whatsapp",
		"bot_token":    "fake",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/channels/discover", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Discover(c)

	if w.Code != 400 {
		t.Errorf("expected 400 for unsupported type, got %d", w.Code)
	}
}

func TestChannelHandler_Discover_InvalidBotToken(t *testing.T) {
	handler := NewChannelHandler(newTestChannelManager())

	body, _ := json.Marshal(map[string]interface{}{
		"channel_type": "telegram",
		"bot_token":    "clearly-not-a-real-token",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/channels/discover", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Discover(c)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid token, got %d", w.Code)
	}

	// Verify error is user-friendly (not a raw tgbotapi error)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	if errMsg == "" {
		t.Error("expected error field in response")
	}
}

// ==================== System Caller Prefix ====================

func TestSystemCallerPrefix_ChannelIncluded(t *testing.T) {
	if !isSystemCaller("channel:telegram") {
		t.Error("channel:telegram should be recognized as system caller")
	}
	if !isSystemCaller("channel:slack") {
		t.Error("channel:slack should be recognized as system caller")
	}
	if isSystemCaller("user:someone") {
		t.Error("user:someone should NOT be a system caller")
	}
}
