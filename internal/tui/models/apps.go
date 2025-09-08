// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements application selection and installation UI.
//
//nolint:funcorder // This file groups methods by functionality rather than export/unexport order for better readability
package models

import (
	"cmp"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/stringutil"
	"github.com/janderssonse/karei/internal/tui/styles"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Status indicators for applications.
const (
	StatusNotInstalled = " " // Empty space for unselected items
	StatusInstalled    = "✓" // Checkmark for installed
	StatusSelected     = "✓" // Checkmark for selected to install
	StatusUninstall    = "✗" // X mark for pending removal
	StatusPending      = "⋯" // Status pending/checking
)

// Filter constants for application filtering.
const (
	FilterAll          = "All"
	FilterInstalled    = "Installed"
	FilterNotInstalled = "Not Installed"
	MethodAPTDisplay   = "apt"
	MethodDEBDisplay   = "deb"
	MethodFlatpak      = "flatpak"
	MethodSnap         = "snap"
	MethodMise         = "mise"
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

	// Fast lookup map for O(1) app access
	appLookup map[string]*app // Key -> app pointer for fast updates

	// Navigation state
	currentCat int

	// Viewport for proper scrolling (no pagination hacks)
	viewport viewport.Model

	ready bool

	// Batch status update tracking
	statusUpdatePending bool      // True when status updates are accumulating
	lastViewportUpdate  time.Time // Track last viewport update to throttle
	contentNeedsUpdate  bool      // True when viewport content needs re-rendering

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

	// Help modal
	helpModal *HelpModal
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
	Version       string // Version if available
	Installed     bool
	Selected      bool
	StatusPending bool // True when installation status is being checked
}

// StatusUpdateMsg carries installation status updates from async checks.
type StatusUpdateMsg struct {
	AppName   string
	Installed bool
}

// VersionUpdateMsg carries version information from package manager queries.
type VersionUpdateMsg struct {
	AppKey  string
	Version string
}

// RefreshStatusMsg triggers a refresh of all app installation statuses.
type RefreshStatusMsg struct{}

// ViewportRefreshMsg triggers a viewport content refresh.
type ViewportRefreshMsg struct{}

// StartStatusCheckMsg triggers the initial status checking after UI is ready.
type StartStatusCheckMsg struct{}

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
	appLookup := make(map[string]*app) // Fast lookup map

	// Convert external categories to internal format
	for catIdx, cat := range appCategories {
		apps := make([]app, 0, len(cat.Applications))
		for _, application := range cat.Applications {
			newApp := app{
				Key:           application.Key,
				Name:          application.Name,
				Description:   application.Description,
				Source:        application.Source,
				Version:       "", // Version will be populated by package manager queries
				Installed:     application.Installed,
				Selected:      false,
				StatusPending: true, // Start with pending status, will be updated async
			}
			apps = append(apps, newApp)
		}

		categories = append(categories, category{
			name:       cat.Name,
			apps:       apps,
			selected:   selected,
			currentApp: 0,
		})

		// Now store pointers to the actual apps in the categories
		for appIdx := range categories[catIdx].apps {
			appLookup[categories[catIdx].apps[appIdx].Key] = &categories[catIdx].apps[appIdx]
		}
	}

	// Create help modal
	helpModal := NewHelpModal()
	helpModal.SetScreen("apps")
	helpModal.SetSize(width, height)

	model := &AppsModel{
		styles:             styleConfig,
		width:              width,
		height:             height,
		ctx:                ctx, // Store parent context
		categories:         categories,
		selected:           selected,
		appLookup:          appLookup,                 // Fast lookup map
		appsManager:        apps.NewTUIManager(false), // Use TUI-optimized manager to suppress command output
		keyMap:             DefaultAppsKeyMap(),
		viewport:           viewport.New(width, height),
		lastViewportUpdate: time.Now(),
		contentNeedsUpdate: true, // Initial render needed

		// Initialize filter and sort states
		installStatusFilter: "All",
		packageTypeFilter:   "All",
		sortOption:          "Name",

		// Help modal
		helpModal: helpModal,
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
		// Generous timeouts - since it's async, it won't block UI
		timeout := 10 * time.Second // Increased default timeout for reliability

		// Get app to determine timeout based on method
		if app, exists := apps.Apps[appName]; exists {
			switch app.Method {
			case domain.MethodBinary, domain.MethodMise, domain.MethodAqua:
				timeout = 5 * time.Second // Binary checks need more time
			case domain.MethodAPT, domain.MethodDEB:
				timeout = 10 * time.Second // APT can be slow
			case domain.MethodFlatpak, domain.MethodSnap:
				timeout = 15 * time.Second // These are often very slow
			}
		}

		ctx, cancel := context.WithTimeout(parentCtx, timeout)
		defer cancel()

		// This runs in its own goroutine via Bubble Tea, won't block UI
		installed := manager.IsAppInstalled(ctx, appName)

		return StatusUpdateMsg{
			AppName:   appName,
			Installed: installed,
		}
	}
}

// fetchAppVersion returns a command to fetch the version of an installed app.
func fetchAppVersion(appKey, appName, source string) tea.Cmd {
	return func() tea.Msg {
		var version string

		// Get version using simple command execution
		cmd := getVersionCommand(appName, source)
		if cmd != "" {
			// Execute command with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Use os/exec directly for simplicity
			output, err := execCommand(ctx, "sh", "-c", cmd)
			if err == nil && output != "" {
				version = extractVersion(output, source)
			}
		}

		return VersionUpdateMsg{
			AppKey:  appKey,
			Version: version,
		}
	}
}

// execCommand executes a command with context.
func execCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()

	return string(output), err
}

