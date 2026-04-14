// Package provisioner manages Docker container lifecycle for workspace agents.
package provisioner

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// RuntimeImages maps runtime names to their Docker image tags.
// Each adapter has its own pre-built image extending workspace-template:base,
// with runtime-specific deps pre-installed for fast startup.
// Build all: workspace-template/Dockerfile (base), then each adapters/*/Dockerfile.
var RuntimeImages = map[string]string{
	"langgraph":   "workspace-template:langgraph",
	"claude-code": "workspace-template:claude-code",
	"openclaw":    "workspace-template:openclaw",
	"deepagents":  "workspace-template:deepagents",
	"crewai":      "workspace-template:crewai",
	"autogen":     "workspace-template:autogen",
	"hermes":      "workspace-template:hermes", // Hermes (NousResearch) — adapter.py in adapters/hermes/
}

const (
	// DefaultImage is the fallback workspace Docker image (langgraph is the most common runtime).
	DefaultImage = "workspace-template:langgraph"
	// NOTE: Every runtime MUST have an entry in RuntimeImages above. If a runtime is missing,
	// it falls back to DefaultImage which may have wrong deps. Add new runtimes to both
	// RuntimeImages AND create adapters/<runtime>/Dockerfile.

	// DefaultNetwork is the Docker network workspaces join.
	DefaultNetwork = "molecule-monorepo-net"

	// DefaultPort is the port the A2A server listens on inside the container.
	DefaultPort = "8000"

	// ProvisionTimeout is how long to wait for first heartbeat before marking as failed.
	ProvisionTimeout = 3 * time.Minute
)

// WorkspaceConfig holds the parameters needed to provision a workspace container.
type WorkspaceConfig struct {
	WorkspaceID        string
	TemplatePath       string            // Host path to template dir to copy from (e.g. claude-code-default/)
	ConfigFiles        map[string][]byte // Generated config files to write into /configs volume
	PluginsPath        string            // Host path to plugins directory (mounted at /plugins)
	WorkspacePath      string            // Host path to bind-mount as /workspace (if empty, uses Docker named volume)
	Tier               int
	Runtime            string            // "langgraph" (default) or "claude-code", "codex", "ollama", "custom"
	EnvVars            map[string]string // Additional env vars (API keys, etc.)
	PlatformURL        string
	AwarenessURL       string
	AwarenessNamespace string
	WorkspaceAccess    string // #65: "none" (default), "read_only", or "read_write"
}

// Workspace-access constants for #65. Matches the CHECK constraint on
// the workspaces.workspace_access column (migration 019).
const (
	WorkspaceAccessNone      = "none"
	WorkspaceAccessReadOnly  = "read_only"
	WorkspaceAccessReadWrite = "read_write"
)

// ConfigVolumeName returns the Docker named volume for a workspace's configs.
func ConfigVolumeName(workspaceID string) string {
	id := workspaceID
	if len(id) > 12 {
		id = id[:12]
	}
	return fmt.Sprintf("ws-%s-configs", id)
}

// Provisioner manages Docker containers for workspace agents.
type Provisioner struct {
	cli *client.Client
}

// New creates a new Provisioner connected to the local Docker daemon.
func New() (*Provisioner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker: %w", err)
	}
	return &Provisioner{cli: cli}, nil
}

// ContainerName returns the Docker container name for a workspace.
func ContainerName(workspaceID string) string {
	id := workspaceID
	if len(id) > 12 {
		id = id[:12]
	}
	return fmt.Sprintf("ws-%s", id)
}

// InternalURL returns the Docker-internal URL for a workspace container.
func InternalURL(workspaceID string) string {
	return fmt.Sprintf("http://%s:%s", ContainerName(workspaceID), DefaultPort)
}

