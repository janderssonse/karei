// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package config

import (
	"path/filepath"
)

// GetConfigPath returns config path for component type.
func GetConfigPath(componentType string) string {
	configHome := GetXDGConfigHome()

	return filepath.Join(configHome, "karei", componentType)
}
