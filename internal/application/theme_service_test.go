// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janderssonse/karei/internal/application"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type themeTestCase struct {
	name        string
	themeName   string
	setupMocks  func(*testutil.MockFileManager, *testutil.MockCommandRunner)
	wantErr     bool
	errContains string
}

func setupGnomeMocks(cr *testutil.MockCommandRunner) {
	cr.On("Execute", mock.Anything, "gsettings", "set",
		"org.gnome.desktop.interface", "color-scheme", mock.AnythingOfType("string")).Return(nil).Once()
	cr.On("Execute", mock.Anything, "gsettings", "set",
		"org.gnome.desktop.interface", "gtk-theme", mock.AnythingOfType("string")).Return(nil).Once()
	cr.On("Execute", mock.Anything, "gsettings", "set",
		"org.gnome.desktop.interface", "icon-theme", mock.AnythingOfType("string")).Return(nil).Once()
	cr.On("Execute", mock.Anything, "gsettings", "set",
		"org.gnome.desktop.interface", "cursor-theme", mock.AnythingOfType("string")).Return(nil).Once()
	cr.On("Execute", mock.Anything, "gsettings", "set",
		"org.gnome.desktop.interface", "accent-color", mock.AnythingOfType("string")).Return(nil).Once()
}

func setupBackgroundMocks(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner, theme string) {
	fm.On("FileExists", mock.MatchedBy(func(path string) bool {
		return strings.Contains(path, theme+"/background.jpg")
	})).Return(true).Once()

	cr.On("Execute", mock.Anything, "gsettings", "set",
		"org.gnome.desktop.background", "picture-uri",
		mock.AnythingOfType("string")).Return(nil).Once()
	cr.On("Execute", mock.Anything, "gsettings", "set",
		"org.gnome.desktop.background", "picture-uri-dark",
		mock.AnythingOfType("string")).Return(nil).Once()
	cr.On("Execute", mock.Anything, "gsettings", "set",
		"org.gnome.desktop.background", "picture-options", "zoom").Return(nil).Once()
	cr.On("Execute", mock.Anything, "gsettings", "set",
		"org.gnome.desktop.screensaver", "picture-uri",
		mock.AnythingOfType("string")).Return(nil).Once()
}

