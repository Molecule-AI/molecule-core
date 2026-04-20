package provisioner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

// TestStart_TransportFailureSurfaces — the CP isn't reachable at all
// (DNS fails, TCP refused, TLS handshake error). Start must return an
// error tagged with enough context to find the failed call in logs
// without leaking credentials.
func TestStart_TransportFailureSurfaces(t *testing.T) {
	// Port 1 is reserved by IANA; connect attempts fail immediately.
	p := &CPProvisioner{
		baseURL:    "http://127.0.0.1:1",
		orgID:      "org-1",
		httpClient: &http.Client{Timeout: 500 * time.Millisecond},
	}
	_, err := p.Start(context.Background(), WorkspaceConfig{WorkspaceID: "ws-1", Runtime: "py"})
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	if !strings.Contains(err.Error(), "cp provisioner: send") {
		t.Errorf("error should be tagged cp provisioner: send, got %q", err.Error())
	}
}

// TestStop_SendsBothAuthHeaders — verify #118/#130 compliance on the
// teardown path. Any call to /cp/workspaces/:id must carry both the
// platform-wide shared secret AND the per-tenant admin token, or the
// CP will 401.
func TestStop_SendsBothAuthHeaders(t *testing.T) {
	var sawBearer, sawAdminToken, sawMethod, sawPath string
	var sawInstance string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawBearer = r.Header.Get("Authorization")
		sawAdminToken = r.Header.Get("X-Molecule-Admin-Token")
		sawMethod = r.Method
		sawPath = r.URL.Path
		sawInstance = r.URL.Query().Get("instance_id")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := &CPProvisioner{
		baseURL:      srv.URL,
		orgID:        "org-1",
		sharedSecret: "s3cret",
		adminToken:   "tok-xyz",
		httpClient:   srv.Client(),
	}
	if err := p.Stop(context.Background(), "ws-1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if sawMethod != "DELETE" {
		t.Errorf("method = %q, want DELETE", sawMethod)
	}
	if sawPath != "/cp/workspaces/ws-1" {
		t.Errorf("path = %q, want /cp/workspaces/ws-1", sawPath)
	}
	if sawInstance != "ws-1" {
		t.Errorf("instance_id query = %q, want ws-1", sawInstance)
	}
	if sawBearer != "Bearer s3cret" {
		t.Errorf("bearer = %q, want Bearer s3cret", sawBearer)
	}
	if sawAdminToken != "tok-xyz" {
		t.Errorf("admin token = %q, want tok-xyz", sawAdminToken)
	}
}

// TestStop_TransportErrorSurfaces — same treatment as Start. If the
// teardown call hits a dead CP, the error must surface so the caller
// knows the workspace might still be running and needs retry.
func TestStop_TransportErrorSurfaces(t *testing.T) {
	p := &CPProvisioner{
		baseURL:    "http://127.0.0.1:1",
		orgID:      "org-1",
		httpClient: &http.Client{Timeout: 500 * time.Millisecond},
	}
	err := p.Stop(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	if !strings.Contains(err.Error(), "cp provisioner: stop") {
		t.Errorf("error should be tagged, got %q", err.Error())
	}
}

// TestIsRunning_ParsesStateField — CP returns the EC2 state, we expose
// a bool ("running"/"pending"/"terminated" → true only for "running").
func TestIsRunning_ParsesStateField(t *testing.T) {
	cases := map[string]bool{
		"running":    true,
		"pending":    false,
		"stopping":   false,
		"terminated": false,
	}
	for state, want := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/cp/workspaces/ws-1/status" {
				t.Errorf("path = %q", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"state":"`+state+`"}`)
		}))
		p := &CPProvisioner{
			baseURL:    srv.URL,
			orgID:      "org-1",
			sharedSecret: "s3cret",
			adminToken:   "tok-xyz",
			httpClient: srv.Client(),
		}
		got, err := p.IsRunning(context.Background(), "ws-1")
		srv.Close()
		if err != nil {
			t.Errorf("state=%s: IsRunning error %v", state, err)
			continue
		}
		if got != want {
			t.Errorf("state=%s: got %v, want %v", state, got, want)
		}
	}
}

// TestIsRunning_SendsBothAuthHeaders — parity with Stop. Status reads
// require the same per-tenant auth because they leak public_ip +
// private_ip to the caller.
func TestIsRunning_SendsBothAuthHeaders(t *testing.T) {
	var sawBearer, sawAdminToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawBearer = r.Header.Get("Authorization")
		sawAdminToken = r.Header.Get("X-Molecule-Admin-Token")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"state":"running"}`)
	}))
	defer srv.Close()

	p := &CPProvisioner{
		baseURL:      srv.URL,
		orgID:        "org-1",
		sharedSecret: "s3cret",
		adminToken:   "tok-xyz",
		httpClient:   srv.Client(),
	}
	_, _ = p.IsRunning(context.Background(), "ws-1")
	if sawBearer != "Bearer s3cret" {
		t.Errorf("bearer = %q, want Bearer s3cret", sawBearer)
	}
	if sawAdminToken != "tok-xyz" {
		t.Errorf("admin token = %q, want tok-xyz", sawAdminToken)
	}
}

