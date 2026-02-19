package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/riricardoMa/claude-grid/internal/script"
	"github.com/riricardoMa/claude-grid/internal/session"
)

func NewListCmd(storePath string, executor script.ScriptExecutor) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all active sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := session.NewStore(storePath)
			sessions, err := store.ListSessions()
			if err != nil {
				return fmt.Errorf("failed to list sessions: %w", err)
			}

			if len(sessions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No active sessions.")
				return nil
			}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SESSION\tSTATUS\tBACKEND\tWINDOWS\tDIR\tCREATED")

		for _, sess := range sessions {
			isLive := checkSessionLiveness(cmd.Context(), executor, sess)
			statusCol := sess.Status
			if statusCol == "" {
				statusCol = "active"
			}
			if !isLive {
				statusCol = statusCol + " (stale)"
			}

			displayDir := sess.Dir
			if strings.HasPrefix(displayDir, os.ExpandEnv("$HOME")) {
				displayDir = "~" + strings.TrimPrefix(displayDir, os.ExpandEnv("$HOME"))
			}

			createdStr := sess.CreatedAt.Format("2006-01-02 15:04")
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				sess.Name, statusCol, sess.Backend, len(sess.Windows),
				displayDir, createdStr)
		}

			w.Flush()
			return nil
		},
	}
}

func checkSessionLiveness(ctx context.Context, executor script.ScriptExecutor, sess session.Session) bool {
	if executor == nil {
		return true
	}

	switch sess.Backend {
	case "terminal":
		return checkTerminalLiveness(ctx, executor, sess)
	case "warp":
		return checkWarpLiveness(ctx, executor)
	default:
		return true
	}
}

func checkTerminalLiveness(ctx context.Context, executor script.ScriptExecutor, sess session.Session) bool {
	script := `tell application "Terminal" to get id of every window`
	output, err := executor.RunAppleScript(ctx, script)
	if err != nil {
		return false
	}

	windowIDsStr := strings.Split(output, ", ")
	windowIDsMap := make(map[string]bool)
	for _, id := range windowIDsStr {
		windowIDsMap[strings.TrimSpace(id)] = true
	}

	for _, winRef := range sess.Windows {
		if windowIDsMap[winRef.ID] {
			return true
		}
	}

	return false
}

func checkWarpLiveness(ctx context.Context, executor script.ScriptExecutor) bool {
	script := `tell application "System Events" to tell process "Warp" to count windows`
	output, err := executor.RunAppleScript(ctx, script)
	if err != nil {
		return false
	}

	output = strings.TrimSpace(output)
	return output != "0" && output != ""
}
