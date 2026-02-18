package screen

import (
	"context"
	"errors"
	"testing"
)

type MockScriptExecutor struct {
	output string
	err    error
}

func (m *MockScriptExecutor) RunAppleScript(ctx context.Context, script string) (string, error) {
	return m.output, m.err
}

func TestDetectScreen(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		err       error
		want      ScreenInfo
		wantError bool
	}{
		{
			name:   "standard bounds",
			output: "0, 25, 2560, 1575",
			want:   ScreenInfo{X: 0, Y: 25, Width: 2560, Height: 1550},
		},
		{
			name:   "bounds with different origin",
			output: "100, 50, 1920, 1080",
			want:   ScreenInfo{X: 100, Y: 50, Width: 1820, Height: 1030},
		},
		{
			name:   "bounds with no spaces",
			output: "0,25,2560,1575",
			want:   ScreenInfo{X: 0, Y: 25, Width: 2560, Height: 1550},
		},
		{
			name:   "bounds with extra spaces",
			output: "0 , 25 , 2560 , 1575",
			want:   ScreenInfo{X: 0, Y: 25, Width: 2560, Height: 1550},
		},
		{
			name:      "invalid format - not enough values",
			output:    "0, 25, 2560",
			wantError: true,
		},
		{
			name:      "invalid format - non-numeric",
			output:    "0, abc, 2560, 1575",
			wantError: true,
		},
		{
			name:      "executor error",
			err:       errors.New("AppleScript failed"),
			wantError: true,
		},
		{
			name:      "empty output",
			output:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &MockScriptExecutor{
				output: tt.output,
				err:    tt.err,
			}

			got, err := DetectScreen(executor)

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
	// Verify that Width = right - left and Height = bottom - top
	executor := &MockScriptExecutor{
		output: "10, 20, 110, 120",
	}

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
	if got.Width != 100 { // 110 - 10
		t.Errorf("Width = %d, want 100", got.Width)
	}
	if got.Height != 100 { // 120 - 20
		t.Errorf("Height = %d, want 100", got.Height)
	}
}