// Start provisions and starts a workspace container.
func (p *Provisioner) Start(ctx context.Context, cfg WorkspaceConfig) (string, error) {
	name := ContainerName(cfg.WorkspaceID)
	configVolume := ConfigVolumeName(cfg.WorkspaceID)

	// Create named volume for configs (idempotent — no-op if already exists)
	_, err := p.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name: configVolume,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create config volume %s: %w", configVolume, err)
	}
	log.Printf("Provisioner: config volume %s ready", configVolume)

	env := buildContainerEnv(cfg)

	// Select image based on runtime (each adapter has its own pre-built image)
	image := DefaultImage
	if cfg.Runtime != "" {
		if img, ok := RuntimeImages[cfg.Runtime]; ok {
			image = img
		}
	}

	containerCfg := &container.Config{
		Image: image,
		Env:   env,
		ExposedPorts: nat.PortSet{
			nat.Port(DefaultPort + "/tcp"): {},
		},
	}

	// Host config with volume mounts. #65: workspace_access controls whether
	// a bind-mount is read-only (:ro) or read-write. Default "none" implies
	// isolated volume; "read_only"/"read_write" require WorkspacePath set
	// (validated at the handler layer before we get here).
	workspaceMount := buildWorkspaceMount(cfg)
	log.Printf("Provisioner: workspace mount = %q (access=%q)", workspaceMount, cfg.WorkspaceAccess)

	// Mount configs as read-write named volume (agent and Files API need to write)
	// Plugins are installed per-workspace into /configs/plugins/ via the platform API.
	// No global /plugins mount — each workspace owns its plugin set.
	configMount := fmt.Sprintf("%s:/configs", configVolume)
	binds := []string{
		configMount,
		workspaceMount,
	}

	hostCfg := &container.HostConfig{
		Binds:         binds,
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		PortBindings: nat.PortMap{
			nat.Port(DefaultPort + "/tcp"): []nat.PortBinding{
				{HostIP: "127.0.0.1", HostPort: ""}, // Ephemeral host port
			},
		},
	}

	// Apply tier-based container configuration
	ApplyTierConfig(hostCfg, cfg, configMount, name)

	// Network config — join molecule-monorepo-net with container name as alias
	networkCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			DefaultNetwork: {
				Aliases: []string{name},
			},
		},
	}

	// Ensure no stale container exists with the same name (race with restart policy)
	_ = p.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})

	// Log image resolution for debugging stale-image issues
	imgInspect, _, imgErr := p.cli.ImageInspectWithRaw(ctx, image)
	if imgErr == nil {
		log.Printf("Provisioner: creating %s from image %s (ID: %s, created: %s)",
			name, image, imgInspect.ID[:19], imgInspect.Created[:19])
	} else {
		log.Printf("Provisioner: creating %s from image %s (inspect failed: %v)", name, image, imgErr)
	}

	// Create and start container. If the image isn't available locally,
	// Docker returns a generic "No such image" error that's opaque to
	// operators — wrap it with the resolved tag and the exact build
	// command so last_sample_error surfaces something actionable. Issue #117.
	resp, err := p.cli.ContainerCreate(ctx, containerCfg, hostCfg, networkCfg, nil, name)
	if err != nil {
		if isImageNotFoundErr(err) {
			return "", fmt.Errorf(
				"docker image %q not found — run 'bash workspace-template/build-all.sh %s' to build it (underlying error: %w)",
				image, runtimeTagFromImage(image), err,
			)
		}
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := p.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Clean up created container on start failure
		_ = p.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// Verify the started container uses the expected image
	if startedInfo, siErr := p.cli.ContainerInspect(ctx, resp.ID); siErr == nil {
		log.Printf("Provisioner: started container %s (image: %s)", name, startedInfo.Image[:19])
	}

	// Volume ownership is fixed by the entrypoint (starts as root, chowns
	// /configs and /workspace, then drops to agent via gosu). No per-start
	// chown needed here.

	// Copy template files into /configs if TemplatePath is set
	if cfg.TemplatePath != "" {
		if err := p.CopyTemplateToContainer(ctx, resp.ID, cfg.TemplatePath); err != nil {
			log.Printf("Provisioner: warning — failed to copy template to container %s: %v", name, err)
		}
	}

	// Write generated config files into /configs if ConfigFiles is set
	if len(cfg.ConfigFiles) > 0 {
		if err := p.WriteFilesToContainer(ctx, resp.ID, cfg.ConfigFiles); err != nil {
			log.Printf("Provisioner: warning — failed to write config files to container %s: %v", name, err)
		}
	}

	// Resolve the host-mapped port. Retry inspect up to 3 times if Docker hasn't
	// bound the ephemeral port yet (rare race under heavy load).
	hostURL := InternalURL(cfg.WorkspaceID) // fallback to Docker-internal
	for attempt := 0; attempt < 3; attempt++ {
		info, inspectErr := p.cli.ContainerInspect(ctx, resp.ID)
		if inspectErr != nil {
			break
		}
		portBindings := info.NetworkSettings.Ports[nat.Port(DefaultPort+"/tcp")]
		if len(portBindings) > 0 {
			hostPort := portBindings[0].HostPort
			hostIP := portBindings[0].HostIP
			if hostIP == "" {
				hostIP = "127.0.0.1"
			}
			hostURL = fmt.Sprintf("http://%s:%s", hostIP, hostPort)
			break
		}
		if attempt < 2 {
			time.Sleep(500 * time.Millisecond) // wait for Docker to bind the port
		}
	}

	log.Printf("Provisioner: started container %s for workspace %s at %s (internal: %s)", name, cfg.WorkspaceID, hostURL, InternalURL(cfg.WorkspaceID))
	return hostURL, nil
}

