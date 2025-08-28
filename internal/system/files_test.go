// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileUtils_CopyFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		srcContent  string
		srcPath     string
		dstPath     string
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful file copy",
			srcContent: "test content",
			srcPath:    filepath.Join(tmpDir, "source.txt"),
			dstPath:    filepath.Join(tmpDir, "dest.txt"),
			wantErr:    false,
		},
		{
			name:       "copy to nested directory",
			srcContent: "nested test",
			srcPath:    filepath.Join(tmpDir, "source2.txt"),
			dstPath:    filepath.Join(tmpDir, "nested", "dir", "dest.txt"),
			wantErr:    false,
		},
		{
			name:        "source file does not exist",
			srcContent:  "",
			srcPath:     filepath.Join(tmpDir, "nonexistent.txt"),
			dstPath:     filepath.Join(tmpDir, "dest2.txt"),
			wantErr:     true,
			errContains: "failed to read source",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// Create source file if content provided
			if testCase.srcContent != "" {
				err := os.WriteFile(testCase.srcPath, []byte(testCase.srcContent), 0644) //nolint:gosec
				require.NoError(t, err)
			}

			err := CopyFile(testCase.srcPath, testCase.dstPath)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errContains != "" {
					require.Contains(t, err.Error(), testCase.errContains)
				}
			} else {
				require.NoError(t, err)

				// Verify file was copied correctly
				require.True(t, FileExists(testCase.dstPath))

				dstContent, err := os.ReadFile(testCase.dstPath)
				require.NoError(t, err)
				require.Equal(t, testCase.srcContent, string(dstContent))
			}
		})
	}
}

func TestFileUtils_EnsureDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "create single directory",
			path:    filepath.Join(tmpDir, "testdir"),
			wantErr: false,
		},
		{
			name:    "create nested directories",
			path:    filepath.Join(tmpDir, "nested", "deep", "directory"),
			wantErr: false,
		},
		{
			name:    "directory already exists",
			path:    tmpDir, // tmpDir already exists
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := EnsureDir(testCase.path)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Verify directory exists
				require.True(t, IsDir(testCase.path))
			}
		})
	}
}

func TestFileUtils_SafeWriteFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "write to new file",
			path:    filepath.Join(tmpDir, "newfile.txt"),
			data:    []byte("new content"),
			wantErr: false,
		},
		{
			name:    "overwrite existing file",
			path:    filepath.Join(tmpDir, "existing.txt"),
			data:    []byte("updated content"),
			wantErr: false,
		},
		{
			name:    "create file in nested directory",
			path:    filepath.Join(tmpDir, "deep", "nested", "file.txt"),
			data:    []byte("nested content"),
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// For overwrite test, create file first
			if testCase.name == "overwrite existing file" {
				err := os.WriteFile(testCase.path, []byte("old content"), 0644) //nolint:gosec
				require.NoError(t, err)
			}

			err := SafeWriteFile(testCase.path, testCase.data)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify file was written correctly
				require.True(t, FileExists(testCase.path))

				content, err := os.ReadFile(testCase.path)
				require.NoError(t, err)
				require.Equal(t, testCase.data, content)

				// Check permissions
				stat, err := os.Stat(testCase.path)
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0644), stat.Mode().Perm())
			}
		})
	}
}
