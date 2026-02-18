package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// NewVersionCmd creates and returns the version cobra command.
func NewVersionCmd(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "claude-grid %s (%s/%s) commit:%s built:%s\n",
				version, runtime.GOOS, runtime.GOARCH, commit, date)
			return nil
		},
	}
}
