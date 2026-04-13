package main

// Typed operator subcommands covering the full platform surface.
// Each subcommand is a thin wrapper over callAPI — focused on operator
// ergonomics (good args, clear errors, stable output format) while the
// raw `molecli api` escape hatch (cmd_api.go) covers anything not yet
// wrapped here.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// callAPI is the internal HTTP helper shared by the typed subcommands.
// Returns the response body (already-parsed JSON when possible, raw bytes
// as fallback) so callers can render appropriately.
func callAPI(method, path string, body interface{}) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal body: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL()+path, reader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	return out, resp.StatusCode, err
}

// renderResponse is the shared output path: pretty-print JSON when
// possible, raw text otherwise. Non-2xx statuses are surfaced as errors.
func renderResponse(out []byte, status int) error {
	if flagJSON || looksLikeJSON(out) {
		var v interface{}
		if err := json.Unmarshal(out, &v); err == nil {
			if status >= 400 {
				fmt.Fprintf(os.Stderr, "http %d\n", status)
			}
			if err := printJSON(v); err != nil {
				return err
			}
			if status >= 400 {
				return fmt.Errorf("non-2xx status: %d", status)
			}
			return nil
		}
	}
	fmt.Println(string(out))
	if status >= 400 {
		return fmt.Errorf("non-2xx status: %d", status)
	}
	return nil
}

