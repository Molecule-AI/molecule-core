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
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/supervised"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/ws"
	"github.com/docker/docker/client"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Setup(hub *ws.Hub, broadcaster *events.Broadcaster, prov *provisioner.Provisioner, platformURL, configsDir string, wh *handlers.WorkspaceHandler, channelMgr *channels.Manager) *gin.Engine {
	r := gin.Default()

	// Issue #179 — trust no reverse-proxy headers. Without this call Gin's
	// default is to trust ALL X-Forwarded-For values, which lets any caller
	// spoof their IP and bypass per-IP rate limiting. With nil, c.ClientIP()
	// always returns the real TCP RemoteAddr.
	if err := r.SetTrustedProxies(nil); err != nil {
		panic("router: SetTrustedProxies: " + err.Error())
	}

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

	// Tenant guard — the public repo's only SaaS hook. When MOLECULE_ORG_ID is
	// set (only by the private molecule-controlplane provisioner on tenant Fly
	// Machines), rejects requests whose X-Molecule-Org-Id header doesn't match.
	// Unset (self-hosted / dev / CI) → no-op. Registered after metrics so
	// rejected requests still land on the 4xx counter.
	r.Use(middleware.TenantGuard())

	// Security headers (#151) — sets X-Content-Type-Options, X-Frame-Options,
	// Referrer-Policy, Content-Security-Policy, Permissions-Policy, HSTS on
	// every response. Tests in securityheaders_test.go assert each header is
	// present and that handler-set headers are not overridden. Registered
	// last so a handler can still opt out by setting its own header before
	// c.Next() returns.
	r.Use(middleware.SecurityHeaders())

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// /admin/liveness — moved to admin router (fix/issue-684-admin-network-isolation).
	// Served on ADMIN_PORT (default :8081) which is not published to the host.
	// Calling GET /admin/liveness on the public port now returns 404.

	// Prometheus metrics — exempt from rate limiter via separate registration
	// (registered before Use(limiter) takes effect on this specific route — the
	// middleware.Middleware() still records it for observability).
	// Scrape with: curl http://localhost:8080/metrics
	r.GET("/metrics", metrics.Handler())

	// Single-workspace read — open so canvas nodes can fetch their own state
	// without a token (used by WorkspaceNode polling and health checks).
	r.GET("/workspaces/:id", wh.Get)

	// PATCH /workspaces/:id — back on the open router per #138. Canvas
	// drag-reposition uses session cookies not bearer tokens; gating the
	// whole route behind AdminAuth broke drag-to-reposition and inline
	// rename. Field-level authz lives inside WorkspaceHandler.Update:
	//   - {x, y, canvas} only → passthrough (canvas position persist)
	//   - name / role       → passthrough (inline rename)
	//   - tier / parent_id / runtime / workspace_dir → require bearer token
	// The #120 escalation vectors stay locked; only cosmetic fields are open.
	r.PATCH("/workspaces/:id", wh.Update)

	// C1 + C20: GET/POST/DELETE /workspaces moved to admin router (fix/issue-684).
	// Served on ADMIN_PORT (default :8081), not published to host.

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

		// Live agent transcript proxy — surfaces the runtime-specific session
		// log (claude-code reads ~/.claude/projects/<cwd>/<session>.jsonl).
		// Lets canvas / operators see live tool calls + AI thinking instead
		// of waiting for the high-level activity log to flush.
		trsh := handlers.NewTranscriptHandler()
		wsAuth.GET("/transcript", trsh.Get)

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
		// /approvals/pending moved to admin router (fix/issue-684-admin-network-isolation).
		// Served on ADMIN_PORT (default :8081), not published to host.

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

	// Events — GET /events and GET /events/:workspaceId moved to admin router
	// (fix/issue-684-admin-network-isolation). Served on ADMIN_PORT (default :8081).

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
		// Schedule health — open to CanCommunicate peers (no workspace bearer token
		// required) so peer agents can detect silent cron failures without admin auth.
		// Auth is enforced inside the handler via X-Workspace-ID + CanCommunicate
		// (mirrors the /workspaces/:id/a2a pattern). Issue #249.
		r.GET("/workspaces/:id/schedules/health", schedh.Health)

		// Budget — per-workspace spend ceiling and current usage (#541).
		// GET stays on wsAuth — a workspace agent reading its own budget is legitimate.
		// PATCH moved to admin router (fix/issue-684): workspace agents must not be
		// able to self-clear their spending ceiling.
		budgeth := handlers.NewBudgetHandler()
		wsAuth.GET("/budget", budgeth.GetBudget)
		// PATCH /workspaces/:id/budget served on ADMIN_PORT (default :8081).

		// Token management (user-facing create/list/revoke)
		tokh := handlers.NewTokenHandler()
		wsAuth.GET("/tokens", tokh.List)
		wsAuth.POST("/tokens", tokh.Create)
		wsAuth.DELETE("/tokens/:tokenId", tokh.Revoke)

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

		// Token usage metrics — cost transparency (#593).
		// WorkspaceAuth middleware (on wsAuth) binds the bearer to :id.
		mtrh := handlers.NewMetricsHandler()
		wsAuth.GET("/metrics", mtrh.GetMetrics)

		// Cloudflare Artifacts demo integration (#595).
		// All four routes require workspace-scoped bearer auth (wsAuth).
		// CF credentials read from CF_ARTIFACTS_API_TOKEN / CF_ARTIFACTS_NAMESPACE;
		// missing credentials return 503 so the handler still registers in
		// every deployment — the demo is gated on env vars, not compilation.
		arth := handlers.NewArtifactsHandler()
		wsAuth.POST("/artifacts", arth.Create)
		wsAuth.GET("/artifacts", arth.Get)
		wsAuth.POST("/artifacts/fork", arth.Fork)
		wsAuth.POST("/artifacts/token", arth.Token)
	}

	// Global secrets + admin token + admin GitHub token — all moved to admin
	// router (fix/issue-684-admin-network-isolation). Served on ADMIN_PORT
	// (default :8081), not published to host.
	// Paths affected: /settings/secrets, /admin/secrets, /admin/workspaces/:id/test-token,
	// /admin/github-installation-token.

	// Terminal — shares Docker client with provisioner
	var dockerCli *client.Client
	if prov != nil {
		dockerCli = prov.DockerClient()
	}
	th := handlers.NewTerminalHandler(dockerCli)
	wsAuth.GET("/terminal", th.HandleConnect)

	// Canvas Viewport — #166 + #168: GET stays fully open for bootstrap.
	// PUT uses CanvasOrBearer (accepts Origin-match OR bearer token) so the
	// browser canvas can persist drag/zoom state without a bearer, while
	// bearer-carrying clients (molecli, integration tests) still work.
	// Viewport corruption is cosmetic-only — worst case a user refreshes
	// the page — so the softer check is acceptable here. This middleware
	// MUST NOT be used on routes that leak prompts, create workspaces,
	// or write files (#164/#165/#190 class).
	vh := handlers.NewViewportHandler()
	r.GET("/canvas/viewport", vh.Get)
	r.PUT("/canvas/viewport", middleware.CanvasOrBearer(db.DB), vh.Save)

	// Templates
	tmplh := handlers.NewTemplatesHandler(configsDir, dockerCli)
	r.GET("/templates", tmplh.List)
	// POST /templates/import moved to admin router (fix/issue-684).
	// Served on ADMIN_PORT (default :8081), not published to host.
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

	// Bundles, org/import, and org plugin allowlist moved to admin router
	// (fix/issue-684-admin-network-isolation). Served on ADMIN_PORT (default :8081).
	// Paths: /bundles/export/:id, /bundles/import, /org/import,
	//        /orgs/:id/plugins/allowlist.

	// Org Templates (open read — stays on public router)
	orgDir := findOrgDir(configsDir)
	orgh := handlers.NewOrgHandler(wh, broadcaster, prov, channelMgr, configsDir, orgDir)
	r.GET("/org/templates", orgh.ListTemplates)

	// Channels (social integrations — Telegram, Slack, Discord, etc.)
	chh := handlers.NewChannelHandler(channelMgr)
	r.GET("/channels/adapters", chh.ListAdapters)
	wsAuth.GET("/channels", chh.List)
	wsAuth.POST("/channels", chh.Create)
	wsAuth.PATCH("/channels/:channelId", chh.Update)
	wsAuth.DELETE("/channels/:channelId", chh.Delete)
	wsAuth.POST("/channels/:channelId/send", chh.Send)
	wsAuth.POST("/channels/:channelId/test", chh.Test)
	// POST /channels/discover moved to admin router (fix/issue-684).
	// Served on ADMIN_PORT (default :8081), not published to host.
	r.POST("/webhooks/:type", chh.Webhook)

	// SSE — AG-UI compatible event stream per workspace (#590).
	// WorkspaceAuth middleware (on wsAuth) binds the bearer token to :id.
	sseh := handlers.NewSSEHandler(broadcaster)
	wsAuth.GET("/events/stream", sseh.StreamEvents)

	// WebSocket
	sh := handlers.NewSocketHandler(hub)
	r.GET("/ws", sh.HandleConnect)

	// Canvas reverse proxy — when running as a combined tenant image
	// (Dockerfile.tenant), the Next.js canvas server runs on :3000 inside
	// the same container. Any route not matched by the API handlers above
	// gets proxied to the canvas so the browser only ever talks to :8080.
	//
	// When CANVAS_PROXY_URL is empty (self-hosted / local dev), this is a
	// no-op and Gin returns its default 404. The canvas dev server runs
	// separately on :3000 in that setup.
	if canvasURL := os.Getenv("CANVAS_PROXY_URL"); canvasURL != "" {
		canvasProxy := newCanvasProxy(canvasURL)
		r.NoRoute(canvasProxy)
	}

	return r
}

