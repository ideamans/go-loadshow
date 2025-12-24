package mocks

import (
	"image"
	"image/color"

	"github.com/user/loadshow/pkg/ports"
)

// Renderer is a mock implementation of ports.Renderer.
type Renderer struct {
	CreateCanvasFunc func(width, height int, bg color.Color) ports.Canvas
	DecodeImageFunc  func(data []byte, format ports.ImageFormat) (image.Image, error)
	EncodeImageFunc  func(img image.Image, format ports.ImageFormat, quality int) ([]byte, error)
	ResizeImageFunc  func(img image.Image, width, height int) image.Image
}

func (m *Renderer) CreateCanvas(width, height int, bg color.Color) ports.Canvas {
	if m.CreateCanvasFunc != nil {
		return m.CreateCanvasFunc(width, height, bg)
	}
	return &Canvas{width: width, height: height}
}

func (m *Renderer) DecodeImage(data []byte, format ports.ImageFormat) (image.Image, error) {
	if m.DecodeImageFunc != nil {
		return m.DecodeImageFunc(data, format)
	}
	return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
}

func (m *Renderer) EncodeImage(img image.Image, format ports.ImageFormat, quality int) ([]byte, error) {
	if m.EncodeImageFunc != nil {
		return m.EncodeImageFunc(img, format, quality)
	}
	return []byte{}, nil
}

func (m *Renderer) ResizeImage(img image.Image, width, height int) image.Image {
	if m.ResizeImageFunc != nil {
		return m.ResizeImageFunc(img, width, height)
	}
	return image.NewRGBA(image.Rect(0, 0, width, height))
}

var _ ports.Renderer = (*Renderer)(nil)

// Canvas is a mock implementation of ports.Canvas.
type Canvas struct {
	width  int
	height int
	img    *image.RGBA
}

func (m *Canvas) DrawImage(img image.Image, x, y int) {}

func (m *Canvas) DrawImageScaled(img image.Image, x, y, width, height int) {}

func (m *Canvas) DrawRect(x, y, w, h int, c color.Color) {}

func (m *Canvas) DrawRoundedRect(x, y, w, h, radius int, c color.Color) {}

func (m *Canvas) DrawRectStroke(x, y, w, h int, c color.Color, strokeWidth float64) {}

func (m *Canvas) DrawText(text string, x, y int, style ports.TextStyle) {}

func (m *Canvas) DrawLine(x1, y1, x2, y2 int, c color.Color, width float64) {}

func (m *Canvas) ToImage() image.Image {
	if m.img != nil {
		return m.img
	}
	return image.NewRGBA(image.Rect(0, 0, m.width, m.height))
}

var _ ports.Canvas = (*Canvas)(nil)
