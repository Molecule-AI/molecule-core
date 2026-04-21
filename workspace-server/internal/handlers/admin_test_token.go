// Package handlers — admin test-token endpoint (follow-up to PR #5, issue #6).
//
// GET /admin/workspaces/:id/test-token mints a fresh workspace auth token for
// E2E scripts, eliminating the register-race in test_comprehensive_e2e.sh.
// The endpoint is DELIBERATELY hidden in production: it returns 404 rather
// than 403 when disabled, so an attacker scanning for admin surfaces can't
// distinguish "route exists, forbidden" from "route doesn't exist."
//
// Enablement contract:
//
//   - If MOLECULE_ENABLE_TEST_TOKENS=1 → enabled.
//   - Else if MOLECULE_ENV is set and != "production" → enabled.
//   - Else → disabled (404).
//
// The fallback to MOLECULE_ENV keeps local dev and CI "just work" without
// requiring every operator to set the enable flag, while forcing production
// deployments (which should set MOLECULE_ENV=production) to stay locked.
package handlers

import (
	"crypto/subtle"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/wsauth"
	"github.com/gin-gonic/gin"
)

// TestTokensEnabled reports whether the /admin/workspaces/:id/test-token
// route should respond with tokens. Exported so tests (and operator health
// checks) can share the exact same gating logic.
func TestTokensEnabled() bool {
	if os.Getenv("MOLECULE_ENABLE_TEST_TOKENS") == "1" {
		return true
	}
	// Empty MOLECULE_ENV defaults to enabled — local dev runs don't set it.
	// Production deployments MUST set MOLECULE_ENV=production to lock this.
	return os.Getenv("MOLECULE_ENV") != "production"
}

// AdminTestTokenHandler mints a fresh token for an existing workspace.
type AdminTestTokenHandler struct{}

func NewAdminTestTokenHandler() *AdminTestTokenHandler {
	return &AdminTestTokenHandler{}
}

// GetTestToken handles GET /admin/workspaces/:id/test-token.
func (h *AdminTestTokenHandler) GetTestToken(c *gin.Context) {
	if !TestTokensEnabled() {
		// 404 (not 403) — hide the route's existence entirely in prod.
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// IDOR fix (#112, CRITICAL): when ADMIN_TOKEN is set, require it
	// explicitly. Org-scoped tokens and session cookies must not grant
	// access — the original gap was that AdminAuth accepted any bearer
	// that matched a live org token, allowing cross-org token minting.
	adminSecret := os.Getenv("ADMIN_TOKEN")
	if adminSecret != "" {
		tok := c.GetHeader("Authorization")
		tok = strings.TrimPrefix(tok, "Bearer ")
		if tok == "" || subtle.ConstantTimeCompare([]byte(tok), []byte(adminSecret)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin auth required"})
			return
		}
	}

	workspaceID := c.Param("id")
	if workspaceID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Confirm the workspace exists — a missing workspace also 404s so we
	// can't be used to probe for arbitrary IDs.
	var exists string
	err := db.DB.QueryRowContext(c.Request.Context(),
		`SELECT id FROM workspaces WHERE id = $1`, workspaceID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}

	token, err := wsauth.IssueToken(c.Request.Context(), db.DB, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}

	// INFO log — never include the token itself.
	log.Printf("admin: issued test token for workspace %s", workspaceID)

	c.JSON(http.StatusOK, gin.H{
		"auth_token":   token,
		"workspace_id": workspaceID,
	})
}
