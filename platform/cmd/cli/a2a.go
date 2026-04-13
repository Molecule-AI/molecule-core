package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// A2A JSON-RPC types (Google A2A protocol).

type a2aRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type taskSendParams struct {
	ID      string     `json:"id"`
	Message a2aMessage `json:"message"`
}

type a2aMessage struct {
	Role  string    `json:"role"`
	Parts []a2aPart `json:"parts"`
}

type a2aPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type a2aTask struct {
	ID        string      `json:"id"`
	Status    taskStatus  `json:"status"`
	Artifacts []artifact  `json:"artifacts,omitempty"`
}

type taskStatus struct {
	State   string     `json:"state"` // submitted, working, completed, failed, canceled
	Message *a2aMessage `json:"message,omitempty"`
}

type artifact struct {
	Parts []a2aPart `json:"parts"`
}

type a2aResponse struct {
	JSONRPC string   `json:"jsonrpc"`
	ID      string   `json:"id"`
	Result  *a2aTask `json:"result,omitempty"`
	Error   *a2aError `json:"error,omitempty"`
}

type a2aError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// a2aClient sends tasks to an A2A agent.
type a2aClient struct {
	httpClient *http.Client
	agentURL   string
}

func newA2AClient(agentURL string) *a2aClient {
	return &a2aClient{
		agentURL: agentURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// SendTask sends a user message and returns the agent's text reply.
// Supports both blocking (tasks/send) and streaming (tasks/sendSubscribe via SSE).
func (c *a2aClient) SendTask(text string) (string, error) {
	taskID := uuid.New().String()
	reqBody := a2aRequest{
		JSONRPC: "2.0",
		ID:      uuid.New().String(),
		Method:  "tasks/send",
		Params: taskSendParams{
			ID: taskID,
			Message: a2aMessage{
				Role:  "user",
				Parts: []a2aPart{{Type: "text", Text: text}},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.agentURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("send task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("agent returned status %d: %s", resp.StatusCode, b)
	}

	var a2aResp a2aResponse
	if err := json.NewDecoder(resp.Body).Decode(&a2aResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if a2aResp.Error != nil {
		return "", fmt.Errorf("agent error %d: %s", a2aResp.Error.Code, a2aResp.Error.Message)
	}
	if a2aResp.Result == nil {
		return "", fmt.Errorf("empty result from agent")
	}

	return extractText(a2aResp.Result), nil
}

// SendTaskStreaming calls tasks/sendSubscribe and streams chunks to the provided
// writer. Returns the full concatenated text when done.
func (c *a2aClient) SendTaskStreaming(text string, chunk func(string)) (string, error) {
	taskID := uuid.New().String()
	reqBody := a2aRequest{
		JSONRPC: "2.0",
		ID:      uuid.New().String(),
		Method:  "tasks/sendSubscribe",
		Params: taskSendParams{
			ID: taskID,
			Message: a2aMessage{
				Role:  "user",
				Parts: []a2aPart{{Type: "text", Text: text}},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.agentURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("send subscribe: %w", err)
	}
	defer resp.Body.Close()

	// Fall back to blocking if agent doesn't support streaming
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		var a2aResp a2aResponse
		if err := json.NewDecoder(resp.Body).Decode(&a2aResp); err != nil {
			return "", fmt.Errorf("decode fallback response: %w", err)
		}
		if a2aResp.Error != nil {
			return "", fmt.Errorf("agent error %d: %s", a2aResp.Error.Code, a2aResp.Error.Message)
		}
		if a2aResp.Result == nil {
			return "", fmt.Errorf("empty result from agent")
		}
		text := extractText(a2aResp.Result)
		if chunk != nil {
			chunk(text)
		}
		return text, nil
	}

	// Parse SSE stream
	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var event a2aResponse
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Result == nil {
			continue
		}
		t := extractText(event.Result)
		if t != "" {
			full.WriteString(t)
			if chunk != nil {
				chunk(t)
			}
		}
		if event.Result.Status.State == "completed" || event.Result.Status.State == "failed" {
			break
		}
	}
	return full.String(), scanner.Err()
}

// extractText pulls the first text part from all artifacts in a task.
func extractText(task *a2aTask) string {
	var sb strings.Builder
	for _, art := range task.Artifacts {
		for _, p := range art.Parts {
			if p.Type == "text" {
				sb.WriteString(p.Text)
			}
		}
	}
	// Also check status message (some agents put the reply there)
	if sb.Len() == 0 && task.Status.Message != nil {
		for _, p := range task.Status.Message.Parts {
			if p.Type == "text" {
				sb.WriteString(p.Text)
			}
		}
	}
	return sb.String()
}
