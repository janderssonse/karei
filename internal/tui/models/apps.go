// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models provides Bubble Tea models for the TUI interface.
//
//nolint:funcorder // This file groups methods by functionality rather than export/unexport order for better readability
package models

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/janderssonse/karei/internal/stringutil"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// Status indicators for applications.
const (
	StatusNotInstalled = "○"
	StatusInstalled    = "●"
	StatusSelected     = "✓"
	StatusUninstall    = "✗"
	StatusPending      = "⋯" // Status pending/checking
)

// Filter constants for application filtering.
const (
	FilterAll          = "All"
	FilterInstalled    = "Installed"
	FilterNotInstalled = "Not Installed"
	MethodAPTDisplay   = "apt"
)

// SelectionState represents the intended operation for an application.
type SelectionState int

// Selection states for application operations.
const (
	StateNone      SelectionState = iota // No operation selected
	StateInstall                         // Mark for installation
	StateUninstall                       // Mark for uninstallation
)

// Common messages.
const (
	GoodbyeMessage = "Goodbye!\n"
)

// AppCategory represents a category of applications.
type AppCategory struct {
	Name         string
	Description  string
	Applications []Application
}

// Application represents an installable application.
type Application struct {
	Key         string
	Name        string
	Description string
	Icon        string
	Category    string
	Installed   bool
	Size        string
	Source      string
	Selected    bool
}

// String implements the list.Item interface.
func (a Application) String() string {
	return a.Name
}

// FilterValue implements the list.Item interface.
func (a Application) FilterValue() string {
	return a.Name + " " + a.Description
}

// Title returns the application title with status.
func (a Application) Title() string {
	status := StatusNotInstalled
	if a.Installed {
		status = StatusInstalled
	}

	return fmt.Sprintf("%s %s", status, a.Name)
}

// Desc returns the app description with size.
func (a Application) Desc() string {
	return fmt.Sprintf("%s • %s", a.Description, a.Size)
}

// AppsModel implements the application selection screen using proper Bubble Tea viewport.
//
//nolint:containedctx // TUI models require context for proper cancellation propagation
type AppsModel struct {
	// Core state
	styles   *styles.Styles
	width    int
	height   int
	quitting bool
	ctx      context.Context // Parent context for cancellation/timeout propagation //nolint:containedctx

	// Application data
	categories []category
	selected   map[string]SelectionState

	// Navigation state
	currentCat int

	// Viewport for proper scrolling (no pagination hacks)
	viewport viewport.Model

	ready bool

	// Apps manager for status checking
	appsManager *apps.Manager
	keyMap      AppsKeyMap

	// Search functionality
	searchQuery     string
	filteredApps    []app
	searchActive    bool
	searchSelection int  // Index of currently selected search result
	searchHasFocus  bool // Whether search field has focus (vs search results)

	// Filter and sort functionality
	installStatusFilter string // "All", "Installed", "Not Installed"
	packageTypeFilter   string // "All", "apt", "flatpak", "snap", "deb", "mise", "aqua", "github", "script"
	sortOption          string // "Name", "Status", "Type", "Category"
}

// category represents an internal category with navigation state.
type category struct {
	name       string
	apps       []app
	selected   map[string]SelectionState
	currentApp int
}

// app represents an internal application with state.
type app struct {
	Key           string
	Name          string
	Description   string
	Source        string
	Installed     bool
	Selected      bool
	StatusPending bool // True when installation status is being checked
}

// StatusUpdateMsg carries installation status updates from async checks.
type StatusUpdateMsg struct {
	AppName   string
	Installed bool
}

// RefreshStatusMsg triggers a refresh of all app installation statuses.
type RefreshStatusMsg struct{}

// SearchUpdateMsg is now defined in navigation.go

// FilterUpdateMsg carries filter and sort state updates.
type FilterUpdateMsg struct {
	InstallStatus string // "All", "Installed", "Not Installed"
	PackageType   string // "All", "apt", "flatpak", "snap", "deb", "mise", "aqua", "github", "script"
	SortOption    string // "Name", "Status", "Type", "Category"
}

// NavigationWithRefreshMsg carries navigation message with refresh request.
type NavigationWithRefreshMsg struct {
	NavigateMsg
	RefreshStatus bool
}

// AppsKeyMap defines key bindings for the apps screen.
type AppsKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Select   key.Binding
	Deselect key.Binding
	Install  key.Binding
	Back     key.Binding
	Quit     key.Binding
}

// NewApps creates a new application selection model.
func NewApps(ctx context.Context, styleConfig *styles.Styles) *AppsModel {
	return NewAppsWithSize(ctx, styleConfig, 200, 100)
}

// NewAppsWithSize creates the apps model with specified dimensions.
func NewAppsWithSize(ctx context.Context, styleConfig *styles.Styles, width, height int) *AppsModel {
	adapter := newAppCatalogAdapter()
	appCategories := adapter.getAllCategoriesFast()

	selected := make(map[string]SelectionState)
	categories := make([]category, 0, len(appCategories))

	// Convert external categories to internal format
	for _, cat := range appCategories {
		apps := make([]app, 0, len(cat.Applications))
		for _, application := range cat.Applications {
			apps = append(apps, app{
				Key:           application.Key,
				Name:          application.Name,
				Description:   application.Description,
				Source:        application.Source,
				Installed:     application.Installed,
				Selected:      false,
				StatusPending: true, // Start with pending status, will be updated async
			})
		}

		categories = append(categories, category{
			name:       cat.Name,
			apps:       apps,
			selected:   selected,
			currentApp: 0,
		})
	}

	model := &AppsModel{
		styles:      styleConfig,
		width:       width,
		height:      height,
		ctx:         ctx, // Store parent context
		categories:  categories,
		selected:    selected,
		appsManager: apps.NewTUIManager(false), // Use TUI-optimized manager to suppress command output
		keyMap:      DefaultAppsKeyMap(),
		viewport:    viewport.New(width, height),

		// Initialize filter and sort states
		installStatusFilter: "All",
		packageTypeFilter:   "All",
		sortOption:          "Name",
	}

	return model
}

// DefaultAppsKeyMap returns the default key bindings.
func DefaultAppsKeyMap() AppsKeyMap {
	return AppsKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "prev page"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("J"),
			key.WithHelp("J", "next page"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		Deselect: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "deselect"),
		),
		Install: key.NewBinding(
			key.WithKeys(KeyEnter, "i"),
			key.WithHelp("enter/i", "install"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// NewStatusCheckCommand creates a command to check a single app's installation status.
func NewStatusCheckCommand(parentCtx context.Context, appName string, manager *apps.Manager) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
		defer cancel()

		installed := manager.IsAppInstalled(ctx, appName)

		return StatusUpdateMsg{
			AppName:   appName,
			Installed: installed,
		}
	}
}

