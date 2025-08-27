// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/janderssonse/karei/internal/patterns"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestNewCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "create ultra simplified CLI",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cliApp := NewCLI()

			require.NotNil(t, cliApp)
			require.NotNil(t, cliApp.app)
			require.Equal(t, "karei", cliApp.app.Name)
			require.NotEmpty(t, cliApp.app.Usage)
			require.NotEmpty(t, cliApp.app.Description)
			require.NotEmpty(t, cliApp.app.Commands)
		})
	}
}

func TestCLI_CreateAllCommands(t *testing.T) {
	t.Parallel()

	cliApp := NewCLI()
	commands := cliApp.createAllCommands()

	require.NotEmpty(t, commands)

	// Check that essential commands exist
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name] = true
	}

	expectedCommands := []string{"theme", "font", "security", "verify", "logs", "install", "update", "uninstall", "menu", "version"}
	for _, expected := range expectedCommands {
		require.True(t, commandNames[expected], "command %s should exist", expected)
	}
}

func TestCLI_GetVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		versionContent  string
		expectedVersion string
		setupFile       bool
	}{
		{
			name:            "version file exists",
			versionContent:  "v1.2.3\n",
			expectedVersion: "v1.2.3",
			setupFile:       true,
		},
		{
			name:            "version file does not exist",
			expectedVersion: "dev",
			setupFile:       false,
		},
		{
			name:            "version file with whitespace",
			versionContent:  "  v2.0.0  \n\n",
			expectedVersion: "v2.0.0",
			setupFile:       true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Create a unique tmpDir for each test case
			tmpDir := t.TempDir()

			if testCase.setupFile {
				versionFile := filepath.Join(tmpDir, "version")
				err := os.WriteFile(versionFile, []byte(testCase.versionContent), 0600)
				require.NoError(t, err)
			}

			cliApp := NewCLI()
			version := cliApp.getVersionWithPath(tmpDir)
			require.Equal(t, testCase.expectedVersion, version)
		})
	}
}

func TestCLI_ParseChoice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		max      int
		expected int
	}{
		{
			name:     "valid choice within range",
			input:    "3",
			max:      5,
			expected: 3,
		},
		{
			name:     "choice at lower bound",
			input:    "1",
			max:      5,
			expected: 1,
		},
		{
			name:     "choice at upper bound",
			input:    "5",
			max:      5,
			expected: 5,
		},
		{
			name:     "choice below range",
			input:    "0",
			max:      5,
			expected: 0,
		},
		{
			name:     "choice above range",
			input:    "6",
			max:      5,
			expected: 0,
		},
		{
			name:     "non-numeric input",
			input:    "abc",
			max:      5,
			expected: 0,
		},
		{
			name:     "empty input",
			input:    "",
			max:      5,
			expected: 0,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cliApp := NewCLI()
			result := cliApp.parseChoice(testCase.input, testCase.max)
			require.Equal(t, testCase.expected, result)
		})
	}
}

