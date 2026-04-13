package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type DoctorStatus string

const (
	DoctorStatusPass DoctorStatus = "PASS"
	DoctorStatusWarn DoctorStatus = "WARN"
	DoctorStatusFail DoctorStatus = "FAIL"
)

type DoctorResult struct {
	Name    string       `json:"name"`
	Status  DoctorStatus `json:"status"`
	Summary string       `json:"summary"`
	Fix     string       `json:"fix,omitempty"`
}

type DoctorSummary struct {
	PassCount   int  `json:"pass_count"`
	WarnCount   int  `json:"warn_count"`
	FailCount   int  `json:"fail_count"`
	HasFailures bool `json:"has_failures"`
}

type DoctorReport struct {
	BaseURL string         `json:"base_url"`
	Results []DoctorResult `json:"results"`
	Summary DoctorSummary  `json:"summary"`
}

type doctorCheck struct {
	Name string
	Run  func(context.Context) DoctorResult
}

func runDoctor(ctx context.Context, baseURL string) DoctorReport {
	results := make([]DoctorResult, 0, 6)
	for _, check := range buildDoctorChecks(baseURL) {
		results = append(results, check.Run(ctx))
	}
	return DoctorReport{
		BaseURL: baseURL,
		Results: results,
		Summary: summarizeDoctorResults(results),
	}
}

func summarizeDoctorResults(results []DoctorResult) DoctorSummary {
	var summary DoctorSummary
	for _, result := range results {
		switch result.Status {
		case DoctorStatusPass:
			summary.PassCount++
		case DoctorStatusWarn:
			summary.WarnCount++
		case DoctorStatusFail:
			summary.FailCount++
		}
	}
	summary.HasFailures = summary.FailCount > 0
	return summary
}

func buildDoctorChecks(baseURL string) []doctorCheck {
	return []doctorCheck{
		{Name: "Platform health", Run: func(ctx context.Context) DoctorResult { return checkPlatformHealth(ctx, baseURL) }},
		{Name: "Postgres connection", Run: checkPostgres},
		{Name: "Redis connection", Run: checkRedis},
		{Name: "Platform migrations", Run: checkMigrationsDir},
		{Name: "Workspace templates", Run: checkTemplatesDir},
		{Name: "Docker / provisioner", Run: checkDocker},
	}
}

func checkPlatformHealth(ctx context.Context, baseURL string) DoctorResult {
	result := DoctorResult{Name: "Platform health"}
	endpoint, err := url.JoinPath(baseURL, "health")
	if err != nil {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("Could not build %s health URL", baseURL)
		result.Fix = "Check MOLECLI_URL and make sure it is a valid platform base URL."
		return result
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("Could not create request for %s", endpoint)
		result.Fix = "Check MOLECLI_URL and try again."
		return result
	}

	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("GET %s failed: %v", endpoint, err)
		result.Fix = "Start the platform server or point MOLECLI_URL at a running instance."
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("GET %s returned %s", endpoint, resp.Status)
		result.Fix = strings.TrimSpace(string(body))
		if result.Fix == "" {
			result.Fix = "Check platform logs and confirm the server is healthy."
		}
		return result
	}

	result.Status = DoctorStatusPass
	result.Summary = fmt.Sprintf("GET %s responded OK", endpoint)
	result.Fix = ""
	return result
}

func checkPostgres(ctx context.Context) DoctorResult {
	result := DoctorResult{Name: "Postgres connection"}
	dsn := envOrLocal("DATABASE_URL", "postgres://dev:dev@localhost:5432/molecule?sslmode=disable")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("Could not open DATABASE_URL: %v", err)
		result.Fix = "Check DATABASE_URL and make sure Postgres is installed and reachable."
		return result
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("Could not connect using DATABASE_URL: %v", err)
		result.Fix = "Start infra with ./infra/scripts/setup.sh or update DATABASE_URL."
		return result
	}

	result.Status = DoctorStatusPass
	result.Summary = "Postgres is reachable with DATABASE_URL"
	return result
}

func checkRedis(ctx context.Context) DoctorResult {
	result := DoctorResult{Name: "Redis connection"}
	rawURL := envOrLocal("REDIS_URL", "redis://localhost:6379")
	opts, err := redis.ParseURL(rawURL)
	if err != nil {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("Could not parse REDIS_URL: %v", err)
		result.Fix = "Check REDIS_URL and use a valid redis:// URL."
		return result
	}

	client := redis.NewClient(opts)
	defer client.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("Redis is not reachable at %s: %v", rawURL, err)
		result.Fix = "Start infra with ./infra/scripts/setup.sh or update REDIS_URL."
		return result
	}

	result.Status = DoctorStatusPass
	result.Summary = fmt.Sprintf("Redis is reachable at %s", rawURL)
	return result
}

