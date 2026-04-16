package handlers

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/channels"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
)

// ChannelHandler manages workspace social channel integrations.
type ChannelHandler struct {
	manager *channels.Manager
}

// NewChannelHandler creates a channel handler with the given manager.
func NewChannelHandler(manager *channels.Manager) *ChannelHandler {
	return &ChannelHandler{manager: manager}
}

// ListAdapters returns all available channel adapter types.
func (h *ChannelHandler) ListAdapters(c *gin.Context) {
	c.JSON(http.StatusOK, channels.ListAdapters())
}

// List returns all channels for a workspace.
func (h *ChannelHandler) List(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, workspace_id, channel_type, channel_config, enabled, allowed_users,
		       last_message_at, message_count, created_at, updated_at
		FROM workspace_channels WHERE workspace_id = $1
		ORDER BY created_at
	`, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, wsID, chType string
		var configJSON, allowedJSON []byte
		var enabled bool
		var lastMsg sql.NullTime
		var msgCount int
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(&id, &wsID, &chType, &configJSON, &enabled, &allowedJSON, &lastMsg, &msgCount, &createdAt, &updatedAt); err != nil {
			continue
		}

		var config map[string]interface{}
		json.Unmarshal(configJSON, &config)
		// #319: decrypt sensitive fields first so the mask operates on
		// plaintext (first-4 / last-4 of the real token, not the ciphertext
		// prefix). Decrypt errors are logged but non-fatal — List must keep
		// returning the rest of the channel even if one field is corrupt.
		if err := channels.DecryptSensitiveFields(config); err != nil {
			log.Printf("Channels: decrypt config on list for channel %s: %v", id, err)
		}
		// Mask bot_token in list response
		if _, ok := config["bot_token"]; ok {
			token, _ := config["bot_token"].(string)
			if len(token) > 8 {
				config["bot_token"] = token[:4] + "..." + token[len(token)-4:]
			} else {
				config["bot_token"] = "***"
			}
		}

		var allowed []string
		json.Unmarshal(allowedJSON, &allowed)

		entry := map[string]interface{}{
			"id":            id,
			"workspace_id":  wsID,
			"channel_type":  chType,
			"config":        config,
			"enabled":       enabled,
			"allowed_users": allowed,
			"message_count": msgCount,
			"created_at":    createdAt.Time,
			"updated_at":    updatedAt.Time,
		}
		if lastMsg.Valid {
			entry["last_message_at"] = lastMsg.Time
		}
		result = append(result, entry)
	}

	c.JSON(http.StatusOK, result)
}

// Create adds a new channel to a workspace.
func (h *ChannelHandler) Create(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	var body struct {
		ChannelType  string                 `json:"channel_type"`
		Config       map[string]interface{} `json:"config"`
		AllowedUsers []string               `json:"allowed_users"`
		Enabled      *bool                  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	if body.ChannelType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_type is required"})
		return
	}

	adapter, ok := channels.GetAdapter(body.ChannelType)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported channel_type: " + body.ChannelType})
		return
	}

	if err := adapter.ValidateConfig(body.Config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config: " + err.Error()})
		return
	}

	// #319: encrypt sensitive fields (bot_token, webhook_secret) before
	// persisting so a DB read/backup leak can't recover the credentials.
	// Validation above ran against plaintext; storage is ciphertext.
	if err := channels.EncryptSensitiveFields(body.Config); err != nil {
		log.Printf("Channels: encrypt config failed for workspace %s: %v", workspaceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt failed"})
		return
	}

	configJSON, _ := json.Marshal(body.Config)
	allowedJSON, _ := json.Marshal(body.AllowedUsers)
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	var id string
	err := db.DB.QueryRowContext(ctx, `
		INSERT INTO workspace_channels (workspace_id, channel_type, channel_config, enabled, allowed_users)
		VALUES ($1, $2, $3::jsonb, $4, $5::jsonb)
		RETURNING id
	`, workspaceID, body.ChannelType, string(configJSON), enabled, string(allowedJSON)).Scan(&id)
	if err != nil {
		log.Printf("Channels: create failed for workspace %s: %v", workspaceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel"})
		return
	}

	// Hot reload
	h.manager.Reload(ctx)

	c.JSON(http.StatusCreated, gin.H{
		"id":           id,
		"channel_type": body.ChannelType,
		"enabled":      enabled,
	})
}

// Update modifies a channel's config, allowlist, or enabled state.
func (h *ChannelHandler) Update(c *gin.Context) {
	workspaceID := c.Param("id")
	channelID := c.Param("channelId")
	ctx := c.Request.Context()

	var body struct {
		Config       map[string]interface{} `json:"config"`
		AllowedUsers []string               `json:"allowed_users"`
		Enabled      *bool                  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// COALESCE-based update
	var configArg, allowedArg interface{}
	if body.Config != nil {
		// #319: re-encrypt sensitive fields on every config update — the
		// PATCH body carries plaintext (client already had them plaintext in
		// List response's unmasked path or typed fresh).
		if err := channels.EncryptSensitiveFields(body.Config); err != nil {
			log.Printf("Channels: encrypt update for workspace %s: %v", workspaceID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt failed"})
			return
		}
		j, _ := json.Marshal(body.Config)
		configArg = string(j)
	}
	if body.AllowedUsers != nil {
		j, _ := json.Marshal(body.AllowedUsers)
		allowedArg = string(j)
	}

	result, err := db.DB.ExecContext(ctx, `
		UPDATE workspace_channels
		SET channel_config = COALESCE($3::jsonb, channel_config),
		    allowed_users = COALESCE($4::jsonb, allowed_users),
		    enabled = COALESCE($5, enabled),
		    updated_at = now()
		WHERE id = $1 AND workspace_id = $2
	`, channelID, workspaceID, configArg, allowedArg, body.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	if n, _ := result.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Hot reload
	h.manager.Reload(ctx)

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// Delete removes a channel from a workspace.
func (h *ChannelHandler) Delete(c *gin.Context) {
	workspaceID := c.Param("id")
	channelID := c.Param("channelId")
	ctx := c.Request.Context()

	result, err := db.DB.ExecContext(ctx, `
		DELETE FROM workspace_channels WHERE id = $1 AND workspace_id = $2
	`, channelID, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}

	if n, _ := result.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	// Hot reload
	h.manager.Reload(ctx)

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Send sends an outbound message from a workspace to its social channel.
func (h *ChannelHandler) Send(c *gin.Context) {
	channelID := c.Param("channelId")
	ctx := c.Request.Context()

	var body struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	if err := h.manager.SendOutbound(ctx, channelID, body.Text); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

// Test sends a test message to verify the channel is working.
func (h *ChannelHandler) Test(c *gin.Context) {
	channelID := c.Param("channelId")
	ctx := c.Request.Context()

	if err := h.manager.SendOutbound(ctx, channelID, "🔔 Molecule AI channel test — connection successful!"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "test message sent"})
}

// matchesChatID returns true if chatID is in the channel's comma-separated chat_id list.
// Exact match — avoids the substring-match bug of SQL LIKE.
func matchesChatID(config map[string]interface{}, chatID string) bool {
	raw, _ := config["chat_id"].(string)
	if raw == "" {
		return false
	}
	for _, s := range strings.Split(raw, ",") {
		if strings.TrimSpace(s) == chatID {
			return true
		}
	}
	return false
}

// Discover auto-detects chats/groups a bot has been added to by calling the platform API.
// User flow: enter bot token → add bot to groups → send a message → click Detect → select groups.
func (h *ChannelHandler) Discover(c *gin.Context) {
	var body struct {
		ChannelType string `json:"channel_type"`
		BotToken    string `json:"bot_token"`
		WorkspaceID string `json:"workspace_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.BotToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bot_token is required"})
		return
	}
	// #329: workspace_id is required so PausePollersForToken can scope the
	// decryption lookup to the caller's tenant. Legacy clients that omit
	// the field still work — they just won't be able to pause a previously-
	// saved poller sharing the same token, which fails loudly at Telegram
	// with a 409 Conflict rather than silently decrypting every tenant's
	// token.
	if body.WorkspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id is required"})
		return
	}

	adapter, ok := channels.GetAdapter(body.ChannelType)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported channel_type"})
		return
	}

	// Only Telegram supports discovery currently
	tg, ok := adapter.(*channels.TelegramAdapter)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "discovery not supported for " + body.ChannelType})
		return
	}

	// Pause any active poller in THIS workspace using this bot token to
	// avoid Telegram's "only one getUpdates at a time" 409 Conflict.
	// #329: scoped to workspace_id so we never decrypt other tenants' tokens.
	resumeFn := h.manager.PausePollersForToken(body.WorkspaceID, body.BotToken)
	defer resumeFn()

	result, err := tg.DiscoverChats(c.Request.Context(), body.BotToken)
	if err != nil {
		// Map known errors to user-friendly messages
		msg := err.Error()
		userMsg := "Failed to connect to Telegram. Check your bot token and try again."
		if strings.Contains(msg, "invalid bot token") || strings.Contains(msg, "Unauthorized") || strings.Contains(msg, "Not Found") {
			userMsg = "Invalid bot token. Check the token from @BotFather and try again."
		} else if strings.Contains(msg, "Conflict") || strings.Contains(msg, "terminated by other") {
			userMsg = "This bot is already connected to another channel. Disconnect the existing channel first, or wait 30 seconds and retry."
		} else if strings.Contains(msg, "no route to host") || strings.Contains(msg, "i/o timeout") {
			userMsg = "Cannot reach Telegram API. Check your network connection and try again."
		}
		log.Printf("Channels: discover error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": userMsg})
		return
	}

	hint := "For groups: add bot and send a message. For DMs: send /start to the bot. Then retry."
	var warning string
	if !result.CanReadAllGroupMessages {
		warning = "⚠️ Group privacy mode is ON for this bot — it only sees commands and @mentions in groups. " +
			"To let it see all group messages: open @BotFather → /mybots → " + result.BotUsername +
			" → Bot Settings → Group Privacy → Turn off, then re-add the bot to the group."
	}

	c.JSON(http.StatusOK, gin.H{
		"bot_username":                  result.BotUsername,
		"chats":                         result.Chats,
		"can_read_all_group_messages":   result.CanReadAllGroupMessages,
		"privacy_warning":               warning,
		"hint":                          hint,
	})
}