// SetupAdmin returns an HTTP router that serves ONLY admin-gated routes.
//
// Defence-in-depth for issue #684: this router must be served on a separate
// port (ADMIN_PORT, default 8081) that is intentionally NOT published to the
// host in docker-compose.yml. Workspace containers receive PLATFORM_URL
// pointing to the public port and therefore cannot reach these routes even if
// AdminAuth middleware has a regression. External callers (internet-facing
// traffic) are also blocked because the Docker port mapping omits this port.
//
// Within molecule-monorepo-net, containers that know the admin port can still
// reach it — full network-policy isolation (iptables) is a future hardening
// step. This implementation closes the primary attack surface: host-exposed
// admin access and workspace-agent lateral movement via PLATFORM_URL.
func SetupAdmin(hub *ws.Hub, broadcaster *events.Broadcaster, prov *provisioner.Provisioner, platformURL, configsDir string, wh *handlers.WorkspaceHandler, channelMgr *channels.Manager) *gin.Engine {
	r := gin.Default()

	if err := r.SetTrustedProxies(nil); err != nil {
		panic("admin router: SetTrustedProxies: " + err.Error())
	}

	// Tenant isolation — same guard as public router
	r.Use(middleware.TenantGuard())
	r.Use(middleware.SecurityHeaders())

	// Docker client (needed by bundle + templates import handlers)
	var dockerCli *client.Client
	if prov != nil {
		dockerCli = prov.DockerClient()
	}

	// /admin/liveness — per-subsystem last-tick timestamps (issue #166 / #85).
	// Ops-intel leak in production (scheduler tick cadence reveals fleet size +
	// work pattern). AdminAuth-gated and now also network-isolated on admin port.
	r.GET("/admin/liveness", middleware.AdminAuth(db.DB), func(c *gin.Context) {
		snap := supervised.Snapshot()
		out := make(map[string]interface{}, len(snap))
		now := time.Now()
		for name, last := range snap {
			out[name] = gin.H{
				"last_tick_at": last,
				"seconds_ago":  int(now.Sub(last).Seconds()),
			}
		}
		c.JSON(200, gin.H{"subsystems": out})
	})

	// C1 + C20: workspace list and life-cycle mutations (issue #684 network isolation).
	{
		wsAdmin := r.Group("", middleware.AdminAuth(db.DB))
		wsAdmin.GET("/workspaces", wh.List)
		wsAdmin.POST("/workspaces", wh.Create)
		wsAdmin.DELETE("/workspaces/:id", wh.Delete)
	}

	// Cross-workspace pending approvals — admin enumeration (issue #180).
	apph := handlers.NewApprovalsHandler(broadcaster)
	r.GET("/approvals/pending", middleware.AdminAuth(db.DB), apph.ListAll)

	// Events — raw event log leaks org topology (issue #165).
	eh := handlers.NewEventsHandler()
	{
		eventsAdmin := r.Group("", middleware.AdminAuth(db.DB))
		eventsAdmin.GET("/events", eh.List)
		eventsAdmin.GET("/events/:workspaceId", eh.ListByWorkspace)
	}

	// Budget — admin-only spend ceiling modification (issue #541).
	// Workspace agents must not be able to self-clear their spending ceiling.
	budgeth := handlers.NewBudgetHandler()
	r.PATCH("/workspaces/:id/budget", middleware.AdminAuth(db.DB), budgeth.PatchBudget)

	// Global secrets — canonical path /settings/secrets; /admin/secrets for backward compat.
	{
		adminAuth := r.Group("", middleware.AdminAuth(db.DB))
		sechGlobal := handlers.NewSecretsHandler(wh.RestartByID)
		adminAuth.GET("/settings/secrets", sechGlobal.ListGlobal)
		adminAuth.PUT("/settings/secrets", sechGlobal.SetGlobal)
		adminAuth.POST("/settings/secrets", sechGlobal.SetGlobal)
		adminAuth.DELETE("/settings/secrets/:key", sechGlobal.DeleteGlobal)
		adminAuth.GET("/admin/secrets", sechGlobal.ListGlobal)
		adminAuth.POST("/admin/secrets", sechGlobal.SetGlobal)
		adminAuth.DELETE("/admin/secrets/:key", sechGlobal.DeleteGlobal)
	}

	// Admin — test token minting (issue #6).
	// AdminAuth is fail-open on fresh install (HasAnyLiveTokenGlobal == 0).
	{
		tokh := handlers.NewAdminTestTokenHandler()
		r.GET("/admin/workspaces/:id/test-token", middleware.AdminAuth(db.DB), tokh.GetTestToken)
	}

	// Admin — GitHub App installation token refresh (issue #547).
	{
		ghTokH := handlers.NewGitHubTokenHandler(wh.TokenRegistry())
		r.GET("/admin/github-installation-token", middleware.AdminAuth(db.DB), ghTokH.GetInstallationToken)
	}

	// Templates import — writes arbitrary files into configsDir (issue #190).
	{
		tmplh := handlers.NewTemplatesHandler(configsDir, dockerCli)
		tmplAdmin := r.Group("", middleware.AdminAuth(db.DB))
		tmplAdmin.POST("/templates/import", tmplh.Import)
	}

	// Bundles — CRITICAL: arbitrary workspace creation + full config export (issues #164, #165).
	bh := handlers.NewBundleHandler(broadcaster, prov, platformURL, configsDir, dockerCli)
	{
		bundleAdmin := r.Group("", middleware.AdminAuth(db.DB))
		bundleAdmin.GET("/bundles/export/:id", bh.Export)
		bundleAdmin.POST("/bundles/import", bh.Import)
	}

	// Org import — creates workspaces from uploaded YAML (path-sanitized via resolveInsideRoot).
	{
		orgDir := findOrgDir(configsDir)
		orgh := handlers.NewOrgHandler(wh, broadcaster, prov, channelMgr, configsDir, orgDir)
		r.POST("/org/import", middleware.AdminAuth(db.DB), orgh.Import)
	}

	// Org plugin allowlist — tool governance policy (issue #591).
	{
		allowlistAdmin := r.Group("", middleware.AdminAuth(db.DB))
		aplh := handlers.NewOrgPluginAllowlistHandler()
		allowlistAdmin.GET("/orgs/:id/plugins/allowlist", aplh.GetAllowlist)
		allowlistAdmin.PUT("/orgs/:id/plugins/allowlist", aplh.PutAllowlist)
	}

	// Channels discover — bot-token oracle + webhook discovery (issue #250).
	{
		chh := handlers.NewChannelHandler(channelMgr)
		r.POST("/channels/discover", middleware.AdminAuth(db.DB), chh.Discover)
	}

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