// Init initializes the apps model.
func (m *AppsModel) Init() tea.Cmd {
	// Start async status checking for all apps
	var cmds []tea.Cmd

	for _, cat := range m.categories {
		for _, app := range cat.apps {
			cmds = append(cmds, NewStatusCheckCommand(m.ctx, app.Key, m.appsManager))
		}
	}

	return tea.Batch(cmds...)
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the AppsModel.
func (m *AppsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate viewport height accounting for our own header
		headerHeight := lipgloss.Height(m.renderSearchHeader())
		viewportHeight := msg.Height - headerHeight

		if !m.ready {
			// Initialize viewport with proper size (reserving space for header)
			m.viewport = viewport.New(msg.Width, viewportHeight)
			m.viewport.SetContent(m.renderAllCategories())
			m.ready = true
		} else {
			// Update viewport size (reserving space for header)
			m.viewport.Width = msg.Width
			m.viewport.Height = viewportHeight
		}

		return m, nil

	case StatusUpdateMsg:
		m.updateAppStatus(msg.AppName, msg.Installed)

		return m, nil

	case RefreshStatusMsg:
		// Refresh all app statuses
		return m, m.refreshAllAppStatuses()

	case CompletedOperationsMsg:
		// Handle immediate status updates from completed operations
		m.handleCompletedOperations(msg.Operations)

		return m, nil

	case SearchUpdateMsg:
		// Handle search query updates and sync search active state
		m.searchActive = msg.Active
		m.updateSearchQuery(msg.Query)

		return m, nil

	case FilterUpdateMsg:
		// Handle filter and sort state updates
		m.installStatusFilter = msg.InstallStatus
		m.packageTypeFilter = msg.PackageType
		m.sortOption = msg.SortOption

		// Re-apply search and filters to update display
		if m.searchActive {
			m.updateSearchQuery(m.searchQuery)
		}

		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMessage(msg)
	}

	return m, nil
}

// View renders the viewport content - proper scrolling, no pagination hacks.
func (m *AppsModel) View() string {
	if m.quitting {
		return GoodbyeMessage
	}

	if !m.ready {
		return "Loading..."
	}

	// Build the complete view: header + content
	components := []string{}

	// Always show the search/filter header (as per TUI design)
	searchHeader := m.renderSearchHeader()
	if searchHeader != "" {
		components = append(components, searchHeader)
	}

	// Update viewport content with current state
	m.viewport.SetContent(m.renderAllCategories())

	// Add main content
	components = append(components, m.viewport.View())

	// Compose with Lipgloss
	if len(components) == 1 {
		return components[0]
	}

	return lipgloss.JoinVertical(lipgloss.Top, components...)
}

// CategoryForTesting represents a category structure for testing purposes.
type CategoryForTesting struct {
	Name string
	Apps []Application
}

// GetCategoriesForTesting returns categories in a test-friendly format.
func (m *AppsModel) GetCategoriesForTesting() []CategoryForTesting {
	result := make([]CategoryForTesting, 0, len(m.categories))

	for _, cat := range m.categories {
		testCategory := CategoryForTesting{
			Name: cat.name,
			Apps: make([]Application, 0, len(cat.apps)),
		}

		// Convert internal app structs to public Application structs
		for _, app := range cat.apps {
			testCategory.Apps = append(testCategory.Apps, Application{
				Key:         app.Key,
				Name:        app.Name,
				Description: app.Description,
				Icon:        "📦", // Default icon for testing
				Category:    cat.name,
				Installed:   app.Installed,
				Size:        "Unknown", // Default size for testing
				Source:      app.Source,
				Selected:    m.selected[app.Key] != StateNone,
			})
		}

		result = append(result, testCategory)
	}

	return result
}

// Private methods (unexported - placed after public methods per funcorder)

// handleKeyMessage processes keyboard input with vim-like navigation.
// Global navigation (quit, back, HJKL screen switching) is handled by main App.
//

//nolint:funcorder // Methods grouped logically by functionality
func (m *AppsModel) handleKeyMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search activation/deactivation first
	switch {
	case msg.String() == "/":
		// Activate search mode (idiomatic pattern - handle own search)
		m.searchActive = true
		m.searchHasFocus = true
		m.searchQuery = ""

		// Initialize with all apps for empty query (default behavior)
		m.updateSearchResults()

		// Return command to notify main app
		return m, func() tea.Msg {
			return SearchActivatedMsg{Active: true}
		}
	case msg.String() == "esc" && m.searchActive:
		// Idiomatic UX: Esc clears everything
		query := m.searchQuery
		m.searchActive = false
		m.searchHasFocus = false
		m.searchQuery = "" // Esc clears query
		m.filteredApps = []app{}
		m.searchSelection = -1

		return m, func() tea.Msg {
			return SearchDeactivatedMsg{PreserveQuery: false, Query: query}
		}
	}

	// Handle search input when search field has focus
	if m.searchActive && m.searchHasFocus {
		return m, m.handleSearchInput(msg)
	}

	// Handle installation commands
	if installCmd := m.handleInstallationKeys(msg); installCmd != nil {
		return m, installCmd
	}

	// Handle context switching with {/}
	if cmd := m.handleContextSwitchKeys(msg); cmd != nil {
		return m, cmd
	}

	// Handle navigation (j/k always work for up/down, {/} for context switch)
	m.handleNavigationKeys(msg)

	// Handle selection (only when not in search field)
	if !m.searchActive || !m.searchHasFocus {
		m.handleSelectionKeys(msg)
	}

	return m, nil
}

// renderAllCategories renders ALL categories for the viewport to handle scrolling.
// This is the proper Bubble Tea way - render everything, let viewport scroll.
func (m *AppsModel) renderAllCategories() string {
	// If search is active, show filtered results instead of categories
	if m.searchActive {
		return m.renderSearchResults()
	}

	if len(m.categories) == 0 {
		return "No categories available"
	}

	// Render ALL categories - viewport handles what's visible
	categoryViews := make([]string, 0, len(m.categories))
	for i, cat := range m.categories {
		isCurrent := i == m.currentCat
		categoryViews = append(categoryViews, m.renderCategory(cat, isCurrent))
	}

	// Pure Lipgloss composition - no pagination, no calculations
	return lipgloss.JoinVertical(lipgloss.Top, categoryViews...)
}

// renderCategory renders a single category.
func (m *AppsModel) renderCategory(cat category, isCurrent bool) string {
	installCount, uninstallCount := m.calculateSelectionStats(cat)
	maxNameWidth, maxDescWidth := m.calculateColumnWidths(cat.apps)
	appLines := m.renderAppLines(cat, isCurrent, maxNameWidth, maxDescWidth)

	// Create category title with decorative border style and selection summary
	var selectionSummary string
	if installCount > 0 || uninstallCount > 0 {
		selectionSummary = fmt.Sprintf(" [+%d/-%d of %d]", installCount, uninstallCount, len(cat.apps))
	} else {
		selectionSummary = fmt.Sprintf(" [%d total]", len(cat.apps))
	}

	categoryTitle := fmt.Sprintf("── %s%s ──", cat.name, selectionSummary)

	// Style the title
	styledTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary).
		Render(categoryTitle)

	// Compose category content with styled title above apps
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		styledTitle,
		"",
		lipgloss.JoinVertical(lipgloss.Left, appLines...),
	)

	return m.renderCategoryWithBorder(content, isCurrent)
}

