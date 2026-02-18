package main

import (
	"os"

	"github.com/riricardoMa/claude-grid/cmd"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	rootCmd := cmd.NewRootCommand(version, commit, date)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