func checkTemplatesDir(ctx context.Context) DoctorResult {
	_ = ctx
	result := DoctorResult{Name: "Workspace templates"}
	dir := findDoctorConfigsDir([]string{
		"workspace-configs-templates",
		"../workspace-configs-templates",
		"../../workspace-configs-templates",
	})

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("Workspace template directory not found: %s", dir)
		result.Fix = "Run molecli from the repo or restore workspace-configs-templates/."
		return result
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("Could not read workspace template directory: %v", err)
		result.Fix = "Check filesystem permissions for workspace-configs-templates/."
		return result
	}

	var templateCount int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, entry.Name(), "config.yaml")); err == nil {
			templateCount++
		}
	}

	switch {
	case templateCount == 0:
		result.Status = DoctorStatusWarn
		result.Summary = fmt.Sprintf("Template directory exists at %s but has no template config.yaml files", dir)
		result.Fix = "Add or restore at least one template before creating new workspaces."
	default:
		result.Status = DoctorStatusPass
		result.Summary = fmt.Sprintf("Found %d template(s) in %s", templateCount, dir)
	}
	return result
}

func checkMigrationsDir(ctx context.Context) DoctorResult {
	_ = ctx
	result := DoctorResult{Name: "Platform migrations"}
	dir := findDoctorMigrationsDir([]string{
		"migrations",
		"platform/migrations",
		"../migrations",
		"../../migrations",
	})

	if dir == "" {
		result.Status = DoctorStatusFail
		result.Summary = "Could not find a platform migrations directory"
		result.Fix = "Run molecli from the repo root or restore platform/migrations."
		return result
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("Could not read migrations directory: %v", err)
		result.Fix = "Check filesystem permissions for the migrations directory."
		return result
	}

	var sqlCount int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sql") {
			sqlCount++
		}
	}

	if sqlCount == 0 {
		result.Status = DoctorStatusWarn
		result.Summary = fmt.Sprintf("Migrations directory exists at %s but has no .sql files", dir)
		result.Fix = "Restore the platform migration files before starting the server."
		return result
	}

	result.Status = DoctorStatusPass
	result.Summary = fmt.Sprintf("Found %d migration file(s) in %s", sqlCount, dir)
	return result
}

func checkDocker(ctx context.Context) DoctorResult {
	result := DoctorResult{Name: "Docker / provisioner"}
	if _, err := exec.LookPath("docker"); err != nil {
		result.Status = DoctorStatusFail
		result.Summary = "docker command not found in PATH"
		result.Fix = "Install Docker Desktop or make docker available in PATH."
		return result
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "docker", "info")
	if output, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		result.Status = DoctorStatusFail
		result.Summary = fmt.Sprintf("docker info failed: %s", msg)
		result.Fix = "Start Docker Desktop or fix docker daemon access before provisioning workspaces."
		return result
	}

	result.Status = DoctorStatusPass
	result.Summary = "docker info succeeded"
	return result
}

func findDoctorConfigsDir(candidates []string) string {
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}

		entries, _ := os.ReadDir(candidate)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(candidate, entry.Name(), "config.yaml")); err == nil {
				abs, _ := filepath.Abs(candidate)
				return abs
			}
		}
	}
	return "workspace-configs-templates"
}

func findDoctorMigrationsDir(candidates []string) string {
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}
		abs, _ := filepath.Abs(candidate)
		return abs
	}
	return ""
}

func envOrLocal(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func printDoctorReport(report DoctorReport) {
	fmt.Println("Molecule AI Doctor")
	fmt.Println()

	for _, result := range report.Results {
		fmt.Printf("[%s] %s\n", result.Status, result.Name)
		fmt.Printf("  %s\n", result.Summary)
		if result.Fix != "" {
			fmt.Printf("  Fix: %s\n", result.Fix)
		}
		fmt.Println()
	}

	fmt.Println(doctorNextStep(report))
}

func doctorNextStep(report DoctorReport) string {
	switch {
	case report.Summary.HasFailures:
		return "Next: Fix FAIL items first, then rerun `molecli doctor`."
	case report.Summary.WarnCount > 0:
		return "Next: Review warnings before provisioning new workspaces."
	default:
		return "Next: Environment looks good. You can start the platform and Canvas, then deploy a workspace template."
	}
}

type exitCoder interface {
	error
	ExitCode() int
}

type cliExitError struct {
	code int
	msg  string
}

func (e *cliExitError) Error() string {
	return e.msg
}

func (e *cliExitError) ExitCode() int {
	return e.code
}

func newCLIExitError(code int, msg string) error {
	if code == 0 {
		return nil
	}
	return &cliExitError{code: code, msg: msg}
}

func isCLIExitError(err error) bool {
	var exitErr exitCoder
	return errors.As(err, &exitErr)
}
