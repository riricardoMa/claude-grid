//go:build darwin

package terminal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/riricardoMa/claude-grid/internal/grid"
	"github.com/riricardoMa/claude-grid/internal/script"
	"github.com/riricardoMa/claude-grid/internal/session"
)

type mockScriptExecutor struct {
	output string
	err    error
	runs   []string
	runFn  func(ctx context.Context, input string) (string, error)
}

func (m *mockScriptExecutor) RunAppleScript(ctx context.Context, input string) (string, error) {
	m.runs = append(m.runs, input)
	if m.runFn != nil {
		return m.runFn(ctx, input)
	}
	return m.output, m.err
}

func TestTerminalAppName(t *testing.T) {
	backend := NewTerminalAppBackend(&mockScriptExecutor{})
	if got := backend.Name(); got != "terminal" {
		t.Errorf("Name() = %q, want %q", got, "terminal")
	}
}

func TestTerminalAppAvailable(t *testing.T) {
	backend := NewTerminalAppBackend(&mockScriptExecutor{})
	if !backend.Available() {
		t.Error("Available() = false, want true")
	}
}

func TestTerminalAppSpawnScript(t *testing.T) {
	tests := []struct {
		name           string
		count          int
		command        string
		dir            string
		bounds         []grid.WindowBounds
		executorOutput string
		wantBounds     []string
		wantErr        bool
	}{
		{
			name:           "two windows with expected bounds format",
			count:          2,
			command:        "claude",
			dir:            "/tmp/workspace",
			executorOutput: "101,102",
			bounds: []grid.WindowBounds{
				{X: 0, Y: 0, Width: 800, Height: 600},
				{X: 800, Y: 0, Width: 800, Height: 600},
			},
			wantBounds: []string{
				"set bounds of window id windowID0 to {0, 0, 800, 600}",
				"set bounds of window id windowID1 to {800, 0, 1600, 600}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mockScriptExecutor{output: tt.executorOutput}
			backend := NewTerminalAppBackend(executor)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			windows, err := backend.SpawnWindows(ctx, SpawnOptions{
				Count:   tt.count,
				Command: tt.command,
				Dir:     tt.dir,
				Bounds:  tt.bounds,
			})

			if (err != nil) != tt.wantErr {
				t.Fatalf("SpawnWindows() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if len(executor.runs) != 1 {
				t.Fatalf("RunAppleScript call count = %d, want 1", len(executor.runs))
			}

			gotScript := executor.runs[0]
			if !strings.Contains(gotScript, "tell application \"Terminal\"") {
				t.Fatalf("script missing Terminal tell block: %s", gotScript)
			}

			if strings.Count(gotScript, "do script") != 2 {
				t.Errorf("do script count = %d, want 2", strings.Count(gotScript, "do script"))
			}

			if strings.Count(gotScript, "id of front window") != 2 {
				t.Errorf("id capture count = %d, want 2", strings.Count(gotScript, "id of front window"))
			}

			for _, wantLine := range tt.wantBounds {
				if !strings.Contains(gotScript, wantLine) {
					t.Errorf("script missing expected bounds line: %q", wantLine)
				}
			}

			if len(windows) != 2 {
				t.Fatalf("len(windows) = %d, want 2", len(windows))
			}

			if windows[0].ID != "101" || windows[1].ID != "102" {
				t.Errorf("window IDs = [%s, %s], want [101, 102]", windows[0].ID, windows[1].ID)
			}

			if windows[0].Index != 0 || windows[1].Index != 1 {
				t.Errorf("window indexes = [%d, %d], want [0, 1]", windows[0].Index, windows[1].Index)
			}
		})
	}
}

func TestTerminalAppEscaping(t *testing.T) {
	executor := &mockScriptExecutor{output: "201"}
	backend := NewTerminalAppBackend(executor)

	dir := `/tmp/My "Special" Dir`
	command := `printf "path\\value"`

	_, err := backend.SpawnWindows(context.Background(), SpawnOptions{
		Count:   1,
		Dir:     dir,
		Command: command,
		Bounds: []grid.WindowBounds{
			{X: 1, Y: 2, Width: 300, Height: 400},
		},
	})
	if err != nil {
		t.Fatalf("SpawnWindows() error = %v", err)
	}

	if len(executor.runs) != 1 {
		t.Fatalf("RunAppleScript call count = %d, want 1", len(executor.runs))
	}

	gotScript := executor.runs[0]
	wantDir := script.SanitizeForAppleScript(dir)
	wantCommand := script.SanitizeForAppleScript(command)

	if !strings.Contains(gotScript, wantDir) {
		t.Errorf("script does not contain sanitized dir: %q", wantDir)
	}

	if !strings.Contains(gotScript, wantCommand) {
		t.Errorf("script does not contain sanitized command: %q", wantCommand)
	}
}

func TestTerminalAppCloseGraceful(t *testing.T) {
	tempDir := t.TempDir()
	store := session.NewStore(tempDir)

	if err := store.SaveSession(session.Session{
		Name:    "grid-close",
		Backend: "terminal",
		Count:   2,
		Dir:     "/tmp",
		Windows: []session.WindowRef{{ID: "10", Index: 0}, {ID: "11", Index: 1}},
	}); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	executor := &mockScriptExecutor{
		runFn: func(ctx context.Context, input string) (string, error) {
			if strings.Contains(input, "close window id 11") {
				return "", errors.New("window not found")
			}
			return "", nil
		},
	}

	backend := NewTerminalAppBackend(executor)
	backend.store = store

	if err := backend.CloseSession("grid-close"); err != nil {
		t.Fatalf("CloseSession() error = %v, want nil", err)
	}

	if len(executor.runs) != 2 {
		t.Errorf("RunAppleScript calls = %d, want 2", len(executor.runs))
	}
}

func TestTerminalAppImplementsInterface(t *testing.T) {
	var _ TerminalBackend = (*TerminalAppBackend)(nil)
}
