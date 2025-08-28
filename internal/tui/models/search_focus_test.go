// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// TestSearchFocusInitialization tests initial search focus state.
func TestSearchFocusInitialization(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Initially, search should not be active
	if model.searchActive {
		t.Error("Search should not be active initially")
	}

	if model.searchHasFocus {
		t.Error("Search field should not have focus initially")
	}

	if model.GetSearchHasFocus() {
		t.Error("GetSearchHasFocus() should return false initially")
	}
}

// TestSearchActivationFocus tests focus behavior when search is activated.
func TestSearchActivationFocus(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Simulate search activation via SearchUpdateMsg
	updatedModel, _ := model.Update(SearchUpdateMsg{
		Query:  "",
		Active: true,
	})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Search should be active
	if !appsModel.searchActive {
		t.Error("Search should be active after SearchUpdateMsg with Active=true")
	}

	// Search field should have focus initially
	if !appsModel.GetSearchHasFocus() {
		t.Error("Search field should have focus when search is first activated")
	}
}

// TestContextSwitchingKeys tests {/} key handling for focus switching.
func TestContextSwitchingKeys(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Activate search with some results
	updatedModel, _ := model.Update(SearchUpdateMsg{Query: "git", Active: true})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Initially search field should have focus
	if !appsModel.GetSearchHasFocus() {
		t.Error("Search field should have focus initially")
	}

	// Test } key moves focus to search results
	updatedModel, cmd := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if cmd == nil {
		t.Error("Should return a command when handling context switch key")
	}

	if appsModel.GetSearchHasFocus() {
		t.Error("Search results should have focus after pressing '}'")
	}

	// Test { key moves focus back to search field
	updatedModel, cmd = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if cmd == nil {
		t.Error("Should return a command when handling context switch key")
	}

	if !appsModel.GetSearchHasFocus() {
		t.Error("Search field should have focus after pressing '{'")
	}
}

// TestSearchResultsNavigation tests j/k navigation in search results.
func TestSearchResultsNavigation(t *testing.T) {
	t.Parallel()

	appsModel := setupSearchResultsNavigationTest(t)

	// Verify we have search results
	if len(appsModel.filteredApps) == 0 {
		t.Skip("Skipping navigation test - no search results found for 'git'")
	}

	// Verify focus is on search results
	if appsModel.GetSearchHasFocus() {
		t.Error("Search results should have focus for navigation test")
	}

	testDownNavigationInSearchResults(t, appsModel)
	testUpNavigationInSearchResults(t, appsModel)
}

// setupSearchResultsNavigationTest sets up navigation test with search results.
func setupSearchResultsNavigationTest(t *testing.T) *AppsModel {
	t.Helper()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Activate search with query that should return results
	updatedModel, _ := model.Update(SearchUpdateMsg{Query: "git", Active: true})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Move focus to search results
	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	return appsModel
}

// testDownNavigationInSearchResults tests j key navigation.
func testDownNavigationInSearchResults(t *testing.T, appsModel *AppsModel) {
	t.Helper()

	// Test j key navigation (down)
	initialSelection := appsModel.searchSelection

	updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		*appsModel = *newModel
	}

	if len(appsModel.filteredApps) > 1 {
		// Should move selection down if there are multiple results
		if appsModel.searchSelection <= initialSelection {
			t.Error("'j' key should move selection down in search results")
		}
	}
}

// testUpNavigationInSearchResults tests k key navigation.
func testUpNavigationInSearchResults(t *testing.T, appsModel *AppsModel) {
	t.Helper()

	// Test k key navigation (up)
	if appsModel.searchSelection > 0 {
		previousSelection := appsModel.searchSelection

		updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		if newModel, ok := updatedModel.(*AppsModel); ok {
			*appsModel = *newModel
		}

		if appsModel.searchSelection >= previousSelection {
			t.Error("'k' key should move selection up in search results")
		}
	}
}

// TestSearchSelectionOperations tests selection operations in search results.
func TestSearchSelectionOperations(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Activate search
	updatedModel, _ := model.Update(SearchUpdateMsg{Query: "vim", Active: true})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Move focus to search results
	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if len(appsModel.filteredApps) == 0 {
		t.Skip("Skipping selection test - no search results found for 'vim'")
	}

	// Get the first search result
	if appsModel.searchSelection < 0 {
		appsModel.searchSelection = 0
	}

	firstApp := appsModel.filteredApps[appsModel.searchSelection]

	// Test space key toggles selection
	initialState := appsModel.selected[firstApp.Key]

	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	newState := appsModel.selected[firstApp.Key]
	if newState == initialState {
		t.Error("Space key should toggle selection state in search results")
	}

	// Test 'd' key marks for uninstall
	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	finalState := appsModel.selected[firstApp.Key]
	if finalState != StateUninstall {
		t.Error("'d' key should mark app for uninstallation in search results")
	}
}

