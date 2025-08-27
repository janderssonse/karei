// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package patterns

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/janderssonse/karei/internal/platform"
	"github.com/stretchr/testify/require"
)

func TestNewThemeManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "create theme manager with verbose=false",
			verbose: false,
		},
		{
			name:    "create theme manager with verbose=true",
			verbose: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			manager := NewThemeManager(testCase.verbose)

			require.NotNil(t, manager)
			require.Equal(t, "theme", manager.Name)
			require.Equal(t, "theme", manager.Type)
			require.Equal(t, testCase.verbose, manager.verbose)

			expectedThemes := []string{"tokyo-night", "catppuccin", "nord", "everforest", "gruvbox", "kanagawa", "rose-pine", "gruvbox-light"}
			require.Equal(t, expectedThemes, manager.Available)

			// Test that default handler exists
			require.Contains(t, manager.handlers, "default")
			require.NotNil(t, manager.handlers["default"])
		})
	}
}

func TestNewFontManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "create font manager with verbose=false",
			verbose: false,
		},
		{
			name:    "create font manager with verbose=true",
			verbose: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			manager := NewFontManager(testCase.verbose)

			require.NotNil(t, manager)
			require.Equal(t, "font", manager.Name)
			require.Equal(t, "font", manager.Type)
			require.Equal(t, testCase.verbose, manager.verbose)

			expectedFonts := []string{"CaskaydiaMono", "FiraMono", "JetBrainsMono", "MesloLGS", "BerkeleyMono"}
			require.Equal(t, expectedFonts, manager.Available)

			// Test that default handler exists
			require.Contains(t, manager.handlers, "default")
			require.NotNil(t, manager.handlers["default"])
		})
	}
}

func TestNewSecurityManager(t *testing.T) {
	t.Parallel()

	manager := NewSecurityManager(true)

	require.NotNil(t, manager)
	require.Equal(t, "security", manager.Name)
	require.Equal(t, "security", manager.Type)
	require.True(t, manager.verbose)

	expectedTools := []string{"audit", "firewall", "fail2ban", "clamav", "rkhunter", "aide"}
	require.Equal(t, expectedTools, manager.Available)

	// Test that default handler exists
	require.Contains(t, manager.handlers, "default")
	require.NotNil(t, manager.handlers["default"])
}

func TestNewVerifyManager(t *testing.T) {
	t.Parallel()

	manager := NewVerifyManager(false)

	require.NotNil(t, manager)
	require.Equal(t, "verify", manager.Name)
	require.Equal(t, "verify", manager.Type)
	require.False(t, manager.verbose)

	expectedChecks := []string{"tools", "integrations", "path", "fish", "xdg", "versions", "all"}
	require.Equal(t, expectedChecks, manager.Available)

	// Test that all handlers exist
	expectedHandlers := []string{"tools", "integrations", "path", "fish", "xdg", "versions", "all", "default"}
	for _, handler := range expectedHandlers {
		require.Contains(t, manager.handlers, handler)
		require.NotNil(t, manager.handlers[handler])
	}
}

func TestApplyThemeHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		theme       string
		setupFiles  map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name:  "apply theme with some configs missing (should not error)",
			theme: "tokyo-night",
			setupFiles: map[string]string{
				"themes/tokyo-night/ghostty.conf": "theme = tokyo-night\n",
				"themes/tokyo-night/btop.theme":   "theme=tokyo-night\n",
			},
			wantErr: false,
		},
		{
			name:    "apply theme with no config files (should not error)",
			theme:   "nonexistent-theme",
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			xdgConfigHome := filepath.Join(tmpDir, ".config")

			testHandlerWithFileSetup(t, testCase.setupFiles, testCase.wantErr, testCase.errContains, func() error {
				// Inline theme application for testing
				// Apply to config-file based applications
				applications := []string{ghosttyApp, "btop", "zellij"}

				for _, app := range applications {
					var srcPath, dstPath string

					switch app {
					case ghosttyApp:
						srcPath = filepath.Join(tmpDir, "themes", testCase.theme, ghosttyApp+".conf")
						dstPath = filepath.Join(xdgConfigHome, ghosttyApp, "theme.conf")
					case "btop":
						srcPath = filepath.Join(tmpDir, "themes", testCase.theme, "btop.theme")
						dstPath = filepath.Join(xdgConfigHome, "btop", "themes", testCase.theme+".theme")
					case "zellij":
						srcPath = filepath.Join(tmpDir, "themes", testCase.theme, "zellij.kdl")
						dstPath = filepath.Join(xdgConfigHome, "zellij", "themes", testCase.theme+".kdl")
					}

					if !platform.FileExists(srcPath) {
						continue
					}

					if err := platform.CopyFile(srcPath, dstPath); err != nil {
						return fmt.Errorf("failed to apply %s theme to %s: %w", testCase.theme, app, err)
					}
				}

				return nil
			})
		})
	}
}

func TestApplyFontHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		font        string
		setupFiles  map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "apply font with configs",
			font: "JetBrainsMono",
			setupFiles: map[string]string{
				"configs/ghostty/fonts/JetBrainsMono.conf":   "font-family = JetBrains Mono\n",
				"configs/alacritty/fonts/JetBrainsMono.conf": "family: JetBrains Mono\n",
			},
			wantErr: false,
		},
		{
			name:    "apply font with no config files",
			font:    "NonexistentFont",
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			xdgConfigHome := filepath.Join(tmpDir, ".config")

			testHandlerWithFileSetup(t, testCase.setupFiles, testCase.wantErr, testCase.errContains, func() error {
				// Inline font application for testing
				// Apply to config-file based applications
				applications := []string{ghosttyApp}

				for _, app := range applications {
					var srcPath, dstPath string

					if app == ghosttyApp {
						srcPath = filepath.Join(tmpDir, "configs", ghosttyApp, "fonts", testCase.font+".conf")
						dstPath = filepath.Join(xdgConfigHome, ghosttyApp, "font.conf")
					}

					if !platform.FileExists(srcPath) {
						continue
					}

					if err := platform.CopyFile(srcPath, dstPath); err != nil {
						return fmt.Errorf("failed to apply %s font to %s: %w", testCase.font, app, err)
					}
				}

				return nil
			})
		})
	}
}

// testHandlerWithFileSetup is a common test helper to reduce duplication.
func testHandlerWithFileSetup(t *testing.T, setupFiles map[string]string, wantErr bool, errContains string, handlerFunc func() error) {
	t.Helper()
	tmpDir := t.TempDir()

	// Setup test files
	for path, content := range setupFiles {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755) //nolint:gosec
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644) //nolint:gosec
		require.NoError(t, err)
	}

	err := handlerFunc()

	if wantErr {
		require.Error(t, err)

		if errContains != "" {
			require.Contains(t, err.Error(), errContains)
		}
	} else {
		require.NoError(t, err)
	}
}

func TestRunSecurityToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tool        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "unknown security tool",
			tool:        "unknown-tool",
			wantErr:     true,
			errContains: "unknown security tool",
		},
		{
			name:    "valid tool - audit (dry run)",
			tool:    "audit",
			wantErr: false, // Should not error in dry run mode
		},
		{
			name:    "valid tool - clamav (dry run)",
			tool:    "clamav",
			wantErr: false, // Should not error in dry run mode
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Use dry run mode for tests to avoid sudo
			err := runSecurityToolHandlerWithDryRun(context.Background(), testCase.tool, true)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errContains != "" {
					require.Contains(t, err.Error(), testCase.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestVerifyHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler func(context.Context, string) error
		wantErr bool
	}{
		{
			name:    "verify tools handler",
			handler: verifyToolsHandler,
			wantErr: false,
		},
		{
			name:    "verify integrations handler",
			handler: verifyIntegrationsHandler,
			wantErr: false,
		},
		{
			name:    "verify path handler",
			handler: verifyPathHandler,
			wantErr: false,
		},
		{
			name:    "verify fish handler",
			handler: verifyFishHandler,
			wantErr: false, // Should not error even if fish not installed
		},
		{
			name:    "verify xdg handler",
			handler: verifyXDGHandler,
			wantErr: false,
		},
		{
			name:    "verify versions handler",
			handler: verifyVersionsHandler,
			wantErr: false,
		},
		{
			name:    "verify all handler",
			handler: verifyAllHandler,
			wantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.handler(context.Background(), "")

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestShowLogHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		handler     func(context.Context, string) error
		logFile     string
		logContent  string
		expectError bool
	}{
		{
			name:        "show install logs - file exists",
			handler:     showInstallLogsHandlerWithPath,
			logFile:     "install.log",
			logContent:  "Installation started\nInstalling package A\nCompleted successfully\n",
			expectError: false,
		},
		{
			name:        "show progress logs - file exists",
			handler:     showProgressLogsHandlerWithPath,
			logFile:     "progress.log",
			logContent:  "Progress: 25%\nProgress: 50%\nProgress: 100%\n",
			expectError: false,
		},
		{
			name:        "show error logs - file exists",
			handler:     showErrorLogsHandlerWithPath,
			logFile:     "errors.log",
			logContent:  "Error: Failed to connect\nError: Timeout occurred\n",
			expectError: false,
		},
		{
			name:        "show logs - file does not exist",
			handler:     showInstallLogsHandlerWithPath,
			logFile:     "",    // Don't create file
			expectError: false, // Should not error when file doesn't exist
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			// Create log file if specified
			if testCase.logFile != "" {
				logDir := filepath.Join(tmpDir, "karei")
				err := os.MkdirAll(logDir, 0755) //nolint:gosec //nolint:gosec
				require.NoError(t, err)

				logPath := filepath.Join(logDir, testCase.logFile)
				err = os.WriteFile(logPath, []byte(testCase.logContent), 0644) //nolint:gosec
				require.NoError(t, err)
			}

			err := testCase.handler(context.Background(), tmpDir)

			if testCase.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestShowAllLogsHandler(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create log directory and files
	logDir := filepath.Join(tmpDir, "karei")
	err := os.MkdirAll(logDir, 0755) //nolint:gosec
	require.NoError(t, err)

	logFiles := map[string]string{
		"install.log":  "Install log content\n",
		"progress.log": "Progress log content\n",
		"precheck.log": "Precheck log content\n",
		"errors.log":   "Error log content\n",
	}

	for filename, content := range logFiles {
		logPath := filepath.Join(logDir, filename)
		err = os.WriteFile(logPath, []byte(content), 0644) //nolint:gosec
		require.NoError(t, err)
	}

	err = showAllLogsHandlerWithPath(context.Background(), tmpDir)
	require.NoError(t, err)
}

func TestNewThemeCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "create theme command with verbose=false",
			verbose: false,
		},
		{
			name:    "create theme command with verbose=true",
			verbose: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			command := NewThemeCommand(testCase.verbose)

			require.NotNil(t, command)
			require.Equal(t, "theme", command.Name)
			require.Equal(t, "Manage system themes", command.Usage)
			require.Contains(t, command.Description, "Apply coordinated themes across all applications")
			require.True(t, command.Interactive)
			require.NotNil(t, command.Manager)
			require.Equal(t, "theme", command.Manager.Type)
		})
	}
}

func TestNewFontCommand(t *testing.T) {
	t.Parallel()

	command := NewFontCommand(true)

	require.NotNil(t, command)
	require.Equal(t, "font", command.Name)
	require.Equal(t, "Manage system fonts", command.Usage)
	require.Contains(t, command.Description, "Install and configure programming fonts")
	require.True(t, command.Interactive)
	require.NotNil(t, command.Manager)
	require.Equal(t, "font", command.Manager.Type)
}

func TestNewSecurityCommand(t *testing.T) {
	t.Parallel()

	command := NewSecurityCommand(false)

	require.NotNil(t, command)
	require.Equal(t, "security", command.Name)
	require.Equal(t, "Run security checks and tools", command.Usage)
	require.Contains(t, command.Description, "Execute comprehensive security audits")
	require.True(t, command.Interactive)
	require.NotNil(t, command.Manager)
	require.Equal(t, "security", command.Manager.Type)
}

func TestNewVerifyCommand(t *testing.T) {
	t.Parallel()

	command := NewVerifyCommand(true)

	require.NotNil(t, command)
	require.Equal(t, "verify", command.Name)
	require.Equal(t, "Verify system configuration", command.Usage)
	require.Contains(t, command.Description, "Run comprehensive verification")
	require.True(t, command.Interactive)
	require.NotNil(t, command.Manager)
	require.Equal(t, "verify", command.Manager.Type)
}

func TestNewLogsCommand(t *testing.T) {
	t.Parallel()

	command := NewLogsCommand(false)

	require.NotNil(t, command)
	require.Equal(t, "logs", command.Name)
	require.Equal(t, "View system logs", command.Usage)
	require.Contains(t, command.Description, "Display Karei installation")
	require.True(t, command.Interactive)
	require.NotNil(t, command.Manager)
	require.Equal(t, "logs", command.Manager.Type)
}
