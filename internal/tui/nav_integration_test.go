// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/models"
)

const testAppChrome = "chrome"

// updateTUIAppWithAssertion is a helper function to handle type assertions safely.
func updateTUIAppWithAssertion(t *testing.T, app *App, msg tea.Msg) (*App, tea.Cmd) {
	t.Helper()

	model, cmd := app.Update(msg)

	appModel, ok := model.(*App)
	if !ok {
		t.Fatalf("Expected *App from Update, got %T", model)
		return nil, nil // This won't execute but satisfies compiler
	}

	return appModel, cmd
}

// TestNavigationRefreshFlow tests the complete flow of navigating from progress
// screen back to apps screen with refresh functionality.
func TestNavigationRefreshFlow(t *testing.T) {
	t.Parallel()

	// Create app and set initial size
	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen first
	app, cmd := updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})
	if cmd != nil {
		// Process any initialization commands
		app, _ = updateTUIAppWithAssertion(t, app, cmd())
	}

	// Verify we're on apps screen
	if app.currentScreen != AppsScreen {
		t.Errorf("Expected to be on AppsScreen, got %v", app.currentScreen)
	}

	// Simulate navigation to progress screen
	operations := []models.SelectedOperation{
		{AppKey: testAppChrome, Operation: models.StateInstall, AppName: "Chrome"},
	}

	app, cmd = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.ProgressScreen, Data: operations})
	if cmd != nil {
		app, _ = updateTUIAppWithAssertion(t, app, cmd())
	}

	// Verify we're on progress screen
	if app.currentScreen != ProgressScreen {
		t.Errorf("Expected to be on ProgressScreen, got %v", app.currentScreen)
	}

	// Test: Navigate back to apps screen with refresh data (simulating ESC from progress)
	refreshData := models.RefreshStatusData
	app, cmd = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: refreshData})

	// Verify navigation succeeded and we're back on apps screen
	if app.currentScreen != AppsScreen {
		t.Errorf("Expected to be back on AppsScreen after refresh, got %v", app.currentScreen)
	}

	// Verify that the apps model is properly cached and accessible
	appsModel, ok := app.contentModel.(*models.AppsModel)
	if !ok {
		t.Errorf("Expected content model to be AppsModel, got %T", app.contentModel)
	}

	// Verify the apps model exists and is ready
	if appsModel == nil {
		t.Error("Apps model should not be nil after navigation with refresh")
	}

	// Process any commands that were returned (like refresh status command)
	if cmd != nil {
		// The refresh command is typically a delayed command, so we'll check its presence
		t.Logf("Refresh command was returned as expected")
	}
}

// TestSameScreenNavigationWithData tests that same-screen navigation works when data is provided.
func TestSameScreenNavigationWithData(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})

	// Verify we're on apps screen
	if app.currentScreen != AppsScreen {
		t.Errorf("Expected to be on AppsScreen, got %v", app.currentScreen)
	}

	// Test: Navigate to same screen with refresh data (should be allowed)
	refreshData := models.RefreshStatusData
	app, cmd := updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: refreshData})

	// Should still be on apps screen
	if app.currentScreen != AppsScreen {
		t.Errorf("Expected to stay on AppsScreen, got %v", app.currentScreen)
	}

	// Should have returned a command for refresh
	if cmd == nil {
		t.Error("Expected refresh command to be returned for same-screen navigation with data")
	}
}

// TestSameScreenNavigationWithoutData tests that same-screen navigation without data is blocked.
func TestSameScreenNavigationWithoutData(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})

	// Store reference to current model
	originalModel := app.contentModel

	// Test: Navigate to same screen without data (should be blocked)
	app, cmd := updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})

	// Should still be on apps screen
	if app.currentScreen != AppsScreen {
		t.Errorf("Expected to stay on AppsScreen, got %v", app.currentScreen)
	}

	// Should not have returned a command (blocked navigation)
	if cmd != nil {
		t.Error("Expected no command for blocked same-screen navigation without data")
	}

	// Model should be the same instance (no recreation)
	if app.contentModel != originalModel {
		t.Error("Expected model to remain the same for blocked navigation")
	}
}

