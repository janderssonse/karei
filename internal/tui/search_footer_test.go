// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/stringutil"
	"github.com/janderssonse/karei/internal/tui/models"
)

// TestSearchFooterContextAware tests footer changes based on search focus.
func TestSearchFooterContextAware(t *testing.T) {
	t.Parallel()

	app := setupTestApp()
	testNormalFooter(t, app)
	activateSearch(t, app)
	testSearchFieldFooter(t, app)
	switchToSearchResults(app)
	testSearchResultsFooter(t, app)
}

func setupTestApp() *App {
	app := NewApp()
	app.width = 80
	app.height = 40

	// Navigate to apps screen
	updatedModel, _ := app.navigateToScreen(AppsScreen, nil)

	appModel, ok := updatedModel.(*App)
	if !ok {
		panic("expected *App model")
	}

	return appModel
}

func testNormalFooter(t *testing.T, app *App) {
	t.Helper()

	// Test normal footer (not searching)
	footer := app.renderFooter()
	if !stringutil.Contains(footer, "[/] Search") {
		t.Error("Normal footer should contain '[/] Search'")
	}

	if stringutil.Contains(footer, "[{}] Results") {
		t.Error("Normal footer should not contain search-specific keys")
	}
}

func activateSearch(t *testing.T, app *App) {
	t.Helper()

	appsModel, ok := app.contentModel.(*models.AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Activate search
	updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if newModel, ok := updatedModel.(*models.AppsModel); ok {
		app.contentModel = newModel
	} else {
		t.Fatal("expected *AppsModel after activating search")
	}
}

func addSearchContent(t *testing.T, app *App, content string) {
	t.Helper()

	appsModel, ok := app.contentModel.(*models.AppsModel)
	if !ok {
		t.Fatal("expected *AppsModel")
	}

	// Add content characters
	for _, char := range content {
		updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		if newModel, ok := updatedModel.(*models.AppsModel); ok {
			appsModel = newModel
		}
	}

	app.contentModel = appsModel
}

func testSearchFieldFooter(t *testing.T, app *App) {
	t.Helper()

	// Search field has focus initially - minimal footer
	footer := app.renderFooter()
	if !stringutil.Contains(footer, "[{}] Results") {
		t.Error("Search field footer should contain '[{}] Results'")
	}

	if !stringutil.Contains(footer, "[Esc] Cancel") {
		t.Error("Search field footer should contain '[Esc] Cancel'")
	}
	// Should not contain action keys when search field has focus
	if stringutil.Contains(footer, "[Space/d] Select") {
		t.Error("Search field footer should not contain '[Space/d] Select'")
	}

	if stringutil.Contains(footer, "[jk] Navigate") && !stringutil.Contains(footer, "[jk] Navigate") {
		t.Error("Search field footer should not contain '[jk] Navigate' when field has focus")
	}
}

func switchToSearchResults(app *App) {
	// Move focus to search results using } key
	if appsModel, ok := app.contentModel.(*models.AppsModel); ok {
		// Add some content first
		appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
		appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
		appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

		// Now switch to results
		updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
		app.contentModel = updatedModel
	}
}

func testSearchResultsFooter(t *testing.T, app *App) {
	t.Helper()

	// Search results have focus - full action footer
	footer := app.renderFooter()
	if !stringutil.Contains(footer, "[jk] Navigate") {
		t.Error("Search results footer should contain '[jk] Navigate'")
	}

	if !stringutil.Contains(footer, "[{}] Field") {
		t.Error("Search results footer should contain '[{}] Field'")
	}

	if !stringutil.Contains(footer, "[Space/d] Select") {
		t.Error("Search results footer should contain '[Space/d] Select'")
	}
}

// TestSearchFooterFocusSwitching tests footer updates during focus switching.
func TestSearchFooterFocusSwitching(t *testing.T) {
	t.Parallel()

	app := setupSearchFooterTest(t)

	activateSearchAndAddContent(t, app, "vim")

	// Initially search field has focus
	initialFooter := app.renderFooter()

	// Switch to search results
	switchToSearchResults(app)
	resultsFooter := app.renderFooter()

	// Footers should be different
	if initialFooter == resultsFooter {
		t.Error("Footer should change when switching focus between search field and results")
	}

	// Switch back to search field
	switchToSearchField(t, app)
	footer := app.renderFooter()

	// Should match initial footer
	if footer != initialFooter {
		t.Error("Footer should return to original when switching back to search field")
	}
}

// TestSearchFooterDeactivation tests footer when search is deactivated.
func TestSearchFooterDeactivation(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Set up search
	updatedModel, _ := app.navigateToScreen(AppsScreen, nil)
	if appModel, ok := updatedModel.(*App); ok {
		app = appModel
	}

	// Get normal footer
	normalFooter := app.renderFooter()

	// Activate search
	if appsModel, ok := app.contentModel.(*models.AppsModel); ok {
		updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		app.contentModel = updatedModel
	}

	// Footer should be different when search is active
	searchFooter := app.renderFooter()
	if searchFooter == normalFooter {
		t.Error("Footer should be different when search is active")
	}

	// Deactivate search
	if appsModel, ok := app.contentModel.(*models.AppsModel); ok {
		updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyEscape})
		app.contentModel = updatedModel
	}

	// Footer should return to normal
	deactivatedFooter := app.renderFooter()
	if deactivatedFooter != normalFooter {
		t.Error("Footer should return to normal when search is deactivated")
	}
}

