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
// workspace provision API. The control plane holds the Fly API token and
// manages billing/quotas/cleanup. The tenant platform never talks to Fly
// directly.
//
// Set CONTAINER_BACKEND=controlplane to activate. Requires CP_PROVISION_URL
// (control plane base URL, e.g. "https://api.moleculesai.app").
type CPProvisioner struct {
	baseURL    string // e.g. "https://api.moleculesai.app"
	orgID      string // MOLECULE_ORG_ID — identifies which org is provisioning
	httpClient *http.Client
}

// NewCPProvisioner creates a provisioner that delegates to the control plane.
func NewCPProvisioner() (*CPProvisioner, error) {
	orgID := os.Getenv("MOLECULE_ORG_ID")
	if orgID == "" {
		return nil, fmt.Errorf("MOLECULE_ORG_ID required for controlplane provisioner")
	}

	// Auto-derive control plane URL. Priority:
	// 1. Explicit CP_PROVISION_URL (override for testing)
	// 2. Explicit MOLECULE_CP_URL
	// 3. Default: https://api.moleculesai.app (production SaaS)
	baseURL := os.Getenv("CP_PROVISION_URL")
	if baseURL == "" {
		baseURL = os.Getenv("MOLECULE_CP_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.moleculesai.app"
	}

	return &CPProvisioner{
		baseURL: baseURL,
		orgID:   orgID,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
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
	MachineID string `json:"machine_id"`
	Name      string `json:"name"`
	Region    string `json:"region"`
	Status    string `json:"status"`
	Error     string `json:"error"`
}

// Start provisions a workspace by calling the control plane.
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

	respBody, _ := io.ReadAll(resp.Body)
	var result cpProvisionResponse
	json.Unmarshal(respBody, &result)

	if resp.StatusCode != http.StatusCreated {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = string(respBody)
		}
		return "", fmt.Errorf("cp provisioner: provision failed (%d): %s", resp.StatusCode, errMsg)
	}

	log.Printf("CP provisioner: workspace %s → machine %s in %s", cfg.WorkspaceID, result.MachineID, result.Region)
	return result.MachineID, nil
}

// Stop destroys the workspace machine via the control plane.
func (p *CPProvisioner) Stop(ctx context.Context, workspaceID string) error {
	url := fmt.Sprintf("%s/cp/workspaces/%s", p.baseURL, workspaceID)
	body, _ := json.Marshal(map[string]string{
		"org_id":       p.orgID,
		"workspace_id": workspaceID,
	})

	req, _ := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cp provisioner: stop: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cp provisioner: stop failed (%d)", resp.StatusCode)
	}
	return nil
}

// IsRunning checks workspace machine status via the control plane.
func (p *CPProvisioner) IsRunning(ctx context.Context, workspaceID string) (bool, error) {
	url := fmt.Sprintf("%s/cp/workspaces/%s/status?machine_id=%s", p.baseURL, workspaceID, workspaceID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		State string `json:"state"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.State == "started", nil
}

// Close is a no-op.
func (p *CPProvisioner) Close() error { return nil }
