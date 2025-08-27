// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janderssonse/karei/internal/cli"
	"github.com/janderssonse/karei/internal/patterns"
	"github.com/stretchr/testify/require"
)

// Integration tests that test the interaction between multiple components

func TestThemeManagement_Integration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Setup test environment
	setupTestEnvironment(t, tmpDir)

	tests := []struct {
		name            string
		theme           string
		expectSuccess   bool
		setupThemeFiles map[string]string
	}{
		{
			name:          "apply complete theme",
			theme:         "tokyo-night",
			expectSuccess: true,
			setupThemeFiles: map[string]string{
				"themes/tokyo-night/ghostty.conf": "theme = tokyo-night\nbackground = #1a1b26\n",
				"themes/tokyo-night/btop.theme":   "theme_background=\"#1a1b26\"\n",
				"themes/tokyo-night/zellij.kdl":   "theme \"tokyo-night\"\n",
			},
		},
		{
			name:          "apply theme with missing files",
			theme:         "nord",
			expectSuccess: true, // Should succeed even with missing files
			setupThemeFiles: map[string]string{
				"themes/nord/ghostty.conf": "theme = nord\nbackground = #2e3440\n",
				// Missing other theme files - should not cause failure
			},
		},
		{
			name:          "apply non-existent theme",
			theme:         "nonexistent",
			expectSuccess: false, // Should fail with proper validation
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// Setup theme files
			for path, content := range testCase.setupThemeFiles {
				fullPath := filepath.Join(tmpDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755) //nolint:gosec
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0644) //nolint:gosec
				require.NoError(t, err)
			}

			// Create theme manager with dry run enabled
			themeManager := patterns.NewThemeManagerWithDryRun(true, true)

			// Test validation
			if testCase.theme != "nonexistent" {
				require.True(t, themeManager.IsValid(testCase.theme))
			}

			// Apply theme
			err := themeManager.Apply(context.Background(), testCase.theme)

			if testCase.expectSuccess {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			// Check if config was saved
			if testCase.expectSuccess && testCase.theme != "nonexistent" {
				require.Equal(t, testCase.theme, themeManager.Current)
			}
		})
	}
}

