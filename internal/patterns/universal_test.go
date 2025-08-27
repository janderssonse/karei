// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package patterns

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	ErrInvalidTheme = errors.New("invalid theme")
)

func TestUniversalManager_BasicOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    UniversalConfig
		wantValid []string
		wantType  string
	}{
		{
			name: "theme manager configuration",
			config: UniversalConfig{
				Name:      "theme",
				Type:      TypeTheme,
				Available: []string{"tokyo-night", "catppuccin", "nord"},
				Verbose:   false,
				Handlers:  map[string]func(context.Context, string) error{"default": noOpSuccessHandler},
			},
			wantValid: []string{"tokyo-night", "catppuccin", "nord"},
			wantType:  "theme",
		},
		{
			name: "font manager configuration",
			config: UniversalConfig{
				Name:      "font",
				Type:      TypeFont,
				Available: []string{"JetBrainsMono", "FiraMono"},
				Verbose:   true,
				Handlers:  map[string]func(context.Context, string) error{"default": noOpSuccessHandler},
			},
			wantValid: []string{"JetBrainsMono", "FiraMono"},
			wantType:  "font",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			manager := NewUniversalManager(testCase.config)

			// Test basic properties
			require.Equal(t, testCase.config.Name, manager.Name)
			require.Equal(t, testCase.wantType, manager.Type)
			require.Equal(t, testCase.wantValid, manager.Available)
			require.Equal(t, testCase.config.Verbose, manager.verbose)

			// Test validation
			for _, valid := range testCase.wantValid {
				require.True(t, manager.IsValid(valid), "should validate %s", valid)
			}

			require.False(t, manager.IsValid("invalid-choice"))
			require.False(t, manager.IsValid(""))

			// Test GetAvailable
			require.Equal(t, testCase.wantValid, manager.GetAvailable())

			// Test GetCurrent (should return first available as default)
			current := manager.GetCurrent()
			require.NotEmpty(t, current)
			require.Contains(t, testCase.wantValid, current)
		})
	}
}

func TestUniversalManager_ConfigDetection(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		configType    string
		configContent string
		expectedValue string
		setup         func(string) string
	}{
		{
			name:          "detect theme from config file",
			configType:    "theme",
			configContent: "KAREI_THEME=tokyo-night\n",
			expectedValue: "tokyo-night",
			setup: func(dir string) string {
				configPath := filepath.Join(dir, "theme")

				return configPath
			},
		},
		{
			name:          "detect font from config file",
			configType:    "font",
			configContent: "KAREI_FONT=JetBrainsMono\n",
			expectedValue: "JetBrainsMono",
			setup: func(dir string) string {
				configPath := filepath.Join(dir, "font")

				return configPath
			},
		},
		{
			name:          "ignore invalid value in config",
			configType:    "theme",
			configContent: "KAREI_THEME=invalid-theme\n",
			expectedValue: "tokyo-night", // Should fall back to default (first available)
			setup: func(dir string) string {
				configPath := filepath.Join(dir, "theme")

				return configPath
			},
		},
		{
			name:          "ignore malformed config",
			configType:    "theme",
			configContent: "invalid config format\n",
			expectedValue: "tokyo-night", // Should fall back to default
			setup: func(dir string) string {
				configPath := filepath.Join(dir, "theme")

				return configPath
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			configPath := testCase.setup(tmpDir)

			// Write config content
			err := os.WriteFile(configPath, []byte(testCase.configContent), 0644) //nolint:gosec
			require.NoError(t, err)

			// Create manager with custom config path
			var available []string
			if testCase.configType == "font" {
				available = []string{"JetBrainsMono", "FiraMono"}
			} else {
				available = []string{"tokyo-night", "catppuccin"}
			}

			manager := &UniversalManager{
				Name:       testCase.configType,
				Type:       testCase.configType,
				Available:  available,
				ConfigPath: configPath,
				verbose:    false,
			}

			current := manager.GetCurrent()
			require.Equal(t, testCase.expectedValue, current)
		})
	}
}

