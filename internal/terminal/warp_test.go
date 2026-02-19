//go:build darwin

package terminal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/riricardoMa/claude-grid/internal/grid"
)

type warpMockExecutor struct {
	runFn   func(ctx context.Context, script string) (string, error)
	scripts []string
}

func (m *warpMockExecutor) RunAppleScript(ctx context.Context, script string) (string, error) {
	m.scripts = append(m.scripts, script)
	if m.runFn == nil {
		return "", nil
	}
	return m.runFn(ctx, script)
}

func TestWarpName(t *testing.T) {
	b := NewWarpBackend(&warpMockExecutor{})
	if got := b.Name(); got != "warp" {
		t.Fatalf("Name() = %q, want %q", got, "warp")
	}
}

func TestWarpAvailable(t *testing.T) {
	tests := []struct {
		name     string
		statErr  error
		expected bool
	}{
		{name: "app exists", expected: true},
		{name: "app missing", statErr: os.ErrNotExist, expected: false},
		{name: "permission error", statErr: errors.New("permission denied"), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewWarpBackend(&warpMockExecutor{})
			b.statFn = func(path string) (os.FileInfo, error) {
				if path != "/Applications/Warp.app" {
					t.Fatalf("stat path = %q, want %q", path, "/Applications/Warp.app")
				}
				return nil, tt.statErr
			}

			if got := b.Available(); got != tt.expected {
				t.Fatalf("Available() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWarpURIConstruction(t *testing.T) {
	b := NewWarpBackend(&warpMockExecutor{})
	var uris []string
	b.runOpen = func(ctx context.Context, uri string) error {
		uris = append(uris, uri)
		return nil
	}
	b.sleepFn = func(time.Duration) {}
	b.waitForWindowCountFn = func(context.Context, int) error { return nil }
	b.tileWindowsFn = func(context.Context, []grid.WindowBounds) error { return nil }

	_, err := b.SpawnWindows(context.Background(), SpawnOptions{
		Count: 1,
		Dir:   "/Users/bob/my project",
		Bounds: []grid.WindowBounds{
			{X: 0, Y: 0, Width: 100, Height: 100},
		},
	})
	if err != nil {
		t.Fatalf("SpawnWindows() error = %v", err)
	}

	if len(uris) != 1 {
		t.Fatalf("open call count = %d, want 1", len(uris))
	}

	want := "warp://action/new_window?path=/Users/bob/my%20project"
	if uris[0] != want {
		t.Fatalf("URI = %q, want %q", uris[0], want)
	}
}

func TestWarpTilingScript(t *testing.T) {
	executor := &warpMockExecutor{}
	bounds := []grid.WindowBounds{
		{X: 0, Y: 0, Width: 100, Height: 80},
		{X: 100, Y: 0, Width: 100, Height: 80},
		{X: 0, Y: 80, Width: 100, Height: 80},
		{X: 100, Y: 80, Width: 100, Height: 80},
	}

	if err := TileWarpWindows(executor, bounds); err != nil {
		t.Fatalf("TileWarpWindows() error = %v", err)
	}

	if len(executor.scripts) != 1 {
		t.Fatalf("RunAppleScript call count = %d, want 1", len(executor.scripts))
	}

	script := executor.scripts[0]
	for i, b := range bounds {
		idx := i + 1
		positionLine := fmt.Sprintf("set position of window %d to {%d, %d}", idx, b.X, b.Y)
		sizeLine := fmt.Sprintf("set size of window %d to {%d, %d}", idx, b.Width, b.Height)
		if !strings.Contains(script, positionLine) {
			t.Fatalf("missing line: %q\nscript:\n%s", positionLine, script)
		}
		if !strings.Contains(script, sizeLine) {
			t.Fatalf("missing line: %q\nscript:\n%s", sizeLine, script)
		}
	}
}

func TestWarpBackoff(t *testing.T) {
	executor := &warpMockExecutor{}
	b := NewWarpBackend(executor)

	b.runOpen = func(context.Context, string) error { return nil }
	b.sleepFn = func(time.Duration) {}
	b.tileWindowsFn = func(context.Context, []grid.WindowBounds) error { return nil }

	counts := []string{"0", "1", "2", "4"}
	pollCount := 0
	executor.runFn = func(ctx context.Context, script string) (string, error) {
		if strings.Contains(script, "count windows") {
			if pollCount >= len(counts) {
				return counts[len(counts)-1], nil
			}
			out := counts[pollCount]
			pollCount++
			return out, nil
		}
		return "", nil
	}

	_, err := b.SpawnWindows(context.Background(), SpawnOptions{
		Count: 4,
		Dir:   "/tmp",
		Bounds: []grid.WindowBounds{
			{X: 0, Y: 0, Width: 10, Height: 10},
			{X: 10, Y: 0, Width: 10, Height: 10},
			{X: 0, Y: 10, Width: 10, Height: 10},
			{X: 10, Y: 10, Width: 10, Height: 10},
		},
	})
	if err != nil {
		t.Fatalf("SpawnWindows() error = %v", err)
	}

	if pollCount != 4 {
		t.Fatalf("poll count = %d, want %d", pollCount, 4)
	}
}

func TestWarpAccessibilityError(t *testing.T) {
	executor := &warpMockExecutor{
		runFn: func(ctx context.Context, script string) (string, error) {
			return "", errors.New("execution error: not allowed assistive access")
		},
	}

	err := TileWarpWindows(executor, []grid.WindowBounds{{X: 0, Y: 0, Width: 100, Height: 100}})
	if err == nil {
		t.Fatalf("TileWarpWindows() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "Accessibility permission") {
		t.Fatalf("error = %q, want mention of Accessibility permission", err)
	}
}

func TestWarpSpawnWindowsScenarios(t *testing.T) {
	tests := []struct {
		name      string
		opts      SpawnOptions
		openErrAt int
		wantErr   bool
	}{
		{
			name: "single window success",
			opts: SpawnOptions{
				Count: 1,
				Dir:   "/tmp",
				Bounds: []grid.WindowBounds{
					{X: 0, Y: 0, Width: 100, Height: 100},
				},
			},
		},
		{
			name: "multiple windows success",
			opts: SpawnOptions{
				Count: 3,
				Dir:   "/tmp",
				Bounds: []grid.WindowBounds{
					{X: 0, Y: 0, Width: 100, Height: 100},
					{X: 100, Y: 0, Width: 100, Height: 100},
					{X: 0, Y: 100, Width: 100, Height: 100},
				},
			},
		},
		{
			name: "open error",
			opts: SpawnOptions{
				Count: 2,
				Dir:   "/tmp",
				Bounds: []grid.WindowBounds{
					{X: 0, Y: 0, Width: 100, Height: 100},
					{X: 100, Y: 0, Width: 100, Height: 100},
				},
			},
			openErrAt: 2,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewWarpBackend(&warpMockExecutor{})
			openCalls := 0
			b.runOpen = func(ctx context.Context, uri string) error {
				openCalls++
				if tt.openErrAt > 0 && openCalls == tt.openErrAt {
					return errors.New("open failed")
				}
				return nil
			}
			b.sleepFn = func(time.Duration) {}
			b.waitForWindowCountFn = func(context.Context, int) error { return nil }
			b.tileWindowsFn = func(context.Context, []grid.WindowBounds) error { return nil }

			got, err := b.SpawnWindows(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SpawnWindows() err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if len(got) != tt.opts.Count {
				t.Fatalf("window info count = %d, want %d", len(got), tt.opts.Count)
			}
			for i := range got {
				if got[i].Backend != "warp" {
					t.Fatalf("window %d backend = %q, want %q", i, got[i].Backend, "warp")
				}
				if got[i].Index != i {
					t.Fatalf("window %d index = %d, want %d", i, got[i].Index, i)
				}
			}
		})
	}
}

func TestWarpPerWindowDirs(t *testing.T) {
	tests := []struct {
		name     string
		opts     SpawnOptions
		wantURIs []string
	}{
		{
			name: "dirs override produces per-window URIs",
			opts: SpawnOptions{
				Count: 3,
				Dir:   "/fallback",
				Dirs:  []string{"/proj/alpha", "/proj/beta", "/proj/gamma"},
				Bounds: []grid.WindowBounds{
					{X: 0, Y: 0, Width: 100, Height: 100},
					{X: 100, Y: 0, Width: 100, Height: 100},
					{X: 0, Y: 100, Width: 100, Height: 100},
				},
			},
			wantURIs: []string{
				"warp://action/new_window?path=/proj/alpha",
				"warp://action/new_window?path=/proj/beta",
				"warp://action/new_window?path=/proj/gamma",
			},
		},
		{
			name: "empty dirs falls back to Dir for all windows",
			opts: SpawnOptions{
				Count: 2,
				Dir:   "/shared",
				Bounds: []grid.WindowBounds{
					{X: 0, Y: 0, Width: 100, Height: 100},
					{X: 100, Y: 0, Width: 100, Height: 100},
				},
			},
			wantURIs: []string{
				"warp://action/new_window?path=/shared",
				"warp://action/new_window?path=/shared",
			},
		},
		{
			name: "dirs with spaces are encoded correctly",
			opts: SpawnOptions{
				Count: 2,
				Dir:   "/tmp",
				Dirs:  []string{"/my project", "/other dir"},
				Bounds: []grid.WindowBounds{
					{X: 0, Y: 0, Width: 100, Height: 100},
					{X: 100, Y: 0, Width: 100, Height: 100},
				},
			},
			wantURIs: []string{
				"warp://action/new_window?path=/my%20project",
				"warp://action/new_window?path=/other%20dir",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewWarpBackend(&warpMockExecutor{})
			var gotURIs []string
			b.runOpen = func(ctx context.Context, uri string) error {
				gotURIs = append(gotURIs, uri)
				return nil
			}
			b.sleepFn = func(time.Duration) {}
			b.waitForWindowCountFn = func(context.Context, int) error { return nil }
			b.tileWindowsFn = func(context.Context, []grid.WindowBounds) error { return nil }

			_, err := b.SpawnWindows(context.Background(), tt.opts)
			if err != nil {
				t.Fatalf("SpawnWindows() error = %v", err)
			}

			if len(gotURIs) != len(tt.wantURIs) {
				t.Fatalf("URI count = %d, want %d", len(gotURIs), len(tt.wantURIs))
			}
			for i, want := range tt.wantURIs {
				if gotURIs[i] != want {
					t.Errorf("URI[%d] = %q, want %q", i, gotURIs[i], want)
				}
			}
		})
	}
}

func TestWarpCloseSession(t *testing.T) {
	executor := &warpMockExecutor{}
	b := NewWarpBackend(executor)

	err := b.CloseSession("session-123")
	if err != nil {
		t.Fatalf("CloseSession() error = %v", err)
	}

	if len(executor.scripts) != 1 {
		t.Fatalf("RunAppleScript call count = %d, want 1", len(executor.scripts))
	}

	if !strings.Contains(executor.scripts[0], "close window") {
		t.Fatalf("CloseSession script missing close window command:\n%s", executor.scripts[0])
	}
}

func TestWarpPerWindowPrompts(t *testing.T) {
	tests := []struct {
		name              string
		prompts           []string
		count             int
		wantKeystrokeStrs []string
	}{
		{
			name:              "per-window different prompts",
			prompts:           []string{"fix login", "add tests", "update docs"},
			count:             3,
			wantKeystrokeStrs: []string{"fix login", "add tests", "update docs"},
		},
		{
			name:              "no prompts — backward compat",
			prompts:           nil,
			count:             2,
			wantKeystrokeStrs: []string{"claude"},
		},
		{
			name:              "partial prompts — first window has prompt, rest bare",
			prompts:           []string{"fix login"},
			count:             3,
			wantKeystrokeStrs: []string{"fix login", "claude"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &warpMockExecutor{}
			b := NewWarpBackend(executor)
			b.runOpen = func(ctx context.Context, uri string) error { return nil }
			b.sleepFn = func(time.Duration) {}
			b.waitForWindowCountFn = func(context.Context, int) error { return nil }
			b.tileWindowsFn = func(context.Context, []grid.WindowBounds) error { return nil }

			bounds := make([]grid.WindowBounds, tt.count)
			for i := range bounds {
				bounds[i] = grid.WindowBounds{X: i * 100, Y: 0, Width: 100, Height: 100}
			}

			_, err := b.SpawnWindows(context.Background(), SpawnOptions{
				Count:   tt.count,
				Dir:     "/tmp",
				Bounds:  bounds,
				Prompts: tt.prompts,
			})
			if err != nil {
				t.Fatalf("SpawnWindows() error = %v", err)
			}

			// Find the keystroke script (last RunAppleScript call that contains "keystroke")
			var keystrokeScript string
			for _, s := range executor.scripts {
				if strings.Contains(s, "keystroke") {
					keystrokeScript = s
					break
				}
			}
			if keystrokeScript == "" {
				t.Fatalf("no keystroke script found in captured scripts: %v", executor.scripts)
			}

			for _, want := range tt.wantKeystrokeStrs {
				if !strings.Contains(keystrokeScript, want) {
					t.Errorf("keystroke script missing %q\nscript:\n%s", want, keystrokeScript)
				}
			}
		})
	}
}