// getAppIndicator returns the appropriate indicator for an app's state.
func (m *AppsModel) getAppIndicator(app app, selectionState SelectionState) string {
	switch selectionState {
	case StateInstall:
		return StatusSelected // ✓
	case StateUninstall:
		return StatusUninstall // ✗
	default:
		// No selection - show install status
		if app.StatusPending {
			return StatusPending // ⋯ - status being checked
		}

		if app.Installed {
			return StatusInstalled // ●
		}

		return StatusNotInstalled // ○
	}
}

// Navigation methods - viewport automatically follows selection.
func (m *AppsModel) navigateDown() {
	if len(m.categories) == 0 {
		return
	}

	// Get current category
	if m.currentCat < len(m.categories) {
		cat := &m.categories[m.currentCat]

		if cat.currentApp < len(cat.apps)-1 {
			// Move to next app in current category
			cat.currentApp++
		} else if m.currentCat < len(m.categories)-1 {
			// Move to next category
			m.currentCat++
			m.categories[m.currentCat].currentApp = 0
		}
	}

	// Use Bubble Tea viewport's natural scrolling - no arithmetic
	m.ensureSelectionVisible()
}

func (m *AppsModel) navigateUp() {
	if len(m.categories) == 0 {
		return
	}

	// Get current category
	if m.currentCat < len(m.categories) {
		cat := &m.categories[m.currentCat]

		if cat.currentApp > 0 {
			// Move to previous app in current category
			cat.currentApp--
		} else if m.currentCat > 0 {
			// Move to previous category
			m.currentCat--
			prevCat := &m.categories[m.currentCat]
			prevCat.currentApp = len(prevCat.apps) - 1
		}
	}

	// Use Bubble Tea viewport's natural scrolling - no arithmetic
	m.ensureSelectionVisible()
}

// navigateSearchDown navigates down in search results.
func (m *AppsModel) navigateSearchDown() {
	if len(m.filteredApps) == 0 {
		return
	}

	if m.searchSelection < len(m.filteredApps)-1 {
		m.searchSelection++
	}

	// Use Bubble Tea viewport's natural scrolling - no arithmetic
	m.ensureSearchSelectionVisible()
}

// navigateSearchUp navigates up in search results.
func (m *AppsModel) navigateSearchUp() {
	if len(m.filteredApps) == 0 {
		return
	}

	if m.searchSelection > 0 {
		m.searchSelection--
	}

	// Use Bubble Tea viewport's natural scrolling - no arithmetic
	m.ensureSearchSelectionVisible()
}

func (m *AppsModel) toggleInstallSelection() {
	if m.currentCat >= len(m.categories) {
		return
	}

	cat := &m.categories[m.currentCat]
	if cat.currentApp >= len(cat.apps) {
		return
	}

	app := cat.apps[cat.currentApp]

	// Defensive check: If app is not installed and has StateUninstall selection,
	// this is a state corruption bug - reset to StateNone first
	currentState := m.selected[app.Key]
	if currentState == StateUninstall && !app.Installed {
		// State corruption: app is marked for uninstall but is not installed
		// This can happen due to race conditions in model caching
		delete(m.selected, app.Key)

		currentState = StateNone
	}

	// Toggle behavior: None -> Install -> None
	switch currentState {
	case StateNone:
		m.selected[app.Key] = StateInstall
	case StateInstall:
		delete(m.selected, app.Key) // Return to None state
	case StateUninstall:
		m.selected[app.Key] = StateInstall // Switch from uninstall to install
	}

	// DEBUG: Log final state
	finalState := m.selected[app.Key]
	_ = finalState // Keep finalState for potential future use
}

func (m *AppsModel) markForUninstall() {
	if m.currentCat >= len(m.categories) {
		return
	}

	cat := &m.categories[m.currentCat]
	if cat.currentApp >= len(cat.apps) {
		return
	}

	app := cat.apps[cat.currentApp]
	m.selected[app.Key] = StateUninstall
}

// Legacy method for backward compatibility - now uses toggle behavior.
func (m *AppsModel) toggleSelection() {
	m.toggleInstallSelection()
}

// toggleInstallSelectionForSearchResult toggles selection for the currently selected search result.
func (m *AppsModel) toggleInstallSelectionForSearchResult() {
	if !m.searchActive || m.searchSelection < 0 || m.searchSelection >= len(m.filteredApps) {
		return
	}

	app := m.filteredApps[m.searchSelection]

	// Same logic as toggleInstallSelection but for search result
	currentState := m.selected[app.Key]
	if currentState == StateUninstall && !app.Installed {
		// State corruption fix
		delete(m.selected, app.Key)

		currentState = StateNone
	}

	// Toggle behavior: None -> Install -> None
	switch currentState {
	case StateNone:
		m.selected[app.Key] = StateInstall
	case StateInstall:
		delete(m.selected, app.Key) // Return to None state
	case StateUninstall:
		m.selected[app.Key] = StateInstall // Switch from uninstall to install
	}
}

// markForUninstallForSearchResult marks the currently selected search result for uninstallation.
func (m *AppsModel) markForUninstallForSearchResult() {
	if !m.searchActive || m.searchSelection < 0 || m.searchSelection >= len(m.filteredApps) {
		return
	}

	app := m.filteredApps[m.searchSelection]
	m.selected[app.Key] = StateUninstall
}

// handleInstallationKeys processes installation-related key presses.
func (m *AppsModel) handleInstallationKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keyMap.Install), msg.String() == KeyEnter:
		operations := m.getSelectedOperations()
		if len(operations) > 0 {
			// First go to password screen, then to progress
			return func() tea.Msg {
				return NavigateMsg{Screen: PasswordScreen, Data: operations}
			}
		}
	}

	return nil
}

// handleSearchInput processes search input when in search field.
func (m *AppsModel) handleSearchInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "backspace":
		return m.handleBackspace()
	case "enter":
		return m.handleEnterInSearch()
	case "{":
		return m.handleUpFocus()
	case "}":
		return m.handleDownFocus()
	default:
		return m.handleCharacterInput(msg)
	}
}

func (m *AppsModel) handleBackspace() tea.Cmd {
	if len(m.searchQuery) > 0 {
		m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		m.updateSearchResults()
	}

	return nil
}

func (m *AppsModel) handleEnterInSearch() tea.Cmd {
	// Idiomatic UX: Enter deactivates search but preserves query
	if m.searchActive {
		query := m.searchQuery
		m.searchActive = false
		m.searchHasFocus = false
		// Query preserved for potential reactivation
		m.filteredApps = []app{}
		m.searchSelection = -1

		return func() tea.Msg {
			return SearchDeactivatedMsg{PreserveQuery: true, Query: query}
		}
	}

	return nil
}

func (m *AppsModel) handleUpFocus() tea.Cmd {
	// { in search field: do nothing (can't go higher than search field)
	return func() tea.Msg {
		return ContextSwitchMsg{Direction: "up", Context: "search-input"}
	}
}

