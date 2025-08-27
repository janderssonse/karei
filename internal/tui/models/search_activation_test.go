// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// TestSearchActivation tests the search activation flow.
func TestSearchActivation(t *testing.T) {
	t.Parallel()

	model := NewTestAppsModel(styles.New(), 80, 40)

	// Test "/" key activates search
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	if !appsModel.IsSearchActive() {
		t.Error("Search should be active after pressing '/'")
	}

	if !appsModel.GetSearchHasFocus() {
		t.Error("Search field should have focus when activated")
	}

	if cmd == nil {
		t.Error("Should return a command to sync search state")
	}

	// Verify the command is SearchActivatedMsg
	if msg := cmd(); msg == nil {
		t.Error("Command should return SearchActivatedMsg")
	}
}

// TestSearchFieldContextAwareJKHandling tests j/k key handling based on focus context.
func TestSearchFieldContextAwareJKHandling(t *testing.T) {
	t.Parallel()

	model := NewTestAppsModel(styles.New(), 80, 40)

	// Activate search - focus should start on search field
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Verify search field has focus initially
	if !appsModel.GetSearchHasFocus() {
		t.Error("Search field should have focus when search is activated")
	}

	// Test j/k keys are added as literal characters when search field has focus
	originalQuery := appsModel.GetSearchQuery()

	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if appsModel.GetSearchQuery() != originalQuery+"j" {
		t.Error("'j' should be added to search query when search field has focus")
	}

	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if appsModel.GetSearchQuery() != originalQuery+"jk" {
		t.Error("'k' should be added to search query after 'j'")
	}
}

// TestContextSwitchingBehavior tests {/} key behavior for switching focus.
func TestContextSwitchingBehavior(t *testing.T) {
	t.Parallel()

	model := NewTestAppsModel(styles.New(), 80, 40)

	// Activate search
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Set up a query to enable results
	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	// Initially, search field should have focus
	if !appsModel.GetSearchHasFocus() {
		t.Error("Search field should have focus initially")
	}

	// Test } key moves focus to search results
	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if appsModel.GetSearchHasFocus() {
		t.Error("Search results should have focus after pressing '}'")
	}

	// Test { key moves focus back to search field
	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if !appsModel.GetSearchHasFocus() {
		t.Error("Search field should have focus after pressing '{'")
	}
}

// TestSearchFieldNotConsumeNavigationKeys tests that {/} are not typed as literals.
func TestSearchFieldNotConsumeNavigationKeys(t *testing.T) {
	t.Parallel()

	model := NewTestAppsModel(styles.New(), 80, 40)

	// Activate search
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Test that { and } are not added to search query
	originalQuery := appsModel.GetSearchQuery()

	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}})

	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if appsModel.GetSearchQuery() != originalQuery {
		t.Error("'{' should not be added to search query - it's a navigation key")
	}

	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})

	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if appsModel.GetSearchQuery() != originalQuery {
		t.Error("'}' should not be added to search query - it's a navigation key")
	}
}

// TestSearchQueryHandling tests basic search query manipulation.
func TestSearchQueryHandling(t *testing.T) {
	t.Parallel()

	model := NewTestAppsModel(styles.New(), 80, 40)

	// Activate search
	appsModel := activateSearchForTest(t, model)

	// Test adding regular characters
	appsModel = addSearchCharacterForTest(t, appsModel, 'v')
	appsModel = addSearchCharacterForTest(t, appsModel, 'i')
	appsModel = addSearchCharacterForTest(t, appsModel, 'm')

	if appsModel.GetSearchQuery() != "vim" {
		t.Errorf("Expected search query 'vim', got '%s'", appsModel.GetSearchQuery())
	}

	// Test backspace
	appsModel = testBackspaceForSearch(t, appsModel)

	if appsModel.GetSearchQuery() != "vi" {
		t.Errorf("Expected search query 'vi' after backspace, got '%s'", appsModel.GetSearchQuery())
	}

	// Test enter preserves query but deactivates search
	testEnterPreservesQueryForSearch(t, appsModel)

	// Test escape clears search and query
	testEscapeClearsQueryForSearch(t, appsModel)
}

