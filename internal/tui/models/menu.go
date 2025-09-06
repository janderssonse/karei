// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements the main menu navigation interface.
package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// Key constants for common key inputs.
const (
	KeyCtrlC = "ctrl+c"
	KeyEnter = "enter"
	KeyEsc   = "esc"
)

// UI constants for menu display.
const (
	SelectedPrefix = "❯ "
)

// MenuItem represents a menu option.
type MenuItem struct {
	Title       string
	Description string
	Icon        string
	Action      string
}

// Menu represents the main menu model.
type Menu struct {
	styles   *styles.Styles
	items    []MenuItem
	cursor   int
	width    int
	height   int
	quitting bool
}

// NewMenu creates a new menu model.
func NewMenu(styleConfig *styles.Styles) *Menu {
	items := []MenuItem{
		{
			Title:       "Applications",
			Description: "Browse and install development tools",
			Icon:        "📦",
			Action:      "install",
		},
		{
			Title:       "Themes",
			Description: "Apply beautiful coordinated themes",
			Icon:        "🎨",
			Action:      "theme",
		},
		{
			Title:       "Settings",
			Description: "Configure fonts, shell, and preferences",
			Icon:        "⚙️",
			Action:      "config",
		},
		{
			Title:       "Status",
			Description: "Check installation and system health",
			Icon:        "📊",
			Action:      "status",
		},
		{
			Title:       "Update",
			Description: "Keep your tools up to date",
			Icon:        "🔄",
			Action:      "update",
		},
		{
			Title:       "Documentation",
			Description: "Help and usage guides",
			Icon:        "📚",
			Action:      "help",
		},
	}

	return &Menu{
		styles: styleConfig,
		items:  items,
		cursor: 0,
	}
}

// Init initializes the menu model.
func (m *Menu) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the Menu model.
func (m *Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	}

	return m, nil
}

// View renders only the menu content since header/footer are handled by main App.
func (m *Menu) View() string {
	if m.quitting {
		return GoodbyeMessage
	}

	// Only render menu content - header and footer are handled by main App
	menu := m.renderMenu()

	// Center the menu content if width is available
	if m.width > 0 {
		menu = m.centerContent(menu)
	}

	return menu
}

// GetSelectedAction returns the action identifier of the selected menu item.
func (m *Menu) GetSelectedAction() string {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return m.items[m.cursor].Action
	}

	return ""
}

// handleKeyMsg processes keyboard input for the menu.
//

func (m *Menu) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case KeyCtrlC, "q", KeyEsc:
		m.quitting = true

		return m, tea.Quit
	case "up", "k":
		return m.handleCursorMovement(-1)
	case "down", "j":
		return m.handleCursorMovement(1)
	case KeyEnter, " ":
		return m.handleMenuSelection()
	}

	return m, nil
}

// handleCursorMovement moves the cursor up or down.
//

func (m *Menu) handleCursorMovement(direction int) (tea.Model, tea.Cmd) {
	newCursor := m.cursor + direction
	if newCursor >= 0 && newCursor < len(m.items) {
		m.cursor = newCursor
	}

	return m, nil
}

// handleMenuSelection processes menu item selection.
//

func (m *Menu) handleMenuSelection() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.items) {
		return m, nil
	}

	selectedItem := m.items[m.cursor]

	return m, m.createNavigationCmd(selectedItem.Action)
}

// createNavigationCmd creates a command for navigation based on action.
func (m *Menu) createNavigationCmd(action string) tea.Cmd {
	var screen int

	switch action {
	case OperationInstall:
		screen = AppsScreen
	case "theme":
		screen = ThemeScreen
	case "config":
		screen = ConfigScreen
	case "status":
		screen = StatusScreen
	case "help":
		screen = HelpScreen
	case "update":
		// Handle update action - navigate to progress screen for updates
		return nil
	default:
		return nil
	}

	return func() tea.Msg {
		return NavigateMsg{Screen: screen}
	}
}

// handleWindowSizeMsg processes window resize messages.
//

func (m *Menu) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	return m, nil
}

// renderMenu creates the menu items list.
func (m *Menu) renderMenu() string {
	var builder strings.Builder

	// Menu container with a more engaging prompt
	builder.WriteString(m.styles.Title.Render("Build your perfect development foundation"))
	builder.WriteString("\n\n")

	for itemIndex, item := range m.items {
		var (
			style  lipgloss.Style
			prefix string
		)

		if itemIndex == m.cursor {
			style = m.styles.Selected
			prefix = SelectedPrefix
		} else {
			style = m.styles.Unselected
			prefix = "  "
		}

		line := fmt.Sprintf("%s%s %s", prefix, item.Icon, item.Title)
		builder.WriteString(style.Render(line))
		builder.WriteString("\n")

		// Show description for selected item
		if itemIndex == m.cursor {
			descStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)
			desc := descStyle.Render("    " + item.Description)
			builder.WriteString(desc)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// centerContent centers the content horizontally using lipgloss methods.
func (m *Menu) centerContent(content string) string {
	if m.width <= 0 {
		return content
	}

	// Use lipgloss to handle centering instead of manual arithmetic
	return lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(content)
}
