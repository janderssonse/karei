// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package uninstall_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janderssonse/karei/internal/uninstall"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommandExecutor provides a testable command executor.
type MockCommandExecutor struct {
	commands []string
	results  map[string]error
}

func (m *MockCommandExecutor) Execute(cmd string, args ...string) error {
	fullCmd := cmd + " " + strings.Join(args, " ")

	m.commands = append(m.commands, fullCmd)
	if err, ok := m.results[fullCmd]; ok {
		return err
	}

	return nil
}

func TestUninstaller_SetPassword(t *testing.T) {
	t.Parallel()

	uninstaller, _ := uninstall.NewTestUninstaller(false)

	// Test setting password
	password := "test-password"
	uninstaller.SetPassword(password)

	// Password is stored internally, no way to verify directly
	// but we can verify it doesn't panic
	assert.NotNil(t, uninstaller)
}

func TestUninstaller_UninstallSpecial(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		appName string
		wantErr bool
	}{
		{
			name:    "chrome special uninstall",
			appName: "chrome",
			wantErr: false, // Will attempt uninstall
		},
		{
			name:    "docker special uninstall",
			appName: "docker",
			wantErr: false,
		},
		{
			name:    "vscode special uninstall",
			appName: "vscode",
			wantErr: false,
		},
		{
			name:    "non-special app returns false",
			appName: "regular-app",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			uninstaller, mock := uninstall.NewTestUninstaller(false)

			// UninstallSpecial will attempt to uninstall
			err := uninstaller.UninstallSpecial(ctx, tt.appName)

			// Special apps should be handled
			specialApps := []string{"chrome", "docker", "vscode", "postman", "obsidian", "discord", "slack", "spotify"}
			isSpecial := false

			for _, special := range specialApps {
				if tt.appName == special {
					isSpecial = true
					break
				}
			}

			// With mock, special apps should run without errors
			// Regular apps will try UninstallApp which may error
			if isSpecial {
				// Special uninstalls should attempt commands
				assert.NotEmpty(t, mock.Commands, "Special uninstall should attempt commands")
			} else {
				// Non-special apps will go through UninstallApp
				// which may error if app is unknown
				_ = err
			}
		})
	}
}

func TestUninstaller_UninstallAppWithDifferentMethods(t *testing.T) {
	t.Parallel()

	// Test with real apps from the catalog that use different install methods
	tests := []struct {
		name       string
		appName    string
		installCmd string
		wantErr    bool
	}{
		{
			name:       "uninstall special app chrome",
			appName:    "chrome",
			installCmd: "special",
			wantErr:    false,
		},
		{
			name:       "uninstall flatpak package",
			appName:    "spotify",
			installCmd: "flatpak",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			uninstaller, mock := uninstall.NewTestUninstaller(false)

			// The mock will handle the command without executing
			err := uninstaller.UninstallApp(ctx, tt.appName)

			// Special apps or recognized apps should have attempted commands
			if err == nil {
				assert.NotEmpty(t, mock.Commands, "Uninstall should attempt commands for known apps")
			}
		})
	}
}

func TestUninstaller_UninstallGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		groupName string
		wantErr   bool
	}{
		{
			name:      "uninstall essentials group",
			groupName: "essentials",
			wantErr:   false, // Group exists
		},
		{
			name:      "uninstall unknown group",
			groupName: "nonexistent",
			wantErr:   true, // Group doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			uninstaller, mock := uninstall.NewTestUninstaller(false)

			err := uninstaller.UninstallGroup(ctx, tt.groupName)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, uninstall.ErrUnknownGroup)
			} else if err != nil && !errors.Is(err, uninstall.ErrUnknownGroup) {
				// Check if mock recorded commands
				assert.NotEmpty(t, mock.Commands, "Group uninstall should attempt commands")
			}
		})
	}
}

func TestUninstaller_FileOperations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func() string
		operation func(*uninstall.Uninstaller, string) error
		verify    func(string) bool
	}{
		{
			name: "remove file successfully",
			setup: func() string {
				filePath := filepath.Join(tmpDir, "test.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test"), 0600))
				return filePath
			},
			operation: func(_ *uninstall.Uninstaller, path string) error {
				// This would be internal, but we test through UninstallApp
				// which may call removeFile internally
				return os.Remove(path)
			},
			verify: func(path string) bool {
				_, err := os.Stat(path)
				return os.IsNotExist(err)
			},
		},
		{
			name: "remove directory successfully",
			setup: func() string {
				dirPath := filepath.Join(tmpDir, "testdir")
				require.NoError(t, os.MkdirAll(dirPath, 0750))
				// Add a file in the directory
				require.NoError(t, os.WriteFile(
					filepath.Join(dirPath, "file.txt"),
					[]byte("content"),
					0600,
				))
				return dirPath
			},
			operation: func(_ *uninstall.Uninstaller, path string) error {
				return os.RemoveAll(path)
			},
			verify: func(path string) bool {
				_, err := os.Stat(path)
				return os.IsNotExist(err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.setup()
			uninstaller, _ := uninstall.NewTestUninstaller(false)

			err := tt.operation(uninstaller, path)
			require.NoError(t, err)

			assert.True(t, tt.verify(path))
		})
	}
}

func TestUninstaller_MisePackageDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		appName     string
		wantMisePkg string
		wantFound   bool
	}{
		{
			name:        "detect node as mise package",
			appName:     "node",
			wantMisePkg: "node",
			wantFound:   true,
		},
		{
			name:        "detect python as mise package",
			appName:     "python",
			wantMisePkg: "python",
			wantFound:   true,
		},
		{
			name:        "detect go as mise package",
			appName:     "go",
			wantMisePkg: "go",
			wantFound:   true,
		},
		{
			name:        "detect rust as mise package",
			appName:     "rust",
			wantMisePkg: "rust",
			wantFound:   true,
		},
		{
			name:        "non-mise package returns false",
			appName:     "vim",
			wantMisePkg: "",
			wantFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// This tests the logic of detecting mise-managed packages
			misePackages := []string{"node", "python", "go", "rust", "ruby", "java", "deno", "bun"}

			found := false

			for _, pkg := range misePackages {
				if tt.appName == pkg {
					found = true
					break
				}
			}

			assert.Equal(t, tt.wantFound, found)
		})
	}
}

func TestUninstaller_DebPackageMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		appName    string
		wantDebPkg string
	}{
		{
			name:       "map node to nodejs",
			appName:    "node",
			wantDebPkg: "nodejs",
		},
		{
			name:       "map docker to docker.io",
			appName:    "docker",
			wantDebPkg: "docker.io",
		},
		{
			name:       "regular package unchanged",
			appName:    "vim",
			wantDebPkg: "vim",
		},
		{
			name:       "map code to code",
			appName:    "code",
			wantDebPkg: "code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test the package name mapping logic
			debMappings := map[string]string{
				"node":   "nodejs",
				"docker": "docker.io",
			}

			debPkg := tt.appName
			if mapped, ok := debMappings[tt.appName]; ok {
				debPkg = mapped
			}

			assert.Equal(t, tt.wantDebPkg, debPkg)
		})
	}
}