// buildWorkspaceMount returns the Docker volume spec for /workspace (#65).
//
// Selection matrix:
//
//   cfg.WorkspacePath | cfg.WorkspaceAccess     | mount
//   ------------------+-------------------------+--------------------------------
//   ""                | "" / "none"             | <named-volume>:/workspace  (isolated, current default)
//   "<host-dir>"      | "" / "read_write"       | <host-dir>:/workspace      (current PM behaviour)
//   "<host-dir>"      | "read_only"             | <host-dir>:/workspace:ro   (research agents get read access without write risk)
//   ""                | "read_only"/"read_write"| <named-volume>:/workspace  (degraded — access requires a mount; validated at handler layer)
//
// Kept pure + side-effect-free so it's unit-testable.
func buildWorkspaceMount(cfg WorkspaceConfig) string {
	// Named volume when no host path is configured.
	if cfg.WorkspacePath == "" {
		volumeName := fmt.Sprintf("ws-%s-workspace", cfg.WorkspaceID)
		return fmt.Sprintf("%s:/workspace", volumeName)
	}
	// Host bind mount. Append :ro for read-only mode; otherwise default
	// (implicit read-write). "none" explicitly opts out of the mount
	// even when a path is set.
	if cfg.WorkspaceAccess == WorkspaceAccessNone {
		volumeName := fmt.Sprintf("ws-%s-workspace", cfg.WorkspaceID)
		return fmt.Sprintf("%s:/workspace", volumeName)
	}
	if cfg.WorkspaceAccess == WorkspaceAccessReadOnly {
		return fmt.Sprintf("%s:/workspace:ro", cfg.WorkspacePath)
	}
	return fmt.Sprintf("%s:/workspace", cfg.WorkspacePath)
}

// ValidateWorkspaceAccess checks that a (access, path) pair is consistent.
// Returns a clear error on mismatch so the handler layer can reject bad
// payloads with a 400 before provisioning.
//
//   - read_only / read_write with empty path → error (needs a host dir)
//   - unknown access value                   → error
//   - none / ""                              → always valid
func ValidateWorkspaceAccess(access, workspacePath string) error {
	switch access {
	case "", WorkspaceAccessNone:
		return nil
	case WorkspaceAccessReadOnly, WorkspaceAccessReadWrite:
		if workspacePath == "" {
			return fmt.Errorf("workspace_access=%q requires workspace_dir to be set", access)
		}
		return nil
	default:
		return fmt.Errorf("workspace_access=%q — must be 'none', 'read_only', or 'read_write'", access)
	}
}