// getVersionCommand returns the appropriate command to get version info.
// This is designed to be extensible and handle various package manager quirks.
func getVersionCommand(appName, source string) string {
	normalizedName := strings.ToLower(appName)

	// Handle special case for mise
	if source == MethodMise {
		return getMiseVersionCommand(normalizedName)
	}

	// Map sources to their version commands
	versionCommands := map[string]string{
		MethodAPTDisplay: fmt.Sprintf("dpkg -s %s 2>/dev/null | grep '^Version:' | cut -d' ' -f2", appName),
		MethodDEBDisplay: fmt.Sprintf("dpkg -s %s 2>/dev/null | grep '^Version:' | cut -d' ' -f2", appName),
		MethodFlatpak:    fmt.Sprintf("flatpak info %s 2>/dev/null | grep 'Version:' | awk '{print $2}'", appName),
		MethodSnap:       fmt.Sprintf("snap info %s 2>/dev/null | grep 'installed:' | awk '{print $2}'", normalizedName),
		"aqua":           fmt.Sprintf("aqua list 2>/dev/null | grep -i '^%s\\s' | awk '{print $2}'", normalizedName),
		"cargo":          fmt.Sprintf("cargo install --list | grep -E '^%s\\s' | awk '{print $2}' | tr -d '()'", normalizedName),
		"npm":            fmt.Sprintf("npm list -g %s 2>/dev/null | grep %s@ | sed 's/.*@//'", normalizedName, normalizedName),
		"yarn":           fmt.Sprintf("npm list -g %s 2>/dev/null | grep %s@ | sed 's/.*@//'", normalizedName, normalizedName),
		"pnpm":           fmt.Sprintf("npm list -g %s 2>/dev/null | grep %s@ | sed 's/.*@//'", normalizedName, normalizedName),
		"pip":            fmt.Sprintf("pip show %s 2>/dev/null | grep '^Version:' | awk '{print $2}'", normalizedName),
		"pipx":           fmt.Sprintf("pip show %s 2>/dev/null | grep '^Version:' | awk '{print $2}'", normalizedName),
	}

	if cmd, exists := versionCommands[source]; exists {
		return cmd
	}

	// Default: try common version flags
	return fmt.Sprintf("which %s > /dev/null 2>&1 && %s --version 2>/dev/null | head -1", normalizedName, normalizedName)
}

// getMiseVersionCommand returns the version command for mise-managed tools.
func getMiseVersionCommand(normalizedName string) string {
	if normalizedName == MethodMise {
		// Special case: mise itself uses --version
		return "which mise > /dev/null 2>&1 && mise --version 2>/dev/null | head -1 | cut -d' ' -f1"
	}
	// For mise-managed tools: match tool name flexibly
	// Tools can appear as "toolname", "aqua:org/toolname", "aqua:toolname"
	return fmt.Sprintf("mise list 2>/dev/null | awk 'tolower($1) ~ /(^|[:/])%s$/ {print $2; exit}'", normalizedName)
}

// extractVersion extracts and cleans the version string.
func extractVersion(output, source string) string {
	version := strings.TrimSpace(output)

	// Clean up debian version format (epoch:version-revision)
	if source == MethodAPTDisplay || source == MethodDEBDisplay {
		// Remove epoch
		if idx := strings.Index(version, ":"); idx != -1 {
			version = version[idx+1:]
		}
		// Remove debian revision if it's long
		if idx := strings.Index(version, "-"); idx != -1 && idx > 5 {
			version = version[:idx]
		}
	}

	// Truncate if too long
	if len(version) > 15 {
		version = version[:12] + "..."
	}

	return version
}

// BatchStatusCheckMsg triggers the next batch of status checks.
type BatchStatusCheckMsg struct {
	BatchIndex int
}

// startBatchedStatusCheck initiates batched status checking to avoid system overload.
func (m *AppsModel) startBatchedStatusCheck() tea.Cmd {
	// Start with visible apps for immediate feedback
	return m.checkVisibleAppsFirst()
}

// checkVisibleAppsFirst checks currently visible apps before others.
func (m *AppsModel) checkVisibleAppsFirst() tea.Cmd {
	// Don't check anything immediately - just schedule the first small batch
	// This prevents blocking the UI on startup
	return func() tea.Msg {
		// Small delay to let UI initialize first
		time.Sleep(50 * time.Millisecond)
		return BatchStatusCheckMsg{BatchIndex: 0}
	}
}

// checkCategoryApps checks ONE app at a time to keep UI 100% responsive.
func (m *AppsModel) checkCategoryApps(_, batchIndex int) tea.Cmd {
	// Only check ONE app at a time
	// This is key: Bubble Tea runs commands in separate goroutines,
	// so even if the check takes time, it won't block the UI
	var appToCheck string

	found := false

	// Find next unchecked app
	for catIdx := 0; catIdx < len(m.categories) && !found; catIdx++ {
		cat := m.categories[catIdx]
		for _, a := range cat.apps {
			if a.StatusPending {
				appToCheck = a.Key
				found = true

				break
			}
		}
	}

	if !found {
		return nil // All apps checked
	}

	// Check just ONE app, then immediately schedule next
	// The status check runs in a goroutine and won't block
	return tea.Sequence(
		NewStatusCheckCommand(m.ctx, appToCheck, m.appsManager),
		func() tea.Msg {
			// Immediately queue next check - the previous one is still running async
			return BatchStatusCheckMsg{BatchIndex: batchIndex + 1}
		},
	)
}

