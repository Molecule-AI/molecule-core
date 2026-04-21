package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ---------- Export: missing workspace ID → 404 ----------

func TestBundleExport_MissingID(t *testing.T) {
	// BundleHandler requires Docker + provisioner — both nil here.
	// Export should fail gracefully (no Docker → bundle.Export returns error).
	h := &BundleHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/bundles/export/nonexistent", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}

	h.Export(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if body["error"] != "bundle not found" {
		t.Errorf("expected 'bundle not found' error, got %q", body["error"])
	}
}

// ---------- Import: invalid JSON → 400 ----------

func TestBundleImport_InvalidJSON(t *testing.T) {
	h := &BundleHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bundles/import", strings.NewReader(`not json`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if body["error"] != "invalid bundle" {
		t.Errorf("expected 'invalid bundle' error, got %q", body["error"])
	}
}

// ---------- Import: empty JSON body → 400 ----------

func TestBundleImport_EmptyBody(t *testing.T) {
	h := &BundleHandler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bundles/import", strings.NewReader(``))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
