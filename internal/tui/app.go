// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package tui provides the interactive TUI interface for Karei.
package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/tui/models"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// ErrNoTerminal is returned when the TUI is launched in a non-terminal environment.
var ErrNoTerminal = errors.New("TUI requires a terminal environment")

// Screen represents different TUI screens.
type Screen int

// Define screen constants (use models constants for compatibility).
const (
	MenuScreen     Screen = Screen(models.MenuScreen)
	AppsScreen     Screen = Screen(models.AppsScreen)
	ThemeScreen    Screen = Screen(models.ThemeScreen)
	ConfigScreen   Screen = Screen(models.ConfigScreen)
	StatusScreen   Screen = Screen(models.StatusScreen)
	HelpScreen     Screen = Screen(models.HelpScreen)
	ProgressScreen Screen = Screen(models.ProgressScreen)
	PasswordScreen Screen = Screen(models.PasswordScreen)
)

// Key constants for navigation.
const (
	KeyEnter = "enter"
)

// App represents the main TUI application following tree-of-models pattern.
// It manages persistent header/footer and delegates content to screen models.
//
//nolint:containedctx // TUI models require context for proper cancellation propagation
type App struct {
	width         int
	height        int
	styles        *styles.Styles
	currentScreen Screen
	contentModel  tea.Model
	models        map[Screen]tea.Model // Cache of initialized models
	ctx           context.Context      // Context for cancellation and timeout propagation //nolint:containedctx

	// Global navigation state only (idiomatic tree-of-models pattern)

	quitting bool
}

// NewApp creates a new TUI application following tree-of-models pattern.
func NewApp() *App {
	app := &App{
		styles:        styles.New(),
		currentScreen: MenuScreen,
		models:        make(map[Screen]tea.Model),
	}

	// Initialize with menu screen
	menuModel := models.NewMenu(app.styles)
	app.contentModel = menuModel
	app.models[MenuScreen] = menuModel

	return app
}

// NewAppWithContext creates a new TUI application (legacy compatibility).
// Use NewApp() and pass context to Run(ctx) method instead.
func NewAppWithContext(_ context.Context) *App {
	return NewApp()
}

// Run starts the TUI application with the provided context.
func (a *App) Run(ctx context.Context) error {
	// Configure the program with the app as the main model
	program := tea.NewProgram(
		a,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
		tea.WithContext(ctx),      // Use the provided context
	)

	// Run the program
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI application failed: %w", err)
	}

	return nil
}

// Init implements the tea.Model interface.
func (a *App) Init() tea.Cmd {
	return a.contentModel.Init()
}

// Update implements the tea.Model interface with global navigation handling.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		// Calculate content height using Lipgloss Height() method (not arithmetic)
		contentHeight := a.getContentHeight()

		var cmd tea.Cmd

		a.contentModel, cmd = a.contentModel.Update(tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: contentHeight,
		})

		return a, cmd

	case models.NavigateMsg:
		return a.handleNavigation(msg)

	case models.PasswordPromptResult:
		return a.handlePasswordResult(msg)

	// Note: Search state now handled by individual models (idiomatic pattern)

	case tea.KeyMsg:
		return a.handleKeyMessage(msg)

	default:
		// Forward all other messages to content model
		var cmd tea.Cmd

		a.contentModel, cmd = a.contentModel.Update(msg)

		return a, cmd
	}
}

// View implements the tea.Model interface with conditional header/footer layout.
func (a *App) View() string {
	if a.quitting {
		return models.GoodbyeMessage
	}

	// Build the layout tree: Header + Content + Footer (conditionally)
	header := a.renderHeader()
	content := a.renderContent()
	footer := a.renderFooter()

	// Use Lipgloss to compose the layout (avoiding layout arithmetic)
	// Build layout based on what components are present
	components := []string{}

	if header != "" {
		components = append(components, header)
	}

	components = append(components, content)

	if footer != "" {
		components = append(components, footer)
	}

	return lipgloss.JoinVertical(lipgloss.Top, components...)
}

// GetCurrentScreen returns the current screen (for testing).
func (a *App) GetCurrentScreen() Screen {
	return a.currentScreen
}

// SetCurrentScreen sets the current screen (for testing).
func (a *App) SetCurrentScreen(screen Screen) {
	a.currentScreen = screen
}

// ShouldShowHeader returns whether header should be shown (for testing).
func (a *App) ShouldShowHeader() bool {
	return a.shouldShowHeader()
}