// TestEmptySearchQueryBehavior tests behavior with empty search query.
func TestEmptySearchQueryBehavior(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Activate search with empty query
	updatedModel, _ := model.Update(SearchUpdateMsg{Query: "", Active: true})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Empty query should show all apps
	if len(appsModel.filteredApps) == 0 {
		t.Error("Empty search query should show all apps")
	}

	// Should still be able to navigate with empty query
	if len(appsModel.filteredApps) > 0 && appsModel.searchSelection < 0 {
		t.Error("Search selection should be initialized for empty query results")
	}
}

// TestSearchQueryUpdateFocus tests focus management during query updates.
func TestSearchQueryUpdateFocus(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Test search activation sets focus correctly
	updatedModel, _ := model.Update(SearchUpdateMsg{Query: "test", Active: true})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	if !appsModel.GetSearchHasFocus() {
		t.Error("Search field should have focus when search becomes active")
	}

	// Test search deactivation clears focus
	updatedModel, _ = appsModel.Update(SearchUpdateMsg{Query: "", Active: false})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if appsModel.GetSearchHasFocus() {
		t.Error("Search field should not have focus when search becomes inactive")
	}

	if appsModel.searchActive {
		t.Error("Search should not be active after SearchUpdateMsg with Active=false")
	}
}

// TestSearchResultsDisplayMode tests rendering when search is active.
func TestSearchResultsDisplayMode(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Initialize viewport
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// Test normal category view
	normalView := model.View()
	if normalView == "" {
		t.Error("Normal view should not be empty")
	}

	// Activate search
	updatedModel, _ = model.Update(SearchUpdateMsg{Query: "git", Active: true})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// Test search results view
	searchView := model.View()
	if searchView == "" {
		t.Error("Search view should not be empty")
	}

	// Views should be different (search shows results, normal shows categories)
	if searchView == normalView {
		t.Error("Search view should be different from normal category view")
	}
}

// TestFuzzySearchAccuracy tests the fuzzy search algorithm.
func TestFuzzySearchAccuracy(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Test exact matches are found
	updatedModel, _ := model.Update(SearchUpdateMsg{Query: "git", Active: true})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Should find git-related apps
	found := false

	for _, app := range appsModel.filteredApps {
		if strings.Contains(strings.ToLower(app.Name), "git") || strings.Contains(strings.ToLower(app.Description), "git") {
			found = true

			break
		}
	}

	if !found && len(appsModel.filteredApps) > 0 {
		t.Error("Search for 'git' should find git-related applications")
	}

	// Test that very dissimilar apps are filtered out
	updatedModel, _ = appsModel.Update(SearchUpdateMsg{Query: "mise", Active: true})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	// Should not find completely unrelated apps like "Spotify"
	for _, app := range appsModel.filteredApps {
		if app.Name == "Spotify" {
			t.Error("Search for 'mise' should not return Spotify (too dissimilar)")
		}
	}
}

// TestSearchResultsSelectionBounds tests selection bounds in search results.
func TestSearchResultsSelectionBounds(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Activate search with results
	updatedModel, _ := model.Update(SearchUpdateMsg{Query: "vim", Active: true})

	appsModel, ok := updatedModel.(*AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Move focus to search results
	updatedModel, _ = appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		appsModel = newModel
	}

	if len(appsModel.filteredApps) == 0 {
		t.Skip("Skipping bounds test - no search results")
	}

	// Selection should be within bounds
	if appsModel.searchSelection < 0 || appsModel.searchSelection >= len(appsModel.filteredApps) {
		t.Errorf("Search selection %d should be within bounds [0, %d)",
			appsModel.searchSelection, len(appsModel.filteredApps))
	}

	// Navigate to bottom
	for range len(appsModel.filteredApps) + 5 {
		appsModel.navigateSearchDown()
	}

	// Should not exceed bounds
	if appsModel.searchSelection >= len(appsModel.filteredApps) {
		t.Errorf("Search selection should not exceed bounds after navigating down")
	}

	// Navigate to top
	for range len(appsModel.filteredApps) + 5 {
		appsModel.navigateSearchUp()
	}

	// Should not go below zero
	if appsModel.searchSelection < 0 {
		t.Error("Search selection should not go below zero after navigating up")
	}
}

// Helper functions
