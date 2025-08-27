// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// GetKareiPath returns the Karei installation path
// Consolidates logic repeated 8+ times across files.
func GetKareiPath() string {
	return GetKareiPathWithEnv(os.Getenv("KAREI_PATH"))
}

// GetKareiPathWithEnv returns the Karei path with custom environment override for testing.
func GetKareiPathWithEnv(kareiPath string) string {
	if kareiPath != "" {
		return kareiPath
	}

	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "karei")
	}

	return ""
}

// GetXDGConfigHome returns XDG config directory
// Consolidates logic repeated 6+ times across files.
func GetXDGConfigHome() string {
	return GetXDGConfigHomeWithEnv(os.Getenv("XDG_CONFIG_HOME"))
}

// GetXDGConfigHomeWithEnv returns XDG config directory with custom environment override for testing.
func GetXDGConfigHomeWithEnv(xdgConfigHome string) string {
	if xdgConfigHome != "" {
		return xdgConfigHome
	}

	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config")
	}

	return ""
}

// GetXDGDataHome returns XDG data directory.
func GetXDGDataHome() string {
	return GetXDGDataHomeWithEnv(os.Getenv("XDG_DATA_HOME"))
}

// GetXDGDataHomeWithEnv returns XDG data directory with custom environment override for testing.
func GetXDGDataHomeWithEnv(xdgDataHome string) string {
	if xdgDataHome != "" {
		return xdgDataHome
	}

	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share")
	}

	return ""
}

// GetUserBinDir returns user binary directory.
func GetUserBinDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "bin")
	}

	return ""
}

// GetUserBinDirWithEnv returns user binary directory with custom home directory for testing.
func GetUserBinDirWithEnv(homeDir string) string {
	if homeDir != "" {
		return filepath.Join(homeDir, ".local", "bin")
	}

	return GetUserBinDir()
}

// ExpandPath expands ~ and environment variables.
func ExpandPath(path string) string {
	return ExpandPathWithEnv(path, "", "")
}

// ExpandPathWithEnv expands paths with custom XDG environment variables for testing.
func ExpandPathWithEnv(path, xdgConfigHome, xdgDataHome string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}

	if strings.HasPrefix(path, "$XDG_CONFIG_HOME") {
		configHome := xdgConfigHome
		if configHome == "" {
			configHome = GetXDGConfigHome()
		}

		if after, found := strings.CutPrefix(path, "$XDG_CONFIG_HOME"); found {
			return configHome + after
		}
	}

	if strings.HasPrefix(path, "$XDG_DATA_HOME") {
		dataHome := xdgDataHome
		if dataHome == "" {
			dataHome = GetXDGDataHome()
		}

		if after, found := strings.CutPrefix(path, "$XDG_DATA_HOME"); found {
			return dataHome + after
		}
	}

	return path
}