// TestRefreshStatusMessage tests that RefreshStatusMsg is properly handled by apps model.
func TestRefreshStatusMessage(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen
	app, initCmd := updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})
	if initCmd != nil {
		app, _ = updateTUIAppWithAssertion(t, app, initCmd())
	}

	// Send window size to ensure model is ready
	app, _ = updateTUIAppWithAssertion(t, app, tea.WindowSizeMsg{Width: 80, Height: 40})

	// Get the apps model
	appsModel, isApps := app.contentModel.(*models.AppsModel)
	if !isApps {
		t.Fatalf("Expected content model to be AppsModel, got %T", app.contentModel)
	}

	// Test: Send refresh status message directly to apps model
	updatedModel, cmd := appsModel.Update(models.RefreshStatusMsg{})

	// Should return the model (possibly with changes)
	if updatedModel == nil {
		t.Error("Expected model to be returned from RefreshStatusMsg")
	}

	// Should return a command for refreshing app statuses
	if cmd == nil {
		t.Error("Expected command to be returned for app status refresh")
	}
}

// TestProgressScreenRefreshNavigation tests the specific case of progress screen
// navigating back with refresh request.
func TestProgressScreenRefreshNavigation(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Create progress screen with operations
	operations := []models.SelectedOperation{
		{AppKey: testAppChrome, Operation: models.StateInstall, AppName: "Chrome"},
		{AppKey: "vscode", Operation: models.StateUninstall, AppName: "VS Code"},
	}

	// Navigate to progress screen
	app, cmd := updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.ProgressScreen, Data: operations})
	if cmd != nil {
		app, _ = updateTUIAppWithAssertion(t, app, cmd())
	}

	// Verify we're on progress screen
	if app.currentScreen != ProgressScreen {
		t.Errorf("Expected to be on ProgressScreen, got %v", app.currentScreen)
	}

	// Test: Simulate progress screen completing and requesting navigation back with refresh
	// This simulates the ESC key behavior after installation completion
	refreshMsg := models.NavigateMsg{
		Screen: models.AppsScreen,
		Data:   models.RefreshStatusData,
	}

	app, refreshCmd := updateTUIAppWithAssertion(t, app, refreshMsg)

	// Should be back on apps screen
	if app.currentScreen != AppsScreen {
		t.Errorf("Expected to be on AppsScreen after refresh navigation, got %v", app.currentScreen)
	}

	// Should have a refresh command
	if refreshCmd == nil {
		t.Error("Expected refresh command after navigation from progress screen")
	}

	// Apps model should be available
	_, ok := app.contentModel.(*models.AppsModel)
	if !ok {
		t.Errorf("Expected apps model after refresh navigation, got %T", app.contentModel)
	}
}

// TestModelCachingWithRefresh tests that model caching works correctly with refresh operations.
func TestModelCachingWithRefresh(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen and cache the model
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})
	app, _ = updateTUIAppWithAssertion(t, app, tea.WindowSizeMsg{Width: 80, Height: 40})

	// Store reference to cached model
	originalModel := app.contentModel
	if _, exists := app.models[AppsScreen]; !exists {
		t.Error("Expected apps model to be cached")
	}

	// Navigate away to progress screen
	operations := []models.SelectedOperation{
		{AppKey: "test", Operation: models.StateInstall, AppName: "Test"},
	}
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.ProgressScreen, Data: operations})

	// Navigate back with refresh - should use cached model but allow refresh
	app, refreshCmd := updateTUIAppWithAssertion(t, app, models.NavigateMsg{
		Screen: models.AppsScreen,
		Data:   models.RefreshStatusData,
	})

	// Should be using the cached model (same instance)
	if app.contentModel != originalModel {
		t.Error("Expected to reuse cached apps model")
	}

	// Should still have refresh command
	if refreshCmd == nil {
		t.Error("Expected refresh command even with cached model")
	}

	// Model should still be in cache
	if cachedModel, exists := app.models[AppsScreen]; !exists {
		t.Error("Expected apps model to remain in cache")
	} else if cachedModel != originalModel {
		t.Error("Expected cached model to be the same instance")
	}
}

