// Package ports defines the Logger interface for logging abstraction.
package ports

// LogLevel represents the severity level of a log message.
type LogLevel int

const (
	// LevelDebug is for detailed debugging information.
	// Used for component-level internal processing logs.
	LevelDebug LogLevel = iota
	// LevelInfo is for informational messages.
	// Used for orchestration-level logs.
	LevelInfo
	// LevelWarn is for warning messages.
	// Used for recoverable problems that don't stop processing.
	LevelWarn
	// LevelError is for error messages.
	// Used for unrecoverable problems that stop processing.
	LevelError
	// LevelQuiet suppresses all log output.
	LevelQuiet
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelQuiet:
		return "quiet"
	default:
		return "unknown"
	}
}

// ParseLogLevel parses a string into a LogLevel.
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	case "quiet":
		return LevelQuiet
	default:
		return LevelInfo
	}
}

// Logger abstracts logging operations with multi-language support.
type Logger interface {
	// Debug logs a debug message with optional format arguments.
	// Debug messages are for internal component processing details.
	// The msg parameter is the message key that can be translated.
	Debug(msg string, args ...interface{})

	// Info logs an informational message with optional format arguments.
	// Info messages are for orchestration-level progress updates.
	Info(msg string, args ...interface{})

	// Warn logs a warning message with optional format arguments.
	// Warn messages indicate recoverable problems.
	Warn(msg string, args ...interface{})

	// Error logs an error message with optional format arguments.
	// Error messages indicate unrecoverable problems.
	Error(msg string, args ...interface{})

	// WithComponent returns a new Logger that prefixes messages with the component name.
	// Component loggers are typically used at debug level for internal processing logs.
	WithComponent(component string) Logger
}
