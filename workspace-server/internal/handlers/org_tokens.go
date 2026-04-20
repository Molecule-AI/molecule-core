package handlers

import (
	"log"
	"net/http"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/orgtoken"
	"github.com/gin-gonic/gin"
)

// OrgTokenHandler exposes CRUD for organization-scoped API tokens.
//
// Routes (all AdminAuth-gated, mounted at root):
//
//	GET    /org/tokens         list live tokens
//	POST   /org/tokens         mint a new token; plaintext returned once
//	DELETE /org/tokens/:id     revoke
//
// Sibling of TokenHandler (workspace-scoped); deliberately kept
// separate because the admin surface is wider — an org token can
// mint/revoke other org tokens, escalate workspace perms, etc. —
// and conflating them with workspace tokens makes revoke UX
// confusing.
type OrgTokenHandler struct{}

func NewOrgTokenHandler() *OrgTokenHandler {
	return &OrgTokenHandler{}
}

// List returns live (non-revoked) tokens, newest-first. Prefix only —
// never plaintext or hash.
func (h *OrgTokenHandler) List(c *gin.Context) {
	tokens, err := orgtoken.List(c.Request.Context(), db.DB)
	if err != nil {
		log.Printf("orgtoken list: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tokens"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tokens": tokens, "count": len(tokens)})
}

type createOrgTokenRequest struct {
	Name string `json:"name"`
}

type createOrgTokenResponse struct {
	ID       string `json:"id"`
	Prefix   string `json:"prefix"`
	Name     string `json:"name,omitempty"`
	Token    string `json:"auth_token"` // plaintext — shown ONCE
	Warning  string `json:"warning"`    // UX hint: copy now
}

// Create mints a new org token. The plaintext is returned exactly
// once in the response body. Mirrors wsauth's Issue semantics so UI
// flow (copy-once, dismiss, no retrieval) is consistent across
// token types.
//
// created_by is captured from the org_token_id or admin-token
// provenance of the current request — so an audit trail points back
// to who minted what. For the bootstrap ADMIN_TOKEN path, created_by
// is "admin-token" (no session identity available).
func (h *OrgTokenHandler) Create(c *gin.Context) {
	var req createOrgTokenRequest
	// Optional body — an empty POST should still work (unnamed token).
	_ = c.ShouldBindJSON(&req)
	if len(req.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name too long (max 100 chars)"})
		return
	}

	createdBy := orgTokenActor(c)

	plaintext, id, err := orgtoken.Issue(c.Request.Context(), db.DB, req.Name, createdBy)
	if err != nil {
		log.Printf("orgtoken issue: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mint token"})
		return
	}
	log.Printf("orgtoken: minted id=%s by=%s name=%q", id, createdBy, req.Name)

	c.JSON(http.StatusOK, createOrgTokenResponse{
		ID:      id,
		Prefix:  plaintext[:8],
		Name:    req.Name,
		Token:   plaintext,
		Warning: "copy this token now; it will not be shown again",
	})
}

// Revoke flips revoked_at. 404 when the id doesn't exist OR was
// already revoked — idempotent from the caller's perspective.
func (h *OrgTokenHandler) Revoke(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	ok, err := orgtoken.Revoke(c.Request.Context(), db.DB, id)
	if err != nil {
		log.Printf("orgtoken revoke: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found or already revoked"})
		return
	}
	actor := orgTokenActor(c)
	log.Printf("orgtoken: revoked id=%s by=%s", id, actor)
	c.JSON(http.StatusOK, gin.H{"revoked": id})
}

// orgTokenActor derives a short provenance string for audit.
//
//   - If the request was authed via another org token, return
//     "org-token:<short>" so revoke audits show which token minted
//     the new one.
//   - If authed via session cookie (AdminAuth's session tier), the
//     middleware doesn't set anything on c for us — return "session"
//     as a generic label. When we grow a session-user-id capture
//     upgrade this to return the real WorkOS user_id.
//   - Else (ADMIN_TOKEN / bootstrap), return "admin-token".
func orgTokenActor(c *gin.Context) string {
	if v, ok := c.Get("org_token_id"); ok {
		if s, ok := v.(string); ok && len(s) >= 8 {
			return "org-token:" + s[:8]
		}
	}
	// Session-tier auth doesn't stash an identity in the gin context
	// today. Until it does, treat session requests as "session". A
	// follow-up issue captures WorkOS user_id here.
	if c.GetHeader("Cookie") != "" {
		return "session"
	}
	return "admin-token"
}
