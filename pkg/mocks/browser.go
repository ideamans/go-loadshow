// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/user/loadshow/pkg/ports"
)

// Browser is a mock implementation of ports.Browser.
type Browser struct {
	LaunchFunc               func(ctx context.Context, opts ports.BrowserOptions) error
	NavigateFunc             func(url string) error
	SetViewportFunc          func(viewportWidth, viewportHeight, screenWidth, screenHeight int, deviceScaleFactor float64) error
	SetNetworkConditionsFunc func(conditions ports.NetworkConditions) error
	SetCPUThrottlingFunc     func(rate float64) error
	StartScreencastFunc      func(quality, maxWidth, maxHeight int) (<-chan ports.ScreenFrame, error)
	StopScreencastFunc       func() error
	GetPageInfoFunc          func() (*ports.PageInfo, error)
	GetPerformanceTimingFunc func() (*ports.PerformanceTiming, error)
	CloseFunc                func() error
}

func (m *Browser) Launch(ctx context.Context, opts ports.BrowserOptions) error {
	if m.LaunchFunc != nil {
		return m.LaunchFunc(ctx, opts)
	}
	return nil
}

func (m *Browser) Navigate(url string) error {
	if m.NavigateFunc != nil {
		return m.NavigateFunc(url)
	}
	return nil
}

func (m *Browser) SetViewport(viewportWidth, viewportHeight, screenWidth, screenHeight int, deviceScaleFactor float64) error {
	if m.SetViewportFunc != nil {
		return m.SetViewportFunc(viewportWidth, viewportHeight, screenWidth, screenHeight, deviceScaleFactor)
	}
	return nil
}

func (m *Browser) SetNetworkConditions(conditions ports.NetworkConditions) error {
	if m.SetNetworkConditionsFunc != nil {
		return m.SetNetworkConditionsFunc(conditions)
	}
	return nil
}

func (m *Browser) SetCPUThrottling(rate float64) error {
	if m.SetCPUThrottlingFunc != nil {
		return m.SetCPUThrottlingFunc(rate)
	}
	return nil
}

func (m *Browser) StartScreencast(quality, maxWidth, maxHeight int) (<-chan ports.ScreenFrame, error) {
	if m.StartScreencastFunc != nil {
		return m.StartScreencastFunc(quality, maxWidth, maxHeight)
	}
	ch := make(chan ports.ScreenFrame)
	close(ch)
	return ch, nil
}

func (m *Browser) StopScreencast() error {
	if m.StopScreencastFunc != nil {
		return m.StopScreencastFunc()
	}
	return nil
}

func (m *Browser) GetPageInfo() (*ports.PageInfo, error) {
	if m.GetPageInfoFunc != nil {
		return m.GetPageInfoFunc()
	}
	return &ports.PageInfo{}, nil
}

func (m *Browser) GetPerformanceTiming() (*ports.PerformanceTiming, error) {
	if m.GetPerformanceTimingFunc != nil {
		return m.GetPerformanceTimingFunc()
	}
	return &ports.PerformanceTiming{}, nil
}

func (m *Browser) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// Ensure Browser implements ports.Browser
var _ ports.Browser = (*Browser)(nil)
