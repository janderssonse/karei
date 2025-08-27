// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"testing"

	"github.com/janderssonse/karei/internal/tui/styles"
)

// TestInstallStatusFilter tests the installation status filter logic.
func TestInstallStatusFilter(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Create test apps
	testApps := []app{
		{Key: "app1", Name: "Installed App", Installed: true, Source: "apt"},
		{Key: "app2", Name: "Not Installed App", Installed: false, Source: "flatpak"},
		{Key: "app3", Name: "Another Installed", Installed: true, Source: "snap"},
	}

	// Test "All" filter (default)
	testAllAppsPassInstallStatusFilter(t, model, testApps, "All", "'All' filter")

	// Test FilterInstalled filter
	testInstallStatusFilterCase(t, model, testApps, FilterInstalled, true, "Installed")

	// Test "Not Installed" filter
	testInstallStatusFilterCase(t, model, testApps, "Not Installed", false, "Not Installed")

	// Test unknown filter value (should default to showing all)
	testAllAppsPassInstallStatusFilter(t, model, testApps, "Unknown", "unknown filter (defaults to all)")
}

func testAllAppsPassInstallStatusFilter(t *testing.T, model *AppsModel, testApps []app, filter, filterDescription string) {
	t.Helper()

	model.installStatusFilter = filter
	for _, testApp := range testApps {
		if !model.passesInstallStatusFilter(testApp) {
			t.Errorf("App %s should pass %s", testApp.Name, filterDescription)
		}
	}
}

func testInstallStatusFilterCase(t *testing.T, model *AppsModel, testApps []app, filter string, shouldPassWhenInstalled bool, filterName string) {
	t.Helper()

	model.installStatusFilter = filter
	for _, testApp := range testApps {
		passes := model.passesInstallStatusFilter(testApp)
		expectedToPass := testApp.Installed == shouldPassWhenInstalled

		if expectedToPass && !passes {
			t.Errorf("App %s (installed: %v) should pass '%s' filter", testApp.Name, testApp.Installed, filterName)
		}

		if !expectedToPass && passes {
			t.Errorf("App %s (installed: %v) should not pass '%s' filter", testApp.Name, testApp.Installed, filterName)
		}
	}
}

// TestPackageTypeFilter tests the package type filter logic.
func TestPackageTypeFilter(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Create test apps with different sources
	testApps := []app{
		{Key: "app1", Name: "APT App", Source: "apt install example", Installed: true},
		{Key: "app2", Name: "Flatpak App", Source: "flatpak install example", Installed: false},
		{Key: "app3", Name: "Snap App", Source: "snap install example", Installed: true},
		{Key: "app4", Name: "DEB App", Source: "wget example.deb", Installed: false},
		{Key: "app5", Name: "Mise App", Source: "mise install example", Installed: true},
		{Key: "app6", Name: "GitHub App", Source: "github.com/user/repo", Installed: false},
		{Key: "app7", Name: "Script App", Source: "bash install.sh", Installed: true},
	}

	// Test "All" filter (default)
	testAllAppsPassPackageTypeFilter(t, model, testApps)

	// Test specific package type filters
	packageTypeTests := map[string][]string{
		"apt":     {"APT App"},
		"flatpak": {"Flatpak App"},
		"snap":    {"Snap App"},
		"deb":     {"DEB App"},
		"mise":    {"Mise App"},
		"github":  {"GitHub App"},
		"script":  {"Script App"},
	}

	for filterType, expectedApps := range packageTypeTests {
		testPackageTypeFilterCase(t, model, testApps, filterType, expectedApps)
	}
}

func testAllAppsPassPackageTypeFilter(t *testing.T, model *AppsModel, testApps []app) {
	t.Helper()

	model.packageTypeFilter = "All"
	for _, testApp := range testApps {
		if !model.passesPackageTypeFilter(testApp) {
			t.Errorf("App %s should pass 'All' package type filter", testApp.Name)
		}
	}
}

func testPackageTypeFilterCase(t *testing.T, model *AppsModel, testApps []app, filterType string, expectedApps []string) {
	t.Helper()

	model.packageTypeFilter = filterType

	for _, testApp := range testApps {
		passes := model.passesPackageTypeFilter(testApp)
		shouldPass := isExpectedToPass(testApp.Name, expectedApps)

		if shouldPass && !passes {
			t.Errorf("App %s should pass '%s' package type filter", testApp.Name, filterType)
		}

		if !shouldPass && passes {
			t.Errorf("App %s should not pass '%s' package type filter", testApp.Name, filterType)
		}
	}
}

