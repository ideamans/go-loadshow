package ports

// FileSystem abstracts file system operations.
type FileSystem interface {
	// ReadFile reads the entire contents of a file.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes data to a file, creating it if necessary.
	WriteFile(path string, data []byte) error

	// MkdirAll creates a directory and all parent directories.
	MkdirAll(path string) error

	// Exists checks if a file or directory exists.
	Exists(path string) (bool, error)

	// Remove deletes a file or empty directory.
	Remove(path string) error
}
