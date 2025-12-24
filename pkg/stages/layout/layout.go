// Package layout implements the layout calculation stage.
package layout

import (
	"context"

	"github.com/user/loadshow/pkg/pipeline"
)

// Stage calculates the layout for the recording composition.
// This is a pure function with no external dependencies.
type Stage struct{}

// NewStage creates a new layout stage.
func NewStage() *Stage {
	return &Stage{}
}

// Execute calculates the layout based on the input parameters.
// The layout determines how the recorded page will be displayed in multiple columns.
func (s *Stage) Execute(ctx context.Context, input pipeline.LayoutInput) (pipeline.LayoutResult, error) {
	return ComputeLayout(input), nil
}

// ComputeLayout performs the layout calculation.
// This is exposed as a standalone function for testing and reuse.
//
// IMPORTANT: Following TypeScript implementation exactly:
// - Layout positions are RELATIVE to content area (not including banner/progress offset)
// - column.y = padding + (isFirst ? 0 : indent)
// - Composition will add canvasOffset (bannerHeight + progressHeight) when drawing
func ComputeLayout(input pipeline.LayoutInput) pipeline.LayoutResult {
	// Calculate column width
	// columnWidth = (canvasWidth - padding*2 - gap*(columns-1)) / columns
	totalGaps := input.Gap * (input.Columns - 1)
	totalPadding := input.Padding * 2
	availableWidth := input.CanvasWidth - totalPadding - totalGaps
	columnWidth := availableWidth / input.Columns

	// Base height for columns (content area height, not including banner/progress)
	// TypeScript: baseHeight = canvasHeight - padding * 2
	baseHeight := input.CanvasHeight - input.Padding*2

	columns := make([]pipeline.Rectangle, input.Columns)
	windows := make([]pipeline.Window, input.Columns)
	currentScrollTop := 0

	for i := 0; i < input.Columns; i++ {
		isFirst := i == 0
		isLast := i == input.Columns-1

		// Column rectangle (for decoration/border)
		// TypeScript: column.x = padding + i * (columnWidth + gap)
		// TypeScript: column.y = padding + (isFirst ? 0 : indent)
		colX := input.Padding + i*(columnWidth+input.Gap)
		colY := input.Padding
		if !isFirst {
			colY += input.Indent
		}
		// TypeScript: column.height = baseHeight - (isFirst ? outdent : indent)
		colHeight := baseHeight
		if isFirst {
			colHeight -= input.Outdent
		} else {
			colHeight -= input.Indent
		}

		columns[i] = pipeline.Rectangle{
			X:      colX,
			Y:      colY,
			Width:  columnWidth,
			Height: colHeight,
		}

		// Window rectangle (actual content display area, inside border)
		// TypeScript:
		// window.x = column.x + borderWidth
		// window.y = column.y + (isFirst ? borderWidth : 0)
		// window.width = column.width - borderWidth*2
		// window.height = column.height - (isLast ? borderWidth : 0)
		winX := colX + input.BorderWidth
		winY := colY
		if isFirst {
			winY += input.BorderWidth
		}
		winWidth := columnWidth - input.BorderWidth*2
		winHeight := colHeight
		if isLast {
			winHeight -= input.BorderWidth
		}

		windows[i] = pipeline.Window{
			Rectangle: pipeline.Rectangle{
				X:      winX,
				Y:      winY,
				Width:  winWidth,
				Height: winHeight,
			},
			ScrollTop: currentScrollTop,
		}

		currentScrollTop += winHeight
	}

	// Calculate scroll dimensions (viewport size for recording)
	// TypeScript: scroll.width = columnWidth (full column width, not window width)
	scrollWidth := columnWidth
	scrollHeight := currentScrollTop

	// Banner area (at the very top, full width minus padding)
	bannerArea := pipeline.Rectangle{}
	if input.BannerHeight > 0 {
		bannerArea = pipeline.Rectangle{
			X:      0,
			Y:      0,
			Width:  input.CanvasWidth,
			Height: input.BannerHeight,
		}
	}

	// Progress bar area (just below banner, at top: bannerHeight)
	progressArea := pipeline.Rectangle{}
	if input.ProgressHeight > 0 {
		progressArea = pipeline.Rectangle{
			X:      0,
			Y:      input.BannerHeight,
			Width:  input.CanvasWidth,
			Height: input.ProgressHeight,
		}
	}

	// Content area (relative positions, will be offset by canvasOffset in composition)
	contentArea := pipeline.Rectangle{
		X:      input.Padding,
		Y:      input.Padding,
		Width:  input.CanvasWidth - input.Padding*2,
		Height: baseHeight,
	}

	return pipeline.LayoutResult{
		Scroll: pipeline.Dimension{
			Width:  scrollWidth,
			Height: scrollHeight,
		},
		Columns:      columns,
		Windows:      windows,
		BannerArea:   bannerArea,
		ProgressArea: progressArea,
		ContentArea:  contentArea,
	}
}
