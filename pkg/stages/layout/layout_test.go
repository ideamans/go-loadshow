package layout

import (
	"context"
	"testing"

	"github.com/user/loadshow/pkg/pipeline"
)

// TestComputeLayout_MatchTypeScript tests that the Go implementation produces
// the exact same output as the TypeScript implementation.
// Expected values are from loadshow/src/layout.test.ts
func TestComputeLayout_MatchTypeScript(t *testing.T) {
	// TypeScript defaultLayoutSpec():
	// canvasWidth: 512, canvasHeight: 640, columns: 3, gap: 20,
	// padding: 20, borderWidth: 1, indent: 20, outdent: 20, progressHeight: 16
	// Note: TypeScript doesn't have bannerHeight in the base layout calculation
	input := pipeline.LayoutInput{
		CanvasWidth:    512,
		CanvasHeight:   640,
		Columns:        3,
		Gap:            20,
		Padding:        20,
		BorderWidth:    1,
		Indent:         20,
		Outdent:        20,
		BannerHeight:   0, // TypeScript layout doesn't include banner
		ProgressHeight: 0, // TypeScript layout doesn't include progress in main layout
	}

	result := ComputeLayout(input)

	// Expected from TypeScript test:
	// scroll: { width: 144, height: 1739 }
	if result.Scroll.Width != 144 {
		t.Errorf("scroll.width: expected 144, got %d", result.Scroll.Width)
	}
	if result.Scroll.Height != 1739 {
		t.Errorf("scroll.height: expected 1739, got %d", result.Scroll.Height)
	}

	// columns: [
	//   { x: 20, y: 20, width: 144, height: 580 },
	//   { x: 184, y: 40, width: 144, height: 580 },
	//   { x: 348, y: 40, width: 144, height: 580 },
	// ]
	expectedColumns := []pipeline.Rectangle{
		{X: 20, Y: 20, Width: 144, Height: 580},
		{X: 184, Y: 40, Width: 144, Height: 580},
		{X: 348, Y: 40, Width: 144, Height: 580},
	}
	if len(result.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(result.Columns))
	}
	for i, expected := range expectedColumns {
		got := result.Columns[i]
		if got != expected {
			t.Errorf("columns[%d]: expected %+v, got %+v", i, expected, got)
		}
	}

	// windows: [
	//   { x: 21, y: 21, width: 142, height: 580, scrollTop: 0 },
	//   { x: 185, y: 40, width: 142, height: 580, scrollTop: 580 },
	//   { x: 349, y: 40, width: 142, height: 579, scrollTop: 1160 },
	// ]
	expectedWindows := []pipeline.Window{
		{Rectangle: pipeline.Rectangle{X: 21, Y: 21, Width: 142, Height: 580}, ScrollTop: 0},
		{Rectangle: pipeline.Rectangle{X: 185, Y: 40, Width: 142, Height: 580}, ScrollTop: 580},
		{Rectangle: pipeline.Rectangle{X: 349, Y: 40, Width: 142, Height: 579}, ScrollTop: 1160},
	}
	if len(result.Windows) != 3 {
		t.Fatalf("expected 3 windows, got %d", len(result.Windows))
	}
	for i, expected := range expectedWindows {
		got := result.Windows[i]
		if got != expected {
			t.Errorf("windows[%d]: expected %+v, got %+v", i, expected, got)
		}
	}
}

// TestComputeLayout_ColumnCalculation tests the column width calculation.
func TestComputeLayout_ColumnCalculation(t *testing.T) {
	tests := []struct {
		name          string
		canvasWidth   int
		padding       int
		gap           int
		columns       int
		expectedWidth int
	}{
		{
			name:          "default 3 columns",
			canvasWidth:   512,
			padding:       20,
			gap:           20,
			columns:       3,
			expectedWidth: 144, // floor((512 - 40 - 40) / 3) = 144
		},
		{
			name:          "2 columns",
			canvasWidth:   512,
			padding:       20,
			gap:           20,
			columns:       2,
			expectedWidth: 226, // floor((512 - 40 - 20) / 2) = 226
		},
		{
			name:          "4 columns",
			canvasWidth:   512,
			padding:       20,
			gap:           20,
			columns:       4,
			expectedWidth: 103, // floor((512 - 40 - 60) / 4) = 103
		},
		{
			name:          "1 column",
			canvasWidth:   512,
			padding:       20,
			gap:           20,
			columns:       1,
			expectedWidth: 472, // floor((512 - 40 - 0) / 1) = 472
		},
		{
			name:          "narrow canvas",
			canvasWidth:   300,
			padding:       10,
			gap:           10,
			columns:       3,
			expectedWidth: 86, // floor((300 - 20 - 20) / 3) = 86
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := pipeline.LayoutInput{
				CanvasWidth:  tt.canvasWidth,
				CanvasHeight: 640,
				Columns:      tt.columns,
				Gap:          tt.gap,
				Padding:      tt.padding,
				BorderWidth:  1,
				Indent:       20,
				Outdent:      20,
			}
			result := ComputeLayout(input)
			if result.Columns[0].Width != tt.expectedWidth {
				t.Errorf("expected column width %d, got %d", tt.expectedWidth, result.Columns[0].Width)
			}
		})
	}
}