func TestUniversalManager_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		target      string
		available   []string
		handlers    map[string]func(context.Context, string) error
		wantErr     bool
		errContains string
		expectCall  string
	}{
		{
			name:      "successful apply with default handler",
			target:    "tokyo-night",
			available: []string{"tokyo-night", "catppuccin"},
			handlers: map[string]func(context.Context, string) error{
				"default": func(_ context.Context, target string) error { //nolint:unparam
					require.Equal(t, "tokyo-night", target)

					return nil
				},
			},
			wantErr:    false,
			expectCall: "tokyo-night",
		},
		{
			name:      "successful apply with specific handler",
			target:    "special-theme",
			available: []string{"special-theme", "normal-theme"},
			handlers: map[string]func(context.Context, string) error{
				"special-theme": func(_ context.Context, target string) error { //nolint:unparam
					require.Equal(t, "special-theme", target)

					return nil
				},
				"default": func(context.Context, string) error { //nolint:unparam
					t.Error("Should not call default handler")

					return nil
				},
			},
			wantErr:    false,
			expectCall: "special-theme",
		},
		{
			name:        "invalid target",
			target:      "invalid-target",
			available:   []string{"valid-target"},
			handlers:    map[string]func(context.Context, string) error{"default": noOpSuccessHandler},
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name:        "no handler available",
			target:      "valid-target",
			available:   []string{"valid-target"},
			handlers:    map[string]func(context.Context, string) error{}, // No handlers
			wantErr:     true,
			errContains: "no handler available",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			manager := &UniversalManager{
				Name:      "test",
				Type:      "test",
				Available: testCase.available,
				handlers:  testCase.handlers,
			}

			err := manager.Apply(context.Background(), testCase.target)

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

func TestUniversalManager_SaveCurrent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		choice      string
		available   []string
		wantErr     bool
		errContains string
	}{
		{
			name:      "save valid choice",
			choice:    "tokyo-night",
			available: []string{"tokyo-night", "catppuccin"},
			wantErr:   false,
		},
		{
			name:        "save invalid choice",
			choice:      "invalid-choice",
			available:   []string{"tokyo-night", "catppuccin"},
			wantErr:     true,
			errContains: "invalid",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			configPath := filepath.Join(tmpDir, testCase.name+"_config")

			manager := &UniversalManager{
				Name:       "test",
				Type:       "theme",
				Available:  testCase.available,
				ConfigPath: configPath,
			}

			err := manager.SaveCurrent(testCase.choice)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errContains != "" {
					require.Contains(t, err.Error(), testCase.errContains)
				}
			} else {
				require.NoError(t, err)

				// Verify file was written correctly
				content, err := os.ReadFile(configPath) //nolint:gosec
				require.NoError(t, err)

				expected := "KAREI_THEME=" + testCase.choice + "\n"
				require.Equal(t, expected, string(content))

				// Verify manager state was updated
				require.Equal(t, testCase.choice, manager.Current)
			}
		})
	}
}

func TestUniversalManager_Status(t *testing.T) {
	t.Parallel()

	manager := &UniversalManager{
		Name:       "test-manager",
		Type:       "theme",
		Available:  []string{"theme1", "theme2"},
		Current:    "theme1",
		ConfigPath: "/test/config/path",
	}

	status := manager.Status()

	require.Equal(t, "theme", status["type"])
	require.Equal(t, "theme1", status["current"])
	require.Equal(t, []string{"theme1", "theme2"}, status["available"])
	require.Equal(t, "/test/config/path", status["config"])
}

func TestUniversalCommand_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		available   []string
		interactive bool
		wantErr     bool
		errContains string
		expectCall  bool
	}{
		{
			name:       "execute with valid argument",
			args:       []string{"tokyo-night"},
			available:  []string{"tokyo-night", "catppuccin"},
			expectCall: true,
			wantErr:    false,
		},
		{
			name:       "execute with list argument",
			args:       []string{"list"},
			available:  []string{"tokyo-night", "catppuccin"},
			expectCall: false,
			wantErr:    false,
		},
		{
			name:        "execute with invalid argument",
			args:        []string{"invalid-theme"},
			available:   []string{"tokyo-night", "catppuccin"},
			expectCall:  false, // Should not call handler for invalid input
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name:        "execute with no args, non-interactive",
			args:        []string{},
			available:   []string{"tokyo-night", "catppuccin"},
			interactive: false,
			expectCall:  false,
			wantErr:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			handlerCalled := false
			handler := func(_ context.Context, target string) error {
				handlerCalled = true

				if target == "invalid-theme" {
					return fmt.Errorf("%w: %s", ErrInvalidTheme, target)
				}

				return nil
			}

			manager := NewUniversalManager(UniversalConfig{
				Name:      "test",
				Type:      TypeTheme,
				Available: testCase.available,
				Handlers:  map[string]func(context.Context, string) error{"default": handler},
			})

			command := &UniversalCommand{
				Name:        "test",
				Manager:     manager,
				Interactive: testCase.interactive,
			}

			err := command.Execute(context.Background(), testCase.args)

			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errContains != "" {
					require.Contains(t, err.Error(), testCase.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, testCase.expectCall, handlerCalled)
		})
	}
}

