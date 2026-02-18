package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/riricardoMa/claude-grid/internal/grid"
	"github.com/riricardoMa/claude-grid/internal/screen"
	"github.com/riricardoMa/claude-grid/internal/script"
	"github.com/riricardoMa/claude-grid/internal/session"
	"github.com/riricardoMa/claude-grid/internal/terminal"
)

// NewRootCommand creates and returns the root cobra command.
func NewRootCommand(version, commit, date string) *cobra.Command {
	var (
		terminalFlag string
		dirFlag      string
		nameFlag     string
		layoutFlag   string
	)

	cmd := &cobra.Command{
		Use:           "claude-grid <count>",
		Short:         "Claude Grid - Terminal UI for managing AI agent sessions",
		Long:          "Claude Grid is a terminal UI application for managing and monitoring AI agent sessions.",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stderr := cmd.ErrOrStderr()
			stdout := cmd.OutOrStdout()

			showVersion, _ := cmd.Flags().GetBool("version")
			if showVersion {
				fmt.Fprintf(stdout, "claude-grid version %s (commit: %s, date: %s)\n", version, commit, date)
				return nil
			}

			if len(args) != 1 {
				fmt.Fprintln(stderr, "count argument is required: claude-grid <count>")
				return fmt.Errorf("invalid arguments")
			}

			count, err := strconv.Atoi(strings.TrimSpace(args[0]))
			if err != nil {
				fmt.Fprintf(stderr, "invalid count %q: must be a number between 1 and 16\n", args[0])
				return fmt.Errorf("invalid count")
			}
			if count < 1 || count > 16 {
				fmt.Fprintf(stderr, "invalid count %d: must be between 1 and 16\n", count)
				return fmt.Errorf("invalid count")
			}

			claudePath, err := exec.LookPath("claude")
			if err != nil {
				fmt.Fprintln(stderr, "'claude' not found in PATH. Install: npm install -g @anthropic-ai/claude-code")
				fmt.Fprintln(stderr, "Or specify a different location (v0.2).")
				return fmt.Errorf("claude not found")
			}

			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(stderr, "failed to determine current directory: %v\n", err)
				return fmt.Errorf("get working directory: %w", err)
			}

			resolvedDir := dirFlag
			if strings.TrimSpace(resolvedDir) == "" {
				resolvedDir = cwd
			}
			resolvedDir, err = filepath.Abs(resolvedDir)
			if err != nil {
				fmt.Fprintf(stderr, "failed to resolve directory %q: %v\n", dirFlag, err)
				return fmt.Errorf("resolve directory: %w", err)
			}

			executor := script.NewOSAExecutor()
			screenInfo, err := screen.DetectScreen(executor)
			if err != nil {
				fmt.Fprintf(stderr, "warning: failed to detect screen, using fallback 1920x1080: %v\n", err)
				screenInfo = screen.ScreenInfo{X: 0, Y: 0, Width: 1920, Height: 1080}
			}

			var gridLayout grid.GridLayout
			if strings.TrimSpace(layoutFlag) != "" {
				gridLayout, err = grid.ParseLayout(layoutFlag)
				if err != nil {
					fmt.Fprintf(stderr, "invalid layout %q: %v\n", layoutFlag, err)
					return fmt.Errorf("parse layout: %w", err)
				}
			} else {
				gridLayout = grid.CalculateGrid(count)
			}

			bounds := grid.CalculateWindowBounds(gridLayout, grid.ScreenInfo{
				X:      screenInfo.X,
				Y:      screenInfo.Y,
				Width:  screenInfo.Width,
				Height: screenInfo.Height,
			}, count)

			minWidth := 0
			minHeight := 0
			for i := 0; i < count && i < len(bounds); i++ {
				if i == 0 || bounds[i].Width < minWidth {
					minWidth = bounds[i].Width
				}
				if i == 0 || bounds[i].Height < minHeight {
					minHeight = bounds[i].Height
				}
			}
			if minWidth < 400 || minHeight < 200 {
				fmt.Fprintf(stderr, "warning: small windows detected (%dx%d minimum). Readability may be reduced.\n", minWidth, minHeight)
			}

			backend, err := terminal.DetectBackend(terminalFlag)
			if err != nil {
				fmt.Fprintf(stderr, "failed to select terminal backend: %v\n", err)
				fmt.Fprintln(stderr, "Try --terminal terminal or install Warp.")
				return fmt.Errorf("detect backend: %w", err)
			}

			store := session.NewStore("")
			sessionName := strings.TrimSpace(nameFlag)
			if sessionName == "" {
				sessionName = store.GenerateSessionName()
			}

			if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
				fmt.Fprintf(stdout, "Verbose: claude=%s backend=%s\n", claudePath, backend.Name())
			}

			fmt.Fprintf(stdout, "Detected: %s, terminal %s, screen %dx%d\n", runtime.GOOS, backend.Name(), screenInfo.Width, screenInfo.Height)
			fmt.Fprintf(stdout, "Layout: %dx%d grid (%dx%d per window)\n", gridLayout.Rows, gridLayout.Cols, minWidth, minHeight)
			fmt.Fprintf(stdout, "Directory: %s\n", resolvedDir)
			fmt.Fprintf(stdout, "Spawning %d Claude Code instances...\n", count)

			spawnOptions := terminal.SpawnOptions{
				Count:     count,
				Command:   "claude",
				Dir:       resolvedDir,
				Grid:      gridLayout,
				Screen:    screenInfo,
				Bounds:    bounds,
				SessionID: sessionName,
			}

			windows, err := backend.SpawnWindows(cmd.Context(), spawnOptions)
			if err != nil {
				_ = backend.CloseSession(sessionName)
				fmt.Fprintf(stderr, "failed to spawn windows: %v\n", err)
				return fmt.Errorf("spawn windows: %w", err)
			}

			sessionWindows := make([]session.WindowRef, 0, len(windows))
			for _, window := range windows {
				sessionWindows = append(sessionWindows, session.WindowRef{ID: window.ID, Index: window.Index})
			}

			err = store.SaveSession(session.Session{
				Name:      sessionName,
				Backend:   backend.Name(),
				Count:     count,
				Dir:       resolvedDir,
				CreatedAt: time.Now(),
				Windows:   sessionWindows,
			})
			if err != nil {
				_ = backend.CloseSession(sessionName)
				fmt.Fprintf(stderr, "failed to save session: %v\n", err)
				return fmt.Errorf("save session: %w", err)
			}

			fmt.Fprintf(stdout, "Session %q created. Use `claude-grid kill %s` to close all.\n", sessionName, sessionName)
			return nil
		},
	}

	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	cmd.PersistentFlags().Bool("version", false, "Print version information")
	cmd.Flags().StringVarP(&terminalFlag, "terminal", "t", "", "Terminal backend: terminal, warp (default: auto-detect)")
	cmd.Flags().StringVarP(&dirFlag, "dir", "d", "", "Working directory (default: current directory)")
	cmd.Flags().StringVarP(&nameFlag, "name", "n", "", "Session name (default: auto-generated)")
	cmd.Flags().StringVarP(&layoutFlag, "layout", "l", "", "Grid layout, e.g. 2x3 (default: auto)")
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if len(os.Args) >= 2 {
			candidate := strings.TrimSpace(os.Args[1])
			if parsed, convErr := strconv.Atoi(candidate); convErr == nil {
				if parsed < 1 || parsed > 16 {
					fmt.Fprintf(c.ErrOrStderr(), "invalid count %d: must be between 1 and 16\n", parsed)
					return fmt.Errorf("invalid count")
				}
			}
		}
		fmt.Fprintln(c.ErrOrStderr(), err)
		return err
	})

	cmd.AddCommand(NewVersionCmd(version, commit, date))
	cmd.AddCommand(NewListCmd("", script.NewOSAExecutor()))
	cmd.AddCommand(NewKillCmd("", script.NewOSAExecutor()))

	return cmd
}
