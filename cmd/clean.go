//go:build darwin

package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/riricardoMa/claude-grid/internal/git"
	"github.com/riricardoMa/claude-grid/internal/session"
)

func NewCleanCmd(storePath string) *cobra.Command {
	return &cobra.Command{
		Use:   "clean <session-name>",
		Short: "Clean a session by removing worktrees",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName := args[0]

			store := session.NewStore(storePath)
			sess, err := store.LoadSession(sessionName)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Session '%s' not found. Run 'claude-grid list' to see active sessions.\n", sessionName)
				return fmt.Errorf("session '%s' not found", sessionName)
			}

			if len(sess.Worktrees) == 0 {
				return fmt.Errorf("session '%s' has no worktrees to clean", sessionName)
			}

			manager, err := git.NewManager(sess.RepoPath)
			if err != nil {
				return fmt.Errorf("failed to create git manager for %q: %w", sess.RepoPath, err)
			}

			var errs []error
			var warnings []string
			removed := 0

			for _, wt := range sess.Worktrees {
				checkCmd := exec.Command("git", "-C", wt.Path, "status", "--porcelain")
				output, checkErr := checkCmd.CombinedOutput()
				if checkErr == nil && strings.TrimSpace(string(output)) != "" {
					warnings = append(warnings, fmt.Sprintf("worktree %q (%s) has uncommitted changes", wt.Path, wt.Branch))
				}

				if err := manager.RemoveWorktree(wt.Path); err != nil {
					errs = append(errs, fmt.Errorf("failed to remove worktree %q: %w", wt.Path, err))
				} else {
					removed++
				}
			}

			if pruneErr := manager.Prune(); pruneErr != nil {
				errs = append(errs, fmt.Errorf("failed to prune: %w", pruneErr))
			}

			if err := store.DeleteSession(sessionName); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to delete session file: %v\n", err)
			}

			for _, w := range warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", w)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Session '%s' cleaned. %d/%d worktrees removed.\n", sessionName, removed, len(sess.Worktrees))

			if len(errs) > 0 {
				msgs := make([]string, len(errs))
				for i, e := range errs {
					msgs[i] = e.Error()
				}
				return fmt.Errorf("clean completed with errors: %s", strings.Join(msgs, "; "))
			}

			return nil
		},
	}
}
