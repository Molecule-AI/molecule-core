package provisioner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNewCPProvisioner_RequiresOrgID — self-hosted deployments don't
// have a MOLECULE_ORG_ID, and the provisioner must refuse to construct
// rather than silently phone home to the prod CP with an empty tenant.
func TestNewCPProvisioner_RequiresOrgID(t *testing.T) {
	t.Setenv("MOLECULE_ORG_ID", "")
	if _, err := NewCPProvisioner(); err == nil {
		t.Error("want error when MOLECULE_ORG_ID is unset, got nil")
	}
}

// TestNewCPProvisioner_FallsBackToProvisionSharedSecret — operators
// may set PROVISION_SHARED_SECRET on both sides of the wire with a
// single value; the tenant accepts that name as a fallback for
// MOLECULE_CP_SHARED_SECRET. The fallback is documented in
// NewCPProvisioner; this test is the regression gate.
func TestNewCPProvisioner_FallsBackToProvisionSharedSecret(t *testing.T) {
	t.Setenv("MOLECULE_ORG_ID", "org-abc")
	t.Setenv("MOLECULE_CP_SHARED_SECRET", "")
	t.Setenv("PROVISION_SHARED_SECRET", "from-fallback")

	p, err := NewCPProvisioner()
	if err != nil {
		t.Fatalf("NewCPProvisioner: %v", err)
	}
	if p.sharedSecret != "from-fallback" {
		t.Errorf("sharedSecret = %q, want %q", p.sharedSecret, "from-fallback")
	}
}

// TestAuthHeaders_NoopWhenBothEmpty — the self-hosted path that
// doesn't gate /cp/workspaces/* must not add stray auth headers
// (bearer-like content would surprise non-bearer intermediaries).
func TestAuthHeaders_NoopWhenBothEmpty(t *testing.T) {
	p := &CPProvisioner{sharedSecret: "", adminToken: ""}
	req := httptest.NewRequest("GET", "http://x/", nil)
	p.authHeaders(req)
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization set to %q with empty secret; want unset", got)
	}
	if got := req.Header.Get("X-Molecule-Admin-Token"); got != "" {
		t.Errorf("X-Molecule-Admin-Token set to %q with empty token; want unset", got)
	}
}

// TestAuthHeaders_SetsBothWhenBothProvided — happy path for SaaS
// tenants. Both the platform-wide shared secret and the per-tenant
// admin_token land on every outbound call.
func TestAuthHeaders_SetsBothWhenBothProvided(t *testing.T) {
	p := &CPProvisioner{sharedSecret: "the-secret", adminToken: "tok-abc"}
	req := httptest.NewRequest("GET", "http://x/", nil)
	p.authHeaders(req)
	if got := req.Header.Get("Authorization"); got != "Bearer the-secret" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer the-secret")
	}
	if got := req.Header.Get("X-Molecule-Admin-Token"); got != "tok-abc" {
		t.Errorf("X-Molecule-Admin-Token = %q, want tok-abc", got)
	}
}

// TestAuthHeaders_OnlyAdminTokenWhenSecretEmpty — in the transition
// window where the tenant has admin_token but PROVISION_SHARED_SECRET
// isn't set, still send the admin token. CP middleware decides whether
// the shared secret is required.
func TestAuthHeaders_OnlyAdminTokenWhenSecretEmpty(t *testing.T) {
	p := &CPProvisioner{sharedSecret: "", adminToken: "tok-abc"}
	req := httptest.NewRequest("GET", "http://x/", nil)
	p.authHeaders(req)
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want unset", got)
	}
	if got := req.Header.Get("X-Molecule-Admin-Token"); got != "tok-abc" {
		t.Errorf("X-Molecule-Admin-Token = %q, want tok-abc", got)
	}
}

// TestStart_HappyPath — Start posts to the stubbed CP, passes the
// bearer, and parses the returned instance_id.
func TestStart_HappyPath(t *testing.T) {
	var sawBearer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawBearer = r.Header.Get("Authorization")
		if r.URL.Path != "/cp/workspaces/provision" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		// Verify the request body round-trips our fields
		var body cpProvisionRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.WorkspaceID != "ws-1" || body.Runtime != "python" {
			t.Errorf("body mismatch: %+v", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"instance_id":"i-abc123","state":"pending"}`)
	}))
	defer srv.Close()

	p := &CPProvisioner{
		baseURL:      srv.URL,
		orgID:        "org-1",
		sharedSecret: "s3cret",
		httpClient:   srv.Client(),
	}

	id, err := p.Start(context.Background(), WorkspaceConfig{
		WorkspaceID: "ws-1", Runtime: "python", Tier: 1, PlatformURL: "http://tenant",
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if id != "i-abc123" {
		t.Errorf("instance id = %q, want i-abc123", id)
	}
	if sawBearer != "Bearer s3cret" {
		t.Errorf("server saw Authorization = %q, want Bearer s3cret", sawBearer)
	}
}

// TestStart_Non201ReturnsStructuredError — when CP returns 401 with a
// structured {"error":"..."} body, Start surfaces that error message.
// Verifies the defense against log-leaking raw upstream bodies.
func TestStart_Non201ReturnsStructuredError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"invalid credentials"}`)
	}))
	defer srv.Close()

	p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}

	_, err := p.Start(context.Background(), WorkspaceConfig{WorkspaceID: "ws-1", Runtime: "py"})
	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("error message %q should include upstream error field", err.Error())
	}
}

// TestStart_NoStructuredErrorFallsBackToSize — the anti-leak path:
// when upstream returns non-JSON, we refuse to log the body and
// report only the byte count, preventing Authorization header echoes
// from landing in our logs.
func TestStart_NoStructuredErrorFallsBackToSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "raw proxy error page, could contain echoed headers")
	}))
	defer srv.Close()

	p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}

	_, err := p.Start(context.Background(), WorkspaceConfig{WorkspaceID: "ws-1", Runtime: "py"})
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
	if strings.Contains(err.Error(), "raw proxy error") {
		t.Errorf("error leaked raw body: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "<unstructured body") {
		t.Errorf("expected byte-count fallback, got %q", err.Error())
	}
}
