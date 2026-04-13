package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/events"
	"github.com/gin-gonic/gin"
)

type WebhookHandler struct {
	workspaces *WorkspaceHandler
}

func NewWebhookHandler(broadcaster *events.Broadcaster) *WebhookHandler {
	return &WebhookHandler{
		workspaces: NewWorkspaceHandler(broadcaster, nil, "", ""),
	}
}

func NewWebhookHandlerWithWorkspace(workspaces *WorkspaceHandler) *WebhookHandler {
	return &WebhookHandler{
		workspaces: workspaces,
	}
}

// GitHub handles POST /webhooks/github/:id
// It verifies X-Hub-Signature-256, maps supported events to A2A message/send,
// then forwards through the same proxy flow used by /workspaces/:id/a2a.
func (h *WebhookHandler) GitHub(c *gin.Context) {
	workspaceID := c.Param("id")

	secret := strings.TrimSpace(os.Getenv("GITHUB_WEBHOOK_SECRET"))
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "github webhook secret is not configured"})
		return
	}

	rawBody, err := io.ReadAll(io.LimitReader(c.Request.Body, maxProxyRequestBody))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	if !verifyGitHubSignature(secret, rawBody, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")
	deliveryID := c.GetHeader("X-GitHub-Delivery")
	payloadWorkspaceID, a2aPayload, buildErr := buildGitHubA2APayload(eventType, deliveryID, rawBody)
	if buildErr != nil {
		if buildErr == errUnsupportedGitHubEvent || buildErr == errIgnoredGitHubAction {
			c.JSON(http.StatusAccepted, gin.H{"status": "ignored", "reason": "unsupported event type"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": buildErr.Error()})
		return
	}
	if workspaceID == "" {
		workspaceID = payloadWorkspaceID
	}
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing workspace id"})
		return
	}

	forwardBody, err := json.Marshal(a2aPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal a2a payload"})
		return
	}

	status, respBody, proxyErr := h.workspaces.proxyA2ARequest(
		c.Request.Context(),
		workspaceID,
		forwardBody,
		"webhook:github",
		true,
	)
	if proxyErr != nil {
		c.JSON(proxyErr.Status, proxyErr.Response)
		return
	}

	c.Data(status, "application/json", respBody)
}

var errUnsupportedGitHubEvent = fmt.Errorf("unsupported github event")
var errIgnoredGitHubAction = fmt.Errorf("ignored github action")

func verifyGitHubSignature(secret string, body []byte, header string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}

	gotHex := strings.TrimPrefix(header, prefix)
	got, err := hex.DecodeString(gotHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(got, expected)
}

type githubRepository struct {
	FullName string `json:"full_name"`
}

type githubSender struct {
	Login string `json:"login"`
}

type githubComment struct {
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

type githubIssue struct {
	Number int `json:"number"`
}

type githubPullRequest struct {
	Number int `json:"number"`
}

type githubIssueCommentEvent struct {
	WorkspaceID string           `json:"workspace_id"`
	Action      string           `json:"action"`
	Repository  githubRepository `json:"repository"`
	Sender      githubSender     `json:"sender"`
	Issue       githubIssue      `json:"issue"`
	Comment     githubComment    `json:"comment"`
}

type githubPRReviewCommentEvent struct {
	WorkspaceID string            `json:"workspace_id"`
	Action      string            `json:"action"`
	Repository  githubRepository  `json:"repository"`
	Sender      githubSender      `json:"sender"`
	PullRequest githubPullRequest `json:"pull_request"`
	Comment     githubComment     `json:"comment"`
}

func buildGitHubA2APayload(eventType, deliveryID string, rawBody []byte) (string, map[string]interface{}, error) {
	switch eventType {
	case "issue_comment":
		var payload githubIssueCommentEvent
		if err := json.Unmarshal(rawBody, &payload); err != nil {
			return "", nil, fmt.Errorf("invalid issue_comment payload: %w", err)
		}
		if payload.Action != "created" {
			return payload.WorkspaceID, nil, errIgnoredGitHubAction
		}
		text := fmt.Sprintf(
			"GitHub issue_comment event (%s) in %s issue #%d by %s:\n%s",
			payload.Action,
			payload.Repository.FullName,
			payload.Issue.Number,
			payload.Sender.Login,
			strings.TrimSpace(payload.Comment.Body),
		)
		return payload.WorkspaceID, newGitHubMessagePayload(text, map[string]interface{}{
			"source":       "github",
			"event":        eventType,
			"action":       payload.Action,
			"delivery_id":  deliveryID,
			"repository":   payload.Repository.FullName,
			"sender":       payload.Sender.Login,
			"issue_number": payload.Issue.Number,
			"comment_url":  payload.Comment.HTMLURL,
		}), nil
	case "pull_request_review_comment":
		var payload githubPRReviewCommentEvent
		if err := json.Unmarshal(rawBody, &payload); err != nil {
			return "", nil, fmt.Errorf("invalid pull_request_review_comment payload: %w", err)
		}
		if payload.Action != "created" {
			return payload.WorkspaceID, nil, errIgnoredGitHubAction
		}
		text := fmt.Sprintf(
			"GitHub pull_request_review_comment event (%s) in %s PR #%d by %s:\n%s",
			payload.Action,
			payload.Repository.FullName,
			payload.PullRequest.Number,
			payload.Sender.Login,
			strings.TrimSpace(payload.Comment.Body),
		)
		return payload.WorkspaceID, newGitHubMessagePayload(text, map[string]interface{}{
			"source":           "github",
			"event":            eventType,
			"action":           payload.Action,
			"delivery_id":      deliveryID,
			"repository":       payload.Repository.FullName,
			"sender":           payload.Sender.Login,
			"pull_request_num": payload.PullRequest.Number,
			"comment_url":      payload.Comment.HTMLURL,
		}), nil
	default:
		return "", nil, errUnsupportedGitHubEvent
	}
}

func newGitHubMessagePayload(text string, metadata map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"method": "message/send",
		"params": map[string]interface{}{
			"message": map[string]interface{}{
				"role": "user",
				"parts": []map[string]string{
					{"text": text},
				},
			},
			"metadata": metadata,
		},
	}
}