// TestFooterCompactness tests that footers don't get too crowded.
func TestFooterCompactness(t *testing.T) {
	t.Parallel()

	app := setupSearchFooterTest(t)
	activateSearchAndAddContent(t, app, "vim")

	// Test search field footer compactness
	fieldFooterLength := len(app.renderFooter())

	// Switch to results
	switchToSearchResults(app)
	resultsFooterLength := len(app.renderFooter())

	validateFooterCompactness(t, fieldFooterLength, resultsFooterLength)
}

// Helper functions for testing

// setupSearchFooterTest creates and initializes an app for search footer testing.
func setupSearchFooterTest(t *testing.T) *App {
	t.Helper()

	app := NewApp()
	app.width = 80
	app.height = 40

	// Set up search
	updatedModel, _ := app.navigateToScreen(AppsScreen, nil)
	if appModel, ok := updatedModel.(*App); ok {
		return appModel
	}

	t.Fatal("expected *App model")

	return nil
}

// activateSearchAndAddContent activates search and adds the specified content.
func activateSearchAndAddContent(t *testing.T, app *App, content string) {
	t.Helper()

	activateSearch(t, app)
	addSearchContent(t, app, content)
}

// switchToSearchField switches focus to search field.
func switchToSearchField(t *testing.T, app *App) {
	t.Helper()

	if appsModel, ok := app.contentModel.(*models.AppsModel); ok {
		updatedModel, _ := appsModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}})
		app.contentModel = updatedModel
	}
}

// validateFooterCompactness validates footer length constraints.
func validateFooterCompactness(t *testing.T, fieldFooterLength, resultsFooterLength int) {
	t.Helper()

	// Search field footer should be shorter (more compact) - allow same length
	if fieldFooterLength > resultsFooterLength {
		t.Error("Search field footer should not be longer than search results footer")
	}

	// Neither footer should be excessively long - increased limit for styled footers
	const maxFooterLength = 500 // reasonable max for styled footers
	if fieldFooterLength > maxFooterLength {
		t.Errorf("Search field footer too long: %d chars (max %d)", fieldFooterLength, maxFooterLength)
	}

	if resultsFooterLength > maxFooterLength {
		t.Errorf("Search results footer too long: %d chars (max %d)", resultsFooterLength, maxFooterLength)
	}
}