func (m *AppsModel) handleDownFocus() tea.Cmd {
	// } in search field: navigate down to search results
	m.searchHasFocus = false
	if len(m.filteredApps) > 0 {
		// Go to search results
		if m.searchSelection < 0 {
			m.searchSelection = 0
		}
	}

	return func() tea.Msg {
		return ContextSwitchMsg{Direction: "down", Context: "search-input"}
	}
}

func (m *AppsModel) handleCharacterInput(msg tea.KeyMsg) tea.Cmd {
	// Add character to search query (but exclude control keys)
	keyStr := msg.String()
	if len(keyStr) == 1 && keyStr >= " " && keyStr <= "~" {
		m.searchQuery += keyStr
		m.updateSearchResults()
	}

	return nil
}

// updateSearchResults updates the filtered results based on current search query.
func (m *AppsModel) updateSearchResults() {
	// Use the sophisticated search function instead of simple fuzzy matching
	m.filteredApps = m.performFuzzySearch(m.searchQuery)

	// Reset selection
	m.searchSelection = -1
	if len(m.filteredApps) > 0 {
		m.searchSelection = 0
	}
}

// renderSearchHeader renders the always-visible search/filter header bar.
func (m *AppsModel) renderSearchHeader() string {
	// Title line
	title := "📦 Select Applications to Install"
	titleLine := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary).
		Render(title)

	// Create search input field with enhanced visual feedback
	searchField := m.renderSearchField()

	// Add status and type filters, sort dropdown (matching the screenshot)
	statusField := fmt.Sprintf("Status: [%s ▼]", m.installStatusFilter)
	typeField := fmt.Sprintf("Type: [%s ▼]", m.packageTypeFilter)
	sortField := fmt.Sprintf("Sort: [%s ▼]", m.sortOption)

	// Combine search/filter elements on second line
	controlsLine := fmt.Sprintf("%s  %s  %s  %s", searchField, statusField, typeField, sortField)

	// Add search results info if search is active
	if m.searchActive && m.searchQuery != "" {
		if len(m.filteredApps) > 0 {
			controlsLine += fmt.Sprintf("  (%d results)", len(m.filteredApps))
		} else {
			controlsLine += "  (no matches)"
		}
	}

	// Combine title and controls with proper spacing
	headerContent := lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		"",
		controlsLine,
	)

	// Style the header with proper padding and border
	return lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder(), false, false, true, false).
		BorderForeground(m.styles.Primary).
		Render(headerContent)
}

// ActivateSearchMsg signals the main app to activate search.
type ActivateSearchMsg struct{}

// handleContextSwitchKeys handles upward/downward navigation using {/}.
// { = move up (previous category, or up to search field)
// } = move down (next category, or down to search results/categories)
// Returns a command if the key was handled, nil otherwise.
func (m *AppsModel) handleContextSwitchKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "{":
		return m.handleUpContextSwitch()
	case "}":
		return m.handleDownContextSwitch()
	}

	return nil // Key not handled
}

func (m *AppsModel) handleUpContextSwitch() tea.Cmd {
	// { = Navigate upward
	if m.searchActive {
		return m.handleUpFromSearch()
	}

	return m.handleUpFromCategories()
}

func (m *AppsModel) handleUpFromSearch() tea.Cmd {
	if m.searchHasFocus {
		// In search field: { does nothing (can't go higher)
		return func() tea.Msg {
			return ContextSwitchMsg{Direction: "up", Context: "search-field"}
		}
	}

	// In search results: { goes back to search field
	m.searchHasFocus = true

	return func() tea.Msg {
		return ContextSwitchMsg{Direction: "up", Context: "search-results"}
	}
}

func (m *AppsModel) handleUpFromCategories() tea.Cmd {
	// Regular categories: navigate to previous category OR up to search if at top
	if m.currentCat > 0 {
		// Go to previous category
		m.currentCat--
		m.categories[m.currentCat].currentApp = 0
		m.ensureSelectionVisible()

		return func() tea.Msg {
			return ContextSwitchMsg{Direction: "up", Context: "categories"}
		}
	}

	// At top category: send signal to main app to activate search
	return func() tea.Msg { return ActivateSearchMsg{} }
}

func (m *AppsModel) handleDownContextSwitch() tea.Cmd {
	// } = Navigate downward
	if m.searchActive {
		return m.handleDownFromSearch()
	}

	return m.handleDownFromCategories()
}

func (m *AppsModel) handleDownFromSearch() tea.Cmd {
	if m.searchHasFocus {
		// In search field: go down to search results
		m.searchHasFocus = false
		if len(m.filteredApps) > 0 {
			// Go to search results
			if m.searchSelection < 0 {
				m.searchSelection = 0
			}
		}

		return func() tea.Msg {
			return ContextSwitchMsg{Direction: "down", Context: "search-field"}
		}
	}

	// In search results: } goes back to search field (wrap around)
	m.searchHasFocus = true

	return func() tea.Msg {
		return ContextSwitchMsg{Direction: "down", Context: "search-results"}
	}
}

func (m *AppsModel) handleDownFromCategories() tea.Cmd {
	// Regular categories: navigate to next category
	if m.currentCat < len(m.categories)-1 {
		m.currentCat++
		m.categories[m.currentCat].currentApp = 0
		m.ensureSelectionVisible()
	}

	return func() tea.Msg {
		return ContextSwitchMsg{Direction: "down", Context: "categories"}
	}
}

// handleNavigationKeys processes navigation key presses with viewport scrolling.
// j/k work for both regular navigation and search results.
func (m *AppsModel) handleNavigationKeys(msg tea.KeyMsg) {
	switch {
	case key.Matches(msg, m.keyMap.Down), msg.String() == "j":
		m.handleDownNavigation()
	case key.Matches(msg, m.keyMap.Up), msg.String() == "k":
		m.handleUpNavigation()
	case key.Matches(msg, m.keyMap.PageDown), msg.String() == "J":
		// Scroll viewport down (J key for page navigation)
		m.viewport.ScrollDown(5)
	case key.Matches(msg, m.keyMap.PageUp), msg.String() == "K":
		// Scroll viewport up (K key for page navigation)
		m.viewport.ScrollUp(5)
	}
}

func (m *AppsModel) handleDownNavigation() {
	if m.searchActive && !m.searchHasFocus && len(m.filteredApps) > 0 {
		// Navigate down in search results
		m.navigateSearchDown()
	} else if !m.searchActive || !m.searchHasFocus {
		// Navigate down in regular categories
		m.navigateDown()
	}
}

func (m *AppsModel) handleUpNavigation() {
	if m.searchActive && !m.searchHasFocus && len(m.filteredApps) > 0 {
		// Navigate up in search results
		m.navigateSearchUp()
	} else if !m.searchActive || !m.searchHasFocus {
		// Navigate up in regular categories
		m.navigateUp()
	}
}

// handleSelectionKeys processes selection key presses.
func (m *AppsModel) handleSelectionKeys(msg tea.KeyMsg) {
	switch {
	case key.Matches(msg, m.keyMap.Select), msg.String() == " ":
		if m.searchActive && len(m.filteredApps) > 0 && m.searchSelection >= 0 {
			m.toggleInstallSelectionForSearchResult()
		} else {
			m.toggleInstallSelection()
		}
	case msg.String() == "d":
		if m.searchActive && len(m.filteredApps) > 0 && m.searchSelection >= 0 {
			m.markForUninstallForSearchResult()
		} else {
			m.markForUninstall()
		}
	}
}