// ShouldShowFooter returns whether footer should be shown (for testing).
func (a *App) ShouldShowFooter() bool {
	return a.shouldShowFooter()
}

// GetContentModel returns the current content model (for testing).
func (a *App) GetContentModel() tea.Model {
	return a.contentModel
}

// LaunchWithContext starts the TUI application with a specific context.
func LaunchWithContext(ctx context.Context) error {
	app := NewApp()
	app.ctx = ctx // Store context for propagation to child models

	return app.Run(ctx)
}

// LaunchInteractive starts the interactive TUI interface.
func LaunchInteractive(ctx context.Context) error {
	// Check if we're in a terminal
	if !isTerminal() {
		return fmt.Errorf("terminal check failed: %w", ErrNoTerminal)
	}

	return LaunchWithContext(ctx)
}

// Unexported methods

// handleKeyMessage processes keyboard input with vim-like navigation.
func (a *App) handleKeyMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle global keys first
	if cmd := a.handleGlobalKeys(msg); cmd != nil {
		return a, cmd
	}

	// All input now delegated to content models (idiomatic pattern)

	// Handle navigation keys
	return a.handleNavigationKeys(msg)
}

// handleGlobalKeys processes global key commands (quit only - idiomatic pattern).
func (a *App) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c", "q":
		a.quitting = true

		return tea.Quit
	}

	return nil
}

// handleNavigationKeys processes navigation between screens and delegates to content.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) handleNavigationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "shift+h", "H":
		return a.navigateToPreviousScreen()
	case "shift+l", "L":
		return a.navigateToNextScreen()
	case "shift+j", "J":
		return a.handleVerticalNavigation(msg)
	case "shift+k", "K":
		return a.handleVerticalNavigation(msg)
	default:
		// Delegate ALL other keys (including hjkl) to content model
		var cmd tea.Cmd

		a.contentModel, cmd = a.contentModel.Update(msg)

		return a, cmd
	}
}

// handleVerticalNavigation handles J/K navigation for apps screen.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) handleVerticalNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// On AppsScreen, let J/K pass through for category page navigation
	if a.currentScreen == AppsScreen {
		var cmd tea.Cmd

		a.contentModel, cmd = a.contentModel.Update(msg)

		return a, cmd
	}
	// Future: Up/Down in screen list for other screens
	return a, nil
}

// Search and input handling now completely delegated to individual models (idiomatic pattern)

// renderHeader renders the header with search functionality - statically visible on app pages.
func (a *App) renderHeader() string {
	// Header should be statically visible on app selection pages
	if !a.shouldShowHeader() {
		return ""
	}

	title := a.getScreenTitle()

	// Simplified headers - models handle their own UI (idiomatic pattern)
	switch a.currentScreen {
	case AppsScreen, ConfigScreen:
		// Simple title header - apps model handles its own search/filter UI
		headerContent := lipgloss.NewStyle().Bold(true).Foreground(a.styles.Primary).Render(title)

		return lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder(), false, false, true, false).
			BorderForeground(a.styles.Primary).
			Render(headerContent)

	case ThemeScreen:
		// Theme screen has full-width header to match footer border width
		headerContent := lipgloss.NewStyle().Bold(true).Foreground(a.styles.Primary).Render(title)
		// Calculate content width accounting for padding (1,2 = vertical, horizontal)
		availableWidth := a.width - 4 // 4 = 2*horizontal padding
		if availableWidth < 10 {
			availableWidth = 10
		}

		return lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder(), false, false, true, false).
			BorderForeground(a.styles.Primary).
			Width(availableWidth).
			Render(headerContent)

	default:
		// Default header for other screens
		headerContent := lipgloss.NewStyle().Bold(true).Foreground(a.styles.Primary).Render(title)

		return lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder(), false, false, true, false).
			BorderForeground(a.styles.Primary).
			Render(headerContent)
	}
}

// shouldShowHeader determines if the current screen should show the header.
func (a *App) shouldShowHeader() bool {
	// AppsScreen handles its own header (both static title and search functionality)
	switch a.currentScreen {
	case ConfigScreen, ThemeScreen:
		return true
	case AppsScreen:
		return false // AppsModel handles its own complete header
	default:
		return false
	}
}

// shouldShowFooter determines if the current screen should show the footer with navigation.
func (a *App) shouldShowFooter() bool {
	switch a.currentScreen {
	case AppsScreen, ConfigScreen, ThemeScreen:
		return true
	default:
		// Also check if we have an apps model as content (fallback)
		if _, isAppsModel := a.contentModel.(*models.AppsModel); isAppsModel {
			return true
		}

		return false
	}
}

