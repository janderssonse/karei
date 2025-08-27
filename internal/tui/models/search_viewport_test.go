// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// TestSearchResultsViewportScrolling tests proper viewport scrolling in search results.
func TestSearchResultsViewportScrolling(t *testing.T) {
	t.Parallel()

	model := setupSearchViewportTest(t)

	if len(model.filteredApps) < 5 {
		t.Skip("Skipping viewport test - need at least 5 search results")
	}

	testInitialViewportPosition(t, model)
	testDownwardScrolling(t, model)
	testUpwardScrolling(t, model)
}

// setupSearchViewportTest sets up a search viewport test model.
func setupSearchViewportTest(t *testing.T) *AppsModel {
	t.Helper()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 20) // Small viewport to force scrolling

	// Initialize viewport
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// Activate search with query that should return multiple results
	updatedModel, _ = model.Update(SearchUpdateMsg{Query: "code", Active: true})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// Move focus to search results
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	return model
}

// testInitialViewportPosition tests that viewport starts at top.
func testInitialViewportPosition(t *testing.T, model *AppsModel) {
	t.Helper()

	initialOffset := model.viewport.YOffset
	if initialOffset != 0 {
		t.Error("Viewport should start at offset 0 for new search results")
	}
}

// testDownwardScrolling tests viewport scrolling when navigating down.
func testDownwardScrolling(t *testing.T, model *AppsModel) {
	t.Helper()

	initialOffset := model.viewport.YOffset

	// Navigate down multiple times
	for range 10 {
		model.navigateSearchDown()
	}

	// Should have scrolled down to follow selection
	if model.viewport.YOffset <= initialOffset {
		// Check if we're at the bottom of results (might not need to scroll)
		if model.searchSelection < len(model.filteredApps)-1 {
			t.Error("Viewport should scroll down when navigating beyond visible area")
		}
	}
}

// testUpwardScrolling tests viewport scrolling when navigating up.
func testUpwardScrolling(t *testing.T, model *AppsModel) {
	t.Helper()

	// Record position after scrolling down
	scrolledOffset := model.viewport.YOffset

	// Navigate back up multiple times
	for range 8 {
		model.navigateSearchUp()
	}

	// Should have scrolled up to follow selection
	if model.viewport.YOffset >= scrolledOffset {
		// Only error if we're not at the top
		if model.searchSelection > 0 {
			t.Error("Viewport should scroll up when navigating back toward top")
		}
	}
}

// TestSearchResultsViewportReset tests viewport reset on new search.
func TestSearchResultsViewportReset(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 15)

	// Initialize viewport
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 15})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// First search
	updatedModel, _ = model.Update(SearchUpdateMsg{Query: "vim", Active: true})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// Move focus to results and scroll down
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// Navigate down to scroll the viewport
	for range 5 {
		model.navigateSearchDown()
	}

	// Should have some scroll offset
	scrollOffset := model.viewport.YOffset

	// New search with different query
	updatedModel, _ = model.Update(SearchUpdateMsg{Query: "git", Active: true})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// Viewport should be reset to top
	if model.viewport.YOffset != 0 {
		t.Errorf("Viewport should reset to offset 0 for new search, got %d", model.viewport.YOffset)
	}

	// Selection should be reset to first result
	if len(model.filteredApps) > 0 && model.searchSelection != 0 {
		t.Errorf("Search selection should reset to 0 for new search, got %d", model.searchSelection)
	}

	// Verify that previous scroll offset was actually non-zero
	if scrollOffset == 0 {
		t.Log("Note: Previous search didn't scroll, test might need adjustment")
	}
}

// TestSearchResultsViewportBounds tests viewport bounds checking.
func TestSearchResultsViewportBounds(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 10) // Very small viewport

	// Initialize viewport
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// Search with results
	updatedModel, _ = model.Update(SearchUpdateMsg{Query: "dev", Active: true})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	// Move to search results
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	if len(model.filteredApps) == 0 {
		t.Skip("Skipping bounds test - no search results")
	}

	// Navigate to last item
	for model.searchSelection < len(model.filteredApps)-1 {
		model.navigateSearchDown()
	}

	// Should not exceed viewport bounds
	maxOffset := model.viewport.TotalLineCount() - model.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	if model.viewport.YOffset > maxOffset {
		t.Errorf("Viewport offset %d should not exceed max offset %d",
			model.viewport.YOffset, maxOffset)
	}

	// Navigate to first item
	for model.searchSelection > 0 {
		model.navigateSearchUp()
	}

	// Should not go below zero
	if model.viewport.YOffset < 0 {
		t.Errorf("Viewport offset should not go below 0, got %d", model.viewport.YOffset)
	}
}

// TestSearchSelectionLineCalculation tests search selection line calculation.
func TestSearchSelectionLineCalculation(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 20)

	// Initialize and setup search
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	updatedModel, _ = model.Update(SearchUpdateMsg{Query: "test", Active: true})
	if newModel, ok := updatedModel.(*AppsModel); ok {
		model = newModel
	}

	if len(model.filteredApps) == 0 {
		t.Skip("Skipping line calculation test - no search results")
	}

	// Test line calculation for different selections
	for selectionIndex := range min(5, len(model.filteredApps)) {
		model.searchSelection = selectionIndex
		line := model.calculateSearchSelectionLine()

		// Line should increase with selection index
		expectedMinLine := 3 + selectionIndex // Header + empty + padding + selection index
		if line < expectedMinLine {
			t.Errorf("Selection line %d should be at least %d for selection %d",
				line, expectedMinLine, selectionIndex)
		}
	}

	// Test with selection 0
	model.searchSelection = 0

	line := model.calculateSearchSelectionLine()
	if line <= 0 {
		t.Error("Selection line should be positive for valid selection")
	}
}
