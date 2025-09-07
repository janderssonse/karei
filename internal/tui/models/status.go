// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements system status display UI.
package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/application"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// SystemStatus represents the overall system status.
type SystemStatus struct {
	Health             string // "excellent", "good", "warning", "error"
	InstalledApps      int
	AvailableApps      int
	CurrentTheme       string
	LastUpdate         time.Time
	DiskSpaceUsed      string
	DiskSpaceAvailable string
	SystemUptime       time.Duration
}

// CategoryStatus represents status for an application category.
type CategoryStatus struct {
	Name       string
	Installed  int
	Available  int
	Completion float64
	Status     string // "complete", "partial", "empty"
}

// ActivityEntry represents a recent activity log entry.
type ActivityEntry struct {
	Timestamp   time.Time
	Action      string
	Target      string
	Status      string // "success", "warning", "error"
	Description string
}

// Status represents the system status screen model.
type Status struct {
	styles         *styles.Styles
	width          int
	height         int
	systemStatus   SystemStatus
	categories     []CategoryStatus
	recentActivity []ActivityEntry
	suggestions    []string
	refreshing     bool
	quitting       bool
	keyMap         StatusKeyMap
	helpModal      *HelpModal
	statusService  *application.StatusService // Optional real data service
}

// StatusKeyMap defines key bindings for the status screen.
type StatusKeyMap struct {
	Refresh key.Binding
	Back    key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// DefaultStatusKeyMap returns the default key bindings.
func DefaultStatusKeyMap() StatusKeyMap {
	return StatusKeyMap{
		Refresh: key.NewBinding(
			key.WithKeys("r", "f5"),
			key.WithHelp("r/F5", "refresh status"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back to menu"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// NewStatus creates a new status model.
func NewStatus(styleConfig *styles.Styles) *Status {
	return NewStatusWithService(styleConfig, nil)
}

// NewStatusWithService creates a status model with optional real data service.
func NewStatusWithService(styleConfig *styles.Styles, service *application.StatusService) *Status {
	// Default mock data - will be replaced by real data if service is provided
	systemStatus := SystemStatus{
		Health:             "excellent",
		InstalledApps:      23,
		AvailableApps:      45,
		CurrentTheme:       "🌃 Tokyo Night (Dark)",
		LastUpdate:         time.Now().Add(-2 * time.Hour),
		DiskSpaceUsed:      "2.1 GB",
		DiskSpaceAvailable: "125.4 GB",
		SystemUptime:       24*time.Hour + 15*time.Minute,
	}

	// Sample category data
	categories := []CategoryStatus{
		{
			Name:       "Development Tools",
			Installed:  8,
			Available:  12,
			Completion: 0.67,
			Status:     "partial",
		},
		{
			Name:       "System Utilities",
			Installed:  6,
			Available:  8,
			Completion: 0.75,
			Status:     "partial",
		},
		{
			Name:       "Media & Graphics",
			Installed:  3,
			Available:  10,
			Completion: 0.30,
			Status:     "partial",
		},
		{
			Name:       "Productivity",
			Installed:  4,
			Available:  8,
			Completion: 0.50,
			Status:     "partial",
		},
		{
			Name:       "Security Tools",
			Installed:  2,
			Available:  7,
			Completion: 0.29,
			Status:     "partial",
		},
	}

	// Sample recent activity
	recentActivity := []ActivityEntry{
		{
			Timestamp:   time.Now().Add(-15 * time.Minute),
			Action:      "install",
			Target:      "Neovim",
			Status:      "success",
			Description: "Neovim installation completed successfully",
		},
		{
			Timestamp:   time.Now().Add(-32 * time.Minute),
			Action:      "install",
			Target:      "Visual Studio Code",
			Status:      "success",
			Description: "Visual Studio Code installation completed",
		},
		{
			Timestamp:   time.Now().Add(-1 * time.Hour),
			Action:      "theme",
			Target:      "Tokyo Night",
			Status:      "success",
			Description: "Applied Tokyo Night theme successfully",
		},
		{
			Timestamp:   time.Now().Add(-2 * time.Hour),
			Action:      "update",
			Target:      "System packages",
			Status:      "success",
			Description: "Updated 142 system packages",
		},
		{
			Timestamp:   time.Now().Add(-1 * 24 * time.Hour),
			Action:      "install",
			Target:      "Docker Desktop",
			Status:      "success",
			Description: "Docker Desktop installation completed",
		},
	}

	// Sample suggestions
	suggestions := []string{
		"Install GIMP for advanced image editing",
		"Consider adding Spotify for music streaming",
		"Update available for 3 applications",
		"Security audit recommended (run 'karei security')",
		"Backup system configuration with Timeshift",
	}

	// Create help modal
	helpModal := NewHelpModal()
	helpModal.SetScreen("status")

	return &Status{
		styles:         styleConfig,
		systemStatus:   systemStatus,
		categories:     categories,
		recentActivity: recentActivity,
		suggestions:    suggestions,
		keyMap:         DefaultStatusKeyMap(),
		helpModal:      helpModal,
		statusService:  service,
	}
}

// Init initializes the status model.
func (m *Status) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the Status model.
//
//nolint:cyclop // Complex but necessary for handling various UI interactions
func (m *Status) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle help modal toggle first
		if key.Matches(msg, m.keyMap.Help) {
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

		switch {
		case key.Matches(msg, m.keyMap.Quit):
			m.quitting = true

			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Back):
			// Navigate back to menu
			return m, func() tea.Msg {
				return NavigateMsg{Screen: MenuScreen}
			}

		case key.Matches(msg, m.keyMap.Refresh):
			// Refresh status data
			m.refreshing = true

			return m, m.refreshStatus()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update help modal size
		if m.helpModal != nil {
			m.helpModal.SetSize(msg.Width, msg.Height)
		}

	case refreshCompleteMsg:
		m.refreshing = false
	}

	return m, nil
}

// refreshCompleteMsg is sent when status refresh is complete.
type refreshCompleteMsg struct{}

// View renders the status screen.
func (m *Status) View() string {
	if m.quitting {
		return GoodbyeMessage
	}

	// If help modal is visible, show it as an overlay
	if m.helpModal != nil && m.helpModal.IsVisible() {
		modalView := m.helpModal.View()

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			modalView,
		)
	}

	return m.renderBaseView()
}

// renderBaseView renders the main status view without overlays.
func (m *Status) renderBaseView() string {
	var components []string

	// Clean header
	header := m.renderCleanHeader()
	components = append(components, header)

	// Calculate available height for content
	headerHeight := 3 // Header with border
	footerHeight := 3 // Footer with border
	contentHeight := m.height - headerHeight - footerHeight

	// Main content in columns with proper height
	content := m.renderContent()

	// Wrap content to fill available space
	contentStyled := lipgloss.NewStyle().
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(content)

	components = append(components, contentStyled)

	// Clean footer
	footer := m.renderCleanFooter()
	components = append(components, footer)

	// Use lipgloss.JoinVertical for proper composition
	return lipgloss.JoinVertical(lipgloss.Top, components...)
}

// renderCleanHeader renders the new simplified header format.
func (m *Status) renderCleanHeader() string {
	// Left side: App name » Current location
	location := "Karei » Status"
	leftSide := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary).
		Render(location)

	// Right side: Status (health or refreshing)
	var status string
	if m.refreshing {
		status = "Refreshing..."
	}

	rightSide := lipgloss.NewStyle().
		Foreground(m.styles.Muted).
		Render(status)

	// Calculate spacing
	totalWidth := m.width
	leftWidth := lipgloss.Width(leftSide)
	rightWidth := lipgloss.Width(rightSide)
	spacerWidth := totalWidth - leftWidth - rightWidth - 4

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

// renderCleanFooter renders the new simplified footer with context-aware actions.
func (m *Status) renderCleanFooter() string {
	// Context-aware footer actions with styled keys and descriptions
	var actions []string

	// Styles for different parts (matching apps page)
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary) // Keys in primary color (blue)

	bracketStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary) // Brackets also in primary color

	actionStyle := lipgloss.NewStyle().
		Foreground(m.styles.Muted) // Actions in muted color

	// Helper function to format action (same as apps page)
	formatAction := func(key, action string) string {
		return bracketStyle.Render("[") +
			keyStyle.Render(key) +
			bracketStyle.Render("]") +
			" " +
			actionStyle.Render(action)
	}

	// Status page actions
	actions = []string{
		formatAction("R", "Refresh"),
		formatAction("C", "Clear"),
		formatAction("Esc", "Back"),
	}

	// Always add help with special styling (dim yellow to stand out)
	helpKey := bracketStyle.Render("[") +
		lipgloss.NewStyle().Bold(true).Foreground(m.styles.Warning).Render("?") +
		bracketStyle.Render("]")
	actions = append(actions, helpKey+" "+actionStyle.Render("Help"))

	// Join actions with more spacing
	footerText := strings.Join(actions, "   ")

	// Style the footer container (exactly matching apps page)
	return lipgloss.NewStyle().
		Padding(0, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width).
		Render(footerText)
}

// Commented out - replaced by renderCleanHeader for new simplified design
// renderHeader creates the header with system health.
// func (m *Status) renderHeader() string {
// 	var builder strings.Builder

// 	// Title with refresh indicator
// 	title := "📊 System Status Dashboard"
// 	if m.refreshing {
// 		title += " (Refreshing...)"
// 	}

// 	titleStyled := m.styles.Title.Render(title)
// 	builder.WriteString(titleStyled)
// 	builder.WriteString("\n")

// 	// Subtitle with last update time
// 	lastUpdate := m.systemStatus.LastUpdate.Format("2006-01-02 15:04:05")
// 	subtitle := "Last updated: " + lastUpdate
// 	subtitleStyled := m.styles.Subtitle.Render(subtitle)
// 	builder.WriteString(subtitleStyled)

// 	return builder.String()
// }

// renderContent creates the main content area.
func (m *Status) renderContent() string {
	// Calculate column widths - give more room by removing separator
	// Account for borders (2 chars per box) and minimal gap
	availableWidth := m.width - 4 // Account for box borders
	leftWidth := availableWidth / 2
	rightWidth := availableWidth / 2

	// Calculate equal box heights
	// Total content height minus spacing between rows
	headerHeight := 3 // Header with border
	footerHeight := 3 // Footer with border
	contentHeight := m.height - headerHeight - footerHeight

	// Make boxes smaller and account for borders properly
	// Each box gets less than half to leave breathing room
	// We want the content + borders to fit, so we calculate content height
	boxContentHeight := (contentHeight - 2) / 3 // Smaller boxes, more spacing
	// Add 2 for the borders (top + bottom) that Lipgloss will add
	boxHeight := boxContentHeight

	// Left column: System overview + categories
	leftContent := m.renderLeftColumn(leftWidth, boxHeight)

	// Right column: Recent activity + suggestions
	rightContent := m.renderRightColumn(rightWidth, boxHeight)

	// Join columns directly without separator
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftContent,
		rightContent,
	)
}