func looksLikeJSON(b []byte) bool {
	s := strings.TrimSpace(string(b))
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

// ----------------------------------------------------------------------
// ws lifecycle: restart, pause, resume
// ----------------------------------------------------------------------

func buildWSLifecycleCmds() []*cobra.Command {
	restart := &cobra.Command{
		Use:          "restart <id>",
		Short:        "Restart a workspace container",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/restart", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	}
	pause := &cobra.Command{
		Use:          "pause <id>",
		Short:        "Pause a workspace (stops container, preserves state)",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/pause", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	}
	resume := &cobra.Command{
		Use:          "resume <id>",
		Short:        "Resume a paused workspace",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/resume", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	}
	return []*cobra.Command{restart, pause, resume}
}

// ----------------------------------------------------------------------
// plugin
// ----------------------------------------------------------------------

func buildPluginCmd() *cobra.Command {
	root := &cobra.Command{Use: "plugin", Short: "Manage workspace plugins"}
	root.AddCommand(&cobra.Command{
		Use: "registry", Short: "List the platform plugin registry",
		RunE: func(_ *cobra.Command, _ []string) error {
			out, st, err := callAPI("GET", "/plugins", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "sources", Short: "List registered plugin install-source schemes",
		RunE: func(_ *cobra.Command, _ []string) error {
			out, st, err := callAPI("GET", "/plugins/sources", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "list <workspace-id>", Short: "List plugins installed on a workspace",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/plugins", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "available <workspace-id>", Short: "List plugins supported by this workspace's runtime",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/plugins/available", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "install <workspace-id> <source>", Short: "Install a plugin (source = scheme://spec, e.g. local://my-plugin)",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/plugins",
				map[string]string{"source": args[1]})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "uninstall <workspace-id> <plugin-name>", Short: "Uninstall a plugin from a workspace",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("DELETE", "/workspaces/"+args[0]+"/plugins/"+args[1], nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}

// ----------------------------------------------------------------------
// secret (workspace + global)
// ----------------------------------------------------------------------

func buildSecretCmd() *cobra.Command {
	root := &cobra.Command{Use: "secret", Short: "Manage workspace and global secrets"}
	root.AddCommand(&cobra.Command{
		Use: "list <workspace-id>", Short: "List workspace secret keys (values masked)",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/secrets", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "set <workspace-id> <key> <value>", Short: "Set a workspace secret (auto-restarts the workspace)",
		Args: cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/secrets",
				map[string]string{"key": args[1], "value": args[2]})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "delete <workspace-id> <key>", Short: "Delete a workspace secret",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("DELETE", "/workspaces/"+args[0]+"/secrets/"+args[1], nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "list-global", Short: "List global secret keys",
		RunE: func(_ *cobra.Command, _ []string) error {
			out, st, err := callAPI("GET", "/settings/secrets", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "set-global <key> <value>", Short: "Set a global secret",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/settings/secrets",
				map[string]string{"key": args[0], "value": args[1]})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "delete-global <key>", Short: "Delete a global secret",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("DELETE", "/settings/secrets/"+args[0], nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}

// ----------------------------------------------------------------------
// schedule (cron)
// ----------------------------------------------------------------------

func buildScheduleCmd() *cobra.Command {
	root := &cobra.Command{Use: "schedule", Short: "Manage workspace cron schedules"}
	root.AddCommand(&cobra.Command{
		Use: "list <workspace-id>", Short: "List schedules",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/schedules", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "add <workspace-id> <name> <cron-expr> <prompt>", Short: "Create a cron schedule",
		Args: cobra.ExactArgs(4),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/schedules",
				map[string]interface{}{"name": args[1], "cron_expr": args[2], "prompt": args[3], "enabled": true})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "remove <workspace-id> <schedule-id>", Short: "Delete a schedule",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("DELETE", "/workspaces/"+args[0]+"/schedules/"+args[1], nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "run <workspace-id> <schedule-id>", Short: "Trigger a schedule manually",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/schedules/"+args[1]+"/run", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "history <workspace-id> <schedule-id>", Short: "Show past schedule runs",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/schedules/"+args[1]+"/history", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}

// ----------------------------------------------------------------------
// channel
// ----------------------------------------------------------------------

func buildChannelCmd() *cobra.Command {
	root := &cobra.Command{Use: "channel", Short: "Manage social channels (Telegram, Slack, etc.)"}
	root.AddCommand(&cobra.Command{
		Use: "adapters", Short: "List available channel adapters",
		RunE: func(_ *cobra.Command, _ []string) error {
			out, st, err := callAPI("GET", "/channels/adapters", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "list <workspace-id>", Short: "List channels on a workspace",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/channels", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "remove <workspace-id> <channel-id>", Short: "Remove a channel",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("DELETE", "/workspaces/"+args[0]+"/channels/"+args[1], nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "send <workspace-id> <channel-id> <message>", Short: "Send a message through a channel",
		Args: cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/channels/"+args[1]+"/send",
				map[string]string{"message": args[2]})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "test <workspace-id> <channel-id>", Short: "Test a channel connection",
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/channels/"+args[1]+"/test", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}

// ----------------------------------------------------------------------
// approval
// ----------------------------------------------------------------------

func buildApprovalCmd() *cobra.Command {
	root := &cobra.Command{Use: "approval", Short: "Manage human-in-the-loop approvals"}
	root.AddCommand(&cobra.Command{
		Use: "pending", Short: "List all pending approvals across workspaces",
		RunE: func(_ *cobra.Command, _ []string) error {
			out, st, err := callAPI("GET", "/approvals/pending", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "list <workspace-id>", Short: "List approvals for a workspace",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/approvals", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "decide <workspace-id> <approval-id> <approve|deny>", Short: "Approve or deny a pending request",
		Args: cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			decision := strings.ToLower(args[2])
			if decision != "approve" && decision != "deny" {
				return fmt.Errorf("decision must be 'approve' or 'deny', got %q", args[2])
			}
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/approvals/"+args[1]+"/decide",
				map[string]string{"decision": decision})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}

// ----------------------------------------------------------------------
// delegation
// ----------------------------------------------------------------------

func buildDelegationCmd() *cobra.Command {
	root := &cobra.Command{Use: "delegation", Short: "List and create delegations"}
	root.AddCommand(&cobra.Command{
		Use: "list <workspace-id>", Short: "List delegations for a workspace (source side)",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/delegations", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "create <source-id> <target-id> <task>", Short: "Delegate a task from one workspace to another",
		Args: cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/delegate",
				map[string]string{"target_id": args[1], "task": args[2]})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}

// ----------------------------------------------------------------------
// bundle (export / import)
// ----------------------------------------------------------------------

func buildBundleCmd() *cobra.Command {
	root := &cobra.Command{Use: "bundle", Short: "Export / import workspace bundles"}
	root.AddCommand(&cobra.Command{
		Use: "export <workspace-id>", Short: "Export a workspace as a JSON bundle",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/bundles/export/"+args[0], nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "import <bundle.json>", Short: "Import a workspace bundle from a file",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read bundle: %w", err)
			}
			var bundle interface{}
			if err := json.Unmarshal(data, &bundle); err != nil {
				return fmt.Errorf("bundle is not valid JSON: %w", err)
			}
			out, st, err := callAPI("POST", "/bundles/import",
				map[string]interface{}{"bundle": bundle})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}

// ----------------------------------------------------------------------
// org (templates + import)
// ----------------------------------------------------------------------

func buildOrgCmd() *cobra.Command {
	root := &cobra.Command{Use: "org", Short: "Manage organization-level templates"}
	root.AddCommand(&cobra.Command{
		Use: "templates", Short: "List available org templates",
		RunE: func(_ *cobra.Command, _ []string) error {
			out, st, err := callAPI("GET", "/org/templates", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "import <dir>", Short: "Import an org template directory (creates all workspaces)",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/org/import",
				map[string]string{"dir": args[0]})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}

// ----------------------------------------------------------------------
// Remaining small commands: traces, activity, memory (HMA + K/V)
// ----------------------------------------------------------------------

func buildTracesCmd() *cobra.Command {
	return &cobra.Command{
		Use: "traces <workspace-id>", Short: "List recent Langfuse traces for a workspace",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/traces", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	}
}

func buildActivityCmd() *cobra.Command {
	root := &cobra.Command{Use: "activity", Short: "Read or write workspace activity logs"}
	root.AddCommand(&cobra.Command{
		Use: "list <workspace-id>", Short: "List recent activity (a2a_receive, task_update, etc.)",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/activity", nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}

func buildHMAMemoryCmd() *cobra.Command {
	root := &cobra.Command{Use: "hma", Short: "Hierarchical memory (LOCAL / TEAM / GLOBAL scopes)"}
	root.AddCommand(&cobra.Command{
		Use: "commit <workspace-id> <scope> <content>", Short: "Commit a memory with a scope",
		Args: cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			out, st, err := callAPI("POST", "/workspaces/"+args[0]+"/memories",
				map[string]string{"scope": args[1], "content": args[2]})
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	root.AddCommand(&cobra.Command{
		Use: "search <workspace-id> [query]", Short: "Search workspace memories",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			q := ""
			if len(args) > 1 {
				q = "?q=" + args[1]
			}
			out, st, err := callAPI("GET", "/workspaces/"+args[0]+"/memories"+q, nil)
			if err != nil {
				return err
			}
			return renderResponse(out, st)
		},
	})
	return root
}
