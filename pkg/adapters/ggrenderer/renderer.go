// Package ggrenderer provides a renderer implementation using the gg library.
package ggrenderer

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"

	"github.com/fogleman/gg"
	"golang.org/x/image/draw"

	"github.com/user/loadshow/pkg/ports"
)

// Renderer implements ports.Renderer using the gg library.
type Renderer struct{}

// New creates a new Renderer.
func New() *Renderer {
	return &Renderer{}
}

// CreateCanvas creates a new drawing canvas.
func (r *Renderer) CreateCanvas(width, height int, bg color.Color) ports.Canvas {
	dc := gg.NewContext(width, height)
	dc.SetColor(bg)
	dc.Clear()
	return &Canvas{dc: dc}
}

// DecodeImage decodes image data into an image.Image.
func (r *Renderer) DecodeImage(data []byte, format ports.ImageFormat) (image.Image, error) {
	reader := bytes.NewReader(data)

	switch format {
	case ports.FormatJPEG:
		return jpeg.Decode(reader)
	case ports.FormatPNG:
		return png.Decode(reader)
	default:
		// Try to auto-detect
		img, _, err := image.Decode(reader)
		return img, err
	}
}

// EncodeImage encodes an image to the specified format.
func (r *Renderer) EncodeImage(img image.Image, format ports.ImageFormat, quality int) ([]byte, error) {
	var buf bytes.Buffer

	switch format {
	case ports.FormatJPEG:
		opts := &jpeg.Options{Quality: quality}
		if err := jpeg.Encode(&buf, img, opts); err != nil {
			return nil, fmt.Errorf("encode JPEG: %w", err)
		}
	case ports.FormatPNG:
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("encode PNG: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %d", format)
	}

	return buf.Bytes(), nil
}

// ResizeImage resizes an image to the specified dimensions.
func (r *Renderer) ResizeImage(img image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

// Ensure Renderer implements ports.Renderer
var _ ports.Renderer = (*Renderer)(nil)

// Canvas implements ports.Canvas using gg.Context.
type Canvas struct {
	dc *gg.Context
}

// DrawImage draws an image at the specified position.
func (c *Canvas) DrawImage(img image.Image, x, y int) {
	c.dc.DrawImage(img, x, y)
}

// DrawImageScaled draws an image scaled to the specified dimensions.
func (c *Canvas) DrawImageScaled(img image.Image, x, y, width, height int) {
	c.dc.Push()
	defer c.dc.Pop()

	bounds := img.Bounds()
	scaleX := float64(width) / float64(bounds.Dx())
	scaleY := float64(height) / float64(bounds.Dy())

	c.dc.Translate(float64(x), float64(y))
	c.dc.Scale(scaleX, scaleY)
	c.dc.DrawImage(img, 0, 0)
}

// DrawRect draws a filled rectangle.
func (c *Canvas) DrawRect(x, y, w, h int, col color.Color) {
	c.dc.SetColor(col)
	c.dc.DrawRectangle(float64(x), float64(y), float64(w), float64(h))
	c.dc.Fill()
}

// DrawRoundedRect draws a filled rounded rectangle.
func (c *Canvas) DrawRoundedRect(x, y, w, h, radius int, col color.Color) {
	c.dc.SetColor(col)
	c.dc.DrawRoundedRectangle(float64(x), float64(y), float64(w), float64(h), float64(radius))
	c.dc.Fill()
}

// DrawRectStroke draws a rectangle outline.
func (c *Canvas) DrawRectStroke(x, y, w, h int, col color.Color, strokeWidth float64) {
	c.dc.SetColor(col)
	c.dc.SetLineWidth(strokeWidth)
	c.dc.DrawRectangle(float64(x), float64(y), float64(w), float64(h))
	c.dc.Stroke()
}

// DrawText draws text at the specified position.
func (c *Canvas) DrawText(text string, x, y int, style ports.TextStyle) {
	c.dc.SetColor(style.Color)

	// Try to load font if specified
	if style.FontPath != "" {
		if err := c.dc.LoadFontFace(style.FontPath, style.FontSize); err != nil {
			// Fall back to default
		}
	}

	// Calculate alignment offset
	ax := 0.0
	switch style.Align {
	case ports.AlignCenter:
		ax = 0.5
	case ports.AlignRight:
		ax = 1.0
	}

	c.dc.DrawStringAnchored(text, float64(x), float64(y), ax, 0.5)
}

// DrawLine draws a line between two points.
func (c *Canvas) DrawLine(x1, y1, x2, y2 int, col color.Color, width float64) {
	c.dc.SetColor(col)
	c.dc.SetLineWidth(width)
	c.dc.DrawLine(float64(x1), float64(y1), float64(x2), float64(y2))
	c.dc.Stroke()
}

// ToImage returns the canvas as an image.Image.
func (c *Canvas) ToImage() image.Image {
	return c.dc.Image()
}

// Ensure Canvas implements ports.Canvas
var _ ports.Canvas = (*Canvas)(nil)
