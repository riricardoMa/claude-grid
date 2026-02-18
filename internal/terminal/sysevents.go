//go:build darwin

package terminal

import (
	"context"
	"fmt"
	"strings"

	"github.com/riricardoMa/claude-grid/internal/grid"
	"github.com/riricardoMa/claude-grid/internal/script"
)

func TileWarpWindows(executor script.ScriptExecutor, bounds []grid.WindowBounds) error {
	lines := []string{
		"tell application \"System Events\"",
		"  tell process \"Warp\"",
	}

	for i, b := range bounds {
		index := i + 1
		lines = append(lines,
			fmt.Sprintf("    set position of window %d to {%d, %d}", index, b.X, b.Y),
			fmt.Sprintf("    set size of window %d to {%d, %d}", index, b.Width, b.Height),
		)
	}

	lines = append(lines,
		"  end tell",
		"end tell",
	)

	_, err := executor.RunAppleScript(context.Background(), strings.Join(lines, "\n"))
	if err != nil {
		return wrapAccessibilityError(fmt.Errorf("tile warp windows: %w", err))
	}

	return nil
}

func wrapAccessibilityError(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not allowed assistive access") {
		return fmt.Errorf("%w: %s", err, warpAccessibilityGuide)
	}

	return err
}
