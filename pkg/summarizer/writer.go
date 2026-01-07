package summarizer

import (
	"fmt"
	"os"
	"path/filepath"
)

// Writer writes formatted summaries to files.
type Writer struct {
	formatter Formatter
}

// NewWriter creates a new Writer with the given Formatter.
func NewWriter(formatter Formatter) *Writer {
	return &Writer{
		formatter: formatter,
	}
}

// Write formats the summary and writes it to the specified path.
// Creates parent directories if they don't exist.
func (w *Writer) Write(path string, summary *Summary) error {
	// Format the summary
	content := w.formatter.Format(summary)

	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
