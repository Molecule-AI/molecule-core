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
// Strict: every request MUST carry Authorization: Bearer <token> matching a
// live (non-revoked) token for the workspace. No grace period, no fail-open.
//
// History: originally this middleware had a lazy-bootstrap grace period for
// pre-Phase-30.1 workspaces without a live token, so rolling upgrades didn't
// brick in-flight agents. #318 tightened the fake-UUID leak (non-existent
// workspace IDs were falling through). #351 then showed the remaining hole:
// test-artifact workspaces from prior DAST runs still exist in the DB with
// empty configs and no tokens, so they pass WorkspaceExists + fall through
// the grace period — leaking global-secret key names to any unauth caller on
// the Docker network. Phase 30.1 shipped months ago; every live workspace has
// since gone through multiple boot cycles and acquired a token. The grace
// period no longer serves legitimate traffic. Removing it entirely closes
// #351 without affecting registration (which is on /registry/register,
// outside this middleware's scope).
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

		tok := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
		if tok != "" {
			if err := wsauth.ValidateToken(ctx, database, workspaceID, tok); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid workspace auth token"})
				return
			}
			c.Next()
			return
		}
		// Same-origin canvas on tenant image — Referer matches Host.
		if isSameOriginCanvas(c) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing workspace auth token"})
		return
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
			// Bearer token path — agents, CLI, and API clients.
			tok := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
			if tok != "" {
				if err := wsauth.ValidateAnyToken(ctx, database, tok); err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin auth token"})
					return
				}
				c.Next()
				return
			}
			// Canvas origin path — cross-origin canvas (CORS_ORIGINS match).
			if canvasOriginAllowed(c.GetHeader("Origin")) {
				c.Next()
				return
			}
			// Same-origin canvas path — tenant image where canvas + API share a host.
			if isSameOriginCanvas(c) {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin auth required"})
			return
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

		// Path 2: canvas origin match (cross-origin canvas).
		if canvasOriginAllowed(c.GetHeader("Origin")) {
			c.Next()
			return
		}

		// Path 3: same-origin canvas (tenant image).
		if isSameOriginCanvas(c) {
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

// isSameOriginCanvas returns true when the request appears to come from the
// canvas UI served by the same Go process (tenant image). In this topology,
// the browser sends same-origin requests with an empty Origin header but a
// Referer matching the request Host. We accept these requests because the
// canvas is the trusted frontend — same as if Origin matched CORS_ORIGINS.
//
// This only fires when CANVAS_PROXY_URL is set (i.e. the combined tenant
// image is active), so self-hosted / dev setups with separate canvas and
// platform origins are unaffected.
// canvasProxyActive is true when the platform runs as a combined tenant
// image (CANVAS_PROXY_URL set at boot). Cached once to avoid os.Getenv
// on every request.
var canvasProxyActive = os.Getenv("CANVAS_PROXY_URL") != ""

func isSameOriginCanvas(c *gin.Context) bool {
	if !canvasProxyActive {
		return false
	}
	referer := c.GetHeader("Referer")
	if referer == "" {
		return false
	}
	host := c.Request.Host
	if host == "" {
		return false
	}
	// Referer must start with https://<host>/ or http://<host>/ (trailing
	// slash required to prevent hongming-wang.moleculesai.app.evil.com from
	// matching hongming-wang.moleculesai.app).
	return strings.HasPrefix(referer, "https://"+host+"/") ||
		strings.HasPrefix(referer, "http://"+host+"/") ||
		referer == "https://"+host ||
		referer == "http://"+host
}
