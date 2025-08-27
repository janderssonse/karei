// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// CopyFile copies a file with automatic directory creation
// Consolidates logic repeated 5+ times across files.
func CopyFile(src, dst string) error {
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	srcData, err := os.ReadFile(src) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	return os.WriteFile(dst, srcData, 0644) //nolint:gosec
}

// EnsureDir creates directory with parents if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755) //nolint:gosec
}

// SafeWriteFile writes file with automatic directory creation.
func SafeWriteFile(path string, data []byte) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644) //nolint:gosec
}

// FileExists checks if file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

// IsDir checks if path is a directory.
func IsDir(path string) bool {
	if stat, err := os.Stat(path); err == nil {
		return stat.IsDir()
	}

	return false
}
