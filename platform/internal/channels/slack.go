package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	slackWebhookPrefix = "https://hooks.slack.com/"
	slackHTTPTimeout   = 10 * time.Second
)

// SlackAdapter implements ChannelAdapter for Slack Incoming Webhooks.
//
// Outbound messages are sent via Slack Incoming Webhooks (the simple,
// no-OAuth path). Inbound messages require Slack Event API / slash command
// configuration on the Slack App side; ParseWebhook handles the JSON payload
// that Slack POSTs to the registered webhook URL.
type SlackAdapter struct{}

func (s *SlackAdapter) Type() string        { return "slack" }
func (s *SlackAdapter) DisplayName() string { return "Slack" }

// ValidateConfig checks that the channel config contains a valid Slack
// Incoming Webhook URL (must start with https://hooks.slack.com/).
// Returns an error whose message becomes part of the 400 response body so
// keep it human-readable for the canvas UI.
func (s *SlackAdapter) ValidateConfig(config map[string]interface{}) error {
	webhookURL, _ := config["webhook_url"].(string)
	if webhookURL == "" {
		return fmt.Errorf("missing required field: webhook_url")
	}
	if !strings.HasPrefix(webhookURL, slackWebhookPrefix) {
		return fmt.Errorf("invalid Slack webhook URL")
	}
	return nil
}

// SendMessage posts text to the configured Slack Incoming Webhook.
// chatID is ignored for Slack webhooks — the channel is encoded in the URL.
func (s *SlackAdapter) SendMessage(ctx context.Context, config map[string]interface{}, _ string, text string) error {
	webhookURL, _ := config["webhook_url"].(string)
	if webhookURL == "" {
		return fmt.Errorf("webhook_url not configured")
	}
	if !strings.HasPrefix(webhookURL, slackWebhookPrefix) {
		return fmt.Errorf("invalid Slack webhook URL")
	}

	payload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("slack: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: slackHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("slack: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack: webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// ParseWebhook handles a Slack slash command or event API POST.
// The payload is either URL-encoded (slash commands) or JSON (Events API).
// Returns nil, nil for non-message events (e.g. url_verification challenge).
func (s *SlackAdapter) ParseWebhook(c *gin.Context, _ map[string]interface{}) (*InboundMessage, error) {
	contentType := c.GetHeader("Content-Type")

	var text, userID, username, channelID, msgID string

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// Slack slash command payload
		if err := c.Request.ParseForm(); err != nil {
			return nil, fmt.Errorf("slack: parse form: %w", err)
		}
		text = c.Request.FormValue("text")
		userID = c.Request.FormValue("user_id")
		username = c.Request.FormValue("user_name")
		channelID = c.Request.FormValue("channel_id")
		msgID = c.Request.FormValue("trigger_id")
		// Slash command: prepend the command itself so agent sees the full invocation
		if cmd := c.Request.FormValue("command"); cmd != "" {
			text = cmd + " " + text
		}
	} else {
		// Slack Events API JSON payload
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("slack: read body: %w", err)
		}

		var payload struct {
			Type      string `json:"type"`
			Challenge string `json:"challenge"` // url_verification
			Event     struct {
				Type    string `json:"type"`
				User    string `json:"user"`
				Text    string `json:"text"`
				Channel string `json:"channel"`
				Ts      string `json:"ts"`
			} `json:"event"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("slack: parse event: %w", err)
		}

		// url_verification handshake — no message, respond via the handler layer
		if payload.Type == "url_verification" {
			log.Printf("Channels: Slack url_verification challenge (not handled by ParseWebhook)")
			return nil, nil
		}

		if payload.Event.Type != "message" || payload.Event.Text == "" {
			return nil, nil // Ignore non-message events
		}

		text = payload.Event.Text
		userID = payload.Event.User
		channelID = payload.Event.Channel
		msgID = payload.Event.Ts
	}

	if text == "" {
		return nil, nil
	}

	return &InboundMessage{
		ChatID:    channelID,
		UserID:    userID,
		Username:  username,
		Text:      text,
		MessageID: msgID,
		Metadata:  map[string]string{"platform": "slack"},
	}, nil
}

// StartPolling returns nil immediately. Slack does not support long-polling
// for Incoming Webhooks — use the Slack Events API + webhook route instead.
func (s *SlackAdapter) StartPolling(_ context.Context, _ map[string]interface{}, _ MessageHandler) error {
	return nil
}
