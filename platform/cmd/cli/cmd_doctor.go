package main

import (
	"context"

	"github.com/spf13/cobra"
)

func buildDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "doctor",
		Short:        "Run local platform preflight checks",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			report := runDoctor(context.Background(), baseURL())
			if flagJSON {
				if err := printJSON(report); err != nil {
					return err
				}
			} else {
				printDoctorReport(report)
			}

			if report.Summary.HasFailures {
				return newCLIExitError(1, "")
			}
			return nil
		},
	}
}
