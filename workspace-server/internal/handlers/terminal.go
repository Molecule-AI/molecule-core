package handlers

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
	"github.com/creack/pty"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const terminalSessionTimeout = 30 * time.Minute

var termUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" ||
			strings.HasPrefix(origin, "http://localhost:") ||
			strings.HasPrefix(origin, "https://localhost:") {
			return true
		}
		// Also allow origins from CORS_ORIGINS env var
		if corsOrigins := os.Getenv("CORS_ORIGINS"); corsOrigins != "" {
			for _, allowed := range strings.Split(corsOrigins, ",") {
				if strings.TrimSpace(allowed) == origin {
					return true
				}
			}
		}
		return false
	},
}

type TerminalHandler struct {
	docker *client.Client
}

func NewTerminalHandler(cli *client.Client) *TerminalHandler {
	return &TerminalHandler{docker: cli}
}

// HandleConnect handles WS /workspaces/:id/terminal. Routes to the remote
// path (aws ec2-instance-connect ssh + docker exec) when the workspace row
// has an instance_id; falls back to local Docker otherwise.
func (h *TerminalHandler) HandleConnect(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	// Check for CP-provisioned workspace (instance_id persisted by
	// provisionWorkspaceCP → migration 038). Null instance_id means the
	// workspace runs as a local Docker container on this tenant.
	var instanceID string
	db.DB.QueryRowContext(ctx,
		`SELECT COALESCE(instance_id, '') FROM workspaces WHERE id = $1`,
		workspaceID).Scan(&instanceID)

	if instanceID != "" {
		h.handleRemoteConnect(c, workspaceID, instanceID)
		return
	}

	h.handleLocalConnect(c, workspaceID)
}

// handleLocalConnect attaches to a Docker container running on this
// tenant's Docker daemon. Original behavior preserved exactly.
func (h *TerminalHandler) handleLocalConnect(c *gin.Context, workspaceID string) {
	if h.docker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker not available"})
		return
	}

	ctx := c.Request.Context()

	// Try multiple container name patterns:
	// 1. Provisioner naming: ws-{id[:12]}
	// 2. Full workspace ID fallback
	// 3. Workspace name from DB (normalized to lowercase-hyphen)
	name := provisioner.ContainerName(workspaceID)
	candidates := []string{name}
	if name != "ws-"+workspaceID {
		candidates = append(candidates, "ws-"+workspaceID)
	}

	// Look up workspace name for manual container naming
	var wsName string
	if _, err := h.docker.Ping(ctx); err == nil {
		db.DB.QueryRowContext(ctx, `SELECT LOWER(REPLACE(name, ' ', '-')) FROM workspaces WHERE id = $1`, workspaceID).Scan(&wsName)
		if wsName != "" {
			candidates = append(candidates, wsName)
		}
	}

	// Find the first running container that matches
	var containerName string
	for _, name := range candidates {
		info, err := h.docker.ContainerInspect(ctx, name)
		if err == nil && info.State.Running {
			containerName = name
			break
		}
	}

	if containerName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not running"})
		return
	}

	// Upgrade to WebSocket
	conn, err := termUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Terminal WebSocket upgrade error: %v", err)
		return
	}

	// No hard session deadline — terminal stays open as long as there is activity.
	// The idle timeout (terminalSessionTimeout) resets on each keystroke in the
	// WebSocket→stdin bridge loop below.
	// The container exec ends when the user types 'exit' or the container stops.

	// Try bash first for better UX (tab completion, history), fall back to sh.
	// ContainerExecCreate succeeds even if the binary doesn't exist — the error
	// only surfaces at attach/start time, so we must retry at the attach level.
	var resp types.HijackedResponse
	var execErr error
	for _, shell := range []string{"/bin/bash", "/bin/sh"} {
		execCfg := container.ExecOptions{
			Cmd:          []string{shell},
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
		}
		execID, createErr := h.docker.ContainerExecCreate(ctx, containerName, execCfg)
		if createErr != nil {
			execErr = createErr
			continue
		}
		resp, execErr = h.docker.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{Tty: true})
		if execErr == nil {
			defer resp.Close()
			break
		}
	}
	if execErr != nil {
		log.Printf("Terminal exec error: %v", execErr)
		conn.WriteMessage(websocket.TextMessage, []byte("Error: failed to create shell session\r\n"))
		conn.Close()
		return
	}

	// Bridge: container stdout → WebSocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := resp.Reader.Read(buf)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("Terminal read error: %v", err)
				}
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
		}
	}()

	// Bridge: WebSocket → container stdin
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if _, err := resp.Conn.Write(msg); err != nil {
			break
		}
		// Reset read deadline on activity
		conn.SetReadDeadline(time.Now().Add(terminalSessionTimeout))
	}

	<-done
}

