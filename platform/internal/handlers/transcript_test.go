package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/gin-gonic/gin"
)

// helper: register a workspace row + return its ID
func seedWorkspace(t *testing.T, agentURL string) string {
	t.Helper()
	id := "11111111-2222-3333-4444-555555555555"
	_, err := db.DB.Exec(
		`INSERT INTO workspaces (id, name, agent_card, status) VALUES ($1, 'transcript-test', $2, 'online')
		 ON CONFLICT (id) DO UPDATE SET agent_card = EXCLUDED.agent_card`,
		id, []byte(`{"url":"`+agentURL+`"}`),
	)
	if err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	return id
}

// ==================== GET /workspaces/:id/transcript ====================

func TestTranscript_WorkspaceNotFound(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "00000000-0000-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/00000000-0000-0000-0000-000000000000/transcript", nil)
	h.Get(c)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTranscript_ProxyForwardsAndReturnsBody(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	// Spin up a fake "workspace" agent that returns a canned transcript
	gotPath := ""
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"runtime":"claude-code","supported":true,"lines":[{"type":"user"}],"cursor":1,"more":false}`))
	}))
	defer stub.Close()

	wsID := seedWorkspace(t, stub.URL)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: wsID}}
	c.Request = httptest.NewRequest("GET", "/workspaces/"+wsID+"/transcript?since=5&limit=20", nil)
	h.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if gotPath != "/transcript" {
		t.Errorf("expected proxy to hit /transcript, got %q", gotPath)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if resp["runtime"] != "claude-code" {
		t.Errorf("expected runtime=claude-code, got %v", resp["runtime"])
	}
	if lines, ok := resp["lines"].([]interface{}); !ok || len(lines) != 1 {
		t.Errorf("expected 1 line, got %v", resp["lines"])
	}
}

func TestTranscript_ProxyPropagatesQueryString(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	gotQuery := ""
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{}`))
	}))
	defer stub.Close()

	wsID := seedWorkspace(t, stub.URL)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: wsID}}
	c.Request = httptest.NewRequest("GET", "/workspaces/"+wsID+"/transcript?since=42&limit=7", nil)
	h.Get(c)
	if gotQuery != "since=42&limit=7" {
		t.Errorf("expected query forwarded, got %q", gotQuery)
	}
}

func TestTranscript_UnreachableWorkspaceReturns502(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	wsID := seedWorkspace(t, "http://127.0.0.1:1") // refused

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: wsID}}
	c.Request = httptest.NewRequest("GET", "/workspaces/"+wsID+"/transcript", nil)
	h.Get(c)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}
