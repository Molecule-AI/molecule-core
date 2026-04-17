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

var slackHTTPClient = &http.Client{Timeout: slackHTTPTimeout}

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
	botToken, _ := config["bot_token"].(string)
	webhookURL, _ := config["webhook_url"].(string)
	if botToken == "" && webhookURL == "" {
		return fmt.Errorf("missing required field: bot_token or webhook_url")
	}
	if botToken != "" {
		if cid, _ := config["channel_id"].(string); cid == "" {
			return fmt.Errorf("bot_token mode requires channel_id")
		}
	}
	if webhookURL != "" && !strings.HasPrefix(webhookURL, slackWebhookPrefix) {
		return fmt.Errorf("invalid Slack webhook URL")
	}
	return nil
}

// SendMessage posts text to Slack. Supports two modes:
//
//   - Bot API (bot_token set): uses chat.postMessage with per-agent identity
//     via chat:write.customize scope. Supports username + icon_emoji overrides.
//   - Webhook (webhook_url set, legacy): simple POST, no identity override.
//
// chatID overrides channel_id from config if non-empty (for multi-channel routing).
func (s *SlackAdapter) SendMessage(ctx context.Context, config map[string]interface{}, chatID string, text string) error {
	botToken, _ := config["bot_token"].(string)
	if botToken != "" {
		return s.sendBotMessage(ctx, config, chatID, text)
	}
	return s.sendWebhookMessage(ctx, config, text)
}

func (s *SlackAdapter) sendBotMessage(ctx context.Context, config map[string]interface{}, chatID, text string) error {
	botToken, _ := config["bot_token"].(string)
	channelID := chatID
	if channelID == "" {
		channelID, _ = config["channel_id"].(string)
	}
	if channelID == "" {
		return fmt.Errorf("slack: no channel_id")
	}

	username, _ := config["username"].(string)
	iconEmoji, _ := config["icon_emoji"].(string)

	// Convert Markdown → Slack mrkdwn before sending
	text = markdownToMrkdwn(text)

	// Split long messages at newline boundaries
	chunks := slackSplitMessage(text, 3000)
	for _, chunk := range chunks {
		payload := map[string]interface{}{
			"channel": channelID,
			"text":    chunk,
		}
		if username != "" {
			payload["username"] = username
		}
		if iconEmoji != "" {
			payload["icon_emoji"] = iconEmoji
		}

		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("slack: build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Authorization", "Bearer "+botToken)

		resp, err := slackHTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("slack: send: %w", err)
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		var result struct {
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &result) == nil && !result.OK {
			return fmt.Errorf("slack: API error: %s", result.Error)
		}
	}
	return nil
}

func (s *SlackAdapter) sendWebhookMessage(ctx context.Context, config map[string]interface{}, text string) error {
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

	resp, err := slackHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack: send: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack: webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// markdownToMrkdwn converts standard Markdown to Slack's mrkdwn format.
// Agents output standard MD (Claude Code default); Slack renders mrkdwn.
//
//	MD **bold** → mrkdwn *bold*
//	MD __italic__ or *italic* (standalone) → mrkdwn _italic_
//	MD ### heading → mrkdwn *heading* (bold, no heading syntax in Slack)
//	MD [text](url) → mrkdwn <url|text>
//	MD --- → mrkdwn ———
//	MD > quote → mrkdwn > quote (same, works as-is)
//	MD `code` → mrkdwn `code` (same)
//	MD ```block``` → mrkdwn ```block``` (same)
func markdownToMrkdwn(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Headings: ### Text → *Text*
		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimLeft(trimmed, "# ")
			if heading != "" {
				lines[i] = "*" + heading + "*"
				continue
			}
		}

		// Horizontal rules
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			lines[i] = "———"
			continue
		}

		// Links: [text](url) → <url|text>
		for {
			start := strings.Index(lines[i], "[")
			if start < 0 {
				break
			}
			mid := strings.Index(lines[i][start:], "](")
			if mid < 0 {
				break
			}
			mid += start
			end := strings.Index(lines[i][mid+2:], ")")
			if end < 0 {
				break
			}
			end += mid + 2
			linkText := lines[i][start+1 : mid]
			url := lines[i][mid+2 : end]
			lines[i] = lines[i][:start] + "<" + url + "|" + linkText + ">" + lines[i][end+1:]
		}

		// Bold: **text** → *text* (Slack bold is single asterisk)
		for strings.Contains(lines[i], "**") {
			first := strings.Index(lines[i], "**")
			second := strings.Index(lines[i][first+2:], "**")
			if second < 0 {
				break
			}
			second += first + 2
			inner := lines[i][first+2 : second]
			lines[i] = lines[i][:first] + "*" + inner + "*" + lines[i][second+2:]
		}
	}
	return strings.Join(lines, "\n")
}

func slackSplitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}
	var chunks []string
	for len(text) > 0 {
		end := maxLen
		if end > len(text) {
			end = len(text)
		}
		if end < len(text) {
			if idx := strings.LastIndex(text[:end], "\n"); idx > 0 {
				end = idx + 1
			}
		}
		chunks = append(chunks, text[:end])
		text = text[end:]
	}
	return chunks
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
			Challenge string `json:"challenge"`
			Event     struct {
				Type    string `json:"type"`
				User    string `json:"user"`
				Text    string `json:"text"`
				Channel string `json:"channel"`
				Ts      string `json:"ts"`
				BotID   string `json:"bot_id"`
				Subtype string `json:"subtype"`
			} `json:"event"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("slack: parse event: %w", err)
		}

		// url_verification handshake — respond with challenge directly
		if payload.Type == "url_verification" {
			c.JSON(200, gin.H{"challenge": payload.Challenge})
			return nil, nil
		}

		// Ignore bot messages to prevent echo loops. Our own auto-posts
		// via chat.postMessage fire Events API callbacks with bot_id set.
		if payload.Event.BotID != "" || payload.Event.Subtype == "bot_message" {
			return nil, nil
		}
		if payload.Event.Type != "message" || payload.Event.Text == "" {
			return nil, nil
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
