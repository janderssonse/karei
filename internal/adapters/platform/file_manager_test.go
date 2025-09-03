// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janderssonse/karei/internal/adapters/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileManager_FileExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fm := platform.NewFileManager(false)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0600))

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{"existing file", testFile, true},
		{"non-existing file", filepath.Join(tmpDir, "nonexistent.txt"), false},
		{"directory", tmpDir, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, fm.FileExists(tt.path))
		})
	}
}

func TestFileManager_EnsureDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fm := platform.NewFileManager(false)

	// Test creating nested directories
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	require.NoError(t, fm.EnsureDir(nestedDir))
	assert.DirExists(t, nestedDir)

	// Test idempotency - should not error on existing dir
	require.NoError(t, fm.EnsureDir(nestedDir))
}

func TestFileManager_CopyFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fm := platform.NewFileManager(false)

	// Create source file
	srcFile := filepath.Join(tmpDir, "source.txt")
	srcContent := []byte("test content")
	require.NoError(t, os.WriteFile(srcFile, srcContent, 0600))

	// Copy to new location
	dstFile := filepath.Join(tmpDir, "subdir", "dest.txt")
	require.NoError(t, fm.CopyFile(srcFile, dstFile))

	// Verify content
	dstContent, err := os.ReadFile(filepath.Clean(dstFile))
	require.NoError(t, err)
	assert.Equal(t, srcContent, dstContent)

	// Test error on non-existing source
	err = fm.CopyFile(filepath.Join(tmpDir, "nonexistent"), dstFile)
	assert.Error(t, err)
}

func TestFileManager_WriteAndReadFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fm := platform.NewFileManager(false)

	// Test write and read
	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("hello world")

	require.NoError(t, fm.WriteFile(testFile, testData))

	readData, err := fm.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, testData, readData)

	// Test read non-existing file
	_, err = fm.ReadFile(filepath.Join(tmpDir, "nonexistent"))
	assert.Error(t, err)
}

func TestFileManager_RemoveFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fm := platform.NewFileManager(false)

	// Create and remove file
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0600))

	require.NoError(t, fm.RemoveFile(testFile))
	assert.NoFileExists(t, testFile)

	// Remove non-existing file returns error
	err := fm.RemoveFile(testFile)
	assert.Error(t, err)
}
