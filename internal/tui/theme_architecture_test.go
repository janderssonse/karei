package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/janderssonse/karei/internal/tui/models"
	"github.com/janderssonse/karei/internal/tui/styles"
)

func TestFixedThemeArchitecture(t *testing.T) {
	t.Parallel()

	// Test 1: Create theme screen with proper idiomatic architecture
	styleConfig := &styles.Styles{
		Primary:   lipgloss.Color("#7aa2f7"),
		Secondary: lipgloss.Color("#bb9af7"),
		Success:   lipgloss.Color("#9ece6a"),
		Warning:   lipgloss.Color("#e0af68"),
		Error:     lipgloss.Color("#f7768e"),
	}

	themeModel := models.NewThemes(styleConfig)

	// Test 2: Verify initialization
	themeModel.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Test 3: Navigate through themes to test viewport
	for range 3 {
		themeModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	// Test 4: Test view rendering (should use viewport)
	view := themeModel.View()

	if len(view) == 0 {
		t.Error("Theme view should not be empty")
	}

	hasThemeContent := strings.Contains(view, "Tokyo Night") || strings.Contains(view, "Catppuccin")
	if !hasThemeContent {
		t.Error("Theme view should contain theme names")
	}

	// Test 5: Verify no manual string building artifacts
	hasStringBuilder := strings.Contains(view, "\n\n\n") // Manual spacing artifacts
	if hasStringBuilder {
		t.Error("Theme view should not have manual string building artifacts")
	}
}

func TestMainAppThemeIntegration(t *testing.T) {
	t.Parallel()

	// Test main app integration
	app := NewApp()
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Navigate to theme screen
	app.Update(models.NavigateMsg{Screen: int(ThemeScreen)})

	// Test main app view with theme screen
	appView := app.View()

	// Verify header is present (should show "🎨 Choose Your Theme")
	hasThemeHeader := strings.Contains(appView, "🎨 Choose Your Theme")
	if !hasThemeHeader {
		t.Error("App should show theme header")
	}

	// Verify no search controls in theme header
	hasSearchControls := strings.Contains(appView, "Status:") || strings.Contains(appView, "Type:") || strings.Contains(appView, "Sort:")
	if hasSearchControls {
		t.Error("Theme header should not have search controls")
	}

	// Verify theme-specific footer
	hasThemeFooter := strings.Contains(appView, "[p] Toggle Preview") && strings.Contains(appView, "[Enter] Apply Theme")
	if !hasThemeFooter {
		t.Error("App should show theme footer with correct keybindings")
	}

	// Verify no search keybinding in theme footer
	hasSearchKeybinding := strings.Contains(appView, "[/] Search")
	if hasSearchKeybinding {
		t.Error("Theme footer should not have search keybinding")
	}
}

func TestThemeViewportBehavior(t *testing.T) {
	t.Parallel()

	// Test viewport scrolling behavior with small terminal
	app := NewApp()
	app.Update(tea.WindowSizeMsg{Width: 40, Height: 15})
	app.Update(models.NavigateMsg{Screen: int(ThemeScreen)})

	smallView := app.View()
	smallLines := strings.Split(smallView, "\n")

	if len(smallLines) > 18 { // Allow some margin for borders
		t.Errorf("Small terminal view should fit in constraints, got %d lines", len(smallLines))
	}

	// Test that content still shows themes
	hasThemeContent := strings.Contains(smallView, "Tokyo Night") || strings.Contains(smallView, "Catppuccin")
	if !hasThemeContent {
		t.Error("Small terminal should still show theme content")
	}
}

func TestThemeArchitecturePatterns(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app.Update(models.NavigateMsg{Screen: int(ThemeScreen)})

	appView := app.View()
	lines := strings.Split(appView, "\n")

	// Verify structure: should have header + content + footer
	hasHeader := false
	hasContent := false
	hasFooter := false

	for _, line := range lines {
		if strings.Contains(line, "🎨 Choose Your Theme") {
			hasHeader = true
		}

		if strings.Contains(line, "Tokyo Night") || strings.Contains(line, "Catppuccin") {
			hasContent = true
		}

		if strings.Contains(line, "[Enter] Apply Theme") {
			hasFooter = true
		}
	}

	if !hasHeader {
		t.Error("Theme screen should have proper header")
	}

	if !hasContent {
		t.Error("Theme screen should have theme content")
	}

	if !hasFooter {
		t.Error("Theme screen should have proper footer")
	}

	// Verify idiomatic composition (no manual string artifacts)
	hasStringBuilder := strings.Contains(appView, "\n\n\n")
	if hasStringBuilder {
		t.Error("App view should use idiomatic Lipgloss composition, not manual string building")
	}
}
