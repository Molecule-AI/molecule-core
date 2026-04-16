package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/channels"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/crypto"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/events"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/handlers"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/registry"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/router"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/scheduler"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/supervised"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/ws"

	// External plugin — registers an EnvMutator that injects GITHUB_TOKEN /
	// GH_TOKEN from a GitHub App installation token. Soft-dep: only active
	// when GITHUB_APP_ID env var is set (see main() for the gate).
	pluginloader "github.com/Molecule-AI/molecule-ai-plugin-github-app-auth/pluginloader"
)

func main() {
	// Secrets encryption. In MOLECULE_ENV=prod, boot refuses to start
	// without a valid SECRETS_ENCRYPTION_KEY (fail-secure — Top-5 #5).
	// In any other environment, missing keys just log a warning and
	// continue with encryption disabled for dev ergonomics.
	if err := crypto.InitStrict(); err != nil {
		log.Fatalf("Secrets encryption: %v", err)
	}
	if crypto.IsEnabled() {
		log.Println("Secrets encryption: AES-256-GCM enabled")
	} else {
		log.Println("Secrets encryption: disabled (set SECRETS_ENCRYPTION_KEY — required when MOLECULE_ENV=prod)")
	}

	// Database
	databaseURL := envOr("DATABASE_URL", "postgres://dev:dev@localhost:5432/molecule?sslmode=disable")
	if err := db.InitPostgres(databaseURL); err != nil {
		log.Fatalf("Postgres init failed: %v", err)
	}

	// Run migrations
	migrationsDir := findMigrationsDir()
	if migrationsDir != "" {
		if err := db.RunMigrations(migrationsDir); err != nil {
			log.Fatalf("Migrations failed: %v", err)
		}
	}

	// Redis
	redisURL := envOr("REDIS_URL", "redis://localhost:6379")
	if err := db.InitRedis(redisURL); err != nil {
		log.Fatalf("Redis init failed: %v", err)
	}

	// WebSocket Hub — inject CanCommunicate as a function to avoid import cycles
	hub := ws.NewHub(registry.CanCommunicate)
	go hub.Run()

	// Event Broadcaster
	broadcaster := events.NewBroadcaster(hub)

	// Start Redis pub/sub subscriber
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Every long-running subsystem below is wrapped by supervised.RunWithRecover:
	// a panic (e.g. from a single bad tenant row) is logged + the subsystem is
	// restarted with exponential backoff instead of silently dying forever.
	// Motivation: issue #85 (scheduler silent outage for 12+ hours) and #92
	// (systemic — affects every background goroutine).
	go supervised.RunWithRecover(ctx, "broadcaster", broadcaster.Subscribe)

	// Activity log retention — configurable via env vars
	retentionDays := envOr("ACTIVITY_RETENTION_DAYS", "7")
	cleanupHours := envOr("ACTIVITY_CLEANUP_INTERVAL_HOURS", "6")
	cleanupInterval, _ := time.ParseDuration(cleanupHours + "h")
	if cleanupInterval == 0 {
		cleanupInterval = 6 * time.Hour
	}
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, err := db.DB.ExecContext(ctx, `DELETE FROM activity_logs WHERE created_at < now() - ($1 || ' days')::interval`, retentionDays)
				if err != nil {
					log.Printf("Activity log cleanup error: %v", err)
				} else if n, _ := result.RowsAffected(); n > 0 {
					log.Printf("Activity log cleanup: purged %d old entries", n)
				}
			}
		}
	}()

	// Provisioner — auto-detect backend:
	//   1. MOLECULE_ORG_ID set → SaaS tenant → control plane provisioner
	//   2. Docker available     → self-hosted → Docker provisioner
	//   3. Neither              → provisioner disabled (external agents only)
	var prov *provisioner.Provisioner
	var cpProv *provisioner.CPProvisioner
	if os.Getenv("MOLECULE_ORG_ID") != "" {
		// SaaS tenant — provision via control plane (holds Fly token, manages billing)
		if cp, err := provisioner.NewCPProvisioner(); err != nil {
			log.Printf("Control plane provisioner unavailable: %v", err)
		} else {
			cpProv = cp
			defer cpProv.Close()
			log.Println("Provisioner: Control Plane (auto-detected SaaS tenant)")
		}
	} else {
		// Self-hosted — use local Docker daemon
		if p, err := provisioner.New(); err != nil {
			log.Printf("Provisioner disabled (Docker not available): %v", err)
		} else {
			prov = p
			defer prov.Close()
			log.Println("Provisioner: Docker")
		}
	}

	port := envOr("PORT", "8080")
	platformURL := envOr("PLATFORM_URL", fmt.Sprintf("http://host.docker.internal:%s", port))
	configsDir := envOr("CONFIGS_DIR", findConfigsDir())

	// Init order: wh → onWorkspaceOffline → liveness/healthSweep → router
	// WorkspaceHandler is created before the router so RestartByID can be wired into
	// the offline callbacks used by both the liveness monitor and the health sweep.
	wh := handlers.NewWorkspaceHandler(broadcaster, prov, platformURL, configsDir)
	if cpProv != nil {
		wh.SetCPProvisioner(cpProv)
	}

	// github-app-auth plugin — injects GITHUB_TOKEN + GH_TOKEN into every
	// workspace env using the App's installation access token (rotates ~hourly).
	// Soft-skip when GITHUB_APP_* env vars are absent so dev/self-hosters
	// without an App configured keep working; fail-loud only on MISCONFIG
	// (e.g. APP_ID set but key file missing), not on unset.
	if os.Getenv("GITHUB_APP_ID") != "" {
		if reg, err := pluginloader.BuildRegistry(); err != nil {
			log.Fatalf("github-app-auth plugin: %v", err)
		} else {
			wh.SetEnvMutators(reg)
			log.Printf("github-app-auth: registered, %d mutator(s) in chain", reg.Len())
		}
	} else {
		log.Println("github-app-auth: GITHUB_APP_ID unset — skipping plugin registration (agents will use any PAT from .env)")
	}

	// Offline handler: broadcast event + auto-restart the dead workspace
	onWorkspaceOffline := func(innerCtx context.Context, workspaceID string) {
		if err := broadcaster.RecordAndBroadcast(innerCtx, "WORKSPACE_OFFLINE", workspaceID, map[string]interface{}{}); err != nil {
			log.Printf("Offline broadcast error for %s: %v", workspaceID, err)
		}
		// Auto-restart: bring the workspace back automatically
		go wh.RestartByID(workspaceID)
	}

	// Start Liveness Monitor — Redis TTL expiry-based offline detection + auto-restart
	go supervised.RunWithRecover(ctx, "liveness-monitor", func(c context.Context) {
		registry.StartLivenessMonitor(c, onWorkspaceOffline)
	})

	// Proactive container health sweep — detects dead containers faster than Redis TTL.
	// Checks all "online" workspaces against Docker every 15 seconds.
	if prov != nil {
		go supervised.RunWithRecover(ctx, "health-sweep", func(c context.Context) {
			registry.StartHealthSweep(c, prov, 15*time.Second, onWorkspaceOffline)
		})
	}

	// Cron Scheduler — fires A2A messages to workspaces on user-defined schedules
	cronSched := scheduler.New(wh, broadcaster)
	go supervised.RunWithRecover(ctx, "scheduler", cronSched.Start)

	// Channel Manager — social channel integrations (Telegram, Slack, etc.)
	channelMgr := channels.NewManager(wh, broadcaster)
	go supervised.RunWithRecover(ctx, "channel-manager", channelMgr.Start)

	// Router
	r := router.Setup(hub, broadcaster, prov, platformURL, configsDir, wh, channelMgr)

	// HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Platform starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gracefully...")

	// Cancel background goroutines (liveness monitor, Redis subscriber)
	cancel()

	// Drain HTTP connections (30s timeout)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}

	// Close WebSocket hub
	hub.Close()

	log.Println("Platform stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func findConfigsDir() string {
	candidates := []string{
		"workspace-configs-templates",
		"../workspace-configs-templates",
		"../../workspace-configs-templates",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			// Verify the directory has at least one template with a config.yaml
			entries, _ := os.ReadDir(c)
			hasTemplate := false
			for _, e := range entries {
				if e.IsDir() {
					if _, err := os.Stat(filepath.Join(c, e.Name(), "config.yaml")); err == nil {
						hasTemplate = true
						break
					}
				}
			}
			if !hasTemplate {
				continue
			}
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "workspace-configs-templates"
}

func findMigrationsDir() string {
	candidates := []string{
		"migrations",
		"platform/migrations",
		"../migrations",
		"../../migrations",
	}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "migrations"),
			filepath.Join(dir, "..", "migrations"),
			filepath.Join(dir, "..", "..", "migrations"),
		)
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			log.Printf("Found migrations at: %s", abs)
			return abs
		}
	}
	log.Println("No migrations directory found")
	return ""
}
