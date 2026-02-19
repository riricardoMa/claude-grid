//go:build darwin

package terminal

import (
	"context"
	"fmt"
	"strings"

	"github.com/riricardoMa/claude-grid/internal/grid"
	"github.com/riricardoMa/claude-grid/internal/script"
	"github.com/riricardoMa/claude-grid/internal/screen"
)

// TerminalBackend defines the interface for spawning and managing terminal windows.
type TerminalBackend interface {
	// Name returns the name of the backend (e.g., "terminal", "warp").
	Name() string

	// Available checks if the backend is available on the system.
	Available() bool

	// SpawnWindows spawns terminal windows according to the provided options.
	// Tiling is atomic with spawning (no separate Tile() method).
	SpawnWindows(ctx context.Context, opts SpawnOptions) ([]WindowInfo, error)

	// CloseSession closes all windows associated with a session.
	CloseSession(sessionID string) error
}

// SpawnOptions contains the configuration for spawning terminal windows.
type SpawnOptions struct {
	// Count is the number of windows to spawn.
	Count int

	// Command is the command to run in each window (e.g., "claude").
	// Defaults to "claude" if empty.
	Command string

	// Dir is the absolute working directory for the windows.
	Dir string

	// Dirs is an optional list of per-window directories. If set, Dirs[i] is used for window i. If empty, Dir is used for all windows.
	Dirs []string

	// Grid specifies the grid layout (rows and columns).
	Grid grid.GridLayout

	// Screen contains the screen dimensions and position.
	Screen screen.ScreenInfo

	// Bounds contains pre-calculated window bounds for each grid cell.
	// Length should equal Grid.Rows * Grid.Cols.
	Bounds []grid.WindowBounds

	// SessionID is a unique identifier for tracking this session.
	SessionID string
}

// WindowInfo contains information about a spawned terminal window.
type WindowInfo struct {
	// ID is the unique identifier for the window (Terminal.app window ID or Warp window index).
	ID string

	// Index is the 0-based position in the grid.
	Index int

	// Backend is the name of the backend that created this window ("terminal" or "warp").
	Backend string
}

// DetectBackend detects and returns the appropriate terminal backend.
// If preferred is non-empty, it attempts to use that backend first.
// Falls back to auto-detection (Warp > Terminal.app) if preferred is unavailable.
func DetectBackend(preferred string) (TerminalBackend, error) {
	executor := script.NewOSAExecutor()
	warpBackend := NewWarpBackend(executor)
	terminalBackend := NewTerminalAppBackend(executor)

	normalized := strings.ToLower(strings.TrimSpace(preferred))
	if normalized == "" || normalized == "auto" {
		if warpBackend.Available() {
			return warpBackend, nil
		}
		return terminalBackend, nil
	}

	switch normalized {
	case "warp":
		if warpBackend.Available() {
			return warpBackend, nil
		}
		return nil, fmt.Errorf("warp backend is not available. Install Warp or use --terminal terminal")
	case "terminal":
		if terminalBackend.Available() {
			return terminalBackend, nil
		}
		return nil, fmt.Errorf("terminal backend is not available")
	default:
		return nil, fmt.Errorf("unsupported terminal backend %q. Available options: warp, terminal", preferred)
	}
}