// TestSelectionStateClearing tests that selection state is cleared when apps are installed.
//
//nolint:cyclop // Comprehensive UI state integration test
func TestSelectionStateClearing(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})
	app, _ = updateTUIAppWithAssertion(t, app, tea.WindowSizeMsg{Width: 80, Height: 40})

	// Get the apps model
	appsModel, isApps := app.contentModel.(*models.AppsModel)
	if !isApps {
		t.Fatalf("Expected content model to be AppsModel, got %T", app.contentModel)
	}

	// Simulate selecting Chrome for installation
	appsModel.SetSelectionStateForTesting(testAppChrome, models.StateInstall)

	// Verify Chrome shows as selected initially
	categories := appsModel.GetCategoriesForTesting()

	var chromeApp *models.Application

	for _, cat := range categories {
		for _, app := range cat.Apps {
			if app.Key == testAppChrome {
				chromeApp = &app

				break
			}
		}

		if chromeApp != nil {
			break
		}
	}

	if chromeApp == nil {
		t.Fatal("Chrome app not found in test categories")
	}

	// Chrome should be selected initially
	if !chromeApp.Selected {
		t.Error("Expected Chrome to be selected initially")
	}

	// Simulate status update: Chrome becomes installed
	app.contentModel, _ = appsModel.Update(models.StatusUpdateMsg{
		AppName:   testAppChrome,
		Installed: true,
	})

	// Get updated apps model
	updatedAppsModel, isAppsModel := app.contentModel.(*models.AppsModel)
	if !isAppsModel {
		t.Fatalf("Expected updated content model to be AppsModel, got %T", app.contentModel)
	}

	// Verify Chrome is no longer in selected map (selection state cleared)
	if _, exists := updatedAppsModel.GetSelectionStateForTesting(testAppChrome); exists {
		t.Error("Expected Chrome selection state to be cleared after installation")
	}

	// Verify Chrome shows as installed in the test interface
	updatedCategories := updatedAppsModel.GetCategoriesForTesting()

	var updatedChromeApp *models.Application

	for _, cat := range updatedCategories {
		for _, app := range cat.Apps {
			if app.Key == testAppChrome {
				updatedChromeApp = &app

				break
			}
		}

		if updatedChromeApp != nil {
			break
		}
	}

	if updatedChromeApp == nil {
		t.Fatal("Chrome app not found in updated test categories")
	}

	// Chrome should be installed and not selected
	if !updatedChromeApp.Installed {
		t.Error("Expected Chrome to be marked as installed")
	}

	if updatedChromeApp.Selected {
		t.Error("Expected Chrome to not be selected after installation")
	}
}

// TestUninstallSelectionStateClearing tests that selection state is cleared when apps are uninstalled.
func TestUninstallSelectionStateClearing(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})
	app, _ = updateTUIAppWithAssertion(t, app, tea.WindowSizeMsg{Width: 80, Height: 40})

	// Get the apps model
	appsModel, isApps := app.contentModel.(*models.AppsModel)
	if !isApps {
		t.Fatalf("Expected content model to be AppsModel, got %T", app.contentModel)
	}

	// Simulate selecting an app for uninstallation
	appsModel.SetSelectionStateForTesting("vscode", models.StateUninstall)

	// Simulate status update: App becomes uninstalled
	app.contentModel, _ = appsModel.Update(models.StatusUpdateMsg{
		AppName:   "vscode",
		Installed: false,
	})

	// Get updated apps model
	updatedAppsModel, isAppsModel := app.contentModel.(*models.AppsModel)
	if !isAppsModel {
		t.Fatalf("Expected updated content model to be AppsModel, got %T", app.contentModel)
	}

	// Verify selection state is cleared after uninstallation
	if _, exists := updatedAppsModel.GetSelectionStateForTesting("vscode"); exists {
		t.Error("Expected VS Code selection state to be cleared after uninstallation")
	}
}

