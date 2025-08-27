// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// Test constants.
const (
	operationInstall   = "install"
	operationUninstall = "uninstall"
)

func TestUninstallMode_StateTransitions(t *testing.T) {
	t.Parallel()

	// Check if we have apps available for testing
	testModel := NewTestAppsModel(styles.New(), 80, 40)
	testModel.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	if len(testModel.categories) == 0 {
		t.Skip("No categories available for testing")
	}

	// Test state transitions
	transitions := []struct {
		name           string
		initialState   SelectionState
		keyPress       string
		expectedState  SelectionState
		expectedVisual string
		description    string
	}{
		{
			name:           "None to Install",
			initialState:   StateNone,
			keyPress:       " ",
			expectedState:  StateInstall,
			expectedVisual: StatusSelected,
			description:    "Space marks for installation",
		},
		{
			name:           "None to Uninstall",
			initialState:   StateNone,
			keyPress:       "d",
			expectedState:  StateUninstall,
			expectedVisual: StatusUninstall,
			description:    "d marks for uninstallation",
		},
		{
			name:           "Install to Uninstall",
			initialState:   StateInstall,
			keyPress:       "d",
			expectedState:  StateUninstall,
			expectedVisual: StatusUninstall,
			description:    "d switches from install to uninstall",
		},
		{
			name:           "Uninstall to Install",
			initialState:   StateUninstall,
			keyPress:       " ",
			expectedState:  StateInstall,
			expectedVisual: StatusSelected,
			description:    "Space switches from uninstall to install",
		},
		{
			name:           "Install to None via Space",
			initialState:   StateInstall,
			keyPress:       " ",
			expectedState:  StateNone,
			expectedVisual: StatusNotInstalled,
			description:    "Space toggles install off",
		},
	}

	for _, transition := range transitions {
		t.Run(transition.name, func(t *testing.T) {
			t.Parallel()
			// Create fresh model instance for each test case - eliminates race conditions
			model := NewTestAppsModel(styles.New(), 80, 40)
			model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

			// Get the current model's first app and ensure it's marked as not installed for consistent testing
			currentApp := model.categories[0].apps[0]
			currentApp.Installed = false

			// Set initial state
			if transition.initialState == StateNone {
				delete(model.selected, currentApp.Key)
			} else {
				model.selected[currentApp.Key] = transition.initialState
			}

			// Position cursor on first app
			model.currentCat = 0
			model.categories[0].currentApp = 0

			// Simulate key press
			var cmd tea.Cmd

			switch transition.keyPress {
			case " ":
				model.toggleInstallSelection()
			case "d":
				model.markForUninstall()
			}

			// Check final state
			actualState := model.selected[currentApp.Key]
			if actualState != transition.expectedState {
				t.Errorf("%s: expected state %v, got %v", transition.description, transition.expectedState, actualState)
			}

			// Check visual indicator
			visual := model.getAppIndicator(currentApp, actualState)
			if visual != transition.expectedVisual {
				t.Errorf("%s: expected visual %s, got %s", transition.description, transition.expectedVisual, visual)
			}

			_ = cmd // Satisfy linter
		})
	}
}

func TestUninstallMode_MixedOperations(t *testing.T) {
	t.Parallel()

	model := setupMixedOperationsTest(t)
	operations := model.getSelectedOperations()

	verifyOperationCounts(t, operations)
	verifyOperationDetails(t, operations)
}

func TestUninstallMode_CategoryStatistics(t *testing.T) {
	t.Parallel()

	model := NewTestAppsModel(styles.New(), 80, 40)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	if len(model.categories) == 0 || len(model.categories[0].apps) < 3 {
		t.Skip("Need at least 3 apps for category statistics testing")
	}

	category := model.categories[0]
	apps := category.apps[:3]

	// Mark one for install, one for uninstall, leave one unselected
	model.selected[apps[0].Key] = StateInstall
	model.selected[apps[1].Key] = StateUninstall
	// apps[2] remains unselected (StateNone)

	// Render category to verify statistics
	categoryView := model.renderCategory(category, true)

	// Check that the header contains the correct counts
	// Format should be: "─ CategoryName [+1/-1 of X]"
	if !strings.Contains(categoryView, "+1") {
		t.Error("Category header should show +1 for install count")
	}

	if !strings.Contains(categoryView, "-1") {
		t.Error("Category header should show -1 for uninstall count")
	}

	// Verify visual indicators in the rendered output
	installIndicator := model.getAppIndicator(apps[0], StateInstall)
	uninstallIndicator := model.getAppIndicator(apps[1], StateUninstall)
	noneIndicator := model.getAppIndicator(apps[2], StateNone)

	if installIndicator != StatusSelected {
		t.Errorf("Install indicator should be %s, got %s", StatusSelected, installIndicator)
	}

	if uninstallIndicator != StatusUninstall {
		t.Errorf("Uninstall indicator should be %s, got %s", StatusUninstall, uninstallIndicator)
	}

	// apps[2] is "hadolint" which is installed in the test data, so it should show StatusInstalled
	if noneIndicator != StatusInstalled {
		t.Errorf("None indicator for installed app should be %s, got %s", StatusInstalled, noneIndicator)
	}
}

