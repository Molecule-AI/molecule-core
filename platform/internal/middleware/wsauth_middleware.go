package middleware

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/wsauth"
	"github.com/gin-gonic/gin"
)

// WorkspaceAuth returns a Gin middleware that enforces per-workspace bearer-token
// authentication on /workspaces/:id/* sub-routes.
//
// Same lazy-bootstrap contract as secrets.Values: workspaces that have no live
// token on file are grandfathered through so in-flight agents keep working
// during a rolling upgrade. Once a workspace has at least one live token every
// request MUST present a valid one in Authorization: Bearer <token>.
//
// Intended for route groups that cover all /workspaces/:id/* paths.
// The /workspaces/:id/a2a route must be registered on the root router (outside
// this group) because it already authenticates callers via CanCommunicate.
func WorkspaceAuth(database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		workspaceID := c.Param("id")
		if workspaceID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing workspace ID"})
			return
		}
		ctx := c.Request.Context()

		hasLive, err := wsauth.HasAnyLiveToken(ctx, database, workspaceID)
		if err != nil {
			log.Printf("wsauth: WorkspaceAuth: HasAnyLiveToken(%s) failed: %v", workspaceID, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth check failed"})
			return
		}
		if hasLive {
			tok := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
			if tok == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing workspace auth token"})
				return
			}
			if err := wsauth.ValidateToken(ctx, database, workspaceID, tok); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid workspace auth token"})
				return
			}
			c.Next()
			return
		}

		// #318: fail-open path. The grandfather window only exists for
		// workspaces that actually exist in the DB but pre-date Phase 30.1
		// token issuance. A fabricated UUID must NOT be let through —
		// without this check, unauthenticated callers could probe
		// `/workspaces/<fake>/secrets` and enumerate global-secret key
		// names via the fall-through 200 OK.
		exists, err := wsauth.WorkspaceExists(ctx, database, workspaceID)
		if err != nil {
			log.Printf("wsauth: WorkspaceAuth: WorkspaceExists(%s) failed: %v", workspaceID, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth check failed"})
			return
		}
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// AdminAuth returns a Gin middleware for global/admin routes (e.g.
// /settings/secrets, /admin/secrets) that have no per-workspace scope.
//
// Same lazy-bootstrap contract as WorkspaceAuth: if no live token exists
// anywhere on the platform (fresh install / pre-Phase-30 upgrade), requests
// are let through so existing deployments keep working. Once any workspace
// has a live token every request to these routes MUST present a valid one.
//
// Any valid workspace bearer token is accepted — the route is not scoped to
// a specific workspace so we only verify the token is live and unrevoked.
func AdminAuth(database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		hasLive, err := wsauth.HasAnyLiveTokenGlobal(ctx, database)
		if err != nil {
			log.Printf("wsauth: AdminAuth: HasAnyLiveTokenGlobal failed: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth check failed"})
			return
		}
		if hasLive {
			tok := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
			if tok == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin auth required"})
				return
			}
			if err := wsauth.ValidateAnyToken(ctx, database, tok); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin auth token"})
				return
			}
		}
		c.Next()
	}
}

// CanvasOrBearer is a softer admin-auth variant used ONLY for cosmetic
// canvas routes where forging the request has zero security impact (PUT
// /canvas/viewport: worst case an attacker resets the shared viewport
// position, user refreshes the page, problem solved).
//
// Accepts either:
//
//  1. A valid bearer token (same contract as AdminAuth) — covers molecli,
//     agent-to-platform calls, and anyone using the API directly.
//  2. A browser Origin header that matches CORS_ORIGINS (canvas itself).
//     This is NOT a strict auth boundary — curl can forge Origin — but for
//     cosmetic-only routes the trade-off is acceptable. Non-cosmetic routes
//     MUST NOT use this middleware (see #194 review on why it would re-open
//     #164 CRITICAL if applied to /bundles/import).
//
// Lazy-bootstrap fail-open preserved: zero-token installs pass everything
// through so fresh self-hosted / dev sessions aren't bricked.
func CanvasOrBearer(database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		hasLive, err := wsauth.HasAnyLiveTokenGlobal(ctx, database)
		if err != nil {
			log.Printf("wsauth: CanvasOrBearer HasAnyLiveTokenGlobal failed: %v — allowing request", err)
			c.Next()
			return
		}
		if !hasLive {
			c.Next()
			return
		}

		// Path 1: bearer present → bearer MUST validate. Do not fall through
		// to Origin on an invalid bearer — an attacker with a revoked /
		// expired token + a matching Origin would otherwise bypass auth.
		// Empty bearer → skip to Origin path (canvas never sends one).
		if tok := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization")); tok != "" {
			if err := wsauth.ValidateAnyToken(ctx, database, tok); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin auth token"})
				return
			}
			c.Next()
			return
		}

		// Path 2: canvas origin match. Read CORS_ORIGINS at request time so
		// tests can override via t.Setenv. canvasOriginAllowed returns true
		// iff Origin is non-empty AND exactly matches one of the configured
		// origins. Empty Origin (same-origin / server-to-server) does NOT
		// pass this check — those callers must use the bearer path.
		if canvasOriginAllowed(c.GetHeader("Origin")) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin auth required"})
	}
}

// canvasOriginAllowed returns true if origin matches any entry in the
// CORS_ORIGINS env var (comma-separated) or the localhost defaults.
// Exact-match only; no prefix or wildcard logic — that's handled by the
// real CORS middleware upstream. The intent here is "did this request come
// from the canvas page the user is already logged into?" — a binary check.
func canvasOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	allowed := []string{"http://localhost:3000", "http://localhost:3001"}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		for _, o := range strings.Split(v, ",") {
			if o = strings.TrimSpace(o); o != "" {
				allowed = append(allowed, o)
			}
		}
	}
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}
