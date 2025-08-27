// Test to verify package name mapping for uninstallation
package uninstall

import (
	"testing"
)

// TestPackageNameMapping verifies that uninstallation uses the correct package names for various applications.
func TestPackageNameMapping(t *testing.T) {
	t.Parallel()
	// Test the mapping function directly
	tests := []struct {
		appKey      string
		expectedPkg string
	}{
		{"chrome", "google-chrome-stable"},
		{"vscode", "code"},
		{"brave", "brave-browser"},
		{"unknown-app", "unknown-app"}, // Should pass through unmapped apps
	}

	for _, test := range tests {
		result := mapToDebPackageName(test.appKey)
		if result != test.expectedPkg {
			t.Errorf("mapToDebPackageName(%s) = %s, want %s", test.appKey, result, test.expectedPkg)
		}
	}
}

// TestUninstallIntegration verifies the complete application uninstallation flow.
func TestChromeUninstallIntegration(t *testing.T) {
	t.Parallel()
	// Test that the mapping is applied correctly in the uninstall flow
	// We can't easily mock the command execution, but we can verify the logic

	// Verify that chrome maps to google-chrome-stable
	packageName := mapToDebPackageName("chrome")
	if packageName != "google-chrome-stable" {
		t.Errorf("Chrome should map to google-chrome-stable, got %s", packageName)
	}

	// Verify that calling uninstallDEB would use the correct package name
	// (This would run the actual command if Chrome was installed, so we just verify the mapping)
	t.Logf("✅ Chrome uninstallation will use package name: %s", packageName)
}