// sshCommandFactory builds the argv that opens an interactive shell into a
// CP-provisioned workspace. Exposed as a var so tests can override it.
// Real builds invoke the AWS CLI's EIC shortcut, which handles ephemeral
// key generation, SendSSHPublicKey, OpenTunnel, and SSH in one command.
// Requires aws-cli v2 + openssh-client in the tenant image.
var sshCommandFactory = func(instanceID, osUser, containerName string) *exec.Cmd {
	return exec.Command(
		"aws", "ec2-instance-connect", "ssh",
		"--instance-id", instanceID,
		"--connection-type", "eice", // via EIC Endpoint
		"--os-user", osUser,
		"--",
		"docker", "exec", "-it", containerName, "/bin/bash",
	)
}

// handleRemoteConnect tunnels a terminal session to a workspace running on
// a separate EC2 via EC2 Instance Connect. Design: docs/infra/workspace-terminal.md.
func (h *TerminalHandler) handleRemoteConnect(c *gin.Context, workspaceID, instanceID string) {
	osUser := os.Getenv("WORKSPACE_EC2_OS_USER")
	if osUser == "" {
		osUser = "ec2-user" // AL2023 default; override via env if AMI changes
	}
	containerName := provisioner.ContainerName(workspaceID)

	conn, err := termUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Terminal WebSocket upgrade error (remote): %v", err)
		return
	}
	defer conn.Close()

	// PTY so interactive bash works (prompts, line editing, colors).
	// os/exec alone can't allocate a controlling terminal; creack/pty
	// opens the pty pair and wires it to the child's stdin/stdout/stderr.
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	cmd := sshCommandFactory(instanceID, osUser, containerName)
	cmd.Env = os.Environ() // inherit AWS_REGION, AWS credentials, etc.

	ptmx, err := pty.Start(cmd)
	if err != nil {
		// Most likely causes: aws CLI missing, EIC Endpoint not set up,
		// IAM perms missing. Report a specific hint, not the raw error
		// (avoids leaking account ids or ARNs to the client).
		log.Printf("Terminal EIC start error for ws=%s instance=%s: %v", workspaceID, instanceID, err)
		_ = conn.WriteMessage(websocket.TextMessage,
			[]byte("Error: failed to open EIC tunnel — check tenant aws CLI + IAM (see docs/infra/workspace-terminal.md)\r\n"))
		return
	}
	defer func() { _ = ptmx.Close() }()

	done := make(chan struct{})

	// PTY → WebSocket
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if wErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); wErr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("Terminal remote read error: %v", err)
				}
				_ = conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
		}
	}()

	// WebSocket → PTY (stdin)
	go func() {
		for {
			_, msg, rErr := conn.ReadMessage()
			if rErr != nil {
				cancel()
				return
			}
			if _, wErr := ptmx.Write(msg); wErr != nil {
				cancel()
				return
			}
			conn.SetReadDeadline(time.Now().Add(terminalSessionTimeout))
		}
	}()

	// Wait on either pipe to finish or context cancel.
	select {
	case <-done:
	case <-ctx.Done():
	}

	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
}
