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

	"github.com/riricardoMa/claude-grid/internal/git"
	"github.com/riricardoMa/claude-grid/internal/grid"
	"github.com/riricardoMa/claude-grid/internal/manifest"
	"github.com/riricardoMa/claude-grid/internal/pathutil"
	"github.com/riricardoMa/claude-grid/internal/screen"
	"github.com/riricardoMa/claude-grid/internal/script"
	"github.com/riricardoMa/claude-grid/internal/session"
	"github.com/riricardoMa/claude-grid/internal/terminal"
	"github.com/spf13/cobra"
)

// NewRootCommand creates and returns the root cobra command.
func NewRootCommand(version, commit, date string) *cobra.Command {
	var (
		terminalFlag     string
		dirFlags         []string
		promptFlags      []string
		manifestFlag     string
		nameFlag         string
		layoutFlag       string
		worktreesFlag    bool
		branchPrefixFlag string
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

			// Manifest conflict detection
			if manifestFlag != "" {
				if len(dirFlags) > 0 || len(promptFlags) > 0 || worktreesFlag || len(args) > 0 {
					fmt.Fprintln(stderr, "--manifest cannot be combined with --dir, --prompt, --worktrees, or count argument")
					return fmt.Errorf("conflicting flags")
				}
			}

			// Count determination
			var count int
			var parsedManifest manifest.Manifest

			if manifestFlag != "" {
				expandedManifestPath, err := pathutil.ExpandTilde(manifestFlag)
				if err != nil {
					fmt.Fprintf(stderr, "invalid manifest path %q: %v\n", manifestFlag, err)
					return fmt.Errorf("invalid manifest path: %w", err)
				}
				absManifestPath, err := filepath.Abs(expandedManifestPath)
				if err != nil {
					fmt.Fprintf(stderr, "failed to resolve manifest path %q: %v\n", manifestFlag, err)
					return fmt.Errorf("resolve manifest path: %w", err)
				}
				m, err := manifest.Parse(absManifestPath)
				if err != nil {
					fmt.Fprintf(stderr, "failed to parse manifest %q: %v\n", manifestFlag, err)
					return fmt.Errorf("parse manifest: %w", err)
				}
				parsedManifest = m
				count = len(parsedManifest.Instances)
			} else if len(args) == 1 {
				c, err := strconv.Atoi(strings.TrimSpace(args[0]))
				if err != nil {
					fmt.Fprintf(stderr, "invalid count %q: must be a number between 1 and 16\n", args[0])
					return fmt.Errorf("invalid count")
				}
				if c < 1 || c > 16 {
					fmt.Fprintf(stderr, "invalid count %d: must be between 1 and 16\n", c)
					return fmt.Errorf("invalid count")
				}
				count = c
			} else if len(args) == 0 && len(dirFlags) > 0 {
				count = len(dirFlags)
			} else {
				fmt.Fprintln(stderr, "count argument is required: claude-grid <count>")
				return fmt.Errorf("invalid arguments")
			}

			// Validate count range for non-manifest paths
			if manifestFlag == "" && (count < 1 || count > 16) {
				fmt.Fprintf(stderr, "invalid count %d: must be between 1 and 16\n", count)
				return fmt.Errorf("invalid count")
			}

			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(stderr, "failed to determine current directory: %v\n", err)
				return fmt.Errorf("get working directory: %w", err)
			}

			// Dir resolution
			var resolvedDirs []string
			var resolvedPrompts []string

			if manifestFlag != "" {
				resolvedDirs = make([]string, len(parsedManifest.Instances))
				resolvedPrompts = make([]string, len(parsedManifest.Instances))
				for i, inst := range parsedManifest.Instances {
					resolvedDirs[i] = inst.Dir
					resolvedPrompts[i] = inst.Prompt
				}
			} else if len(dirFlags) > 0 {
				expanded, err := pathutil.ExpandTildeAll(dirFlags)
				if err != nil {
					fmt.Fprintf(stderr, "invalid directory: %v\n", err)
					return fmt.Errorf("expand dir: %w", err)
				}
				absDirs := make([]string, len(expanded))
				for i, d := range expanded {
					abs, err := filepath.Abs(d)
					if err != nil {
						fmt.Fprintf(stderr, "failed to resolve directory %q: %v\n", d, err)
						return fmt.Errorf("resolve directory: %w", err)
					}
					absDirs[i] = abs
				}

				if len(absDirs) > count {
					fmt.Fprintf(stderr, "more --dir flags (%d) than instances (%d)\n", len(absDirs), count)
					return fmt.Errorf("too many dirs")
				}

				resolvedDirs = make([]string, count)
				copy(resolvedDirs, absDirs)
				for i := len(absDirs); i < count; i++ {
					resolvedDirs[i] = absDirs[len(absDirs)-1]
				}
			} else {
				resolvedDirs = make([]string, count)
				for i := range resolvedDirs {
					resolvedDirs[i] = cwd
				}
			}

			// Prompt resolution
			if manifestFlag == "" {
				resolvedPrompts = make([]string, count)
				if len(promptFlags) > count {
					fmt.Fprintf(stderr, "more --prompt flags (%d) than instances (%d)\n", len(promptFlags), count)
					return fmt.Errorf("too many prompts")
				}
				copy(resolvedPrompts, promptFlags)
			}

			// Directory existence validation
			for _, d := range resolvedDirs {
				if _, err := os.Stat(d); err != nil {
					fmt.Fprintf(stderr, "directory does not exist: %s\n", d)
					return fmt.Errorf("directory does not exist: %s", d)
				}
			}

			resolvedDir := resolvedDirs[0]

			// Branch checkout for manifest instances
			if manifestFlag != "" {
				for i, inst := range parsedManifest.Instances {
					if inst.Branch == "" {
						continue
					}
					checkoutCmd := exec.CommandContext(cmd.Context(), "git", "-C", resolvedDirs[i], "checkout", inst.Branch)
					if out, err := checkoutCmd.CombinedOutput(); err != nil {
						fmt.Fprintf(stderr, "failed to checkout branch %q in %s: %v\n%s", inst.Branch, resolvedDirs[i], err, string(out))
						return fmt.Errorf("checkout branch %q in %s: %w", inst.Branch, resolvedDirs[i], err)
					}
				}
			}

			claudePath, err := exec.LookPath("claude")
			if err != nil {
				fmt.Fprintln(stderr, "'claude' not found in PATH. Install: npm install -g @anthropic-ai/claude-code")
				fmt.Fprintln(stderr, "Or specify a different location (v0.2).")
				return fmt.Errorf("claude not found")
			}

			var worktreeDirs []string
			var worktreeRefs []session.WorktreeRef
			var repoPath string
			var cleanupWorktrees func()
			spawnSucceeded := false
			defer func() {
				if cleanupWorktrees != nil && !spawnSucceeded {
					cleanupWorktrees()
				}
			}()

			if worktreesFlag {
				manager, err := git.NewManager(resolvedDir)
				if err != nil {
					fmt.Fprintf(stderr, "failed to initialize git worktree manager: %v\n", err)
					return fmt.Errorf("init worktree manager: %w", err)
				}

				if manager.DetectSubmodules() {
					fmt.Fprintln(stderr, "warning: git submodules detected; worktree operations may require additional setup")
				}

				prefix := strings.TrimSpace(branchPrefixFlag)
				if prefix == "" {
					prefix = git.GenerateBranchPrefix()
				} else if err := git.ValidateBranchPrefix(prefix); err != nil {
					fmt.Fprintf(stderr, "invalid branch prefix %q: %v\n", branchPrefixFlag, err)
					return fmt.Errorf("validate branch prefix: %w", err)
				}

				repoPath = manager.RepoPath()
				worktreeDirs = make([]string, 0, count)
				worktreeRefs = make([]session.WorktreeRef, 0, count)

				cleanupWorktrees = func() {
					for _, ref := range worktreeRefs {
						if err := manager.RemoveWorktree(ref.Path); err != nil {
							fmt.Fprintf(stderr, "warning: failed to clean up worktree %q: %v\n", ref.Path, err)
						}
					}
				}

				for i := 0; i < count; i++ {
					branch := fmt.Sprintf("%s-%d", prefix, i+1)
					path, err := manager.CreateWorktree(branch)
					if err != nil {
						fmt.Fprintf(stderr, "failed to create worktree for branch %q: %v\n", branch, err)
						return fmt.Errorf("create worktree: %w", err)
					}

					worktreeDirs = append(worktreeDirs, path)
					worktreeRefs = append(worktreeRefs, session.WorktreeRef{Path: path, Branch: branch})
				}
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
			if allDirsSame(resolvedDirs) {
				fmt.Fprintf(stdout, "Directory: %s\n", resolvedDir)
			} else {
				fmt.Fprintf(stdout, "Directories: %d different directories\n", len(resolvedDirs))
			}
			fmt.Fprintf(stdout, "Spawning %d Claude Code instances...\n", count)

			spawnOptions := terminal.SpawnOptions{
				Count:     count,
				Command:   "claude",
				Dir:       resolvedDir,
				Dirs:      resolvedDirs,
				Prompts:   resolvedPrompts,
				Grid:      gridLayout,
				Screen:    screenInfo,
				Bounds:    bounds,
				SessionID: sessionName,
			}
			if len(worktreeDirs) > 0 {
				spawnOptions.Dirs = worktreeDirs
			}

			windows, err := backend.SpawnWindows(cmd.Context(), spawnOptions)
			if err != nil {
				if cleanupWorktrees != nil {
					cleanupWorktrees()
					cleanupWorktrees = nil
				}
				_ = backend.CloseSession(sessionName)
				fmt.Fprintf(stderr, "failed to spawn windows: %v\n", err)
				return fmt.Errorf("spawn windows: %w", err)
			}
			spawnSucceeded = true

			sessionWindows := make([]session.WindowRef, 0, len(windows))
			for _, window := range windows {
				sessionWindows = append(sessionWindows, session.WindowRef{ID: window.ID, Index: window.Index})
			}

			sess := session.Session{
				Name:      sessionName,
				Backend:   backend.Name(),
				Count:     count,
				Dir:       resolvedDir,
				Dirs:      resolvedDirs,
				Prompts:   resolvedPrompts,
				CreatedAt: time.Now(),
				Windows:   sessionWindows,
			}
			if manifestFlag != "" {
				sess.ManifestPath = manifestFlag
			}
			if len(worktreeRefs) > 0 {
				sess.Worktrees = worktreeRefs
				sess.Status = "active"
				sess.RepoPath = repoPath
			}

			err = store.SaveSession(sess)
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
	cmd.Flags().StringArrayVarP(&dirFlags, "dir", "d", nil, "Working directory (repeatable); infers count")
	cmd.Flags().StringArrayVar(&promptFlags, "prompt", nil, "Per-instance prompt (repeatable; paired with --dir by index)")
	cmd.Flags().StringVarP(&manifestFlag, "manifest", "M", "", "YAML manifest file defining instances")
	cmd.Flags().StringVarP(&nameFlag, "name", "n", "", "Session name (default: auto-generated)")
	cmd.Flags().StringVarP(&layoutFlag, "layout", "l", "", "Grid layout, e.g. 2x3 (default: auto)")
	cmd.Flags().BoolVarP(&worktreesFlag, "worktrees", "w", false, "Create git worktrees for each window")
	cmd.Flags().StringVarP(&branchPrefixFlag, "branch-prefix", "b", "", "Branch prefix for worktrees (default: auto-generated)")
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
	cmd.AddCommand(NewCleanCmd(""))

	return cmd
}

func allDirsSame(ss []string) bool {
	if len(ss) <= 1 {
		return true
	}
	for _, s := range ss[1:] {
		if s != ss[0] {
			return false
		}
	}
	return true
}