func TestUninstallMode_ProgressScreenIntegration(t *testing.T) {
	t.Parallel()

	operations := createTestOperations()
	progressModel := createTestProgressModel(t, operations)

	verifyProgressTasks(t, progressModel)
	verifyTaskOperations(t, progressModel)
}

func TestUninstallMode_LegacyCompatibility(t *testing.T) {
	t.Parallel()

	model := NewTestAppsModel(styles.New(), 80, 40)

	if len(model.categories) == 0 {
		t.Skip("No categories available for testing")
	}

	// Test legacy getSelectedApps method
	firstApp := model.categories[0].apps[0]

	// Mark one for install, one for uninstall
	if len(model.categories[0].apps) >= 2 {
		secondApp := model.categories[0].apps[1]

		model.selected[firstApp.Key] = StateInstall
		model.selected[secondApp.Key] = StateUninstall

		// Legacy method should only return install selections
		selectedApps := model.getSelectedApps()

		if len(selectedApps) != 1 {
			t.Errorf("Legacy getSelectedApps should return 1 app, got %d", len(selectedApps))
		}

		if selectedApps[0] != firstApp.Key {
			t.Errorf("Legacy getSelectedApps should return install app, got %s", selectedApps[0])
		}
	}

	// Test legacy toggleSelection method (should mark for install)
	model.currentCat = 0
	model.categories[0].currentApp = 0

	delete(model.selected, firstApp.Key) // Clear selection
	model.toggleSelection()              // Should mark for install

	if model.selected[firstApp.Key] != StateInstall {
		t.Errorf("Legacy toggleSelection should mark for install, got %v", model.selected[firstApp.Key])
	}
}

func TestUninstallMode_KeyHandling(t *testing.T) {
	t.Parallel()

	// Check if we have apps available for testing
	testModel := NewTestAppsModel(styles.New(), 80, 40)
	testModel.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	if len(testModel.categories) == 0 {
		t.Skip("No categories available for testing")
	}

	// Test key message handling
	keyTests := []struct {
		key           string
		expectedState SelectionState
		description   string
	}{
		{" ", StateInstall, "Space key should mark for install"},
		{"d", StateUninstall, "d key should mark for uninstall"},
	}

	for _, keyTest := range keyTests {
		t.Run(keyTest.description, func(t *testing.T) {
			t.Parallel()
			// Create fresh model for each test
			model := NewTestAppsModel(styles.New(), 80, 40)
			model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

			// Position on first app
			model.currentCat = 0
			model.categories[0].currentApp = 0
			firstApp := model.categories[0].apps[0]

			// Clear selection
			delete(model.selected, firstApp.Key)

			// Create key message
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(keyTest.key[0])}}

			// Handle selection keys
			model.handleSelectionKeys(keyMsg)

			// Check result
			actualState := model.selected[firstApp.Key]
			if actualState != keyTest.expectedState {
				t.Errorf("%s: expected state %v, got %v", keyTest.description, keyTest.expectedState, actualState)
			}
		})
	}
}

func TestUninstallMode_SpaceToggleBehavior(t *testing.T) {
	t.Parallel()

	// Check if we have apps available for testing
	testModel := NewTestAppsModel(styles.New(), 80, 40)
	testModel.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	if len(testModel.categories) == 0 {
		t.Skip("No categories available for testing")
	}

	// Get first app reference for testing
	firstApp := testModel.categories[0].apps[0]

	// Test Space toggle cycle: None -> Install -> None
	t.Run("Space Toggle Cycle", func(t *testing.T) {
		t.Parallel()
		// Create fresh model for this test
		model := NewTestAppsModel(styles.New(), 80, 40)
		model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
		model.currentCat = 0
		model.categories[0].currentApp = 0

		// Start with no selection
		delete(model.selected, firstApp.Key)

		// First space: None -> Install
		model.toggleInstallSelection()

		if model.selected[firstApp.Key] != StateInstall {
			t.Errorf("First Space should mark for install, got %v", model.selected[firstApp.Key])
		}

		// Second space: Install -> None
		model.toggleInstallSelection()

		if model.selected[firstApp.Key] != StateNone {
			t.Errorf("Second Space should deselect, got %v", model.selected[firstApp.Key])
		}

		// Third space: None -> Install (cycle repeats)
		model.toggleInstallSelection()

		if model.selected[firstApp.Key] != StateInstall {
			t.Errorf("Third Space should mark for install again, got %v", model.selected[firstApp.Key])
		}
	})

	// Test Space behavior from uninstall state
	t.Run("Space From Uninstall State", func(t *testing.T) {
		t.Parallel()
		// Create fresh model for this test
		model := NewTestAppsModel(styles.New(), 80, 40)
		model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
		model.currentCat = 0
		model.categories[0].currentApp = 0

		// Set to uninstall state
		model.selected[firstApp.Key] = StateUninstall

		// Space should switch to install
		model.toggleInstallSelection()

		if model.selected[firstApp.Key] != StateInstall {
			t.Errorf("Space from uninstall should switch to install, got %v", model.selected[firstApp.Key])
		}
	})

	// Test that 'd' key behavior remains unchanged
	t.Run("d Key Still Works", func(t *testing.T) {
		t.Parallel()
		// Create fresh model for this test
		model := NewTestAppsModel(styles.New(), 80, 40)
		model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
		model.currentCat = 0
		model.categories[0].currentApp = 0

		// Clear selection
		delete(model.selected, firstApp.Key)

		// d should mark for uninstall
		model.markForUninstall()

		if model.selected[firstApp.Key] != StateUninstall {
			t.Errorf("d key should mark for uninstall, got %v", model.selected[firstApp.Key])
		}

		// d on already uninstall should keep uninstall (no-op)
		model.markForUninstall()

		if model.selected[firstApp.Key] != StateUninstall {
			t.Errorf("d key on uninstall should keep uninstall, got %v", model.selected[firstApp.Key])
		}
	})
}

