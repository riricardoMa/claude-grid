//go:build darwin

package screen

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/riricardoMa/claude-grid/internal/script"
)

type ScreenInfo struct {
	X      int
	Y      int
	Width  int
	Height int
}

func DetectScreen(executor script.ScriptExecutor) (ScreenInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*1000*1000*1000) // 10 seconds
	defer cancel()

	applescript := `tell application "Finder" to get bounds of window of desktop`
	output, err := executor.RunAppleScript(ctx, applescript)
	if err != nil {
		return ScreenInfo{}, fmt.Errorf("failed to get screen bounds: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return ScreenInfo{}, fmt.Errorf("empty response from AppleScript")
	}

	parts := strings.Split(output, ",")
	if len(parts) != 4 {
		return ScreenInfo{}, fmt.Errorf("expected 4 bounds values, got %d", len(parts))
	}

	var bounds [4]int
	for i, part := range parts {
		val, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return ScreenInfo{}, fmt.Errorf("invalid bounds value at position %d: %q", i, part)
		}
		bounds[i] = val
	}

	left, top, right, bottom := bounds[0], bounds[1], bounds[2], bounds[3]

	return ScreenInfo{
		X:      left,
		Y:      top,
		Width:  right - left,
		Height: bottom - top,
	}, nil
}