// renderLeftColumn creates the left column content.
func (m *Status) renderLeftColumn(width int, boxHeight int) string {
	var builder strings.Builder

	// System overview
	overview := m.renderSystemOverview(width, boxHeight)
	builder.WriteString(overview)
	builder.WriteString("\n") // Reduced from \n\n to single newline

	// Categories overview
	categories := m.renderCategoriesOverview(width, boxHeight)
	builder.WriteString(categories)

	return builder.String()
}

// renderRightColumn creates the right column content.
func (m *Status) renderRightColumn(width int, boxHeight int) string {
	var builder strings.Builder

	// Recent activity
	activity := m.renderRecentActivity(width, boxHeight)
	builder.WriteString(activity)
	builder.WriteString("\n") // Reduced from \n\n to single newline

	// Suggestions
	suggestions := m.renderSuggestions(width, boxHeight)
	builder.WriteString(suggestions)

	return builder.String()
}

// renderSystemOverview creates the system overview card.
func (m *Status) renderSystemOverview(width int, boxHeight int) string {
	var builder strings.Builder

	// Use tighter card style for dashboard with fixed height
	// Height is for content only - borders are added on top
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Muted).
		Padding(0, 1). // Reduced padding from (1, 2)
		Width(width).
		Height(boxHeight) // This is content height - borders are extra

	var content strings.Builder
	content.WriteString(m.styles.Title.Render("Installation Summary"))
	content.WriteString("\n\n")

	// Key metrics
	metrics := []struct {
		label string
		value string
		icon  string
	}{
		{"Applications Installed", fmt.Sprintf("%d / %d available", m.systemStatus.InstalledApps, m.systemStatus.AvailableApps), "📦"},
		{"Current Theme", m.systemStatus.CurrentTheme, "🎨"},
		{"System Health", m.getHealthDisplay(m.systemStatus.Health), "🏥"},
		{"Disk Space Used", fmt.Sprintf("%s (available: %s)", m.systemStatus.DiskSpaceUsed, m.systemStatus.DiskSpaceAvailable), "💾"},
		{"System Uptime", m.formatDuration(m.systemStatus.SystemUptime), "⏰"},
	}

	for _, metric := range metrics {
		line := fmt.Sprintf("%s %s: %s", metric.icon, metric.label, metric.value)
		content.WriteString(line)
		content.WriteString("\n")
	}

	builder.WriteString(cardStyle.Render(content.String()))

	return builder.String()
}

