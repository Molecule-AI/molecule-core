package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func buildWSCmd() *cobra.Command {
	ws := &cobra.Command{
		Use:     "ws",
		Aliases: []string{"workspace", "workspaces"},
		Short:   "Manage workspaces",
	}

	ws.AddCommand(buildWSListCmd())
	ws.AddCommand(buildWSGetCmd())
	ws.AddCommand(buildWSCreateCmd())
	ws.AddCommand(buildWSUpdateCmd())
	ws.AddCommand(buildWSDeleteCmd())
	// Lifecycle ops (restart / pause / resume) — defined in cmd_ops.go
	for _, c := range buildWSLifecycleCmds() {
		ws.AddCommand(c)
	}

	return ws
}

// molecli ws list

func buildWSListCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "list",
		Aliases:      []string{"ls"},
		Short:        "List all workspaces",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			workspaces, err := client.FetchWorkspaces()
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(workspaces)
			}
			printWorkspaceTable(workspaces)
			return nil
		},
	}
}

// molecli ws get <id>

func buildWSGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "get <id>",
		Short:        "Get a workspace by ID",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			ws, err := client.GetWorkspace(args[0])
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(ws)
			}
			printWorkspaceDetail(*ws)
			return nil
		},
	}
}

// molecli ws create

func buildWSCreateCmd() *cobra.Command {
	var (
		name     string
		role     string
		tier     int
		parentID string
	)

	cmd := &cobra.Command{
		Use:          "create",
		Short:        "Create a new workspace",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			client := NewPlatformClient(baseURL())
			resp, err := client.CreateWorkspace(CreateWorkspaceRequest{
				Name:     name,
				Role:     role,
				Tier:     tier,
				ParentID: parentID,
			})
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(resp)
			}
			fmt.Printf("Created workspace %s (status: %s)\n", resp.ID, resp.Status)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Workspace name (required)")
	cmd.Flags().StringVar(&role, "role", "", "Agent role")
	cmd.Flags().IntVar(&tier, "tier", 1, "Workspace tier")
	cmd.Flags().StringVar(&parentID, "parent", "", "Parent workspace ID")

	return cmd
}

// molecli ws update <id>

func buildWSUpdateCmd() *cobra.Command {
	var (
		name     string
		role     string
		tier     int
		parentID string
		hasTier  bool
	)

	cmd := &cobra.Command{
		Use:          "update <id>",
		Short:        "Update a workspace",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := UpdateWorkspaceRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("role") {
				req.Role = &role
			}
			if cmd.Flags().Changed("tier") {
				hasTier = true
				req.Tier = &tier
			}
			_ = hasTier
			if cmd.Flags().Changed("parent") {
				req.ParentID = &parentID
			}
			if req.Name == nil && req.Role == nil && req.Tier == nil && req.ParentID == nil {
				return fmt.Errorf("provide at least one flag to update (--name, --role, --tier, --parent)")
			}
			client := NewPlatformClient(baseURL())
			if err := client.UpdateWorkspace(args[0], req); err != nil {
				return err
			}
			if !flagJSON {
				fmt.Printf("Updated workspace %s\n", args[0])
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New workspace name")
	cmd.Flags().StringVar(&role, "role", "", "New agent role")
	cmd.Flags().IntVar(&tier, "tier", 0, "New workspace tier")
	cmd.Flags().StringVar(&parentID, "parent", "", "New parent workspace ID")

	return cmd
}

// molecli ws delete <id>

func buildWSDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:          "delete <id>",
		Aliases:      []string{"rm"},
		Short:        "Delete a workspace",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if !force {
				fmt.Printf("Delete workspace %s? [y/N] ", id)
				var answer string
				fmt.Scanln(&answer)
				if answer != "y" && answer != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}
			client := NewPlatformClient(baseURL())
			if err := client.DeleteWorkspace(id); err != nil {
				return err
			}
			if !flagJSON {
				fmt.Printf("Deleted workspace %s\n", id)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