// buildContainerEnv assembles the initial environment variables injected
// into every workspace container.
//
//   - PLATFORM_URL: canonical env var the workspace runtime reads for
//     heartbeat / register / A2A proxy.
//   - MOLECULE_URL: canonical env var the Molecule AI MCP client reads
//     (mcp-server/src/index.ts). Injecting it at provision time so
//     mcp__molecule__* tools called FROM inside the agent container
//     reach the host platform instead of localhost:8080 (which is the
//     container itself). Fixes #67.
//
// Extracted from Start() so it's unit-testable without standing up a
// Docker daemon.
func buildContainerEnv(cfg WorkspaceConfig) []string {
	env := []string{
		fmt.Sprintf("WORKSPACE_ID=%s", cfg.WorkspaceID),
		"WORKSPACE_CONFIG_PATH=/configs",
		fmt.Sprintf("PLATFORM_URL=%s", cfg.PlatformURL),
		fmt.Sprintf("MOLECULE_URL=%s", cfg.PlatformURL),
		fmt.Sprintf("TIER=%d", cfg.Tier),
		"PLUGINS_DIR=/plugins",
	}
	if cfg.AwarenessNamespace != "" && cfg.AwarenessURL != "" {
		env = append(env, fmt.Sprintf("AWARENESS_NAMESPACE=%s", cfg.AwarenessNamespace))
		env = append(env, fmt.Sprintf("AWARENESS_URL=%s", cfg.AwarenessURL))
	}
	for k, v := range cfg.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

// Per-tier resource defaults. Configurable via TIERn_MEMORY_MB and
// TIERn_CPU_SHARES env vars (n in {2,3,4}). CPU shares follow the convention
// 1024 shares == 1 CPU; internally translated to NanoCPUs for a hard cap.
//
// Defaults reflect the tier sizing agreed in issue #14:
//   - T2: 512 MiB,  1024 shares (1 CPU)  — unchanged historical default
//   - T3: 2048 MiB, 2048 shares (2 CPU)  — new cap (previously uncapped)
//   - T4: 4096 MiB, 4096 shares (4 CPU)  — new cap (previously uncapped)
const (
	defaultTier2MemoryMB  = 512
	defaultTier2CPUShares = 1024
	defaultTier3MemoryMB  = 2048
	defaultTier3CPUShares = 2048
	defaultTier4MemoryMB  = 4096
	defaultTier4CPUShares = 4096
)

// getTierMemoryMB returns the memory cap (MiB) for the given tier, reading
// TIERn_MEMORY_MB env var with fallback to the hardcoded default. Returns 0
// for tiers with no cap (e.g. tier 1).
func getTierMemoryMB(tier int) int64 {
	var def int64
	switch tier {
	case 2:
		def = defaultTier2MemoryMB
	case 3:
		def = defaultTier3MemoryMB
	case 4:
		def = defaultTier4MemoryMB
	default:
		return 0
	}
	if v := os.Getenv(fmt.Sprintf("TIER%d_MEMORY_MB", tier)); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// getTierCPUShares returns the CPU allocation (shares, where 1024 == 1 CPU)
// for the given tier, reading TIERn_CPU_SHARES env var with fallback to the
// hardcoded default. Returns 0 for tiers with no cap.
func getTierCPUShares(tier int) int64 {
	var def int64
	switch tier {
	case 2:
		def = defaultTier2CPUShares
	case 3:
		def = defaultTier3CPUShares
	case 4:
		def = defaultTier4CPUShares
	default:
		return 0
	}
	if v := os.Getenv(fmt.Sprintf("TIER%d_CPU_SHARES", tier)); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// applyTierResources writes Memory + NanoCPUs to hostCfg from the tier's
// configured limits (env override or default). Returns the resolved values
// for logging.
func applyTierResources(hostCfg *container.HostConfig, tier int) (memMB, cpuShares int64) {
	memMB = getTierMemoryMB(tier)
	cpuShares = getTierCPUShares(tier)
	if memMB > 0 {
		hostCfg.Resources.Memory = memMB * 1024 * 1024
	}
	if cpuShares > 0 {
		// shares -> NanoCPUs: 1024 shares == 1 CPU == 1e9 NanoCPUs
		hostCfg.Resources.NanoCPUs = (cpuShares * 1_000_000_000) / 1024
	}
	return memMB, cpuShares
}

// ApplyTierConfig configures a HostConfig based on the workspace tier.
// Extracted from Start() so it can be tested independently.
//
//   - Tier 1 (Sandboxed):  readonly rootfs, tmpfs /tmp, strip /workspace mount
//   - Tier 2 (Standard):   resource limits (default 512 MiB, 1 CPU)
//   - Tier 3 (Privileged): privileged + host PID, Docker network, capped resources
//   - Tier 4 (Full access): privileged, host PID, host network, Docker socket, capped resources
//
// Per-tier memory/CPU caps are overridable via TIERn_MEMORY_MB /
// TIERn_CPU_SHARES env vars (n in {2,3,4}).
//
// Unknown/zero tiers default to Tier 2 behavior (safe resource-limited container).
func ApplyTierConfig(hostCfg *container.HostConfig, cfg WorkspaceConfig, configMount, name string) {
	switch cfg.Tier {
	case 1:
		// Sandboxed: strip /workspace mount, keep only config (plugins are in /configs/plugins/)
		tier1Binds := []string{configMount}
		hostCfg.Binds = tier1Binds
		// Readonly root filesystem with tmpfs for /tmp (agent needs scratch space)
		hostCfg.ReadonlyRootfs = true
		hostCfg.Tmpfs = map[string]string{
			"/tmp": "rw,noexec,nosuid,size=64m",
		}
		log.Printf("Provisioner: T1 sandboxed mode for %s (readonly, no /workspace)", name)

	case 3:
		// Privileged access: privileged mode + host PID.
		// Keep the Docker network (not host network) so containers can still reach
		// each other by name. Host networking conflicts with Docker networks and
		// causes port collisions when multiple T3 containers run simultaneously.
		hostCfg.Privileged = true
		hostCfg.PidMode = "host"
		memMB, shares := applyTierResources(hostCfg, 3)
		log.Printf("Provisioner: T3 privileged mode for %s (privileged, host PID, %dm memory, %d CPU shares)", name, memMB, shares)

	case 4:
		// Full host access: everything from T3 + host network + Docker socket + all capabilities.
		// Use for workspaces that need to manage other containers or access host services directly.
		hostCfg.Privileged = true
		hostCfg.PidMode = "host"
		hostCfg.NetworkMode = "host"
		// Mount Docker socket so workspace can manage containers
		hostCfg.Binds = append(hostCfg.Binds, "/var/run/docker.sock:/var/run/docker.sock")
		memMB, shares := applyTierResources(hostCfg, 4)
		log.Printf("Provisioner: T4 full-host mode for %s (privileged, host PID, host network, docker socket, %dm memory, %d CPU shares)", name, memMB, shares)

	default:
		// Tier 2 (Standard) and unknown tiers: normal container with resource limits.
		// This is the safe default — no privileged access, reasonable resource caps.
		memMB, shares := applyTierResources(hostCfg, 2)
		log.Printf("Provisioner: T2 standard mode for %s (%dm memory, %d CPU shares)", name, memMB, shares)
	}
}

// CopyTemplateToContainer copies files from a host directory into /configs in the container.
func (p *Provisioner) CopyTemplateToContainer(ctx context.Context, containerID, templatePath string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(templatePath, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create tar from %s: %w", templatePath, err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	return p.cli.CopyToContainer(ctx, containerID, "/configs", &buf, container.CopyToContainerOptions{})
}

// WriteFilesToContainer writes in-memory files into /configs in the container.
func (p *Provisioner) WriteFilesToContainer(ctx context.Context, containerID string, files map[string][]byte) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	createdDirs := map[string]bool{}
	for name, data := range files {
		// Create parent directories in tar (deduplicated)
		dir := filepath.Dir(name)
		if dir != "." && !createdDirs[dir] {
			if err := tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     dir + "/",
				Mode:     0755,
			}); err != nil {
				return fmt.Errorf("failed to write tar dir header for %s: %w", dir, err)
			}
			createdDirs[dir] = true
		}

		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(data)),
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", name, err)
		}
		if _, err := tw.Write(data); err != nil {
			return fmt.Errorf("failed to write tar data for %s: %w", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	return p.cli.CopyToContainer(ctx, containerID, "/configs", &buf, container.CopyToContainerOptions{})
}

// CopyToContainer exposes CopyToContainer from the Docker client for use by other packages.
func (p *Provisioner) CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader) error {
	return p.cli.CopyToContainer(ctx, containerID, dstPath, content, container.CopyToContainerOptions{})
}

// ExecRead runs "cat <filePath>" in an existing container and returns the output.
// Used to read config files from a running container before stopping it.
func (p *Provisioner) ExecRead(ctx context.Context, containerName, filePath string) ([]byte, error) {
	exec, err := p.cli.ContainerExecCreate(ctx, containerName, container.ExecOptions{
		Cmd:          []string{"cat", filePath},
		AttachStdout: true,
	})
	if err != nil {
		return nil, err
	}
	attach, err := p.cli.ContainerExecAttach(ctx, exec.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, err
	}
	defer attach.Close()
	data, err := io.ReadAll(attach.Reader)
	if err != nil {
		return nil, err
	}
	// Docker multiplexed stream: strip 8-byte headers
	var clean []byte
	for len(data) >= 8 {
		size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		if 8+size > len(data) {
			break
		}
		clean = append(clean, data[8:8+size]...)
		data = data[8+size:]
	}
	return clean, nil
}

// ReadFromVolume reads a file from a Docker named volume using a throwaway container.
// Used as a fallback when ExecRead fails (container already stopped).
func (p *Provisioner) ReadFromVolume(ctx context.Context, volumeName, filePath string) ([]byte, error) {
	resp, err := p.cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"cat", "/vol/" + filePath},
	}, &container.HostConfig{
		Binds: []string{volumeName + ":/vol:ro"},
	}, nil, nil, "")
	if err != nil {
		return nil, err
	}
	defer p.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	if err := p.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, err
	}
	waitCh, errCh := p.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case <-waitCh:
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	}
	reader, err := p.cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	// Strip Docker multiplexed stream headers
	var clean []byte
	for len(data) >= 8 {
		size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		if 8+size > len(data) {
			break
		}
		clean = append(clean, data[8:8+size]...)
		data = data[8+size:]
	}
	return clean, nil
}

