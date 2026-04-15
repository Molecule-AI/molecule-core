package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/events"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/models"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/wsauth"
	"github.com/gin-gonic/gin"
)

type RegistryHandler struct {
	broadcaster *events.Broadcaster
}

func NewRegistryHandler(b *events.Broadcaster) *RegistryHandler {
	return &RegistryHandler{broadcaster: b}
}

// validateRegistrationURL checks that a workspace-supplied URL is safe to store.
//
// C6 security fix: blocks SSRF-capable URLs. An attacker who can register an
// arbitrary URL could trick the platform into fetching cloud metadata endpoints
// (169.254.169.254 — AWS/GCP IMDS) or probing RFC-1918 internal services.
//
// Allowed: http:// or https:// with a public host (including 127.0.0.1 which is
// used legitimately by Docker-provisioned workspaces).
// Rejected: non-http/https schemes, link-local (169.254.*), RFC-1918 private
// ranges (10.*, 172.16-31.*, 192.168.*).
func validateRegistrationURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme %q is not allowed — only http and https are accepted", u.Scheme)
	}
	hostname := u.Hostname()
	ip := net.ParseIP(hostname)
	if ip == nil {
		// DNS hostname — not an IP; allow (DNS-based SSRF is out of scope here).
		return nil
	}
	// Block link-local unicast: 169.254.0.0/16 (IPv4 AWS/GCP IMDS) and
	// fe80::/10 (IPv6 link-local) are both caught by IsLinkLocalUnicast().
	if ip.IsLinkLocalUnicast() {
		return fmt.Errorf("URL %q targets a link-local address which is not allowed", rawURL)
	}
	// Block RFC-1918 IPv4 private ranges, IPv6 loopback, and IPv6 ULA.
	// fe80::/10 is already handled above; 127.0.0.0/8 is intentionally ALLOWED
	// because the provisioner sets workspace URLs to http://127.0.0.1:<port>.
	// ::1/128 has no legitimate use here and must be blocked (it is the IPv6
	// equivalent of 127.0.0.1 but no Docker workspace URL uses it).
	// fc00::/7 covers both fc00::/8 and fd00::/8 (IPv6 ULA) — PR #169 gap.
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"::1/128",   // IPv6 loopback
		"fc00::/7",  // IPv6 ULA (covers fc00::/8 and fd00::/8)
	}
	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return fmt.Errorf("URL %q targets a private IP range which is not allowed", rawURL)
		}
	}
	return nil
}

