package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// CPProvisioner provisions workspace agents by calling the control plane's
// workspace provision API. The control plane creates EC2 instances with
// Docker + the workspace runtime installed at boot from PyPI.
//
// Auto-activated when MOLECULE_ORG_ID is set (SaaS tenant).
type CPProvisioner struct {
	baseURL    string
	orgID      string
	httpClient *http.Client
}

// NewCPProvisioner creates a provisioner that delegates to the control plane.
func NewCPProvisioner() (*CPProvisioner, error) {
	orgID := os.Getenv("MOLECULE_ORG_ID")
	if orgID == "" {
		return nil, fmt.Errorf("MOLECULE_ORG_ID required for control plane provisioner")
	}

	// Auto-derive control plane URL.
	baseURL := os.Getenv("CP_PROVISION_URL")
	if baseURL == "" {
		baseURL = os.Getenv("MOLECULE_CP_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.moleculesai.app"
	}

	return &CPProvisioner{
		baseURL:    baseURL,
		orgID:      orgID,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

type cpProvisionRequest struct {
	OrgID       string            `json:"org_id"`
	WorkspaceID string            `json:"workspace_id"`
	Runtime     string            `json:"runtime"`
	Tier        int               `json:"tier"`
	PlatformURL string            `json:"platform_url"`
	Env         map[string]string `json:"env"`
}

type cpProvisionResponse struct {
	InstanceID string `json:"instance_id"`
	PrivateIP  string `json:"private_ip"`
	State      string `json:"state"`
	Error      string `json:"error"`
}

// Start provisions a workspace by calling the control plane → EC2.
func (p *CPProvisioner) Start(ctx context.Context, cfg WorkspaceConfig) (string, error) {
	req := cpProvisionRequest{
		OrgID:       p.orgID,
		WorkspaceID: cfg.WorkspaceID,
		Runtime:     cfg.Runtime,
		Tier:        cfg.Tier,
		PlatformURL: cfg.PlatformURL,
		Env:         cfg.EnvVars,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("cp provisioner: marshal: %w", err)
	}

	url := p.baseURL + "/cp/workspaces/provision"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("cp provisioner: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("cp provisioner: send: %w", err)
	}
	defer resp.Body.Close()

	// Cap body read at 64 KiB — the CP only ever returns small JSON
	// responses; an unbounded read could be weaponized into log-flood
	// DoS by a compromised upstream.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var result cpProvisionResponse
	json.Unmarshal(respBody, &result)

	if resp.StatusCode != http.StatusCreated {
		// Prefer the structured {"error":"..."} field. Do NOT fall back
		// to string(respBody) — our logs ingest errors, and an upstream
		// misconfiguration that echoed the Authorization header or
		// request body into the response would leak bearer tokens.
		errMsg := result.Error
		if errMsg == "" {
			errMsg = fmt.Sprintf("<unstructured body, %d bytes>", len(respBody))
		}
		return "", fmt.Errorf("cp provisioner: provision failed (%d): %s", resp.StatusCode, errMsg)
	}

	log.Printf("CP provisioner: workspace %s → EC2 instance %s (%s)", cfg.WorkspaceID, result.InstanceID, result.State)
	return result.InstanceID, nil
}

// Stop terminates the workspace's EC2 instance via the control plane.
func (p *CPProvisioner) Stop(ctx context.Context, workspaceID string) error {
	url := fmt.Sprintf("%s/cp/workspaces/%s?instance_id=%s", p.baseURL, workspaceID, workspaceID)
	req, _ := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cp provisioner: stop: %w", err)
	}
	resp.Body.Close()
	return nil
}

// IsRunning checks workspace EC2 instance state via the control plane.
func (p *CPProvisioner) IsRunning(ctx context.Context, workspaceID string) (bool, error) {
	url := fmt.Sprintf("%s/cp/workspaces/%s/status?instance_id=%s", p.baseURL, workspaceID, workspaceID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var result struct{ State string `json:"state"` }
	json.NewDecoder(resp.Body).Decode(&result)
	return result.State == "running", nil
}

// Close is a no-op.
func (p *CPProvisioner) Close() error { return nil }
