package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

// ==================== GET /workspaces/:id/traces ====================

func TestTracesList_NoLangfuseConfig(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewTracesHandler()

	// Ensure Langfuse env vars are not set
	os.Unsetenv("LANGFUSE_HOST")
	os.Unsetenv("LANGFUSE_PUBLIC_KEY")
	os.Unsetenv("LANGFUSE_SECRET_KEY")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-traces"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-traces/traces", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should return empty array when Langfuse is not configured
	var resp []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty list when Langfuse not configured, got %d items", len(resp))
	}
}

func TestTracesList_PartialLangfuseConfig(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewTracesHandler()

	// Set only host, missing keys
	os.Setenv("LANGFUSE_HOST", "http://localhost:3000")
	os.Unsetenv("LANGFUSE_PUBLIC_KEY")
	os.Unsetenv("LANGFUSE_SECRET_KEY")
	defer func() {
		os.Unsetenv("LANGFUSE_HOST")
	}()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-traces-partial"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-traces-partial/traces", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 0 {
		t.Errorf("expected empty list with partial config, got %d items", len(resp))
	}
}

func TestTracesList_LangfuseUnreachable(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewTracesHandler()

	// Set all env vars but point to unreachable host
	os.Setenv("LANGFUSE_HOST", "http://localhost:99999")
	os.Setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
	os.Setenv("LANGFUSE_SECRET_KEY", "sk-test")
	defer func() {
		os.Unsetenv("LANGFUSE_HOST")
		os.Unsetenv("LANGFUSE_PUBLIC_KEY")
		os.Unsetenv("LANGFUSE_SECRET_KEY")
	}()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-traces-down"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-traces-down/traces", nil)

	handler.List(c)

	// Should gracefully return empty when Langfuse is unreachable
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 0 {
		t.Errorf("expected empty list when Langfuse unreachable, got %d items", len(resp))
	}
}
