// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package platform provides shared file management functionality.
package platform

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/janderssonse/karei/internal/domain"
)

// FileManager implements the FileManager port for real file operations.
type FileManager struct {
	verbose bool
}

// NewFileManager creates a new file manager.
func NewFileManager(verbose bool) *FileManager {
	return &FileManager{
		verbose: verbose,
	}
}

// FileExists checks if a file exists.
func (f *FileManager) FileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// EnsureDir creates a directory and all parent directories if they don't exist.
func (f *FileManager) EnsureDir(path string) error {
	if f.verbose {
		fmt.Printf("Ensuring directory exists: %s\n", path)
	}

	// #nosec G301 - Standard directory permissions for application directories
	return os.MkdirAll(path, 0755)
}

// CopyFile copies a file from source to destination.
func (f *FileManager) CopyFile(src, dest string) error {
	if f.verbose {
		fmt.Printf("Copying file: %s -> %s\n", src, dest)
	}

	// Ensure destination directory exists
	if err := f.EnsureDir(filepath.Dir(dest)); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// #nosec G304 - File path comes from trusted application code
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}

	defer func() { _ = srcFile.Close() }()

	// #nosec G304 - File path comes from trusted application code
	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}

	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return destFile.Sync()
}

// WriteFile writes data to a file.
func (f *FileManager) WriteFile(path string, data []byte) error {
	if f.verbose {
		fmt.Printf("Writing file: %s (%d bytes)\n", path, len(data))
	}

	// Ensure directory exists
	if err := f.EnsureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// #nosec G306 - Standard file permissions for configuration files
	return os.WriteFile(path, data, 0644)
}

// ReadFile reads data from a file.
func (f *FileManager) ReadFile(path string) ([]byte, error) {
	if f.verbose {
		fmt.Printf("Reading file: %s\n", path)
	}

	// #nosec G304 - File path comes from trusted application code
	return os.ReadFile(path)
}

// RemoveFile removes a file.
func (f *FileManager) RemoveFile(path string) error {
	if f.verbose {
		fmt.Printf("Removing file: %s\n", path)
	}

	return os.Remove(path)
}

// MockFileManager implements the FileManager port for testing.
type MockFileManager struct {
	files   map[string][]byte // path -> content
	verbose bool
}

// NewMockFileManager creates a new mock file manager for testing.
func NewMockFileManager(verbose bool) *MockFileManager {
	return &MockFileManager{
		files:   make(map[string][]byte),
		verbose: verbose,
	}
}

// SetMockFile sets the content of a mock file.
func (f *MockFileManager) SetMockFile(path string, content []byte) {
	f.files[path] = content
}

// FileExists checks if a mock file exists.
func (f *MockFileManager) FileExists(path string) bool {
	_, exists := f.files[path]

	return exists
}

// EnsureDir does nothing in mock mode.
func (f *MockFileManager) EnsureDir(path string) error {
	if f.verbose {
		fmt.Printf("MOCK: Ensuring directory: %s\n", path)
	}

	return nil
}

// CopyFile copies between mock files.
func (f *MockFileManager) CopyFile(src, dest string) error {
	if f.verbose {
		fmt.Printf("MOCK: Copying %s -> %s\n", src, dest)
	}

	content, exists := f.files[src]
	if !exists {
		return domain.ErrMockFileNotFound
	}

	f.files[dest] = content

	return nil
}

// WriteFile writes to a mock file.
func (f *MockFileManager) WriteFile(path string, data []byte) error {
	if f.verbose {
		fmt.Printf("MOCK: Writing file %s (%d bytes)\n", path, len(data))
	}

	f.files[path] = data

	return nil
}

// ReadFile reads from a mock file.
func (f *MockFileManager) ReadFile(path string) ([]byte, error) {
	if f.verbose {
		fmt.Printf("MOCK: Reading file %s\n", path)
	}

	content, exists := f.files[path]
	if !exists {
		return nil, domain.ErrMockFileNotFound
	}

	return content, nil
}

// RemoveFile removes a mock file.
func (f *MockFileManager) RemoveFile(path string) error {
	if f.verbose {
		fmt.Printf("MOCK: Removing file %s\n", path)
	}

	delete(f.files, path)

	return nil
}
