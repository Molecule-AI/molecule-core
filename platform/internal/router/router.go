package router

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/channels"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/events"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/handlers"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/metrics"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/middleware"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/ws"
	"github.com/docker/docker/client"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Setup(hub *ws.Hub, broadcaster *events.Broadcaster, prov *provisioner.Provisioner, platformURL, configsDir string, wh *handlers.WorkspaceHandler, channelMgr *channels.Manager) *gin.Engine {
	r := gin.Default()

	// CORS origins — configurable via CORS_ORIGINS env var (comma-separated)
	corsOrigins := []string{"http://localhost:3000", "http://localhost:3001"}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		corsOrigins = strings.Split(v, ",")
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-Workspace-ID"},
		AllowCredentials: true,
	}))

	// Rate limiting — configurable via RATE_LIMIT env var (default 600 req/min)
	// 15 workspaces × 2 heartbeats/min + canvas polling + user actions needs headroom
	rateLimit := 600
	if v := os.Getenv("RATE_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rateLimit = n
		}
	}
	limiter := middleware.NewRateLimiter(rateLimit, time.Minute, context.Background())
	r.Use(limiter.Middleware())

	// Prometheus metrics middleware — records every request's method/path/status/latency.
	// Must be registered after rate limiter so aborted requests are also counted.
	r.Use(metrics.Middleware())

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Prometheus metrics — exempt from rate limiter via separate registration
	// (registered before Use(limiter) takes effect on this specific route — the
	// middleware.Middleware() still records it for observability).
	// Scrape with: curl http://localhost:8080/metrics
	r.GET("/metrics", metrics.Handler())

	// Workspaces CRUD
	r.POST("/workspaces", wh.Create)
	r.GET("/workspaces", wh.List)
	r.GET("/workspaces/:id", wh.Get)
	// Phase 30.4 — lightweight token-gated state polling for remote agents
	// that can't reach the platform's WebSocket. Returns {status, paused,
	// deleted}. Separate from /workspaces/:id so the canvas path stays
	// unauthenticated and returns its full config payload.
	r.GET("/workspaces/:id/state", wh.State)
	r.PATCH("/workspaces/:id", wh.Update)
	r.DELETE("/workspaces/:id", wh.Delete)
	r.POST("/workspaces/:id/restart", wh.Restart)
	r.POST("/workspaces/:id/pause", wh.Pause)
	r.POST("/workspaces/:id/resume", wh.Resume)
	r.POST("/workspaces/:id/a2a", wh.ProxyA2A)

	// Async Delegation
	delh := handlers.NewDelegationHandler(wh, broadcaster)
	r.POST("/workspaces/:id/delegate", delh.Delegate)
	r.GET("/workspaces/:id/delegations", delh.ListDelegations)
	// Record-only endpoint for agent-initiated delegations (#64). Agent-side
	// delegate_to_workspace fires A2A directly for speed + OTEL propagation;
	// this endpoint just adds an activity_logs row so GET /delegations returns
	// the same set the agent's local `check_delegation_status` sees.
	r.POST("/workspaces/:id/delegations/record", delh.Record)
	r.POST("/workspaces/:id/delegations/:delegation_id/update", delh.UpdateStatus)

	// Traces (Langfuse proxy)
	trh := handlers.NewTracesHandler()
	r.GET("/workspaces/:id/traces", trh.List)

	// Agent Memories (HMA)
	memsh := handlers.NewMemoriesHandler()
	r.POST("/workspaces/:id/memories", memsh.Commit)
	r.GET("/workspaces/:id/memories", memsh.Search)
	r.DELETE("/workspaces/:id/memories/:memoryId", memsh.Delete)

	// Approvals
	apph := handlers.NewApprovalsHandler(broadcaster)
	r.GET("/approvals/pending", apph.ListAll)
	r.POST("/workspaces/:id/approvals", apph.Create)
	r.GET("/workspaces/:id/approvals", apph.List)
	r.POST("/workspaces/:id/approvals/:approvalId/decide", apph.Decide)

	// Team Expansion
	teamh := handlers.NewTeamHandler(broadcaster, prov, platformURL, configsDir)
	r.POST("/workspaces/:id/expand", teamh.Expand)
	r.POST("/workspaces/:id/collapse", teamh.Collapse)

	// Agents
	ah := handlers.NewAgentHandler(broadcaster)
	r.POST("/workspaces/:id/agent", ah.Assign)
	r.PATCH("/workspaces/:id/agent", ah.Replace)
	r.DELETE("/workspaces/:id/agent", ah.Remove)
	r.POST("/workspaces/:id/agent/move", ah.Move)

	// Registry
	rh := handlers.NewRegistryHandler(broadcaster)
	r.POST("/registry/register", rh.Register)
	r.POST("/registry/heartbeat", rh.Heartbeat)
	r.POST("/registry/update-card", rh.UpdateCard)

	// Webhooks
	whh := handlers.NewWebhookHandlerWithWorkspace(wh)
	r.POST("/webhooks/github", whh.GitHub)
	r.POST("/webhooks/github/:id", whh.GitHub)

	// Discovery
	dh := handlers.NewDiscoveryHandler()
	r.GET("/registry/discover/:id", dh.Discover)
	r.GET("/registry/:id/peers", dh.Peers)
	r.POST("/registry/check-access", dh.CheckAccess)

	// Events
	eh := handlers.NewEventsHandler()
	r.GET("/events", eh.List)
	r.GET("/events/:workspaceId", eh.ListByWorkspace)

	// Activity Logs
	acth := handlers.NewActivityHandler(broadcaster)
	r.GET("/workspaces/:id/activity", acth.List)
	r.GET("/workspaces/:id/session-search", acth.SessionSearch)
	r.POST("/workspaces/:id/activity", acth.Report)
	r.POST("/workspaces/:id/notify", acth.Notify)

	// Config
	cfgh := handlers.NewConfigHandler()
	r.GET("/workspaces/:id/config", cfgh.Get)
	r.PATCH("/workspaces/:id/config", cfgh.Patch)

	// Schedules (cron tasks)
	schedh := handlers.NewScheduleHandler()
	r.GET("/workspaces/:id/schedules", schedh.List)
	r.POST("/workspaces/:id/schedules", schedh.Create)
	r.PATCH("/workspaces/:id/schedules/:scheduleId", schedh.Update)
	r.DELETE("/workspaces/:id/schedules/:scheduleId", schedh.Delete)
	r.POST("/workspaces/:id/schedules/:scheduleId/run", schedh.RunNow)
	r.GET("/workspaces/:id/schedules/:scheduleId/history", schedh.History)

	// Memory
	memh := handlers.NewMemoryHandler()
	r.GET("/workspaces/:id/memory", memh.List)
	r.GET("/workspaces/:id/memory/:key", memh.Get)
	r.POST("/workspaces/:id/memory", memh.Set)
	r.DELETE("/workspaces/:id/memory/:key", memh.Delete)

	// Secrets (auto-restart workspace after secret change)
	sech := handlers.NewSecretsHandler(wh.RestartByID)
	r.GET("/workspaces/:id/secrets", sech.List)
	// Phase 30.2 — decrypted values pull, token-gated. Canvas uses List
	// (keys + metadata only); remote agents use Values to bootstrap env.
	r.GET("/workspaces/:id/secrets/values", sech.Values)
	r.POST("/workspaces/:id/secrets", sech.Set)
	r.PUT("/workspaces/:id/secrets", sech.Set)
	r.DELETE("/workspaces/:id/secrets/:key", sech.Delete)
	r.GET("/workspaces/:id/model", sech.GetModel)

	// Global secrets — /settings/secrets is the canonical path; /admin/secrets kept for backward compat
	r.GET("/settings/secrets", sech.ListGlobal)
	r.PUT("/settings/secrets", sech.SetGlobal)
	r.POST("/settings/secrets", sech.SetGlobal)
	r.DELETE("/settings/secrets/:key", sech.DeleteGlobal)
	r.GET("/admin/secrets", sech.ListGlobal)
	r.POST("/admin/secrets", sech.SetGlobal)
	r.DELETE("/admin/secrets/:key", sech.DeleteGlobal)

	// Terminal — shares Docker client with provisioner
	var dockerCli *client.Client
	if prov != nil {
		dockerCli = prov.DockerClient()
	}
	th := handlers.NewTerminalHandler(dockerCli)
	r.GET("/workspaces/:id/terminal", th.HandleConnect)

	// Canvas Viewport
	vh := handlers.NewViewportHandler()
	r.GET("/canvas/viewport", vh.Get)
	r.PUT("/canvas/viewport", vh.Save)

	// Templates
	tmplh := handlers.NewTemplatesHandler(configsDir, dockerCli)
	r.GET("/templates", tmplh.List)
	r.POST("/templates/import", tmplh.Import)
	r.GET("/workspaces/:id/shared-context", tmplh.SharedContext)
	r.PUT("/workspaces/:id/files", tmplh.ReplaceFiles)
	r.GET("/workspaces/:id/files", tmplh.ListFiles)
	r.GET("/workspaces/:id/files/*path", tmplh.ReadFile)
	r.PUT("/workspaces/:id/files/*path", tmplh.WriteFile)
	r.DELETE("/workspaces/:id/files/*path", tmplh.DeleteFile)

	// Plugins
	pluginsDir := findPluginsDir(configsDir)
	// Runtime lookup lets the plugins handler filter the registry to plugins
	// that declare support for the workspace's runtime, without taking a
	// direct DB dependency in the handler package.
	runtimeLookup := func(workspaceID string) (string, error) {
		var runtime string
		err := db.DB.QueryRowContext(
			context.Background(),
			`SELECT COALESCE(runtime, 'langgraph') FROM workspaces WHERE id = $1`,
			workspaceID,
		).Scan(&runtime)
		return runtime, err
	}
	plgh := handlers.NewPluginsHandler(pluginsDir, dockerCli, wh.RestartByID).
		WithRuntimeLookup(runtimeLookup)
	r.GET("/plugins", plgh.ListRegistry)
	r.GET("/plugins/sources", plgh.ListSources)
	r.GET("/workspaces/:id/plugins", plgh.ListInstalled)
	r.GET("/workspaces/:id/plugins/available", plgh.ListAvailableForWorkspace)
	r.GET("/workspaces/:id/plugins/compatibility", plgh.CheckRuntimeCompatibility)
	r.POST("/workspaces/:id/plugins", plgh.Install)
	r.DELETE("/workspaces/:id/plugins/:name", plgh.Uninstall)
	// Phase 30.3 — stream plugin as tar.gz so remote agents can pull +
	// unpack locally instead of going through Docker exec.
	r.GET("/workspaces/:id/plugins/:name/download", plgh.Download)

	// Bundles
	bh := handlers.NewBundleHandler(broadcaster, prov, platformURL, configsDir, dockerCli)
	r.GET("/bundles/export/:id", bh.Export)
	r.POST("/bundles/import", bh.Import)

	// Org Templates
	orgDir := findOrgDir(configsDir)
	orgh := handlers.NewOrgHandler(wh, broadcaster, prov, channelMgr, configsDir, orgDir)
	r.GET("/org/templates", orgh.ListTemplates)
	r.POST("/org/import", orgh.Import)

	// Channels (social integrations — Telegram, Slack, Discord, etc.)
	chh := handlers.NewChannelHandler(channelMgr)
	r.GET("/channels/adapters", chh.ListAdapters)
	r.GET("/workspaces/:id/channels", chh.List)
	r.POST("/workspaces/:id/channels", chh.Create)
	r.PATCH("/workspaces/:id/channels/:channelId", chh.Update)
	r.DELETE("/workspaces/:id/channels/:channelId", chh.Delete)
	r.POST("/workspaces/:id/channels/:channelId/send", chh.Send)
	r.POST("/workspaces/:id/channels/:channelId/test", chh.Test)
	r.POST("/channels/discover", chh.Discover)
	r.POST("/webhooks/:type", chh.Webhook)

	// WebSocket
	sh := handlers.NewSocketHandler(hub)
	r.GET("/ws", sh.HandleConnect)

	return r
}

func findPluginsDir(configsDir string) string {
	// configsDir-relative is most reliable (avoids empty platform/plugins/)
	candidates := []string{
		filepath.Join(configsDir, "..", "plugins"),
		"../plugins",
		"plugins",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			// Must have at least one plugin subfolder to be valid
			entries, _ := os.ReadDir(c)
			for _, e := range entries {
				if e.IsDir() {
					abs, _ := filepath.Abs(c)
					return abs
				}
			}
		}
	}
	abs, _ := filepath.Abs(filepath.Join(configsDir, "..", "plugins"))
	return abs
}

func findOrgDir(configsDir string) string {
	candidates := []string{
		"org-templates",
		"../org-templates",
		filepath.Join(configsDir, "..", "org-templates"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "org-templates"
}
