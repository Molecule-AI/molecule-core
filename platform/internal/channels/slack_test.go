package channels

import (
	"context"
	"strings"
	"testing"
)

func TestSlackSplitMessage_Short(t *testing.T) {
	chunks := slackSplitMessage("hello", 3000)
	if len(chunks) != 1 || chunks[0] != "hello" {
		t.Errorf("expected 1 chunk 'hello', got %v", chunks)
	}
}

func TestSlackSplitMessage_Long(t *testing.T) {
	long := strings.Repeat("a", 6000)
	chunks := slackSplitMessage(long, 3000)
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
	for _, c := range chunks {
		if len(c) > 3000 {
			t.Errorf("chunk exceeds max: %d", len(c))
		}
	}
}

func TestSlackSplitMessage_SplitAtNewline(t *testing.T) {
	text := strings.Repeat("x", 2900) + "\n" + strings.Repeat("y", 200)
	chunks := slackSplitMessage(text, 3000)
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
	if !strings.HasSuffix(chunks[0], "\n") {
		t.Error("first chunk should end at newline boundary")
	}
}

func TestSlackValidateConfig_BotToken(t *testing.T) {
	a := &SlackAdapter{}
	err := a.ValidateConfig(map[string]interface{}{
		"bot_token":  "xoxb-test",
		"channel_id": "C123",
	})
	if err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestSlackValidateConfig_BotTokenMissingChannel(t *testing.T) {
	a := &SlackAdapter{}
	err := a.ValidateConfig(map[string]interface{}{
		"bot_token": "xoxb-test",
	})
	if err == nil {
		t.Error("expected error for missing channel_id")
	}
}

func TestSlackValidateConfig_WebhookURL(t *testing.T) {
	a := &SlackAdapter{}
	err := a.ValidateConfig(map[string]interface{}{
		"webhook_url": "https://hooks.slack.com/services/T000/B000/xxx",
	})
	if err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestSlackValidateConfig_InvalidWebhook(t *testing.T) {
	a := &SlackAdapter{}
	err := a.ValidateConfig(map[string]interface{}{
		"webhook_url": "https://evil.com/steal",
	})
	if err == nil {
		t.Error("expected error for invalid webhook URL")
	}
}

func TestSlackValidateConfig_NeitherSet(t *testing.T) {
	a := &SlackAdapter{}
	err := a.ValidateConfig(map[string]interface{}{})
	if err == nil {
		t.Error("expected error when neither bot_token nor webhook_url set")
	}
}

func TestFetchChannelHistory_EmptyToken(t *testing.T) {
	msgs, err := FetchChannelHistory(context.Background(), "", "C123", 10)
	if err != nil || msgs != nil {
		t.Errorf("expected nil,nil for empty token, got %v,%v", msgs, err)
	}
}

func TestFetchChannelHistory_EmptyChannel(t *testing.T) {
	msgs, err := FetchChannelHistory(context.Background(), "xoxb-test", "", 10)
	if err != nil || msgs != nil {
		t.Errorf("expected nil,nil for empty channel, got %v,%v", msgs, err)
	}
}

func TestSlackAdapter_Type(t *testing.T) {
	a := &SlackAdapter{}
	if a.Type() != "slack" {
		t.Errorf("expected 'slack', got %q", a.Type())
	}
}

func TestSlackAdapter_DisplayName(t *testing.T) {
	a := &SlackAdapter{}
	if a.DisplayName() != "Slack" {
		t.Errorf("expected 'Slack', got %q", a.DisplayName())
	}
}
