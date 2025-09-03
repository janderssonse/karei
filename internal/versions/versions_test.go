// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package versions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janderssonse/karei/internal/versions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockPathResolver provides test paths for version config.
type MockPathResolver struct {
	userConfigPath string
	xdgConfigHome  string
}

func (m *MockPathResolver) GetUserVersionsConfigPath() string {
	return m.userConfigPath
}

func (m *MockPathResolver) GetXDGConfigHome() string {
	return m.xdgConfigHome
}

// TestVersionPriority tests the business logic of version resolution priority
// Business Rule: User config > System config > "latest".
func TestVersionPriority(t *testing.T) {
	// Create temp directories for configs
	tempDir := t.TempDir()
	systemConfigPath := filepath.Join(tempDir, "system", "versions.toml")
	userConfigPath := filepath.Join(tempDir, "user", "versions.toml")

	// Create system config
	require.NoError(t, os.MkdirAll(filepath.Dir(systemConfigPath), 0750))

	systemConfig := `[tools]
go = "1.20"
node = "18.0.0"
rust = "1.70.0"`
	require.NoError(t, os.WriteFile(systemConfigPath, []byte(systemConfig), 0600))

	// Create user config (overrides some versions)
	require.NoError(t, os.MkdirAll(filepath.Dir(userConfigPath), 0750))

	userConfig := `[tools]
go = "1.21"
python = "3.11"`
	require.NoError(t, os.WriteFile(userConfigPath, []byte(userConfig), 0600))

	// Create version manager with mock resolver
	mockResolver := &MockPathResolver{
		userConfigPath: userConfigPath,
		xdgConfigHome:  tempDir,
	}
	manager := versions.NewVersionManagerWithResolver(systemConfigPath, mockResolver)

	tests := []struct {
		name        string
		tool        string
		expected    string
		description string
	}{
		{
			name:        "user config overrides system config",
			tool:        "go",
			expected:    "1.21",
			description: "Business rule: user config takes priority",
		},
		{
			name:        "system config used when no user override",
			tool:        "node",
			expected:    "18.0.0",
			description: "Business rule: fallback to system config",
		},
		{
			name:        "user config adds new tool not in system",
			tool:        "python",
			expected:    "3.11",
			description: "Business rule: user can add tools",
		},
		{
			name:        "default to latest when tool not configured",
			tool:        "ruby",
			expected:    "latest",
			description: "Business rule: default to latest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			version, err := manager.GetVersion(tc.tool)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, version, tc.description)
		})
	}
}

// TestVersionManagerWithMissingConfigs tests behavior when configs don't exist.
func TestVersionManagerWithMissingConfigs(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "does-not-exist", "versions.toml")

	mockResolver := &MockPathResolver{
		userConfigPath: filepath.Join(tempDir, "also-does-not-exist", "versions.toml"),
		xdgConfigHome:  tempDir,
	}
	manager := versions.NewVersionManagerWithResolver(nonExistentPath, mockResolver)

	// Business rule: When no config exists, always return "latest" without error
	version, err := manager.GetVersion("any-tool")
	require.NoError(t, err, "Missing config should not cause error")
	assert.Equal(t, "latest", version, "Should default to latest when no config")
}

// TestVersionManagerWithCorruptConfig tests behavior with invalid TOML.
func TestVersionManagerWithCorruptConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "versions.toml")

	// Write invalid TOML
	invalidTOML := `[tools
this is not valid TOML`
	require.NoError(t, os.WriteFile(configPath, []byte(invalidTOML), 0600))

	mockResolver := &MockPathResolver{
		userConfigPath: filepath.Join(tempDir, "no-user-config", "versions.toml"),
		xdgConfigHome:  tempDir,
	}
	manager := versions.NewVersionManagerWithResolver(configPath, mockResolver)

	// Business rule: Corrupt config defaults to "latest" (graceful degradation)
	version, err := manager.GetVersion("go")
	require.NoError(t, err, "Corrupt config should not error on GetVersion")
	assert.Equal(t, "latest", version, "Should default to latest on corrupt config")

	// GetAllVersions should return error for corrupt config
	_, err = manager.GetAllVersions()
	assert.Error(t, err, "GetAllVersions should error on corrupt config")
}