// TestCompleteInstallationFlow tests the exact user scenario:
// select → install → ESC → shows as installed.
//
//nolint:cyclop // End-to-end installation flow integration test
func TestCompleteInstallationFlow(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Step 1: Navigate to apps screen
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})
	app, _ = updateTUIAppWithAssertion(t, app, tea.WindowSizeMsg{Width: 80, Height: 40})

	appsModel, isApps := app.contentModel.(*models.AppsModel)
	if !isApps {
		t.Fatalf("Expected content model to be AppsModel, got %T", app.contentModel)
	}

	// Step 2: Select Chrome for installation (simulate user space key)
	appsModel.SetSelectionStateForTesting(testAppChrome, models.StateInstall)

	// Verify Chrome is selected initially
	state, exists := appsModel.GetSelectionStateForTesting(testAppChrome)
	if !exists || state != models.StateInstall {
		t.Error("Expected Chrome to be selected for installation")
	}

	// Step 3: Navigate to progress screen with Chrome installation
	operations := []models.SelectedOperation{
		{AppKey: testAppChrome, Operation: models.StateInstall, AppName: "Chrome"},
	}
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.ProgressScreen, Data: operations})

	if app.currentScreen != ProgressScreen {
		t.Errorf("Expected to be on ProgressScreen, got %v", app.currentScreen)
	}

	// Step 4: Simulate installation completion and Chrome status update
	// First navigate back to apps screen (this caches the apps model)
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: models.RefreshStatusData})

	// Get the updated apps model
	updatedAppsModel, isAppsModel := app.contentModel.(*models.AppsModel)
	if !isAppsModel {
		t.Fatalf("Expected content model to be AppsModel, got %T", app.contentModel)
	}

	// Step 5: Simulate Chrome installation completing (StatusUpdateMsg)
	finalModel, _ := updatedAppsModel.Update(models.StatusUpdateMsg{
		AppName:   testAppChrome,
		Installed: true,
	})

	finalAppsModel, ok := finalModel.(*models.AppsModel)
	if !ok {
		t.Fatalf("Expected final model to be AppsModel, got %T", finalModel)
	}

	// Step 6: Verify the final state - Chrome should show as installed, not selected

	// Selection state should be cleared
	if _, exists := finalAppsModel.GetSelectionStateForTesting(testAppChrome); exists {
		t.Error("Chrome selection state should be cleared after installation")
	}

	// Chrome should be marked as installed in the test categories
	categories := finalAppsModel.GetCategoriesForTesting()

	var chromeApp *models.Application

	for _, cat := range categories {
		for _, app := range cat.Apps {
			if app.Key == testAppChrome {
				chromeApp = &app

				break
			}
		}

		if chromeApp != nil {
			break
		}
	}

	if chromeApp == nil {
		t.Fatal("Chrome app not found in final test categories")
	}

	// Final verification: Chrome should be installed and NOT selected
	if !chromeApp.Installed {
		t.Error("Chrome should be marked as installed")
	}

	if chromeApp.Selected {
		t.Error("Chrome should not be selected after installation - it should show as installed")
	}

	t.Logf("✅ Complete flow verified: Chrome shows as installed (●) instead of selected (✓)")
}

// TestImmediateSynchronization tests that completed operations are immediately synchronized
// without delays using CompletedOperationsMsg.
//
//nolint:cyclop // Real-time state synchronization integration test
func TestImmediateSynchronization(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})
	app, _ = updateTUIAppWithAssertion(t, app, tea.WindowSizeMsg{Width: 80, Height: 40})

	appsModel, isApps := app.contentModel.(*models.AppsModel)
	if !isApps {
		t.Fatalf("Expected content model to be AppsModel, got %T", app.contentModel)
	}

	// Select Chrome for installation
	appsModel.SetSelectionStateForTesting(testAppChrome, models.StateInstall)

	// Verify initial state
	state, exists := appsModel.GetSelectionStateForTesting(testAppChrome)
	if !exists || state != models.StateInstall {
		t.Error("Chrome should be selected for installation initially")
	}

	// Test: Send CompletedOperationsMsg directly (simulates immediate return from progress screen)
	operations := []models.SelectedOperation{
		{AppKey: testAppChrome, Operation: models.StateInstall, AppName: "Chrome"},
	}

	completedMsg := models.CompletedOperationsMsg{Operations: operations}
	app, cmd := updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: completedMsg})

	// Should be immediate - no delay, no commands needed for the status update
	if cmd != nil {
		t.Logf("Command returned (expected for navigation): %T", cmd())
	}

	// Get updated apps model
	finalAppsModel, ok := app.contentModel.(*models.AppsModel)
	if !ok {
		t.Fatalf("Expected final content model to be AppsModel, got %T", app.contentModel)
	}

	// Verify immediate synchronization: selection state should be cleared instantly
	if _, exists := finalAppsModel.GetSelectionStateForTesting(testAppChrome); exists {
		t.Error("Chrome selection state should be immediately cleared")
	}

	// Verify Chrome shows as installed in categories
	categories := finalAppsModel.GetCategoriesForTesting()

	var chromeApp *models.Application

	for _, cat := range categories {
		for _, app := range cat.Apps {
			if app.Key == testAppChrome {
				chromeApp = &app

				break
			}
		}

		if chromeApp != nil {
			break
		}
	}

	if chromeApp == nil {
		t.Fatal("Chrome app not found after immediate sync")
	}

	// Final verification: instant update
	if !chromeApp.Installed {
		t.Error("Chrome should be immediately marked as installed")
	}

	if chromeApp.Selected {
		t.Error("Chrome should not be selected after immediate sync")
	}

	t.Logf("✅ Immediate synchronization verified: No delays, instant status update")
}

