package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func buildChatCmd() *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:   "chat <id>",
		Short: "Chat with an agent via A2A protocol",
		Long: `Send messages to an agent using the A2A (Agent-to-Agent) protocol.

Without --message, starts an interactive REPL session.
With --message, sends a single message and exits.`,
		Example: `  molecli agent chat abc123
  molecli agent chat abc123 --message "Summarize the latest events"
  molecli agent chat abc123 -m "What can you do?"`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewPlatformClient(baseURL())

			// Discover agent URL
			disc, err := client.DiscoverWorkspace(args[0], "")
			if err != nil {
				return fmt.Errorf("discover agent: %w", err)
			}
			if disc.URL == "" {
				return fmt.Errorf("agent %s has no URL (is it online?)", args[0])
			}

			a2a := newA2AClient(disc.URL)

			if message != "" {
				return runOneShot(a2a, message)
			}
			return runREPL(a2a, disc.ID, disc.URL)
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "Send a single message and exit")

	return cmd
}

func runOneShot(a2a *a2aClient, message string) error {
	reply, err := a2a.SendTaskStreaming(message, func(chunk string) {
		fmt.Print(chunk)
	})
	if err != nil {
		return err
	}
	if reply != "" {
		fmt.Println()
	}
	return nil
}

func runREPL(a2a *a2aClient, agentID, agentURL string) error {
	fmt.Printf("Connected to %s (%s)\n", shortID(agentID), agentURL)
	fmt.Println("Type a message and press Enter. Ctrl+C or 'exit' to quit.")
	fmt.Println(strings.Repeat("─", 50))

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nyou> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			break
		}

		fmt.Print("agent> ")
		reply, err := a2a.SendTaskStreaming(input, func(chunk string) {
			fmt.Print(chunk)
		})
		if err != nil {
			fmt.Printf("\n[error] %v\n", err)
			continue
		}
		if reply != "" {
			fmt.Println()
		}
	}
	return scanner.Err()
}
