// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package versions provides version management utilities for karei.
package versions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/janderssonse/karei/internal/platform"
	"github.com/pelletier/go-toml/v2"
)

// VersionConfig represents the structure of versions.toml file.
type VersionConfig struct {
	Tools map[string]string `toml:"tools"`
}

// PathResolver provides path resolution functions.
type PathResolver interface {
	GetUserVersionsConfigPath() string
	GetXDGConfigHome() string
}

// DefaultPathResolver implements PathResolver using the default paths.
type DefaultPathResolver struct{}

// GetUserVersionsConfigPath returns the path to the user's versions configuration file.
func (p *DefaultPathResolver) GetUserVersionsConfigPath() string {
	return filepath.Join(platform.GetXDGConfigHome(), "mise", "versions.toml")
}

// GetXDGConfigHome returns the XDG config home directory.
func (p *DefaultPathResolver) GetXDGConfigHome() string {
	return platform.GetXDGConfigHome()
}

// VersionManager handles version configuration for mise tools.
type VersionManager struct {
	configPath   string
	pathResolver PathResolver
}

// NewVersionManager creates a new version manager with the given config path.
func NewVersionManager(configPath string) *VersionManager {
	return &VersionManager{
		configPath:   configPath,
		pathResolver: &DefaultPathResolver{},
	}
}

// NewVersionManagerWithResolver creates a new version manager with custom path resolver.
func NewVersionManagerWithResolver(configPath string, resolver PathResolver) *VersionManager {
	return &VersionManager{
		configPath:   configPath,
		pathResolver: resolver,
	}
}

// GetVersion returns the pinned version for a tool, or "latest" if not found.
// Priority: user config > system config > "latest".
func (v *VersionManager) GetVersion(toolName string) (string, error) {
	// First try user config
	userConfigPath := v.pathResolver.GetUserVersionsConfigPath()
	if platform.FileExists(userConfigPath) {
		userManager := NewVersionManagerWithResolver(userConfigPath, v.pathResolver)
		if userConfig, err := userManager.loadVersionConfig(); err == nil {
			if version, exists := userConfig.Tools[toolName]; exists {
				return version, nil
			}
		}
	}

	// Then try system config
	config, err := v.loadVersionConfig()
	if err != nil {
		// If config file doesn't exist or can't be read, return "latest"
		return "latest", nil //nolint:nilerr // intentionally returning nil error with default value
	}

	if version, exists := config.Tools[toolName]; exists {
		return version, nil
	}

	// Default to "latest" if tool not found in config
	return "latest", nil
}

// GetAllVersions returns all pinned versions from the config.
func (v *VersionManager) GetAllVersions() (map[string]string, error) {
	config, err := v.loadVersionConfig()
	if err != nil {
		return make(map[string]string), err
	}

	return config.Tools, nil
}

// loadVersionConfig loads the version configuration from the TOML file.
func (v *VersionManager) loadVersionConfig() (*VersionConfig, error) {
	data, err := os.ReadFile(v.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read version config: %w", err)
	}

	var config VersionConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse version config: %w", err)
	}

	return &config, nil
}

// GetVersionsConfigPath returns the path to the versions configuration file.
func GetVersionsConfigPath() string {
	return filepath.Join(platform.GetKareiPath(), "configs", "versions.toml")
}