// TestComputeLayout_IndentOutdent tests that indent and outdent are applied correctly.
func TestComputeLayout_IndentOutdent(t *testing.T) {
	tests := []struct {
		name          string
		indent        int
		outdent       int
		expectedCol0Y int // first column Y
		expectedCol1Y int // second column Y
		expectedCol0H int // first column height
		expectedCol1H int // second column height
	}{
		{
			name:          "default indent/outdent",
			indent:        20,
			outdent:       20,
			expectedCol0Y: 20,  // padding (no indent for first)
			expectedCol1Y: 40,  // padding + indent
			expectedCol0H: 580, // baseHeight - outdent = 600 - 20
			expectedCol1H: 580, // baseHeight - indent = 600 - 20
		},
		{
			name:          "no indent/outdent",
			indent:        0,
			outdent:       0,
			expectedCol0Y: 20,  // padding
			expectedCol1Y: 20,  // padding (no indent)
			expectedCol0H: 600, // baseHeight
			expectedCol1H: 600, // baseHeight
		},
		{
			name:          "large indent",
			indent:        50,
			outdent:       30,
			expectedCol0Y: 20,  // padding
			expectedCol1Y: 70,  // padding + indent
			expectedCol0H: 570, // baseHeight - outdent = 600 - 30
			expectedCol1H: 550, // baseHeight - indent = 600 - 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := pipeline.LayoutInput{
				CanvasWidth:  512,
				CanvasHeight: 640,
				Columns:      3,
				Gap:          20,
				Padding:      20,
				BorderWidth:  1,
				Indent:       tt.indent,
				Outdent:      tt.outdent,
			}
			result := ComputeLayout(input)

			if result.Columns[0].Y != tt.expectedCol0Y {
				t.Errorf("column[0].Y: expected %d, got %d", tt.expectedCol0Y, result.Columns[0].Y)
			}
			if result.Columns[1].Y != tt.expectedCol1Y {
				t.Errorf("column[1].Y: expected %d, got %d", tt.expectedCol1Y, result.Columns[1].Y)
			}
			if result.Columns[0].Height != tt.expectedCol0H {
				t.Errorf("column[0].Height: expected %d, got %d", tt.expectedCol0H, result.Columns[0].Height)
			}
			if result.Columns[1].Height != tt.expectedCol1H {
				t.Errorf("column[1].Height: expected %d, got %d", tt.expectedCol1H, result.Columns[1].Height)
			}
		})
	}
}

// TestComputeLayout_WindowBorderHandling tests that border width is applied correctly to windows.
func TestComputeLayout_WindowBorderHandling(t *testing.T) {
	input := pipeline.LayoutInput{
		CanvasWidth:  512,
		CanvasHeight: 640,
		Columns:      3,
		Gap:          20,
		Padding:      20,
		BorderWidth:  1,
		Indent:       20,
		Outdent:      20,
	}
	result := ComputeLayout(input)

	// First window: X offset by borderWidth, Y offset by borderWidth
	// window.x = column.x + borderWidth = 20 + 1 = 21
	// window.y = column.y + borderWidth = 20 + 1 = 21 (only for first)
	if result.Windows[0].X != 21 {
		t.Errorf("window[0].X: expected 21, got %d", result.Windows[0].X)
	}
	if result.Windows[0].Y != 21 {
		t.Errorf("window[0].Y: expected 21, got %d", result.Windows[0].Y)
	}

	// Second window: X offset by borderWidth, Y same as column (no top border offset)
	// window.x = 184 + 1 = 185
	// window.y = 40 (no offset for non-first)
	if result.Windows[1].X != 185 {
		t.Errorf("window[1].X: expected 185, got %d", result.Windows[1].X)
	}
	if result.Windows[1].Y != 40 {
		t.Errorf("window[1].Y: expected 40, got %d", result.Windows[1].Y)
	}

	// Window width is column width minus 2*borderWidth
	// 144 - 2 = 142
	for i, w := range result.Windows {
		if w.Width != 142 {
			t.Errorf("window[%d].Width: expected 142, got %d", i, w.Width)
		}
	}

	// Only last window's height is reduced by borderWidth
	// First: 580 (same as column, borderWidth subtracted from first column's display)
	// Middle: 580
	// Last: 580 - 1 = 579
	if result.Windows[0].Height != 580 {
		t.Errorf("window[0].Height: expected 580, got %d", result.Windows[0].Height)
	}
	if result.Windows[1].Height != 580 {
		t.Errorf("window[1].Height: expected 580, got %d", result.Windows[1].Height)
	}
	if result.Windows[2].Height != 579 {
		t.Errorf("window[2].Height: expected 579, got %d", result.Windows[2].Height)
	}
}

