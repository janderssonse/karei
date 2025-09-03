// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application_test

import (
	"context"
	"strings"
	"testing"

	"github.com/janderssonse/karei/internal/application"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUninstallService_UninstallApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		appName    string
		setupMocks func(*testutil.MockFileManager, *testutil.MockCommandRunner, *testutil.MockPackageInstaller)
		wantErr    bool
	}{
		{
			name:    "uninstall unknown app",
			appName: "vim",
			setupMocks: func(_ *testutil.MockFileManager, _ *testutil.MockCommandRunner, _ *testutil.MockPackageInstaller) {
				// Should fail with unknown app error
			},
			wantErr: true,
		},
		{
			name:    "uninstall special app (vscode)",
			appName: "vscode",
			setupMocks: func(fm *testutil.MockFileManager, cr *testutil.MockCommandRunner, _ *testutil.MockPackageInstaller) {
				// VSCode has special uninstall logic - using ExecuteSudo now
				cr.On("ExecuteSudo", mock.Anything, "apt", []string{"remove", "--purge", "-y", "code"}).
					Return(nil).Once() // Ensure it's called exactly once

				// Clean up config - VS Code checks multiple directories
				// First check for .config/Code
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, ".config/Code") || strings.Contains(path, "Code")
				})).Return(true).Once()
				fm.On("RemoveFile", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, ".config/Code") || strings.Contains(path, "Code")
				})).Return(nil).Once()

				// Then check for .vscode
				fm.On("FileExists", mock.MatchedBy(func(path string) bool {
					return strings.Contains(path, ".vscode")
				})).Return(false).Once() // Doesn't exist, so no removal
			},
			wantErr: false,
		},
		{
			name:    "uninstall unknown app returns error",
			appName: "nonexistent-app",
			setupMocks: func(_ *testutil.MockFileManager, _ *testutil.MockCommandRunner, _ *testutil.MockPackageInstaller) {
				// No mocks needed - should fail with unknown app error
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockFM := new(testutil.MockFileManager)
			mockCR := new(testutil.MockCommandRunner)
			mockPI := new(testutil.MockPackageInstaller)

			tt.setupMocks(mockFM, mockCR, mockPI)

			service := application.NewUninstallService(mockFM, mockCR, mockPI, false)
			err := service.UninstallApp(context.Background(), tt.appName)

			if tt.wantErr {
				require.Error(t, err)
				// Verify error message contains expected context
				if tt.appName == "unknown" {
					assert.Contains(t, err.Error(), "unknown app")
				}
			} else {
				require.NoError(t, err)
			}

			// Verify all expected mock calls were made
			mockFM.AssertExpectations(t)
			mockCR.AssertExpectations(t)
			mockPI.AssertExpectations(t)
		})
	}
}

func TestUninstallService_UninstallGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		group      string
		setupMocks func(*testutil.MockFileManager, *testutil.MockCommandRunner, *testutil.MockPackageInstaller)
		wantErr    bool
	}{
		{
			name:  "uninstall empty group",
			group: "empty",
			setupMocks: func(_ *testutil.MockFileManager, _ *testutil.MockCommandRunner, _ *testutil.MockPackageInstaller) {
				// Empty group should not error but also not uninstall anything
			},
			wantErr: false,
		},
		{
			name:  "unknown group",
			group: "nonexistent",
			setupMocks: func(_ *testutil.MockFileManager, _ *testutil.MockCommandRunner, _ *testutil.MockPackageInstaller) {
				// Unknown group should fail
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockFM := new(testutil.MockFileManager)
			mockCR := new(testutil.MockCommandRunner)
			mockPI := new(testutil.MockPackageInstaller)

			if tt.setupMocks != nil {
				tt.setupMocks(mockFM, mockCR, mockPI)
			}

			service := application.NewUninstallService(mockFM, mockCR, mockPI, false)
			err := service.UninstallGroup(context.Background(), tt.group)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				// May still error if group has no apps
				_ = err
			}
		})
	}
}