func TestThemeService_ApplyTheme(t *testing.T) {
	t.Parallel()

	tests := []themeTestCase{
		{
			name:      "apply theme successfully with existing config",
			themeName: "gruvbox",
			setupMocks: func(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner) {
				setupGnomeMocks(cr)
				setupBackgroundMocks(fm, cr, "gruvbox")

				// Terminal theme mocks
				cr.On("ExecuteWithOutput", mock.Anything, "gsettings", "get",
					"org.gnome.Terminal.ProfilesList", "default").Return("'default-profile-id'", nil).Once()
				// Check if terminal theme file exists
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "gruvbox/gnome-terminal.json")
				})).Return(false).Once()

				// Btop EnsureDir call (always happens first)
				fm.On("EnsureDir", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "config/btop")
				})).Return(nil).Once()
				// Check if btop theme file exists
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "gruvbox/btop.theme")
				})).Return(false).Once()

				// VSCode theme installation (gruvbox has VSCodeExtension)
				cr.On("Execute", mock.Anything, "code", "--install-extension", "jdinhlife.gruvbox").Return(nil).Once()

				// VSCode settings file exists
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				})).Return(true).Once()
				fm.On("ReadFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				})).Return([]byte("{}"), nil).Once()
				fm.On("WriteFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				}), mock.AnythingOfType("[]uint8")).Return(nil).Once()

				// Chrome theme - check if preferences file exists
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "google-chrome/Default/Preferences")
				})).Return(false).Once()
			},
			wantErr: false,
		},
		{
			name:      "apply gruvbox theme successfully",
			themeName: "gruvbox",
			setupMocks: func(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner) {
				setupGnomeMocks(cr)
				setupBackgroundMocks(fm, cr, "gruvbox")

				// Terminal theme mocks
				cr.On("ExecuteWithOutput", mock.Anything, "gsettings", "get",
					"org.gnome.Terminal.ProfilesList", "default").Return("'default-profile-id'", nil).Once()
				// Check if terminal theme file exists - return true for gruvbox (it has a terminal theme)
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "gruvbox/gnome-terminal.json")
				})).Return(true).Once()

				// Read terminal theme file
				fm.On("ReadFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "gruvbox/gnome-terminal.json")
				})).Return([]byte("{}"), nil).Once()

				// Apply terminal settings (empty theme so nothing happens)

				// Btop EnsureDir call (always happens first)
				fm.On("EnsureDir", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "config/btop")
				})).Return(nil).Once()

				// Btop theme check - return false to skip btop theming
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "gruvbox/btop.theme")
				})).Return(false).Once()

				// VSCode theme installation
				cr.On("Execute", mock.Anything, "code", "--install-extension", "jdinhlife.gruvbox").Return(nil).Once()

				// VSCode settings
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				})).Return(false).Once()
				fm.On("EnsureDir", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User")
				})).Return(nil).Once()
				fm.On("WriteFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				}), mock.AnythingOfType("[]uint8")).Return(nil).Once()
				// Read the file after creation
				fm.On("ReadFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				})).Return([]byte("{}"), nil).Once()
				// Write the updated settings
				fm.On("WriteFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				}), mock.AnythingOfType("[]uint8")).Return(nil).Once()

				// Chrome theme - check if preferences file exists
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "google-chrome/Default/Preferences")
				})).Return(false).Once()
			},
			wantErr: false,
		},
		{
			name:      "unknown theme returns error",
			themeName: "nonexistent-theme",
			setupMocks: func(_ *testutil.MockFileManager, _ *testutil.MockCommandRunner) {
				// No mocks needed - theme is checked before any operations
			},
			wantErr:     true,
			errContains: "unknown theme",
		},
		{
			name:      "config file not found creates new one",
			themeName: "nord",
			setupMocks: func(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner) {
				// Apply GNOME settings (in the order they're called in ApplyGnomeSettings)
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.interface", "color-scheme", mock.AnythingOfType("string")).Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.interface", "gtk-theme", mock.AnythingOfType("string")).Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.interface", "icon-theme", mock.AnythingOfType("string")).Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.interface", "cursor-theme", mock.AnythingOfType("string")).Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.interface", "accent-color", mock.AnythingOfType("string")).Return(nil).Once()

				// Check background file exists in ApplyBackground
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "nord/background.png")
				})).Return(true).Once()

				// Apply background settings (in order from ApplyBackground)
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.background", "picture-uri",
					mock.AnythingOfType("string")).Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.background", "picture-uri-dark",
					mock.AnythingOfType("string")).Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.background", "picture-options", "zoom").Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.screensaver", "picture-uri",
					mock.AnythingOfType("string")).Return(nil).Once()

				// Terminal theme mocks
				cr.On("ExecuteWithOutput", mock.Anything, "gsettings", "get",
					"org.gnome.Terminal.ProfilesList", "default").Return("'default-profile-id'", nil).Once()
				// Check if terminal theme file exists - return false to skip terminal theming
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "nord/gnome-terminal.json")
				})).Return(false).Once()

				// Btop EnsureDir call (always happens first)
				fm.On("EnsureDir", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "config/btop")
				})).Return(nil).Once()

				// Btop theme check - return false to skip btop theming
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "nord/btop.theme")
				})).Return(false).Once()

				// VSCode theme installation (nord has VSCodeExtension)
				cr.On("Execute", mock.Anything, "code", "--install-extension", "arcticicestudio.nord-visual-studio-code").Return(nil).Once()

				// VSCode settings
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				})).Return(false).Once()
				fm.On("EnsureDir", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User")
				})).Return(nil).Once()
				fm.On("WriteFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				}), mock.AnythingOfType("[]uint8")).Return(nil).Once()
				// Read the file after creation
				fm.On("ReadFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				})).Return([]byte("{}"), nil).Once()
				// Write the updated settings
				fm.On("WriteFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "Code/User/settings.json")
				}), mock.AnythingOfType("[]uint8")).Return(nil).Once()

				// Chrome theme - check if preferences file exists
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, "google-chrome/Default/Preferences")
				})).Return(false).Once()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockFM := new(testutil.MockFileManager)
			mockCR := new(testutil.MockCommandRunner)

			if tt.setupMocks != nil {
				tt.setupMocks(mockFM, mockCR)
			}

			tmpDir := t.TempDir()
			service := application.NewThemeService(mockFM, mockCR,
				filepath.Join(tmpDir, "config"),
				filepath.Join(tmpDir, "themes"))

			err := service.ApplyTheme(context.Background(), tt.themeName)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			mockFM.AssertExpectations(t)
			mockCR.AssertExpectations(t)
		})
	}
}

