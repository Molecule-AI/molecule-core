package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func buildRegistryCmd() *cobra.Command {
	reg := &cobra.Command{
		Use:     "registry",
		Aliases: []string{"reg"},
		Short:   "Registry and discovery operations",
	}

	reg.AddCommand(buildDiscoverCmd())
	reg.AddCommand(buildPeersCmd())
	reg.AddCommand(buildCheckAccessCmd())

	return reg
}

// molecli registry discover <id>

func buildDiscoverCmd() *cobra.Command {
	var callerID string

	cmd := &cobra.Command{
		Use:          "discover <id>",
		Short:        "Discover a workspace (URL + status)",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			result, err := client.DiscoverWorkspace(args[0], callerID)
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(result)
			}
			tw := newTabWriter()
			fmt.Fprintf(tw, "ID:\t%s\n", result.ID)
			fmt.Fprintf(tw, "URL:\t%s\n", result.URL)
			if result.Status != "" {
				fmt.Fprintf(tw, "Status:\t%s\n", result.Status)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&callerID, "caller", "", "Caller workspace ID (sets X-Workspace-ID header)")

	return cmd
}

// molecli registry peers <id>

func buildPeersCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "peers <id>",
		Short:        "List peers of a workspace (siblings, parent, children)",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			peers, err := client.GetPeers(args[0])
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(peers)
			}
			printWorkspaceTable(peers)
			return nil
		},
	}
}

// molecli registry check-access

func buildCheckAccessCmd() *cobra.Command {
	var (
		callerID string
		targetID string
	)

	cmd := &cobra.Command{
		Use:          "check-access",
		Short:        "Check whether one workspace can communicate with another",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if callerID == "" || targetID == "" {
				return fmt.Errorf("--caller and --target are both required")
			}
			client := NewPlatformClient(baseURL())
			result, err := client.CheckAccess(callerID, targetID)
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(result)
			}
			if result.Allowed {
				fmt.Printf("allowed: %s → %s\n", shortID(callerID), shortID(targetID))
			} else {
				fmt.Printf("denied:  %s → %s\n", shortID(callerID), shortID(targetID))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&callerID, "caller", "", "Caller workspace ID (required)")
	cmd.Flags().StringVar(&targetID, "target", "", "Target workspace ID (required)")

	return cmd
}