// Register handles POST /registry/register
// Upserts workspace, sets Redis TTL, broadcasts WORKSPACE_ONLINE.
func (h *RegistryHandler) Register(c *gin.Context) {
	var payload models.RegisterPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// C6: reject SSRF-capable URLs before persisting to the database.
	if err := validateRegistrationURL(payload.URL); err != nil {
		log.Printf("Registry register SSRF attempt blocked (id=%s url=%s): %v", payload.ID, payload.URL, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	agentCardStr := string(payload.AgentCard)

	// Upsert workspace: update url, agent_card, status if already exists.
	// On INSERT (workspace not yet created via POST /workspaces), use ID as name placeholder.
	// Keep existing URL if provisioner already set a host-accessible one (starts with http://127.0.0.1).
	//
	// #73 guard: `WHERE workspaces.status IS DISTINCT FROM 'removed'` prevents
	// a late heartbeat from a workspace that was just deleted from resurrecting
	// the row. Without this guard, bulk deletes left tier-3 stragglers because
	// the last pre-teardown heartbeat flipped status back to 'online' after
	// Delete's UPDATE.
	_, err := db.DB.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, url, agent_card, status, last_heartbeat_at)
		VALUES ($1, $2, $3, $4::jsonb, 'online', now())
		ON CONFLICT (id) DO UPDATE SET
			url = CASE
				WHEN workspaces.url LIKE 'http://127.0.0.1%' THEN workspaces.url
				ELSE EXCLUDED.url
			END,
			agent_card = EXCLUDED.agent_card,
			status = 'online',
			last_heartbeat_at = now(),
			updated_at = now()
		WHERE workspaces.status IS DISTINCT FROM 'removed'
	`, payload.ID, payload.ID, payload.URL, agentCardStr)
	if err != nil {
		log.Printf("Registry register error: %v (id=%s)", err, payload.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to register: %v", err)})
		return
	}

	// Set Redis liveness key
	if err := db.SetOnline(ctx, payload.ID); err != nil {
		log.Printf("Registry redis error: %v", err)
	}

	// Cache URL — prefer existing provisioner URL over agent-reported one.
	// The DB CASE already preserves provisioner URLs, so read from DB as source of truth
	// instead of adding a Redis round-trip on every registration.
	cachedURL := payload.URL
	var dbURL string
	if err := db.DB.QueryRowContext(ctx, `SELECT url FROM workspaces WHERE id = $1`, payload.ID).Scan(&dbURL); err == nil {
		if strings.HasPrefix(dbURL, "http://127.0.0.1") {
			cachedURL = dbURL
		}
	}
	if err := db.CacheURL(ctx, payload.ID, cachedURL); err != nil {
		log.Printf("Registry cache url error: %v", err)
	}

	// Cache agent-reported URL separately for workspace-to-workspace discovery
	// (Docker containers can reach each other by hostname but not via host ports)
	if err := db.CacheInternalURL(ctx, payload.ID, payload.URL); err != nil {
		log.Printf("Registry cache internal url error: %v", err)
	}

	// Broadcast WORKSPACE_ONLINE
	if err := h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_ONLINE", payload.ID, map[string]interface{}{
		"url":        cachedURL,
		"agent_card": payload.AgentCard,
	}); err != nil {
		log.Printf("Registry broadcast error: %v", err)
	}

	// Phase 30.1: issue a workspace auth token on first registration.
	//
	// On re-registration (agent restart), we DON'T issue a new token —
	// the agent is expected to keep the one it got the first time.
	// Issuing on every register would flood the table and make log
	// forensics noisier than it needs to be.
	//
	// Legacy workspaces that registered before tokens existed have no
	// live token; they bootstrap one here on their next register call.
	// New workspaces always pass through this path on their first boot.
	response := gin.H{"status": "registered"}
	if hasLive, hasLiveErr := wsauth.HasAnyLiveToken(ctx, db.DB, payload.ID); hasLiveErr == nil && !hasLive {
		token, tokErr := wsauth.IssueToken(ctx, db.DB, payload.ID)
		if tokErr != nil {
			// Don't fail the whole register on token-issuance error — the
			// agent is already online per the upsert above. Log and continue.
			// If needed, the agent can call /registry/register again and
			// we'll retry issuance. Alternative paths (/workspaces/:id/
			// tokens POST, to be added in a later phase) can also mint one.
			log.Printf("Registry: failed to issue auth token for %s: %v", payload.ID, tokErr)
		} else {
			response["auth_token"] = token
		}
	} else if hasLiveErr != nil {
		log.Printf("Registry: token existence check failed for %s: %v", payload.ID, hasLiveErr)
	}

	c.JSON(http.StatusOK, response)
}

// Heartbeat handles POST /registry/heartbeat
func (h *RegistryHandler) Heartbeat(c *gin.Context) {
	var payload models.HeartbeatPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Phase 30.1: require a valid workspace auth token on every heartbeat
	// IF the workspace has any live tokens on file. Legacy workspaces that
	// registered before tokens existed are grandfathered through (tokens
	// get issued on their next /registry/register call); new workspaces
	// always have one. This design lets us ship auth without forcing a
	// synchronized restart of every running workspace.
	if err := h.requireWorkspaceToken(ctx, c, payload.WorkspaceID); err != nil {
		return // response already written
	}

	// Read previous current_task to detect changes (before the UPDATE)
	var prevTask string
	_ = db.DB.QueryRowContext(ctx, `SELECT COALESCE(current_task, '') FROM workspaces WHERE id = $1`, payload.WorkspaceID).Scan(&prevTask)

	// Update heartbeat columns. #73 guard: exclude 'removed' rows so a
	// late heartbeat from a container that's being torn down doesn't
	// refresh last_heartbeat_at on a tombstoned workspace (which would
	// otherwise confuse the liveness monitor).
	_, err := db.DB.ExecContext(ctx, `
		UPDATE workspaces SET
			last_heartbeat_at = now(),
			last_error_rate   = $2,
			last_sample_error = $3,
			active_tasks      = $4,
			uptime_seconds    = $5,
			current_task      = $6,
			updated_at        = now()
		WHERE id = $1 AND status != 'removed'
	`, payload.WorkspaceID, payload.ErrorRate, payload.SampleError,
		payload.ActiveTasks, payload.UptimeSeconds, payload.CurrentTask)
	if err != nil {
		log.Printf("Heartbeat update error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
		return
	}

	// Refresh Redis TTL
	if err := db.RefreshTTL(ctx, payload.WorkspaceID); err != nil {
		log.Printf("Heartbeat redis error: %v", err)
	}

	// Evaluate status transitions
	h.evaluateStatus(c, payload)

	// Broadcast current task update only when it changed (avoid spamming on every heartbeat)
	if payload.CurrentTask != prevTask {
		h.broadcaster.BroadcastOnly(payload.WorkspaceID, "TASK_UPDATED", map[string]interface{}{
			"current_task": payload.CurrentTask,
			"active_tasks": payload.ActiveTasks,
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *RegistryHandler) evaluateStatus(c *gin.Context, payload models.HeartbeatPayload) {
	ctx := c.Request.Context()

	var currentStatus string
	err := db.DB.QueryRowContext(ctx, `SELECT status FROM workspaces WHERE id = $1`, payload.WorkspaceID).
		Scan(&currentStatus)
	if err != nil {
		return
	}

	if currentStatus == "online" && payload.ErrorRate >= 0.5 {
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'degraded', updated_at = now() WHERE id = $1`, payload.WorkspaceID); err != nil {
			log.Printf("Heartbeat: failed to mark %s degraded: %v", payload.WorkspaceID, err)
		}
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_DEGRADED", payload.WorkspaceID, map[string]interface{}{
			"error_rate":   payload.ErrorRate,
			"sample_error": payload.SampleError,
		})
	}

	if currentStatus == "degraded" && payload.ErrorRate < 0.1 {
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'online', updated_at = now() WHERE id = $1`, payload.WorkspaceID); err != nil {
			log.Printf("Heartbeat: failed to recover %s to online: %v", payload.WorkspaceID, err)
		}
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_ONLINE", payload.WorkspaceID, map[string]interface{}{})
	}

	// Recovery: if workspace was offline but is now sending heartbeats, bring it back online.
	// #73 guard: `AND status = 'offline'` makes the flip conditional in a single statement,
	// so a Delete that races with this recovery can't flip 'removed' back to 'online'.
	if currentStatus == "offline" {
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'online', updated_at = now() WHERE id = $1 AND status = 'offline'`, payload.WorkspaceID); err != nil {
			log.Printf("Heartbeat: failed to recover %s from offline: %v", payload.WorkspaceID, err)
		}
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_ONLINE", payload.WorkspaceID, map[string]interface{}{})
	}
}