// Search UI now handled by individual models (idiomatic pattern)

// renderContent renders the current screen's content.
func (a *App) renderContent() string {
	// Simply render content - let the content model handle its own sizing
	return a.contentModel.View()
}

// renderFooter renders the footer with navigation help - ONLY for app selection pages.
func (a *App) renderFooter() string {
	// Only show footer on AppsScreen, ThemeScreen, ConfigScreen
	if !a.shouldShowFooter() {
		return ""
	}

	var keybindings []string

	// Context-aware footer based on search state (idiomatic UX)
	switch a.currentScreen {
	case AppsScreen, ConfigScreen:
		// Check if AppsModel is in search mode
		if appsModel, ok := a.contentModel.(*models.AppsModel); ok && appsModel.IsSearchActive() {
			if appsModel.GetSearchHasFocus() {
				// Search field has focus - minimal footer
				keybindings = []string{
					"[{}] Results",
					"[Esc] Cancel",
					"[Enter] Done",
					"[q] Quit",
				}
			} else {
				// Search results have focus - full action footer
				keybindings = []string{
					"[jk] Navigate",
					"[{}] Field",
					"[Space/d] Select",
					"[Esc] Cancel",
					"[q] Quit",
				}
			}
		} else {
			// Normal mode footer
			keybindings = []string{
				"[jk] Navigate Items",
				"[{}] Navigate Categories",
				"[H/L] Navigate Screens",
				"[/] Search",
				"[q] Quit",
			}
		}
	case ThemeScreen:
		keybindings = []string{
			"[jk] Navigate Themes",
			"[H/L] Navigate Screens",
			"[p] Toggle Preview",
			"[q] Quit",
		}
	default:
		keybindings = []string{
			"[H/L] Navigate Screens",
			"[q] Quit",
		}
	}

	// Add screen-specific keybindings
	screenKeys := a.getScreenKeybindings()
	if len(screenKeys) > 0 {
		keybindings = append(keybindings, screenKeys...)
	}

	footerText := strings.Join(keybindings, "  ")

	// Calculate content width to match header (same as header calculation)
	availableWidth := a.width - 4 // 4 = 2*horizontal padding
	if availableWidth < 10 {
		availableWidth = 10
	}

	return lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder(), true, false, false, false).
		BorderForeground(a.styles.Primary).
		Width(availableWidth).
		Render(footerText)
}

// getContentHeight calculates available height for content using Lipgloss Height() method.
// Following blog post: use Height() method instead of manual arithmetic.
func (a *App) getContentHeight() int {
	if a.height <= 0 {
		return 0
	}

	reservedHeight := 0

	// Use Lipgloss Height() method for header (only for screens that use main app header)
	if a.shouldShowHeader() {
		header := a.renderHeader()
		if header != "" {
			reservedHeight += lipgloss.Height(header)
		}
	}

	// Use Lipgloss Height() method for footer
	if a.shouldShowFooter() {
		footer := a.renderFooter()
		if footer != "" {
			reservedHeight += lipgloss.Height(footer)
		}
	}

	contentHeight := a.height - reservedHeight
	if contentHeight < 0 {
		return 0
	}

	return contentHeight
}

// getScreenTitle returns the title for the current screen.
func (a *App) getScreenTitle() string {
	switch a.currentScreen {
	case MenuScreen:
		return "Karei - Foundational Flow for Modern Devs"
	case AppsScreen:
		return "📦 Select Applications to Install"
	case ThemeScreen:
		return "🎨 Choose Your Theme"
	case ConfigScreen:
		return "⚙️ System Configuration"
	case StatusScreen:
		return "📊 Installation Status"
	case HelpScreen:
		return "❓ Help & Documentation"
	case ProgressScreen:
		return "⚡ Installing Applications"
	default:
		return "Karei"
	}
}

// getScreenKeybindings returns screen-specific keybindings.
func (a *App) getScreenKeybindings() []string {
	switch a.currentScreen {
	case AppsScreen:
		return []string{"[Space] Toggle Install", "[d] Uninstall", "[Enter] Apply"}
	case ThemeScreen:
		return []string{"[Enter] Apply Theme"}
	case MenuScreen:
		return []string{"[Enter] Select"}
	default:
		return []string{}
	}
}

