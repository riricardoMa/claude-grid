package screen

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

type MockScriptExecutor struct {
	output string
	err    error
}

func (m *MockScriptExecutor) RunAppleScript(ctx context.Context, script string) (string, error) {
	return m.output, m.err
}

func TestParseScreenList(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []ScreenInfo
		wantError bool
	}{
		{
			name:  "single screen",
			input: "0,33,1728,1000",
			want:  []ScreenInfo{{X: 0, Y: 33, Width: 1728, Height: 1000}},
		},
		{
			name:  "dual monitor",
			input: "0,33,1728,1000;1728,0,1920,1080",
			want: []ScreenInfo{
				{X: 0, Y: 33, Width: 1728, Height: 1000},
				{X: 1728, Y: 0, Width: 1920, Height: 1080},
			},
		},
		{
			name:  "triple monitor",
			input: "-1920,0,1920,1080;0,33,1728,1000;1728,0,2560,1440",
			want: []ScreenInfo{
				{X: -1920, Y: 0, Width: 1920, Height: 1080},
				{X: 0, Y: 33, Width: 1728, Height: 1000},
				{X: 1728, Y: 0, Width: 2560, Height: 1440},
			},
		},
		{
			name:      "empty output",
			input:     "",
			wantError: true,
		},
		{
			name:      "invalid values",
			input:     "abc,0,1920,1080",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScreenList(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("parseScreenList() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if len(got) != len(tt.want) {
					t.Fatalf("got %d screens, want %d", len(got), len(tt.want))
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("screen[%d] = %+v, want %+v", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestScreenContaining(t *testing.T) {
	screens := []ScreenInfo{
		{X: 0, Y: 33, Width: 1728, Height: 1000},
		{X: 1728, Y: 0, Width: 1920, Height: 1080},
	}

	tests := []struct {
		name string
		x, y int
		want ScreenInfo
	}{
		{name: "primary screen", x: 500, y: 400, want: screens[0]},
		{name: "second screen", x: 2000, y: 500, want: screens[1]},
		{name: "boundary of second", x: 1728, y: 0, want: screens[1]},
		{name: "outside all falls back to first", x: 5000, y: 5000, want: screens[0]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := screenContaining(screens, tt.x, tt.y)
			if got != tt.want {
				t.Errorf("screenContaining(%d,%d) = %+v, want %+v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestGetFrontmostWindowPosition(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		err       error
		wantX     int
		wantY     int
		wantError bool
	}{
		{name: "valid", output: "259, 181", wantX: 259, wantY: 181},
		{name: "second monitor", output: "2000, 100", wantX: 2000, wantY: 100},
		{name: "error", err: errors.New("fail"), wantError: true},
		{name: "invalid", output: "abc", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &MockScriptExecutor{output: tt.output, err: tt.err}
			pos, err := getFrontmostWindowPosition(context.Background(), executor)
			if (err != nil) != tt.wantError {
				t.Errorf("error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && (pos[0] != tt.wantX || pos[1] != tt.wantY) {
				t.Errorf("got {%d,%d}, want {%d,%d}", pos[0], pos[1], tt.wantX, tt.wantY)
			}
		})
	}
}

func TestParseBounds(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      ScreenInfo
		wantError bool
	}{
		{name: "standard", input: "0, 0, 1728, 1117", want: ScreenInfo{X: 0, Y: 0, Width: 1728, Height: 1117}},
		{name: "with offset", input: "10, 20, 110, 120", want: ScreenInfo{X: 10, Y: 20, Width: 100, Height: 100}},
		{name: "empty", input: "", wantError: true},
		{name: "wrong count", input: "0, 25, 2560", wantError: true},
		{name: "non-numeric", input: "0, abc, 2560, 1575", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBounds(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDetectScreenIntegration(t *testing.T) {
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	t.Run("single screen via JXA", func(t *testing.T) {
		execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("echo", "0,33,1728,1000")
		}
		got, err := DetectScreen(&MockScriptExecutor{})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		want := ScreenInfo{X: 0, Y: 33, Width: 1728, Height: 1000}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("multi screen picks current terminal", func(t *testing.T) {
		execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("echo", "0,33,1728,1000;1728,0,1920,1080")
		}
		got, err := DetectScreen(&MockScriptExecutor{output: "2000, 500"})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		want := ScreenInfo{X: 1728, Y: 0, Width: 1920, Height: 1080}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("JXA fails falls back to Finder", func(t *testing.T) {
		execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("false")
		}
		got, err := DetectScreen(&MockScriptExecutor{output: "0, 25, 2560, 1575"})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		want := ScreenInfo{X: 0, Y: 25, Width: 2560, Height: 1550}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}