// execInContainer runs a command inside a running container as root.
// Best-effort: logs errors but does not fail the caller.
func (p *Provisioner) execInContainer(ctx context.Context, containerID string, cmd []string) {
	execCfg := container.ExecOptions{Cmd: cmd, User: "root"}
	execID, err := p.cli.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		log.Printf("Provisioner: exec create failed: %v", err)
		return
	}
	if err := p.cli.ContainerExecStart(ctx, execID.ID, container.ExecStartOptions{}); err != nil {
		log.Printf("Provisioner: exec start failed: %v", err)
	}
}

// RemoveVolume removes the config volume for a workspace.
func (p *Provisioner) RemoveVolume(ctx context.Context, workspaceID string) error {
	volName := ConfigVolumeName(workspaceID)
	if err := p.cli.VolumeRemove(ctx, volName, true); err != nil {
		return fmt.Errorf("failed to remove volume %s: %w", volName, err)
	}
	log.Printf("Provisioner: removed config volume %s", volName)
	return nil
}

// Stop stops and removes a workspace container.
//
// Uses force-remove FIRST to avoid a race with Docker's `unless-stopped`
// restart policy: if we ContainerStop first, the restart policy can
// respawn the container before ContainerRemove runs, leaving a zombie
// that re-registers via heartbeat after deletion.
func (p *Provisioner) Stop(ctx context.Context, workspaceID string) error {
	name := ContainerName(workspaceID)

	// Force-remove kills and removes in one atomic operation, bypassing
	// the restart policy entirely. If the container doesn't exist, the
	// error is harmless.
	if err := p.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true}); err != nil {
		// Container may already be gone — log but don't fail.
		log.Printf("Provisioner: force-remove warning for %s: %v", name, err)
	}

	log.Printf("Provisioner: stopped and removed container %s", name)
	return nil
}