// UpdateCard handles POST /registry/update-card
func (h *RegistryHandler) UpdateCard(c *gin.Context) {
	var payload models.UpdateCardPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Phase 30.1 — same bootstrap-aware token gate as Heartbeat.
	if err := h.requireWorkspaceToken(c.Request.Context(), c, payload.WorkspaceID); err != nil {
		return // response already written
	}

	agentCardStr := string(payload.AgentCard)
	_, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE workspaces SET agent_card = $2::jsonb, updated_at = now() WHERE id = $1
	`, payload.WorkspaceID, agentCardStr)
	if err != nil {
		log.Printf("UpdateCard error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update card"})
		return
	}

	h.broadcaster.RecordAndBroadcast(c.Request.Context(), "AGENT_CARD_UPDATED", payload.WorkspaceID, map[string]interface{}{
		"agent_card": payload.AgentCard,
	})

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// requireWorkspaceToken enforces the Phase 30.1 auth-token contract on an
// inbound registry request (heartbeat / update-card today).
//
// The function has two distinct behaviours gated on whether the workspace
// has any live tokens on file:
//
//   - workspace has at least one live token → Authorization: Bearer <token>
//     is mandatory. Missing / malformed / wrong-workspace → 401.
//   - workspace has zero live tokens → grandfathered. We let the request
//     through and log a single DEBUG line. The agent's next
//     /registry/register call will mint its first token, after which this
//     branch never fires again for that workspace.
//
// Returns a non-nil error (and writes the 401 response via c) when the
// caller should abort. A nil return means the handler may continue.
//
// SECURITY NOTE: the grandfathering path is only safe during the
// transition window. Once every running workspace has re-registered
// post-upgrade, step 30.5 flips this to hard-require.
func (h *RegistryHandler) requireWorkspaceToken(
	ctx gincontext, c *gin.Context, workspaceID string,
) error {
	hasLive, err := wsauth.HasAnyLiveToken(ctx, db.DB, workspaceID)
	if err != nil {
		// DB error checking token existence — fail open so we don't take
		// the whole heartbeat path down on a transient hiccup. Log loudly.
		log.Printf("wsauth: HasAnyLiveToken(%s) failed: %v — allowing request", workspaceID, err)
		return nil
	}
	if !hasLive {
		// Legacy / pre-upgrade workspace. Next register issues a token.
		return nil
	}
	token := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing workspace auth token"})
		return errors.New("missing token")
	}
	if err := wsauth.ValidateToken(ctx, db.DB, workspaceID, token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid workspace auth token"})
		return err
	}
	return nil
}

// gincontext is an alias for context.Context kept separate so callers can
// see "gin.Context.Request.Context() is what we want" without re-typing
// the import-heavy standard type.
type gincontext = context.Context