// Init initializes the apps model.
func (m *AppsModel) Init() tea.Cmd {
	// Don't start status checking immediately - let UI initialize first
	return nil
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the AppsModel.
//
//nolint:cyclop // Complex but necessary for handling various UI interactions
func (m *AppsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Let viewport handle certain messages for scrolling
	var viewportCmd tea.Cmd

	shouldPassToViewport := false

	// Check if this is a key that viewport should handle
	if keyMsg, isKeyMsg := msg.(tea.KeyMsg); isKeyMsg {
		switch keyMsg.String() {
		case "pgup", "pgdown", "ctrl+u", "ctrl+d":
			// These are viewport scrolling keys
			shouldPassToViewport = true
		}
	} else if _, isMouseMsg := msg.(tea.MouseMsg); isMouseMsg {
		// Mouse wheel events
		shouldPassToViewport = true
	}

	if shouldPassToViewport {
		m.viewport, viewportCmd = m.viewport.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model, cmd := m.handleWindowResize(msg)
		return model, tea.Batch(cmd, viewportCmd)

	case StatusUpdateMsg:
		model, cmd := m.handleStatusUpdate(msg)
		return model, tea.Batch(cmd, viewportCmd)

	case VersionUpdateMsg:
		model, cmd := m.handleVersionUpdate(msg)
		return model, tea.Batch(cmd, viewportCmd)

	case SmoothScrollMsg:
		// Handle smooth scrolling after navigation
		m.ensureSelectionVisible()
		return m, viewportCmd

	case StartStatusCheckMsg:
		// Start checking apps now that UI is ready
		return m, tea.Batch(m.checkCategoryApps(0, 0), viewportCmd)

	case BatchStatusCheckMsg:
		// Continue checking the next batch of apps
		return m, tea.Batch(m.checkCategoryApps(m.currentCat, msg.BatchIndex), viewportCmd)

	case RefreshStatusMsg:
		// Refresh all app statuses using batched approach
		return m, tea.Batch(m.startBatchedStatusCheck(), viewportCmd)

	case CompletedOperationsMsg:
		// Handle immediate status updates from completed operations
		m.handleCompletedOperations(msg.Operations)

		return m, viewportCmd

	case SearchUpdateMsg:
		m.searchActive = msg.Active
		m.updateSearchQuery(msg.Query)

		return m, viewportCmd

	case FilterUpdateMsg:
		m.installStatusFilter = msg.InstallStatus
		m.packageTypeFilter = msg.PackageType
		m.sortOption = msg.SortOption

		return m, viewportCmd

	case tea.KeyMsg:
		model, cmd := m.handleKeyMessage(msg)
		return model, tea.Batch(cmd, viewportCmd)
	}

	return m, viewportCmd
}

// handleStatusUpdate handles status update messages.
//
//nolint:unparam // Cmd return is always nil but signature matches Update pattern
func (m *AppsModel) handleVersionUpdate(msg VersionUpdateMsg) (tea.Model, tea.Cmd) {
	// Update version for the app
	for i := range m.categories {
		for j := range m.categories[i].apps {
			if m.categories[i].apps[j].Key == msg.AppKey {
				m.categories[i].apps[j].Version = msg.Version
				m.contentNeedsUpdate = true

				return m, nil
			}
		}
	}

	return m, nil
}

func (m *AppsModel) handleStatusUpdate(msg StatusUpdateMsg) (tea.Model, tea.Cmd) {
	m.updateAppStatus(msg.AppName, msg.Installed)

	// No throttling - this is the idiomatic Bubble Tea way
	// View() will be called automatically after Update()
	// No need for flags or throttling

	// If app is installed, fetch its version
	var versionCmd tea.Cmd

	// Special case: always check mise version since detection might be unreliable
	if msg.AppName == MethodMise {
		msg.Installed = true
	}

	if msg.Installed {
		versionCmd = m.createVersionFetchCommand(msg.AppName)
	}

	return m, versionCmd
}

// createVersionFetchCommand creates a command to fetch the version of an installed app.
func (m *AppsModel) createVersionFetchCommand(appName string) tea.Cmd {
	// Find the app to get its details
	// appName contains the key (lowercase), not the display name
	for _, cat := range m.categories {
		for _, app := range cat.apps {
			if app.Key == appName {
				normalizedSource, toolIdentifier := m.detectAppSource(app)
				return fetchAppVersion(app.Key, toolIdentifier, normalizedSource)
			}
		}
	}

	return nil
}

// detectAppSource determines the package manager and tool identifier for an app.
func (m *AppsModel) detectAppSource(app app) (normalizedSource string, toolIdentifier string) {
	toolIdentifier = app.Name // Default to display name

	if catalogApp, exists := apps.Apps[app.Key]; exists {
		// Use method-based detection for accurate package manager identification
		normalizedSource, toolIdentifier = m.detectSourceFromMethod(catalogApp, app)
	} else {
		// No catalog entry - use heuristic detection
		normalizedSource = getPackageTypeFromSource(app.Source)
	}

	return normalizedSource, toolIdentifier
}

// detectSourceFromMethod determines the source based on the catalog app's method.
func (m *AppsModel) detectSourceFromMethod(catalogApp apps.App, app app) (normalizedSource string, toolIdentifier string) {
	toolIdentifier = app.Name // Default

	switch catalogApp.Method {
	case domain.MethodMise:
		return MethodMise, app.Key // Mise uses lowercase keys
	case domain.MethodFlatpak:
		return "flatpak", catalogApp.Source // Flatpak uses the source ID
	case domain.MethodSnap:
		return "snap", toolIdentifier
	case domain.MethodAPT:
		return MethodAPTDisplay, toolIdentifier
	case domain.MethodDEB:
		return MethodDEBDisplay, toolIdentifier
	default:
		// For other methods, check if source contains hints
		return m.detectSourceFromHints(app.Source), toolIdentifier
	}
}

// detectSourceFromHints detects the source from string hints.
func (m *AppsModel) detectSourceFromHints(source string) string {
	lowerSource := strings.ToLower(source)
	switch {
	case strings.Contains(lowerSource, "cargo"):
		return "cargo"
	case strings.Contains(lowerSource, "npm"):
		return "npm"
	case strings.Contains(lowerSource, "pip"):
		return "pip"
	default:
		// Fall back to source-based detection
		return getPackageTypeFromSource(source)
	}
}

// handleWindowResize handles window resize messages.
func (m *AppsModel) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Update help modal size
	if m.helpModal != nil {
		m.helpModal.SetSize(msg.Width, msg.Height)
	}

	// Calculate header and footer heights for viewport
	header := m.renderCleanHeader()
	footer := m.renderCleanFooter()

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)

	// Reserve space for optional details panel
	detailsHeight := 0
	if m.height > 30 {
		// Details panel: 3 lines content + 2 for border + 1 for margin = 6 total
		detailsHeight = 6
	}

	// Calculate viewport height (leave room for header, footer, and details)
	// Add extra buffer to ensure header stays visible
	viewportHeight := max(msg.Height-headerHeight-footerHeight-detailsHeight-2, 1)

	if !m.ready {
		m.viewport = viewport.New(msg.Width, viewportHeight)
		m.contentNeedsUpdate = true
		m.ready = true

		return m, tea.Tick(time.Millisecond*200, func(_ time.Time) tea.Msg {
			return StartStatusCheckMsg{}
		})
	}

	// Update existing viewport
	m.viewport.Width = msg.Width
	m.viewport.Height = viewportHeight
	m.contentNeedsUpdate = true

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

	// If help modal is visible, show it as an overlay
	if m.helpModal != nil && m.helpModal.IsVisible() {
		return m.renderWithModal()
	}

	return m.renderBaseView()
}