// IsRunning checks if a workspace container is currently running.
func (p *Provisioner) IsRunning(ctx context.Context, workspaceID string) (bool, error) {
	name := ContainerName(workspaceID)
	info, err := p.cli.ContainerInspect(ctx, name)
	if err != nil {
		return false, nil // Container doesn't exist
	}
	return info.State.Running, nil
}

// DockerClient returns the underlying Docker client for sharing with other handlers.
func (p *Provisioner) DockerClient() *client.Client {
	return p.cli
}

// Close cleans up the Docker client.
func (p *Provisioner) Close() error {
	return p.cli.Close()
}

// ValidateConfigSource is a pure check that ensures at least one static
// source of /configs/config.yaml is available before a container starts.
//
// Inputs mirror the fields on WorkspaceConfig:
//   - templatePath: host dir expected to contain config.yaml (copied into /configs)
//   - configFiles:  in-memory files written into /configs at start time
//
// Returns nil if either source will place config.yaml into /configs.
// When both sources are empty, returns ErrNoConfigSource so callers can
// fall through to a volume probe (VolumeHasFile) before giving up.
//
// Used by the platform's provision flow to catch the rogue-restart-loop
// case (#17): a workspace whose config volume was wiped and whose
// auto-restart path passes empty template+configFiles would otherwise
// boot into a FileNotFoundError crash loop under Docker's
// `unless-stopped` restart policy.
func ValidateConfigSource(templatePath string, configFiles map[string][]byte) error {
	if templatePath != "" {
		// Stat the template's config.yaml; an empty/stale template dir
		// without config.yaml is as broken as no template at all.
		info, err := os.Stat(filepath.Join(templatePath, "config.yaml"))
		if err == nil && !info.IsDir() {
			return nil
		}
	}
	if configFiles != nil {
		if data, ok := configFiles["config.yaml"]; ok && len(data) > 0 {
			return nil
		}
	}
	return ErrNoConfigSource
}