// TestGetAllVersions tests retrieving all configured versions.
func TestGetAllVersions(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "versions.toml")

	config := `[tools]
go = "1.21"
node = "20.0.0"
rust = "1.75.0"`
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0600))

	manager := versions.NewVersionManager(configPath)
	allVersions, err := manager.GetAllVersions()

	require.NoError(t, err)
	assert.Len(t, allVersions, 3)
	assert.Equal(t, "1.21", allVersions["go"])
	assert.Equal(t, "20.0.0", allVersions["node"])
	assert.Equal(t, "1.75.0", allVersions["rust"])
}

// TestVersionManagerWithPermissionError tests behavior when config file is unreadable.
func TestVersionManagerWithPermissionError(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "versions.toml")

	// Create config file
	config := `[tools]
go = "1.21"`
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0600))

	// Make file unreadable (only works on Unix-like systems)
	err := os.Chmod(configPath, 0000)
	if err != nil {
		t.Skip("Cannot test permission errors on this system")
	}

	defer func() {
		_ = os.Chmod(configPath, 0600) // Restore permissions for cleanup
	}()

	manager := versions.NewVersionManager(configPath)

	// Business rule: Permission errors should gracefully default to "latest"
	version, err := manager.GetVersion("go")
	require.NoError(t, err, "GetVersion should not error on permission issues")
	assert.Equal(t, "latest", version, "Should default to latest on permission error")
}

// TestVersionManagerWithEmptyToolsSection tests behavior with empty [tools] section.
func TestVersionManagerWithEmptyToolsSection(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "versions.toml")

	// Empty tools section
	config := `[tools]`
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0600))

	manager := versions.NewVersionManager(configPath)

	// Business rule: Empty config should return "latest" for any tool
	version, err := manager.GetVersion("any-tool")
	require.NoError(t, err)
	assert.Equal(t, "latest", version)

	// GetAllVersions should return empty map
	allVersions, err := manager.GetAllVersions()
	require.NoError(t, err)
	assert.Empty(t, allVersions)
}

// TestVersionManagerWithPartiallyCorruptConfig tests mixed valid/invalid config.
func TestVersionManagerWithPartiallyCorruptConfig(t *testing.T) {
	tempDir := t.TempDir()
	systemPath := filepath.Join(tempDir, "system", "versions.toml")
	userPath := filepath.Join(tempDir, "user", "versions.toml")

	// System config is valid
	require.NoError(t, os.MkdirAll(filepath.Dir(systemPath), 0750))

	systemConfig := `[tools]
go = "1.20"
node = "18.0.0"`
	require.NoError(t, os.WriteFile(systemPath, []byte(systemConfig), 0600))

	// User config is corrupt
	require.NoError(t, os.MkdirAll(filepath.Dir(userPath), 0750))

	userConfig := `[tools
this is invalid`
	require.NoError(t, os.WriteFile(userPath, []byte(userConfig), 0600))

	mockResolver := &MockPathResolver{
		userConfigPath: userPath,
		xdgConfigHome:  tempDir,
	}
	manager := versions.NewVersionManagerWithResolver(systemPath, mockResolver)

	// Business rule: Falls back to system config when user config is corrupt
	version, err := manager.GetVersion("go")
	require.NoError(t, err)
	assert.Equal(t, "1.20", version, "Should use system config when user config is corrupt")
}

// TestVersionManagerCaseInsensitivity tests tool name case handling.
func TestVersionManagerCaseInsensitivity(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "versions.toml")

	config := `[tools]
Go = "1.21"
NODE = "20.0.0"
RuSt = "1.75.0"`
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0600))

	manager := versions.NewVersionManager(configPath)

	// Test exact case matches
	version, err := manager.GetVersion("Go")
	require.NoError(t, err)
	assert.Equal(t, "1.21", version)

	// Test case variations - should return "latest" if case-sensitive
	version, err = manager.GetVersion("go")
	require.NoError(t, err)
	assert.Equal(t, "latest", version, "Tool names should be case-sensitive")

	version, err = manager.GetVersion("node")
	require.NoError(t, err)
	assert.Equal(t, "latest", version, "Tool names should be case-sensitive")
}
