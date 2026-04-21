package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ---------- ListSources: returns registered schemes ----------

func TestPluginListSources_ReturnsSchemes(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/plugins/sources", nil)

	h.ListSources(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string][]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	schemes := body["schemes"]
	if len(schemes) == 0 {
		t.Fatal("expected at least one scheme, got empty list")
	}

	// Default handler registers "local" and "github" resolvers
	found := map[string]bool{}
	for _, s := range schemes {
		found[s] = true
	}
	if !found["local"] {
		t.Errorf("expected 'local' scheme, got %v", schemes)
	}
	if !found["github"] {
		t.Errorf("expected 'github' scheme, got %v", schemes)
	}
}

// ---------- ListSources: response shape ----------

func TestPluginListSources_ResponseShape(t *testing.T) {
	h := NewPluginsHandler(t.TempDir(), nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/plugins/sources", nil)

	h.ListSources(c)

	// Verify the response is a JSON object with a "schemes" key
	var raw map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := raw["schemes"]; !ok {
		t.Error("response missing 'schemes' key")
	}
}
