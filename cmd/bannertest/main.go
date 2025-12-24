package main

import (
	"context"
	"fmt"
	"image/png"
	"os"

	"github.com/user/loadshow/pkg/adapters/capturehtml"
	"github.com/user/loadshow/pkg/stages/banner"
)

func main() {
	capturer := capturehtml.New()

	widths := []int{400, 512, 640, 800}

	for _, width := range widths {
		vars := banner.NewTemplateVars(
			width,
			"https://example.com/very-long-url-path/that/should/be/truncated",
			"サンプルページタイトル - Example Page Title",
			2500,
			1024*1024*5, // 5MB
		)

		html, err := banner.RenderHTML(vars)
		if err != nil {
			fmt.Printf("Error rendering HTML: %v\n", err)
			continue
		}

		img, err := capturer.CaptureHTMLWithViewport(context.Background(), html, width, 200)
		if err != nil {
			fmt.Printf("Error capturing HTML: %v\n", err)
			continue
		}

		filename := fmt.Sprintf("tmp/banner_%d.png", width)
		f, err := os.Create(filename)
		if err != nil {
			fmt.Printf("Error creating file: %v\n", err)
			continue
		}

		if err := png.Encode(f, img); err != nil {
			fmt.Printf("Error encoding PNG: %v\n", err)
		}
		f.Close()

		fmt.Printf("Generated %s (%dx%d)\n", filename, img.Bounds().Dx(), img.Bounds().Dy())
	}
}
