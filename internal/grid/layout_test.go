package grid

import (
	"testing"
)

func TestCalculateGrid(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected GridLayout
	}{
		{
			name:     "count=1",
			count:    1,
			expected: GridLayout{Rows: 1, Cols: 1},
		},
		{
			name:     "count=2",
			count:    2,
			expected: GridLayout{Rows: 1, Cols: 2},
		},
		{
			name:     "count=3",
			count:    3,
			expected: GridLayout{Rows: 1, Cols: 3},
		},
		{
			name:     "count=4",
			count:    4,
			expected: GridLayout{Rows: 2, Cols: 2},
		},
		{
			name:     "count=5 (uniform grid with 1 empty cell)",
			count:    5,
			expected: GridLayout{Rows: 2, Cols: 3},
		},
		{
			name:     "count=6",
			count:    6,
			expected: GridLayout{Rows: 2, Cols: 3},
		},
		{
			name:     "count=7 (uniform grid with 2 empty cells)",
			count:    7,
			expected: GridLayout{Rows: 3, Cols: 3},
		},
		{
			name:     "count=8",
			count:    8,
			expected: GridLayout{Rows: 2, Cols: 4},
		},
		{
			name:     "count=9",
			count:    9,
			expected: GridLayout{Rows: 3, Cols: 3},
		},
		{
			name:     "count=12",
			count:    12,
			expected: GridLayout{Rows: 3, Cols: 4},
		},
		{
			name:     "count=16",
			count:    16,
			expected: GridLayout{Rows: 4, Cols: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateGrid(tt.count)
			if got.Rows != tt.expected.Rows || got.Cols != tt.expected.Cols {
				t.Errorf("CalculateGrid(%d) = {Rows: %d, Cols: %d}, want {Rows: %d, Cols: %d}",
					tt.count, got.Rows, got.Cols, tt.expected.Rows, tt.expected.Cols)
			}
		})
	}
}

func TestCalculateWindowBounds(t *testing.T) {
	// Test 2x2 grid on 2560x1575 screen
	grid := GridLayout{Rows: 2, Cols: 2}
	screen := ScreenInfo{X: 0, Y: 0, Width: 2560, Height: 1575}

	bounds := CalculateWindowBounds(grid, screen, 4)

	if len(bounds) != 4 {
		t.Errorf("CalculateWindowBounds returned %d bounds, want 4", len(bounds))
	}

	tests := []struct {
		index    int
		expected WindowBounds
	}{
		{
			index:    0,
			expected: WindowBounds{X: 0, Y: 0, Width: 1280, Height: 787},
		},
		{
			index:    1,
			expected: WindowBounds{X: 1280, Y: 0, Width: 1280, Height: 787},
		},
		{
			index:    2,
			expected: WindowBounds{X: 0, Y: 787, Width: 1280, Height: 788},
		},
		{
			index:    3,
			expected: WindowBounds{X: 1280, Y: 787, Width: 1280, Height: 788},
		},
	}

	for _, tt := range tests {
		if bounds[tt.index].X != tt.expected.X ||
			bounds[tt.index].Y != tt.expected.Y ||
			bounds[tt.index].Width != tt.expected.Width ||
			bounds[tt.index].Height != tt.expected.Height {
			t.Errorf("Window %d: got {X: %d, Y: %d, Width: %d, Height: %d}, want {X: %d, Y: %d, Width: %d, Height: %d}",
				tt.index,
				bounds[tt.index].X, bounds[tt.index].Y, bounds[tt.index].Width, bounds[tt.index].Height,
				tt.expected.X, tt.expected.Y, tt.expected.Width, tt.expected.Height)
		}
	}
}

func TestCalculateWindowBoundsWithEmptyCell(t *testing.T) {
	// Test count=5 in 2x3 grid produces 6 bounds (last one unused)
	grid := GridLayout{Rows: 2, Cols: 3}
	screen := ScreenInfo{X: 0, Y: 0, Width: 2560, Height: 1575}

	bounds := CalculateWindowBounds(grid, screen, 5)

	if len(bounds) != 6 {
		t.Errorf("CalculateWindowBounds for count=5 returned %d bounds, want 6", len(bounds))
	}

	// Verify first 5 bounds are valid
	for i := 0; i < 5; i++ {
		if bounds[i].Width <= 0 || bounds[i].Height <= 0 {
			t.Errorf("Window %d has invalid dimensions: {Width: %d, Height: %d}", i, bounds[i].Width, bounds[i].Height)
		}
	}
}

func TestParseLayout(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  GridLayout
		wantError bool
	}{
		{
			name:      "valid lowercase 2x3",
			input:     "2x3",
			expected:  GridLayout{Rows: 2, Cols: 3},
			wantError: false,
		},
		{
			name:      "valid uppercase 3X2",
			input:     "3X2",
			expected:  GridLayout{Rows: 3, Cols: 2},
			wantError: false,
		},
		{
			name:      "invalid format abc",
			input:     "abc",
			expected:  GridLayout{},
			wantError: true,
		},
		{
			name:      "invalid zero rows",
			input:     "0x1",
			expected:  GridLayout{},
			wantError: true,
		},
		{
			name:      "invalid zero cols",
			input:     "1x0",
			expected:  GridLayout{},
			wantError: true,
		},
		{
			name:      "invalid negative",
			input:     "-1x2",
			expected:  GridLayout{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLayout(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseLayout(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
			if !tt.wantError && (got.Rows != tt.expected.Rows || got.Cols != tt.expected.Cols) {
				t.Errorf("ParseLayout(%q) = {Rows: %d, Cols: %d}, want {Rows: %d, Cols: %d}",
					tt.input, got.Rows, got.Cols, tt.expected.Rows, tt.expected.Cols)
			}
		})
	}
}
