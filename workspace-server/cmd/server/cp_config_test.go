package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestRefreshEnvFromCP_NoopWhenNotSaaS: without MOLECULE_ORG_ID or
// ADMIN_TOKEN, the function short-circuits silently — self-hosted dev
// must not fail or log spam here.
func TestRefreshEnvFromCP_NoopWhenNotSaaS(t *testing.T) {
	t.Setenv("MOLECULE_ORG_ID", "")
	t.Setenv("ADMIN_TOKEN", "")
	if err := refreshEnvFromCP(); err != nil {
		t.Errorf("expected nil on non-SaaS, got %v", err)
	}
}

// TestRefreshEnvFromCP_AppliesCPResponse: wire a stub CP, run refresh,
// confirm the returned env vars ended up in os.Environ().
func TestRefreshEnvFromCP_AppliesCPResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tenant-admin-token" {
			t.Errorf("bearer: got %q", got)
		}
		if got := r.Header.Get("X-Molecule-Org-Id"); got != "org-abc" {
			t.Errorf("org id header: got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"MOLECULE_CP_SHARED_SECRET":"new-secret","MOLECULE_CP_URL":"https://api.moleculesai.app"}`)
	}))
	defer srv.Close()

	t.Setenv("MOLECULE_ORG_ID", "org-abc")
	t.Setenv("ADMIN_TOKEN", "tenant-admin-token")
	t.Setenv("MOLECULE_CP_URL", srv.URL)
	t.Setenv("MOLECULE_CP_SHARED_SECRET", "") // clear before refresh

	if err := refreshEnvFromCP(); err != nil {
		t.Fatalf("refreshEnvFromCP: %v", err)
	}
	if got := os.Getenv("MOLECULE_CP_SHARED_SECRET"); got != "new-secret" {
		t.Errorf("SHARED_SECRET: want new-secret, got %q", got)
	}
}

// TestRefreshEnvFromCP_CPUnreachableDoesNotFailBoot: network errors must
// return non-nil BUT main.go treats that as warn-and-continue. We assert
// the function returns an error (not a panic) so the caller can log.
func TestRefreshEnvFromCP_CPUnreachableDoesNotFailBoot(t *testing.T) {
	t.Setenv("MOLECULE_ORG_ID", "org-abc")
	t.Setenv("ADMIN_TOKEN", "t")
	t.Setenv("MOLECULE_CP_URL", "http://127.0.0.1:1") // closed port
	err := refreshEnvFromCP()
	if err == nil {
		t.Error("expected an error when CP is unreachable")
	}
}

// TestRefreshEnvFromCP_NonOKPropagates: CP returns 500 → error.
func TestRefreshEnvFromCP_NonOKPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("MOLECULE_ORG_ID", "org-abc")
	t.Setenv("ADMIN_TOKEN", "t")
	t.Setenv("MOLECULE_CP_URL", srv.URL)
	if err := refreshEnvFromCP(); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

// TestRefreshEnvFromCP_RejectsOversizedValue: a single-value-over-4KiB
// payload must NOT poison the environment.
func TestRefreshEnvFromCP_RejectsOversizedValue(t *testing.T) {
	giant := make([]byte, 5<<10)
	for i := range giant {
		giant[i] = 'x'
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"MOLECULE_CP_SHARED_SECRET":%q}`, string(giant))
	}))
	defer srv.Close()
	t.Setenv("MOLECULE_ORG_ID", "org-abc")
	t.Setenv("ADMIN_TOKEN", "t")
	t.Setenv("MOLECULE_CP_URL", srv.URL)
	t.Setenv("MOLECULE_CP_SHARED_SECRET", "original")
	if err := refreshEnvFromCP(); err != nil {
		t.Fatalf("refreshEnvFromCP: %v", err)
	}
	if got := os.Getenv("MOLECULE_CP_SHARED_SECRET"); got != "original" {
		t.Errorf("oversized value was applied — want %q, got %d bytes",
			"original", len(got))
	}
}