// updateAppStatus updates app installation status and clears selection state after operations.
func (m *AppsModel) updateAppStatus(appName string, installed bool) {
	// Clear selection state after any status update
	for i := range m.categories {
		for j := range m.categories[i].apps {
			if m.categories[i].apps[j].Key == appName {
				m.categories[i].apps[j].Installed = installed
				m.categories[i].apps[j].StatusPending = false // Status check completed

				// Clear selection state after any status update
				// This ensures apps show their current status (● installed, ○ not installed)
				// instead of the intended operation (✓ install, ✗ uninstall)
				delete(m.selected, appName)

				return
			}
		}
	}
}

// refreshAllAppStatuses creates commands to refresh all app installation statuses.
func (m *AppsModel) refreshAllAppStatuses() tea.Cmd {
	var cmds []tea.Cmd

	// Mark all apps as pending before starting checks
	for i := range m.categories {
		for j := range m.categories[i].apps {
			m.categories[i].apps[j].StatusPending = true
		}
	}

	for _, cat := range m.categories {
		for _, app := range cat.apps {
			cmds = append(cmds, NewStatusCheckCommand(m.ctx, app.Key, m.appsManager))
		}
	}

	return tea.Batch(cmds...)
}

// ensureSelectionVisible calculates exact line position of selection in rendered content.
// Accounts for category headers, borders, padding, and spacing.
func (m *AppsModel) ensureSelectionVisible() {
	if !m.ready || len(m.categories) == 0 {
		return
	}

	// Calculate EXACT line position of current selection in rendered content
	selectionLine := m.calculateActualSelectionLine()

	// Get current viewport window
	viewportTop := m.viewport.YOffset
	viewportBottom := viewportTop + m.viewport.Height - 1

	// Buffer zones - scroll when selection gets close to edges
	topBuffer := 6    // Keep 6 lines above selection visible (very aggressive upward scrolling)
	bottomBuffer := 3 // Keep 3 lines below selection visible (for category borders)

	// Check if selection is outside comfortable viewing area
	if selectionLine <= viewportTop+topBuffer {
		// Selection too close to top - scroll up to maintain buffer
		newOffset := selectionLine - topBuffer
		if newOffset < 0 {
			newOffset = 0
		}

		m.viewport.SetYOffset(newOffset)
	} else if selectionLine >= viewportBottom-bottomBuffer {
		// Selection too close to bottom - scroll down to maintain buffer
		newOffset := selectionLine - m.viewport.Height + bottomBuffer + 1
		totalLines := m.viewport.TotalLineCount()

		maxOffset := totalLines - m.viewport.Height
		if maxOffset < 0 {
			maxOffset = 0
		}

		if newOffset > maxOffset {
			newOffset = maxOffset
		}

		m.viewport.SetYOffset(newOffset)
	}
	// Selection is comfortably visible - no scroll needed
}

// ensureSearchSelectionVisible calculates exact line position of search selection and scrolls viewport.
func (m *AppsModel) ensureSearchSelectionVisible() {
	if !m.ready || !m.searchActive || len(m.filteredApps) == 0 || m.searchSelection < 0 {
		return
	}

	// Calculate EXACT line position of current search selection in rendered content
	selectionLine := m.calculateSearchSelectionLine()

	// Get current viewport window
	viewportTop := m.viewport.YOffset
	viewportBottom := viewportTop + m.viewport.Height - 1

	// Buffer zones - scroll when selection gets close to edges
	topBuffer := 3    // Keep 3 lines above selection visible
	bottomBuffer := 2 // Keep 2 lines below selection visible

	// Check if selection is outside comfortable viewing area
	if selectionLine <= viewportTop+topBuffer {
		// Selection too close to top - scroll up to maintain buffer
		newOffset := selectionLine - topBuffer
		if newOffset < 0 {
			newOffset = 0
		}

		m.viewport.SetYOffset(newOffset)
	} else if selectionLine >= viewportBottom-bottomBuffer {
		// Selection too close to bottom - scroll down to maintain buffer
		newOffset := selectionLine - m.viewport.Height + bottomBuffer + 1
		totalLines := m.viewport.TotalLineCount()

		maxOffset := totalLines - m.viewport.Height
		if maxOffset < 0 {
			maxOffset = 0
		}

		if newOffset > maxOffset {
			newOffset = maxOffset
		}

		m.viewport.SetYOffset(newOffset)
	}
	// Selection is comfortably visible - no scroll needed
}

// calculateSearchSelectionLine calculates exact line position of search selection.
func (m *AppsModel) calculateSearchSelectionLine() int {
	// Start with search results header
	// Title line: "── Search Results for 'query' ── [N matches] ──"
	line := 1 // Title line

	// Empty line after title
	line++ // Empty line

	// Border top and padding from the border style (1 line for padding)
	line++ // Border/padding

	// Each search result is one line, so selection is at position m.searchSelection
	line += m.searchSelection

	return line
}

// calculateActualSelectionLine calculates exact line position accounting for all rendered elements.
func (m *AppsModel) calculateActualSelectionLine() int {
	line := 0

	// Count lines for each category before current one
	for catIdx := 0; catIdx < m.currentCat && catIdx < len(m.categories); catIdx++ {
		cat := m.categories[catIdx]

		// Render this category to count its exact lines
		categoryContent := m.renderCategory(cat, false)
		categoryLines := lipgloss.Height(categoryContent)
		line += categoryLines

		// Add spacing between categories (if not last)
		if catIdx < len(m.categories)-1 {
			line++ // Empty line between categories
		}
	}

	// Add lines within current category up to current app
	if m.currentCat < len(m.categories) {
		currentCat := m.categories[m.currentCat]

		// Category header (1 line)
		line++

		// Border top and padding
		line += 2

		// Apps before current selection
		line += currentCat.currentApp
	}

	return line
}

// SelectedOperation represents an operation to perform on an application.
type SelectedOperation struct {
	AppKey    string
	Operation SelectionState
	AppName   string
}

// appCatalogAdapter provides adapter for the apps catalog.
type appCatalogAdapter struct {
	manager *apps.Manager
}

// getSelectedOperations returns all selected operations (install and uninstall).
func (m *AppsModel) getSelectedOperations() []SelectedOperation {
	operations := make([]SelectedOperation, 0, len(m.selected))

	// Iterate through categories in order to maintain deterministic order
	for _, cat := range m.categories {
		for _, app := range cat.apps {
			if state, exists := m.selected[app.Key]; exists && state != StateNone {
				operations = append(operations, SelectedOperation{
					AppKey:    app.Key,
					Operation: state,
					AppName:   app.Name,
				})
			}
		}
	}

	return operations
}

