package models

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// TestUninstallReinstallFlow tests the exact user flow that causes the bug:
// 1. Uninstall hadolint
// 2. Complete uninstallation (selection state cleared)
// 3. Mark hadolint for installation (Space key - should show ✓)
// 4. Navigate to password screen (Enter key)
// 5. Password screen should show "Installing" not "Uninstalling".
//
//nolint:cyclop,paralleltest // Comprehensive integration test with complex state validation - no parallel due to state interactions
func TestUninstallReinstallFlow(t *testing.T) {
	// Setup: Create apps model with hadolint
	styleConfig := styles.New()
	appsModel := NewTestAppsModel(styleConfig, 80, 40)
	appsModel.Init()
	appsModel.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	// Find hadolint in the categories
	hadolintFound := false
	hadolintCatIndex := -1
	hadolintAppIndex := -1
	categories := appsModel.GetCategoriesForTesting()

	for catIndex, cat := range categories {
		for appIndex, app := range cat.Apps {
			if app.Key == "hadolint" {
				hadolintFound = true
				hadolintCatIndex = catIndex
				hadolintAppIndex = appIndex

				break
			}
		}

		if hadolintFound {
			break
		}
	}

	if !hadolintFound {
		t.Fatal("hadolint not found in apps list")
	}

	// Navigate to hadolint
	appsModel.SetCurrentPositionForTesting(hadolintCatIndex, hadolintAppIndex)

	// STEP 1: Mark hadolint for uninstallation
	updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	var isValid bool

	appsModel, isValid = updatedModel.(*AppsModel)
	if !isValid {
		t.Fatalf("Expected *AppsModel, got %T", updatedModel)
	}

	// Verify hadolint is marked for uninstall
	state, exists := appsModel.GetSelectionStateForTesting("hadolint")
	if !exists || state != StateUninstall {
		t.Errorf("Expected hadolint to be marked for uninstall (StateUninstall=%v), got exists=%v, state=%v", StateUninstall, exists, state)
	}

	// STEP 2: Simulate uninstallation completion
	// This should clear the selection state like a real uninstallation
	uninstallOperations := []SelectedOperation{
		{AppKey: "hadolint", Operation: StateUninstall, AppName: "Hadolint"},
	}
	appsModel.handleCompletedOperations(uninstallOperations)

	// Verify selection state is cleared after uninstallation
	state, exists = appsModel.GetSelectionStateForTesting("hadolint")
	if exists {
		t.Errorf("Selection state should be cleared after uninstallation, but hadolint still has state %v", state)
	}

	// STEP 3: Mark hadolint for installation (user presses Space)
	updatedModel2, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	appsModel, isValid = updatedModel2.(*AppsModel)
	if !isValid {
		t.Fatalf("Expected *AppsModel, got %T", updatedModel2)
	}

	// Verify hadolint is now marked for installation
	state, exists = appsModel.GetSelectionStateForTesting("hadolint")
	if !exists {
		t.Error("Expected hadolint to be in selection map after Space press")
	} else if state != StateInstall {
		t.Errorf("Expected hadolint to be marked for install (StateInstall=%v), got %v", StateInstall, state)
	}

	// STEP 4: Get operations for password screen (user presses Enter)
	operations := appsModel.GetSelectedOperationsForTesting()

	// Verify we have exactly one operation for installation
	if len(operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(operations))
	}

	operation := operations[0]
	if operation.AppKey != "hadolint" {
		t.Errorf("Expected operation for hadolint, got %s", operation.AppKey)
	}

	if operation.Operation != StateInstall {
		t.Errorf("Expected StateInstall operation (%v), got %v", StateInstall, operation.Operation)
	}

	// STEP 5: Create password screen with operations (what user sees)
	passwordPrompt := NewPasswordPrompt(context.Background(), styleConfig, operations)

	// CRITICAL TEST: Password screen should show "Installing" message
	expectedMessage := "Installing 1 applications requires administrator privileges."
	if passwordPrompt.message != expectedMessage {
		t.Errorf("Password screen shows wrong message:\n  Expected: %s\n  Actual:   %s", expectedMessage, passwordPrompt.message)
	}

	// Verify operations in password screen
	if len(passwordPrompt.operations) != 1 {
		t.Fatalf("Password screen should have 1 operation, got %d", len(passwordPrompt.operations))
	}

	passOp := passwordPrompt.operations[0]
	if passOp.Operation != StateInstall {
		t.Errorf("Password screen operation should be StateInstall (%v), got %v", StateInstall, passOp.Operation)
	}
}

// TestToggleLogicAfterCompletion specifically tests the toggle logic
// after operations complete to ensure state transitions are correct.
//
//nolint:cyclop,paralleltest // Integration test with multiple scenario validation - no parallel due to state interactions
func TestToggleLogicAfterCompletion(t *testing.T) {
	styleConfig := styles.New()
	appsModel := NewTestAppsModel(styleConfig, 80, 40)
	appsModel.Init()
	appsModel.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	// Find any app for testing
	categories := appsModel.GetCategoriesForTesting()
	if len(categories) == 0 || len(categories[0].Apps) == 0 {
		t.Fatal("No apps available for testing")
	}

	testApp := categories[0].Apps[0]

	appsModel.SetCurrentPositionForTesting(0, 0)

	// Test sequence: None → Install → Complete → None → Install

	// Initial state: should be None
	initialState, exists := appsModel.GetSelectionStateForTesting(testApp.Key)
	if exists && initialState != StateNone {
		t.Errorf("Initial state should be None, got exists=%v, state=%v", exists, initialState)
	}

	// Mark for installation
	updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatalf("Expected *AppsModel, got %T", updatedModel)
	}

	state, exists := appsModel.GetSelectionStateForTesting(testApp.Key)
	if !exists || state != StateInstall {
		t.Errorf("After Space: expected StateInstall, got exists=%v, state=%v", exists, state)
	}

	// Complete installation (simulate what happens after successful install)
	completedOps := []SelectedOperation{
		{AppKey: testApp.Key, Operation: StateInstall, AppName: testApp.Name},
	}
	appsModel.handleCompletedOperations(completedOps)

	// After completion: selection should be cleared
	state, exists = appsModel.GetSelectionStateForTesting(testApp.Key)
	if exists {
		t.Errorf("After completion: selection should be cleared, got state=%v", state)
	}

	// Mark for installation again (this is where the bug might occur)
	updatedModel2, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	var ok2 bool

	appsModel, ok2 = updatedModel2.(*AppsModel)
	if !ok2 {
		t.Fatalf("Expected *AppsModel, got %T", updatedModel2)
	}

	state, exists = appsModel.GetSelectionStateForTesting(testApp.Key)
	if !exists || state != StateInstall {
		t.Errorf("After second Space: expected StateInstall, got exists=%v, state=%v", exists, state)
	}

	// Get operations - should be StateInstall
	operations := appsModel.GetSelectedOperationsForTesting()
	if len(operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(operations))
	}

	if operations[0].Operation != StateInstall {
		t.Errorf("Operation should be StateInstall (%v), got %v", StateInstall, operations[0].Operation)
	}
}