// TestComputeLayout_ScrollTopCumulative tests that scrollTop values are cumulative.
func TestComputeLayout_ScrollTopCumulative(t *testing.T) {
	input := pipeline.LayoutInput{
		CanvasWidth:  512,
		CanvasHeight: 640,
		Columns:      3,
		Gap:          20,
		Padding:      20,
		BorderWidth:  1,
		Indent:       20,
		Outdent:      20,
	}
	result := ComputeLayout(input)

	// scrollTop values should be cumulative sum of window heights
	// window[0].scrollTop = 0
	// window[1].scrollTop = window[0].height = 580
	// window[2].scrollTop = window[0].height + window[1].height = 580 + 580 = 1160
	expectedScrollTops := []int{0, 580, 1160}
	for i, expected := range expectedScrollTops {
		if result.Windows[i].ScrollTop != expected {
			t.Errorf("window[%d].ScrollTop: expected %d, got %d", i, expected, result.Windows[i].ScrollTop)
		}
	}

	// Total scroll height should be sum of all window heights
	// 580 + 580 + 579 = 1739
	if result.Scroll.Height != 1739 {
		t.Errorf("scroll.Height: expected 1739, got %d", result.Scroll.Height)
	}
}

// TestComputeLayout_ScrollWidth tests that scroll width equals column width.
func TestComputeLayout_ScrollWidth(t *testing.T) {
	input := pipeline.LayoutInput{
		CanvasWidth:  512,
		CanvasHeight: 640,
		Columns:      3,
		Gap:          20,
		Padding:      20,
		BorderWidth:  1,
		Indent:       20,
		Outdent:      20,
	}
	result := ComputeLayout(input)

	// TypeScript: scroll.width = columnWidth = 144
	// This is the full column width, not the window width
	expectedScrollWidth := 144
	if result.Scroll.Width != expectedScrollWidth {
		t.Errorf("scroll.Width: expected %d, got %d", expectedScrollWidth, result.Scroll.Width)
	}
}

// TestComputeLayout_SingleColumn tests layout with only one column.
func TestComputeLayout_SingleColumn(t *testing.T) {
	input := pipeline.LayoutInput{
		CanvasWidth:  400,
		CanvasHeight: 600,
		Columns:      1,
		Gap:          0,
		Padding:      20,
		BorderWidth:  1,
		Indent:       0,
		Outdent:      0,
	}

	result := ComputeLayout(input)

	if len(result.Columns) != 1 {
		t.Fatalf("expected 1 column, got %d", len(result.Columns))
	}

	// Single column gets full width: 400 - 40 = 360
	if result.Columns[0].Width != 360 {
		t.Errorf("column width: expected 360, got %d", result.Columns[0].Width)
	}

	// Single column is both first and last
	// baseHeight = 600 - 40 = 560
	// column.height = 560 (no indent/outdent)
	// window.height = 560 - borderWidth = 559 (last window)
	if result.Windows[0].Height != 559 {
		t.Errorf("window height: expected 559, got %d", result.Windows[0].Height)
	}

	// scroll.width = columnWidth = 360
	if result.Scroll.Width != 360 {
		t.Errorf("scroll width: expected 360, got %d", result.Scroll.Width)
	}
}

// TestComputeLayout_LargeBorderWidth tests with larger border width.
func TestComputeLayout_LargeBorderWidth(t *testing.T) {
	input := pipeline.LayoutInput{
		CanvasWidth:  512,
		CanvasHeight: 640,
		Columns:      3,
		Gap:          20,
		Padding:      20,
		BorderWidth:  5,
		Indent:       20,
		Outdent:      20,
	}
	result := ComputeLayout(input)

	// Window width = columnWidth - 2*borderWidth = 144 - 10 = 134
	if result.Windows[0].Width != 134 {
		t.Errorf("window width: expected 134, got %d", result.Windows[0].Width)
	}

	// First window X = column.x + borderWidth = 20 + 5 = 25
	if result.Windows[0].X != 25 {
		t.Errorf("window[0].X: expected 25, got %d", result.Windows[0].X)
	}

	// First window Y = column.y + borderWidth = 20 + 5 = 25
	if result.Windows[0].Y != 25 {
		t.Errorf("window[0].Y: expected 25, got %d", result.Windows[0].Y)
	}

	// Last window height = column.height - borderWidth = 580 - 5 = 575
	lastIdx := len(result.Windows) - 1
	if result.Windows[lastIdx].Height != 575 {
		t.Errorf("last window height: expected 575, got %d", result.Windows[lastIdx].Height)
	}
}

// TestStage_Execute tests the stage wrapper.
func TestStage_Execute(t *testing.T) {
	stage := NewStage()
	input := pipeline.LayoutInput{
		CanvasWidth:  512,
		CanvasHeight: 640,
		Columns:      3,
		Gap:          20,
		Padding:      20,
		BorderWidth:  1,
		Indent:       20,
		Outdent:      20,
	}

	result, err := stage.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Columns) != input.Columns {
		t.Errorf("expected %d columns, got %d", input.Columns, len(result.Columns))
	}
}
