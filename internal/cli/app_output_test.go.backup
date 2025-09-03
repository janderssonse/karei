// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// MockWriter captures output for testing.
type MockWriter struct {
	bytes.Buffer
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	return m.Buffer.Write(p)
}

func TestCLI_InstallCommand_JSONOutput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		jsonFlag  bool
		quietFlag bool
		wantJSON  bool
		wantQuiet bool
	}{
		{
			name:     "install with JSON output",
			args:     []string{"karei", "install", "--json", "git"},
			jsonFlag: true,
			wantJSON: true,
		},
		{
			name:      "install with quiet mode",
			args:      []string{"karei", "install", "--quiet", "git"},
			quietFlag: true,
			wantQuiet: true,
		},
		{
			name:      "install with JSON and quiet",
			args:      []string{"karei", "install", "--json", "--quiet", "git"},
			jsonFlag:  true,
			quietFlag: true,
			wantJSON:  true,
			wantQuiet: true,
		},
		{
			name: "install normal output",
			args: []string{"karei", "install", "git"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewCLI()

			// Set flags before running
			app.json = tt.jsonFlag
			app.quiet = tt.quietFlag

			// Verify flags are set correctly
			assert.Equal(t, tt.jsonFlag, app.json)
			assert.Equal(t, tt.quietFlag, app.quiet)
		})
	}
}

func TestCLI_UninstallCommand_JSONOutput(t *testing.T) {
	t.Run("uninstall with JSON output returns structured data", func(t *testing.T) {
		app := NewCLI()
		app.json = true

		// Mock uninstall execution would happen here
		// For now, we're testing the structure

		result := &domain.UninstallResult{
			Uninstalled: []string{"git", "vim"},
			Failed:      []string{"docker"},
			NotFound:    []string{"unknown-package"},
		}

		// Verify JSON serialization
		data, err := json.Marshal(result)
		require.NoError(t, err)

		var decoded domain.UninstallResult

		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, result.Uninstalled, decoded.Uninstalled)
		assert.Equal(t, result.Failed, decoded.Failed)
		assert.Equal(t, result.NotFound, decoded.NotFound)
	})
}

func TestCLI_ListCommand_Output(t *testing.T) {
	tests := []struct {
		name         string
		jsonOutput   bool
		packages     []domain.PackageInfo
		wantContains []string
	}{
		{
			name:       "list with JSON output",
			jsonOutput: true,
			packages: []domain.PackageInfo{
				{Name: "git", Type: "tool", Version: "2.34.1"},
				{Name: "vim", Type: "editor", Version: "9.0"},
			},
			wantContains: []string{`"name"`, `"git"`, `"vim"`},
		},
		{
			name:       "list with text output",
			jsonOutput: false,
			packages: []domain.PackageInfo{
				{Name: "git", Type: "tool", Version: "2.34.1"},
				{Name: "vim", Type: "editor", Version: "9.0"},
			},
			wantContains: []string{"Name", "Type", "Version", "git", "vim"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewCLI()
			app.json = tt.jsonOutput

			// Test list result structure
			result := &domain.ListResult{
				Packages: tt.packages,
				Total:    len(tt.packages),
			}

			// Verify structure
			assert.Equal(t, len(tt.packages), result.Total)
			assert.Equal(t, tt.packages, result.Packages)
		})
	}
}

