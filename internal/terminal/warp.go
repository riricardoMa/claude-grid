//go:build darwin

package terminal

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/riricardoMa/claude-grid/internal/grid"
	"github.com/riricardoMa/claude-grid/internal/script"
)

const (
	warpAppPath            = "/Applications/Warp.app"
	warpSpawnDelay         = 500 * time.Millisecond
	warpInitialBackoff     = 100 * time.Millisecond
	warpMaxBackoff         = time.Second
	warpWaitTimeout        = 15 * time.Second
	warpFreshLaunchDelay   = 3 * time.Second
	warpAccessibilityGuide = "Accessibility permission required. Go to System Settings > Privacy & Security > Accessibility and add your terminal app"
)

var execCommandContext = exec.CommandContext

var _ TerminalBackend = (*WarpBackend)(nil)

type WarpBackend struct {
	executor script.ScriptExecutor
	statFn   func(path string) (os.FileInfo, error)
	runOpen  func(ctx context.Context, uri string) error
	sleepFn  func(d time.Duration)

	isWarpRunningFn    func(ctx context.Context) (bool, error)
	waitForWindowCountFn func(ctx context.Context, target int) error
	tileWindowsFn      func(ctx context.Context, bounds []grid.WindowBounds) error
}

func NewWarpBackend(executor script.ScriptExecutor) *WarpBackend {
	b := &WarpBackend{
		executor: executor,
		statFn:   os.Stat,
		sleepFn:  time.Sleep,
	}
	b.runOpen = b.defaultRunOpen
	b.isWarpRunningFn = b.isWarpRunning
	b.waitForWindowCountFn = b.waitForWindowCount
	b.tileWindowsFn = b.tileWindows
	return b
}

func (b *WarpBackend) Name() string {
	return "warp"
}

func (b *WarpBackend) Available() bool {
	_, err := b.statFn(warpAppPath)
	return err == nil
}

func (b *WarpBackend) SpawnWindows(ctx context.Context, opts SpawnOptions) ([]WindowInfo, error) {
	if opts.Count <= 0 {
		return nil, fmt.Errorf("count must be > 0")
	}
	if len(opts.Bounds) < opts.Count {
		return nil, fmt.Errorf("insufficient bounds: got %d, need %d", len(opts.Bounds), opts.Count)
	}

	wasRunning, err := b.isWarpRunningFn(ctx)
	if err != nil {
		return nil, err
	}

	var dirs []string
	if len(opts.Dirs) > 0 {
		dirs = opts.Dirs
	} else {
		dirs = make([]string, opts.Count)
		for i := range dirs {
			dirs[i] = opts.Dir
		}
	}

	for i := 0; i < opts.Count; i++ {
		encodedPath := strings.ReplaceAll(url.PathEscape(dirs[i]), "%2F", "/")
		uri := fmt.Sprintf("warp://action/new_window?path=%s", encodedPath)
		if err := b.runOpen(ctx, uri); err != nil {
			return nil, fmt.Errorf("open warp uri: %w", err)
		}
		if i < opts.Count-1 {
			b.sleepFn(warpSpawnDelay)
		}
	}

	if !wasRunning {
		b.sleepFn(warpFreshLaunchDelay)
	}

	if err := b.waitForWindowCountFn(ctx, opts.Count); err != nil {
		return nil, err
	}

	if err := b.tileWindowsFn(ctx, opts.Bounds[:opts.Count]); err != nil {
		return nil, err
	}

	command := opts.Command
	if strings.TrimSpace(command) == "" {
		command = "claude"
	}
	if err := b.sendCommandToWindows(ctx, opts.Count, command); err != nil {
		return nil, fmt.Errorf("send command to warp windows: %w", err)
	}

	windows := make([]WindowInfo, opts.Count)
	for i := 0; i < opts.Count; i++ {
		windows[i] = WindowInfo{
			ID:      strconv.Itoa(i + 1),
			Index:   i,
			Backend: b.Name(),
		}
	}

	return windows, nil
}

func (b *WarpBackend) CloseSession(sessionID string) error {
	_ = sessionID
	scriptText := strings.Join([]string{
		"tell application \"System Events\"",
		"  tell process \"Warp\"",
		"    repeat while (count windows) > 0",
		"      close window 1",
		"    end repeat",
		"  end tell",
		"end tell",
	}, "\n")

	_, err := b.executor.RunAppleScript(context.Background(), scriptText)
	if err != nil {
		return wrapAccessibilityError(fmt.Errorf("close warp windows: %w", err))
	}
	return nil
}

func (b *WarpBackend) defaultRunOpen(ctx context.Context, uri string) error {
	cmd := execCommandContext(ctx, "open", uri)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (b *WarpBackend) isWarpRunning(ctx context.Context) (bool, error) {
	output, err := b.executor.RunAppleScript(ctx, `tell application "System Events" to (name of processes) contains "Warp"`)
	if err != nil {
		return false, wrapAccessibilityError(fmt.Errorf("check warp process: %w", err))
	}

	value := strings.TrimSpace(strings.ToLower(output))
	return value == "true", nil
}

func (b *WarpBackend) waitForWindowCount(ctx context.Context, target int) error {
	deadline := time.Now().Add(warpWaitTimeout)
	delay := warpInitialBackoff

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		count, err := b.currentWindowCount(ctx)
		if err != nil {
			return err
		}
		if count >= target {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for warp windows: got %d, want %d", count, target)
		}

		b.sleepFn(delay)
		delay *= 2
		if delay > warpMaxBackoff {
			delay = warpMaxBackoff
		}
	}
}

func (b *WarpBackend) currentWindowCount(ctx context.Context) (int, error) {
	output, err := b.executor.RunAppleScript(ctx, `tell application "System Events" to tell process "Warp" to count windows`)
	if err != nil {
		return 0, wrapAccessibilityError(fmt.Errorf("count warp windows: %w", err))
	}

	count, convErr := strconv.Atoi(strings.TrimSpace(output))
	if convErr != nil {
		return 0, fmt.Errorf("parse warp window count %q: %w", output, convErr)
	}
	return count, nil
}

func (b *WarpBackend) sendCommandToWindows(ctx context.Context, count int, command string) error {
	sanitizedCmd := script.SanitizeForAppleScript(command)
	lines := []string{
		"tell application \"System Events\"",
		"  tell process \"Warp\"",
	}
	for i := 1; i <= count; i++ {
		lines = append(lines,
			fmt.Sprintf("    set frontmost to true"),
			fmt.Sprintf("    perform action \"AXRaise\" of window %d", i),
			fmt.Sprintf("    delay 0.3"),
			fmt.Sprintf("    keystroke \"%s\"", sanitizedCmd),
			fmt.Sprintf("    keystroke return"),
			fmt.Sprintf("    delay 0.2"),
		)
	}
	lines = append(lines, "  end tell", "end tell")

	_, err := b.executor.RunAppleScript(ctx, strings.Join(lines, "\n"))
	if err != nil {
		return wrapAccessibilityError(err)
	}
	return nil
}

func (b *WarpBackend) tileWindows(ctx context.Context, bounds []grid.WindowBounds) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return TileWarpWindows(b.executor, bounds)
}
