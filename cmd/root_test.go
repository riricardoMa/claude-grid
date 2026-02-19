package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		outContains string
	}{
		{
			name:        "missing count arg",
			args:        []string{},
			wantErr:     true,
			errContains: "count argument is required",
		},
		{
			name:        "invalid count non-number",
			args:        []string{"abc"},
			wantErr:     true,
			errContains: "must be a number between 1 and 16",
		},
		{
			name:        "count too low zero",
			args:        []string{"0"},
			wantErr:     true,
			errContains: "must be between 1 and 16",
		},
		{
			name:        "count too high seventeen",
			args:        []string{"17"},
			wantErr:     true,
			errContains: "must be between 1 and 16",
		},
		{
			name:        "version flag",
			args:        []string{"--version"},
			wantErr:     false,
			outContains: "claude-grid version",
		},
		{
			name:    "valid count without claude in path",
			args:    []string{"4"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand("test", "abc123", "2026-01-01")
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.name == "valid count without claude in path" {
				if err != nil {
					stderrStr := stderr.String()
					if strings.Contains(stderrStr, "must be between 1 and 16") {
						t.Errorf("got count validation error for valid count 4: stderr=%q", stderrStr)
					}
				}
				return
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.errContains != "" {
				stderrStr := stderr.String()
				if !strings.Contains(stderrStr, tt.errContains) {
					t.Errorf("stderr = %q, want it to contain %q", stderrStr, tt.errContains)
				}
			}

			if tt.outContains != "" {
				stdoutStr := stdout.String()
				if !strings.Contains(stdoutStr, tt.outContains) {
					t.Errorf("stdout = %q, want it to contain %q", stdoutStr, tt.outContains)
				}
			}
		})
	}
}

func TestRootCommandDirFlags(t *testing.T) {
	cmd := NewRootCommand("test", "abc123", "2026-01-01")
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", "/tmp", "--dir", "/tmp"})
	err := cmd.Execute()
	stderrStr := stderr.String()
	if strings.Contains(stderrStr, "count argument is required") {
		t.Errorf("with --dir flags, should not get 'count argument is required', got: %q", stderrStr)
	}
	_ = err
}

func TestRootCommandManifestConflicts(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{
			name:    "manifest + dir",
			args:    []string{"--manifest", "/tmp/test.yaml", "--dir", "/tmp"},
			wantMsg: "--manifest cannot be combined",
		},
		{
			name:    "manifest + count",
			args:    []string{"--manifest", "/tmp/test.yaml", "3"},
			wantMsg: "--manifest cannot be combined",
		},
		{
			name:    "manifest + prompt",
			args:    []string{"--manifest", "/tmp/test.yaml", "--prompt", "do X"},
			wantMsg: "--manifest cannot be combined",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand("test", "abc123", "2026-01-01")
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			stderrStr := stderr.String()
			if !strings.Contains(stderrStr, tt.wantMsg) {
				t.Errorf("stderr = %q, want to contain %q", stderrStr, tt.wantMsg)
			}
		})
	}
}

func TestRootCommandDirValidation(t *testing.T) {
	cmd := NewRootCommand("test", "abc123", "2026-01-01")
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"2", "--dir", "/nonexistent/path/xyz123", "--dir", "/nonexistent/path/xyz456"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "directory does not exist") {
		t.Errorf("stderr = %q, want to contain 'directory does not exist'", stderrStr)
	}
}

func TestRootCommandTooManyPrompts(t *testing.T) {
	cmd := NewRootCommand("test", "abc123", "2026-01-01")
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"2", "--prompt", "a", "--prompt", "b", "--prompt", "c"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "more --prompt flags") {
		t.Errorf("stderr = %q, want to contain 'more --prompt flags'", stderrStr)
	}
}
