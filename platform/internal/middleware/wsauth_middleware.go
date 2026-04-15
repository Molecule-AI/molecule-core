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
//
// Issue #168 — canvas Origin fallback:
// Canvas makes all its fetch calls with credentials:"include" but does NOT
// set an Authorization header. PR #167 gated several canvas-facing routes
// (viewport, events, bundles) behind AdminAuth, breaking them silently.
//
// Fix: after Bearer auth fails (no header), allow requests whose Origin
// header matches the CORS_ORIGINS env var or the localhost defaults. This
// is not a strict auth boundary — non-browser clients can set an arbitrary
// Origin — but it matches what CORS already enforces in the browser. The
// real perimeter defence against external threats is the network layer
// (CORS_ORIGINS is set to the canonical canvas URL in production).
// Bearer token auth is unchanged for API clients and agents.
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
			// Primary path: Authorization: Bearer <token> header (API clients,
			// molecli, agent-to-platform calls).
			tok := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
			if tok != "" {
				if err := wsauth.ValidateAnyToken(ctx, database, tok); err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin auth token"})
					return
				}
				c.Next()
				return
			}

			// Canvas fallback (#168): trust requests from a configured canvas
			// origin. Origin is set by the browser automatically for all
			// cross-origin fetch() calls and cannot be overridden by page JS.
			// Non-browser clients (curl/agents) are expected to use Bearer.
			origin := c.GetHeader("Origin")
			if origin != "" {
				for _, allowed := range canvasOrigins() {
					if strings.TrimSpace(allowed) == origin {
						c.Next()
						return
					}
				}
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin auth required"})
			return
		}
		c.Next()
	}
}

// canvasOrigins returns the set of browser origins that AdminAuth trusts for the
// canvas-fallback path. Reads CORS_ORIGINS at call time (not init) so the value
// can be overridden in tests via t.Setenv without a process restart.
func canvasOrigins() []string {
	origins := []string{"http://localhost:3000", "http://localhost:3001"}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		for _, o := range strings.Split(v, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
	}
	return origins
}
