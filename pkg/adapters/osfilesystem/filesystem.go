// Package osfilesystem provides a filesystem implementation using the os package.
package osfilesystem

import (
	"os"
	"path/filepath"

	"github.com/user/loadshow/pkg/ports"
)

// FileSystem implements ports.FileSystem using the os package.
type FileSystem struct{}

// New creates a new FileSystem.
func New() *FileSystem {
	return &FileSystem{}
}

// ReadFile reads the entire contents of a file.
func (fs *FileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file, creating it if necessary.
func (fs *FileSystem) WriteFile(path string, data []byte) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0644)
}

// MkdirAll creates a directory and all parent directories.
func (fs *FileSystem) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

// Exists checks if a file or directory exists.
func (fs *FileSystem) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Remove deletes a file or empty directory.
func (fs *FileSystem) Remove(path string) error {
	return os.Remove(path)
}

// Ensure FileSystem implements ports.FileSystem
var _ ports.FileSystem = (*FileSystem)(nil)
