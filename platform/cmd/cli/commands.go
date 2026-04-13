package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// flagJSON is the shared --json flag value read by output helpers.
var flagJSON bool

func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "molecli",
		Short: "Terminal dashboard and CLI for Molecule AI",
		Long: `molecli is a TUI dashboard and CLI for managing Molecule AI workspaces.

Run without arguments to launch the interactive TUI dashboard.
Use subcommands for scriptable, non-interactive access to the platform API.

Environment:
  MOLECLI_URL   Platform base URL (default: http://localhost:8080)`,
		// No args → launch TUI
		RunE: func(cmd *cobra.Command, args []string) error {
			m := NewModel(baseURL())
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
		// Don't print usage on RunE errors (e.g. connection refused)
		SilenceUsage: true,
	}

	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON")

	root.AddCommand(buildAgentCmd())
	root.AddCommand(buildDoctorCmd())
	root.AddCommand(buildWSCmd())
	root.AddCommand(buildEventsCmd())
	root.AddCommand(buildRegistryCmd())

	// 100% platform coverage: operator subcommands + raw escape hatch.
	root.AddCommand(buildAPICmd())
	root.AddCommand(buildPluginCmd())
	root.AddCommand(buildSecretCmd())
	root.AddCommand(buildScheduleCmd())
	root.AddCommand(buildChannelCmd())
	root.AddCommand(buildApprovalCmd())
	root.AddCommand(buildDelegationCmd())
	root.AddCommand(buildBundleCmd())
	root.AddCommand(buildOrgCmd())
	root.AddCommand(buildTracesCmd())
	root.AddCommand(buildActivityCmd())
	root.AddCommand(buildHMAMemoryCmd())

	return root
}

// Output helpers

// printJSON marshals v to indented JSON on stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// newTabWriter returns a tabwriter flushed to stdout.
func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// printWorkspaceTable prints a slice of WorkspaceInfo as a table.
//
// Phase 30 — added a RUNTIME column so operators can see at a glance
// which workspaces are local Docker containers vs. remote agents
// (runtime='external'). Remote agents skip the auto-restart and
// container-health-sweep paths, so when one shows offline the operator
// knows to look at the agent's host machine, not Docker.
func printWorkspaceTable(workspaces []WorkspaceInfo) {
	tw := newTabWriter()
	fmt.Fprintln(tw, "ID\tNAME\tSTATUS\tRUNTIME\tTIER\tTASKS\tERR%\tUPTIME")
	fmt.Fprintln(tw, strings.Repeat("-", 8)+"\t"+
		strings.Repeat("-", 20)+"\t"+
		strings.Repeat("-", 12)+"\t"+
		strings.Repeat("-", 11)+"\t"+
		strings.Repeat("-", 4)+"\t"+
		strings.Repeat("-", 5)+"\t"+
		strings.Repeat("-", 4)+"\t"+
		strings.Repeat("-", 8))
	for _, ws := range workspaces {
		runtime := ws.Runtime
		if runtime == "" {
			runtime = "langgraph" // platform's default; matches DB COALESCE
		}
		// Visual cue: prepend ★ for remote agents so they pop in a long table.
		if runtime == "external" {
			runtime = "★ external"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%d\t%.0f%%\t%s\n",
			shortID(ws.ID),
			truncate(ws.Name, 20),
			ws.Status,
			runtime,
			ws.Tier,
			ws.ActiveTasks,
			ws.LastErrorRate*100,
			formatDuration(ws.UptimeSeconds),
		)
	}
	tw.Flush()
}

// printWorkspaceDetail prints a single WorkspaceInfo verbosely.
func printWorkspaceDetail(ws WorkspaceInfo) {
	tw := newTabWriter()
	fmt.Fprintf(tw, "ID:\t%s\n", ws.ID)
	fmt.Fprintf(tw, "Name:\t%s\n", ws.Name)
	fmt.Fprintf(tw, "Status:\t%s\n", ws.Status)
	if ws.Runtime != "" {
		runtimeLabel := ws.Runtime
		if ws.Runtime == "external" {
			runtimeLabel = "external (Phase 30 remote agent)"
		}
		fmt.Fprintf(tw, "Runtime:\t%s\n", runtimeLabel)
	}
	fmt.Fprintf(tw, "Tier:\t%d\n", ws.Tier)
	if ws.Role != nil && *ws.Role != "" {
		fmt.Fprintf(tw, "Role:\t%s\n", *ws.Role)
	}
	if ws.ParentID != nil && *ws.ParentID != "" {
		fmt.Fprintf(tw, "Parent:\t%s\n", *ws.ParentID)
	}
	if ws.URL != "" {
		fmt.Fprintf(tw, "URL:\t%s\n", ws.URL)
	}
	fmt.Fprintf(tw, "Tasks:\t%d\n", ws.ActiveTasks)
	fmt.Fprintf(tw, "Error Rate:\t%.0f%%\n", ws.LastErrorRate*100)
	if ws.LastSampleError != "" {
		fmt.Fprintf(tw, "Last Error:\t%s\n", ws.LastSampleError)
	}
	fmt.Fprintf(tw, "Uptime:\t%s\n", formatDuration(ws.UptimeSeconds))
	card := ParseAgentCard(ws.AgentCard)
	if card != nil && len(card.Skills) > 0 {
		names := make([]string, 0, len(card.Skills))
		for _, s := range card.Skills {
			if s.Name != "" {
				names = append(names, s.Name)
			} else {
				names = append(names, s.ID)
			}
		}
		fmt.Fprintf(tw, "Skills:\t%s\n", strings.Join(names, ", "))
	}
	tw.Flush()
}
