// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package versions provides version management utilities for karei.
package versions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const nonExistentUserConfigPath = "/non/existent/user/config"

// TestPathResolver is a test implementation of PathResolver.
type TestPathResolver struct {
	userConfigPath string
	xdgConfigHome  string
}

func (p *TestPathResolver) GetUserVersionsConfigPath() string {
	return p.userConfigPath
}

func (p *TestPathResolver) GetXDGConfigHome() string {
	return p.xdgConfigHome
}

func TestVersionManager_GetVersion(t *testing.T) {
	t.Parallel() // Now safe to run in parallel
	// Note: Not parallel due to potential user config interference
	// Create temporary directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "versions.toml")

	// Create test config content
	testConfig := `[tools]
java = "17.0.0"
maven = "3.9.0"
nonexistent = "1.0.0"`

	require.NoError(t, os.WriteFile(configPath, []byte(testConfig), 0600))

	// Create test path resolver that returns non-existent user config path
	testResolver := &TestPathResolver{
		userConfigPath: nonExistentUserConfigPath,
		xdgConfigHome:  tmpDir,
	}

	// Create version manager with test config and resolver
	versionManager := NewVersionManagerWithResolver(configPath, testResolver)

	tests := []struct {
		name        string
		toolName    string
		expectedVer string
		expectError bool
	}{
		{
			name:        "existing tool with version",
			toolName:    "java",
			expectedVer: "17.0.0",
			expectError: false,
		},
		{
			name:        "another existing tool",
			toolName:    "maven",
			expectedVer: "3.9.0",
			expectError: false,
		},
		{
			name:        "non-existent tool returns latest",
			toolName:    "unknown-tool",
			expectedVer: "latest",
			expectError: false,
		},
		{
			name:        "empty tool name returns latest",
			toolName:    "",
			expectedVer: "latest",
			expectError: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			version, err := versionManager.GetVersion(testCase.toolName)

			if testCase.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedVer, version)
			}
		})
	}
}

func TestVersionManager_GetVersion_NonExistentConfig(t *testing.T) {
	t.Parallel()

	// Create test path resolver that returns non-existent user config path
	testResolver := &TestPathResolver{
		userConfigPath: nonExistentUserConfigPath,
		xdgConfigHome:  "/tmp",
	}

	// Create version manager with non-existent config and test resolver
	versionManager := NewVersionManagerWithResolver("/non/existent/path/versions.toml", testResolver)

	version, err := versionManager.GetVersion("java")
	require.NoError(t, err)
	assert.Equal(t, "latest", version)
}

func TestVersionManager_GetVersion_UserConfigPriority(t *testing.T) {
	t.Parallel() // Now safe to run in parallel
	// Create temporary directories
	tmpDir := t.TempDir()
	systemConfigPath := filepath.Join(tmpDir, "system-versions.toml")
	userConfigPath := filepath.Join(tmpDir, "user-versions.toml")

	// Create system config
	systemConfig := `[tools]
java = "17.0.0"
maven = "3.9.0"`
	require.NoError(t, os.WriteFile(systemConfigPath, []byte(systemConfig), 0600))

	// Create user config with different versions
	userConfig := `[tools]
java = "21.0.0"
gradle = "8.5"`
	require.NoError(t, os.WriteFile(userConfigPath, []byte(userConfig), 0600))

	// Create test path resolver that returns the user config path
	testResolver := &TestPathResolver{
		userConfigPath: userConfigPath,
		xdgConfigHome:  tmpDir,
	}

	// Create version manager with system config and test resolver
	versionManager := NewVersionManagerWithResolver(systemConfigPath, testResolver)

	tests := []struct {
		name        string
		toolName    string
		expectedVer string
	}{
		{
			name:        "user config overrides system config",
			toolName:    "java",
			expectedVer: "21.0.0",
		},
		{
			name:        "user config provides version not in system",
			toolName:    "gradle",
			expectedVer: "8.5",
		},
		{
			name:        "falls back to system config",
			toolName:    "maven",
			expectedVer: "3.9.0",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			version, err := versionManager.GetVersion(testCase.toolName)
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedVer, version)
		})
	}
}

func TestVersionManager_GetAllVersions(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "versions.toml")

	// Create test config content
	testConfig := `[tools]
java = "17.0.0"
maven = "3.9.0"
gradle = "8.5"`

	require.NoError(t, os.WriteFile(configPath, []byte(testConfig), 0600))

	// Create version manager with test config
	versionManager := NewVersionManager(configPath)

	versions, err := versionManager.GetAllVersions()
	require.NoError(t, err)

	expected := map[string]string{
		"java":   "17.0.0",
		"maven":  "3.9.0",
		"gradle": "8.5",
	}

	assert.Equal(t, expected, versions)
}

func TestVersionManager_GetAllVersions_NonExistentConfig(t *testing.T) {
	t.Parallel()

	// Create version manager with non-existent config
	versionManager := NewVersionManager("/non/existent/path/versions.toml")

	versions, err := versionManager.GetAllVersions()
	require.Error(t, err)
	assert.Empty(t, versions)
}

func TestVersionManager_LoadVersionConfig_InvalidTOML(t *testing.T) {
	t.Parallel()

	// Create test path resolver that returns non-existent user config path
	testResolver := &TestPathResolver{
		userConfigPath: nonExistentUserConfigPath,
		xdgConfigHome:  "/tmp",
	}

	// Create temporary directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-versions.toml")

	// Create invalid TOML content
	invalidConfig := `[tools
java = "17.0.0"`

	require.NoError(t, os.WriteFile(configPath, []byte(invalidConfig), 0600))

	// Create version manager with invalid config and test resolver
	versionManager := NewVersionManagerWithResolver(configPath, testResolver)

	version, err := versionManager.GetVersion("java")
	require.NoError(t, err) // Should not error, just return "latest"
	assert.Equal(t, "latest", version)
}

func TestGetVersionsConfigPath(t *testing.T) {
	t.Parallel()

	path := GetVersionsConfigPath()
	assert.Contains(t, path, "configs/versions.toml")
	assert.Contains(t, path, "karei")
}

func TestDefaultPathResolver_GetUserVersionsConfigPath(t *testing.T) {
	t.Parallel()

	resolver := &DefaultPathResolver{}
	path := resolver.GetUserVersionsConfigPath()
	assert.Contains(t, path, "mise/versions.toml")
	assert.Contains(t, path, ".config")
}
