//go:build darwin

package screen

import (
	"context"
	"fmt"
	"os/exec"
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

var jxaScreenScript = strings.TrimSpace(`
ObjC.import("AppKit");
var screens = $.NSScreen.screens;
var primary = screens.objectAtIndex(0).frame;
var pH = primary.size.height;
var result = [];
for (var i = 0; i < screens.count; i++) {
    var v = screens.objectAtIndex(i).visibleFrame;
    var x = Math.round(v.origin.x);
    var y = Math.round(pH - v.origin.y - v.size.height);
    var w = Math.round(v.size.width);
    var h = Math.round(v.size.height);
    result.push(x + "," + y + "," + w + "," + h);
}
result.join(";");
`)

var execCommand = exec.CommandContext

func DetectScreen(executor script.ScriptExecutor) (ScreenInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	screens, err := detectAllScreens(ctx)
	if err != nil {
		return detectScreenFallback(ctx, executor)
	}

	if len(screens) == 1 {
		return screens[0], nil
	}

	windowPos, posErr := getFrontmostWindowPosition(ctx, executor)
	if posErr != nil {
		return screens[0], nil
	}

	return screenContaining(screens, windowPos[0], windowPos[1]), nil
}

func detectAllScreens(ctx context.Context) ([]ScreenInfo, error) {
	cmd := execCommand(ctx, "osascript", "-l", "JavaScript", "-e", jxaScreenScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("JXA screen detection failed: %w", err)
	}

	return parseScreenList(strings.TrimSpace(string(output)))
}

func parseScreenList(output string) ([]ScreenInfo, error) {
	if output == "" {
		return nil, fmt.Errorf("empty screen output")
	}

	parts := strings.Split(output, ";")
	screens := make([]ScreenInfo, 0, len(parts))
	for _, p := range parts {
		info, err := parseXYWH(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		screens = append(screens, info)
	}

	if len(screens) == 0 {
		return nil, fmt.Errorf("no screens detected")
	}

	return screens, nil
}

func parseXYWH(s string) (ScreenInfo, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return ScreenInfo{}, fmt.Errorf("expected 4 values, got %d in %q", len(parts), s)
	}

	vals := make([]int, 4)
	for i, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return ScreenInfo{}, fmt.Errorf("invalid value at position %d: %q", i, p)
		}
		vals[i] = v
	}

	return ScreenInfo{X: vals[0], Y: vals[1], Width: vals[2], Height: vals[3]}, nil
}

func getFrontmostWindowPosition(ctx context.Context, executor script.ScriptExecutor) ([2]int, error) {
	output, err := executor.RunAppleScript(ctx,
		`tell application "System Events" to get position of window 1 of (first process whose frontmost is true)`)
	if err != nil {
		return [2]int{}, fmt.Errorf("get frontmost window position: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(output), ",")
	if len(parts) != 2 {
		return [2]int{}, fmt.Errorf("unexpected position output: %q", output)
	}

	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return [2]int{}, fmt.Errorf("parse x: %w", err)
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return [2]int{}, fmt.Errorf("parse y: %w", err)
	}

	return [2]int{x, y}, nil
}

func screenContaining(screens []ScreenInfo, x, y int) ScreenInfo {
	for _, s := range screens {
		if x >= s.X && x < s.X+s.Width && y >= s.Y && y < s.Y+s.Height {
			return s
		}
	}
	return screens[0]
}

func detectScreenFallback(ctx context.Context, executor script.ScriptExecutor) (ScreenInfo, error) {
	output, err := executor.RunAppleScript(ctx,
		`tell application "Finder" to get bounds of window of desktop`)
	if err != nil {
		return ScreenInfo{}, fmt.Errorf("failed to get screen bounds: %w", err)
	}
	return parseBounds(output)
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
