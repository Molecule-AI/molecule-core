package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func buildAgentCmd() *cobra.Command {
	agent := &cobra.Command{
		Use:   "agent",
		Short: "Spawn and manage agents",
		Long: `High-level agent management commands.

These commands operate on the full agent lifecycle — workspace creation,
agent card configuration, role and tier assignment — in a single step.

For lower-level workspace operations use the 'ws' subcommand.
For registry/discovery use the 'registry' subcommand.`,
	}

	agent.AddCommand(buildAgentSpawnCmd())
	agent.AddCommand(buildAgentEditCmd())
	agent.AddCommand(buildAgentCardCmd())
	agent.AddCommand(buildChatCmd())
	agent.AddCommand(buildAgentConfigCmd())
	agent.AddCommand(buildAgentMemoryCmd())
	agent.AddCommand(buildAgentSessionCmd())
	agent.AddCommand(buildAgentSkillCmd())

	return agent
}

// ── molecli agent spawn ───────────────────────────────────────────────────────

func buildAgentSpawnCmd() *cobra.Command {
	var (
		name     string
		role     string
		tier     int
		parentID string
		cardFile string
		cardJSON string
	)

	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Create a new agent workspace (optionally with an initial agent card)",
		Example: `  molecli agent spawn --name "Echo Agent" --role worker --tier 1
  molecli agent spawn --name "Planner" --card card.json
  molecli agent spawn --name "Analyst" --card-json '{"name":"Analyst","skills":[]}'`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			card, err := resolveCard(cardFile, cardJSON)
			if err != nil {
				return err
			}

			client := NewPlatformClient(baseURL())

			// Create the workspace
			resp, err := client.CreateWorkspace(CreateWorkspaceRequest{
				Name:     name,
				Role:     role,
				Tier:     tier,
				ParentID: parentID,
			})
			if err != nil {
				return err
			}

			// Optionally set the initial agent card
			if card != nil {
				if err := client.UpdateAgentCard(resp.ID, card); err != nil {
					// Workspace was created — warn but don't fail
					fmt.Fprintf(os.Stderr, "warning: workspace created (%s) but agent card upload failed: %v\n", resp.ID, err)
				}
			}

			if flagJSON {
				return printJSON(resp)
			}
			fmt.Printf("Spawned agent %s (status: %s)\n", resp.ID, resp.Status)
			if card != nil {
				fmt.Printf("Agent card set.\n")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Agent name (required)")
	cmd.Flags().StringVar(&role, "role", "", "Agent role (e.g. worker, planner, coordinator)")
	cmd.Flags().IntVar(&tier, "tier", 1, "Workspace tier")
	cmd.Flags().StringVar(&parentID, "parent", "", "Parent workspace ID")
	cmd.Flags().StringVar(&cardFile, "card", "", "Path to agent card JSON file")
	cmd.Flags().StringVar(&cardJSON, "card-json", "", "Agent card as an inline JSON string")

	return cmd
}

// ── molecli agent edit <id> ───────────────────────────────────────────────────

func buildAgentEditCmd() *cobra.Command {
	var (
		name     string
		role     string
		tier     int
		parentID string
		cardFile string
		cardJSON string
	)

	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Update an agent's workspace properties and/or agent card",
		Example: `  molecli agent edit abc123 --role coordinator
  molecli agent edit abc123 --name "New Name" --card updated-card.json
  molecli agent edit abc123 --card-json '{"name":"Echo","skills":[{"id":"echo","name":"Echo"}]}'`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			card, err := resolveCard(cardFile, cardJSON)
			if err != nil {
				return err
			}

			// Build workspace update (only set fields that were explicitly passed)
			req := UpdateWorkspaceRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("role") {
				req.Role = &role
			}
			if cmd.Flags().Changed("tier") {
				req.Tier = &tier
			}
			if cmd.Flags().Changed("parent") {
				req.ParentID = &parentID
			}

			if req.Name == nil && req.Role == nil && req.Tier == nil && req.ParentID == nil && card == nil {
				return fmt.Errorf("provide at least one flag to update (--name, --role, --tier, --parent, --card, --card-json)")
			}

			client := NewPlatformClient(baseURL())

			// Patch workspace fields if any were provided
			if req.Name != nil || req.Role != nil || req.Tier != nil || req.ParentID != nil {
				if err := client.UpdateWorkspace(id, req); err != nil {
					return err
				}
			}

			// Update agent card if provided
			if card != nil {
				if err := client.UpdateAgentCard(id, card); err != nil {
					return err
				}
			}

			if !flagJSON {
				parts := []string{}
				if req.Name != nil || req.Role != nil || req.Tier != nil || req.ParentID != nil {
					parts = append(parts, "workspace properties")
				}
				if card != nil {
					parts = append(parts, "agent card")
				}
				fmt.Printf("Updated %s: %s\n", shortID(id), strings.Join(parts, " and "))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New agent name")
	cmd.Flags().StringVar(&role, "role", "", "New agent role")
	cmd.Flags().IntVar(&tier, "tier", 0, "New workspace tier")
	cmd.Flags().StringVar(&parentID, "parent", "", "New parent workspace ID")
	cmd.Flags().StringVar(&cardFile, "card", "", "Path to agent card JSON file")
	cmd.Flags().StringVar(&cardJSON, "card-json", "", "Agent card as an inline JSON string")

	return cmd
}

// ── molecli agent card ────────────────────────────────────────────────────────

func buildAgentCardCmd() *cobra.Command {
	card := &cobra.Command{
		Use:   "card",
		Short: "View or update an agent's card",
	}
	card.AddCommand(buildAgentCardGetCmd())
	card.AddCommand(buildAgentCardSetCmd())
	return card
}

func buildAgentCardGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "get <id>",
		Short:        "Show an agent's current card",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			ws, err := client.GetWorkspace(args[0])
			if err != nil {
				return err
			}
			if len(ws.AgentCard) == 0 || string(ws.AgentCard) == "null" {
				fmt.Println("(no agent card set)")
				return nil
			}
			// Pretty-print the raw JSON
			var pretty any
			if err := json.Unmarshal(ws.AgentCard, &pretty); err != nil {
				// Fall back to raw bytes if not valid JSON
				fmt.Println(string(ws.AgentCard))
				return nil
			}
			return printJSON(pretty)
		},
	}
}

