package mocks

import (
	"fmt"
	"sync"

	"github.com/user/loadshow/pkg/ports"
)

// FileSystem is a mock implementation of ports.FileSystem.
type FileSystem struct {
	mu    sync.RWMutex
	files map[string][]byte
	dirs  map[string]bool

	ReadFileFunc  func(path string) ([]byte, error)
	WriteFileFunc func(path string, data []byte) error
	MkdirAllFunc  func(path string) error
	ExistsFunc    func(path string) (bool, error)
	RemoveFunc    func(path string) error
}

// NewFileSystem creates a new mock FileSystem.
func NewFileSystem() *FileSystem {
	return &FileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *FileSystem) ReadFile(path string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(path)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func (m *FileSystem) WriteFile(path string, data []byte) error {
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(path, data)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[path] = data
	return nil
}

func (m *FileSystem) MkdirAll(path string) error {
	if m.MkdirAllFunc != nil {
		return m.MkdirAllFunc(path)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirs[path] = true
	return nil
}

func (m *FileSystem) Exists(path string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(path)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.files[path]; ok {
		return true, nil
	}
	if _, ok := m.dirs[path]; ok {
		return true, nil
	}
	return false, nil
}

func (m *FileSystem) Remove(path string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(path)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, path)
	delete(m.dirs, path)
	return nil
}

// GetFile returns the contents of a file (for test verification).
func (m *FileSystem) GetFile(path string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.files[path]
	return data, ok
}

// GetAllFiles returns all files (for test verification).
func (m *FileSystem) GetAllFiles() map[string][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string][]byte)
	for k, v := range m.files {
		result[k] = v
	}
	return result
}

var _ ports.FileSystem = (*FileSystem)(nil)
