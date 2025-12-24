// Package logger provides logging implementations.
package logger

import (
	"fmt"
	"os"

	"github.com/ideamans/go-l10n"
	"github.com/mattn/go-isatty"
	"github.com/user/loadshow/pkg/ports"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

// ConsoleLogger logs messages to the console with color support.
type ConsoleLogger struct {
	level     ports.LogLevel
	component string
	color     bool
}

// NewConsole creates a new console logger with the specified level.
// Color output is automatically enabled when stdout is a terminal.
func NewConsole(level ports.LogLevel) *ConsoleLogger {
	return &ConsoleLogger{
		level:     level,
		component: "",
		color:     isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()),
	}
}

// Debug logs a debug message.
func (l *ConsoleLogger) Debug(msg string, args ...interface{}) {
	if l.level > ports.LevelDebug {
		return
	}
	l.log(ports.LevelDebug, msg, args...)
}

// Info logs an informational message.
func (l *ConsoleLogger) Info(msg string, args ...interface{}) {
	if l.level > ports.LevelInfo {
		return
	}
	l.log(ports.LevelInfo, msg, args...)
}

// Warn logs a warning message.
func (l *ConsoleLogger) Warn(msg string, args ...interface{}) {
	if l.level > ports.LevelWarn {
		return
	}
	l.log(ports.LevelWarn, msg, args...)
}

// Error logs an error message.
func (l *ConsoleLogger) Error(msg string, args ...interface{}) {
	if l.level > ports.LevelError {
		return
	}
	l.log(ports.LevelError, msg, args...)
}

// WithComponent returns a new logger with the specified component name.
func (l *ConsoleLogger) WithComponent(component string) ports.Logger {
	return &ConsoleLogger{
		level:     l.level,
		component: component,
		color:     l.color,
	}
}

// log outputs a log message with appropriate formatting.
func (l *ConsoleLogger) log(level ports.LogLevel, msg string, args ...interface{}) {
	// Translate message using go-l10n
	translated := l10n.F(msg, args...)

	// Build output line
	var output string
	if l.component != "" {
		if l.color {
			output = fmt.Sprintf("%s[%s]%s %s", colorCyan, l.component, colorReset, translated)
		} else {
			output = fmt.Sprintf("[%s] %s", l.component, translated)
		}
	} else {
		output = translated
	}

	// Apply level-specific coloring
	if l.color {
		switch level {
		case ports.LevelDebug:
			output = colorGray + output + colorReset
		case ports.LevelWarn:
			output = colorYellow + output + colorReset
		case ports.LevelError:
			output = colorRed + output + colorReset
		}
	}

	// Output to appropriate stream
	if level >= ports.LevelWarn {
		fmt.Fprintln(os.Stderr, output)
	} else {
		fmt.Fprintln(os.Stdout, output)
	}
}
