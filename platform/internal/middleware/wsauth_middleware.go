package middleware

import (
	"database/sql"
	"log"
	"net/http"

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
