// Package handlers — transcript proxy.
//
// GET /workspaces/:id/transcript proxies to the workspace's own
// /transcript endpoint, which surfaces the live agent session log
// (claude-code reads ~/.claude/projects/<cwd>/<session>.jsonl). Other
// runtimes return supported:false.
//
// Why this lives in the platform: docker exec works for local dev but
// not for remote (Phase 30) workspaces on Fly Machines. The platform's
// network proxy is the only path that scales to both.
package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/gin-gonic/gin"
)

// TranscriptHandler proxies /workspaces/:id/transcript to the workspace agent.
type TranscriptHandler struct {
	httpClient *http.Client
}

func NewTranscriptHandler() *TranscriptHandler {
	return &TranscriptHandler{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Get handles GET /workspaces/:id/transcript?since=N&limit=N.
//
// Looks up the workspace's URL, mints a workspace-scoped bearer token,
// forwards the GET, and streams the response back. Caps payload at 1MB
// to keep a runaway transcript from saturating canvas.
func (h *TranscriptHandler) Get(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	var workspaceURL string
	if err := db.DB.QueryRowContext(ctx,
		`SELECT agent_card->>'url' FROM workspaces WHERE id = $1`,
		workspaceID,
	).Scan(&workspaceURL); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}
	if workspaceURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace not registered (no URL on file)"})
		return
	}

	// No bearer minting needed — workspace /transcript trusts the internal
	// Docker network (same model as POST / for A2A). Phase 30 remote work-
	// spaces will need an auth story; tracked as follow-up.
	target, err := url.Parse(workspaceURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid workspace URL"})
		return
	}
	target.Path = "/transcript"
	target.RawQuery = c.Request.URL.RawQuery

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, "GET", target.String(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build request"})
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("workspace unreachable: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Cap at 1 MB so a giant transcript doesn't melt the canvas.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read workspace response"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
