// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/janderssonse/karei/internal/application"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFontService_DownloadAndInstallFont(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fontName    string
		setupMocks  func(*testutil.MockFileManager, *testutil.MockCommandRunner, *testutil.MockNetworkClient)
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful font installation",
			fontName: "JetBrainsMono",
			setupMocks: func(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner, nc *testutil.MockNetworkClient) {
				// Create temp dir
				fm.On("EnsureDir", mock.AnythingOfType("string")).Return(nil)

				// Download font
				nc.On("DownloadFile", mock.Anything,
					"https://github.com/ryanoasis/nerd-fonts/releases/latest/download/JetBrainsMono.zip",
					mock.AnythingOfType("string")).Return(nil)

				// Read zip file - return a valid empty zip
				emptyZip := []byte{0x50, 0x4b, 0x05, 0x06, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
				fm.On("ReadFile", mock.AnythingOfType("string")).Return(emptyZip, nil)

				// Clean up temp dir
				cr.On("Execute", mock.Anything, "rm", "-rf", mock.AnythingOfType("string")).Return(nil)

				// Update font cache
				cr.On("Execute", mock.Anything, "fc-cache", "-f").Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "unknown font",
			fontName: "UnknownFont",
			setupMocks: func(_ *testutil.MockFileManager, _ *testutil.MockCommandRunner, _ *testutil.MockNetworkClient) {
				// No mocks needed - should fail early
			},
			wantErr:     true,
			errContains: "unknown font",
		},
		{
			name:     "berkeley mono (local only)",
			fontName: "BerkeleyMono",
			setupMocks: func(_ *testutil.MockFileManager, _ *testutil.MockCommandRunner, _ *testutil.MockNetworkClient) {
				// Berkeley Mono has no URL, should return nil immediately
			},
			wantErr: false,
		},
		{
			name:     "download failure",
			fontName: "FiraMono",
			setupMocks: func(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner, nc *testutil.MockNetworkClient) {
				fm.On("EnsureDir", mock.AnythingOfType("string")).Return(nil)
				nc.On("DownloadFile", mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("network error"))
				cr.On("Execute", mock.Anything, "rm", "-rf", mock.AnythingOfType("string")).Return(nil)
			},
			wantErr:     true,
			errContains: "failed to download",
		},
		{
			name:     "font cache update failure",
			fontName: "MesloLGS",
			setupMocks: func(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner, nc *testutil.MockNetworkClient) {
				fm.On("EnsureDir", mock.AnythingOfType("string")).Return(nil)
				nc.On("DownloadFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				// Return valid empty zip
				emptyZip := []byte{0x50, 0x4b, 0x05, 0x06, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
				fm.On("ReadFile", mock.AnythingOfType("string")).Return(emptyZip, nil)
				cr.On("Execute", mock.Anything, "rm", "-rf", mock.AnythingOfType("string")).Return(nil)
				cr.On("Execute", mock.Anything, "fc-cache", "-f").Return(errors.New("cache update failed"))
			},
			wantErr:     true,
			errContains: "cache update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup mocks
			mockFM := new(testutil.MockFileManager)
			mockCR := new(testutil.MockCommandRunner)
			mockNC := new(testutil.MockNetworkClient)

			if tt.setupMocks != nil {
				tt.setupMocks(mockFM, mockCR, mockNC)
			}

			// Create service with temp directory
			tmpDir := t.TempDir()
			fontsDir := filepath.Join(tmpDir, "fonts")
			configDir := filepath.Join(tmpDir, "config")

			service := application.NewFontService(mockFM, mockCR, mockNC, fontsDir, configDir)

			// Execute
			err := service.DownloadAndInstallFont(context.Background(), tt.fontName)

			// Assert
			if tt.wantErr {
				require.Error(t, err)

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify expectations
			mockFM.AssertExpectations(t)
			mockCR.AssertExpectations(t)
			mockNC.AssertExpectations(t)
		})
	}
}

type fontSizeTestCase struct {
	name       string
	setupMocks func(*testutil.MockCommandRunner)
	wantErr    bool
}

func testFontSizeChange(t *testing.T, tests []fontSizeTestCase, operation func(*application.FontService, context.Context) error) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockCR := new(testutil.MockCommandRunner)
			mockFM := new(testutil.MockFileManager)
			mockNC := new(testutil.MockNetworkClient)

			tt.setupMocks(mockCR)

			tmpDir := t.TempDir()
			service := application.NewFontService(mockFM, mockCR, mockNC,
				filepath.Join(tmpDir, "fonts"), filepath.Join(tmpDir, "config"))

			err := operation(service, context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			mockCR.AssertExpectations(t)
		})
	}
}

