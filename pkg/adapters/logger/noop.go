package logger

import "github.com/user/loadshow/pkg/ports"

// NoopLogger is a logger that discards all messages.
// Used for quiet mode when no output is desired.
type NoopLogger struct{}

// NewNoop creates a new no-op logger.
func NewNoop() *NoopLogger {
	return &NoopLogger{}
}

// Debug does nothing.
func (l *NoopLogger) Debug(msg string, args ...interface{}) {}

// Info does nothing.
func (l *NoopLogger) Info(msg string, args ...interface{}) {}

// Warn does nothing.
func (l *NoopLogger) Warn(msg string, args ...interface{}) {}

// Error does nothing.
func (l *NoopLogger) Error(msg string, args ...interface{}) {}

// WithComponent returns the same no-op logger.
func (l *NoopLogger) WithComponent(component string) ports.Logger {
	return l
}
