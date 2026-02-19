package script

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestSanitizeForAppleScript tests the sanitization function
func TestSanitizeForAppleScript(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "backslash only",
			input:    "\\",
			expected: "\\\\",
		},
		{
			name:     "double quote only",
			input:    "\"",
			expected: "\\\"",
		},
		{
			name:     "backslash then quote",
			input:    "\\\"",
			expected: "\\\\\\\"",
		},
		{
			name:     "path with spaces",
			input:    "/Users/John Doe/Documents",
			expected: "/Users/John Doe/Documents",
		},
		{
			name:     "path with backslash",
			input:    "C:\\Users\\John",
			expected: "C:\\\\Users\\\\John",
		},
		{
			name:     "quoted string",
			input:    "say \"hello\"",
			expected: "say \\\"hello\\\"",
		},
		{
			name:     "dollar sign alone",
			input:    "$HOME",
			expected: "$HOME",
		},
		{
			name:     "semicolon",
			input:    "tell app \"Finder\"; activate",
			expected: "tell app \\\"Finder\\\"; activate",
		},
		{
			name:     "complex path with quotes and backslashes",
			input:    "C:\\Program Files\\\"My App\"",
			expected: "C:\\\\Program Files\\\\\\\"My App\\\"",
		},
		{
			name:     "multiple backslashes",
			input:    "\\\\\\",
			expected: "\\\\\\\\\\\\",
		},
		{
			name:     "multiple quotes",
			input:    "\"\"\"",
			expected: "\\\"\\\"\\\"",
		},
		{
			name:     "mixed special chars",
			input:    "test\\path\"with\"quotes",
			expected: "test\\\\path\\\"with\\\"quotes",
		},
		// New prompt-specific escaping tests
		{
			name:     "newline character",
			input:    "line1\nline2",
			expected: "line1\\nline2",
		},
		{
			name:     "carriage return",
			input:    "line1\rline2",
			expected: "line1\\rline2",
		},
		{
			name:     "backtick command substitution",
			input:    "run `whoami`",
			expected: "run \\`whoami\\`",
		},
		{
			name:     "dollar-paren shell expansion",
			input:    "fix $(pwd) issues",
			expected: "fix \\$(pwd) issues",
		},
		{
			name:     "dollar-brace variable expansion",
			input:    "use ${HOME} path",
			expected: "use \\${HOME} path",
		},
		{
			name:     "combined prompt injection attempt",
			input:    "fix the \"login\" page\nand run $(test)",
			expected: "fix the \\\"login\\\" page\\nand run \\$(test)",
		},
		{
			name:     "prompt with single quotes",
			input:    "don't break this",
			expected: "don't break this",
		},
		{
			name:     "complex prompt with all special chars",
			input:    "check `status`\nrun ${CMD} and $(echo hi)\\done",
			expected: "check \\`status\\`\\nrun \\${CMD} and \\$(echo hi)\\\\done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForAppleScript(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeForAppleScript(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// MockExecutor is a mock implementation of ScriptExecutor for testing
type MockExecutor struct {
	RunAppleScriptFunc func(ctx context.Context, script string) (string, error)
}

func (m *MockExecutor) RunAppleScript(ctx context.Context, script string) (string, error) {
	if m.RunAppleScriptFunc != nil {
		return m.RunAppleScriptFunc(ctx, script)
	}
	return "", nil
}

// TestScriptExecutorInterface verifies the interface exists
func TestScriptExecutorInterface(t *testing.T) {
	var _ ScriptExecutor = (*OSAExecutor)(nil)
	var _ ScriptExecutor = (*MockExecutor)(nil)
}

// TestOSAExecutorRunAppleScript tests the OSAExecutor implementation
func TestOSAExecutorRunAppleScript(t *testing.T) {
	executor := NewOSAExecutor()

	tests := []struct {
		name      string
		script    string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "simple echo command",
			script:    "return \"hello\"",
			shouldErr: false,
		},
		{
			name:      "invalid AppleScript",
			script:    "invalid syntax here }{",
			shouldErr: true,
			errMsg:    "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			output, err := executor.RunAppleScript(ctx, tt.script)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("RunAppleScript(%q) expected error, got nil", tt.script)
				}
			} else {
				if err != nil {
					t.Errorf("RunAppleScript(%q) unexpected error: %v", tt.script, err)
				}
				if output == "" && tt.script != "" {
					t.Logf("RunAppleScript(%q) returned empty output (may be expected)", tt.script)
				}
			}
		})
	}
}

// TestOSAExecutorTimeout tests that context timeout is respected
func TestOSAExecutorTimeout(t *testing.T) {
	executor := NewOSAExecutor()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond)

	_, err := executor.RunAppleScript(ctx, "delay 10")

	if err == nil {
		t.Errorf("RunAppleScript with timeout expected error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("RunAppleScript timeout error: %v (type: %T)", err, err)
	}
}

// TestOSAExecutorContextCancellation tests that context cancellation is respected
func TestOSAExecutorContextCancellation(t *testing.T) {
	executor := NewOSAExecutor()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := executor.RunAppleScript(ctx, "return \"test\"")

	if err == nil {
		t.Errorf("RunAppleScript with cancelled context expected error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Logf("RunAppleScript cancellation error: %v (type: %T)", err, err)
	}
}

// TestSanitizeIntegration tests sanitization in context of AppleScript execution
func TestSanitizeIntegration(t *testing.T) {
	executor := NewOSAExecutor()

	testCases := []struct {
		name   string
		input  string
		script string
	}{
		{
			name:   "path with quotes",
			input:  "/Users/John \"Doc\" Doe",
			script: "return \"" + SanitizeForAppleScript("/Users/John \"Doc\" Doe") + "\"",
		},
		{
			name:   "path with backslashes",
			input:  "C:\\Users\\John",
			script: "return \"" + SanitizeForAppleScript("C:\\Users\\John") + "\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := executor.RunAppleScript(ctx, tc.script)
			if err != nil {
				t.Logf("RunAppleScript with sanitized input returned error: %v", err)
			}
		})
	}
}
