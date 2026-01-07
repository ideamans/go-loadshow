// Package summarizer provides summary generation for recording results.
package summarizer

// Formatter defines the interface for formatting a Summary.
type Formatter interface {
	// Format converts a Summary to a formatted string.
	Format(summary *Summary) string
}

// FormatFunc is a function adapter for the Formatter interface.
type FormatFunc func(summary *Summary) string

// Format implements the Formatter interface.
func (f FormatFunc) Format(summary *Summary) string {
	return f(summary)
}