// TestIsRunning_TransportErrorReturnsTrue — when the CP is
// unreachable, IsRunning must return (true, err) — matching the
// Docker provisioner contract so a2a_proxy stays on the alive path
// during a transient CP outage. Returning false here would trigger
// restart cascades on every brief CP blip.
//
// The sweeper (healthsweep.go) inspects err independently and skips
// on any error, so (true, err) is equally safe for that caller.
func TestIsRunning_TransportErrorReturnsTrue(t *testing.T) {
	p := &CPProvisioner{
		baseURL:    "http://127.0.0.1:1",
		orgID:      "org-1",
		httpClient: &http.Client{Timeout: 500 * time.Millisecond},
	}
	got, err := p.IsRunning(context.Background(), "ws-1")
	if err == nil {
		t.Errorf("expected transport error, got nil (got=%v)", got)
	}
	if !got {
		t.Errorf("transport failure must report running=true so a2a_proxy stays on the alive path (matches Docker provisioner contract); got false")
	}
}

// TestIsRunning_Non2xxSurfacesError — a CP 500/502/etc. must NOT
// be silently treated as "workspace stopped". Previously the handler
// would decode an empty body → State="" → return (false, nil) and
// the sweeper would see the workspace as not-running. Now every
// non-2xx is an error the caller can log + retry.
func TestIsRunning_Non2xxSurfacesError(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{"500 internal", 500},
		{"502 bad gateway", 502},
		{"503 unavailable", 503},
		{"401 unauthorized", 401},
		{"404 not found", 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, `{"state":"running"}`) // liar body — must not be trusted
			}))
			defer srv.Close()
			p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}

			got, err := p.IsRunning(context.Background(), "ws-1")
			if err == nil {
				t.Errorf("status %d: expected error, got nil", tc.status)
			}
			if !got {
				t.Errorf("status %d: must report running=true on non-2xx so a2a_proxy stays on alive path; got false", tc.status)
			}
			// Error must NOT echo the upstream body — CP 5xx bodies
			// can contain echoed headers and we don't want logs to
			// leak bearer tokens.
			if err != nil && strings.Contains(err.Error(), "running") {
				t.Errorf("status %d: error leaked upstream body: %q", tc.status, err.Error())
			}
		})
	}
}

// TestIsRunning_MalformedJSONBodyReturnsError — 200 but invalid JSON
// must surface an error rather than silently returning false. Prevents
// a middleware glitch (HTML error page with 200) from looking like
// "workspace stopped".
func TestIsRunning_MalformedJSONBodyReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = io.WriteString(w, "<html>maintenance mode</html>")
	}))
	defer srv.Close()
	p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}

	got, err := p.IsRunning(context.Background(), "ws-1")
	if err == nil {
		t.Errorf("malformed body: expected error, got nil (got=%v)", got)
	}
	if !got {
		t.Errorf("malformed body must report running=true so a2a_proxy stays on alive path; got false")
	}
}

// TestIsRunning_ContractCompat_A2AProxy — codifies the critical
// invariant that a2a_proxy.go line ~534 depends on: during CP
// transient errors, the handler must inspect `running`, see true,
// and skip the restart cascade. If this contract drifts (e.g., a
// future refactor returns false on error), every brief CP outage
// cascades into a workspace restart storm.
//
// This is a regression guard, not a functional test — it asserts
// the documented contract values rather than simulating the whole
// a2a_proxy flow.
func TestIsRunning_ContractCompat_A2AProxy(t *testing.T) {
	// Simulate every error path and assert running==true for each.
	t.Run("transport error", func(t *testing.T) {
		p := &CPProvisioner{
			baseURL: "http://127.0.0.1:1", orgID: "org-1",
			httpClient: &http.Client{Timeout: 500 * time.Millisecond},
		}
		running, err := p.IsRunning(context.Background(), "ws-1")
		if err == nil || !running {
			t.Errorf("want (true, err); got (%v, %v)", running, err)
		}
	})
	t.Run("CP 500 response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(500)
		}))
		defer srv.Close()
		p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}
		running, err := p.IsRunning(context.Background(), "ws-1")
		if err == nil || !running {
			t.Errorf("want (true, err); got (%v, %v)", running, err)
		}
	})
	t.Run("malformed 200 body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			_, _ = io.WriteString(w, "garbage")
		}))
		defer srv.Close()
		p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}
		running, err := p.IsRunning(context.Background(), "ws-1")
		if err == nil || !running {
			t.Errorf("want (true, err); got (%v, %v)", running, err)
		}
	})
	// And the non-error paths must still report the truth.
	t.Run("2xx stopped → false nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			_, _ = io.WriteString(w, `{"state":"stopped"}`)
		}))
		defer srv.Close()
		p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}
		running, err := p.IsRunning(context.Background(), "ws-1")
		if err != nil || running {
			t.Errorf("want (false, nil); got (%v, %v)", running, err)
		}
	})
	t.Run("2xx running → true nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			_, _ = io.WriteString(w, `{"state":"running"}`)
		}))
		defer srv.Close()
		p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}
		running, err := p.IsRunning(context.Background(), "ws-1")
		if err != nil || !running {
			t.Errorf("want (true, nil); got (%v, %v)", running, err)
		}
	})
}

// TestClose_Noop — explicit contract: Close has no side effects and
// no error. Exists for the Provisioner interface; compliance guard.
func TestClose_Noop(t *testing.T) {
	p := &CPProvisioner{}
	if err := p.Close(); err != nil {
		t.Errorf("Close should return nil, got %v", err)
	}
}
