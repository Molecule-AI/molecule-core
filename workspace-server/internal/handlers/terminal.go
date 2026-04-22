package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
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

// HandleConnect handles WS /workspaces/:id/terminal
func (h *TerminalHandler) HandleConnect(c *gin.Context) {
	if h.docker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker not available"})
		return
	}

	workspaceID := c.Param("id")
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