// getSelectedApps returns the list of selected applications (legacy compatibility).
func (m *AppsModel) getSelectedApps() []string {
	selected := make([]string, 0, len(m.selected))
	for appKey, state := range m.selected {
		if state == StateInstall {
			selected = append(selected, appKey)
		}
	}

	return selected
}

// newAppCatalogAdapter creates adapter for the apps catalog.
func newAppCatalogAdapter() *appCatalogAdapter {
	return &appCatalogAdapter{
		manager: apps.NewTUIManager(false), // Use TUI-optimized manager to suppress command output
	}
}

func (a *appCatalogAdapter) getAllCategoriesFast() []AppCategory {
	categories := make(map[string]*AppCategory)

	// Get all apps from the catalog
	allApps := apps.Apps

	// Group apps by category - NO synchronous installation checks
	for key, app := range allApps {
		// Start with unknown installation status - will be updated async
		tuiApp := a.transformApp(key, app, false) // false = assume not installed initially

		if cat, exists := categories[app.Group]; exists {
			cat.Applications = append(cat.Applications, tuiApp)
		} else {
			categories[app.Group] = &AppCategory{
				Name:         cases.Title(language.Und).String(app.Group),
				Description:  a.getCategoryDescription(app.Group),
				Applications: []Application{tuiApp},
			}
		}
	}

	// Convert to slice and sort
	result := make([]AppCategory, 0, len(categories))
	for _, cat := range categories {
		slices.SortFunc(cat.Applications, func(a, b Application) int {
			return cmp.Compare(a.Name, b.Name)
		})
		result = append(result, *cat)
	}

	slices.SortFunc(result, func(a, b AppCategory) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return result
}

func (a *appCatalogAdapter) transformApp(key string, app apps.App, installed bool) Application {
	return Application{
		Key:         key,
		Name:        app.Name,
		Description: app.Description,
		Icon:        a.getIconForApp(app),
		Category:    cases.Title(language.Und).String(app.Group),
		Installed:   installed,
		Size:        a.estimateSize(app),
		Source:      a.formatSource(app.Method),
		Selected:    false,
	}
}

func (a *appCatalogAdapter) getCategoryDescription(group string) string {
	descriptions := map[string]string{
		"development":   "Software development tools and IDEs",
		"browsers":      "Web browsers and internet tools",
		"communication": "Chat, video, and messaging applications",
		"media":         "Audio, video, and multimedia applications",
		"productivity":  "Office tools and productivity applications",
		"graphics":      "Image editing and graphics tools",
		"utilities":     "System utilities and tools",
		"gaming":        "Games and gaming platforms",
		"terminal":      "Command-line tools and terminal applications",
		"golang":        "Go programming language tools",
		"javalang":      "Java programming language tools",
		"rustlang":      "Rust programming language tools",
		"pythonlang":    "Python programming language tools",
		"linters":       "Code analysis and linting tools",
	}

	if desc, exists := descriptions[group]; exists {
		return desc
	}

	return "Application tools and utilities"
}

func (a *appCatalogAdapter) getIconForApp(app apps.App) string {
	icons := map[string]string{
		"development":   "◆",
		"browsers":      "◯",
		"communication": "◈",
		"media":         "▶",
		"productivity":  "▣",
		"graphics":      "◉",
		"utilities":     "▪",
		"gaming":        "♦",
		"terminal":      "▸",
		"golang":        "◐",
		"javalang":      "◑",
		"rustlang":      "◈",
		"pythonlang":    "◊",
		"linters":       "✓",
	}

	if icon, exists := icons[app.Group]; exists {
		return icon
	}

	return "•"
}

func (a *appCatalogAdapter) estimateSize(app apps.App) string {
	switch app.Method {
	case domain.MethodAPT:
		return "5-50 MB"
	case domain.MethodSnap:
		return "50-200 MB"
	case domain.MethodFlatpak:
		return "100-500 MB"
	case domain.MethodDEB:
		return "50-300 MB"
	case domain.MethodMise:
		return "1-100 MB"
	case domain.MethodScript:
		return "1-50 MB"
	case domain.MethodAqua:
		return "1-20 MB"
	default:
		return "Unknown"
	}
}

func (a *appCatalogAdapter) formatSource(method domain.InstallMethod) string {
	methodNames := map[domain.InstallMethod]string{
		domain.MethodAPT:          "apt",
		domain.MethodSnap:         "snap",
		domain.MethodFlatpak:      "flatpak",
		domain.MethodDEB:          "deb",
		domain.MethodMise:         "mise",
		domain.MethodScript:       "script",
		domain.MethodBinary:       "binary",
		domain.MethodAqua:         "aqua",
		domain.MethodGitHub:       "github",
		domain.MethodGitHubBinary: "github-bin",
		domain.MethodGitHubBundle: "github-app",
		domain.MethodGitHubJava:   "github-java",
	}

	if name, exists := methodNames[method]; exists {
		return name
	}

	return "unknown"
}

// calculateSelectionStats counts install and uninstall selections in a category.
func (m *AppsModel) calculateSelectionStats(cat category) (int, int) {
	installCount := 0
	uninstallCount := 0

	for _, app := range cat.apps {
		switch m.selected[app.Key] {
		case StateInstall:
			installCount++
		case StateUninstall:
			uninstallCount++
		}
	}

	return installCount, uninstallCount
}

// calculateColumnWidths determines the maximum width for name and description columns.
func (m *AppsModel) calculateColumnWidths(apps []app) (int, int) {
	maxNameWidth := 0
	maxDescWidth := 0

	for _, app := range apps {
		nameLen := len(app.Name)
		descLen := len(app.Description)

		if nameLen > maxNameWidth {
			maxNameWidth = nameLen
		}

		if descLen > maxDescWidth {
			maxDescWidth = descLen
		}
	}

	return maxNameWidth, maxDescWidth
}

// renderAppLines creates formatted lines for all apps in a category.
func (m *AppsModel) renderAppLines(cat category, isCurrent bool, maxNameWidth, maxDescWidth int) []string {
	appLines := make([]string, 0, len(cat.apps))

	for appIdx, app := range cat.apps {
		indicator := m.getAppIndicator(app, m.selected[app.Key])

		// Format with aligned columns: [indicator] [name] [description] [source]
		line := fmt.Sprintf("%s %-*s  %-*s  %s",
			indicator,
			maxNameWidth, app.Name,
			maxDescWidth, app.Description,
			app.Source)

		// Highlight current app in current category
		if isCurrent && appIdx == cat.currentApp {
			line = m.styles.Selected.Render(line)
		} else {
			line = m.styles.Unselected.Render(line)
		}

		appLines = append(appLines, line)
	}

	return appLines
}

// renderCategoryWithBorder applies border styling to category content.
func (m *AppsModel) renderCategoryWithBorder(content string, isCurrent bool) string {
	borderStyle := lipgloss.RoundedBorder()
	if isCurrent {
		borderStyle = lipgloss.ThickBorder()
	}

	return lipgloss.NewStyle().
		Border(borderStyle).
		BorderForeground(m.styles.Primary).
		Padding(1).
		Render(content)
}

// GetSearchHasFocus returns whether the search field currently has focus.
func (m *AppsModel) GetSearchHasFocus() bool {
	return m.searchHasFocus
}

// getFilteredApps returns the current filtered apps list (for testing).
func (m *AppsModel) getFilteredApps() []app {
	return m.filteredApps
}

// IsSearchActive returns whether search is currently active.
func (m *AppsModel) IsSearchActive() bool {
	return m.searchActive
}

// GetSearchQuery returns the current search query.
func (m *AppsModel) GetSearchQuery() string {
	return m.searchQuery
}

// renderSearchField renders the search input field with visual feedback.
func (m *AppsModel) renderSearchField() string {
	if !m.searchActive {
		// Inactive search field
		searchText := strings.Repeat("_", 15)

		return fmt.Sprintf("Search: [%s] 🔍", searchText)
	}

	// Active search field with query
	searchText := m.searchQuery

	padding := 15 - len(searchText)
	if padding > 0 {
		searchText += strings.Repeat("_", padding)
	} else if len(searchText) > 15 {
		// Truncate long queries with ellipsis
		searchText = searchText[:12] + "..."
	}

	// Add cursor when search field has focus
	if m.searchHasFocus {
		// Replace last underscore with cursor
		if strings.HasSuffix(searchText, "_") {
			searchText = searchText[:len(searchText)-1] + "│" // Vertical bar cursor
		} else {
			searchText += "│"
		}
	}

	// Apply styling based on focus state
	searchStyle := lipgloss.NewStyle()
	if m.searchHasFocus {
		// Focused: highlighted background
		searchStyle = searchStyle.
			Background(m.styles.Secondary).
			Foreground(lipgloss.Color("#1a1b26")).
			Bold(true)
	} else if m.searchActive {
		// Active but not focused: subtle highlight
		searchStyle = searchStyle.
			Foreground(m.styles.Primary).
			Bold(false)
	}

	return fmt.Sprintf("Search: [%s] 🔍", searchStyle.Render(searchText))
}

// GetSelectionStateForTesting returns the selection state for a given app key (for testing).
func (m *AppsModel) GetSelectionStateForTesting(appKey string) (SelectionState, bool) {
	state, exists := m.selected[appKey]

	return state, exists
}

// SetSelectionStateForTesting sets the selection state for a given app key (for testing).
func (m *AppsModel) SetSelectionStateForTesting(appKey string, state SelectionState) {
	m.selected[appKey] = state
}

// SetCurrentPositionForTesting sets the current category and app position (for testing).
func (m *AppsModel) SetCurrentPositionForTesting(catIndex, appIndex int) {
	m.currentCat = catIndex
	if catIndex < len(m.categories) {
		m.categories[catIndex].currentApp = appIndex
	}
}

// MarkForUninstallForTesting calls markForUninstall method (for testing).
func (m *AppsModel) MarkForUninstallForTesting() {
	m.markForUninstall()
}

// GetSelectedOperationsForTesting calls getSelectedOperations method (for testing).
func (m *AppsModel) GetSelectedOperationsForTesting() []SelectedOperation {
	return m.getSelectedOperations()
}

// handleCompletedOperations immediately updates app statuses based on completed operations.
func (m *AppsModel) handleCompletedOperations(operations []SelectedOperation) {
	for _, operation := range operations {
		switch operation.Operation {
		case StateInstall:
			// App was successfully installed
			m.updateAppStatus(operation.AppKey, true)
		case StateUninstall:
			// App was successfully uninstalled
			m.updateAppStatus(operation.AppKey, false)
		}
	}
}

// Cleanup closes the debug logger when the model is done.
func (m *AppsModel) Cleanup() {
	// No cleanup needed currently
}

// updateSearchQuery updates the search query and filters apps accordingly.
func (m *AppsModel) updateSearchQuery(query string) {
	m.searchQuery = query

	// Search is active when searchActive flag is set (regardless of query content)
	if m.searchActive {
		m.activateSearchMode(query)
	} else {
		m.deactivateSearchMode()
	}
}

// activateSearchMode handles search activation logic.
func (m *AppsModel) activateSearchMode(query string) {
	// When search becomes active, focus starts on search field
	if !m.searchHasFocus {
		m.searchHasFocus = true
	}

	// Perform fuzzy search (empty query shows all apps)
	m.filteredApps = m.performFuzzySearch(query)

	// Auto-select first result if any results found
	if len(m.filteredApps) > 0 {
		m.searchSelection = 0
	} else {
		m.searchSelection = -1 // No results
	}

	// Reset viewport scroll position for new search results
	if m.ready {
		m.viewport.SetYOffset(0)
	}
}

// deactivateSearchMode handles search deactivation logic.
func (m *AppsModel) deactivateSearchMode() {
	// Clear filtered results and reset selection
	m.filteredApps = nil
	m.searchSelection = -1
	m.searchHasFocus = false
}

// No custom data structures needed for lithammer/fuzzysearch - it works with string slices

// performFuzzySearch performs hybrid fuzzy search with word boundary preference and distance filtering.
// Empty query returns all apps.
func (m *AppsModel) performFuzzySearch(query string) []app {
	// Collect apps from all categories with filter application
	var allApps []app
	for _, cat := range m.categories {
		for _, app := range cat.apps {
			// Apply installation status filter
			if !m.passesInstallStatusFilter(app) {
				continue
			}

			// Apply package type filter
			if !m.passesPackageTypeFilter(app) {
				continue
			}

			allApps = append(allApps, app)
		}
	}

	// Apply sorting to collected apps FIRST
	allApps = m.sortApps(allApps)

	// Build search strings AFTER sorting to maintain sync
	searchStrings := make([]string, 0, len(allApps))

	for _, app := range allApps {
		// Search in both name and description for better matches
		searchStr := app.Name + " " + app.Description
		searchStrings = append(searchStrings, searchStr)
	}

	// If query is empty, return all sorted apps
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return allApps
	}

	// Phase 1: Find exact word boundary matches (highest priority)
	var exactMatches []app

	for appIndex, searchStr := range searchStrings {
		lowerSearchStr := strings.ToLower(searchStr)
		words := strings.Fields(lowerSearchStr)

		// Check if query appears in any word
		for _, word := range words {
			if strings.Contains(word, query) {
				app := allApps[appIndex]
				exactMatches = append(exactMatches, app)

				break
			}
		}
	}

	// Only use exact word matches - no fuzzy matching to avoid false positives
	result := exactMatches

	// Sort the combined results based on current sort option
	result = m.sortApps(result)

	return result
}

