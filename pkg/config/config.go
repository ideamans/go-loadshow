// Package config provides configuration loading and management.
package config

import (
	"image/color"
	"os"

	"github.com/user/loadshow/pkg/orchestrator"
	"github.com/user/loadshow/pkg/ports"
	"gopkg.in/yaml.v3"
)

// Config represents the full configuration for loadshow.
type Config struct {
	// Input/Output
	URL        string `yaml:"url"`
	OutputPath string `yaml:"output"`

	// Layout
	CanvasWidth    int `yaml:"canvas_width"`
	CanvasHeight   int `yaml:"canvas_height"`
	Columns        int `yaml:"columns"`
	Gap            int `yaml:"gap"`
	Padding        int `yaml:"padding"`
	BorderWidth    int `yaml:"border_width"`
	Indent         int `yaml:"indent"`
	Outdent        int `yaml:"outdent"`
	BannerHeight   int `yaml:"banner_height"`
	ProgressHeight int `yaml:"progress_height"`

	// Recording
	ViewportWidth int                    `yaml:"viewport_width"`
	TimeoutMs     int                    `yaml:"timeout_ms"`
	Network       NetworkConfig          `yaml:"network"`
	CPUThrottling float64                `yaml:"cpu_throttling"`
	Headers       map[string]string      `yaml:"headers"`
	UserAgent     string                 `yaml:"user_agent"`
	Headless      bool                   `yaml:"headless"`
	ChromePath    string                 `yaml:"chrome_path"`

	// Banner
	BannerEnabled bool        `yaml:"banner"`
	BannerTheme   ThemeConfig `yaml:"banner_theme"`

	// Composite
	Workers      int         `yaml:"workers"`
	ShowProgress bool        `yaml:"show_progress"`
	Theme        ThemeConfig `yaml:"theme"`

	// Encoding
	Quality int     `yaml:"quality"`
	Bitrate int     `yaml:"bitrate"`
	FPS     float64 `yaml:"fps"`
	OutroMs int     `yaml:"outro_ms"`

	// Debug
	Debug    bool   `yaml:"debug"`
	DebugDir string `yaml:"debug_dir"`
}

// NetworkConfig represents network throttling settings.
type NetworkConfig struct {
	LatencyMs     int  `yaml:"latency_ms"`
	DownloadSpeed int  `yaml:"download_speed"`
	UploadSpeed   int  `yaml:"upload_speed"`
	Offline       bool `yaml:"offline"`
}

// ThemeConfig represents theming options.
type ThemeConfig struct {
	BackgroundColor  string `yaml:"background_color"`
	TextColor        string `yaml:"text_color"`
	AccentColor      string `yaml:"accent_color"`
	BorderColor      string `yaml:"border_color"`
	ProgressBarColor string `yaml:"progress_bar_color"`
}

// Defaults returns a Config with default values.
func Defaults() Config {
	return Config{
		// Layout
		CanvasWidth:    512,
		CanvasHeight:   640,
		Columns:        3,
		Gap:            20,
		Padding:        20,
		BorderWidth:    1,
		Indent:         20,
		Outdent:        20,
		BannerHeight:   80,
		ProgressHeight: 16,

		// Recording
		ViewportWidth: 1200,
		TimeoutMs:     30000,
		CPUThrottling: 1.0,
		Headless:      true,

		// Banner
		BannerEnabled: true,
		BannerTheme: ThemeConfig{
			BackgroundColor: "#1a1a2e",
			TextColor:       "#ffffff",
			AccentColor:     "#4ade80",
		},

		// Composite
		Workers:      4,
		ShowProgress: true,
		Theme: ThemeConfig{
			BackgroundColor:  "#1a1a2e",
			BorderColor:      "#333355",
			ProgressBarColor: "#4ade80",
		},

		// Encoding
		Quality: 30,
		FPS:     30.0,
		OutroMs: 1000,

		// Debug
		DebugDir: "./debug",
	}
}

// LoadFromFile loads configuration from a YAML file.
func LoadFromFile(path string) (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// ParseColor parses a hex color string to color.Color.
func ParseColor(hex string) color.Color {
	if len(hex) == 0 {
		return color.Black
	}

	if hex[0] == '#' {
		hex = hex[1:]
	}

	if len(hex) != 6 {
		return color.Black
	}

	var r, g, b uint8
	for i, c := range []byte{hex[0], hex[1]} {
		v := hexValue(c)
		if i == 0 {
			r = v << 4
		} else {
			r |= v
		}
	}
	for i, c := range []byte{hex[2], hex[3]} {
		v := hexValue(c)
		if i == 0 {
			g = v << 4
		} else {
			g |= v
		}
	}
	for i, c := range []byte{hex[4], hex[5]} {
		v := hexValue(c)
		if i == 0 {
			b = v << 4
		} else {
			b |= v
		}
	}

	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func hexValue(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

// ToOrchestratorConfig converts Config to orchestrator.Config.
func (c Config) ToOrchestratorConfig() orchestrator.Config {
	return orchestrator.Config{
		URL:        c.URL,
		OutputPath: c.OutputPath,

		CanvasWidth:    c.CanvasWidth,
		CanvasHeight:   c.CanvasHeight,
		Columns:        c.Columns,
		Gap:            c.Gap,
		Padding:        c.Padding,
		BorderWidth:    c.BorderWidth,
		Indent:         c.Indent,
		Outdent:        c.Outdent,
		ProgressHeight: c.ProgressHeight,

		ViewportWidth: c.ViewportWidth,
		TimeoutMs:     c.TimeoutMs,
		NetworkConditions: ports.NetworkConditions{
			LatencyMs:     c.Network.LatencyMs,
			DownloadSpeed: c.Network.DownloadSpeed,
			UploadSpeed:   c.Network.UploadSpeed,
			Offline:       c.Network.Offline,
		},
		CPUThrottling: c.CPUThrottling,
		Headers:       c.Headers,

		BannerEnabled: c.BannerEnabled,
		BannerHeight:  c.BannerHeight,
		ShowProgress:  c.ShowProgress,

		Quality: c.Quality,
		Bitrate: c.Bitrate,
		FPS:     c.FPS,
		OutroMs: c.OutroMs,
	}
}