func TestFontManagement_Integration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	setupTestEnvironment(t, tmpDir)

	tests := []struct {
		name           string
		font           string
		expectSuccess  bool
		setupFontFiles map[string]string
	}{
		{
			name:          "apply complete font",
			font:          "JetBrainsMono",
			expectSuccess: true,
			setupFontFiles: map[string]string{
				"configs/ghostty/fonts/JetBrainsMono.conf":   "font-family = JetBrains Mono\n",
				"configs/alacritty/fonts/JetBrainsMono.conf": "family: JetBrains Mono\n",
			},
		},
		{
			name:          "apply font with partial configs",
			font:          "FiraMono",
			expectSuccess: true,
			setupFontFiles: map[string]string{
				"configs/ghostty/fonts/FiraMono.conf": "font-family = Fira Mono\n",
				// Missing alacritty config - should not fail
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// Setup font files
			for path, content := range testCase.setupFontFiles {
				fullPath := filepath.Join(tmpDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755) //nolint:gosec
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0644) //nolint:gosec
				require.NoError(t, err)
			}

			// Create font manager with dry run enabled
			fontManager := patterns.NewFontManagerWithDryRun(true, true)

			// Test validation
			require.True(t, fontManager.IsValid(testCase.font))

			// Apply font
			err := fontManager.Apply(context.Background(), testCase.font)

			if testCase.expectSuccess {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestInstallationWorkflow_Integration - Simplified without installer dependency.
func TestInstallationWorkflow_Integration(t *testing.T) {
	t.Parallel()
	t.Skip("Installation workflow test simplified - installer unified")
}

func TestUniversalManager_ConfigPersistence_Integration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	setupTestEnvironment(t, tmpDir)

	// Create XDG config directory
	configDir := filepath.Join(tmpDir, ".config", "karei")
	err := os.MkdirAll(configDir, 0755) //nolint:gosec
	require.NoError(t, err)

	tests := []struct {
		name       string
		manager    *patterns.UniversalManager
		choice     string
		configFile string
	}{
		{
			name:       "theme manager config persistence",
			manager:    patterns.NewThemeManagerWithDryRun(false, true),
			choice:     "tokyo-night",
			configFile: "theme",
		},
		{
			name:       "font manager config persistence",
			manager:    patterns.NewFontManagerWithDryRun(false, true),
			choice:     "JetBrainsMono",
			configFile: "font",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// Override config path to use temp directory
			testCase.manager.ConfigPath = filepath.Join(configDir, testCase.configFile)

			// Save configuration
			err := testCase.manager.SaveCurrent(testCase.choice)
			require.NoError(t, err)

			// Verify file was created
			require.FileExists(t, testCase.manager.ConfigPath)

			// Verify content
			content, err := os.ReadFile(testCase.manager.ConfigPath)
			require.NoError(t, err)

			expectedContent := "KAREI_" + strings.ToUpper(testCase.manager.Type) + "=" + testCase.choice + "\n"
			require.Equal(t, expectedContent, string(content))

			// Create new manager and verify it detects the saved config
			var newManager *patterns.UniversalManager
			if testCase.manager.Type == "theme" {
				newManager = patterns.NewThemeManagerWithDryRun(false, true)
			} else {
				newManager = patterns.NewFontManagerWithDryRun(false, true)
			}

			newManager.ConfigPath = testCase.manager.ConfigPath

			current := newManager.GetCurrent()
			require.Equal(t, testCase.choice, current)
		})
	}
}

func TestCLIWorkflow_Integration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	setupTestEnvironment(t, tmpDir)

	// Create version file
	versionFile := filepath.Join(tmpDir, "version")
	err := os.WriteFile(versionFile, []byte("v1.0.0-test"), 0644) //nolint:gosec
	require.NoError(t, err)

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "CLI creation and basic operations",
			testFunc: func(t *testing.T) {
				t.Helper()
				cliApp := cli.NewCLI()
				require.NotNil(t, cliApp)
				// Simplified test - detailed testing moved to unit tests
			},
		},
		{
			name: "universal command integration",
			testFunc: func(t *testing.T) {
				t.Helper()
				// Test theme command
				themeCmd := patterns.NewThemeCommand(false)
				require.NotNil(t, themeCmd)
				require.Equal(t, "theme", themeCmd.Name)

				// Test font command
				fontCmd := patterns.NewFontCommand(false)
				require.NotNil(t, fontCmd)
				require.Equal(t, "font", fontCmd.Name)

				// Test command execution with list argument
				err := themeCmd.Execute(context.Background(), []string{"list"})
				require.NoError(t, err)

				err = fontCmd.Execute(context.Background(), []string{"list"})
				require.NoError(t, err)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.testFunc(t)
		})
	}
}

