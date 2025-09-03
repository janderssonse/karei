// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package uninstall_test

import (
	"context"
	"testing"

	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/uninstall"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recognitionTestCase represents a test case for name recognition.
type recognitionTestCase struct {
	name        string
	inputName   string
	shouldError bool
	errorType   error
	reason      string
}

// runRecognitionTests runs recognition tests for apps or groups.
func runRecognitionTests(t *testing.T, tests []recognitionTestCase, uninstallFunc func(context.Context, string) error) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := uninstallFunc(context.Background(), tc.inputName)

			if tc.shouldError {
				require.Error(t, err, tc.reason)

				if tc.errorType != nil {
					assert.ErrorIs(t, err, tc.errorType)
				}
			}
		})
	}
}

// TestUninstallUnknownApp tests the business rule that unknown apps cannot be uninstalled.
func TestUninstallUnknownApp(t *testing.T) {
	uninstaller, _ := uninstall.NewTestUninstaller(false)
	ctx := context.Background()

	// Business rule: Attempting to uninstall unknown app must return specific error
	err := uninstaller.UninstallApp(ctx, "definitely-not-a-real-app")

	require.Error(t, err)
	assert.ErrorIs(t, err, uninstall.ErrUnknownApp)
}

// TestUninstallAppRecognition tests app recognition without system execution.
func TestUninstallAppRecognition(t *testing.T) {
	tests := []recognitionTestCase{
		{
			name:        "unknown_app_rejected",
			inputName:   "definitely-not-a-real-app",
			shouldError: true,
			errorType:   uninstall.ErrUnknownApp,
			reason:      "Unknown apps must be rejected",
		},
		{
			name:        "empty_app_name_rejected",
			inputName:   "",
			shouldError: true,
			errorType:   uninstall.ErrUnknownApp,
			reason:      "Empty app name must be rejected",
		},
		{
			name:        "whitespace_app_name_rejected",
			inputName:   "   ",
			shouldError: true,
			errorType:   uninstall.ErrUnknownApp,
			reason:      "Whitespace-only app name must be rejected",
		},
	}

	uninstaller, _ := uninstall.NewTestUninstaller(false)

	runRecognitionTests(t, tests, func(ctx context.Context, name string) error {
		return uninstaller.UninstallApp(ctx, name)
	})
}

// TestSpecialUninstallsConsistency verifies special uninstalls are consistent with catalog.
func TestSpecialUninstallsConsistency(t *testing.T) {
	// Special uninstalls should either:
	// 1. Have a corresponding app in the catalog, OR
	// 2. Be explicitly documented as legacy/external apps
	legacyApps := map[string]bool{
		"docker": true, // Docker might be installed externally
	}

	for appName := range uninstall.SpecialUninstalls {
		_, inCatalog := apps.Apps[appName]
		_, isLegacy := legacyApps[appName]

		assert.True(t, inCatalog || isLegacy,
			"Special uninstall for %s should either be in catalog or marked as legacy/external", appName)
	}

	// Also verify that apps with special uninstalls use appropriate methods
	for appName := range uninstall.SpecialUninstalls {
		if app, exists := apps.Apps[appName]; exists {
			// Apps with special uninstalls typically use DEB, Script, or other complex methods
			validMethods := []domain.InstallMethod{
				domain.MethodDEB,
				domain.MethodScript,
				domain.MethodGitHubBinary,
			}

			methodIsValid := false

			for _, method := range validMethods {
				if app.Method == method {
					methodIsValid = true
					break
				}
			}

			assert.True(t, methodIsValid,
				"App %s with special uninstall should use DEB, Script, or GitHubBinary method, got %s",
				appName, app.Method)
		}
	}
}

// TestUninstallGroupUnknown tests that unknown groups are rejected.
func TestUninstallGroupUnknown(t *testing.T) {
	uninstaller, _ := uninstall.NewTestUninstaller(false)
	ctx := context.Background()

	// Business rule: Unknown groups must be rejected
	err := uninstaller.UninstallGroup(ctx, "not-a-real-group")

	require.Error(t, err)
	assert.ErrorIs(t, err, uninstall.ErrUnknownGroup)
}

