package pathutil

import (
	"os"
	"strings"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "tilde with slash",
			path: "~/foo",
			want: homeDir + "/foo",
		},
		{
			name: "tilde alone",
			path: "~",
			want: homeDir,
		},
		{
			name: "absolute path passthrough",
			path: "/absolute/path",
			want: "/absolute/path",
		},
		{
			name: "relative path passthrough",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "empty string passthrough",
			path: "",
			want: "",
		},
		{
			name:    "tilde user syntax error",
			path:    "~user/foo",
			wantErr: true,
			errMsg:  "unsupported ~user syntax",
		},
		{
			name:    "tilde user without slash",
			path:    "~otheruser",
			wantErr: true,
			errMsg:  "unsupported ~user syntax",
		},
		{
			name: "tilde with nested path",
			path: "~/projects/frontend/src",
			want: homeDir + "/projects/frontend/src",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandTilde(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandTilde(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ExpandTilde(%q) error = %q, want containing %q", tt.path, err.Error(), tt.errMsg)
				}
				return
			}
			if got != tt.want {
				t.Errorf("ExpandTilde(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestExpandTildeAll(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name    string
		paths   []string
		want    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:  "multiple valid paths",
			paths: []string{"~/a", "~/b"},
			want:  []string{homeDir + "/a", homeDir + "/b"},
		},
		{
			name:    "error on second element stops early",
			paths:   []string{"~/valid", "~otheruser/bad", "~/alsovalid"},
			wantErr: true,
			errMsg:  "unsupported ~user syntax",
		},
		{
			name:  "empty slice",
			paths: []string{},
			want:  []string{},
		},
		{
			name:  "mixed paths",
			paths: []string{"~/home", "/absolute", "relative"},
			want:  []string{homeDir + "/home", "/absolute", "relative"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandTildeAll(tt.paths)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandTildeAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ExpandTildeAll() error = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ExpandTildeAll() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ExpandTildeAll()[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}