// renderSearchResults renders the search results in a flat list without categories.
func (m *AppsModel) renderSearchResults() string {
	if len(m.filteredApps) == 0 {
		if m.searchQuery == "" {
			return "Start typing to search apps..."
		}

		return fmt.Sprintf("No apps found matching '%s'", m.searchQuery)
	}

	// Create a title
	title := fmt.Sprintf("── Search Results for '%s' ── [%d matches] ──", m.searchQuery, len(m.filteredApps))
	styledTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary).
		Render(title)

	// Calculate column widths for search results
	maxNameWidth, maxDescWidth := m.calculateSearchResultWidths(m.filteredApps)

	// Render search results as a flat list
	appLines := make([]string, 0, len(m.filteredApps))
	for appIndex, app := range m.filteredApps {
		indicator := m.getAppIndicator(app, m.selected[app.Key])

		// Format with aligned columns: [indicator] [name] [description] [source]
		line := fmt.Sprintf("%s %-*s  %-*s  %s",
			indicator,
			maxNameWidth, app.Name,
			maxDescWidth, app.Description,
			app.Source)

		// Apply styling - highlight the selected search result
		var styledLine string
		if appIndex == m.searchSelection {
			styledLine = m.styles.Selected.Render(line)
		} else {
			styledLine = m.styles.Unselected.Render(line)
		}

		appLines = append(appLines, styledLine)
	}

	// Compose the complete search view
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		styledTitle,
		"",
		lipgloss.JoinVertical(lipgloss.Left, appLines...),
	)

	// Wrap in a border for consistency
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Primary).
		Padding(1).
		Render(content)
}

