package handlers

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/metrics"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/ws"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/wsauth"
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
// Fix D (Cycle 5): agent connections (X-Workspace-ID present) are now validated
// via bearer token before the WebSocket upgrade. Canvas clients (no X-Workspace-ID)
// remain unauthenticated. Pre-token workspaces are grandfathered through.
func (h *SocketHandler) HandleConnect(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")

	// Authenticate workspace agents (not canvas browser clients).
	if workspaceID != "" {
		ctx := c.Request.Context()
		hasLive, err := wsauth.HasAnyLiveToken(ctx, db.DB, workspaceID)
		if err != nil {
			log.Printf("wsauth: WebSocket HasAnyLiveToken(%s) failed: %v", workspaceID, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth check failed"})
			return
		}
		if hasLive {
			tok := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
			if tok == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing workspace auth token"})
				return
			}
			if err := wsauth.ValidateToken(ctx, db.DB, workspaceID, tok); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid workspace auth token"})
				return
			}
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
