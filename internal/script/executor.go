package script

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 10 * time.Second

// ScriptExecutor defines the interface for executing AppleScript
type ScriptExecutor interface {
	RunAppleScript(ctx context.Context, script string) (string, error)
}

// OSAExecutor implements ScriptExecutor using osascript
type OSAExecutor struct {
	timeout time.Duration
}

// NewOSAExecutor creates a new OSAExecutor with default timeout
func NewOSAExecutor() *OSAExecutor {
	return &OSAExecutor{
		timeout: defaultTimeout,
	}
}

// NewOSAExecutorWithTimeout creates a new OSAExecutor with custom timeout
func NewOSAExecutorWithTimeout(timeout time.Duration) *OSAExecutor {
	return &OSAExecutor{
		timeout: timeout,
	}
}

// RunAppleScript executes an AppleScript and returns the output
func (e *OSAExecutor) RunAppleScript(ctx context.Context, script string) (string, error) {
	// Create a context with timeout if not already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	// Create the osascript command with context
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		// Include stderr content in error message
		return "", fmt.Errorf("osascript execution failed: %w (output: %s)", err, outputStr)
	}

	return outputStr, nil
}

// SanitizeForAppleScript escapes special characters to prevent injection
// Order is critical: escape backslashes FIRST, then double quotes
func SanitizeForAppleScript(s string) string {
	// Step 1: Escape backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")

	// Step 2: Escape double quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")

	return s
}
