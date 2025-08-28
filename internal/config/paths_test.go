// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathUtils_GetKareiPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		envValue string
		want     string
		setup    func()
		cleanup  func()
	}{
		{
			name:     "uses KAREI_PATH env var when set",
			envValue: "/custom/karei/path",
			want:     "/custom/karei/path",
			setup:    func() {},
			cleanup: func() {
				_ = os.Unsetenv("KAREI_PATH")
			},
		},
		{
			name:     "falls back to default path when env var not set",
			envValue: "",
			want:     "", // Will be set dynamically in test
			setup: func() {
				_ = os.Unsetenv("KAREI_PATH")
			},
			cleanup: func() {},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := GetKareiPathWithEnv(testCase.envValue)

			if testCase.want == "" {
				// Dynamic expectation for default path
				home, err := os.UserHomeDir()
				require.NoError(t, err)

				expected := filepath.Join(home, ".local", "share", "karei")
				require.Equal(t, expected, got)
			} else {
				require.Equal(t, testCase.want, got)
			}
		})
	}
}

func TestPathUtils_GetXDGConfigHome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		envValue string
		want     string
		setup    func()
		cleanup  func()
	}{
		{
			name:     "uses XDG_CONFIG_HOME when set",
			envValue: "/custom/config",
			want:     "/custom/config",
			setup:    func() {},
			cleanup: func() {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			},
		},
		{
			name:     "falls back to ~/.config when not set",
			envValue: "",
			want:     "", // Will be set dynamically
			setup: func() {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			},
			cleanup: func() {},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := GetXDGConfigHomeWithEnv(testCase.envValue)

			if testCase.want == "" {
				// Dynamic expectation
				home, err := os.UserHomeDir()
				require.NoError(t, err)

				expected := filepath.Join(home, ".config")
				require.Equal(t, expected, got)
			} else {
				require.Equal(t, testCase.want, got)
			}
		})
	}
}

func TestPathUtils_ExpandPath(t *testing.T) {
	t.Parallel()

	// Get actual home directory
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "expands tilde to home",
			path: "~/test",
			want: filepath.Join(home, "test"),
		},
		{
			name: "handles plain tilde",
			path: "~",
			want: home,
		},
		{
			name: "expands XDG_CONFIG_HOME",
			path: "$XDG_CONFIG_HOME/app/config",
			want: filepath.Join(GetXDGConfigHome(), "app/config"),
		},
		{
			name: "expands XDG_DATA_HOME",
			path: "$XDG_DATA_HOME/app/data",
			want: filepath.Join(GetXDGDataHome(), "app/data"),
		},
		{
			name: "leaves absolute paths unchanged",
			path: "/absolute/path",
			want: "/absolute/path",
		},
		{
			name: "leaves relative paths unchanged",
			path: "relative/path",
			want: "relative/path",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ExpandPath(testCase.path)
			require.Equal(t, testCase.want, got)
		})
	}
}
