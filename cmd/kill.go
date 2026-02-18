//go:build darwin

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/riricardoMa/claude-grid/internal/script"
	"github.com/riricardoMa/claude-grid/internal/session"
	"github.com/riricardoMa/claude-grid/internal/terminal"
)

func NewKillCmd(storePath string, executor script.ScriptExecutor) *cobra.Command {
	return &cobra.Command{
		Use:   "kill <session-name>",
		Short: "Kill a session and close all its windows",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName := args[0]

			store := session.NewStore(storePath)
			sess, err := store.LoadSession(sessionName)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Session '%s' not found. Run 'claude-grid list' to see active sessions.\n", sessionName)
				return fmt.Errorf("session '%s' not found", sessionName)
			}

			var backend terminal.TerminalBackend
			switch sess.Backend {
			case "terminal":
				backend = terminal.NewTerminalAppBackend(executor)
			case "warp":
				backend = terminal.NewWarpBackend(executor)
			default:
				return fmt.Errorf("unknown backend: %s", sess.Backend)
			}

			if err := backend.CloseSession(sessionName); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to close windows: %v\n", err)
			}

			if err := store.DeleteSession(sessionName); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to delete session file: %v\n", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Session '%s' killed. %d windows closed.\n", sessionName, len(sess.Windows))
			return nil
		},
	}
}
