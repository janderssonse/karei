// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package config provides configuration management utilities for Karei.
package config

import (
	"path/filepath"
)

// GetConfigPath returns the configuration path for a given component type.
func GetConfigPath(componentType string) string {
	configHome := GetXDGConfigHome()

	return filepath.Join(configHome, "karei", componentType)
}
