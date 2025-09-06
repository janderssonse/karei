// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements system status display UI.
package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	// Simulate system status data - in real implementation this would come from hexagonal architecture
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

	return &Status{
		styles:         styleConfig,
		systemStatus:   systemStatus,
		categories:     categories,
		recentActivity: recentActivity,
		suggestions:    suggestions,
		keyMap:         DefaultStatusKeyMap(),
	}
}

// Init initializes the status model.
func (m *Status) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the Status model.
func (m *Status) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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

		case key.Matches(msg, m.keyMap.Help):
			// Show context-sensitive help (navigate to help screen with status context)
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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

	var components []string

	// Header
	header := m.renderHeader()
	components = append(components, header)

	// Main content in columns
	content := m.renderContent()
	components = append(components, content)

	// Use lipgloss.JoinVertical for proper composition
	// Footer is handled by app.go (universal controls only)
	return lipgloss.JoinVertical(lipgloss.Left, components...)
}

// renderHeader creates the header with system health.
func (m *Status) renderHeader() string {
	var builder strings.Builder

	// Title with refresh indicator
	title := "📊 System Status Dashboard"
	if m.refreshing {
		title += " (Refreshing...)"
	}

	titleStyled := m.styles.Title.Render(title)
	builder.WriteString(titleStyled)
	builder.WriteString("\n")

	// Subtitle with last update time
	lastUpdate := m.systemStatus.LastUpdate.Format("2006-01-02 15:04:05")
	subtitle := "Last updated: " + lastUpdate
	subtitleStyled := m.styles.Subtitle.Render(subtitle)
	builder.WriteString(subtitleStyled)

	return builder.String()
}

// renderContent creates the main content area.
func (m *Status) renderContent() string {
	// Create separator first and measure its actual width
	separator := lipgloss.NewStyle().Render("  ") // 2 spaces as styled element
	separatorWidth := lipgloss.Width(separator)

	// Calculate column widths dynamically after accounting for separator
	availableWidth := m.width - separatorWidth
	leftWidth := availableWidth / 2
	rightWidth := availableWidth - leftWidth

	// Left column: System overview + categories
	leftContent := m.renderLeftColumn(leftWidth)

	// Right column: Recent activity + suggestions
	rightContent := m.renderRightColumn(rightWidth)

	// Join columns using measured separator
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftContent,
		separator,
		rightContent,
	)
}

// renderLeftColumn creates the left column content.
func (m *Status) renderLeftColumn(width int) string {
	var builder strings.Builder

	// System overview
	overview := m.renderSystemOverview(width)
	builder.WriteString(overview)
	builder.WriteString("\n\n")

	// Categories overview
	categories := m.renderCategoriesOverview(width)
	builder.WriteString(categories)

	return builder.String()
}

// renderRightColumn creates the right column content.
func (m *Status) renderRightColumn(width int) string {
	var builder strings.Builder

	// Recent activity
	activity := m.renderRecentActivity(width)
	builder.WriteString(activity)
	builder.WriteString("\n\n")

	// Suggestions
	suggestions := m.renderSuggestions(width)
	builder.WriteString(suggestions)

	return builder.String()
}

// renderSystemOverview creates the system overview card.
func (m *Status) renderSystemOverview(width int) string {
	var builder strings.Builder

	cardStyle := m.styles.Card.Width(width)

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
func (m *Status) renderCategoriesOverview(width int) string {
	var builder strings.Builder

	cardStyle := m.styles.Card.Width(width)

	var content strings.Builder
	content.WriteString(m.styles.Title.Render("Categories Overview"))
	content.WriteString("\n\n")

	// Table header
	headerLine := fmt.Sprintf("%-18s %8s %8s %12s", "Category", "Installed", "Available", "Progress")
	content.WriteString(headerLine)
	content.WriteString("\n")

	// Create separator line matching actual header width - NO hardcoded arithmetic
	separatorLine := strings.Repeat("─", lipgloss.Width(headerLine))
	content.WriteString(separatorLine)
	content.WriteString("\n")

	// Category rows
	for _, cat := range m.categories {
		progressBar := m.styles.ProgressBar(cat.Installed, cat.Available, 12)

		line := fmt.Sprintf("%-18s %8s %8s %s",
			cat.Name,
			strconv.Itoa(cat.Installed),
			strconv.Itoa(cat.Available),
			progressBar,
		)
		content.WriteString(line)
		content.WriteString("\n")
	}

	builder.WriteString(cardStyle.Render(content.String()))

	return builder.String()
}

// renderRecentActivity creates the recent activity log.
func (m *Status) renderRecentActivity(width int) string {
	var builder strings.Builder

	cardStyle := m.styles.Card.Width(width)

	var content strings.Builder
	content.WriteString(m.styles.Title.Render("Recent Activity"))
	content.WriteString("\n\n")

	for _, activity := range m.recentActivity {
		timeStr := activity.Timestamp.Format("15:04")
		statusIcon := m.styles.StatusIcon(activity.Status)

		line := fmt.Sprintf("[%s] %s %s", timeStr, statusIcon, activity.Description)
		content.WriteString(line)
		content.WriteString("\n")
	}

	builder.WriteString(cardStyle.Render(content.String()))

	return builder.String()
}

// renderSuggestions creates the suggestions panel.
func (m *Status) renderSuggestions(width int) string {
	var builder strings.Builder

	cardStyle := m.styles.Card.Width(width)

	var content strings.Builder
	content.WriteString(m.styles.Title.Render("🚀 Suggestions"))
	content.WriteString("\n\n")

	for _, suggestion := range m.suggestions {
		content.WriteString("• " + suggestion)
		content.WriteString("\n")
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

// refreshStatus simulates refreshing the status data.
func (m *Status) refreshStatus() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(_ time.Time) tea.Msg {
		// In real implementation, this would query the hexagonal architecture
		// for updated system status
		return refreshCompleteMsg{}
	})
}
