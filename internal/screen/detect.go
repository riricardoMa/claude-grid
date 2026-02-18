//go:build darwin

package screen

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/riricardoMa/claude-grid/internal/script"
)

type ScreenInfo struct {
	X      int
	Y      int
	Width  int
	Height int
}

func DetectScreen(executor script.ScriptExecutor) (ScreenInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	applescript := `tell application "Finder" to get bounds of window of desktop`
	output, err := executor.RunAppleScript(ctx, applescript)
	if err != nil {
		return ScreenInfo{}, fmt.Errorf("failed to get screen bounds: %w", err)
	}

	info, err := parseBounds(output)
	if err != nil {
		return ScreenInfo{}, err
	}

	dock, dockErr := detectDock(ctx, executor)
	if dockErr == nil {
		info = subtractDock(info, dock)
	}

	return info, nil
}

func parseBounds(output string) (ScreenInfo, error) {
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

type dockInfo struct {
	x, y          int
	width, height int
	orientation   string
}

func detectDock(ctx context.Context, executor script.ScriptExecutor) (dockInfo, error) {
	autoHideScript := `tell application "System Events" to get autohide of dock preferences`
	ahOutput, ahErr := executor.RunAppleScript(ctx, autoHideScript)
	if ahErr == nil && strings.TrimSpace(strings.ToLower(ahOutput)) == "true" {
		return dockInfo{}, fmt.Errorf("dock is auto-hidden")
	}

	posScript := `tell application "System Events" to tell process "Dock"
	set dockPos to position of list 1
	set dockSize to size of list 1
	return {item 1 of dockPos, item 2 of dockPos, item 1 of dockSize, item 2 of dockSize}
end tell`

	output, err := executor.RunAppleScript(ctx, posScript)
	if err != nil {
		return dockInfo{}, fmt.Errorf("detect dock: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(output), ",")
	if len(parts) != 4 {
		return dockInfo{}, fmt.Errorf("unexpected dock output: %q", output)
	}

	vals := make([]int, 4)
	for i, p := range parts {
		v, convErr := strconv.Atoi(strings.TrimSpace(p))
		if convErr != nil {
			return dockInfo{}, fmt.Errorf("parse dock value %q: %w", p, convErr)
		}
		vals[i] = v
	}

	d := dockInfo{x: vals[0], y: vals[1], width: vals[2], height: vals[3]}

	if d.width > d.height {
		if d.y > d.height {
			d.orientation = "bottom"
		} else {
			d.orientation = "top"
		}
	} else {
		if d.x < d.width {
			d.orientation = "left"
		} else {
			d.orientation = "right"
		}
	}

	return d, nil
}

func subtractDock(s ScreenInfo, d dockInfo) ScreenInfo {
	switch d.orientation {
	case "bottom":
		dockTop := d.y
		screenBottom := s.Y + s.Height
		if dockTop < screenBottom {
			s.Height = dockTop - s.Y
		}
	case "left":
		dockRight := d.x + d.width
		if dockRight > s.X {
			s.Width -= dockRight - s.X
			s.X = dockRight
		}
	case "right":
		dockLeft := d.x
		screenRight := s.X + s.Width
		if dockLeft < screenRight {
			s.Width = dockLeft - s.X
		}
	}
	return s
}
