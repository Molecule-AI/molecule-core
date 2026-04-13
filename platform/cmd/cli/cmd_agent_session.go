package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func buildAgentSessionCmd() *cobra.Command {
	sess := &cobra.Command{
		Use:   "session",
		Short: "Search recent session activity and memory",
	}
	sess.AddCommand(buildAgentSessionSearchCmd())
	return sess
}

func buildAgentSessionSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <id> [query]",
		Short: "Search a workspace's session activity and memories",
		Example: `  molecli agent session search abc123
  molecli agent session search abc123 "deployment failed"
  molecli agent session search abc123 "memory" --limit 25`,
		Args:         cobra.RangeArgs(1, 2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			query := ""
			if len(args) > 1 {
				query = args[1]
			}
			client := NewPlatformClient(baseURL())
			items, err := client.SearchSession(id, query, limit)
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(items)
			}
			if len(items) == 0 {
				fmt.Println("(no session items found)")
				return nil
			}

			tw := newTabWriter()
			fmt.Fprintln(tw, "KIND\tLABEL\tCONTENT\tCREATED")
			fmt.Fprintln(tw, strings.Repeat("-", 8)+"\t"+strings.Repeat("-", 14)+"\t"+strings.Repeat("-", 50)+"\t"+strings.Repeat("-", 19))
			for _, item := range items {
				label := truncate(item.Label, 14)
				content := truncate(item.Content, 50)
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					item.Kind,
					label,
					content,
					item.CreatedAt.Local().Format("2006-01-02 15:04:05"),
				)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum results to return")
	return cmd
}
