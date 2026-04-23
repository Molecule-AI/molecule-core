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
//
// orgID is the caller's org workspace ID, captured at mint time.
// requireCallerOwnsOrg (org_plugin_allowlist.go:116) uses this to
// enforce org isolation (#1200 / F1094).
func (h *OrgTokenHandler) Create(c *gin.Context) {
	var req createOrgTokenRequest
	// Optional body — an empty POST should still work (unnamed token).
	_ = c.ShouldBindJSON(&req)
	if len(req.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name too long (max 100 chars)"})
		return
	}

	createdBy, orgID := orgTokenActor(c)

	plaintext, id, err := orgtoken.Issue(c.Request.Context(), db.DB, req.Name, createdBy, orgID)
	if err != nil {
		log.Printf("orgtoken issue: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mint token"})
		return
	}
	log.Printf("orgtoken: minted id=%s by=%s org=%s name=%q", id, createdBy, orgID, req.Name)

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
	actor, _ := orgTokenActor(c)
	log.Printf("orgtoken: revoked id=%s by=%s", id, actor)
	c.JSON(http.StatusOK, gin.H{"revoked": id})
}

// Provenance labels used in the org_api_tokens.created_by column
// and in mint/revoke audit logs. Kept as constants so the labels
// are greppable across services (log pipelines, audit queries).
const (
	actorOrgTokenPrefix = "org-token:" // appended: 8-char plaintext prefix from the UI
	actorSession        = "session"    // WorkOS-session-verified call
	actorAdminToken     = "admin-token" // bootstrap ADMIN_TOKEN env
)

// callerContext returns the caller's org workspace ID for use in
// org-token creation (#1200 / F1094). It reads org_token_id from the
// gin context (set by AdminAuth when an org token authed the request)
// and looks up the token's org_id.
//
// For session/ADMIN_TOKEN callers (no org_token_id in context), returns
// ("", "") so the token is minted as "unanchored" (org_id=NULL).
// Unanchored tokens cannot access org-scoped routes — safer than
// permitting cross-org access until the operator explicitly sets org_id.
func callerOrg(c *gin.Context) string {
	tokenID, ok := c.Get("org_token_id")
	if !ok {
		return ""
	}
	tokID, ok := tokenID.(string)
	if !ok || tokID == "" {
		return ""
	}
	orgID, err := orgtoken.OrgIDByTokenID(c.Request.Context(), db.DB, tokID)
	if err != nil || orgID == "" {
		return ""
	}
	return orgID
}

// orgTokenActor returns (createdBy, orgID) for the current request.
//
//   - If authed via another org token (org_token_id in context),
//     createdBy = "org-token:<prefix>" and orgID = token's org_id.
//   - If authed via session cookie (AdminAuth's session tier),
//     createdBy = "session", orgID = "" (session → org mapping not
//     available in the handler; must be filled by the CP or left null).
//   - If ADMIN_TOKEN / bootstrap, createdBy = "admin-token",
//     orgID = "".
func orgTokenActor(c *gin.Context) (createdBy, orgID string) {
	if v, ok := c.Get("org_token_prefix"); ok {
		if s, ok := v.(string); ok && s != "" {
			return actorOrgTokenPrefix + s, callerOrg(c)
		}
	}
	// Session-tier auth doesn't stash a WorkOS user_id in the gin
	// context today. Until it does, treat session requests as "session"
	// with no org anchor.
	if c.GetHeader("Cookie") != "" {
		return actorSession, ""
	}
	return actorAdminToken, ""
}
