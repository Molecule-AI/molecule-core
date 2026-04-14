package handlers

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/metrics"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate against CORS_ORIGINS. In dev, allow all.
		origins := os.Getenv("CORS_ORIGINS")
		if origins == "" {
			return true // dev mode — no restriction
		}
		origin := r.Header.Get("Origin")
		for _, allowed := range strings.Split(origins, ",") {
			if strings.EqualFold(strings.TrimSpace(allowed), origin) {
				return true
			}
		}
		return false
	},
}

type SocketHandler struct {
	hub *ws.Hub
}

func NewSocketHandler(hub *ws.Hub) *SocketHandler {
	return &SocketHandler{hub: hub}
}

// HandleConnect handles WebSocket upgrade at GET /ws.
// Canvas clients connect without X-Workspace-ID — they receive all events.
// Workspace agents send X-Workspace-ID — events are filtered by CanCommunicate.
//
// WS auth: when X-Workspace-ID is present the caller is treated as a workspace
// agent (not the canvas). requireWorkspaceAuth validates their bearer token
// using the same Phase 30.1 bootstrap-aware logic: workspaces with no live
// tokens on file are grandfathered through; those with tokens must present a
// valid one. Canvas clients (no X-Workspace-ID) are always allowed — they
// receive events via the frontend session and are not considered agents.
func (h *SocketHandler) HandleConnect(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID != "" {
		// Validate bearer token before upgrading — once the WebSocket
		// handshake completes we can no longer send HTTP error responses.
		if err := requireWorkspaceAuth(c.Request.Context(), c, workspaceID); err != nil {
			return // 401 already written by requireWorkspaceAuth
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &ws.Client{
		Conn:        conn,
		WorkspaceID: workspaceID,
		Send:        make(chan []byte, 256),
	}

	h.hub.Register <- client
	metrics.TrackWSConnect()

	// Wrap WritePump and ReadPump so the gauge is decremented exactly once
	// when the client's write goroutine exits (WritePump owns conn lifetime).
	go func() {
		ws.WritePump(client)
		metrics.TrackWSDisconnect()
	}()
	go ws.ReadPump(client, h.hub)
}