// renderWithModal renders the view with modal overlay.
func (m *AppsModel) renderWithModal() string {
	// Get modal view
	modalView := m.helpModal.View()

	// The idiomatic Bubble Tea approach: center the modal on a dark background
	// This provides a clean modal experience without complex compositing
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modalView,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("235")), // Dark gray background
	)
}

// renderBaseView renders the main application view without overlays.
func (m *AppsModel) renderBaseView() string {
	// Build the complete view: header + content + footer
	components := []string{}

	// Add the new clean header - always show it
	header := m.renderCleanHeader()
	if header != "" {
		components = append(components, header)
	}

	// Only update viewport content when necessary
	if m.contentNeedsUpdate {
		// Save current scroll position before updating content
		currentOffset := m.viewport.YOffset

		// Update content
		m.viewport.SetContent(m.renderAllCategories())
		m.contentNeedsUpdate = false

		// Restore scroll position if in search mode (to prevent jumping)
		if m.searchActive && !m.searchHasFocus {
			m.viewport.SetYOffset(currentOffset)
			// After restoring position, ensure selection is still visible
			m.ensureSearchSelectionVisible()
		}
	}

	// Add main content
	components = append(components, m.viewport.View())

	// Add details panel below main content if space allows
	if m.height > 30 && len(m.categories) > 0 {
		// Show compact details below
		detailsPanel := m.renderCompactDetails()
		if detailsPanel != "" {
			components = append(components, detailsPanel)
		}
	}

	// Add the new clean footer
	footer := m.renderCleanFooter()
	components = append(components, footer)

	// Compose with Lipgloss
	if len(components) == 1 {
		return components[0]
	}

	return lipgloss.JoinVertical(lipgloss.Top, components...)
}

// renderCleanHeader renders the new simplified header format: "Karei » Package Selection" with status.
func (m *AppsModel) renderCleanHeader() string {
	// If search is active, show search bar instead of normal header
	if m.searchActive {
		return m.renderSearchHeader()
	}

	// Left side: App name » Current location
	location := "Karei » Package Selection"
	leftSide := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary).
		Render(location)

	// Right side: Status (selected count)
	selectedCount := 0

	for _, state := range m.selected {
		if state == StateInstall {
			selectedCount++
		}
	}

	status := ""
	if selectedCount > 0 {
		status = fmt.Sprintf("%d selected", selectedCount)
	}

	rightSide := lipgloss.NewStyle().
		Foreground(m.styles.Muted).
		Render(status)

	// Calculate spacing
	totalWidth := m.width
	leftWidth := lipgloss.Width(leftSide)
	rightWidth := lipgloss.Width(rightSide)
	spacerWidth := totalWidth - leftWidth - rightWidth - 4 // Account for padding

	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := strings.Repeat(" ", spacerWidth)

	// Combine with spacing
	headerLine := leftSide + spacer + rightSide

	// Style the header with subtle border
	return lipgloss.NewStyle().
		Padding(0, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width).
		Render(headerLine)
}

