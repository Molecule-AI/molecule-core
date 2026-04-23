package middleware

import (
	"os"
	"strings"
)

// Dev-mode escape hatch — factored out of AdminAuth + WorkspaceAuth so a
// future third caller (or a change to what "dev mode" means) touches one
// place. Narrowing the exposed seam also makes it grep-able from security
// reviews: every `isDevModeFailOpen()` call is an intentional fail-open.
//
// Why the helper exists at all: on `go run ./cmd/server` the Canvas (at
// localhost:3000) calls the platform (at localhost:8080) cross-port. Both
// `isSameOriginCanvas` (Referer==Host) and the AdminAuth Tier-1 fail-open
// (no tokens in DB) close the moment the user creates their first
// workspace. Without this hatch the Canvas 401s on every /workspaces
// enumeration and every /workspaces/:id/* read until the operator sets
// `ADMIN_TOKEN` and rebuilds the Canvas bundle with a matching
// `NEXT_PUBLIC_ADMIN_TOKEN`. That's too much friction for a local smoke
// test — hence the hatch.
//
// Why it's safe for SaaS: hosted tenants are provisioned with both
// `ADMIN_TOKEN` (a random secret, checked by Tier-2 above) and
// `MOLECULE_ENV=production`. Either one being set makes this helper
// return false, so the fail-open branch is unreachable in production.
// The convention matches `handlers/admin_test_token.go`, which gates
// the e2e test-token mint on `MOLECULE_ENV != "production"`.

// devModeEnvValues is the set of MOLECULE_ENV values that count as
// "explicit dev mode". Production callers don't set any of these.
// Case-insensitive compare via strings.ToLower below.
var devModeEnvValues = map[string]struct{}{
	"development": {},
	"dev":         {},
}

// isDevModeFailOpen reports whether the AdminAuth / WorkspaceAuth
// middleware should let a bearer-less request through despite live
// workspace tokens existing in the DB.
//
// True only when BOTH:
//   - `ADMIN_TOKEN` is empty (operator has not opted in to the #684
//     closure), AND
//   - `MOLECULE_ENV` is explicitly a dev value ("development" / "dev").
//
// Either condition failing returns false — that's the SaaS safety
// guarantee. Tests: `devmode_test.go` covers every branch.
func isDevModeFailOpen() bool {
	if os.Getenv("ADMIN_TOKEN") != "" {
		return false
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("MOLECULE_ENV")))
	_, ok := devModeEnvValues[env]
	return ok
}