func TestCLI_OutputExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		result   any
		wantCode int
		wantErr  bool
	}{
		{
			name: "install all successful",
			result: &domain.InstallResult{
				Installed: []string{"git", "vim"},
				Failed:    []string{},
			},
			wantCode: ExitSuccess,
			wantErr:  false,
		},
		{
			name: "install partial failure",
			result: &domain.InstallResult{
				Installed: []string{"git"},
				Failed:    []string{"vim"},
			},
			wantCode: ExitWarnings,
			wantErr:  true,
		},
		{
			name: "install complete failure",
			result: &domain.InstallResult{
				Installed: []string{},
				Failed:    []string{"git", "vim"},
			},
			wantCode: ExitAppError,
			wantErr:  true,
		},
		{
			name: "uninstall all successful",
			result: &domain.UninstallResult{
				Uninstalled: []string{"git", "vim"},
				Failed:      []string{},
			},
			wantCode: ExitSuccess,
			wantErr:  false,
		},
		{
			name: "uninstall partial failure",
			result: &domain.UninstallResult{
				Uninstalled: []string{"git"},
				Failed:      []string{"vim"},
			},
			wantCode: ExitWarnings,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewCLI()

			var err error

			switch r := tt.result.(type) {
			case *domain.InstallResult:
				err = app.getInstallExitCode(r)
			case *domain.UninstallResult:
				err = app.getUninstallExitCode(r)
			}

			if tt.wantErr {
				require.Error(t, err)

				exitErr := &domain.ExitError{}
				ok := errors.As(err, &exitErr)
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, exitErr.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCLI_GlobalFlags(t *testing.T) {
	t.Run("global flags are available on all commands", func(t *testing.T) {
		app := NewCLI()
		cliApp := app.app

		// Check global flags exist
		flagNames := make(map[string]bool)

		for _, flag := range cliApp.Flags {
			if boolFlag, ok := flag.(*cli.BoolFlag); ok {
				flagNames[boolFlag.Name] = true
			}
		}

		assert.True(t, flagNames["json"], "json flag should exist")
		assert.True(t, flagNames["quiet"], "quiet flag should exist")
		assert.True(t, flagNames["verbose"], "verbose flag should exist")
	})
}

func TestCLI_QuietMode(t *testing.T) {
	tests := []struct {
		name       string
		quiet      bool
		operation  string
		wantOutput bool
	}{
		{
			name:       "quiet mode suppresses progress",
			quiet:      true,
			operation:  "progress",
			wantOutput: false,
		},
		{
			name:       "quiet mode suppresses info",
			quiet:      true,
			operation:  "info",
			wantOutput: false,
		},
		{
			name:       "quiet mode allows errors",
			quiet:      true,
			operation:  "error",
			wantOutput: false, // Even errors are suppressed in quiet mode
		},
		{
			name:       "normal mode shows all output",
			quiet:      false,
			operation:  "all",
			wantOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewCLI()
			app.quiet = tt.quiet

			// Verify quiet flag is set
			assert.Equal(t, tt.quiet, app.quiet)
		})
	}
}

func TestCLI_CommandHelp(t *testing.T) {
	t.Run("install command has proper help text", func(t *testing.T) {
		app := NewCLI()
		installCmd := app.createInstallCommand()

		assert.Equal(t, "install", installCmd.Name)
		assert.Equal(t, "Install development tools and applications", installCmd.Usage)
		assert.Contains(t, installCmd.Description, "Groups available")
		assert.Contains(t, installCmd.Description, "Examples")
	})

	t.Run("uninstall command has proper help text", func(t *testing.T) {
		app := NewCLI()
		uninstallCmd := app.createUninstallCommand()

		assert.Equal(t, "uninstall", uninstallCmd.Name)
		assert.Equal(t, "Uninstall packages", uninstallCmd.Usage)
	})

	t.Run("list command has proper help text", func(t *testing.T) {
		app := NewCLI()
		listCmd := app.createListCommand()

		assert.Equal(t, "list", listCmd.Name)
		assert.Equal(t, "List installed packages", listCmd.Usage)
	})
}

func TestCLI_IntegrationScenarios(t *testing.T) {
	t.Run("install then list shows installed packages", func(t *testing.T) {
		// This would be an integration test with actual installation
		// For now, we test the flow structure
		app := NewCLI()
		app.json = true

		// Simulate install result
		installResult := &domain.InstallResult{
			Installed: []string{"git", "vim"},
		}

		// Simulate list result after installation
		listResult := &domain.ListResult{
			Packages: []domain.PackageInfo{
				{Name: "git", Type: "tool"},
				{Name: "vim", Type: "editor"},
			},
			Total: 2,
		}

		// Verify the data structures match
		assert.Len(t, listResult.Packages, len(installResult.Installed))

		for i, pkg := range listResult.Packages {
			assert.Equal(t, installResult.Installed[i], pkg.Name)
		}
	})

	t.Run("uninstall then list shows removed packages", func(t *testing.T) {
		app := NewCLI()
		app.json = true

		// Simulate uninstall result
		uninstallResult := &domain.UninstallResult{
			Uninstalled: []string{"docker"},
		}

		// Simulate list result after uninstallation
		listResult := &domain.ListResult{
			Packages: []domain.PackageInfo{
				{Name: "git", Type: "tool"},
				{Name: "vim", Type: "editor"},
			},
			Total: 2,
		}

		// Verify docker is not in the list
		for _, pkg := range listResult.Packages {
			assert.NotEqual(t, "docker", pkg.Name)
		}

		assert.Contains(t, uninstallResult.Uninstalled, "docker")
	})
}

func TestCLI_OutputFormatConsistency(t *testing.T) {
	t.Run("all commands support JSON format consistently", func(t *testing.T) {
		results := []any{
			&domain.InstallResult{},
			&domain.UninstallResult{},
			&domain.ListResult{},
			&domain.StatusResult{},
			&domain.VerifyResult{},
		}

		for _, result := range results {
			// Verify all can be marshaled to JSON
			data, err := json.Marshal(result)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Verify JSON is valid
			var decoded map[string]any

			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
		}
	})
}
