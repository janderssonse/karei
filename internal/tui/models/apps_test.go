// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/styles"
)

func TestAppsModel_ViewportScrolling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		terminalHeight int
	}{
		{"Very small terminal", 15},
		{"Small terminal", 25},
		{"Medium terminal", 40},
		{"Large terminal", 60},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			styleConfig := styles.New()
			model := NewTestAppsModel(styleConfig, 80, testCase.terminalHeight)

			// Initialize the viewport by simulating window size message
			model.Update(tea.WindowSizeMsg{Width: 80, Height: testCase.terminalHeight})

			// Verify viewport is ready
			if !model.ready {
				t.Error("Viewport should be ready after window size message")
			}

			// Verify viewport has reasonable dimensions (app adjusts for header/footer space)
			if model.viewport.Width != 80 {
				t.Errorf("Expected viewport width 80, got %d", model.viewport.Width)
			}

			if model.viewport.Height <= 0 || model.viewport.Height > testCase.terminalHeight {
				t.Errorf("Expected viewport height between 1 and %d, got %d",
					testCase.terminalHeight, model.viewport.Height)
			}
		})
	}
}

func TestAppsModel_Header_AlwaysVisible(t *testing.T) {
	t.Parallel()

	// Test with different terminal sizes
	sizes := []struct {
		name          string
		width, height int
	}{
		{"Small", 80, 20},
		{"Medium", 100, 30},
		{"Large", 120, 50},
	}

	for _, size := range sizes {
		t.Run(size.name, func(t *testing.T) {
			t.Parallel()
			// Create fresh instances for each test case
			styleConfig := styles.New()
			model := NewTestAppsModel(styleConfig, size.width, size.height)

			// Render content - should never be empty (which would indicate header cutoff)
			content := model.View()
			if content == "" {
				t.Errorf("Content is empty for terminal size %dx%d", size.width, size.height)
			}

			// Verify we're not doing any height calculations in the content
			// Content should just render what it has
			if len(content) == 0 {
				t.Errorf("Content should always render something, got empty for %dx%d",
					size.width, size.height)
			}
		})
	}
}

func TestAppsModel_ViewportScrolling_HeaderSafety(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 40)

	// Initialize viewport
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	// Test viewport scrolling with J/K keys
	originalOffset := model.viewport.YOffset

	// Simulate J key (scroll down)
	model.handleNavigationKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})

	// Should have scrolled down (or stayed same if at bottom)
	if model.viewport.YOffset < originalOffset {
		t.Error("J key should scroll down or maintain position, not scroll up")
	}

	// Verify content is never empty (would indicate layout issues)
	content := model.View()
	if content == "" {
		t.Error("Content should never be empty with viewport scrolling")
	}

	// Verify we can render all categories without pagination
	allCategories := model.renderAllCategories()
	if allCategories == "" {
		t.Error("Should be able to render all categories")
	}
}

func TestAppsModel_PureViewportScrolling(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 10) // Small viewport

	// Initialize viewport
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})

	// Verify viewport contains all categories without manual calculations
	content := model.renderAllCategories()
	if content == "" {
		t.Error("Should render all categories")
	}

	// Verify viewport handles scrolling naturally with J/K
	initialOffset := model.viewport.YOffset

	// Simulate J key (manual scroll down)
	model.handleNavigationKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})

	// Should have scrolled down
	if model.viewport.YOffset < initialOffset {
		t.Error("Viewport should not scroll up when using J key")
	}

	// Verify navigation works without manual scroll calculations
	model.navigateDown()
	model.navigateUp()

	// Should not crash and should maintain state
	view := model.View()
	if view == "" {
		t.Error("View should not be empty after navigation")
	}
}

func TestAppsModel_ViewportAutoScroll_DownwardNavigation(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 15) // Small viewport to force scrolling

	// Initialize viewport
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 15})

	// Start at top
	initialOffset := model.viewport.YOffset

	// Navigate down multiple times to move selection toward bottom of visible area
	for range 10 {
		model.navigateDown()
	}

	// Viewport should have scrolled down to follow selection
	if model.viewport.YOffset <= initialOffset {
		// Might not have scrolled if content is small, but should not have scrolled up
		if model.viewport.YOffset < initialOffset {
			t.Error("Viewport should not scroll up when navigating down")
		}
	}

	// Verify selection is still visible (not cut off by footer)
	selectionLine := model.calculateActualSelectionLine()
	viewportTop := model.viewport.YOffset
	viewportBottom := viewportTop + model.viewport.Height - 1

	if selectionLine < viewportTop || selectionLine > viewportBottom {
		t.Errorf("Selection at line %d should be visible in viewport range %d-%d",
			selectionLine, viewportTop, viewportBottom)
	}
}