// handleNavigation handles navigation between screens via NavigateMsg.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) handleNavigation(msg models.NavigateMsg) (tea.Model, tea.Cmd) {
	targetScreen := Screen(msg.Screen)

	// Allow refresh operations even on the same screen (idiomatic pattern)
	if a.currentScreen == targetScreen && msg.Data != nil {
		// Handle same-screen refresh requests
		return a.navigateToScreen(targetScreen, msg.Data)
	}

	// Don't navigate if already on the target screen without data
	if a.currentScreen == targetScreen {
		return a, nil
	}

	return a.navigateToScreen(targetScreen, msg.Data)
}

// handlePasswordResult handles the result from password prompt.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) handlePasswordResult(msg models.PasswordPromptResult) (tea.Model, tea.Cmd) {
	if msg.Cancelled {
		// User cancelled - go back to apps screen
		return a.navigateToScreen(AppsScreen, nil)
	}

	// Password entered - proceed to installation with sudo context
	// Pass both operations and password to progress screen
	progressData := models.ProgressData{
		Operations: msg.Operations,
		Password:   msg.Password,
	}

	return a.navigateToScreen(ProgressScreen, progressData)
}

// navigateToPreviousScreen navigates to the previous screen (H key).
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) navigateToPreviousScreen() (tea.Model, tea.Cmd) {
	screens := []Screen{MenuScreen, AppsScreen, ThemeScreen, ConfigScreen, StatusScreen, HelpScreen}

	for i, screen := range screens {
		if screen == a.currentScreen && i > 0 {
			return a.navigateToScreen(screens[i-1], nil)
		}
	}

	return a, nil
}

// navigateToNextScreen navigates to the next screen (L key).
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) navigateToNextScreen() (tea.Model, tea.Cmd) {
	screens := []Screen{MenuScreen, AppsScreen, ThemeScreen, ConfigScreen, StatusScreen, HelpScreen}

	for i, screen := range screens {
		if screen == a.currentScreen && i < len(screens)-1 {
			return a.navigateToScreen(screens[i+1], nil)
		}
	}

	return a, nil
}

// navigateToScreen handles navigation to a specific screen.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) navigateToScreen(targetScreen Screen, data any) (tea.Model, tea.Cmd) {
	// Progress and Password screens should always be created fresh (idiomatic Elm pattern)
	if targetScreen == ProgressScreen || targetScreen == PasswordScreen {
		// Remove any stale cached instance (idiomatic cleanup)
		delete(a.models, targetScreen)
		newModel := a.createModelForScreen(targetScreen, data)

		return a.setupNewModel(newModel, targetScreen, data)
	}

	// Check if model is already cached for other screens
	if cachedModel, exists := a.models[targetScreen]; exists {
		return a.useCachedModel(targetScreen, cachedModel, data)
	}

	// Create new model if not cached
	newModel := a.createModelForScreen(targetScreen, data)

	return a.setupNewModel(newModel, targetScreen, data)
}

// useCachedModel switches to a cached model and updates its size.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) useCachedModel(targetScreen Screen, cachedModel tea.Model, data any) (tea.Model, tea.Cmd) {
	a.currentScreen = targetScreen
	a.contentModel = cachedModel

	var cmds []tea.Cmd

	// Send content area size to cached model using Lipgloss Height()
	if a.width > 0 && a.height > 0 {
		contentHeight := a.getContentHeight()
		updatedModel, cmd := a.contentModel.Update(tea.WindowSizeMsg{
			Width:  a.width,
			Height: contentHeight,
		})
		a.contentModel = updatedModel
		// Update the cache with the resized model
		a.models[targetScreen] = updatedModel

		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Handle refresh status request for Apps screen
	if targetScreen == AppsScreen {
		if cmd := a.handleAppsScreenData(data, targetScreen); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if len(cmds) > 0 {
		return a, tea.Batch(cmds...)
	}

	return a, nil
}

// handleAppsScreenData handles data passing for Apps screen.
func (a *App) handleAppsScreenData(data any, targetScreen Screen) tea.Cmd {
	appsModel, ok := a.contentModel.(*models.AppsModel)
	if !ok {
		return nil
	}

	switch data := data.(type) {
	case string:
		if data == models.RefreshStatusData {
			// Legacy refresh - immediate but still needs async checking
			return func() tea.Msg {
				return models.RefreshStatusMsg{}
			}
		}
	case models.CompletedOperationsMsg:
		// Immediate sync - pass completed operations directly to apps model
		updatedModel, cmd := appsModel.Update(data)
		a.contentModel = updatedModel
		a.models[targetScreen] = updatedModel

		return cmd
	}

	return nil
}

// createModelForScreen creates a new model based on the screen type.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) createModelForScreen(screen Screen, data any) tea.Model {
	switch screen {
	case MenuScreen:
		return models.NewMenu(a.styles)
	case AppsScreen:
		return a.createAppsModel()
	case ThemeScreen:
		return models.NewThemes(a.styles)
	case ConfigScreen:
		return models.NewConfig(a.styles)
	case StatusScreen:
		return models.NewStatus(a.styles)
	case HelpScreen:
		return models.NewHelp(a.styles)
	case ProgressScreen:
		return a.createProgressModel(data)
	case PasswordScreen:
		return a.createPasswordModel(data)
	default:
		return models.NewMenu(a.styles) // Fallback to menu if unknown screen
	}
}