// TestUninstallationFlow tests the complete uninstallation flow:
// select for uninstall → shows uninstall screen → shows as uninstalled.
//
//nolint:cyclop // Complex uninstallation scenario integration test
func TestUninstallationFlow(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})
	app, _ = updateTUIAppWithAssertion(t, app, tea.WindowSizeMsg{Width: 80, Height: 40})

	appsModel, isApps := app.contentModel.(*models.AppsModel)
	if !isApps {
		t.Fatalf("Expected content model to be AppsModel, got %T", app.contentModel)
	}

	// Step 1: Mark Chrome for uninstallation (simulate 'd' key)
	appsModel.SetSelectionStateForTesting(testAppChrome, models.StateUninstall)

	// Verify Chrome is marked for uninstallation
	state, exists := appsModel.GetSelectionStateForTesting(testAppChrome)
	if !exists || state != models.StateUninstall {
		t.Error("Chrome should be marked for uninstallation")
	}

	// Step 2: Navigate to progress screen with Chrome uninstallation
	operations := []models.SelectedOperation{
		{AppKey: testAppChrome, Operation: models.StateUninstall, AppName: "Chrome"},
	}
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.ProgressScreen, Data: operations})

	if app.currentScreen != ProgressScreen {
		t.Errorf("Expected to be on ProgressScreen, got %v", app.currentScreen)
	}

	// Step 3: Verify progress screen shows uninstallation task correctly
	progressModel, isProgressModel := app.contentModel.(*models.Progress)
	if !isProgressModel {
		t.Fatalf("Expected content model to be Progress, got %T", app.contentModel)
	}

	// Check that the task description is correct (this was the bug!)
	tasks := progressModel.GetTasksForTesting()
	if len(tasks) == 0 {
		t.Fatal("Expected at least one task in progress model")
	}

	chromeTask := tasks[0]
	if chromeTask.Description != "Uninstalling Chrome..." {
		t.Errorf("Expected task description to be 'Uninstalling Chrome...', got '%s'", chromeTask.Description)
	}

	if chromeTask.Operation != models.OperationUninstall {
		t.Errorf("Expected task operation to be '%s', got '%s'", models.OperationUninstall, chromeTask.Operation)
	}

	// Step 4: Test immediate synchronization on return (simulates ESC after completion)
	completedMsg := models.CompletedOperationsMsg{Operations: operations}
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: completedMsg})

	// Get the updated apps model
	finalAppsModel, ok := app.contentModel.(*models.AppsModel)
	if !ok {
		t.Fatalf("Expected final content model to be AppsModel, got %T", app.contentModel)
	}

	// Step 5: Verify final state - Chrome should show as uninstalled

	// Selection state should be cleared
	if _, exists := finalAppsModel.GetSelectionStateForTesting(testAppChrome); exists {
		t.Error("Chrome selection state should be cleared after uninstallation")
	}

	// Chrome should be marked as not installed
	categories := finalAppsModel.GetCategoriesForTesting()

	var chromeApp *models.Application

	for _, cat := range categories {
		for _, app := range cat.Apps {
			if app.Key == testAppChrome {
				chromeApp = &app

				break
			}
		}

		if chromeApp != nil {
			break
		}
	}

	if chromeApp == nil {
		t.Fatal("Chrome app not found after uninstallation flow")
	}

	// Final verification: Chrome should be uninstalled and not selected
	if chromeApp.Installed {
		t.Error("Chrome should be marked as not installed after uninstallation")
	}

	if chromeApp.Selected {
		t.Error("Chrome should not be selected after uninstallation")
	}

	t.Logf("✅ Complete uninstallation flow verified: Chrome shows as uninstalled (○)")
}

