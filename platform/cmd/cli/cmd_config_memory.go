package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// ── molecli agent config ──────────────────────────────────────────────────────

func buildAgentConfigCmd() *cobra.Command {
	cfg := &cobra.Command{
		Use:   "config",
		Short: "Get or update agent configuration",
	}
	cfg.AddCommand(buildAgentConfigGetCmd())
	cfg.AddCommand(buildAgentConfigSetCmd())
	return cfg
}

func buildAgentConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "get <id>",
		Short:        "Show an agent's current configuration",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			data, err := client.GetConfig(args[0])
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
}

func buildAgentConfigSetCmd() *cobra.Command {
	var (
		inlineJSON string
		filePath   string
	)

	cmd := &cobra.Command{
		Use:   "set <id>",
		Short: "Merge a JSON patch into the agent's configuration",
		Example: `  molecli agent config set abc123 --json '{"system_prompt":"You are helpful"}'
  molecli agent config set abc123 --file config.json`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			patch, err := resolveCard(filePath, inlineJSON)
			if err != nil {
				return err
			}
			if patch == nil {
				return fmt.Errorf("provide --json or --file")
			}
			client := NewPlatformClient(baseURL())
			if err := client.PatchConfig(args[0], patch); err != nil {
				return err
			}
			fmt.Printf("Config updated for %s\n", shortID(args[0]))
			return nil
		},
	}

	cmd.Flags().StringVar(&inlineJSON, "json", "", "JSON patch as inline string")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "JSON patch from file")

	return cmd
}

// ── molecli agent memory ──────────────────────────────────────────────────────

func buildAgentMemoryCmd() *cobra.Command {
	mem := &cobra.Command{
		Use:   "memory",
		Short: "Manage agent memory (key-value store)",
	}
	mem.AddCommand(buildMemoryListCmd())
	mem.AddCommand(buildMemoryGetCmd())
	mem.AddCommand(buildMemorySetCmd())
	mem.AddCommand(buildMemoryDeleteCmd())
	return mem
}

func buildMemoryListCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "list <id>",
		Aliases:      []string{"ls"},
		Short:        "List all memory entries for an agent",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			entries, err := client.ListMemory(args[0])
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(entries)
			}
			if len(entries) == 0 {
				fmt.Println("(no memory entries)")
				return nil
			}
			tw := newTabWriter()
			fmt.Fprintln(tw, "KEY\tVALUE\tEXPIRES")
			fmt.Fprintln(tw, strings.Repeat("-", 20)+"\t"+strings.Repeat("-", 30)+"\t"+strings.Repeat("-", 20))
			for _, e := range entries {
				val := truncate(string(e.Value), 40)
				exp := "never"
				if e.ExpiresAt != nil {
					exp = e.ExpiresAt.Local().Format("2006-01-02 15:04:05")
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\n", e.Key, val, exp)
			}
			tw.Flush()
			return nil
		},
	}
}

func buildMemoryGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "get <id> <key>",
		Short:        "Get a single memory entry",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			entry, err := client.GetMemory(args[0], args[1])
			if err != nil {
				return err
			}
			if flagJSON {
				return printJSON(entry)
			}
			// Pretty-print the value
			var pretty any
			if err := json.Unmarshal(entry.Value, &pretty); err != nil {
				fmt.Println(string(entry.Value))
				return nil
			}
			return printJSON(pretty)
		},
	}
}

func buildMemorySetCmd() *cobra.Command {
	var ttl int

	cmd := &cobra.Command{
		Use:   "set <id> <key> <value>",
		Short: "Set a memory entry (value is a JSON string, number, or object)",
		Example: `  molecli agent memory set abc123 user_name '"Alice"'
  molecli agent memory set abc123 preferences '{"theme":"dark"}'
  molecli agent memory set abc123 session_token '"tok_xyz"' --ttl 3600`,
		Args:         cobra.ExactArgs(3),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rawValue := []byte(args[2])
			// If the value isn't valid JSON, treat it as a plain string
			if !json.Valid(rawValue) {
				quoted, err := json.Marshal(args[2])
				if err != nil {
					return fmt.Errorf("encode value: %w", err)
				}
				rawValue = quoted
			}
			client := NewPlatformClient(baseURL())
			if err := client.SetMemory(args[0], args[1], rawValue, ttl); err != nil {
				return err
			}
			fmt.Printf("Set %s = %s\n", args[1], truncate(string(rawValue), 60))
			return nil
		},
	}

	cmd.Flags().IntVar(&ttl, "ttl", 0, "Time-to-live in seconds (0 = no expiry)")

	return cmd
}

func buildMemoryDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "delete <id> <key>",
		Aliases:      []string{"rm"},
		Short:        "Delete a memory entry",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())
			if err := client.DeleteMemory(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Deleted memory key %q from %s\n", args[1], shortID(args[0]))
			return nil
		},
	}
}

// strconv import used for memory set value detection
var _ = strconv.Itoa