func TestFontService_IncreaseFontSize(t *testing.T) {
	t.Parallel()

	tests := []fontSizeTestCase{
		{
			name: "increase from 12 to 13",
			setupMocks: func(cr *testutil.MockCommandRunner) {
				// Get current size
				cr.On("ExecuteWithOutput", mock.Anything, "gsettings", "get",
					"org.gnome.desktop.interface", "font-name").Return("'Sans 12'", nil)
				// Set new size - gsettings needs quotes
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.interface", "font-name", "'Sans 13'").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "already at max size",
			setupMocks: func(cr *testutil.MockCommandRunner) {
				cr.On("ExecuteWithOutput", mock.Anything, "gsettings", "get",
					"org.gnome.desktop.interface", "font-name").Return("'Sans 24'", nil)
			},
			wantErr: true,
		},
	}

	testFontSizeChange(t, tests, func(s *application.FontService, ctx context.Context) error {
		return s.IncreaseFontSize(ctx)
	})
}

func TestFontService_DecreaseFontSize(t *testing.T) {
	t.Parallel()

	tests := []fontSizeTestCase{
		{
			name: "decrease from 12 to 11",
			setupMocks: func(cr *testutil.MockCommandRunner) {
				// Get current size
				cr.On("ExecuteWithOutput", mock.Anything, "gsettings", "get",
					"org.gnome.desktop.interface", "font-name").Return("'Sans 12'", nil)
				// Set new size - output keeps quotes
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.interface", "font-name", "'Sans 11'").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "already at min size",
			setupMocks: func(cr *testutil.MockCommandRunner) {
				cr.On("ExecuteWithOutput", mock.Anything, "gsettings", "get",
					"org.gnome.desktop.interface", "font-name").Return("'Sans 6'", nil)
			},
			wantErr: true,
		},
	}

	testFontSizeChange(t, tests, func(s *application.FontService, ctx context.Context) error {
		return s.DecreaseFontSize(ctx)
	})
}

func TestFontService_GetAvailableFonts(t *testing.T) {
	t.Parallel()

	service := application.NewFontService(nil, nil, nil, "", "")
	fonts := service.GetAvailableFonts()

	// Should have predefined fonts
	assert.NotEmpty(t, fonts)
	assert.Contains(t, fonts, "JetBrainsMono")
	assert.Contains(t, fonts, "FiraMono")
	assert.Contains(t, fonts, "CaskaydiaMono")
	assert.Contains(t, fonts, "MesloLGS")
	assert.Contains(t, fonts, "BerkeleyMono")

	// Verify Berkeley Mono has no URL (local only)
	berkeley := fonts["BerkeleyMono"]
	assert.Empty(t, berkeley.URL)
}

func TestFontService_ApplySystemFont(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fontName   string
		setupMocks func(*testutil.MockFileManager, *testutil.MockCommandRunner)
		wantErr    bool
	}{
		{
			name:     "apply JetBrainsMono",
			fontName: "JetBrainsMono",
			setupMocks: func(_ *testutil.MockFileManager, cr *testutil.MockCommandRunner) {
				// Apply to terminal - get profile first
				cr.On("ExecuteWithOutput", mock.Anything, "gsettings", "get",
					"org.gnome.Terminal.ProfilesList", "default").
					Return("'b1dcc9dd-5262-4d8d-a863-c897e6d979b9'", nil)
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.Terminal.Legacy.Profile:/org/gnome/terminal/legacy/profiles:/:b1dcc9dd-5262-4d8d-a863-c897e6d979b9/",
					"font", "'JetBrainsMono Nerd Font 11'").Return(nil)

				// Apply to editor
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.gedit.preferences.editor", "editor-font",
					"'JetBrainsMono Nerd Font 11'").Return(nil)
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.gedit.preferences.editor", "use-default-font", "false").Return(nil)

				// Apply to system monospace font
				cr.On("Execute", mock.Anything, "gsettings", "set",
					"org.gnome.desktop.interface", "monospace-font-name",
					"'JetBrainsMono Nerd Font 11'").Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "unknown font",
			fontName: "UnknownFont",
			setupMocks: func(_ *testutil.MockFileManager, _ *testutil.MockCommandRunner) {
				// No mocks needed - should fail early
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockFM := new(testutil.MockFileManager)
			mockCR := new(testutil.MockCommandRunner)
			mockNC := new(testutil.MockNetworkClient)

			if tt.setupMocks != nil {
				tt.setupMocks(mockFM, mockCR)
			}

			tmpDir := t.TempDir()
			service := application.NewFontService(mockFM, mockCR, mockNC,
				filepath.Join(tmpDir, "fonts"), filepath.Join(tmpDir, "config"))

			err := service.ApplySystemFont(context.Background(), tt.fontName)

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

func TestFontService_ListFonts(t *testing.T) {
	t.Parallel()

	service := application.NewFontService(nil, nil, nil, "", "")
	fonts := service.ListFonts()

	// Should have all predefined fonts
	assert.Len(t, fonts, 5)
	assert.Contains(t, fonts, "JetBrainsMono")
	assert.Contains(t, fonts, "FiraMono")
	assert.Contains(t, fonts, "CaskaydiaMono")
	assert.Contains(t, fonts, "MesloLGS")
	assert.Contains(t, fonts, "BerkeleyMono")
}

func TestFontService_GetFont(t *testing.T) {
	t.Parallel()

	service := application.NewFontService(nil, nil, nil, "", "")

	tests := []struct {
		name     string
		fontName string
		wantErr  bool
	}{
		{
			name:     "existing font",
			fontName: "JetBrainsMono",
			wantErr:  false,
		},
		{
			name:     "non-existing font",
			fontName: "NonExistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			font, err := service.GetFont(tt.fontName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, font)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, font)
				assert.Equal(t, tt.fontName, font.Name)
			}
		})
	}
}
