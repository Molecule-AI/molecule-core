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

// FlyRuntimeImages maps runtime names to their GHCR image tags for Fly.
// These are the same images as RuntimeImages but use the full registry path
// since Fly machines pull from a registry, not a local Docker daemon.
var FlyRuntimeImages = map[string]string{
	"langgraph":   "ghcr.io/molecule-ai/workspace-langgraph:latest",
	"claude-code": "ghcr.io/molecule-ai/workspace-claude-code:latest",
	"openclaw":    "ghcr.io/molecule-ai/workspace-openclaw:latest",
	"deepagents":  "ghcr.io/molecule-ai/workspace-deepagents:latest",
	"crewai":      "ghcr.io/molecule-ai/workspace-crewai:latest",
	"autogen":     "ghcr.io/molecule-ai/workspace-autogen:latest",
	"hermes":      "ghcr.io/molecule-ai/workspace-hermes:latest",
	"gemini-cli":  "ghcr.io/molecule-ai/workspace-gemini-cli:latest",
}

const (
	flyAPIBase     = "https://api.machines.dev/v1"
	flyDefaultSize = "shared-cpu-1x"
)

// FlyProvisioner provisions workspace agents as Fly Machines instead of
// local Docker containers. Used on SaaS tenants where no Docker daemon
// is available. Set CONTAINER_BACKEND=flyio to activate.
type FlyProvisioner struct {
	token  string // FLY_API_TOKEN
	appID  string // Fly app to create machines in (FLY_APP)
	region string // Fly region (FLY_REGION, default "ord")
}

// NewFlyProvisioner creates a provisioner that manages workspaces as Fly Machines.
func NewFlyProvisioner() (*FlyProvisioner, error) {
	token := os.Getenv("FLY_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("FLY_API_TOKEN required for Fly provisioner")
	}
	appID := os.Getenv("FLY_WORKSPACE_APP")
	if appID == "" {
		return nil, fmt.Errorf("FLY_WORKSPACE_APP required (Fly app for workspace machines)")
	}
	region := os.Getenv("FLY_REGION")
	if region == "" {
		region = "ord"
	}
	return &FlyProvisioner{token: token, appID: appID, region: region}, nil
}

// flyMachineRequest is the payload for POST /apps/:app/machines.
type flyMachineRequest struct {
	Name   string           `json:"name"`
	Region string           `json:"region"`
	Config flyMachineConfig `json:"config"`
}

type flyMachineConfig struct {
	Image    string            `json:"image"`
	Env      map[string]string `json:"env"`
	Services []flyService      `json:"services,omitempty"`
	Guest    *flyGuest         `json:"guest,omitempty"`
}

type flyService struct {
	Ports        []flyPort `json:"ports"`
	Protocol     string    `json:"protocol"`
	InternalPort int       `json:"internal_port"`
}

type flyPort struct {
	Port     int    `json:"port"`
	Handlers []string `json:"handlers"`
}

type flyGuest struct {
	CPUKind  string `json:"cpu_kind"`
	CPUs     int    `json:"cpus"`
	MemoryMB int    `json:"memory_mb"`
}

type flyMachineResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	State      string `json:"state"`
	InstanceID string `json:"instance_id"`
	PrivateIP  string `json:"private_ip"`
}

