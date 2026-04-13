package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// WorkspaceInfo represents a workspace from the platform API.
type WorkspaceInfo struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Role            *string         `json:"role"`
	Tier            int             `json:"tier"`
	Status          string          `json:"status"`
	URL             string          `json:"url"`
	ParentID        *string         `json:"parent_id"`
	AgentCard       json.RawMessage `json:"agent_card"`
	ActiveTasks     int             `json:"active_tasks"`
	LastErrorRate   float64         `json:"last_error_rate"`
	LastSampleError string          `json:"last_sample_error"`
	UptimeSeconds   int             `json:"uptime_seconds"`
	// Phase 30 — surface the runtime so molecli can flag remote agents
	// (runtime='external') distinctly from local Docker workspaces.
	Runtime string `json:"runtime"`
}

// AgentCardInfo represents parsed fields from the agent_card JSON.
type AgentCardInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	URL         string      `json:"url"`
	Skills      []SkillInfo `json:"skills"`
}

// SkillInfo is a skill entry in an agent card.
type SkillInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EventInfo represents a structure event from the platform API.
type EventInfo struct {
	ID          string          `json:"id"`
	EventType   string          `json:"event_type"`
	WorkspaceID *string         `json:"workspace_id"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
}

// WorkspaceFile represents a file read through the Files API.
type WorkspaceFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int    `json:"size"`
}

// SessionSearchItem represents a session search result from the platform API.
type SessionSearchItem struct {
	Kind         string          `json:"kind"`
	ID           string          `json:"id"`
	WorkspaceID  string          `json:"workspace_id"`
	Label        string          `json:"label"`
	Content      string          `json:"content"`
	Method       string          `json:"method"`
	Status       string          `json:"status"`
	RequestBody  json.RawMessage `json:"request_body,omitempty"`
	ResponseBody json.RawMessage `json:"response_body,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// WorkspaceBundle represents the exported bundle payload from the platform API.
type WorkspaceBundle struct {
	Schema        string              `json:"schema"`
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	Tier          int                 `json:"tier"`
	Model         string              `json:"model"`
	SystemPrompt  string              `json:"system_prompt"`
	Skills        []WorkspaceBundleSkill `json:"skills"`
	Prompts       map[string]string   `json:"prompts"`
	SubWorkspaces []WorkspaceBundle   `json:"sub_workspaces"`
}

// WorkspaceBundleSkill is a serialized skill entry from a bundle export.
type WorkspaceBundleSkill struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Files       map[string]string `json:"files"`
}

// WSEvent represents a WebSocket event message.
type WSEvent struct {
	Event       string          `json:"event"`
	WorkspaceID string          `json:"workspace_id"`
	Timestamp   time.Time       `json:"timestamp"`
	Payload     json.RawMessage `json:"payload"`
}

// PlatformClient is an HTTP client for the platform API.
type PlatformClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPlatformClient creates a new platform API client.
func NewPlatformClient(baseURL string) *PlatformClient {
	return &PlatformClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchWorkspaces fetches all workspaces from GET /workspaces.
func (c *PlatformClient) FetchWorkspaces() ([]WorkspaceInfo, error) {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces")
	if err != nil {
		return nil, fmt.Errorf("build workspaces URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch workspaces: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("fetch workspaces: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("fetch workspaces: status %d: %s", resp.StatusCode, body)
	}

	var workspaces []WorkspaceInfo
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return nil, fmt.Errorf("decode workspaces: %w", err)
	}
	return workspaces, nil
}

// FetchEvents fetches recent events from GET /events.
func (c *PlatformClient) FetchEvents() ([]EventInfo, error) {
	endpoint, err := url.JoinPath(c.baseURL, "events")
	if err != nil {
		return nil, fmt.Errorf("build events URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("fetch events: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("fetch events: status %d: %s", resp.StatusCode, body)
	}

	var events []EventInfo
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	return events, nil
}

// GetWorkspaceFile fetches a workspace file via GET /workspaces/:id/files/*path.
func (c *PlatformClient) GetWorkspaceFile(id, filePath string) (*WorkspaceFile, error) {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id, "files", filePath)
	if err != nil {
		return nil, fmt.Errorf("build file URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get file: status %d: %s", resp.StatusCode, body)
	}
	var file WorkspaceFile
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return nil, fmt.Errorf("decode file: %w", err)
	}
	return &file, nil
}

