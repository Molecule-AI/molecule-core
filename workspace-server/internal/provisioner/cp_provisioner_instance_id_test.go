package provisioner

// Regression tests for PR #1738 (merged 2026-04-23) — CPProvisioner.Stop +
// IsRunning must look up the real EC2 instance_id (i-*) from the DB
// before calling the control plane, NOT pass the workspace UUID verbatim.
//
// Original bug:
//   url := fmt.Sprintf("%s/cp/workspaces/%s?instance_id=%s",
//                       baseURL, workspaceID, workspaceID)
//                                             ^^^^^^^^^^^^^^
//                                             sends UUID as instance_id
//
// AWS then rejects with InvalidInstanceID.Malformed, the next provision
// hits InvalidGroup.Duplicate on the leftover SG, and Save & Restart
// cascades into a full failure. Production incident 2026-04-22 on
// hongmingwang workspace a8af9d79 + recurrent on every SaaS workspace
// secret update that triggers a restart.
//
// These tests pin two invariants of the fix:
//   1. Stop + IsRunning query resolveInstanceID(ctx, workspaceID) BEFORE
//      hitting CP, and use the returned i-* ID (not the workspace UUID)
//      in the instance_id query param.
//   2. Empty instance_id → no CP call (idempotent no-op).

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestStop_UsesRealInstanceIDNotWorkspaceUUID is the load-bearing
// regression guard for #1738. If someone reverts the resolveInstanceID
// lookup and ships the `workspaceID, workspaceID` version back, this
// test fails immediately.
func TestStop_UsesRealInstanceIDNotWorkspaceUUID(t *testing.T) {
	primeInstanceIDLookup(t, map[string]string{
		"ws-cd5c9906-bfd7-4e2a-8c0b-9f1e2d3a4b5c": "i-0a1b2c3d4e5f67890",
	})

	var sawInstance string
	var sawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawInstance = r.URL.Query().Get("instance_id")
		sawPath = r.URL.Path
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
	if err := p.Stop(context.Background(), "ws-cd5c9906-bfd7-4e2a-8c0b-9f1e2d3a4b5c"); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Load-bearing assertion: the AWS-facing instance_id must be the
	// i-* ID from the DB, NEVER the workspace UUID.
	if sawInstance != "i-0a1b2c3d4e5f67890" {
		t.Errorf("#1738 REGRESSION: instance_id query = %q, want i-0a1b2c3d4e5f67890. "+
			"CP would forward this to AWS TerminateInstances — a UUID triggers "+
			"InvalidInstanceID.Malformed and orphans the EC2. See PR #1738.", sawInstance)
	}

	// Sanity: path still carries the workspace UUID (that's how CP looks
	// up the row). Only the instance_id query param changed.
	if sawPath != "/cp/workspaces/ws-cd5c9906-bfd7-4e2a-8c0b-9f1e2d3a4b5c" {
		t.Errorf("path = %q, want /cp/workspaces/ws-cd5c9906-bfd7-4e2a-8c0b-9f1e2d3a4b5c", sawPath)
	}
}

// TestStop_NoInstanceIDSkipsCPCall — when the workspace has no EC2 on
// file (never provisioned, already deprovisioned, or external runtime),
// Stop must be a no-op. Calling CP with empty instance_id triggers the
// exact AWS error the fix was meant to prevent.
func TestStop_NoInstanceIDSkipsCPCall(t *testing.T) {
	primeInstanceIDLookup(t, map[string]string{}) // empty map → "" for everything

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}
	if err := p.Stop(context.Background(), "ws-never-provisioned"); err != nil {
		t.Errorf("Stop with no instance_id should be no-op, got err: %v", err)
	}
	if called {
		t.Error("#1738 REGRESSION: Stop hit CP with empty instance_id — would trigger " +
			"InvalidInstanceID.Malformed downstream. Fix must short-circuit on empty lookup.")
	}
}

// TestIsRunning_UsesRealInstanceIDNotWorkspaceUUID mirrors the Stop test
// for IsRunning's GET /cp/workspaces/:id/status?instance_id=... path.
// Same class of bug, same acceptance criterion.
func TestIsRunning_UsesRealInstanceIDNotWorkspaceUUID(t *testing.T) {
	primeInstanceIDLookup(t, map[string]string{
		"ws-abc": "i-deadbeef",
	})

	var sawInstance string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawInstance = r.URL.Query().Get("instance_id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"state":"running"}`))
	}))
	defer srv.Close()

	p := &CPProvisioner{baseURL: srv.URL, orgID: "org-1", httpClient: srv.Client()}
	running, err := p.IsRunning(context.Background(), "ws-abc")
	if err != nil {
		t.Fatalf("IsRunning: %v", err)
	}
	if !running {
		t.Errorf("expected running=true")
	}
	if sawInstance != "i-deadbeef" {
		t.Errorf("#1738 REGRESSION: IsRunning sent instance_id=%q, want i-deadbeef", sawInstance)
	}
}