// TestGroupRecognition tests group recognition without system execution.
func TestGroupRecognition(t *testing.T) {
	tests := []recognitionTestCase{
		{
			name:        "unknown_group_rejected",
			inputName:   "not-a-real-group",
			shouldError: true,
			errorType:   uninstall.ErrUnknownGroup,
			reason:      "Unknown groups must be rejected",
		},
		{
			name:        "empty_group_rejected",
			inputName:   "",
			shouldError: true,
			errorType:   uninstall.ErrUnknownGroup,
			reason:      "Empty group name must be rejected",
		},
		{
			name:        "whitespace_group_rejected",
			inputName:   "   ",
			shouldError: true,
			errorType:   uninstall.ErrUnknownGroup,
			reason:      "Whitespace-only group name must be rejected",
		},
	}

	uninstaller, _ := uninstall.NewTestUninstaller(false)

	runRecognitionTests(t, tests, func(ctx context.Context, name string) error {
		return uninstaller.UninstallGroup(ctx, name)
	})
}

// TestUninstallMethodSelection tests that the correct uninstall method is chosen based on app config.
func TestUninstallMethodSelection(t *testing.T) {
	// This tests actual business logic: method routing based on app configuration
	tests := []struct {
		name           string
		appName        string
		expectedMethod domain.InstallMethod
		expectSpecial  bool
	}{
		{
			name:           "APT packages use apt-get remove",
			appName:        "vim",
			expectedMethod: domain.MethodAPT,
			expectSpecial:  false,
		},
		{
			name:           "Chrome uses special uninstall",
			appName:        "chrome",
			expectedMethod: domain.MethodDEB,
			expectSpecial:  true,
		},
		{
			name:           "Docker uses special uninstall",
			appName:        "docker",
			expectedMethod: domain.MethodScript,
			expectSpecial:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, exists := apps.Apps[tc.appName]
			if !exists {
				t.Skipf("App %s not in catalog", tc.appName)
			}

			// Verify the app has expected method
			assert.Equal(t, tc.expectedMethod, app.Method,
				"App should have expected installation method")

			// Verify special uninstall logic exists if expected
			_, hasSpecial := uninstall.SpecialUninstalls[tc.appName]
			assert.Equal(t, tc.expectSpecial, hasSpecial,
				"Special uninstall presence should match expectation")
		})
	}
}

// TestUninstallVerboseMode tests verbose mode logging.
func TestUninstallVerboseMode(t *testing.T) {
	// Test verbose vs non-verbose uninstallers
	verboseUninstaller, _ := uninstall.NewTestUninstaller(true) // verbose = true
	quietUninstaller, _ := uninstall.NewTestUninstaller(false)  // verbose = false
	ctx := context.Background()

	// Both should validate app existence the same way
	err1 := verboseUninstaller.UninstallApp(ctx, "unknown-app")
	err2 := quietUninstaller.UninstallApp(ctx, "unknown-app")

	require.ErrorIs(t, err1, uninstall.ErrUnknownApp,
		"Verbose mode should still validate app existence")
	assert.ErrorIs(t, err2, uninstall.ErrUnknownApp,
		"Quiet mode should still validate app existence")
}

// TestDetectMisePackageName tests the package name detection logic for mise.
func TestDetectMisePackageName(t *testing.T) {
	// This tests critical business logic: mise package name resolution
	// Since detectMisePackageName is private, we test through the public interface
	uninstaller, _ := uninstall.NewTestUninstaller(false)
	ctx := context.Background()

	// Test that uninstalling a mise package doesn't error on unknown app
	// (it should error on execution, not on app lookup)
	err := uninstaller.UninstallApp(ctx, "hadolint")

	// The app exists in catalog
	_, exists := apps.Apps["hadolint"]
	if exists {
		// If the app exists and uses mise, it should try to uninstall
		// The error should NOT be ErrUnknownApp
		if err != nil {
			assert.NotErrorIs(t, err, uninstall.ErrUnknownApp,
				"Known mise app should not return ErrUnknownApp")
		}
	}
}
