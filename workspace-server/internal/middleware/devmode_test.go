package middleware

import (
	"testing"
)

// Unit tests for the isDevModeFailOpen predicate. The AdminAuth and
// WorkspaceAuth middleware tests exercise the same helper indirectly via
// HTTP, but a direct predicate test locks the pure-logic behaviour:
// future callers can add themselves to `devmode.go` with confidence.

func TestIsDevModeFailOpen_DevModeNoAdminToken_True(t *testing.T) {
	t.Setenv("MOLECULE_ENV", "development")
	t.Setenv("ADMIN_TOKEN", "")
	if !isDevModeFailOpen() {
		t.Error("expected dev mode + no admin token to return true")
	}
}

func TestIsDevModeFailOpen_DevModeShortAlias_True(t *testing.T) {
	// "dev" is a valid alias for "development" — matches the convention
	// in handlers/admin_test_token.go.
	t.Setenv("MOLECULE_ENV", "dev")
	t.Setenv("ADMIN_TOKEN", "")
	if !isDevModeFailOpen() {
		t.Error("expected MOLECULE_ENV=dev to be treated as dev mode")
	}
}

func TestIsDevModeFailOpen_AdminTokenSet_False(t *testing.T) {
	// Setting ADMIN_TOKEN is the operator's explicit opt-in to the #684
	// closure. Dev mode must NOT silently override that signal.
	t.Setenv("MOLECULE_ENV", "development")
	t.Setenv("ADMIN_TOKEN", "operator-explicitly-set-this")
	if isDevModeFailOpen() {
		t.Error("explicit ADMIN_TOKEN must suppress the dev-mode hatch")
	}
}

func TestIsDevModeFailOpen_Production_False(t *testing.T) {
	// The SaaS-safety guarantee: production tenants always have
	// MOLECULE_ENV=production, so the hatch is unreachable even if a
	// misconfigured deployment also leaves ADMIN_TOKEN unset.
	t.Setenv("MOLECULE_ENV", "production")
	t.Setenv("ADMIN_TOKEN", "")
	if isDevModeFailOpen() {
		t.Error("production must never hit the dev-mode fail-open branch")
	}
}

func TestIsDevModeFailOpen_CaseInsensitive(t *testing.T) {
	// Operators shouldn't have to remember exact casing for a dev-only
	// convenience. "Development", "DEV", "  dev  " all count.
	cases := []string{"Development", "DEVELOPMENT", "Dev", "DEV", "  dev  "}
	for _, env := range cases {
		t.Run(env, func(t *testing.T) {
			t.Setenv("MOLECULE_ENV", env)
			t.Setenv("ADMIN_TOKEN", "")
			if !isDevModeFailOpen() {
				t.Errorf("MOLECULE_ENV=%q should count as dev mode", env)
			}
		})
	}
}

func TestIsDevModeFailOpen_UnknownEnv_False(t *testing.T) {
	// Arbitrary / unset MOLECULE_ENV values are NOT treated as dev mode.
	// Keeps the fail-open branch narrow — no silent opt-in from a typo.
	cases := []string{"", "staging", "local", "preview", "test", "devel"}
	for _, env := range cases {
		t.Run(env, func(t *testing.T) {
			t.Setenv("MOLECULE_ENV", env)
			t.Setenv("ADMIN_TOKEN", "")
			if isDevModeFailOpen() {
				t.Errorf("MOLECULE_ENV=%q must not enable fail-open", env)
			}
		})
	}
}