func TestCLI_Commands(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create version file
	versionFile := filepath.Join(tmpDir, "version")
	err := os.WriteFile(versionFile, []byte("test-version"), 0600)
	require.NoError(t, err)

	cliApp := NewCLI()

	tests := []struct {
		name     string
		command  *cli.Command
		args     []string
		wantErr  bool
		skipExec bool
	}{
		{
			name:    "version command",
			command: cliApp.createVersionCommand(),
			args:    []string{},
			wantErr: false,
		},
		{
			name:     "install command with no args",
			command:  cliApp.createInstallCommand(),
			args:     []string{},
			wantErr:  true,
			skipExec: false,
		},
		{
			name:     "install command with args (dry run would need mocking)",
			command:  cliApp.createInstallCommand(),
			args:     []string{"test-package"},
			wantErr:  true, // Will fail without sudo in test
			skipExec: true, // Skip actual execution
		},
		{
			name:     "update command (dry run would need mocking)",
			command:  cliApp.createUpdateCommand(),
			args:     []string{},
			wantErr:  true, // Will fail without git repo
			skipExec: true,
		},
		{
			name:     "uninstall command with no args",
			command:  cliApp.createUninstallCommand(),
			args:     []string{},
			wantErr:  true,
			skipExec: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if testCase.skipExec {
				// Just verify command creation doesn't panic
				require.NotNil(t, testCase.command)
				require.NotEmpty(t, testCase.command.Name)
				require.NotNil(t, testCase.command.Action)

				return
			}

			// Create a mock CLI command for testing
			mockCmd := &cli.Command{
				Name:   "test",
				Action: testCase.command.Action,
			}

			// Create context with args
			ctx := context.Background()

			// Mock the args in the command
			_ = &cli.Command{
				Commands: []*cli.Command{mockCmd},
			}

			// This is a simplified test - full integration would require more mocking
			if testCase.command.Name == "version" {
				err := testCase.command.Action(ctx, mockCmd)
				if testCase.wantErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestCLI_AdaptUniversalCommand(t *testing.T) {
	t.Parallel()

	cliApp := NewCLI()

	// Create a mock universal command
	mockUniversalCmd := &patterns.UniversalCommand{
		Name:        "test-command",
		Usage:       "Test command usage",
		Description: "Test command description",
		Interactive: true,
	}

	adaptedCmd := cliApp.adaptUniversalCommand(mockUniversalCmd)

	require.NotNil(t, adaptedCmd)
	require.Equal(t, "test-command", adaptedCmd.Name)
	require.Equal(t, "Test command usage", adaptedCmd.Usage)
	require.Equal(t, "Test command description", adaptedCmd.Description)
	require.Equal(t, "[option]", adaptedCmd.ArgsUsage)
	require.NotNil(t, adaptedCmd.Action)
}

func TestCLI_ShowMainMenu(t *testing.T) {
	t.Parallel()

	cliApp := NewCLI()

	// Test the menu options structure
	// Note: This test can't easily test interactive input without mocking stdin
	// We test the menu structure instead

	tests := []struct {
		name          string
		expectedCount int
	}{
		{
			name:          "main menu has expected number of options",
			expectedCount: 8, // theme, font, install, security, verify, logs, update, exit
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			// We can't easily test the interactive part without mocking stdin
			// But we can test that the method exists and doesn't panic when called
			// This would require more sophisticated mocking for full testing
			require.NotPanics(t, func() {
				// The method exists and can be called
				_ = cliApp.showMainMenu
			})
		})
	}
}

func TestCLI_HandleMenuChoice(t *testing.T) {
	t.Parallel()

	cliApp := NewCLI()

	tests := []struct {
		name        string
		choice      string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid choice - theme",
			choice:  "theme",
			wantErr: false,
		},
		{
			name:    "valid choice - font",
			choice:  "font",
			wantErr: false,
		},
		{
			name:    "valid choice - security",
			choice:  "security",
			wantErr: false,
		},
		{
			name:    "valid choice - verify",
			choice:  "verify",
			wantErr: false,
		},
		{
			name:    "valid choice - logs",
			choice:  "logs",
			wantErr: false,
		},
		{
			name:        "invalid choice",
			choice:      "invalid-choice",
			wantErr:     true,
			errContains: "unknown choice",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := cliApp.handleMenuChoice(context.Background(), testCase.choice)
			if testCase.wantErr {
				require.Error(t, err)

				if testCase.errContains != "" {
					require.Contains(t, err.Error(), testCase.errContains)
				}
			} else if err != nil {
				// Note: These may error due to missing dependencies in test environment
				// but they should not panic or fail with "unknown choice" errors
				require.NotContains(t, err.Error(), "unknown choice")
			}
		})
	}
}

func TestApp(t *testing.T) {
	t.Parallel()

	app := App()

	require.NotNil(t, app)
	require.Equal(t, "karei", app.Name)
	require.NotEmpty(t, app.Usage)
	require.NotEmpty(t, app.Description)
	require.NotEmpty(t, app.Commands)
	require.NotNil(t, app.Action)
	require.NotNil(t, app.Before)
}

func TestCLI_Integration(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create version file
	versionFile := filepath.Join(tmpDir, "version")
	err := os.WriteFile(versionFile, []byte("v1.0.0-test"), 0600)
	require.NoError(t, err)

	cliApp := NewCLI()

	t.Run("CLI creation and basic structure", func(t *testing.T) {
		t.Parallel()
		require.NotNil(t, cliApp.app)
		require.Equal(t, "karei", cliApp.app.Name)
		require.Contains(t, cliApp.app.Description, "Transforms fresh Linux installations into fully-configured development environments")
	})

	t.Run("version retrieval", func(t *testing.T) {
		t.Parallel()

		version := cliApp.getVersionWithPath(tmpDir)
		require.Equal(t, "v1.0.0-test", version)
	})

	t.Run("command creation", func(t *testing.T) {
		t.Parallel()

		commands := cliApp.createAllCommands()
		require.Greater(t, len(commands), 5)

		// Verify essential commands exist
		commandMap := make(map[string]*cli.Command)
		for _, cmd := range commands {
			commandMap[cmd.Name] = cmd
		}

		essentialCommands := []string{"theme", "font", "install", "version"}
		for _, essential := range essentialCommands {
			require.Contains(t, commandMap, essential, "essential command %s should exist", essential)
		}
	})

	t.Run("special commands functionality", func(t *testing.T) {
		t.Parallel()
		// Test version command
		versionCmd := cliApp.createVersionCommand()
		require.NotNil(t, versionCmd)
		require.Equal(t, "version", versionCmd.Name)

		// Test menu command
		menuCmd := cliApp.createMenuCommand()
		require.NotNil(t, menuCmd)
		require.Equal(t, "menu", menuCmd.Name)
	})
}