// renderCategoriesOverview creates the categories overview.
func (m *Status) renderCategoriesOverview(width int, boxHeight int) string {
	var builder strings.Builder

	// Use tighter card style for dashboard with fixed height
	// Height is for content only - borders are added on top
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Muted).
		Padding(0, 1). // Reduced padding from (1, 2)
		Width(width).
		Height(boxHeight) // This is content height - borders are extra

	var content strings.Builder
	content.WriteString(m.styles.Title.Render("Categories"))
	content.WriteString("\n\n")

	// Show category statistics
	categoryStats := []struct {
		name  string
		count int
		icon  string
	}{
		{"Development", 12, "💻"},
		{"Utilities", 8, "🔧"},
		{"System", 6, "⚙️"},
		{"Graphics", 4, "🎨"},
		{"Communication", 3, "💬"},
	}

	for _, cat := range categoryStats {
		line := fmt.Sprintf("%s %s: %d apps", cat.icon, cat.name, cat.count)
		content.WriteString(line)
		content.WriteString("\n")
	}

	builder.WriteString(cardStyle.Render(content.String()))

	return builder.String()
}

// renderRecentActivity creates the recent activity panel.
func (m *Status) renderRecentActivity(width int, boxHeight int) string {
	var builder strings.Builder

	// Use tighter card style for dashboard with fixed height
	// Height is for content only - borders are added on top
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Muted).
		Padding(0, 1). // Reduced padding
		Width(width).
		Height(boxHeight) // This is content height - borders are extra

	var content strings.Builder
	content.WriteString(m.styles.Title.Render("Recent Activity"))
	content.WriteString("\n\n")

	// Show max activities that fit in the box
	maxActivities := boxHeight - 2 // Account for title and spacing

	activityCount := len(m.recentActivity)
	if activityCount > maxActivities && maxActivities > 0 {
		activityCount = maxActivities
	}

	for i := range activityCount {
		activity := m.recentActivity[i]
		timeStr := activity.Timestamp.Format("15:04")
		statusIcon := m.styles.StatusIcon(activity.Status)

		line := fmt.Sprintf("[%s] %s %s", timeStr, statusIcon, activity.Description)
		content.WriteString(line)

		if i < activityCount-1 {
			content.WriteString("\n")
		}
	}

	builder.WriteString(cardStyle.Render(content.String()))

	return builder.String()
}