func TestThemeService_GetAvailableThemes(t *testing.T) {
	t.Parallel()

	service := application.NewThemeService(nil, nil, "", "")
	themes := service.GetAvailableThemes()

	// Verify we have the expected themes
	expectedThemes := []string{
		"gruvbox", "gruvbox-light", "catppuccin",
		"nord", "tokyo-night", "everforest",
	}

	for _, expected := range expectedThemes {
		assert.Contains(t, themes, expected, "should have %s theme", expected)
	}

	// Each theme should have required properties
	for name, theme := range themes {
		assert.NotEmpty(t, theme.Name, "theme %s should have name", name)
		assert.NotEmpty(t, theme.ColorScheme, "theme %s should have color scheme", name)
		// Most themes should have GTK theme set
		if name != "gruvbox-light" {
			assert.NotEmpty(t, theme.GtkTheme, "theme %s should have gtk theme", name)
		}
	}
}

func TestThemeService_ApplyBackground(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		themeName      string
		backgroundFile string
		setupMocks     func(*testutil.MockFileManager, *testutil.MockCommandRunner)
		wantErr        bool
	}{
		{
			name:           "apply background successfully",
			themeName:      "catppuccin",
			backgroundFile: "background.jpg",
			setupMocks: func(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner) {
				fm.On("FileExists", mock.AnythingOfType("string")).Return(true)
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.background", "picture-uri",
					mock.AnythingOfType("string")).Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.background", "picture-uri-dark",
					mock.AnythingOfType("string")).Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.background", "picture-options",
					"zoom").Return(nil).Once()
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.screensaver", "picture-uri",
					mock.AnythingOfType("string")).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name:           "gsettings command fails",
			themeName:      "nord",
			backgroundFile: "background.png",
			setupMocks: func(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner) {
				fm.On("FileExists", mock.AnythingOfType("string")).Return(true)
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.background", "picture-uri",
					mock.AnythingOfType("string")).Return(errors.New("gsettings failed")).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockFM := new(testutil.MockFileManager)

			mockCR := new(testutil.MockCommandRunner)
			if tt.setupMocks != nil {
				tt.setupMocks(mockFM, mockCR)
			}

			service := application.NewThemeService(mockFM, mockCR, "/tmp/config", "/tmp/themes")
			err := service.ApplyBackground(context.Background(), tt.themeName, tt.backgroundFile)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			mockFM.AssertExpectations(t)
			mockCR.AssertExpectations(t)
		})
	}
}

func TestThemeService_ListThemes(t *testing.T) {
	t.Parallel()

	service := application.NewThemeService(nil, nil, "", "")
	themeNames := service.ListThemes()

	// Should return list of theme names
	assert.NotEmpty(t, themeNames)
	assert.Contains(t, themeNames, "gruvbox")
	assert.Contains(t, themeNames, "catppuccin")
	assert.Contains(t, themeNames, "nord")

	// Verify alphabetical order or consistent ordering
	assert.Greater(t, len(themeNames), 5, "should have multiple themes")
}