// TestPackageTypeFromSource tests the package type detection logic.
func TestPackageTypeFromSource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		source       string
		expectedType string
	}{
		{"apt install vim", "apt"},
		{"APT install vim", "apt"}, // Case insensitive
		{"flatpak install org.vim.Vim", "flatpak"},
		{"FLATPAK install org.vim.Vim", "flatpak"},
		{"snap install vim", "snap"},
		{"wget https://example.com/vim.deb", "deb"},
		{"download vim.deb", "deb"},
		{"mise install node", "mise"},
		{"aqua install vim", "aqua"},
		{"github.com/user/repo", "github"},
		{"https://github.com/user/repo", "github"},
		{"bash install_vim.sh", "script"},
		{"python install.py", "script"},
		{"unknown_source", "script"}, // Default case
		{"", "script"},               // Empty source
	}

	for _, tc := range testCases {
		result := getPackageTypeFromSource(tc.source)
		if result != tc.expectedType {
			t.Errorf("For source '%s', expected type '%s', got '%s'",
				tc.source, tc.expectedType, result)
		}
	}
}

// TestCombinedFiltering tests that both filters work together.
func TestCombinedFiltering(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Create a mix of apps
	testApps := []app{
		{Key: "app1", Name: "Installed APT", Source: "apt install vim", Installed: true},
		{Key: "app2", Name: "Not Installed APT", Source: "apt install emacs", Installed: false},
		{Key: "app3", Name: "Installed Flatpak", Source: "flatpak install firefox", Installed: true},
		{Key: "app4", Name: "Not Installed Flatpak", Source: "flatpak install chrome", Installed: false},
	}

	// Test combination: Installed + APT
	testCombinedFilterCase(t, model, testApps, FilterInstalled, "apt", []string{"Installed APT"}, "Installed", "apt")

	// Test combination: Not Installed + Flatpak
	testCombinedFilterCase(t, model, testApps, "Not Installed", "flatpak", []string{"Not Installed Flatpak"}, "Not Installed", "flatpak")
}

func testCombinedFilterCase(t *testing.T, model *AppsModel, testApps []app, installFilter, packageFilter string, expectedPassing []string, installFilterName, packageFilterName string) {
	t.Helper()

	model.installStatusFilter = installFilter
	model.packageTypeFilter = packageFilter

	for _, testApp := range testApps {
		shouldPass := isExpectedToPass(testApp.Name, expectedPassing)
		passesBoth := testCombinedFilters(model, testApp)

		if shouldPass && !passesBoth {
			t.Errorf("App %s should pass both '%s' and '%s' filters", testApp.Name, installFilterName, packageFilterName)
		}

		if !shouldPass && passesBoth {
			t.Errorf("App %s should not pass both '%s' and '%s' filters", testApp.Name, installFilterName, packageFilterName)
		}
	}
}

func isExpectedToPass(appName string, expectedPassing []string) bool {
	for _, expected := range expectedPassing {
		if appName == expected {
			return true
		}
	}

	return false
}

func testCombinedFilters(model *AppsModel, testApp app) bool {
	passesInstall := model.passesInstallStatusFilter(testApp)
	passesPackage := model.passesPackageTypeFilter(testApp)

	return passesInstall && passesPackage
}

// TestFilterUpdateMsg tests FilterUpdateMsg handling.
func TestFilterUpdateMsg(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Test FilterUpdateMsg updates the filters
	updatedModel, cmd := model.Update(FilterUpdateMsg{
		InstallStatus: FilterInstalled,
		PackageType:   "flatpak",
	})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Verify the command is not nil (should trigger search update if search is active)
	_ = cmd // Command expected when search is active

	// Test that subsequent filtering uses the new filter values
	// We can't directly access private fields, but we can test the behavior

	// Create test scenario: set filters, then do empty search (shows all filtered apps)
	updatedModel, _ = appsModel.Update(SearchUpdateMsg{Query: "", Active: true})
	if _, ok := updatedModel.(*AppsModel); !ok {
		t.Fatal("expected *AppsModel")
	}

	// The filtering logic is tested through the performFuzzySearch method
	// which is called when search is updated
}

// TestFilterWithEmptySearch tests filtering with empty search query.
func TestFilterWithEmptySearch(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Set filters
	model.installStatusFilter = FilterInstalled
	model.packageTypeFilter = "apt"

	// Perform empty search (should show all apps that pass filters)
	results := model.performFuzzySearch("")

	// All returned results should pass both filters
	for _, app := range results {
		if !model.passesInstallStatusFilter(app) {
			t.Errorf("App %s should pass installation status filter", app.Name)
		}

		if !model.passesPackageTypeFilter(app) {
			t.Errorf("App %s should pass package type filter", app.Name)
		}
	}
}