// renderSearchHeader renders the search bar in the header area.
func (m *AppsModel) renderSearchHeader() string {
	// Search prompt with query
	searchPrompt := "/"
	if m.searchHasFocus {
		searchPrompt = lipgloss.NewStyle().
			Foreground(m.styles.Primary).
			Bold(true).
			Render("/")
	}

	// Build search bar content
	searchContent := fmt.Sprintf("%s %s", searchPrompt, m.searchQuery)

	// Add cursor if search has focus
	if m.searchHasFocus {
		searchContent += lipgloss.NewStyle().
			Foreground(m.styles.Primary).
			Blink(true).
			Render("█")
	}

	// Show match count on the right
	matchInfo := ""
	if len(m.filteredApps) > 0 {
		matchInfo = fmt.Sprintf("%d matches", len(m.filteredApps))
	} else if m.searchQuery != "" {
		matchInfo = "No matches"
	}

	rightSide := lipgloss.NewStyle().
		Foreground(m.styles.Muted).
		Render(matchInfo)

	// Calculate spacing
	leftWidth := lipgloss.Width(searchContent)
	rightWidth := lipgloss.Width(rightSide)
	spacerWidth := m.width - leftWidth - rightWidth - 4

	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := strings.Repeat(" ", spacerWidth)
	headerLine := searchContent + spacer + rightSide

	// Style with search-specific border color
	return lipgloss.NewStyle().
		Padding(0, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(m.styles.Primary). // Highlight border when searching
		Width(m.width).
		Render(headerLine)
}

// renderCleanFooter renders the new simplified footer with 3-4 actions + help.
func (m *AppsModel) renderCleanFooter() string {
	// Context-aware footer actions with styled keys and descriptions
	var actions []string

	// Styles for different parts
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary) // Keys in primary color (blue)

	bracketStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary) // Brackets also in primary color

	actionStyle := lipgloss.NewStyle().
		Foreground(m.styles.Muted) // Actions in muted color

	// Helper function to format action
	formatAction := func(key, action string) string {
		return bracketStyle.Render("[") +
			keyStyle.Render(key) +
			bracketStyle.Render("]") +
			" " +
			actionStyle.Render(action)
	}

	if m.searchActive {
		// Search mode actions
		actions = []string{
			formatAction("Enter", "Select"),
			formatAction("Tab", "Results"),
			formatAction("Esc", "Cancel"),
		}
	} else {
		// Normal mode actions
		actions = []string{
			formatAction("Space", "Select"),
			formatAction("Enter", "Install"),
			formatAction("u", "Uninstall"),
			formatAction("/", "Search"),
		}
	}

	// Always add help with special styling (dim yellow to stand out)
	helpKey := bracketStyle.Render("[") +
		lipgloss.NewStyle().Bold(true).Foreground(m.styles.Warning).Render("?") +
		bracketStyle.Render("]")
	actions = append(actions, helpKey+" "+actionStyle.Render("Help"))

	// Join actions with more spacing
	footerText := strings.Join(actions, "   ")

	// Style the footer container
	return lipgloss.NewStyle().
		Padding(0, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width).
		Render(footerText)
}

// renderCompactDetails renders a boxed 3-line details panel for the current app.
func (m *AppsModel) renderCompactDetails() string {
	// Get current app
	if m.currentCat >= len(m.categories) {
		return ""
	}

	cat := m.categories[m.currentCat]
	if cat.currentApp >= len(cat.apps) {
		return ""
	}

	app := cat.apps[cat.currentApp]

	// Build status string
	status := "Not installed"
	statusColor := m.styles.MutedText

	if app.Installed {
		status = "Installed"
		statusColor = m.styles.SuccessText
	}

	// Build three lines of content
	lines := make([]string, 3)

	// Line 1: Name and status
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(m.styles.Primary)
	line1Left := nameStyle.Render(app.Name)

	line1Right := statusColor.Render(status)
	if app.Version != "" {
		line1Right = fmt.Sprintf("v%s • %s", app.Version, line1Right)
		line1Right = lipgloss.NewStyle().Foreground(m.styles.Muted).Render(line1Right)
	}

	// Calculate spacing for right alignment
	// Match the EXACT content width of categories: 82 chars
	// (indicator + space + name(22) + spaces(2) + desc(42) + spaces(2) + source(12))
	const categoryContentWidth = 82

	line1Width := categoryContentWidth

	line1SpacerWidth := line1Width - lipgloss.Width(line1Left) - lipgloss.Width(line1Right)
	if line1SpacerWidth < 1 {
		line1SpacerWidth = 1
	}

	lines[0] = line1Left + strings.Repeat(" ", line1SpacerWidth) + line1Right

	// Line 2: Description (truncate to fit)
	descStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)
	truncatedDesc := truncate(app.Description, categoryContentWidth)
	lines[1] = descStyle.Render(truncatedDesc)

	// Line 3: Source and package type (truncate to fit)
	sourceStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)
	sourceText := "Source: " + app.Source
	truncatedSource := truncate(sourceText, categoryContentWidth)
	lines[2] = sourceStyle.Render(truncatedSource)

	// Join lines
	content := strings.Join(lines, "\n")

	// Style as a boxed panel - match category box styling
	// No explicit width, same as categories
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1). // Same padding as categories
		Render(content)
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

//nolint:funcorder,cyclop // Methods grouped logically by functionality, complex but necessary
func (m *AppsModel) handleKeyMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle help modal toggle first
	if msg.String() == "?" {
		if m.helpModal != nil {
			m.helpModal.Toggle()
		}

		return m, nil
	}

	// If help modal is visible, let it handle keys
	if m.helpModal != nil && m.helpModal.IsVisible() {
		if cmd := m.helpModal.Update(msg); cmd != nil {
			return m, cmd
		}
		// Help modal consumed the key event
		return m, nil
	}

	// Handle search activation/deactivation
	switch {
	case msg.String() == "/":
		// Activate search mode (idiomatic pattern - handle own search)
		m.searchActive = true
		m.searchHasFocus = true
		m.searchQuery = ""

		// Initialize with all apps for empty query (default behavior)
		m.updateSearchResults()

		// Mark content for re-render to show search results
		m.contentNeedsUpdate = true

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

		// Mark content for re-render to show categories again
		m.contentNeedsUpdate = true

		return m, func() tea.Msg {
			return SearchDeactivatedMsg{PreserveQuery: false, Query: query}
		}
	}

	// Handle context switching with {/} (before search input to allow focus switching)
	if cmd := m.handleContextSwitchKeys(msg); cmd != nil {
		return m, cmd
	}

	// Handle search input when search field has focus (after context switching)
	if m.searchActive && m.searchHasFocus {
		return m, m.handleSearchInput(msg)
	}

	// Handle installation commands
	if installCmd := m.handleInstallationKeys(msg); installCmd != nil {
		return m, installCmd
	}

	// Handle navigation (j/k always work for up/down, {/} for context switch)
	if navCmd := m.handleNavigationKeys(msg); navCmd != nil {
		return m, navCmd
	}

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
	// Use FIXED widths for ALL categories to ensure vertical alignment
	// Don't calculate per-category as that breaks alignment
	const fixedNameWidth = 22

	const fixedDescWidth = 42

	appLines := m.renderAppLines(cat, isCurrent, fixedNameWidth, fixedDescWidth)

	// Create simplified category title
	categoryTitle := fmt.Sprintf("%s (%d)", cat.name, len(cat.apps))

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

