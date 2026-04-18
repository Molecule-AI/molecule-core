package handlers

// Slack OAuth 2.0 install flow (issue #860).
//
// Endpoints:
//   GET /integrations/slack/install?workspace_id=X
//       → redirects the browser to Slack's authorize URL.
//   GET /integrations/slack/callback?code=Y&state=X
//       → exchanges the code for a bot token, upserts a workspace_channels row,
//         then redirects to the canvas.
//   GET /workspaces/:id/integrations/slack/conversations (WorkspaceAuth)
//       → proxies conversations.list to the bot token stored for this workspace,
//         returns a JSON array of SlackConversation objects for the channel picker.
//
// Credentials are read from env vars:
//   SLACK_CLIENT_ID      – App client ID
//   SLACK_CLIENT_SECRET  – App client secret
//   SLACK_CALLBACK_URL   – Full callback URL (default: PLATFORM_URL + /integrations/slack/callback)
//   CANVAS_URL           – Where to redirect after OAuth success (default: CORS_ORIGINS first entry)

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/channels"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
)

const (
	// slackBotScopes are the OAuth scopes requested when installing the Slack App.
	//   chat:write            – post messages
	//   chat:write.customize  – per-agent username + icon_emoji overrides
	//   channels:read         – list public channels for channel picker
	//   groups:read           – list private channels the bot is in
	//   im:read               – list DMs
	//   mpim:read             – list group DMs
	slackBotScopes = "chat:write,chat:write.customize,channels:read,groups:read,im:read,mpim:read"

	slackOAuthHTTPTimeout = 10 * time.Second

	// slackOAuthNonceTTL is how long a state nonce is valid after Install
	// redirects the browser to Slack.  10 minutes is generous for a human
	// OAuth flow but tight enough to limit the CSRF attack window.
	slackOAuthNonceTTL = 10 * time.Minute

	// slackOAuthNoncePrefix is the Redis key prefix for OAuth state nonces.
	slackOAuthNoncePrefix = "slack:oauth:nonce:"
)

var slackOAuthHTTPClient = &http.Client{Timeout: slackOAuthHTTPTimeout}

// slackOAuthAuthorizeURL and slackOAuthAccessURL are vars (not consts) so
// tests can point them at a local mock server without patching the handler.
var (
	slackOAuthAuthorizeURL = "https://slack.com/oauth/v2/authorize"
	slackOAuthAccessURL    = "https://slack.com/api/oauth.v2.access"
)

// SlackOAuthHandler handles the two-legged Slack OAuth install flow.
type SlackOAuthHandler struct {
	clientID     string
	clientSecret string
	callbackURL  string // full URL Slack will redirect back to
	canvasURL    string // where to send the browser after a successful install
	httpClient   *http.Client // injectable for tests; production uses slackOAuthHTTPClient
}

// NewSlackOAuthHandler reads credentials from env and constructs the handler.
// platformURL is the platform's public base URL (used to derive callbackURL
// when SLACK_CALLBACK_URL is not set explicitly).
func NewSlackOAuthHandler(platformURL string) *SlackOAuthHandler {
	clientID := os.Getenv("SLACK_CLIENT_ID")
	clientSecret := os.Getenv("SLACK_CLIENT_SECRET")

	callbackURL := os.Getenv("SLACK_CALLBACK_URL")
	if callbackURL == "" {
		callbackURL = strings.TrimRight(platformURL, "/") + "/integrations/slack/callback"
	}

	canvasURL := os.Getenv("CANVAS_URL")
	if canvasURL == "" {
		// Default to first CORS_ORIGINS entry (typically http://localhost:3000)
		if cors := os.Getenv("CORS_ORIGINS"); cors != "" {
			parts := strings.SplitN(cors, ",", 2)
			canvasURL = strings.TrimSpace(parts[0])
		}
	}
	if canvasURL == "" {
		canvasURL = "http://localhost:3000"
	}

	return &SlackOAuthHandler{
		clientID:     clientID,
		clientSecret: clientSecret,
		callbackURL:  callbackURL,
		canvasURL:    canvasURL,
		httpClient:   slackOAuthHTTPClient,
	}
}

// isConfigured returns true if the client ID and secret are non-empty.
func (h *SlackOAuthHandler) isConfigured() bool {
	return h.clientID != "" && h.clientSecret != ""
}

