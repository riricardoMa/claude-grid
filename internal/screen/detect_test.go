package screen

import (
	"context"
	"errors"
	"testing"
)

type MockScriptExecutor struct {
	output  string
	err     error
	callIdx int
	calls   []mockCall
}

type mockCall struct {
	output string
	err    error
}

func (m *MockScriptExecutor) RunAppleScript(ctx context.Context, script string) (string, error) {
	if len(m.calls) > 0 {
		if m.callIdx < len(m.calls) {
			c := m.calls[m.callIdx]
			m.callIdx++
			return c.output, c.err
		}
		return "", errors.New("no more mock calls configured")
	}
	return m.output, m.err
}

func noDockMock(boundsOutput string) *MockScriptExecutor {
	return &MockScriptExecutor{
		calls: []mockCall{
			{output: boundsOutput},
			{err: errors.New("dock auto-hidden")},
		},
	}
}

func withDockMock(boundsOutput, dockOutput string) *MockScriptExecutor {
	return &MockScriptExecutor{
		calls: []mockCall{
			{output: boundsOutput},
			{output: "false"},
			{output: dockOutput},
		},
	}
}

func noDockMockErr(err error) *MockScriptExecutor {
	return &MockScriptExecutor{
		calls: []mockCall{
			{err: err},
		},
	}
}

func noDockMockOutput(boundsOutput string) *MockScriptExecutor {
	return noDockMock(boundsOutput)
}

func TestDetectScreen(t *testing.T) {
	tests := []struct {
		name      string
		executor  *MockScriptExecutor
		want      ScreenInfo
		wantError bool
	}{
		{
			name:     "standard bounds no dock",
			executor: noDockMock("0, 25, 2560, 1575"),
			want:     ScreenInfo{X: 0, Y: 25, Width: 2560, Height: 1550},
		},
		{
			name:     "bounds with different origin",
			executor: noDockMock("100, 50, 1920, 1080"),
			want:     ScreenInfo{X: 100, Y: 50, Width: 1820, Height: 1030},
		},
		{
			name:     "bounds with no spaces",
			executor: noDockMock("0,25,2560,1575"),
			want:     ScreenInfo{X: 0, Y: 25, Width: 2560, Height: 1550},
		},
		{
			name:     "bounds with extra spaces",
			executor: noDockMock("0 , 25 , 2560 , 1575"),
			want:     ScreenInfo{X: 0, Y: 25, Width: 2560, Height: 1550},
		},
		{
			name:      "invalid format - not enough values",
			executor:  noDockMock("0, 25, 2560"),
			wantError: true,
		},
		{
			name:      "invalid format - non-numeric",
			executor:  noDockMock("0, abc, 2560, 1575"),
			wantError: true,
		},
		{
			name:      "executor error",
			executor:  noDockMockErr(errors.New("AppleScript failed")),
			wantError: true,
		},
		{
			name:      "empty output",
			executor:  noDockMock(""),
			wantError: true,
		},
		{
			name:     "bottom dock subtracted",
			executor: withDockMock("0, 0, 1728, 1117", "146, 1029, 1436, 78"),
			want:     ScreenInfo{X: 0, Y: 0, Width: 1728, Height: 1029},
		},
		{
			name:     "left dock subtracted",
			executor: withDockMock("0, 25, 1920, 1080", "0, 25, 80, 1055"),
			want:     ScreenInfo{X: 80, Y: 25, Width: 1840, Height: 1055},
		},
		{
			name:     "right dock subtracted",
			executor: withDockMock("0, 25, 1920, 1080", "1840, 25, 80, 1055"),
			want:     ScreenInfo{X: 0, Y: 25, Width: 1840, Height: 1055},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectScreen(tt.executor)

			if (err != nil) != tt.wantError {
				t.Errorf("DetectScreen() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && got != tt.want {
				t.Errorf("DetectScreen() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDetectScreenBoundsCalculation(t *testing.T) {
	executor := noDockMock("10, 20, 110, 120")

	got, err := DetectScreen(executor)
	if err != nil {
		t.Fatalf("DetectScreen() error = %v", err)
	}

	if got.X != 10 {
		t.Errorf("X = %d, want 10", got.X)
	}
	if got.Y != 20 {
		t.Errorf("Y = %d, want 20", got.Y)
	}
	if got.Width != 100 {
		t.Errorf("Width = %d, want 100", got.Width)
	}
	if got.Height != 100 {
		t.Errorf("Height = %d, want 100", got.Height)
	}
}

func TestParseBounds(t *testing.T) {
	got, err := parseBounds("0, 0, 1728, 1117")
	if err != nil {
		t.Fatalf("parseBounds() error = %v", err)
	}
	if got.Width != 1728 || got.Height != 1117 {
		t.Errorf("parseBounds() = %+v, want 1728x1117", got)
	}
}

func TestSubtractDock(t *testing.T) {
	tests := []struct {
		name   string
		screen ScreenInfo
		dock   dockInfo
		want   ScreenInfo
	}{
		{
			name:   "bottom dock",
			screen: ScreenInfo{X: 0, Y: 0, Width: 1728, Height: 1117},
			dock:   dockInfo{x: 146, y: 1029, width: 1436, height: 78, orientation: "bottom"},
			want:   ScreenInfo{X: 0, Y: 0, Width: 1728, Height: 1029},
		},
		{
			name:   "left dock",
			screen: ScreenInfo{X: 0, Y: 25, Width: 1920, Height: 1055},
			dock:   dockInfo{x: 0, y: 25, width: 80, height: 1055, orientation: "left"},
			want:   ScreenInfo{X: 80, Y: 25, Width: 1840, Height: 1055},
		},
		{
			name:   "right dock",
			screen: ScreenInfo{X: 0, Y: 25, Width: 1920, Height: 1055},
			dock:   dockInfo{x: 1840, y: 25, width: 80, height: 1055, orientation: "right"},
			want:   ScreenInfo{X: 0, Y: 25, Width: 1840, Height: 1055},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := subtractDock(tt.screen, tt.dock)
			if got != tt.want {
				t.Errorf("subtractDock() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
