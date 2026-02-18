package grid

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// GridLayout represents the rows and columns of a grid
type GridLayout struct {
	Rows int
	Cols int
}

// ScreenInfo represents screen dimensions
type ScreenInfo struct {
	X      int
	Y      int
	Width  int
	Height int
}

// WindowBounds represents the position and size of a window
type WindowBounds struct {
	X      int
	Y      int
	Width  int
	Height int
}

// CalculateGrid calculates the optimal grid layout for a given count of windows.
// Special cases for 1-3 windows (horizontal layout), then sqrt-based for rest.
// Prefers wider windows (more cols than rows).
func CalculateGrid(count int) GridLayout {
	if count <= 0 {
		return GridLayout{Rows: 1, Cols: 1}
	}

	// Special cases for 1-3: horizontal layout
	if count <= 3 {
		return GridLayout{Rows: 1, Cols: count}
	}

	// For count >= 4: use sqrt-based approach
	// Find the ceiling of square root to get a starting point
	sqrtVal := int(math.Ceil(math.Sqrt(float64(count))))

	// Try layouts starting from sqrtVal rows, decreasing if needed
	// Find the layout closest to square that is wider than tall
	var bestLayout GridLayout
	minDiff := count

	for rows := sqrtVal; rows >= 1; rows-- {
		cols := int(math.Ceil(float64(count) / float64(rows)))
		// Only consider layouts where cols >= rows (wider than tall)
		if cols >= rows {
			// Calculate how close to square this layout is
			diff := cols - rows
			if diff < minDiff {
				minDiff = diff
				bestLayout = GridLayout{Rows: rows, Cols: cols}
			}
		}
	}

	if bestLayout.Rows > 0 {
		return bestLayout
	}

	// Fallback: square layout (shouldn't reach here)
	return GridLayout{Rows: sqrtVal, Cols: sqrtVal}
}

// CalculateWindowBounds calculates the pixel-accurate bounds for each window in the grid.
// Returns a slice of WindowBounds for all cells in the grid (including empty cells).
func CalculateWindowBounds(grid GridLayout, screen ScreenInfo, count int) []WindowBounds {
	totalCells := grid.Rows * grid.Cols
	bounds := make([]WindowBounds, totalCells)

	// Calculate cell dimensions
	cellWidth := screen.Width / grid.Cols
	cellHeight := screen.Height / grid.Rows

	// Distribute remainder pixels to last row/col to avoid gaps
	remainderWidth := screen.Width % grid.Cols
	remainderHeight := screen.Height % grid.Rows

	for i := 0; i < totalCells; i++ {
		row := i / grid.Cols
		col := i % grid.Cols

		x := screen.X + col*cellWidth
		y := screen.Y + row*cellHeight
		width := cellWidth
		height := cellHeight

		// Add remainder to last column
		if col == grid.Cols-1 {
			width += remainderWidth
		}

		// Add remainder to last row
		if row == grid.Rows-1 {
			height += remainderHeight
		}

		bounds[i] = WindowBounds{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
		}
	}

	return bounds
}

// ParseLayout parses a layout string in the format "RxC" or "RXC" where R is rows and C is cols.
// Returns an error if the format is invalid or values are <= 0.
func ParseLayout(s string) (GridLayout, error) {
	// Normalize to lowercase for case-insensitive parsing
	s = strings.ToLower(s)

	// Split by 'x'
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return GridLayout{}, fmt.Errorf("invalid layout format: %q (expected RxC)", s)
	}

	rows, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return GridLayout{}, fmt.Errorf("invalid rows value: %q", parts[0])
	}

	cols, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return GridLayout{}, fmt.Errorf("invalid cols value: %q", parts[1])
	}

	if rows <= 0 || cols <= 0 {
		return GridLayout{}, fmt.Errorf("rows and cols must be positive (got %dx%d)", rows, cols)
	}

	return GridLayout{Rows: rows, Cols: cols}, nil
}
