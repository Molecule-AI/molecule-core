package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var langfuseClient = &http.Client{Timeout: 10 * time.Second}

type TracesHandler struct{}

func NewTracesHandler() *TracesHandler {
	return &TracesHandler{}
}

// List handles GET /workspaces/:id/traces
// Proxies to Langfuse API to get recent traces for a workspace.
func (h *TracesHandler) List(c *gin.Context) {
	workspaceID := c.Param("id")

	langfuseHost := os.Getenv("LANGFUSE_HOST")
	langfusePublic := os.Getenv("LANGFUSE_PUBLIC_KEY")
	langfuseSecret := os.Getenv("LANGFUSE_SECRET_KEY")

	if langfuseHost == "" || langfusePublic == "" || langfuseSecret == "" {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	// Fetch traces from Langfuse, filtered by workspace tag or name
	url := fmt.Sprintf("%s/api/public/traces?limit=20&orderBy=timestamp&orderDir=desc&tags=%s",
		langfuseHost, workspaceID)

	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.SetBasicAuth(langfusePublic, langfuseSecret)

	resp, err := langfuseClient.Do(req)
	if err != nil {
		// Langfuse not available — return empty
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}
