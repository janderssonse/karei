// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetKareiPathWithEnv(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		wantPath   string
		expectHome bool
	}{
		{
			name:       "uses environment variable when set",
			envValue:   "/custom/karei/path",
			wantPath:   "/custom/karei/path",
			expectHome: false,
		},
		{
			name:       "falls back to default when empty",
			envValue:   "",
			wantPath:   "",
			expectHome: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetKareiPathWithEnv(tt.envValue)

			if tt.expectHome {
				home, _ := os.UserHomeDir()
				expected := filepath.Join(home, ".local", "share", "karei")
				assert.Equal(t, expected, result)
			} else {
				assert.Equal(t, tt.wantPath, result)
			}
		})
	}
}

func TestGetXDGConfigHomeWithEnv(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		wantPath   string
		expectHome bool
	}{
		{
			name:       "uses XDG_CONFIG_HOME when set",
			envValue:   "/custom/config",
			wantPath:   "/custom/config",
			expectHome: false,
		},
		{
			name:       "falls back to .config when empty",
			envValue:   "",
			wantPath:   "",
			expectHome: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetXDGConfigHomeWithEnv(tt.envValue)

			if tt.expectHome {
				home, _ := os.UserHomeDir()
				expected := filepath.Join(home, ".config")
				assert.Equal(t, expected, result)
			} else {
				assert.Equal(t, tt.wantPath, result)
			}
		})
	}
}

func TestGetXDGDataHomeWithEnv(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		wantPath   string
		expectHome bool
	}{
		{
			name:       "uses XDG_DATA_HOME when set",
			envValue:   "/custom/data",
			wantPath:   "/custom/data",
			expectHome: false,
		},
		{
			name:       "falls back to .local/share when empty",
			envValue:   "",
			wantPath:   "",
			expectHome: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetXDGDataHomeWithEnv(tt.envValue)

			if tt.expectHome {
				home, _ := os.UserHomeDir()
				expected := filepath.Join(home, ".local", "share")
				assert.Equal(t, expected, result)
			} else {
				assert.Equal(t, tt.wantPath, result)
			}
		})
	}
}

func TestGetUserBinDirWithEnv(t *testing.T) {
	tests := []struct {
		name     string
		homeDir  string
		wantPath string
	}{
		{
			name:     "uses provided home directory",
			homeDir:  "/custom/home",
			wantPath: "/custom/home/.local/bin",
		},
		{
			name:     "falls back to current user home",
			homeDir:  "",
			wantPath: "", // Will be checked dynamically
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserBinDirWithEnv(tt.homeDir)

			if tt.homeDir == "" {
				home, _ := os.UserHomeDir()
				expected := filepath.Join(home, ".local", "bin")
				assert.Equal(t, expected, result)
			} else {
				assert.Equal(t, tt.wantPath, result)
			}
		})
	}
}

func TestExpandPathWithEnv(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name          string
		path          string
		xdgConfigHome string
		xdgDataHome   string
		want          string
	}{
		{
			name: "expands tilde to home",
			path: "~/documents",
			want: filepath.Join(home, "documents"),
		},
		{
			name: "expands tilde alone",
			path: "~",
			want: home,
		},
		{
			name:          "expands XDG_CONFIG_HOME variable",
			path:          "$XDG_CONFIG_HOME/app",
			xdgConfigHome: "/custom/config",
			want:          "/custom/config/app",
		},
		{
			name:        "expands XDG_DATA_HOME variable",
			path:        "$XDG_DATA_HOME/app",
			xdgDataHome: "/custom/data",
			want:        "/custom/data/app",
		},
		{
			name: "leaves regular paths unchanged",
			path: "/absolute/path",
			want: "/absolute/path",
		},
		{
			name:          "falls back to default XDG_CONFIG_HOME",
			path:          "$XDG_CONFIG_HOME/app",
			xdgConfigHome: "",
			want:          filepath.Join(home, ".config", "app"),
		},
		{
			name:        "falls back to default XDG_DATA_HOME",
			path:        "$XDG_DATA_HOME/app",
			xdgDataHome: "",
			want:        filepath.Join(home, ".local", "share", "app"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPathWithEnv(tt.path, tt.xdgConfigHome, tt.xdgDataHome)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	tests := []struct {
		name          string
		componentType string
	}{
		{
			name:          "returns path for fonts component",
			componentType: "fonts",
		},
		{
			name:          "returns path for themes component",
			componentType: "themes",
		},
		{
			name:          "returns path for config component",
			componentType: "config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetConfigPath(tt.componentType)

			home, _ := os.UserHomeDir()
			expected := filepath.Join(home, ".config", "karei", tt.componentType)
			assert.Equal(t, expected, result)
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "expands home directory",
			path: "~/test",
			want: filepath.Join(home, "test"),
		},
		{
			name: "handles absolute paths",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.path)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGetKareiPath(t *testing.T) {
	t.Run("uses KAREI_PATH environment variable", func(t *testing.T) {
		testPath := "/test/karei"
		t.Setenv("KAREI_PATH", testPath)
		assert.Equal(t, testPath, GetKareiPath())
	})

	t.Run("falls back to default path", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".local", "share", "karei")
		assert.Equal(t, expected, GetKareiPath())
	})
}

func TestGetXDGConfigHome(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME environment variable", func(t *testing.T) {
		testPath := "/test/config"
		t.Setenv("XDG_CONFIG_HOME", testPath)
		assert.Equal(t, testPath, GetXDGConfigHome())
	})

	t.Run("falls back to default path", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config")
		assert.Equal(t, expected, GetXDGConfigHome())
	})
}

func TestGetXDGDataHome(t *testing.T) {
	t.Run("uses XDG_DATA_HOME environment variable", func(t *testing.T) {
		testPath := "/test/data"
		t.Setenv("XDG_DATA_HOME", testPath)
		assert.Equal(t, testPath, GetXDGDataHome())
	})

	t.Run("falls back to default path", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".local", "share")
		assert.Equal(t, expected, GetXDGDataHome())
	})
}

func TestGetUserBinDir(t *testing.T) {
	result := GetUserBinDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".local", "bin")
	assert.Equal(t, expected, result)
}
