package osfilesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystem_WriteAndReadFile(t *testing.T) {
	fs := New()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "osfilesystem_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write file
	testPath := filepath.Join(tmpDir, "test.txt")
	testData := []byte("hello world")

	err = fs.WriteFile(testPath, testData)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read file
	data, err := fs.ReadFile(testPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("expected %q, got %q", testData, data)
	}
}

func TestFileSystem_WriteFileCreatesParentDirs(t *testing.T) {
	fs := New()

	tmpDir, err := os.MkdirTemp("", "osfilesystem_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write to nested path
	testPath := filepath.Join(tmpDir, "a", "b", "c", "test.txt")
	err = fs.WriteFile(testPath, []byte("test"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify file exists
	exists, err := fs.Exists(testPath)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("expected file to exist")
	}
}

func TestFileSystem_MkdirAll(t *testing.T) {
	fs := New()

	tmpDir, err := os.MkdirTemp("", "osfilesystem_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testPath := filepath.Join(tmpDir, "a", "b", "c")
	err = fs.MkdirAll(testPath)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	exists, err := fs.Exists(testPath)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("expected directory to exist")
	}
}

func TestFileSystem_Exists(t *testing.T) {
	fs := New()

	tmpDir, err := os.MkdirTemp("", "osfilesystem_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test existing file
	testPath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testPath, []byte("test"), 0644)

	exists, err := fs.Exists(testPath)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("expected file to exist")
	}

	// Test non-existing file
	exists, err = fs.Exists(filepath.Join(tmpDir, "nonexistent.txt"))
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("expected file to not exist")
	}
}

func TestFileSystem_Remove(t *testing.T) {
	fs := New()

	tmpDir, err := os.MkdirTemp("", "osfilesystem_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file
	testPath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testPath, []byte("test"), 0644)

	// Remove file
	err = fs.Remove(testPath)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify removed
	exists, _ := fs.Exists(testPath)
	if exists {
		t.Error("expected file to be removed")
	}
}