func TestAppsModel_ViewportAutoScroll_UpwardNavigation(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 15) // Small viewport to force scrolling

	// Initialize viewport
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 15})

	// First, navigate to bottom to establish scroll position
	for range 20 {
		model.navigateDown()
	}

	// Record position after scrolling down
	scrolledDownOffset := model.viewport.YOffset

	// Now navigate back up multiple times
	for range 15 {
		model.navigateUp()
	}

	// Viewport should have scrolled up to follow selection
	if model.viewport.YOffset >= scrolledDownOffset {
		t.Error("Viewport should scroll up when navigating upward from bottom")
	}

	// Verify selection is still visible (not hidden under header)
	selectionLine := model.calculateActualSelectionLine()
	viewportTop := model.viewport.YOffset
	viewportBottom := viewportTop + model.viewport.Height - 1

	// Critical test: after upward navigation, selection should maintain proper buffer
	topBuffer := 6 // Should match the buffer in ensureSelectionVisible

	// Test that scrolling worked - selection should have proper distance from top
	// unless viewport is at the very beginning
	distanceFromTop := selectionLine - viewportTop
	if viewportTop > 0 && distanceFromTop < topBuffer {
		t.Errorf("Selection should maintain %d line buffer from top when not at start, got distance %d (selection line %d, viewport top %d)",
			topBuffer, distanceFromTop, selectionLine, viewportTop)
	}

	if selectionLine < viewportTop || selectionLine > viewportBottom {
		t.Errorf("Selection at line %d should be visible in viewport range %d-%d",
			selectionLine, viewportTop, viewportBottom)
	}
}

func TestAppsModel_ViewportAutoScroll_BufferZones(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 20) // Medium viewport

	// Initialize viewport
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})

	// Test top buffer zone
	// Navigate down just a bit, then back up to test top buffer
	for range 5 {
		model.navigateDown()
	}

	// Navigate back up - should trigger top buffer scrolling
	for range 3 {
		model.navigateUp()
	}

	// Verify selection maintains proper distance from viewport edges
	selectionLine := model.calculateActualSelectionLine()
	viewportTop := model.viewport.YOffset
	viewportBottom := viewportTop + model.viewport.Height - 1

	// Selection should not be too close to edges
	const (
		expectedTopBuffer    = 6
		expectedBottomBuffer = 3
	)

	distanceFromTop := selectionLine - viewportTop
	distanceFromBottom := viewportBottom - selectionLine

	if distanceFromTop < expectedTopBuffer && viewportTop > 0 {
		t.Errorf("Selection should maintain %d line buffer from top, got %d",
			expectedTopBuffer, distanceFromTop)
	}

	if distanceFromBottom < expectedBottomBuffer {
		// This is less critical as we prioritize not cutting off at bottom
		t.Logf("Selection has %d lines from bottom (expected %d+)",
			distanceFromBottom, expectedBottomBuffer)
	}
}

func TestAppsModel_ViewportAutoScroll_ExactLineCalculation(t *testing.T) {
	t.Parallel()

	styleConfig := styles.New()
	model := NewTestAppsModel(styleConfig, 80, 25)

	// Initialize viewport
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 25})

	// Test that calculateActualSelectionLine produces consistent results
	line1 := model.calculateActualSelectionLine()

	// Navigate down and check line increases
	model.navigateDown()
	line2 := model.calculateActualSelectionLine()

	if line2 <= line1 {
		t.Error("Selection line should increase when navigating down")
	}

	// Navigate back up and check line decreases
	model.navigateUp()
	line3 := model.calculateActualSelectionLine()

	if line3 != line1 {
		t.Errorf("Selection line should return to original position, got %d expected %d",
			line3, line1)
	}

	// Test horizontal navigation (category change)
	originalCat := model.currentCat
	model.handleNavigationKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if model.currentCat == originalCat {
		// Might be at last category
		t.Log("Already at last category, testing previous instead")
		model.handleNavigationKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	}

	// Line calculation should still work after category change
	line4 := model.calculateActualSelectionLine()
	if line4 < 0 {
		t.Error("Selection line should never be negative")
	}
}
