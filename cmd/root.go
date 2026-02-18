package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCommand creates and returns the root cobra command.
func NewRootCommand(version, commit, date string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "claude-grid <count>",
		Short:         "Claude Grid - Terminal UI for managing AI agent sessions",
		Long:          "Claude Grid is a terminal UI application for managing and monitoring AI agent sessions.",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			showVersion, _ := cmd.Flags().GetBool("version")
			if showVersion {
				return nil
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			showVersion, _ := cmd.Flags().GetBool("version")
			if showVersion {
				fmt.Printf("claude-grid version %s (commit: %s, date: %s)\n", version, commit, date)
				return nil
			}
			return nil
		},
	}

	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	cmd.PersistentFlags().Bool("version", false, "Print version information")

	return cmd
}
