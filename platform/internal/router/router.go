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
		AllowHeaders:     []string{"Origin", "Content-Type", "X-Workspace-ID", "Authorization"},
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

	// Workspaces CRUD — bare /workspaces and /workspaces/:id (no sub-path), unauthenticated for canvas
	r.POST("/workspaces", wh.Create)
	r.GET("/workspaces", wh.List)
	r.GET("/workspaces/:id", wh.Get)
	r.PATCH("/workspaces/:id", wh.Update)
	r.DELETE("/workspaces/:id", wh.Delete)

	// A2A proxy — registered outside the auth group; already enforces CanCommunicate access control.
	r.POST("/workspaces/:id/a2a", wh.ProxyA2A)

	// Auth-gated workspace sub-routes — ALL /workspaces/:id/* paths except /a2a.
	// Fix A (Cycle 5): single WorkspaceAuth middleware blocks C2-C5, C7-C9, C12, C13
	// by requiring a valid bearer token for any workspace that has one on file.
	// Legacy workspaces (no token) are grandfathered to allow rolling upgrades.
	wsAuth := r.Group("/workspaces/:id", middleware.WorkspaceAuth(db.DB))
	{
		// Lifecycle
		wsAuth.GET("/state", wh.State)
		wsAuth.POST("/restart", wh.Restart)
		wsAuth.POST("/pause", wh.Pause)
		wsAuth.POST("/resume", wh.Resume)

		// Async Delegation
		delh := handlers.NewDelegationHandler(wh, broadcaster)
		wsAuth.POST("/delegate", delh.Delegate)
		wsAuth.GET("/delegations", delh.ListDelegations)
		// Record-only endpoint for agent-initiated delegations (#64). Agent-side
		// delegate_to_workspace fires A2A directly for speed + OTEL propagation;
		// this endpoint just adds an activity_logs row so GET /delegations returns
		// the same set the agent's local `check_delegation_status` sees.
		wsAuth.POST("/delegations/record", delh.Record)
		wsAuth.POST("/delegations/:delegation_id/update", delh.UpdateStatus)

		// Traces (Langfuse proxy)
		trh := handlers.NewTracesHandler()
		wsAuth.GET("/traces", trh.List)

		// Agent Memories (HMA)
		memsh := handlers.NewMemoriesHandler()
		wsAuth.POST("/memories", memsh.Commit)
		wsAuth.GET("/memories", memsh.Search)
		wsAuth.DELETE("/memories/:memoryId", memsh.Delete)

		// Approvals
		apph := handlers.NewApprovalsHandler(broadcaster)
		wsAuth.POST("/approvals", apph.Create)
		wsAuth.GET("/approvals", apph.List)
		wsAuth.POST("/approvals/:approvalId/decide", apph.Decide)
		// /approvals/pending is a cross-workspace admin path; keep on root router outside wsAuth.
		r.GET("/approvals/pending", apph.ListAll)

		// Team Expansion
		teamh := handlers.NewTeamHandler(broadcaster, prov, platformURL, configsDir)
		wsAuth.POST("/expand", teamh.Expand)
		wsAuth.POST("/collapse", teamh.Collapse)

		// Agents
		ah := handlers.NewAgentHandler(broadcaster)
		wsAuth.POST("/agent", ah.Assign)
		wsAuth.PATCH("/agent", ah.Replace)
		wsAuth.DELETE("/agent", ah.Remove)
		wsAuth.POST("/agent/move", ah.Move)
	}

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

	// Events (not workspace-scoped — exempt from per-workspace auth)
	eh := handlers.NewEventsHandler()
	r.GET("/events", eh.List)
	r.GET("/events/:workspaceId", eh.ListByWorkspace)

	// Remaining auth-gated workspace sub-routes — appended to wsAuth group declared above.
	{
		// Activity Logs
		acth := handlers.NewActivityHandler(broadcaster)
		wsAuth.GET("/activity", acth.List)
		wsAuth.GET("/session-search", acth.SessionSearch)
		wsAuth.POST("/activity", acth.Report)
		wsAuth.POST("/notify", acth.Notify)

		// Config
		cfgh := handlers.NewConfigHandler()
		wsAuth.GET("/config", cfgh.Get)
		wsAuth.PATCH("/config", cfgh.Patch)

		// Schedules (cron tasks)
		schedh := handlers.NewScheduleHandler()
		wsAuth.GET("/schedules", schedh.List)
		wsAuth.POST("/schedules", schedh.Create)
		wsAuth.PATCH("/schedules/:scheduleId", schedh.Update)
		wsAuth.DELETE("/schedules/:scheduleId", schedh.Delete)
		wsAuth.POST("/schedules/:scheduleId/run", schedh.RunNow)
		wsAuth.GET("/schedules/:scheduleId/history", schedh.History)

		// Memory
		memh := handlers.NewMemoryHandler()
		wsAuth.GET("/memory", memh.List)
		wsAuth.GET("/memory/:key", memh.Get)
		wsAuth.POST("/memory", memh.Set)
		wsAuth.DELETE("/memory/:key", memh.Delete)

		// Secrets (auto-restart workspace after secret change)
		sech := handlers.NewSecretsHandler(wh.RestartByID)
		wsAuth.GET("/secrets", sech.List)
		// Phase 30.2 — decrypted values pull, token-gated. Canvas uses List
		// (keys + metadata only); remote agents use Values to bootstrap env.
		wsAuth.GET("/secrets/values", sech.Values)
		wsAuth.POST("/secrets", sech.Set)
		wsAuth.PUT("/secrets", sech.Set)
		wsAuth.DELETE("/secrets/:key", sech.Delete)
		wsAuth.GET("/model", sech.GetModel)
	}

	// Global secrets — /settings/secrets is the canonical path; /admin/secrets kept for backward compat
	// These are admin-level paths outside the per-workspace auth group.
	{
		sechGlobal := handlers.NewSecretsHandler(wh.RestartByID)
		r.GET("/settings/secrets", sechGlobal.ListGlobal)
		r.PUT("/settings/secrets", sechGlobal.SetGlobal)
		r.POST("/settings/secrets", sechGlobal.SetGlobal)
		r.DELETE("/settings/secrets/:key", sechGlobal.DeleteGlobal)
		r.GET("/admin/secrets", sechGlobal.ListGlobal)
		r.POST("/admin/secrets", sechGlobal.SetGlobal)
		r.DELETE("/admin/secrets/:key", sechGlobal.DeleteGlobal)
	}

	// Terminal — shares Docker client with provisioner
	var dockerCli *client.Client
	if prov != nil {
		dockerCli = prov.DockerClient()
	}
	th := handlers.NewTerminalHandler(dockerCli)
	wsAuth.GET("/terminal", th.HandleConnect)

	// Canvas Viewport
	vh := handlers.NewViewportHandler()
	r.GET("/canvas/viewport", vh.Get)
	r.PUT("/canvas/viewport", vh.Save)

	// Templates
	tmplh := handlers.NewTemplatesHandler(configsDir, dockerCli)
	r.GET("/templates", tmplh.List)
	r.POST("/templates/import", tmplh.Import)
	wsAuth.GET("/shared-context", tmplh.SharedContext)
	wsAuth.PUT("/files", tmplh.ReplaceFiles)
	wsAuth.GET("/files", tmplh.ListFiles)
	wsAuth.GET("/files/*path", tmplh.ReadFile)
	wsAuth.PUT("/files/*path", tmplh.WriteFile)
	wsAuth.DELETE("/files/*path", tmplh.DeleteFile)

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
	wsAuth.GET("/plugins", plgh.ListInstalled)
	wsAuth.GET("/plugins/available", plgh.ListAvailableForWorkspace)
	wsAuth.GET("/plugins/compatibility", plgh.CheckRuntimeCompatibility)
	wsAuth.POST("/plugins", plgh.Install)
	wsAuth.DELETE("/plugins/:name", plgh.Uninstall)
	// Phase 30.3 — stream plugin as tar.gz so remote agents can pull +
	// unpack locally instead of going through Docker exec.
	wsAuth.GET("/plugins/:name/download", plgh.Download)

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
	wsAuth.GET("/channels", chh.List)
	wsAuth.POST("/channels", chh.Create)
	wsAuth.PATCH("/channels/:channelId", chh.Update)
	wsAuth.DELETE("/channels/:channelId", chh.Delete)
	wsAuth.POST("/channels/:channelId/send", chh.Send)
	wsAuth.POST("/channels/:channelId/test", chh.Test)
	r.POST("/channels/discover", chh.Discover)
	r.POST("/webhooks/:type", chh.Webhook)

	// WebSocket
	sh := handlers.NewSocketHandler(hub)
	r.GET("/ws", sh.HandleConnect)

	return r
}

func findPluginsDir(configsDir string) string {
	// configsDir-relative is most reliable; plugins live at repo-root plugins/
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
