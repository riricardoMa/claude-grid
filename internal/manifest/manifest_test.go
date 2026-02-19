package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name        string
		yaml        string
		wantErr     bool
		errContains string
		check       func(t *testing.T, m Manifest)
	}{
		{
			name: "valid full manifest",
			yaml: `name: sprint-42
instances:
  - dir: /tmp/frontend
    prompt: "fix the login page"
    branch: fix/login
  - dir: /tmp/backend
    prompt: "add rate limiting"
`,
			check: func(t *testing.T, m Manifest) {
				if m.Name != "sprint-42" {
					t.Errorf("Name = %q, want %q", m.Name, "sprint-42")
				}
				if len(m.Instances) != 2 {
					t.Fatalf("len(Instances) = %d, want 2", len(m.Instances))
				}
				if m.Instances[0].Dir != "/tmp/frontend" {
					t.Errorf("Instances[0].Dir = %q, want /tmp/frontend", m.Instances[0].Dir)
				}
				if m.Instances[0].Prompt != "fix the login page" {
					t.Errorf("Instances[0].Prompt = %q, want fix the login page", m.Instances[0].Prompt)
				}
				if m.Instances[0].Branch != "fix/login" {
					t.Errorf("Instances[0].Branch = %q, want fix/login", m.Instances[0].Branch)
				}
				if m.Instances[1].Branch != "" {
					t.Errorf("Instances[1].Branch should be empty, got %q", m.Instances[1].Branch)
				}
			},
		},
		{
			name: "valid minimal — dir only",
			yaml: `instances:
  - dir: /tmp/alpha
  - dir: /tmp/beta
`,
			check: func(t *testing.T, m Manifest) {
				if len(m.Instances) != 2 {
					t.Fatalf("len(Instances) = %d, want 2", len(m.Instances))
				}
				if m.Instances[0].Prompt != "" {
					t.Errorf("Prompt should be empty")
				}
			},
		},
		{
			name:        "invalid YAML syntax",
			yaml:        "instances: [\nbad yaml: {{",
			wantErr:     true,
			errContains: "",
		},
		{
			name:        "missing instances field",
			yaml:        "name: test\n",
			wantErr:     true,
			errContains: "instances",
		},
		{
			name:        "empty instances list",
			yaml:        "instances: []\n",
			wantErr:     true,
			errContains: "instances",
		},
		{
			name: "instance missing dir",
			yaml: `instances:
  - prompt: "do something"
`,
			wantErr:     true,
			errContains: "dir",
		},
		{
			name: "too many instances (17)",
			yaml: func() string {
				s := "instances:\n"
				for i := 0; i < 17; i++ {
					s += "  - dir: /tmp/x\n"
				}
				return s
			}(),
			wantErr:     true,
			errContains: "16",
		},
		{
			name: "tilde expansion in dir",
			yaml: `instances:
  - dir: ~/projects/foo
`,
			check: func(t *testing.T, m Manifest) {
				want := filepath.Join(homeDir, "projects/foo")
				if m.Instances[0].Dir != want {
					t.Errorf("Dir = %q, want %q (tilde expanded)", m.Instances[0].Dir, want)
				}
			},
		},
		{
			name: "relative dir resolved to manifest location",
			yaml: `instances:
  - dir: subdir/project
`,
			check: func(t *testing.T, m Manifest) {
				// The manifest is in a temp dir, relative dir should be resolved relative to it
				if !filepath.IsAbs(m.Instances[0].Dir) {
					t.Errorf("Dir %q should be absolute after resolution", m.Instances[0].Dir)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			manifestPath := filepath.Join(tmpDir, "sprint.yaml")
			if err := os.WriteFile(manifestPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			m, err := Parse(manifestPath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse() expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, m)
			}
		})
	}
}
