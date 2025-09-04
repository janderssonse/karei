// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileExists(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp(t.TempDir(), "test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file returns true",
			path:     tmpFile.Name(),
			expected: true,
		},
		{
			name:     "non-existing file returns false",
			path:     "/non/existent/file",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FileExists(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDir(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()

	// Create a temp file
	tmpFile, err := os.CreateTemp(t.TempDir(), "testfile")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "directory returns true",
			path:     tmpDir,
			expected: true,
		},
		{
			name:     "file returns false",
			path:     tmpFile.Name(),
			expected: false,
		},
		{
			name:     "non-existent path returns false",
			path:     "/non/existent/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDir(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureDir(t *testing.T) {
	tmpBase := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "creates new directory",
			path:    filepath.Join(tmpBase, "newdir"),
			wantErr: false,
		},
		{
			name:    "creates nested directories",
			path:    filepath.Join(tmpBase, "deep", "nested", "dir"),
			wantErr: false,
		},
		{
			name:    "existing directory succeeds",
			path:    tmpBase,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureDir(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, IsDir(tt.path))
			}
		})
	}
}

func TestSafeWriteFile(t *testing.T) {
	tmpBase := t.TempDir()

	tests := []struct {
		name    string
		path    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "writes file in existing directory",
			path:    filepath.Join(tmpBase, "file.txt"),
			data:    []byte("test content"),
			wantErr: false,
		},
		{
			name:    "creates directory and writes file",
			path:    filepath.Join(tmpBase, "subdir", "file.txt"),
			data:    []byte("nested content"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SafeWriteFile(tt.path, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				// Verify file was written correctly
				content, err := os.ReadFile(tt.path)
				require.NoError(t, err)
				assert.Equal(t, tt.data, content)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tmpBase := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpBase, "source.txt")
	srcContent := []byte("source content")
	err := os.WriteFile(srcPath, srcContent, 0600)
	require.NoError(t, err)

	tests := []struct {
		name    string
		src     string
		dst     string
		wantErr bool
	}{
		{
			name:    "copies file to existing directory",
			src:     srcPath,
			dst:     filepath.Join(tmpBase, "copy.txt"),
			wantErr: false,
		},
		{
			name:    "copies file creating new directory",
			src:     srcPath,
			dst:     filepath.Join(tmpBase, "newdir", "copy.txt"),
			wantErr: false,
		},
		{
			name:    "fails with non-existent source",
			src:     "/non/existent/file",
			dst:     filepath.Join(tmpBase, "fail.txt"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CopyFile(tt.src, tt.dst)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				// Verify file was copied correctly
				dstContent, err := os.ReadFile(tt.dst)
				require.NoError(t, err)
				assert.Equal(t, srcContent, dstContent)
			}
		})
	}
}
