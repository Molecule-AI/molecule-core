package handlers

// auth_helpers.go — shared auth primitives for workspace-scoped and global endpoints.
//
// Phase 30.1 introduced per-workspace bearer tokens (see wsauth/tokens.go).
// This file centralises the validation helpers so delegation.go, activity.go, and
// any future handlers don't duplicate the bootstrap-aware grandfathering logic.

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/wsauth"
	"github.com/gin-gonic/gin"
)

// requireWorkspaceAuth enforces bearer-token auth for workspace-scoped endpoints
// (e.g. POST /workspaces/:id/delegations/record).
//
// Behaviour mirrors RegistryHandler.requireWorkspaceToken (Phase 30.1):
//   - workspace has live tokens → Authorization: Bearer <token> is mandatory
//   - workspace has NO live tokens → grandfathered through (legacy / pre-upgrade)
//
// Returns nil when the caller is authenticated (or grandfathered).
// Returns a non-nil error and writes the 401 response when auth fails — the
// caller MUST return immediately without writing another response.
func requireWorkspaceAuth(ctx context.Context, c *gin.Context, workspaceID string) error {
	hasLive, err := wsauth.HasAnyLiveToken(ctx, db.DB, workspaceID)
	if err != nil {
		// DB hiccup — fail open so a transient error doesn't kill all traffic.
		log.Printf("wsauth: HasAnyLiveToken(%s) failed: %v — allowing request", workspaceID, err)
		return nil
	}
	if !hasLive {
		// Legacy / pre-upgrade workspace. Its next /registry/register will mint a token.
		return nil
	}
	token := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing workspace auth token"})
		return errors.New("missing workspace auth token")
	}
	if err := wsauth.ValidateToken(ctx, db.DB, workspaceID, token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid workspace auth token"})
		return err
	}
	return nil
}

// requireInternalAPISecret enforces a shared-secret bearer-token gate on global
// endpoints (e.g. GET /workspaces) that are not scoped to a single workspace.
//
// The secret is read from the INTERNAL_API_SECRET environment variable.
// If that variable is unset the check is SKIPPED for backward compatibility —
// set it in production to activate enforcement.
//
// Returns nil on success (or when the secret is unset).
// Returns a non-nil error and writes 401 when the caller fails the check — the
// caller MUST return immediately without writing another response.
func requireInternalAPISecret(c *gin.Context) error {
	secret := os.Getenv("INTERNAL_API_SECRET")
	if secret == "" {
		// Bootstrap / development: no secret configured — allow through.
		return nil
	}
	token := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
	if token == "" || token != secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
		return errors.New("unauthorized: missing or invalid INTERNAL_API_SECRET")
	}
	return nil
}