// setupMixedOperationsTest creates a model with mixed install/uninstall operations.
func setupMixedOperationsTest(t *testing.T) *AppsModel {
	t.Helper()

	model := NewTestAppsModel(styles.New(), 80, 40)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	if len(model.categories) == 0 || len(model.categories[0].apps) < 4 {
		t.Skip("Need at least 4 apps for mixed operations testing")
	}

	// Mark apps for different operations
	apps := model.categories[0].apps[:4]

	// Mark first two for install
	model.selected[apps[0].Key] = StateInstall
	model.selected[apps[1].Key] = StateInstall

	// Mark last two for uninstall
	model.selected[apps[2].Key] = StateUninstall
	model.selected[apps[3].Key] = StateUninstall

	return model
}

// verifyOperationCounts checks the expected counts of install and uninstall operations.
func verifyOperationCounts(t *testing.T, operations []SelectedOperation) {
	t.Helper()

	installCount := 0
	uninstallCount := 0

	for _, op := range operations {
		switch op.Operation {
		case StateInstall:
			installCount++
		case StateUninstall:
			uninstallCount++
		}
	}

	if installCount != 2 {
		t.Errorf("Expected 2 install operations, got %d", installCount)
	}

	if uninstallCount != 2 {
		t.Errorf("Expected 2 uninstall operations, got %d", uninstallCount)
	}

	expectedTotal := 4
	if len(operations) != expectedTotal {
		t.Errorf("Expected %d total operations, got %d", expectedTotal, len(operations))
	}
}

// verifyOperationDetails checks that each operation has valid details.
func verifyOperationDetails(t *testing.T, operations []SelectedOperation) {
	t.Helper()

	for _, operation := range operations {
		if operation.AppKey == "" {
			t.Error("Operation should have app key")
		}

		if operation.AppName == "" {
			t.Error("Operation should have app name")
		}

		if operation.Operation != StateInstall && operation.Operation != StateUninstall {
			t.Errorf("Operation should be install or uninstall, got %v", operation.Operation)
		}
	}
}

// createTestOperations creates a standard set of mixed operations for testing.
func createTestOperations() []SelectedOperation {
	return []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
		{AppKey: "vscode", Operation: StateInstall, AppName: "Visual Studio Code"},
		{AppKey: "firefox", Operation: StateUninstall, AppName: "Firefox"},
		{AppKey: "chrome", Operation: StateUninstall, AppName: "Chrome"},
	}
}

// createTestProgressModel creates and validates a progress model.
func createTestProgressModel(t *testing.T, operations []SelectedOperation) *Progress {
	t.Helper()

	progressModel := NewProgressWithOperations(context.Background(), styles.New(), operations)

	if progressModel == nil {
		t.Fatal("Progress model should not be nil")
	}

	return progressModel
}

// verifyProgressTasks checks that the correct number of tasks were created.
func verifyProgressTasks(t *testing.T, progressModel *Progress) {
	t.Helper()

	if len(progressModel.tasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(progressModel.tasks))
	}
}

// verifyTaskOperations checks task operations and descriptions.
func verifyTaskOperations(t *testing.T, progressModel *Progress) {
	t.Helper()

	installTasks := 0
	uninstallTasks := 0

	for _, task := range progressModel.tasks {
		switch task.Operation {
		case operationInstall:
			installTasks++

			if !strings.Contains(task.Description, "Installing") {
				t.Errorf("Install task should contain 'Installing', got: %s", task.Description)
			}
		case operationUninstall:
			uninstallTasks++

			if !strings.Contains(task.Description, "Uninstalling") {
				t.Errorf("Uninstall task should contain 'Uninstalling', got: %s", task.Description)
			}
		default:
			t.Errorf("Unexpected operation type: %s", task.Operation)
		}
	}

	if installTasks != 2 {
		t.Errorf("Expected 2 install tasks, got %d", installTasks)
	}

	if uninstallTasks != 2 {
		t.Errorf("Expected 2 uninstall tasks, got %d", uninstallTasks)
	}
}