// activateSearchForTest activates search and returns the model.
func activateSearchForTest(t *testing.T, model *AppsModel) *AppsModel {
	t.Helper()

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	return appsModel
}

// addSearchCharacterForTest adds a character to search query.
func addSearchCharacterForTest(t *testing.T, model *AppsModel, char rune) *AppsModel {
	t.Helper()

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		return newModel
	}

	return model
}

// testBackspaceForSearch tests backspace functionality.
func testBackspaceForSearch(t *testing.T, model *AppsModel) *AppsModel {
	t.Helper()

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		return newModel
	}

	return model
}

// testEnterPreservesQueryForSearch tests enter key preserves query.
func testEnterPreservesQueryForSearch(t *testing.T, model *AppsModel) {
	t.Helper()

	originalQuery := model.GetSearchQuery()

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	appsModel := model
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if appsModel.IsSearchActive() {
		t.Error("Search should be inactive after enter")
	}

	// Verify the command indicates query preservation
	validatePreserveQueryCommand(t, cmd, originalQuery)
}

// testEscapeClearsQueryForSearch tests escape key clears query.
func testEscapeClearsQueryForSearch(t *testing.T, model *AppsModel) {
	t.Helper()

	// Reactivate first
	appsModel := activateSearchForTest(t, model)

	// Add some query content
	appsModel = addSearchCharacterForTest(t, appsModel, 't')

	// Escape should clear everything
	updatedModel, cmd := appsModel.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if appsModel.IsSearchActive() {
		t.Error("Search should be inactive after escape")
	}

	if appsModel.GetSearchQuery() != "" {
		t.Error("Search query should be empty after escape")
	}

	// Verify the command indicates query clearing
	validateClearQueryCommand(t, cmd)
}

// TestEmptySearchShowsAllApps tests that empty search query shows all apps.
func TestEmptySearchShowsAllApps(t *testing.T) {
	t.Parallel()

	model := NewTestAppsModel(styles.New(), 80, 40)

	// Activate search with empty query
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Should show all apps when query is empty
	if len(appsModel.getFilteredApps()) == 0 {
		t.Error("Empty search query should show all apps, not zero results")
	}
}

// TestSearchFieldVisualFeedback tests visual feedback for search field focus.
func TestSearchFieldVisualFeedback(t *testing.T) {
	t.Parallel()

	model := NewTestAppsModel(styles.New(), 80, 40)

	// Activate search
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Test search field rendering when focused
	searchField := appsModel.renderSearchField()
	if searchField == "" {
		t.Error("Search field should render content when active")
	}

	// The search field should contain visual indicators when focused
	if !containsSearchHighlight(searchField) {
		t.Error("Search field should be highlighted when it has focus")
	}
}

// Helper functions for testing

func containsSearchHighlight(searchField string) bool {
	// Check if search field contains styling that indicates highlighting
	// Look for cursor or highlighted content
	return len(searchField) > 10 && (strings.Contains(searchField, "│") || strings.Contains(searchField, "["))
}

// validatePreserveQueryCommand validates that command preserves query as expected.
func validatePreserveQueryCommand(t *testing.T, cmd tea.Cmd, originalQuery string) {
	t.Helper()

	if cmd == nil {
		return
	}

	msg := cmd()
	if msg == nil {
		return
	}

	deactivateMsg, ok := msg.(SearchDeactivatedMsg)
	if !ok {
		return
	}

	if !deactivateMsg.PreserveQuery {
		t.Error("Enter should preserve query")
	}

	if deactivateMsg.Query != originalQuery {
		t.Error("Deactivate message should contain the preserved query")
	}
}

// validateClearQueryCommand validates that command clears query as expected.
func validateClearQueryCommand(t *testing.T, cmd tea.Cmd) {
	t.Helper()

	if cmd == nil {
		return
	}

	msg := cmd()
	if msg == nil {
		return
	}

	deactivateMsg, ok := msg.(SearchDeactivatedMsg)
	if !ok {
		return
	}

	if deactivateMsg.PreserveQuery {
		t.Error("Escape should not preserve query")
	}
}
