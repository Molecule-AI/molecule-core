package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// buildAPICmd is the generic escape-hatch — hits any platform endpoint.
// Covers 100% of the platform surface without needing per-endpoint typed
// commands. Typed subcommands (plugin, secret, schedule, etc.) layer on
// top for operator ergonomics.
//
// Usage:
//
//   molecli api GET /workspaces
//   molecli api POST /workspaces '{"name":"x","tier":1}'
//   molecli api PATCH /workspaces/ws-123 '{"role":"Reviewer"}'
//   molecli api DELETE /workspaces/ws-123?confirm=true
func buildAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api <METHOD> <PATH> [json-body]",
		Short: "Call any platform API endpoint directly (raw escape hatch)",
		Long: `Raw HTTP call against the platform API. Useful for endpoints that don't yet have typed subcommands.

The body is optional — pass it as the third argument (must be valid JSON).
Use --json to force JSON output (default prints whatever the server returned).`,
		Args:         cobra.RangeArgs(2, 3),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			path := args[1]
			var body io.Reader
			if len(args) == 3 {
				// Validate that it parses as JSON before sending.
				var probe interface{}
				if err := json.Unmarshal([]byte(args[2]), &probe); err != nil {
					return fmt.Errorf("body is not valid JSON: %w", err)
				}
				body = bytes.NewBufferString(args[2])
			}
			base := baseURL()
			if _, err := url.Parse(base + path); err != nil {
				return fmt.Errorf("invalid URL %s%s: %w", base, path, err)
			}
			req, err := http.NewRequest(method, base+path, body)
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 2 * time.Minute}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()
			out, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if resp.StatusCode >= 400 {
				fmt.Fprintf(os.Stderr, "http %d\n", resp.StatusCode)
			}
			// Pretty-print when we got JSON back; fall through to raw bytes otherwise.
			if flagJSON || strings.HasPrefix(strings.TrimSpace(string(out)), "{") || strings.HasPrefix(strings.TrimSpace(string(out)), "[") {
				var v interface{}
				if err := json.Unmarshal(out, &v); err == nil {
					return printJSON(v)
				}
			}
			fmt.Println(string(out))
			if resp.StatusCode >= 400 {
				return fmt.Errorf("non-2xx status: %d", resp.StatusCode)
			}
			return nil
		},
	}
}