// getAppIndicator returns the appropriate colored indicator for an app's state.
func (m *AppsModel) getAppIndicator(app app, selectionState SelectionState) string {
	switch selectionState {
	case StateInstall:
		// Blue checkmark for selected for installation
		return m.styles.PrimaryText.Render("✓")
	case StateUninstall:
		// Red X for selected for removal
		return m.styles.ErrorText.Render("✗")
	default:
		// No selection - show install status
		if app.StatusPending {
			return m.styles.MutedText.Render("⋯") // Status being checked
		}

		if app.Installed {
			// Green checkmark for already installed
			return m.styles.SuccessText.Render("✓")
		}

		// Empty space for not installed
		return " "
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

	// Don't call ensureSelectionVisible here - let async command handle it
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

	// Don't call ensureSelectionVisible here - let async command handle it
}

// jumpToFirst jumps to the first app in the first category (vim g key).
func (m *AppsModel) jumpToFirst() {
	if len(m.categories) == 0 {
		return
	}

	if m.searchActive && !m.searchHasFocus && len(m.filteredApps) > 0 {
		// In search results, jump to first result
		m.searchSelection = 0
		m.contentNeedsUpdate = true
	} else if !m.searchActive {
		// Jump to first app in first category
		m.currentCat = 0
		m.categories[0].currentApp = 0
		m.contentNeedsUpdate = true
	}
}

// jumpToLast jumps to the last app in the last category (vim G key).
func (m *AppsModel) jumpToLast() {
	if len(m.categories) == 0 {
		return
	}

	if m.searchActive && !m.searchHasFocus && len(m.filteredApps) > 0 {
		// In search results, jump to last result
		m.searchSelection = len(m.filteredApps) - 1
		m.contentNeedsUpdate = true
	} else if !m.searchActive {
		// Jump to last app in last category
		m.currentCat = len(m.categories) - 1
		lastCat := &m.categories[m.currentCat]
		lastCat.currentApp = len(lastCat.apps) - 1
		m.contentNeedsUpdate = true
	}
}

// smoothScrollCommand returns a command that triggers smooth scrolling after navigation.
// This decouples navigation from scrolling, making it async and smoother.
func (m *AppsModel) smoothScrollCommand() tea.Cmd {
	// Use a tiny delay to batch rapid navigation (like holding j/k)
	return tea.Tick(time.Millisecond*10, func(_ time.Time) tea.Msg {
		return SmoothScrollMsg{}
	})
}

// navigateSearchDown navigates down in search results.
func (m *AppsModel) navigateSearchDown() {
	if len(m.filteredApps) == 0 {
		return
	}

	if m.searchSelection < len(m.filteredApps)-1 {
		m.searchSelection++
	}
	// Don't call ensureSearchSelectionVisible here - it's called after content update
}

// navigateSearchUp navigates up in search results.
func (m *AppsModel) navigateSearchUp() {
	if len(m.filteredApps) == 0 {
		return
	}

	if m.searchSelection > 0 {
		m.searchSelection--
	}
	// Don't call ensureSearchSelectionVisible here - it's called after content update
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
	// reset invalid state - uninstalled apps cannot be marked for uninstall
	currentState := m.selected[app.Key]
	if currentState == StateUninstall && !app.Installed {
		// Reset invalid state: app is marked for uninstall but is not installed
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

	// Mark content for re-render
	m.contentNeedsUpdate = true
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

	// Mark content for re-render
	m.contentNeedsUpdate = true
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

	// Mark content for re-render
	m.contentNeedsUpdate = true
}

// markForUninstallForSearchResult marks the currently selected search result for uninstallation.
func (m *AppsModel) markForUninstallForSearchResult() {
	if !m.searchActive || m.searchSelection < 0 || m.searchSelection >= len(m.filteredApps) {
		return
	}

	app := m.filteredApps[m.searchSelection]
	m.selected[app.Key] = StateUninstall

	// Mark content for re-render
	m.contentNeedsUpdate = true
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

		// Mark content for re-render to show categories again
		m.contentNeedsUpdate = true

		return func() tea.Msg {
			return SearchDeactivatedMsg{PreserveQuery: true, Query: query}
		}
	}

	return nil
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
	m.filteredApps = m.performFuzzySearch(m.searchQuery)

	// Reset selection
	m.searchSelection = -1
	if len(m.filteredApps) > 0 {
		m.searchSelection = 0
	}

	// Mark content for re-render to show updated search results
	m.contentNeedsUpdate = true
}

// renderNavigationHeader renders the top navigation bar that mirrors the footer.

// renderSearchHeader renders the always-visible search/filter header bar.
// Commented out - replaced by renderCleanHeader for new simplified design

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
		// In search field: { wraps to search results (toggle behavior)
		m.searchHasFocus = false
		if len(m.filteredApps) > 0 {
			// Go to search results
			if m.searchSelection < 0 {
				m.searchSelection = 0
			}
		}

		return func() tea.Msg {
			return ContextSwitchMsg{Direction: "up", Context: "search-field-wrap"}
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
		m.contentNeedsUpdate = true // Mark for re-render

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

	// In search results: } wraps back to search field (toggle behavior)
	m.searchHasFocus = true

	return func() tea.Msg {
		return ContextSwitchMsg{Direction: "down", Context: "search-results-wrap"}
	}
}

func (m *AppsModel) handleDownFromCategories() tea.Cmd {
	// Regular categories: navigate to next category
	if m.currentCat < len(m.categories)-1 {
		m.currentCat++
		m.categories[m.currentCat].currentApp = 0
		m.ensureSelectionVisible()
		m.contentNeedsUpdate = true // Mark for re-render
	}

	return func() tea.Msg {
		return ContextSwitchMsg{Direction: "down", Context: "categories"}
	}
}

// handleNavigationKeys processes navigation key presses with viewport scrolling.
// j/k work for both regular navigation and search results.
func (m *AppsModel) handleNavigationKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keyMap.Down), msg.String() == "j":
		m.handleDownNavigation()
		// Only use smooth scroll for category navigation, not search
		if !m.searchActive {
			return m.smoothScrollCommand()
		}

		return nil
	case key.Matches(msg, m.keyMap.Up), msg.String() == "k":
		m.handleUpNavigation()
		// Only use smooth scroll for category navigation, not search
		if !m.searchActive {
			return m.smoothScrollCommand()
		}

		return nil
	case key.Matches(msg, m.keyMap.PageDown), msg.String() == "J":
		// Scroll viewport down (J key for page navigation)
		m.viewport.ScrollDown(5)
	case key.Matches(msg, m.keyMap.PageUp), msg.String() == "K":
		// Scroll viewport up (K key for page navigation)
		m.viewport.ScrollUp(5)
	case msg.String() == "g":
		// Jump to first app (vim style)
		m.jumpToFirst()
		return m.smoothScrollCommand()
	case msg.String() == "G":
		// Jump to last app (vim style)
		m.jumpToLast()
		return m.smoothScrollCommand()
	}

	return nil
}