func buildAgentCardSetCmd() *cobra.Command {
	var (
		cardFile string
		cardJSON string
	)

	cmd := &cobra.Command{
		Use:   "set <id>",
		Short: "Replace an agent's card",
		Example: `  molecli agent card set abc123 --file card.json
  molecli agent card set abc123 --json '{"name":"Echo","skills":[]}'`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			card, err := resolveCard(cardFile, cardJSON)
			if err != nil {
				return err
			}
			if card == nil {
				return fmt.Errorf("provide --file or --json")
			}
			client := NewPlatformClient(baseURL())
			if err := client.UpdateAgentCard(args[0], card); err != nil {
				return err
			}
			fmt.Printf("Agent card updated for %s\n", shortID(args[0]))
			return nil
		},
	}

	cmd.Flags().StringVarP(&cardFile, "file", "f", "", "Path to agent card JSON file")
	cmd.Flags().StringVar(&cardJSON, "json", "", "Agent card as an inline JSON string")

	return cmd
}

// ── helpers ───────────────────────────────────────────────────────────────────

// resolveCard reads an agent card from a file path or inline JSON string.
// Returns nil if neither is provided.
func resolveCard(filePath, inlineJSON string) (json.RawMessage, error) {
	if filePath != "" && inlineJSON != "" {
		return nil, fmt.Errorf("provide --card/--file or --card-json/--json, not both")
	}
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read card file: %w", err)
		}
		if !json.Valid(data) {
			return nil, fmt.Errorf("card file is not valid JSON")
		}
		return json.RawMessage(data), nil
	}
	if inlineJSON != "" {
		if !json.Valid([]byte(inlineJSON)) {
			return nil, fmt.Errorf("--card-json / --json is not valid JSON")
		}
		return json.RawMessage(inlineJSON), nil
	}
	return nil, nil
}