// calculateSearchResultWidths determines column widths for search results.
func (m *AppsModel) calculateSearchResultWidths(apps []app) (int, int) {
	maxNameWidth := 0
	maxDescWidth := 0

	for _, app := range apps {
		nameLen := len(app.Name)
		descLen := len(app.Description)

		if nameLen > maxNameWidth {
			maxNameWidth = nameLen
		}

		if descLen > maxDescWidth {
			maxDescWidth = descLen
		}
	}

	return maxNameWidth, maxDescWidth
}

// passesInstallStatusFilter checks if app passes the installation status filter.
func (m *AppsModel) passesInstallStatusFilter(app app) bool {
	switch m.installStatusFilter {
	case FilterAll:
		return true
	case FilterInstalled:
		return app.Installed
	case FilterNotInstalled:
		return !app.Installed
	default:
		return true // Default to show all if unknown filter value
	}
}

// passesPackageTypeFilter checks if app passes the package type filter.
func (m *AppsModel) passesPackageTypeFilter(app app) bool {
	if m.packageTypeFilter == FilterAll {
		return true
	}

	// Map app source to package type
	appPackageType := getPackageTypeFromSource(app.Source)

	return appPackageType == m.packageTypeFilter
}

// getPackageTypeFromSource determines package type from app source.
func getPackageTypeFromSource(source string) string {
	source = strings.ToLower(source)

	switch {
	case strings.Contains(source, "apt"):
		return MethodAPTDisplay
	case strings.Contains(source, "flatpak"):
		return "flatpak"
	case strings.Contains(source, "snap"):
		return "snap"
	case strings.Contains(source, ".deb"):
		return "deb"
	case strings.Contains(source, "mise"):
		return "mise"
	case strings.Contains(source, "aqua"):
		return "aqua"
	case strings.Contains(source, "github"):
		return "github"
	case strings.Contains(source, "script"):
		return "script"
	default:
		return "script" // Default to script for unknown sources
	}
}

// sortApps sorts a slice of apps based on the current sort option.
func (m *AppsModel) sortApps(apps []app) []app {
	if len(apps) == 0 {
		return apps
	}

	// Create a copy to avoid modifying the original slice
	sortedApps := make([]app, len(apps))
	copy(sortedApps, apps)

	switch m.sortOption {
	case "Name":
		m.sortByName(sortedApps)
	case "Status":
		m.sortByStatus(sortedApps)
	case "Type":
		m.sortByType(sortedApps)
	case "Category":
		m.sortByCategory(sortedApps)
	default:
		m.sortByName(sortedApps)
	}

	return sortedApps
}

// sortByName sorts apps alphabetically by name.
func (m *AppsModel) sortByName(apps []app) {
	slices.SortFunc(apps, func(a, b app) int {
		return cmp.Compare(a.Name, b.Name)
	})
}

// sortByStatus sorts apps by installation status: Installed first, then not installed.
func (m *AppsModel) sortByStatus(apps []app) {
	slices.SortFunc(apps, func(first, second app) int {
		if first.Installed == second.Installed {
			return cmp.Compare(first.Name, second.Name) // Same status, sort by name
		}

		if first.Installed {
			return -1 // Installed apps come first
		}

		return 1
	})
}

// sortByType sorts apps by package type (source).
func (m *AppsModel) sortByType(apps []app) {
	slices.SortFunc(apps, func(first, second app) int {
		typeFirst := getPackageTypeFromSource(first.Source)

		typeSecond := getPackageTypeFromSource(second.Source)
		if typeFirst == typeSecond {
			return cmp.Compare(first.Name, second.Name) // Same type, sort by name
		}

		return cmp.Compare(typeFirst, typeSecond)
	})
}

// sortByCategory sorts apps by category.
func (m *AppsModel) sortByCategory(apps []app) {
	slices.SortFunc(apps, func(first, second app) int {
		categoryFirst := m.getAppCategory(first)

		categorySecond := m.getAppCategory(second)
		if categoryFirst == categorySecond {
			return cmp.Compare(first.Name, second.Name) // Same category, sort by name
		}

		return cmp.Compare(categoryFirst, categorySecond)
	})
}

// getAppCategory determines the category of an app based on its characteristics.
func (m *AppsModel) getAppCategory(app app) string {
	name := strings.ToLower(app.Name)
	description := strings.ToLower(app.Description)

	if m.isDevelopmentApp(name, description) {
		return "Development"
	}

	if m.isMediaApp(name, description) {
		return "Media"
	}

	if m.isProductivityApp(name, description) {
		return "Productivity"
	}

	if m.isSystemApp(name, description) {
		return "System"
	}

	if m.isCommunicationApp(name, description) {
		return "Communication"
	}

	// Default category
	return "Other"
}

func (m *AppsModel) isDevelopmentApp(name, description string) bool {
	return stringutil.ContainsAny(name, []string{"code", "vim", "git", "docker", "python", "node", "java", "rust", "go"}) ||
		stringutil.ContainsAny(description, []string{"editor", "development", "programming", "code", "developer"})
}

func (m *AppsModel) isMediaApp(name, description string) bool {
	return stringutil.ContainsAny(name, []string{"vlc", "spotify", "gimp", "blender", "obs"}) ||
		stringutil.ContainsAny(description, []string{"media", "video", "audio", "music", "graphics", "image"})
}

func (m *AppsModel) isProductivityApp(name, description string) bool {
	return stringutil.ContainsAny(name, []string{"office", "libre", "calc", "writer", "notes"}) ||
		stringutil.ContainsAny(description, []string{"office", "productivity", "document", "spreadsheet", "notes"})
}

func (m *AppsModel) isSystemApp(name, description string) bool {
	return stringutil.ContainsAny(name, []string{"system", "monitor", "htop", "disk", "backup"}) ||
		stringutil.ContainsAny(description, []string{"system", "utility", "monitor", "performance", "backup"})
}

func (m *AppsModel) isCommunicationApp(name, description string) bool {
	return stringutil.ContainsAny(name, []string{"discord", "slack", "teams", "telegram", "signal"}) ||
		stringutil.ContainsAny(description, []string{"chat", "messaging", "communication", "social"})
}
