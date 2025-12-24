package ports

import (
	"image"
	"image/color"
)

// Renderer abstracts image processing operations.
type Renderer interface {
	// CreateCanvas creates a new drawing canvas with the specified dimensions and background color.
	CreateCanvas(width, height int, bg color.Color) Canvas

	// DecodeImage decodes image data into an image.Image.
	DecodeImage(data []byte, format ImageFormat) (image.Image, error)

	// EncodeImage encodes an image to the specified format.
	EncodeImage(img image.Image, format ImageFormat, quality int) ([]byte, error)

	// ResizeImage resizes an image to the specified dimensions.
	ResizeImage(img image.Image, width, height int) image.Image
}

// Canvas provides drawing operations for compositing images.
type Canvas interface {
	// DrawImage draws an image at the specified position.
	DrawImage(img image.Image, x, y int)

	// DrawImageScaled draws an image scaled to the specified dimensions.
	DrawImageScaled(img image.Image, x, y, width, height int)

	// DrawRect draws a filled rectangle.
	DrawRect(x, y, w, h int, c color.Color)

	// DrawRoundedRect draws a filled rounded rectangle.
	DrawRoundedRect(x, y, w, h, radius int, c color.Color)

	// DrawRectStroke draws a rectangle outline.
	DrawRectStroke(x, y, w, h int, c color.Color, strokeWidth float64)

	// DrawText draws text at the specified position.
	DrawText(text string, x, y int, style TextStyle)

	// MeasureText returns the width and height of the text.
	MeasureText(text string, style TextStyle) (width, height float64)

	// DrawLine draws a line between two points.
	DrawLine(x1, y1, x2, y2 int, c color.Color, width float64)

	// ToImage returns the canvas as an image.Image.
	ToImage() image.Image
}

// TextStyle defines text rendering properties.
type TextStyle struct {
	FontSize float64
	FontPath string
	Color    color.Color
	Align    TextAlign
}

// TextAlign specifies text alignment.
type TextAlign int

const (
	AlignLeft TextAlign = iota
	AlignCenter
	AlignRight
)

// ImageFormat specifies image encoding format.
type ImageFormat int

const (
	FormatJPEG ImageFormat = iota
	FormatPNG
)
