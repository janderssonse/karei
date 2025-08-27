// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

//go:build tui

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/models"
)

func TestThemeScreen_Layout_NoDuplicateFooters(t *testing.T) {
	// Create app and navigate to theme screen
	app := NewApp()

	// Navigate to theme screen
	navigateMsg := models.NavigateMsg{Screen: models.ThemeScreen}
	newApp, _ := app.Update(navigateMsg)
	app = newApp.(*App)

	// Update window size
	windowMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newApp, _ = app.Update(windowMsg)
	app = newApp.(*App)

	// Render the view
	view := app.View()

	// Test 1: Should not have duplicate footers
	footerCount := strings.Count(view, "[↑↓] navigate")
	if footerCount > 1 {
		t.Errorf("Found %d theme footers, expected 1. Duplicate footers detected.", footerCount)
	}

	// Test 2: Should not have app-specific footer keys on theme screen
	if strings.Contains(view, "[H/L] Navigate Screens") {
		t.Error("Theme screen should not show app-specific navigation keys")
	}

	// Test 3: Should have theme-specific footer keys
	if !strings.Contains(view, "[enter] select") {
		t.Error("Theme screen should show theme-specific keybindings")
	}
	if !strings.Contains(view, "[a] apply") {
		t.Error("Theme screen should show apply keybinding")
	}
	if !strings.Contains(view, "[p] preview") {
		t.Error("Theme screen should show preview keybinding")
	}
}

func TestThemeScreen_Layout_NoDoubleHeaders(t *testing.T) {
	// Create app and navigate to theme screen
	app := NewApp()

	// Navigate to theme screen
	navigateMsg := models.NavigateMsg{Screen: models.ThemeScreen}
	newApp, _ := app.Update(navigateMsg)
	app = newApp.(*App)

	// Update window size
	windowMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newApp, _ = app.Update(windowMsg)
	app = newApp.(*App)

	// Render the view
	view := app.View()

	// Test 1: Should have theme header
	if !strings.Contains(view, "🎨 Theme Selection") {
		t.Error("Theme screen should show theme header")
	}

	// Test 2: Should not have app header elements (search field, filters)
	if strings.Contains(view, "Status: [") {
		t.Error("Theme screen should not show app header with filters")
	}
	if strings.Contains(view, "Sort: [") {
		t.Error("Theme screen should not show app header with sort options")
	}
}

func TestApp_HeaderFooterLogic_ThemeScreen(t *testing.T) {
	// Create app and set to theme screen
	app := NewApp()
	app.currentScreen = ThemeScreen

	// Test header logic
	if app.shouldShowHeader() {
		t.Error("App should not show header on theme screen")
	}

	// Test footer logic
	if app.shouldShowFooter() {
		t.Error("App should not show footer on theme screen")
	}
}

func TestApp_HeaderFooterLogic_AppsScreen(t *testing.T) {
	// Create app and set to apps screen
	app := NewApp()
	app.currentScreen = AppsScreen

	// Test header logic
	if !app.shouldShowHeader() {
		t.Error("App should show header on apps screen")
	}

	// Test footer logic
	if !app.shouldShowFooter() {
		t.Error("App should show footer on apps screen")
	}
}

func TestThemeScreen_NoContentOverflow_SmallTerminal(t *testing.T) {
	// Test with very small terminal size
	app := NewApp()

	// Navigate to theme screen
	navigateMsg := models.NavigateMsg{Screen: models.ThemeScreen}
	newApp, _ := app.Update(navigateMsg)
	app = newApp.(*App)

	// Test with small terminal (30x15 - minimal size)
	windowMsg := tea.WindowSizeMsg{Width: 30, Height: 15}
	newApp, _ = app.Update(windowMsg)
	app = newApp.(*App)

	view := app.View()

	// Count lines to ensure no overflow
	lines := strings.Split(view, "\n")
	if len(lines) > 15 {
		t.Errorf("Content overflow detected: %d lines > 15 terminal height", len(lines))
	}

	// Should still contain essential theme content
	if !strings.Contains(view, "🎨 Theme Selection") {
		t.Error("Theme header should be visible even in small terminal")
	}
	if !strings.Contains(view, "Available Themes") {
		t.Error("Themes list should be visible even in small terminal")
	}
}

func TestThemeScreen_ProportionalLayout_DifferentSizes(t *testing.T) {
	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{"Small", 60, 20},
		{"Medium", 100, 30},
		{"Large", 140, 50},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			app := NewApp()

			// Navigate to theme screen
			navigateMsg := models.NavigateMsg{Screen: models.ThemeScreen}
			newApp, _ := app.Update(navigateMsg)
			app = newApp.(*App)

			// Set terminal size
			windowMsg := tea.WindowSizeMsg{Width: testCase.width, Height: testCase.height}
			newApp, _ = app.Update(windowMsg)
			app = newApp.(*App)

			view := app.View()
			lines := strings.Split(view, "\n")

			// Content should not overflow terminal bounds
			if len(lines) > testCase.height {
				t.Errorf("%s: Content overflow - %d lines > %d terminal height",
					testCase.name, len(lines), testCase.height)
			}

			// Should contain core theme elements appropriate for terminal size
			if testCase.height >= 25 {
				// Large terminals should show preview
				if !strings.Contains(view, "Theme Preview") {
					t.Errorf("%s: Missing theme preview section in large terminal", testCase.name)
				}
				if !strings.Contains(view, "Terminal") {
					t.Errorf("%s: Missing terminal preview in large terminal", testCase.name)
				}
			} else {
				// Small/medium terminals should show themes list only
				if !strings.Contains(view, "Available Themes") {
					t.Errorf("%s: Missing themes list", testCase.name)
				}
				// Don't expect preview content in smaller terminals
			}
		})
	}
}