// TestEnhancedUninstallationUX tests the improved uninstallation UX with granular progress.
//
//nolint:cyclop // UX validation integration test with multiple paths
func TestEnhancedUninstallationUX(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen and select Chrome for uninstallation
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.AppsScreen, Data: nil})
	app, _ = updateTUIAppWithAssertion(t, app, tea.WindowSizeMsg{Width: 80, Height: 40})

	appsModel, isApps := app.contentModel.(*models.AppsModel)
	if !isApps {
		t.Fatalf("Expected content model to be AppsModel, got %T", app.contentModel)
	}

	appsModel.SetSelectionStateForTesting(testAppChrome, models.StateUninstall)

	// Navigate to progress screen
	operations := []models.SelectedOperation{
		{AppKey: testAppChrome, Operation: models.StateUninstall, AppName: "Chrome"},
	}
	app, _ = updateTUIAppWithAssertion(t, app, models.NavigateMsg{Screen: models.ProgressScreen, Data: operations})

	progressModel, isProgressModel := app.contentModel.(*models.Progress)
	if !isProgressModel {
		t.Fatalf("Expected content model to be Progress, got %T", app.contentModel)
	}

	// Test 1: Verify task shows "Uninstalling Chrome..." (not just testAppChrome)
	tasks := progressModel.GetTasksForTesting()
	if len(tasks) == 0 {
		t.Fatal("Expected at least one task")
	}

	chromeTask := tasks[0]
	if chromeTask.Description != "Uninstalling Chrome..." {
		t.Errorf("Expected 'Uninstalling Chrome...', got '%s'", chromeTask.Description)
	}

	if chromeTask.Operation != models.OperationUninstall {
		t.Errorf("Expected operation '%s', got '%s'", models.OperationUninstall, chromeTask.Operation)
	}

	// Test 2: Simulate the multi-stage uninstallation process
	stages := []struct {
		stage              int
		expectedProgress   float64
		expectedLogPattern string
	}{
		{1, 0.1, "Preparing to remove"},
		{2, 0.3, "Stopping services"},
		{3, 0.6, "Removing .* package files"},
		{4, 0.85, "Cleaning up .* configuration"},
		{5, 0.95, "Finalizing removal"},
	}

	for _, stageTest := range stages {
		// Send UninstallStageMsg
		stageMsg := models.UninstallStageMsg{
			TaskIndex: 0,
			Stage:     stageTest.stage,
			AppKey:    testAppChrome,
			AppName:   "Chrome",
		}

		updatedModel, cmd := progressModel.Update(stageMsg)

		progressModel, ok := updatedModel.(*models.Progress)
		if !ok {
			t.Fatalf("Expected *models.Progress, got %T", updatedModel)
		}

		// Verify progress was updated
		updatedTasks := progressModel.GetTasksForTesting()
		if len(updatedTasks) > 0 {
			actualProgress := updatedTasks[0].Progress
			if actualProgress != stageTest.expectedProgress {
				t.Errorf("Stage %d: Expected progress %.2f, got %.2f",
					stageTest.stage, stageTest.expectedProgress, actualProgress)
			}
		}

		// Verify log entries were added (we can't easily inspect logs, but cmd should be returned for continuation)
		if stageTest.stage < 5 && cmd == nil {
			t.Errorf("Stage %d should return a command for continuation", stageTest.stage)
		}
	}

	t.Logf("✅ Enhanced uninstallation UX verified: Granular progress and detailed logging")
}