func (m *AppsModel) handleDownNavigation() {
	if m.searchActive && !m.searchHasFocus && len(m.filteredApps) > 0 {
		// Navigate down in search results
		m.navigateSearchDown()
		// For search, content needs update since selection highlighting changes
		m.contentNeedsUpdate = true
	} else if !m.searchActive {
		// Navigate down in regular categories
		m.navigateDown()
		// Mark content for update to refresh details panel
		m.contentNeedsUpdate = true
	}
}

func (m *AppsModel) handleUpNavigation() {
	if m.searchActive && !m.searchHasFocus && len(m.filteredApps) > 0 {
		// Navigate up in search results
		m.navigateSearchUp()
		// For search, content needs update since selection highlighting changes
		m.contentNeedsUpdate = true
	} else if !m.searchActive {
		// Navigate up in regular categories
		m.navigateUp()
		// Mark content for update to refresh details panel
		m.contentNeedsUpdate = true
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

		m.contentNeedsUpdate = true
	case msg.String() == "d":
		if m.searchActive && len(m.filteredApps) > 0 && m.searchSelection >= 0 {
			m.markForUninstallForSearchResult()
		} else {
			m.markForUninstall()
		}

		m.contentNeedsUpdate = true
	}
}

// updateAppStatus updates app installation status using O(1) lookup.
func (m *AppsModel) updateAppStatus(appName string, installed bool) {
	// Fast O(1) lookup instead of O(n²) search
	if app, exists := m.appLookup[appName]; exists {
		app.Installed = installed
		app.StatusPending = false // ALWAYS clear pending state to prevent stuck "..."

		// Clear selection state after any status update
		delete(m.selected, appName)

		// Mark that we have pending status updates
		m.statusUpdatePending = true
	}
}

// ensureSelectionVisible ensures the currently selected app is visible in the viewport.
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
	topBuffer := 3    // Keep 3 lines above selection visible
	bottomBuffer := 3 // Keep 3 lines below selection visible

	// Check if selection is outside comfortable viewing area
	if selectionLine < viewportTop+topBuffer {
		// Selection too close to top - scroll up smoothly
		newOffset := selectionLine - topBuffer
		if newOffset < 0 {
			newOffset = 0
		}

		m.viewport.SetYOffset(newOffset)
	} else if selectionLine > viewportBottom-bottomBuffer {
		// Selection too close to bottom - scroll down smoothly
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
	// Account for the box border and padding structure:
	// Line 0: Top border ╭──────╮
	// Line 1: Padding (empty)
	// Line 2: Title "Search Results (N matches)"
	// Line 3: Empty line after title
	// Line 4+: Search result items
	// ...
	// Line N: Padding (empty)
	// Line N+1: Bottom border ╰──────╯
	line := 0
	line++ // Top border
	line++ // Top padding
	line++ // Title line
	line++ // Empty line after title

	// Add the selection position
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

// truncate truncates a string to maxLen characters, adding "..." if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return "..."
	}

	return s[:maxLen-3] + "..."
}