// PutWorkspaceFile writes a workspace file via PUT /workspaces/:id/files/*path.
func (c *PlatformClient) PutWorkspaceFile(id, filePath, content string) error {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id, "files", filePath)
	if err != nil {
		return fmt.Errorf("build file URL: %w", err)
	}
	body, err := json.Marshal(map[string]any{"content": content})
	if err != nil {
		return fmt.Errorf("marshal file request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build file request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("put file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("put file: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// SearchSession searches a workspace's activity logs and memories.
func (c *PlatformClient) SearchSession(id, query string, limit int) ([]SessionSearchItem, error) {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id, "session-search")
	if err != nil {
		return nil, fmt.Errorf("build session search URL: %w", err)
	}
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if encoded := params.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("search session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("search session: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("search session: status %d: %s", resp.StatusCode, body)
	}

	var items []SessionSearchItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode session search: %w", err)
	}
	return items, nil
}

// ExportBundle fetches a workspace bundle via GET /bundles/export/:id.
func (c *PlatformClient) ExportBundle(id string) (*WorkspaceBundle, error) {
	endpoint, err := url.JoinPath(c.baseURL, "bundles", "export", id)
	if err != nil {
		return nil, fmt.Errorf("build bundle export URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("export bundle: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("export bundle: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("export bundle: status %d: %s", resp.StatusCode, body)
	}

	var bundle WorkspaceBundle
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("decode bundle: %w", err)
	}
	return &bundle, nil
}

// DeleteWorkspace deletes a workspace via DELETE /workspaces/:id.
func (c *PlatformClient) DeleteWorkspace(id string) error {
	endpoint, err := deleteURL(c.baseURL, id)
	if err != nil {
		return fmt.Errorf("build delete URL: %w", err)
	}
	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build delete request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("delete workspace: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("delete workspace: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// Request/response types for mutating operations.

// CreateWorkspaceRequest is the body for POST /workspaces.
type CreateWorkspaceRequest struct {
	Name     string `json:"name"`
	Role     string `json:"role,omitempty"`
	Tier     int    `json:"tier,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
}

// CreateWorkspaceResponse is the response from POST /workspaces.
type CreateWorkspaceResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// UpdateWorkspaceRequest is the body for PATCH /workspaces/:id (all fields optional).
type UpdateWorkspaceRequest struct {
	Name     *string `json:"name,omitempty"`
	Role     *string `json:"role,omitempty"`
	Tier     *int    `json:"tier,omitempty"`
	ParentID *string `json:"parent_id,omitempty"`
}

// DiscoverResponse is the response from GET /registry/discover/:id.
type DiscoverResponse struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Status string `json:"status,omitempty"`
}

// AccessResponse is the response from POST /registry/check-access.
type AccessResponse struct {
	Allowed bool `json:"allowed"`
}

// GetWorkspace fetches a single workspace from GET /workspaces/:id.
func (c *PlatformClient) GetWorkspace(id string) (*WorkspaceInfo, error) {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id)
	if err != nil {
		return nil, fmt.Errorf("build workspace URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("get workspace: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("get workspace: status %d: %s", resp.StatusCode, body)
	}

	var ws WorkspaceInfo
	if err := json.NewDecoder(resp.Body).Decode(&ws); err != nil {
		return nil, fmt.Errorf("decode workspace: %w", err)
	}
	return &ws, nil
}

// CreateWorkspace creates a new workspace via POST /workspaces.
func (c *PlatformClient) CreateWorkspace(req CreateWorkspaceRequest) (*CreateWorkspaceResponse, error) {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces")
	if err != nil {
		return nil, fmt.Errorf("build workspaces URL: %w", err)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal create request: %w", err)
	}
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("create workspace: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("create workspace: status %d: %s", resp.StatusCode, b)
	}

	var result CreateWorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode create response: %w", err)
	}
	return &result, nil
}

// UpdateWorkspace updates a workspace via PATCH /workspaces/:id.
func (c *PlatformClient) UpdateWorkspace(id string, req UpdateWorkspaceRequest) error {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id)
	if err != nil {
		return fmt.Errorf("build workspace URL: %w", err)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal update request: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPatch, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build update request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("update workspace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("update workspace: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("update workspace: status %d: %s", resp.StatusCode, b)
	}
	return nil
}

// FetchEventsByWorkspace fetches events for a specific workspace from GET /events/:workspaceId.
func (c *PlatformClient) FetchEventsByWorkspace(workspaceID string) ([]EventInfo, error) {
	endpoint, err := url.JoinPath(c.baseURL, "events", workspaceID)
	if err != nil {
		return nil, fmt.Errorf("build events URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("fetch events: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("fetch events: status %d: %s", resp.StatusCode, body)
	}

	var events []EventInfo
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	return events, nil
}

// DiscoverWorkspace calls GET /registry/discover/:id.
// callerID is optional; if non-empty it is sent as X-Workspace-ID.
func (c *PlatformClient) DiscoverWorkspace(id, callerID string) (*DiscoverResponse, error) {
	endpoint, err := url.JoinPath(c.baseURL, "registry", "discover", id)
	if err != nil {
		return nil, fmt.Errorf("build discover URL: %w", err)
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build discover request: %w", err)
	}
	if callerID != "" {
		req.Header.Set("X-Workspace-ID", callerID)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discover workspace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("discover workspace: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("discover workspace: status %d: %s", resp.StatusCode, body)
	}

	var result DiscoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode discover response: %w", err)
	}
	return &result, nil
}

// GetPeers calls GET /registry/:id/peers.
func (c *PlatformClient) GetPeers(id string) ([]WorkspaceInfo, error) {
	endpoint, err := url.JoinPath(c.baseURL, "registry", id, "peers")
	if err != nil {
		return nil, fmt.Errorf("build peers URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("get peers: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("get peers: status %d: %s", resp.StatusCode, body)
	}

	var peers []WorkspaceInfo
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, fmt.Errorf("decode peers: %w", err)
	}
	return peers, nil
}

// CheckAccess calls POST /registry/check-access.
func (c *PlatformClient) CheckAccess(callerID, targetID string) (*AccessResponse, error) {
	endpoint, err := url.JoinPath(c.baseURL, "registry", "check-access")
	if err != nil {
		return nil, fmt.Errorf("build check-access URL: %w", err)
	}
	body, err := json.Marshal(map[string]string{"caller_id": callerID, "target_id": targetID})
	if err != nil {
		return nil, fmt.Errorf("marshal check-access request: %w", err)
	}
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("check access: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("check access: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("check access: status %d: %s", resp.StatusCode, b)
	}

	var result AccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode check-access response: %w", err)
	}
	return &result, nil
}

// UpdateAgentCard updates an agent's card via POST /registry/update-card.
func (c *PlatformClient) UpdateAgentCard(workspaceID string, card json.RawMessage) error {
	endpoint, err := url.JoinPath(c.baseURL, "registry", "update-card")
	if err != nil {
		return fmt.Errorf("build update-card URL: %w", err)
	}
	body, err := json.Marshal(map[string]any{
		"workspace_id": workspaceID,
		"agent_card":   card,
	})
	if err != nil {
		return fmt.Errorf("marshal update-card request: %w", err)
	}
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("update agent card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("update agent card: status %d (body read error: %v)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("update agent card: status %d: %s", resp.StatusCode, b)
	}
	return nil
}

// ── Config ────────────────────────────────────────────────────────────────────

// ConfigResponse is the response from GET /workspaces/:id/config.
type ConfigResponse struct {
	Data json.RawMessage `json:"data"`
}

// GetConfig fetches the config for a workspace.
func (c *PlatformClient) GetConfig(id string) (json.RawMessage, error) {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id, "config")
	if err != nil {
		return nil, fmt.Errorf("build config URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get config: status %d: %s", resp.StatusCode, body)
	}
	var result ConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return result.Data, nil
}

// PatchConfig merges patch into the workspace config (JSON merge patch semantics).
func (c *PlatformClient) PatchConfig(id string, patch json.RawMessage) error {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id, "config")
	if err != nil {
		return fmt.Errorf("build config URL: %w", err)
	}
	req, err := http.NewRequest(http.MethodPatch, endpoint, bytes.NewReader(patch))
	if err != nil {
		return fmt.Errorf("build config request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("patch config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("patch config: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// ── Memory ────────────────────────────────────────────────────────────────────

// MemoryEntry is one entry from the workspace memory store.
type MemoryEntry struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	ExpiresAt *time.Time      `json:"expires_at,omitempty"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ListMemory fetches all memory entries for a workspace.
func (c *PlatformClient) ListMemory(id string) ([]MemoryEntry, error) {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id, "memory")
	if err != nil {
		return nil, fmt.Errorf("build memory URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("list memory: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list memory: status %d: %s", resp.StatusCode, body)
	}
	var entries []MemoryEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode memory: %w", err)
	}
	return entries, nil
}

// GetMemory fetches a single memory entry by key.
func (c *PlatformClient) GetMemory(id, key string) (*MemoryEntry, error) {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id, "memory", key)
	if err != nil {
		return nil, fmt.Errorf("build memory URL: %w", err)
	}
	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("key %q not found", key)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get memory: status %d: %s", resp.StatusCode, body)
	}
	var entry MemoryEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("decode memory entry: %w", err)
	}
	return &entry, nil
}

// SetMemory upserts a memory entry. ttlSeconds=0 means no expiry.
func (c *PlatformClient) SetMemory(id, key string, value json.RawMessage, ttlSeconds int) error {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id, "memory")
	if err != nil {
		return fmt.Errorf("build memory URL: %w", err)
	}
	payload := map[string]any{"key": key, "value": value}
	if ttlSeconds > 0 {
		payload["ttl_seconds"] = ttlSeconds
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal memory request: %w", err)
	}
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("set memory: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set memory: status %d: %s", resp.StatusCode, b)
	}
	return nil
}

// DeleteMemory deletes a memory entry by key.
func (c *PlatformClient) DeleteMemory(id, key string) error {
	endpoint, err := url.JoinPath(c.baseURL, "workspaces", id, "memory", key)
	if err != nil {
		return fmt.Errorf("build memory URL: %w", err)
	}
	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build memory delete request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete memory: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// ParseAgentCard parses the agent_card JSON into an AgentCardInfo.
func ParseAgentCard(raw json.RawMessage) *AgentCardInfo {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var card AgentCardInfo
	if err := json.Unmarshal(raw, &card); err != nil {
		return nil
	}
	return &card
}