func TestEndToEndWorkflow_Integration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	setupTestEnvironment(t, tmpDir)

	// Setup complete theme structure
	themeFiles := map[string]string{
		"themes/tokyo-night/ghostty.conf": "theme = tokyo-night\n",
		"themes/tokyo-night/btop.theme":   "theme_background=\"#1a1b26\"\n",
		"themes/catppuccin/ghostty.conf":  "theme = catppuccin\n",
		"themes/catppuccin/btop.theme":    "theme_background=\"#1e1e2e\"\n",
	}

	fontFiles := map[string]string{
		"configs/ghostty/fonts/JetBrainsMono.conf": "font-family = JetBrains Mono\n",
		"configs/ghostty/fonts/FiraMono.conf":      "font-family = Fira Mono\n",
	}

	allFiles := make(map[string]string)
	for k, v := range themeFiles {
		allFiles[k] = v
	}

	for k, v := range fontFiles {
		allFiles[k] = v
	}

	// Setup all files
	for path, content := range allFiles {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755) //nolint:gosec
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644) //nolint:gosec
		require.NoError(t, err)
	}

	t.Run("complete theme and font workflow", func(t *testing.T) {
		t.Parallel()
		// 1. Create managers with dry run enabled
		themeManager := patterns.NewThemeManagerWithDryRun(true, true)
		fontManager := patterns.NewFontManagerWithDryRun(true, true)

		// 2. Verify available options
		themes := themeManager.GetAvailable()
		require.Contains(t, themes, "tokyo-night")
		require.Contains(t, themes, "catppuccin")

		fonts := fontManager.GetAvailable()
		require.Contains(t, fonts, "JetBrainsMono")
		require.Contains(t, fonts, "FiraMono")

		// 3. Apply theme
		err := themeManager.Apply(context.Background(), "tokyo-night")
		require.NoError(t, err)
		require.Equal(t, "tokyo-night", themeManager.GetCurrent())

		// 4. Apply font
		err = fontManager.Apply(context.Background(), "JetBrainsMono")
		require.NoError(t, err)
		require.Equal(t, "JetBrainsMono", fontManager.GetCurrent())

		// 5. Change theme
		err = themeManager.Apply(context.Background(), "catppuccin")
		require.NoError(t, err)
		require.Equal(t, "catppuccin", themeManager.GetCurrent())

		// 6. Verify status
		themeStatus := themeManager.Status()
		require.Equal(t, "theme", themeStatus["type"])
		require.Equal(t, "catppuccin", themeStatus["current"])

		fontStatus := fontManager.Status()
		require.Equal(t, "font", fontStatus["type"])
		require.Equal(t, "JetBrainsMono", fontStatus["current"])
	})

	// NOTE: Installer integration tests disabled pending unified installer refactoring
	// t.Run("installer integration with dry run", func(t *testing.T) {
	//	t.Parallel()
	//	unifiedInstaller := domain.NewUnifiedInstaller(true, true)
	//	// ... test implementation
	// })

	t.Run("command execution patterns", func(t *testing.T) {
		t.Parallel()

		executor := patterns.NewCommandExecutor(false, true) // dry run

		// Test basic execution
		err := executor.Execute(context.Background(), "echo", "test")
		require.NoError(t, err)

		// Test sudo execution
		err = executor.ExecuteSudo(context.Background(), "echo", "test")
		require.NoError(t, err)

		// Test output capture
		output, err := executor.ExecuteWithOutput(context.Background(), "echo", "hello")
		require.NoError(t, err)
		require.Empty(t, output) // dry run returns empty output

		// Test command existence
		require.True(t, executor.CommandExists("echo"))
		require.False(t, executor.CommandExists("nonexistentcommand12345"))
	})
}

func TestErrorHandling_Integration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	setupTestEnvironment(t, tmpDir)

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "invalid theme handling",
			testFunc: func(t *testing.T) {
				t.Helper()
				themeManager := patterns.NewThemeManagerWithDryRun(false, true)

				err := themeManager.Apply(context.Background(), "invalid-theme")
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid")
			},
		},
		{
			name: "invalid font handling",
			testFunc: func(t *testing.T) {
				t.Helper()
				fontManager := patterns.NewFontManagerWithDryRun(false, true)

				err := fontManager.Apply(context.Background(), "InvalidFont")
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid")
			},
		},
		// NOTE: Disabled pending unified installer refactoring
		// {
		//	name: "unsupported install method",
		//	testFunc: func(t *testing.T) {
		//		t.Helper()
		//		unifiedInstaller := domain.NewUnifiedInstaller(false, false)
		//		// ... test implementation
		//	},
		// },
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.testFunc(t)
		})
	}
}

// Helper functions for integration tests

func setupTestEnvironment(t *testing.T, tmpDir string) {
	t.Helper()
	// Create necessary directories without using environment variables
	dirs := []string{
		filepath.Join(tmpDir, "themes"),
		filepath.Join(tmpDir, "configs"),
		filepath.Join(tmpDir, ".config", "karei"),
		filepath.Join(tmpDir, ".local", "share", "karei"),
		filepath.Join(tmpDir, ".local", "bin"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755) //nolint:gosec
		require.NoError(t, err)
	}
}