// renderAppLines creates formatted lines for all apps in a category.
func (m *AppsModel) renderAppLines(cat category, isCurrent bool, nameWidth, descWidth int) []string {
	appLines := make([]string, 0, len(cat.apps))

	// Fixed source column width for alignment
	const sourceWidth = 12 // Wide enough for "github-java" and others

	for appIdx, app := range cat.apps {
		indicator := m.getAppIndicator(app, m.selected[app.Key])

		// Build the line with EXACT formatting for alignment
		// Format: [I] NAME........ DESCRIPTION............          SOURCE
		// Where I = indicator (1 char), followed by space

		// Ensure indicator is always 1 character
		if indicator == " " {
			indicator = " " // Single space
		}

		// Format the main content with fixed widths
		name := truncate(app.Name, nameWidth)
		desc := truncate(app.Description, descWidth)

		// Build main content (indicator + name + description)
		mainContent := fmt.Sprintf("%s %-*s  %-*s",
			indicator,
			nameWidth, name,
			descWidth, desc)

		// Right-align source in a fixed-width column
		// This ensures all sources align regardless of their length
		source := app.Source
		if len(source) > sourceWidth {
			source = truncate(source, sourceWidth)
		}

		// Format source to be right-aligned within sourceWidth
		sourceFormatted := fmt.Sprintf("%*s", sourceWidth, source)

		// Build complete line with consistent spacing
		dimmedSource := m.styles.MutedText.Render(sourceFormatted)

		// Fixed spacing between description and source
		const gapBeforeSource = 2

		line := mainContent + strings.Repeat(" ", gapBeforeSource) + dimmedSource

		// Apply highlighting if this is the current selection
		if isCurrent && appIdx == cat.currentApp {
			// Use a more subtle highlight - not bold, just slightly brighter
			highlightStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("248")) // Subtle gray-white for dim highlight

			// Keep the indicator separate so it maintains its color
			// but highlight the rest of the line
			highlightedContent := fmt.Sprintf("%-*s  %-*s",
				nameWidth, name,
				descWidth, desc)

			highlightedMain := highlightStyle.Render(highlightedContent)

			// Reconstruct the line with original indicator but highlighted content
			line = fmt.Sprintf("%s %s", indicator, highlightedMain) +
				strings.Repeat(" ", gapBeforeSource) + dimmedSource
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

// IsSearchActive returns whether search is currently active.
func (m *AppsModel) IsSearchActive() bool {
	return m.searchActive
}

// GetSearchQuery returns the current search filter text.
func (m *AppsModel) GetSearchQuery() string {
	return m.searchQuery
}

// renderSearchField renders the search input field with visual feedback.
// Commented out - search field now part of footer actions in new design

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

// GetNavigationHints returns screen-specific navigation hints for the footer.
func (m *AppsModel) GetNavigationHints() []string {
	// Screen-specific controls
	if m.searchActive {
		if m.searchHasFocus {
			// Search field has focus
			return []string{
				"Type to Search",
				"[Enter] Done",
				"[{/}] Results",
			}
		}
		// Search results have focus
		return []string{
			"[j/k] Navigate",
			"[Space] Select",
			"[{/}] Search Field",
		}
	}
	// Normal mode - app-specific actions
	return []string{
		"[j/k] Navigate",
		"[{/}] Categories",
		"[Space] Select",
		"[d] Uninstall",
		"[Enter] Install",
		"[/] Search",
	}
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

	// Create a title that looks like a category header
	title := fmt.Sprintf("Search Results (%d matches)", len(m.filteredApps))
	styledTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary).
		Render(title)

	// Use FIXED column widths matching category view for consistency
	// These match the exact widths used in renderAppLines
	const (
		nameWidth            = 22 // Fixed name column width
		descWidth            = 42 // Fixed description column width
		sourceWidth          = 12 // Fixed source column width
		categoryContentWidth = 82 // Total content width matching details panel
	)

	// Render search results with same styling as category items
	appLines := make([]string, 0, len(m.filteredApps))
	for appIndex, app := range m.filteredApps {
		indicator := m.getAppIndicator(app, m.selected[app.Key])

		// Ensure indicator is always 1 character
		if indicator == " " {
			indicator = " "
		}

		// Format the main content with fixed widths (same as category items)
		name := truncate(app.Name, nameWidth)
		desc := truncate(app.Description, descWidth)

		// Build main content (indicator + name + description)
		mainContent := fmt.Sprintf("%s %-*s  %-*s",
			indicator,
			nameWidth, name,
			descWidth, desc)

		// Right-align source in a fixed-width column
		source := app.Source
		if len(source) > sourceWidth {
			source = truncate(source, sourceWidth)
		}

		// Format source to be right-aligned within sourceWidth
		sourceFormatted := fmt.Sprintf("%*s", sourceWidth, source)

		// Build complete line with consistent spacing
		dimmedSource := m.styles.MutedText.Render(sourceFormatted)

		// Fixed spacing between description and source
		const gapBeforeSource = 2

		line := mainContent + strings.Repeat(" ", gapBeforeSource) + dimmedSource

		// Apply highlighting if this is the current selection (same as category items)
		if appIndex == m.searchSelection {
			// Use a more subtle highlight - not bold, just slightly brighter
			highlightStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("248")) // Subtle gray-white for dim highlight

			// Keep the indicator separate so it maintains its color
			// but highlight the rest of the line
			highlightedContent := fmt.Sprintf("%-*s  %-*s",
				nameWidth, name,
				descWidth, desc)

			highlightedMain := highlightStyle.Render(highlightedContent)

			// Reconstruct the line with original indicator but highlighted content
			line = fmt.Sprintf("%s %s%s%s",
				indicator,
				highlightedMain,
				strings.Repeat(" ", gapBeforeSource),
				dimmedSource)
		}

		appLines = append(appLines, line)
	}

	// Compose the complete search view (similar to category layout)
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		styledTitle,
		"",
		lipgloss.JoinVertical(lipgloss.Left, appLines...),
	)

	// Wrap in a box similar to the details panel for consistency
	// This ensures the search results have the same visual treatment
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).                         // Same padding as details panel
		MaxWidth(categoryContentWidth + 4). // Content width + padding + borders
		Render(content)
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
