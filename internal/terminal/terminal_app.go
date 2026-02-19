//go:build darwin

package terminal

import (
	"context"
	"fmt"
	"strings"

	"github.com/riricardoMa/claude-grid/internal/grid"
	"github.com/riricardoMa/claude-grid/internal/script"
	"github.com/riricardoMa/claude-grid/internal/session"
)

const (
	terminalTellStart   = "tell application \"Terminal\""
	terminalTellEnd     = "end tell"
	defaultSpawnCommand = "claude"
)

type TerminalAppBackend struct {
	executor script.ScriptExecutor
	store    *session.Store
}

func NewTerminalAppBackend(executor script.ScriptExecutor) *TerminalAppBackend {
	if executor == nil {
		executor = script.NewOSAExecutor()
	}

	return &TerminalAppBackend{
		executor: executor,
		store:    session.NewStore(""),
	}
}

func (b *TerminalAppBackend) Name() string {
	return "terminal"
}

func (b *TerminalAppBackend) Available() bool {
	return true
}

func (b *TerminalAppBackend) SpawnWindows(ctx context.Context, opts SpawnOptions) ([]WindowInfo, error) {
	if opts.Count <= 0 {
		return nil, fmt.Errorf("window count must be positive")
	}

	if len(opts.Bounds) < opts.Count {
		return nil, fmt.Errorf("insufficient bounds: got %d, need %d", len(opts.Bounds), opts.Count)
	}

	command := opts.Command
	if strings.TrimSpace(command) == "" {
		command = defaultSpawnCommand
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

	spawnScript := buildSpawnScript(opts.Count, dirs, opts.Prompts, command, opts.Bounds)
	output, err := b.executor.RunAppleScript(ctx, spawnScript)
	if err != nil {
		return nil, fmt.Errorf("failed to spawn terminal windows: %w", err)
	}

	ids, err := parseWindowIDs(output, opts.Count)
	if err != nil {
		return nil, err
	}

	windows := make([]WindowInfo, opts.Count)
	for i := 0; i < opts.Count; i++ {
		windows[i] = WindowInfo{
			ID:      ids[i],
			Index:   i,
			Backend: b.Name(),
		}
	}

	return windows, nil
}

func (b *TerminalAppBackend) CloseSession(sessionID string) error {
	sess, err := b.store.LoadSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session %q: %w", sessionID, err)
	}

	for _, window := range sess.Windows {
		closeScript := buildCloseWindowScript(window.ID)
		_, _ = b.executor.RunAppleScript(context.Background(), closeScript)
	}

	return nil
}

func buildSpawnScript(count int, dirs []string, prompts []string, command string, bounds []grid.WindowBounds) string {
	lines := []string{terminalTellStart}

	sanitizedCommand := script.SanitizeForAppleScript(command)

	for i := 0; i < count; i++ {
		sanitizedDir := script.SanitizeForAppleScript(dirs[i])
		windowCommand := sanitizedCommand
		if strings.TrimSpace(sanitizedDir) != "" {
			windowCommand = fmt.Sprintf("cd \\\"%s\\\" && %s", sanitizedDir, sanitizedCommand)
		}
		if i < len(prompts) && strings.TrimSpace(prompts[i]) != "" {
			sanitizedPrompt := script.SanitizeForAppleScript(prompts[i])
			windowCommand = fmt.Sprintf("%s \"%s\"", windowCommand, sanitizedPrompt)
		}

		bound := bounds[i]
		right := bound.X + bound.Width
		bottom := bound.Y + bound.Height

		lines = append(lines,
			fmt.Sprintf("do script \"%s\"", windowCommand),
			fmt.Sprintf("set windowID%d to id of front window", i),
			fmt.Sprintf("set bounds of window id windowID%d to {%d, %d, %d, %d}", i, bound.X, bound.Y, right, bottom),
		)
	}

	returnParts := make([]string, count)
	for i := 0; i < count; i++ {
		returnParts[i] = fmt.Sprintf("windowID%d as text", i)
	}

	lines = append(lines, fmt.Sprintf("return %s", strings.Join(returnParts, " & \",\" & ")))
	lines = append(lines, terminalTellEnd)

	return strings.Join(lines, "\n")
}

func parseWindowIDs(output string, expected int) ([]string, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil, fmt.Errorf("empty window id output")
	}

	parts := strings.Split(trimmed, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) < expected {
		return nil, fmt.Errorf("insufficient window ids: got %d, need %d", len(ids), expected)
	}

	return ids[:expected], nil
}

func buildCloseWindowScript(windowID string) string {
	sanitizedID := script.SanitizeForAppleScript(windowID)

	lines := []string{
		terminalTellStart,
		"try",
		fmt.Sprintf("close window id %s", sanitizedID),
		"on error",
		"end try",
		terminalTellEnd,
	}

	return strings.Join(lines, "\n")
}