// Start creates and starts a Fly Machine for the workspace.
func (p *FlyProvisioner) Start(ctx context.Context, cfg WorkspaceConfig) (string, error) {
	image := FlyRuntimeImages[cfg.Runtime]
	if image == "" {
		image = FlyRuntimeImages["langgraph"]
	}

	name := ContainerName(cfg.WorkspaceID)

	env := map[string]string{
		"WORKSPACE_ID":  cfg.WorkspaceID,
		"PLATFORM_URL":  cfg.PlatformURL,
		"PORT":          DefaultPort,
	}
	if cfg.AwarenessURL != "" {
		env["AWARENESS_URL"] = cfg.AwarenessURL
	}
	if cfg.AwarenessNamespace != "" {
		env["AWARENESS_NAMESPACE"] = cfg.AwarenessNamespace
	}
	// Merge additional env vars (API keys, secrets)
	for k, v := range cfg.EnvVars {
		env[k] = v
	}

	memMB := 512
	cpus := 1
	switch cfg.Tier {
	case 3:
		memMB = 2048
		cpus = 2
	case 4:
		memMB = 4096
		cpus = 4
	}

	req := flyMachineRequest{
		Name:   name,
		Region: p.region,
		Config: flyMachineConfig{
			Image: image,
			Env:   env,
			Services: []flyService{
				{
					InternalPort: 8000,
					Protocol:     "tcp",
					Ports: []flyPort{
						{Port: 443, Handlers: []string{"tls", "http"}},
					},
				},
			},
			Guest: &flyGuest{
				CPUKind:  "shared",
				CPUs:     cpus,
				MemoryMB: memMB,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("fly: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/apps/%s/machines", flyAPIBase, p.appID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("fly: create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("fly: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("fly: create machine failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var machine flyMachineResponse
	if err := json.Unmarshal(respBody, &machine); err != nil {
		return "", fmt.Errorf("fly: parse response: %w", err)
	}

	log.Printf("Fly provisioner: created machine %s (%s) for workspace %s in %s",
		machine.ID, machine.Name, cfg.WorkspaceID, p.region)

	return machine.ID, nil
}

// Stop destroys the Fly Machine for a workspace.
func (p *FlyProvisioner) Stop(ctx context.Context, workspaceID string) error {
	machineID, err := p.findMachine(ctx, workspaceID)
	if err != nil {
		return err
	}
	if machineID == "" {
		return nil // already gone
	}

	url := fmt.Sprintf("%s/apps/%s/machines/%s?force=true", flyAPIBase, p.appID, machineID)
	req, _ := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fly: delete machine: %w", err)
	}
	resp.Body.Close()

	log.Printf("Fly provisioner: deleted machine %s for workspace %s", machineID, workspaceID)
	return nil
}

// IsRunning checks if the workspace's Fly Machine is in "started" state.
func (p *FlyProvisioner) IsRunning(ctx context.Context, workspaceID string) (bool, error) {
	machineID, err := p.findMachine(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	if machineID == "" {
		return false, nil
	}

	url := fmt.Sprintf("%s/apps/%s/machines/%s", flyAPIBase, p.appID, machineID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var machine flyMachineResponse
	json.NewDecoder(resp.Body).Decode(&machine)
	return machine.State == "started", nil
}

// Restart stops and re-creates the machine.
func (p *FlyProvisioner) Restart(ctx context.Context, workspaceID string, cfg WorkspaceConfig) error {
	machineID, err := p.findMachine(ctx, workspaceID)
	if err != nil {
		return err
	}
	if machineID != "" {
		// Restart existing machine
		url := fmt.Sprintf("%s/apps/%s/machines/%s/restart", flyAPIBase, p.appID, machineID)
		req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
		req.Header.Set("Authorization", "Bearer "+p.token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("fly: restart machine: %w", err)
		}
		resp.Body.Close()
		log.Printf("Fly provisioner: restarted machine %s for workspace %s", machineID, workspaceID)
		return nil
	}
	// Machine doesn't exist — create it
	_, err = p.Start(ctx, cfg)
	return err
}

// findMachine looks up the Fly Machine for a workspace by name.
func (p *FlyProvisioner) findMachine(ctx context.Context, workspaceID string) (string, error) {
	name := ContainerName(workspaceID)
	url := fmt.Sprintf("%s/apps/%s/machines", flyAPIBase, p.appID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fly: list machines: %w", err)
	}
	defer resp.Body.Close()

	var machines []flyMachineResponse
	json.NewDecoder(resp.Body).Decode(&machines)

	for _, m := range machines {
		if m.Name == name {
			return m.ID, nil
		}
	}
	return "", nil
}

// Close is a no-op for the Fly provisioner (no persistent connections).
func (p *FlyProvisioner) Close() error { return nil }