// ErrNoConfigSource is returned by ValidateConfigSource when neither the
// template path nor the in-memory config files supply a config.yaml.
var ErrNoConfigSource = fmt.Errorf("no config.yaml source: template path missing config.yaml and configFiles empty")

// VolumeHasFile returns true if the named config volume for a workspace
// already contains the given file path (relative to /configs). Used by
// the auto-restart path to confirm a previously-provisioned volume is
// still populated before reusing it — if the volume was wiped, we must
// regenerate config or fail cleanly rather than loop on FileNotFoundError.
//
// Implementation: run a throwaway alpine `test -f` container bound to the
// volume read-only. Returns (false, nil) if the file is absent and
// (false, err) only on Docker-level failures.
func (p *Provisioner) VolumeHasFile(ctx context.Context, workspaceID, relPath string) (bool, error) {
	volName := ConfigVolumeName(workspaceID)
	// Confirm the volume exists first — Docker auto-creates on bind otherwise.
	if _, err := p.cli.VolumeInspect(ctx, volName); err != nil {
		return false, nil
	}
	resp, err := p.cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"test", "-f", "/vol/" + relPath},
	}, &container.HostConfig{
		Binds: []string{volName + ":/vol:ro"},
	}, nil, nil, "")
	if err != nil {
		return false, err
	}
	defer p.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	if err := p.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return false, err
	}
	waitCh, errCh := p.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case w := <-waitCh:
		return w.StatusCode == 0, nil
	case err := <-errCh:
		return false, err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// isImageNotFoundErr classifies a Docker client error as "image not
// available locally." The daemon wraps this message in a generic
// SystemError type without exposing a typed sentinel, so we fall back
// to substring match on the known messages emitted by moby. Used by
// Start() to rewrite opaque ContainerCreate failures into actionable
// "run build-all.sh" hints. Issue #117.
func isImageNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "no such image") ||
		strings.Contains(m, "not found") && strings.Contains(m, "image")
}

// runtimeTagFromImage extracts the runtime tag portion from a
// "workspace-template:<runtime>" image reference for use in
// user-facing build hints. Falls back to the full image string if the
// shape is unrecognised.
func runtimeTagFromImage(image string) string {
	const prefix = "workspace-template:"
	if strings.HasPrefix(image, prefix) {
		return image[len(prefix):]
	}
	if i := strings.LastIndex(image, ":"); i >= 0 && i < len(image)-1 {
		return image[i+1:]
	}
	return image
}
