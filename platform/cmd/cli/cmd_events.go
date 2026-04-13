package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func buildEventsCmd() *cobra.Command {
	var (
		workspaceID string
		limit       int
	)

	cmd := &cobra.Command{
		Use:          "events",
		Short:        "List recent platform events",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			var (
				events []EventInfo
				err    error
			)
			if workspaceID != "" {
				events, err = client.FetchEventsByWorkspace(workspaceID)
			} else {
				events, err = client.FetchEvents()
			}
			if err != nil {
				return err
			}

			// Apply limit (most recent first)
			if limit > 0 && len(events) > limit {
				events = events[len(events)-limit:]
			}

			if flagJSON {
				return printJSON(events)
			}

			tw := newTabWriter()
			fmt.Fprintln(tw, "TIME\tEVENT\tWORKSPACE\tID")
			fmt.Fprintln(tw, strings.Repeat("-", 8)+"\t"+
				strings.Repeat("-", 25)+"\t"+
				strings.Repeat("-", 8)+"\t"+
				strings.Repeat("-", 8))
			// Print newest first
			for i := len(events) - 1; i >= 0; i-- {
				e := events[i]
				wsID := ""
				if e.WorkspaceID != nil {
					wsID = shortID(*e.WorkspaceID)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					e.CreatedAt.Local().Format("15:04:05"),
					e.EventType,
					wsID,
					shortID(e.ID),
				)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "", "Filter by workspace ID")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of events to show (0 = all)")

	return cmd
}
