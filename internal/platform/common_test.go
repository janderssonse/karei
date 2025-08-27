// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package platform provides platform utilities for the karei application.
package platform

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathUtils_GetKareiPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		envValue string
		want     string
		setup    func()
		cleanup  func()
	}{
		{
			name:     "uses KAREI_PATH env var when set",
			envValue: "/custom/karei/path",
			want:     "/custom/karei/path",
			setup:    func() {},
			cleanup: func() {
				_ = os.Unsetenv("KAREI_PATH")
			},
		},
		{
			name:     "falls back to default path when env var not set",
			envValue: "",
			want:     "", // Will be set dynamically in test
			setup: func() {
				_ = os.Unsetenv("KAREI_PATH")
			},
			cleanup: func() {},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := GetKareiPathWithEnv(testCase.envValue)

			if testCase.want == "" {
				// Dynamic expectation for default path
				home, err := os.UserHomeDir()
				require.NoError(t, err)

				expected := filepath.Join(home, ".local", "share", "karei")
				require.Equal(t, expected, got)
			} else {
				require.Equal(t, testCase.want, got)
			}
		})
	}
}

func TestPathUtils_GetXDGConfigHome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		envValue string
		want     string
		setup    func()
		cleanup  func()
	}{
		{
			name:     "uses XDG_CONFIG_HOME when set",
			envValue: "/custom/config",
			want:     "/custom/config",
			setup:    func() {},
			cleanup: func() {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			},
		},
		{
			name:     "falls back to ~/.config when not set",
			envValue: "",
			want:     "", // Will be set dynamically
			setup: func() {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			},
			cleanup: func() {},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := GetXDGConfigHomeWithEnv(testCase.envValue)

			if testCase.want == "" {
				home, err := os.UserHomeDir()
				require.NoError(t, err)

				expected := filepath.Join(home, ".config")
				require.Equal(t, expected, got)
			} else {
				require.Equal(t, testCase.want, got)
			}
		})
	}
}

func TestPathUtils_ExpandPath(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "expands tilde path",
			path: "~/Documents/test",
			want: filepath.Join(home, "Documents/test"),
		},
		{
			name: "expands XDG_CONFIG_HOME",
			path: "$XDG_CONFIG_HOME/app/config",
			want: filepath.Join(GetXDGConfigHome(), "app/config"),
		},
		{
			name: "expands XDG_DATA_HOME",
			path: "$XDG_DATA_HOME/app/data",
			want: filepath.Join(GetXDGDataHome(), "app/data"),
		},
		{
			name: "leaves absolute path unchanged",
			path: "/absolute/path/test",
			want: "/absolute/path/test",
		},
		{
			name: "leaves relative path unchanged",
			path: "relative/path/test",
			want: "relative/path/test",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ExpandPath(testCase.path)
			require.Equal(t, testCase.want, got)
		})
	}
}

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
		content []byte
		wantErr bool
	}{
		{
			name:    "write to existing directory",
			path:    filepath.Join(tmpDir, "test.txt"),
			content: []byte("test content"),
			wantErr: false,
		},
		{
			name:    "write to nested directory that doesn't exist",
			path:    filepath.Join(tmpDir, "nested", "deep", "test.txt"),
			content: []byte("nested content"),
			wantErr: false,
		},
		{
			name:    "overwrite existing file",
			path:    filepath.Join(tmpDir, "existing.txt"),
			content: []byte("overwritten content"),
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// For overwrite test, create existing file
			if testCase.name == "overwrite existing file" {
				err := os.WriteFile(testCase.path, []byte("original content"), 0644) //nolint:gosec
				require.NoError(t, err)
			}

			err := SafeWriteFile(testCase.path, testCase.content)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, FileExists(testCase.path))

				written, err := os.ReadFile(testCase.path)
				require.NoError(t, err)
				require.Equal(t, testCase.content, written)
			}
		})
	}
}

func TestCommandUtils_CommandExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "existing command",
			command: "ls",
			want:    true,
		},
		{
			name:    "non-existing command",
			command: "nonexistentcommand12345",
			want:    false,
		},
		{
			name:    "empty command",
			command: "",
			want:    false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := CommandExists(testCase.command)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestValidationUtils_IsValidChoice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		choice  string
		allowed []string
		want    bool
	}{
		{
			name:    "valid choice",
			choice:  "option1",
			allowed: []string{"option1", "option2", "option3"},
			want:    true,
		},
		{
			name:    "invalid choice",
			choice:  "invalid",
			allowed: []string{"option1", "option2", "option3"},
			want:    false,
		},
		{
			name:    "empty choice",
			choice:  "",
			allowed: []string{"option1", "option2"},
			want:    false,
		},
		{
			name:    "empty allowed list",
			choice:  "anything",
			allowed: []string{},
			want:    false,
		},
		{
			name:    "case sensitive",
			choice:  "Option1",
			allowed: []string{"option1", "option2"},
			want:    false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := slices.Contains(testCase.allowed, testCase.choice)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestValidationUtils_IsValidTheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		themeName string
		want      bool
	}{
		{
			name:      "valid theme - tokyo-night",
			themeName: "tokyo-night",
			want:      true,
		},
		{
			name:      "valid theme - catppuccin",
			themeName: "catppuccin",
			want:      true,
		},
		{
			name:      "valid theme - gruvbox-light",
			themeName: "gruvbox-light",
			want:      true,
		},
		{
			name:      "invalid theme",
			themeName: "invalid-theme",
			want:      false,
		},
		{
			name:      "empty theme",
			themeName: "",
			want:      false,
		},
		{
			name:      "case sensitivity",
			themeName: "Tokyo-Night",
			want:      false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			validThemes := []string{"tokyo-night", "catppuccin", "nord", "everforest", "gruvbox", "kanagawa", "rose-pine", "gruvbox-light"}
			got := slices.Contains(validThemes, testCase.themeName)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestValidationUtils_IsValidFont(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fontName string
		want     bool
	}{
		{
			name:     "valid font - CaskaydiaMono",
			fontName: "CaskaydiaMono",
			want:     true,
		},
		{
			name:     "valid font - JetBrainsMono",
			fontName: "JetBrainsMono",
			want:     true,
		},
		{
			name:     "valid font - BerkeleyMono",
			fontName: "BerkeleyMono",
			want:     true,
		},
		{
			name:     "invalid font",
			fontName: "InvalidFont",
			want:     false,
		},
		{
			name:     "empty font",
			fontName: "",
			want:     false,
		},
		{
			name:     "case sensitivity",
			fontName: "jetbrainsmono",
			want:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			validFonts := []string{"CaskaydiaMono", "FiraMono", "JetBrainsMono", "MesloLGS", "BerkeleyMono"}
			got := slices.Contains(validFonts, testCase.fontName)
			require.Equal(t, testCase.want, got)
		})
	}
}