// renderSuggestions creates the suggestions panel.
func (m *Status) renderSuggestions(width int, boxHeight int) string {
	var builder strings.Builder

	// Use tighter card style for dashboard with fixed height
	// Height is for content only - borders are added on top
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Muted).
		Padding(0, 1). // Reduced padding from (1, 2)
		Width(width).
		Height(boxHeight) // This is content height - borders are extra

	var content strings.Builder
	content.WriteString(m.styles.Title.Render("🚀 Suggestions"))
	content.WriteString("\n\n")

	// Show max suggestions that fit in the box
	maxSuggestions := boxHeight - 2 // Account for title and spacing

	suggestionCount := len(m.suggestions)
	if suggestionCount > maxSuggestions && maxSuggestions > 0 {
		suggestionCount = maxSuggestions
	}

	for i := range suggestionCount {
		line := "• " + m.suggestions[i]
		content.WriteString(line)

		if i < suggestionCount-1 {
			content.WriteString("\n")
		}
	}

	builder.WriteString(cardStyle.Render(content.String()))

	return builder.String()
}

// GetNavigationHints returns screen-specific navigation hints for the footer.
func (m *Status) GetNavigationHints() []string {
	return []string{
		"[r/F5] Refresh",
		"[e] Export",
		"[c] Clear Activity",
	}
}

// renderNavigationHeader renders the screen-specific navigation hints with styled tabs.

// renderFooter creates the footer with universal keybindings only.
// NOTE: This is kept for compatibility but not used - app.go handles the universal footer.
func (m *Status) getHealthDisplay(health string) string {
	switch health {
	case "excellent":
		return "🟢 Excellent"
	case "good":
		return "🟡 Good"
	case "warning":
		return "🟠 Warning"
	case "error":
		return "🔴 Error"
	default:
		return "❓ Unknown"
	}
}

// formatDuration formats a duration in a human-readable way.
func (m *Status) formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}

// refreshStatus fetches real or simulated status data.
func (m *Status) refreshStatus() tea.Cmd {
	return func() tea.Msg {
		// Use real service if available
		if m.statusService != nil {
			// Fetch real data (context would come from app in production)
			ctx := context.Background()
			if data, err := m.statusService.GetSystemStatus(ctx); err == nil {
				// Update system status with real data
				m.systemStatus.InstalledApps = data.InstalledApps
				m.systemStatus.AvailableApps = data.AvailableApps
				m.systemStatus.CurrentTheme = data.CurrentTheme
				m.systemStatus.DiskSpaceUsed = application.FormatDiskSpace(data.DiskUsageGB)
				m.systemStatus.DiskSpaceAvailable = application.FormatDiskSpace(data.DiskAvailGB)
				m.systemStatus.SystemUptime = application.FormatUptime(data.UptimeHours)
				m.systemStatus.LastUpdate = time.Now()
			}
		}

		// Simulate delay for visual feedback
		time.Sleep(500 * time.Millisecond)

		return refreshCompleteMsg{}
	}
}
