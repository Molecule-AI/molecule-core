package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// urlParse is a tiny wrapper so table-driven tests can keep their lines short.
func urlParse(s string) (*url.URL, error) { return url.Parse(s) }

// expectWorkspaceURLLookup programs the sqlmock to answer the SELECT that
// TranscriptHandler.Get issues for `agent_card->>'url'`. Tests call this
// instead of inserting real rows (we use sqlmock — there's no DB).
//
// Returns the workspace ID as the handler's :id path param.
func expectWorkspaceURLLookup(mock sqlmock.Sqlmock, agentURL string) string {
	id := "11111111-2222-3333-4444-555555555555"
	mock.ExpectQuery("SELECT agent_card->>'url' FROM workspaces WHERE id = \\$1").
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"url"}).AddRow(agentURL))
	return id
}

// ==================== GET /workspaces/:id/transcript ====================

func TestTranscript_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	mock.ExpectQuery("SELECT agent_card->>'url' FROM workspaces WHERE id = \\$1").
		WithArgs("00000000-0000-0000-0000-000000000000").
		WillReturnError(sql.ErrNoRows)

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
	mock := setupTestDB(t)
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

	wsID := expectWorkspaceURLLookup(mock,stub.URL)

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

func TestTranscript_ProxyPropagatesAllowlistedQueryParams(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	gotQuery := ""
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{}`))
	}))
	defer stub.Close()

	wsID := expectWorkspaceURLLookup(mock,stub.URL)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: wsID}}
	c.Request = httptest.NewRequest("GET", "/workspaces/"+wsID+"/transcript?since=42&limit=7&secret=leak&cmd=rm", nil)
	h.Get(c)
	// url.Values.Encode() sorts alphabetically — limit before since.
	// Crucially: secret + cmd are dropped (not in the allowlist).
	if gotQuery != "limit=7&since=42" {
		t.Errorf("expected only allowlisted since/limit forwarded, got %q", gotQuery)
	}
}

// SSRF regression tests — see issue #272. agent_card->>'url' is attacker-
// writable via /registry/register so validateWorkspaceURL must reject
// link-local / cloud-metadata / non-http(s) targets before the outbound
// HTTP call fires.

func TestTranscript_RejectsCloudMetadataIP(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	wsID := expectWorkspaceURLLookup(mock,"http://169.254.169.254/")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: wsID}}
	c.Request = httptest.NewRequest("GET", "/workspaces/"+wsID+"/transcript", nil)
	h.Get(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for IMDS target, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTranscript_RejectsNonHTTPScheme(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	wsID := expectWorkspaceURLLookup(mock,"file:///etc/passwd")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: wsID}}
	c.Request = httptest.NewRequest("GET", "/workspaces/"+wsID+"/transcript", nil)
	h.Get(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for file:// scheme, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTranscript_RejectsMetadataHostname(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	wsID := expectWorkspaceURLLookup(mock,"http://metadata.google.internal/computeMetadata/v1/")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: wsID}}
	c.Request = httptest.NewRequest("GET", "/workspaces/"+wsID+"/transcript", nil)
	h.Get(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for metadata hostname, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTranscript_RejectsLinkLocalIPv6(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	wsID := expectWorkspaceURLLookup(mock,"http://[fe80::1]/")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: wsID}}
	c.Request = httptest.NewRequest("GET", "/workspaces/"+wsID+"/transcript", nil)
	h.Get(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for link-local IPv6, got %d: %s", w.Code, w.Body.String())
	}
}

// validateWorkspaceURL unit tests — pure function, no DB/Redis needed.
func TestValidateWorkspaceURL(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"http localhost allowed (dev)", "http://127.0.0.1:8000", false},
		{"https public allowed", "https://agent.example.com", false},
		{"docker internal allowed", "http://host.docker.internal:8000", false},
		{"IMDS IP rejected", "http://169.254.169.254", true},
		{"GCP metadata hostname rejected", "http://metadata.google.internal", true},
		{"Azure metadata rejected", "http://metadata.azure.com", true},
		{"file scheme rejected", "file:///etc/passwd", true},
		{"gopher rejected", "gopher://internal:70/", true},
		{"IPv6 link-local rejected", "http://[fe80::1]", true},
		{"IPv4 link-local multicast rejected", "http://224.0.0.1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, parseErr := urlParse(tc.raw)
			if parseErr != nil && !tc.wantErr {
				t.Fatalf("parse error: %v", parseErr)
			}
			if parseErr != nil {
				return // unparseable URLs are rejected upstream; not this function's job
			}
			err := validateWorkspaceURL(u)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %q, got nil", tc.raw)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected OK for %q, got %v", tc.raw, err)
			}
		})
	}
}

func TestTranscript_UnreachableWorkspaceReturns502(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	h := NewTranscriptHandler()

	wsID := expectWorkspaceURLLookup(mock,"http://127.0.0.1:1") // refused

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: wsID}}
	c.Request = httptest.NewRequest("GET", "/workspaces/"+wsID+"/transcript", nil)
	h.Get(c)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}