// Install redirects the browser to Slack's OAuth authorize page.
//
//	GET /integrations/slack/install?workspace_id=<uuid>
//
// A cryptographically-random 256-bit nonce is generated, stored in Redis
// (keyed by the nonce, valued by workspace_id, TTL 10 min), and sent as the
// OAuth state parameter.  The Callback handler looks the nonce up and deletes
// it atomically — so the workspace_id never appears as a predictable state
// value and replayed callbacks are rejected.
func (h *SlackOAuthHandler) Install(c *gin.Context) {
	if !h.isConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Slack OAuth is not configured on this platform (missing SLACK_CLIENT_ID / SLACK_CLIENT_SECRET)",
		})
		return
	}

	workspaceID := c.Query("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id query param is required"})
		return
	}

	// Generate a cryptographically-random nonce (256 bits) to use as the
	// OAuth state parameter.  Binding the nonce → workspace_id in Redis
	// provides CSRF protection without exposing the (predictable) workspace_id.
	nonce, err := generateSlackOAuthNonce()
	if err != nil {
		log.Printf("SlackOAuth: failed to generate nonce for workspace %s: %v", workspaceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate OAuth state token"})
		return
	}

	ctx := c.Request.Context()
	if err := db.RDB.Set(ctx, slackOAuthNoncePrefix+nonce, workspaceID, slackOAuthNonceTTL).Err(); err != nil {
		log.Printf("SlackOAuth: failed to store nonce for workspace %s: %v", workspaceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize OAuth state"})
		return
	}

	params := url.Values{}
	params.Set("client_id", h.clientID)
	params.Set("scope", slackBotScopes)
	params.Set("state", nonce)
	params.Set("redirect_uri", h.callbackURL)

	authorizeURL := slackOAuthAuthorizeURL + "?" + params.Encode()
	c.Redirect(http.StatusFound, authorizeURL)
}

// Callback handles Slack's redirect after user authorization.
//
//	GET /integrations/slack/callback?code=<oauth_code>&state=<nonce>
//
// state is a cryptographically-random nonce issued by Install and stored in
// Redis.  Callback validates and atomically deletes the nonce, extracting the
// workspace_id from its Redis value.  Unknown or expired nonces are rejected.
// On success it upserts a workspace_channels row (type="slack") with the bot
// token encrypted via channels.EncryptSensitiveFields, then redirects the
// browser to the canvas.  On error it redirects with an error query param so
// the canvas can surface a friendly message.
func (h *SlackOAuthHandler) Callback(c *gin.Context) {
	ctx := c.Request.Context()

	// Slack sends `error` when the user denies the install.
	if slackErr := c.Query("error"); slackErr != "" {
		log.Printf("SlackOAuth: user denied install: %s", slackErr)
		h.redirectToCanvas(c, "", "slack_install_denied")
		return
	}

	code := c.Query("code")
	nonce := c.Query("state")
	if code == "" || nonce == "" {
		h.redirectToCanvas(c, "", "missing_code_or_state")
		return
	}

	// Validate and consume the nonce atomically (GETDEL).  This rejects:
	//   • replayed callbacks (nonce already consumed)
	//   • expired nonces (TTL elapsed)
	//   • crafted callbacks with a known workspace_id as state (unknown key)
	workspaceID, err := db.RDB.GetDel(ctx, slackOAuthNoncePrefix+nonce).Result()
	if err == redis.Nil {
		log.Printf("SlackOAuth: unknown or expired state nonce")
		h.redirectToCanvas(c, "", "invalid_state")
		return
	}
	if err != nil {
		log.Printf("SlackOAuth: redis error validating state nonce: %v", err)
		h.redirectToCanvas(c, "", "state_lookup_failed")
		return
	}

	// Exchange the authorization code for a bot token.
	token, err := h.exchangeCode(ctx, code)
	if err != nil {
		log.Printf("SlackOAuth: token exchange failed for workspace %s: %v", workspaceID, err)
		h.redirectToCanvas(c, workspaceID, "token_exchange_failed")
		return
	}

	// Build the channel config that will be stored in workspace_channels.
	config := map[string]interface{}{
		"bot_token":  token.BotToken,
		"channel_id": token.IncomingWebhook.ChannelID,
		"team_name":  token.Team.Name,
		"team_id":    token.Team.ID,
		"app_id":     token.AppID,
	}

	// Encrypt sensitive fields (bot_token) before persistence.
	if err := channels.EncryptSensitiveFields(config); err != nil {
		log.Printf("SlackOAuth: encrypt config for workspace %s: %v", workspaceID, err)
		h.redirectToCanvas(c, workspaceID, "encrypt_failed")
		return
	}

	configJSON, _ := json.Marshal(config)

	// Upsert: one Slack channel per workspace.  If one already exists
	// (e.g. re-install after token rotation) update the config in-place.
	_, err = db.DB.ExecContext(ctx, `
		INSERT INTO workspace_channels (workspace_id, channel_type, channel_config, enabled)
		VALUES ($1, 'slack', $2::jsonb, true)
		ON CONFLICT (workspace_id, channel_type)
		DO UPDATE SET
			channel_config = EXCLUDED.channel_config,
			enabled        = true,
			updated_at     = now()
	`, workspaceID, string(configJSON))
	if err != nil {
		log.Printf("SlackOAuth: upsert channel for workspace %s: %v", workspaceID, err)
		h.redirectToCanvas(c, workspaceID, "db_error")
		return
	}

	log.Printf("SlackOAuth: installed Slack for workspace %s (team=%s)", workspaceID, token.Team.Name)
	h.redirectToCanvas(c, workspaceID, "")
}

// ListConversations proxies conversations.list to the Slack bot token stored
// for the caller's workspace.  Used by the canvas channel-picker.
//
//	GET /workspaces/:id/integrations/slack/conversations
//
// The WorkspaceAuth middleware ensures the caller's bearer token matches :id.
func (h *SlackOAuthHandler) ListConversations(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	// Load the Slack channel config for this workspace.
	var configJSON []byte
	err := db.DB.QueryRowContext(ctx, `
		SELECT channel_config
		FROM workspace_channels
		WHERE workspace_id = $1 AND channel_type = 'slack' AND enabled = true
		LIMIT 1
	`, workspaceID).Scan(&configJSON)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no Slack channel configured for this workspace"})
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "malformed channel config"})
		return
	}

	// Decrypt bot_token before use.
	if err := channels.DecryptSensitiveFields(config); err != nil {
		log.Printf("SlackOAuth: decrypt config for workspace %s: %v", workspaceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt channel credentials"})
		return
	}

	botToken, _ := config["bot_token"].(string)
	if botToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slack channel has no bot_token (webhook-only mode)"})
		return
	}

	convs, err := channels.ListConversations(ctx, botToken)
	if err != nil {
		log.Printf("SlackOAuth: conversations.list for workspace %s: %v", workspaceID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to list Slack channels: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, convs)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// generateSlackOAuthNonce returns a cryptographically-random hex string with
// 256 bits of entropy, suitable for use as an OAuth CSRF state parameter.
func generateSlackOAuthNonce() (string, error) {
	b := make([]byte, 32) // 32 bytes = 256 bits
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// slackOAuthTokenResponse is the payload returned by oauth.v2.access.
type slackOAuthTokenResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	AppID   string `json:"app_id"`
	BotToken string `json:"access_token"` // the xoxb-... bot token
	Team    struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
	// IncomingWebhook is populated when the oauth.incoming-webhook scope is
	// included; we include it as a convenience channel_id default.
	IncomingWebhook struct {
		Channel   string `json:"channel"`
		ChannelID string `json:"channel_id"`
	} `json:"incoming_webhook"`
}

// exchangeCode posts to oauth.v2.access and returns the parsed response.
func (h *SlackOAuthHandler) exchangeCode(ctx context.Context, code string) (*slackOAuthTokenResponse, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", h.clientID)
	form.Set("client_secret", h.clientSecret)
	form.Set("redirect_uri", h.callbackURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackOAuthAccessURL,
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST oauth.v2.access: %w", err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	resp.Body.Close()

	var token slackOAuthTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if !token.OK {
		return nil, fmt.Errorf("slack API error: %s", token.Error)
	}
	if token.BotToken == "" {
		return nil, fmt.Errorf("slack returned empty access_token")
	}
	return &token, nil
}

// redirectToCanvas sends the browser to the canvas, appending workspace_id and
// an optional error code as query params so the canvas can react appropriately.
// On success errCode is "".
func (h *SlackOAuthHandler) redirectToCanvas(c *gin.Context, workspaceID, errCode string) {
	target := strings.TrimRight(h.canvasURL, "/")
	if workspaceID != "" || errCode != "" {
		params := url.Values{}
		if workspaceID != "" {
			params.Set("workspace_id", workspaceID)
		}
		if errCode != "" {
			params.Set("slack_error", errCode)
		} else {
			params.Set("slack_connected", "1")
		}
		target += "?" + params.Encode()
	}
	c.Redirect(http.StatusFound, target)
}