// Webhook handles incoming webhooks from any social platform.
func (h *ChannelHandler) Webhook(c *gin.Context) {
	channelType := c.Param("type")
	ctx := c.Request.Context()

	adapter, ok := channels.GetAdapter(channelType)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown channel type"})
		return
	}

	// For webhooks, we need to find the channel by type and match by chat_id in the message
	// Parse the webhook first to get the chat_id
	msg, err := adapter.ParseWebhook(c, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parse error: " + err.Error()})
		return
	}
	if msg == nil {
		c.JSON(http.StatusOK, gin.H{"status": "ignored"}) // Non-message update
		return
	}

	// Look up channels by type and find one whose chat_id list contains msg.ChatID.
	// We can't use SQL LIKE — that matches substrings (chat_id "123" would match "1234").
	// Fetch all enabled channels of this type, then exact-match in code.
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, workspace_id, channel_type, channel_config, enabled, allowed_users
		FROM workspace_channels
		WHERE channel_type = $1 AND enabled = true
	`, channelType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "channel lookup failed"})
		return
	}
	defer rows.Close()

	var ch channels.ChannelRow
	found := false
	for rows.Next() {
		var row channels.ChannelRow
		var configJSON, allowedJSON []byte
		if err := rows.Scan(&row.ID, &row.WorkspaceID, &row.ChannelType, &configJSON, &row.Enabled, &allowedJSON); err != nil {
			continue
		}
		json.Unmarshal(configJSON, &row.Config)
		json.Unmarshal(allowedJSON, &row.AllowedUsers)
		// #319: decrypt sensitive fields before comparing webhook_secret /
		// using bot_token downstream. Skip rows whose decrypt fails so a
		// single corrupt channel cannot block webhooks for all others.
		if err := channels.DecryptSensitiveFields(row.Config); err != nil {
			log.Printf("Channels: decrypt webhook row %s: %v", row.ID, err)
			continue
		}

		// Verify webhook secret_token if the channel has one configured.
		// #337: use constant-time comparison. Go's `!=` short-circuits on
		// the first mismatched byte and leaks timing information; an
		// attacker on the Docker network could enumerate the secret
		// byte-by-byte. subtle.ConstantTimeCompare runs in time
		// proportional to the length of the shorter input and returns
		// 1 on match / 0 otherwise (never -1). Same posture as the
		// cdp-proxy token compare in host-bridge.
		if expectedSecret, _ := row.Config["webhook_secret"].(string); expectedSecret != "" {
			receivedSecret := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
			if subtle.ConstantTimeCompare([]byte(receivedSecret), []byte(expectedSecret)) != 1 {
				continue // Wrong secret — try other channels (could be different bot)
			}
		}

		// Exact match against the comma-separated chat_id list
		if matchesChatID(row.Config, msg.ChatID) {
			ch = row
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusOK, gin.H{"status": "no_channel"})
		return
	}

	// Process asynchronously — don't block the webhook response
	go func() {
		bgCtx := context.Background()
		_ = h.manager.HandleInbound(bgCtx, ch, msg)
	}()

	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}