func TestCommandExecutor_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verbose bool
		dryRun  bool
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "dry run mode",
			verbose: false,
			dryRun:  true,
			command: "echo",
			args:    []string{"test"},
			wantErr: false,
		},
		{
			name:    "successful command execution",
			verbose: false,
			dryRun:  false,
			command: "echo",
			args:    []string{"test"},
			wantErr: false,
		},
		{
			name:    "command not found",
			verbose: false,
			dryRun:  false,
			command: "nonexistentcommand12345",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			executor := NewCommandExecutor(testCase.verbose, testCase.dryRun)

			err := executor.Execute(context.Background(), testCase.command, testCase.args...)

			if testCase.wantErr && !testCase.dryRun {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCommandExecutor_ExecuteWithOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		dryRun       bool
		command      string
		args         []string
		wantContains string
		wantErr      bool
	}{
		{
			name:         "dry run returns empty output",
			dryRun:       true,
			command:      "echo",
			args:         []string{"test"},
			wantContains: "",
			wantErr:      false,
		},
		{
			name:         "successful command with output",
			dryRun:       false,
			command:      "echo",
			args:         []string{"hello world"},
			wantContains: "hello world",
			wantErr:      false,
		},
		{
			name:    "command fails",
			dryRun:  false,
			command: "false", // Always exits with code 1
			args:    []string{},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			executor := NewCommandExecutor(false, testCase.dryRun)

			output, err := executor.ExecuteWithOutput(context.Background(), testCase.command, testCase.args...)

			if testCase.wantErr && !testCase.dryRun {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				if testCase.wantContains != "" && !testCase.dryRun {
					require.Contains(t, output, testCase.wantContains)
				}
			}
		})
	}
}

func TestServiceController_Operations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		operation   string
		serviceName string
		dryRun      bool
		wantErr     bool
	}{
		{
			name:        "dry run operations should not error",
			operation:   "enable",
			serviceName: "test-service",
			dryRun:      true,
			wantErr:     false,
		},
		{
			name:        "enable non-existent service",
			operation:   "enable",
			serviceName: "nonexistent-service-12345",
			dryRun:      true,  // Use dry run to avoid sudo
			wantErr:     false, // Dry run should not error
		},
		{
			name:        "start non-existent service",
			operation:   "start",
			serviceName: "nonexistent-service-12345",
			dryRun:      true,  // Use dry run to avoid sudo
			wantErr:     false, // Dry run should not error
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			controller := NewServiceController(false, testCase.dryRun)

			var err error

			switch testCase.operation {
			case "enable":
				err = controller.Enable(context.Background(), testCase.serviceName)
			case "start":
				err = controller.Start(context.Background(), testCase.serviceName)
			}

			if testCase.wantErr && !testCase.dryRun {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServiceController_IsActive(t *testing.T) {
	t.Parallel()

	controller := NewServiceController(false, true) // Use dry run to avoid sudo

	tests := []struct {
		name         string
		serviceName  string
		expectActive bool
	}{
		{
			name:         "check non-existent service",
			serviceName:  "nonexistent-service-12345",
			expectActive: true, // In dry run mode, IsActive returns true
		},
		// Note: We can't reliably test for active services across different systems
		// This would require mocking or running in a controlled environment
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			active := controller.IsActive(context.Background(), testCase.serviceName)
			require.Equal(t, testCase.expectActive, active)
		})
	}
}

// Helper functions for tests

func noOpSuccessHandler(_ context.Context, _ string) error {
	// No-op handler that always succeeds
	return nil
}