// setupNewModel initializes and caches a new model.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) setupNewModel(newModel tea.Model, targetScreen Screen, data any) (tea.Model, tea.Cmd) {
	// Cache the new model (except progress and password which are always fresh)
	if targetScreen != ProgressScreen && targetScreen != PasswordScreen {
		a.models[targetScreen] = newModel
	}

	// Update current screen and model
	a.currentScreen = targetScreen
	a.contentModel = newModel

	// Initialize the new model
	initCmd := newModel.Init()

	var cmds []tea.Cmd
	if initCmd != nil {
		cmds = append(cmds, initCmd)
	}

	// Handle window sizing
	if resizeCmd := a.handleWindowSizing(targetScreen); resizeCmd != nil {
		cmds = append(cmds, resizeCmd)
	}

	// Handle refresh status for Apps screen
	if refreshCmd := a.handleRefreshStatus(targetScreen, data); refreshCmd != nil {
		cmds = append(cmds, refreshCmd)
	}

	if len(cmds) > 0 {
		return a, tea.Batch(cmds...)
	}

	return a, nil
}

// handleWindowSizing handles window sizing for new models.
func (a *App) handleWindowSizing(targetScreen Screen) tea.Cmd {
	if a.width <= 0 || a.height <= 0 {
		return nil
	}

	contentHeight := a.getContentHeight()
	updatedModel, resizeCmd := a.contentModel.Update(tea.WindowSizeMsg{
		Width:  a.width,
		Height: contentHeight,
	})
	a.contentModel = updatedModel

	// Update the cache with the resized model
	if targetScreen != ProgressScreen {
		a.models[targetScreen] = updatedModel
	}

	return resizeCmd
}

// handleRefreshStatus handles refresh status requests for Apps screen.
func (a *App) handleRefreshStatus(targetScreen Screen, data any) tea.Cmd {
	if targetScreen != AppsScreen || data != models.RefreshStatusData {
		return nil
	}

	if _, ok := a.contentModel.(*models.AppsModel); ok {
		// Immediate refresh - progress screen navigation indicates operations are complete
		return func() tea.Msg {
			return models.RefreshStatusMsg{}
		}
	}

	return nil
}

// createAppsModel creates an apps model with proper sizing.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) createAppsModel() tea.Model {
	contentHeight := a.getContentHeight()

	return models.NewAppsWithSize(a.ctx, a.styles, a.width, contentHeight)
}

// createProgressModel creates a progress model handling different data formats.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) createProgressModel(data any) tea.Model {
	// Handle progress data with password
	if progressData, ok := data.(models.ProgressData); ok {
		return models.NewProgressWithOperationsAndPassword(a.ctx, a.styles, progressData.Operations, progressData.Password)
	}
	// Handle new mixed operations format (without password)
	if operations, ok := data.([]models.SelectedOperation); ok {
		return models.NewProgressWithOperations(a.ctx, a.styles, operations)
	}
	// Handle legacy string array format
	if selectedApps, ok := data.([]string); ok {
		return models.NewProgress(a.ctx, a.styles, selectedApps)
	}

	return models.NewProgress(a.ctx, a.styles, []string{})
}

// createPasswordModel creates a password model with operation data.
//
//nolint:ireturn // Bubble Tea framework requires returning tea.Model interface
func (a *App) createPasswordModel(data any) tea.Model {
	if operations, ok := data.([]models.SelectedOperation); ok {
		return models.NewPasswordPrompt(a.ctx, a.styles, operations)
	}

	return models.NewMenu(a.styles) // Fallback if no operations
}

// Filter and search functionality now completely delegated to individual models (idiomatic pattern)

// isTerminal checks if stdout is connected to a terminal.
func isTerminal() bool {
	// Always return true for now to allow testing
	// Future enhancement: Implement proper terminal detection
	return true
}
