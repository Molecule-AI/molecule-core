// Package channels provides a pluggable adapter system for social channel
// integrations (Telegram, Slack, Discord, etc.). Each platform implements
// the ChannelAdapter interface and registers itself in the adapter registry.
package channels

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ChannelAdapter is the interface every social channel must implement.
type ChannelAdapter interface {
	// Type returns the channel type identifier (e.g. "telegram", "slack").
	Type() string

	// DisplayName returns the human-readable name (e.g. "Telegram").
	DisplayName() string

	// ValidateConfig checks that channel_config JSONB has required fields.
	ValidateConfig(config map[string]interface{}) error

	// SendMessage sends a text message to the social platform.
	SendMessage(ctx context.Context, config map[string]interface{}, chatID string, text string) error

	// ParseWebhook extracts message info from an incoming webhook request.
	ParseWebhook(c *gin.Context, config map[string]interface{}) (*InboundMessage, error)

	// StartPolling begins long-polling for platforms that support it.
	// Returns nil immediately if the platform only supports webhooks.
	StartPolling(ctx context.Context, config map[string]interface{}, onMessage MessageHandler) error
}

// InboundMessage is the standardized message from any social platform.
type InboundMessage struct {
	ChatID    string            // Platform-specific chat/channel ID
	UserID    string            // Platform-specific user ID
	Username  string            // Human-readable username
	Text      string            // Message text
	MessageID string            // Platform-specific message ID (for threading)
	Metadata  map[string]string // Extra platform-specific data
}

// MessageHandler is called by polling adapters when a message arrives.
type MessageHandler func(ctx context.Context, channelID string, msg *InboundMessage) error

// ChannelRow represents a row from the workspace_channels table.
type ChannelRow struct {
	ID           string
	WorkspaceID  string
	ChannelType  string
	Config       map[string]interface{}
	Enabled      bool
	AllowedUsers []string
}
