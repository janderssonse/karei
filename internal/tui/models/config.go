// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements application configuration UI.
package models

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// ConfigSection represents a configuration section.
type ConfigSection struct {
	Title  string
	Fields []ConfigField
}

// ConfigField represents a single configuration field.
type ConfigField struct {
	Name        string
	Label       string
	Type        string // "select", "input", "toggle"
	Value       any
	Options     []ConfigOption
	Description string
}

// ConfigOption represents an option for select fields.
type ConfigOption struct {
	Label string
	Value string
}

// Config represents the configuration screen model.
type Config struct {
	styles     *styles.Styles
	width      int
	height     int
	sections   []ConfigSection
	form       *huh.Form
	quitting   bool
	currentTab int
	tabs       []string
	helpModal  *HelpModal
}

// NewConfig creates a new configuration model.
func NewConfig(styleConfig *styles.Styles) *Config {
	// Define configuration sections
	sections := []ConfigSection{
		{
			Title: "Shell Environment",
			Fields: []ConfigField{
				{
					Name:        "shell",
					Label:       "Default Shell",
					Type:        "select",
					Value:       "fish",
					Description: "Your default shell environment",
					Options: []ConfigOption{
						{Label: "Fish 🐟", Value: "fish"},
						{Label: "Zsh", Value: "zsh"},
						{Label: "Bash", Value: "bash"},
					},
				},
				{
					Name:        "terminal",
					Label:       "Terminal Emulator",
					Type:        "select",
					Value:       "ghostty",
					Description: "Your preferred terminal application",
					Options: []ConfigOption{
						{Label: "Ghostty", Value: "ghostty"},
						{Label: "Alacritty", Value: "alacritty"},
						{Label: "GNOME Terminal", Value: "gnome-terminal"},
					},
				},
				{
					Name:        "font",
					Label:       "Terminal Font",
					Type:        "select",
					Value:       "jetbrains-mono",
					Description: "Font for terminal and code editors",
					Options: []ConfigOption{
						{Label: "JetBrains Mono", Value: "jetbrains-mono"},
						{Label: "Fira Code", Value: "fira-code"},
						{Label: "Cascadia Code", Value: "cascadia-code"},
					},
				},
				{
					Name:        "font_size",
					Label:       "Font Size",
					Type:        "input",
					Value:       "14",
					Description: "Font size in points",
				},
			},
		},
		{
			Title: "Development Tools",
			Fields: []ConfigField{
				{
					Name:        "editor",
					Label:       "Primary Editor",
					Type:        "select",
					Value:       "vscode",
					Description: "Your main code editor",
					Options: []ConfigOption{
						{Label: "VS Code", Value: "vscode"},
						{Label: "Neovim", Value: "neovim"},
						{Label: "Emacs", Value: "emacs"},
					},
				},
				{
					Name:        "git_username",
					Label:       "Git Username",
					Type:        "input",
					Value:       "",
					Description: "Your Git commit username",
				},
				{
					Name:        "git_email",
					Label:       "Git Email",
					Type:        "input",
					Value:       "",
					Description: "Your Git commit email address",
				},
				{
					Name:        "ssh_key",
					Label:       "SSH Key Generation",
					Type:        "toggle",
					Value:       true,
					Description: "Generate SSH key for Git authentication",
				},
			},
		},
		{
			Title: "System Preferences",
			Fields: []ConfigField{
				{
					Name:        "theme",
					Label:       "System Theme",
					Type:        "select",
					Value:       "tokyo-night",
					Description: "Coordinated system-wide theme",
					Options: []ConfigOption{
						{Label: "🌃 Tokyo Night", Value: "tokyo-night"},
						{Label: "🐱 Catppuccin", Value: "catppuccin"},
						{Label: "🧊 Nord", Value: "nord"},
						{Label: "🌲 Everforest", Value: "everforest"},
						{Label: "🟤 Gruvbox", Value: "gruvbox"},
					},
				},
				{
					Name:        "wallpaper",
					Label:       "Wallpaper Style",
					Type:        "select",
					Value:       "theme-default",
					Description: "Desktop wallpaper preference",
					Options: []ConfigOption{
						{Label: "Theme Default", Value: "theme-default"},
						{Label: "Custom Image", Value: "custom"},
						{Label: "Solid Color", Value: "solid"},
					},
				},
				{
					Name:        "icon_theme",
					Label:       "Icon Theme",
					Type:        "select",
					Value:       "papirus-dark",
					Description: "Desktop icon theme",
					Options: []ConfigOption{
						{Label: "Papirus Dark", Value: "papirus-dark"},
						{Label: "Adwaita", Value: "adwaita"},
						{Label: "Breeze", Value: "breeze"},
					},
				},
			},
		},
		{
			Title: "Privacy & Security",
			Fields: []ConfigField{
				{
					Name:        "telemetry",
					Label:       "Telemetry Collection",
					Type:        "select",
					Value:       "disabled",
					Description: "Data collection preferences",
					Options: []ConfigOption{
						{Label: "Disabled", Value: "disabled"},
						{Label: "Minimal", Value: "minimal"},
						{Label: "Full", Value: "full"},
					},
				},
				{
					Name:        "firewall",
					Label:       "Enable Firewall",
					Type:        "toggle",
					Value:       true,
					Description: "Enable UFW firewall protection",
				},
				{
					Name:        "auto_updates",
					Label:       "Automatic Updates",
					Type:        "select",
					Value:       "security",
					Description: "Automatic system updates",
					Options: []ConfigOption{
						{Label: "Security Only", Value: "security"},
						{Label: "All Updates", Value: "all"},
						{Label: "Disabled", Value: "disabled"},
					},
				},
			},
		},
	}

	// Create tabs
	tabs := make([]string, len(sections))
	for i, section := range sections {
		tabs[i] = section.Title
	}

	// Create help modal
	helpModal := NewHelpModal()
	helpModal.SetScreen("config")

	return &Config{
		styles:     styleConfig,
		sections:   sections,
		tabs:       tabs,
		currentTab: 0,
		helpModal:  helpModal,
	}
}

// Init initializes the config model.
func (m *Config) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the Config model.
func (m *Config) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	}

	return m.handleFormUpdate(msg)
}

// handleKeyMsg processes keyboard input messages.

// GetNavigationHints returns screen-specific navigation hints for the footer.
func (m *Config) GetNavigationHints() []string {
	return []string{
		"[←/→] Switch Tabs",
		"[Enter] Edit Section",
		"[s] Save Config",
	}
}

// View renders the config screen.
func (m *Config) View() string {
	if m.quitting {
		return "Configuration saved! Goodbye!\n"
	}

	// If help modal is visible, show it as an overlay
	if m.helpModal != nil && m.helpModal.IsVisible() {
		return m.renderWithModal()
	}

	return m.renderBaseView()
}

// renderWithModal renders the view with modal overlay.
func (m *Config) renderWithModal() string {
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

// renderBaseView renders the main config view without overlays.
func (m *Config) renderBaseView() string {
	// If form is active, show form
	if m.form != nil {
		return m.form.View()
	}

	// Build the complete view: header + tabs + content + footer
	components := []string{}

	// Add the clean header
	header := m.renderCleanHeader()
	components = append(components, header)

	// Add tabs
	tabs := m.renderTabs()
	if tabs != "" {
		components = append(components, tabs)
	}

	// Add current section content
	content := m.renderSection(m.sections[m.currentTab])
	components = append(components, content)

	// Add the clean footer
	footer := m.renderCleanFooter()
	components = append(components, footer)

	// Compose with Lipgloss
	if len(components) == 1 {
		return components[0]
	}

	return lipgloss.JoinVertical(lipgloss.Top, components...)
}

// GetConfiguration returns all configuration field values as a map.
func (m *Config) GetConfiguration() map[string]any {
	config := make(map[string]any)

	for _, section := range m.sections {
		for _, field := range section.Fields {
			config[field.Name] = field.Value
		}
	}

	return config
}

// renderCleanHeader renders the new simplified header format.
func (m *Config) renderCleanHeader() string {
	// Left side: App name » Current location
	location := "Karei » Configuration"
	leftSide := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary).
		Render(location)

	// Right side: Status (unsaved changes indicator)
	status := ""
	// In a real implementation, track if there are unsaved changes
	// For now, we'll leave it empty

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
func (m *Config) renderCleanFooter() string {
	actions := []FooterAction{
		{Key: "Tab", Action: "Section"},
		{Key: "Enter", Action: "Edit"},
		{Key: "S", Action: "Save"},
		{Key: "Esc", Action: "Back"},
	}

	return RenderFooter(m.styles, m.width, actions, true)
}

// renderTabs creates the tab navigation.
func (m *Config) renderTabs() string {
	tabs := make([]string, 0, len(m.tabs))

	for i, tab := range m.tabs {
		var style lipgloss.Style
		if i == m.currentTab {
			// Active tab - bold and highlighted
			style = lipgloss.NewStyle().
				Bold(true).
				Foreground(m.styles.Primary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(m.styles.Primary).
				Padding(0, 2).
				MarginRight(1)
		} else {
			// Inactive tab - muted
			style = lipgloss.NewStyle().
				Foreground(m.styles.Muted).
				Padding(0, 2).
				MarginRight(1)
		}

		tabs = append(tabs, style.Render(tab))
	}

	// Add left padding to align with content
	tabLine := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	return lipgloss.NewStyle().
		PaddingLeft(2).
		MarginTop(1).
		Render(tabLine)
}

// renderSection renders a configuration section.
func (m *Config) renderSection(section ConfigSection) string {
	// Calculate available height for content area
	headerHeight := 3 // Header with border
	footerHeight := 3 // Footer with border
	tabsHeight := 3   // Tabs area
	availableHeight := m.height - headerHeight - footerHeight - tabsHeight

	// Build the content
	var content strings.Builder

	// Section title with some padding
	titleStyle := m.styles.Title.
		MarginTop(1).
		MarginBottom(1).
		PaddingLeft(2)
	content.WriteString(titleStyle.Render(section.Title))
	content.WriteString("\n")

	// Render fields with padding
	for _, field := range section.Fields {
		fieldView := m.renderField(field)
		// Add left padding to fields
		lines := strings.Split(fieldView, "\n")
		for _, line := range lines {
			if line != "" {
				content.WriteString("  " + line + "\n")
			}
		}
	}

	// Count actual content lines
	contentStr := content.String()
	contentLines := strings.Count(contentStr, "\n")

	// Add empty lines to fill the available space
	if availableHeight > contentLines {
		for range availableHeight - contentLines - 1 {
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderField renders a single configuration field.
func (m *Config) renderField(field ConfigField) string {
	var builder strings.Builder

	// Field label
	labelStyle := m.styles.Title.
		Bold(false).
		MarginBottom(0)
	builder.WriteString(labelStyle.Render(field.Label))
	builder.WriteString("\n")

	// Field value/control
	var valueDisplay string

	switch field.Type {
	case "select":
		// Find current option
		currentLabel := fmt.Sprintf("%v", field.Value)
		for _, opt := range field.Options {
			if opt.Value == fmt.Sprintf("%v", field.Value) {
				currentLabel = opt.Label

				break
			}
		}

		valueDisplay = "❯ " + currentLabel

	case "input":
		valueDisplay = fmt.Sprintf("[ %v ]", field.Value)

	case "toggle":
		if field.Value == true {
			valueDisplay = "✅ Enabled"
		} else {
			valueDisplay = "❌ Disabled"
		}
	}

	valueStyle := lipgloss.NewStyle().Foreground(m.styles.Primary).Render(valueDisplay)
	builder.WriteString("  " + valueStyle)
	builder.WriteString("\n")

	// Field description
	if field.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(m.styles.Muted).
			Faint(true)
		builder.WriteString("  " + descStyle.Render(field.Description))
		builder.WriteString("\n")
	}

	return builder.String()
}

// renderFooter creates the footer with keybindings.

// startForm creates and starts a form for the current section.
func (m *Config) startForm() tea.Cmd {
	section := m.sections[m.currentTab]

	var formGroups []huh.Field

	for _, field := range section.Fields {
		switch field.Type {
		case "select":
			options := make([]huh.Option[string], len(field.Options))
			for i, opt := range field.Options {
				options[i] = huh.NewOption(opt.Label, opt.Value)
			}

			if strPtr, ok := field.Value.(*string); ok {
				selectField := huh.NewSelect[string]().
					Title(field.Label).
					Description(field.Description).
					Options(options...).
					Value(strPtr)
				formGroups = append(formGroups, selectField)
			}

		case "input":
			if strPtr, ok := field.Value.(*string); ok {
				inputField := huh.NewInput().
					Title(field.Label).
					Description(field.Description).
					Value(strPtr)
				formGroups = append(formGroups, inputField)
			}

		case "toggle":
			if boolPtr, ok := field.Value.(*bool); ok {
				confirmField := huh.NewConfirm().
					Title(field.Label).
					Description(field.Description).
					Value(boolPtr)
				formGroups = append(formGroups, confirmField)
			}
		}
	}

	m.form = huh.NewForm(huh.NewGroup(formGroups...)).
		WithTheme(huh.ThemeCharm())

	return m.form.Init()
}

// saveConfig saves the current configuration.
func (m *Config) saveConfig() tea.Cmd {
	// Implement actual config saving using hexagonal architecture (requires config service port)
	return nil
}

// handleKeyMsg processes keyboard input messages.
//

//nolint:cyclop // Complex but necessary for handling various UI interactions
func (m *Config) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	switch msg.String() {
	case KeyCtrlC, "q", KeyEsc:
		return m.handleQuitKeys(msg)
	case "left", "h":
		return m.handleTabNavigation(-1)
	case "right", "l":
		return m.handleTabNavigation(1)
	case KeyEnter, " ":
		return m, m.startForm()
	case "s":
		return m, m.saveConfig()
	}

	return m, nil
}

// handleQuitKeys processes quit-related key presses.
//

func (m *Config) handleQuitKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == KeyEsc && m.form != nil {
		// Exit form mode
		m.form = nil

		return m, nil
	}

	m.quitting = true

	return m, tea.Quit
}

// handleTabNavigation changes the current tab.
//

func (m *Config) handleTabNavigation(direction int) (tea.Model, tea.Cmd) {
	newTab := m.currentTab + direction
	if newTab >= 0 && newTab < len(m.tabs) {
		m.currentTab = newTab
	}

	return m, nil
}

// handleWindowSizeMsg processes window resize messages.
//

func (m *Config) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Update help modal size
	if m.helpModal != nil {
		m.helpModal.SetSize(msg.Width, msg.Height)
	}

	return m, nil
}

// handleFormUpdate processes form-related updates.
//

func (m *Config) handleFormUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
			if m.form.State == huh.StateCompleted {
				// Form completed - save and exit form mode
				m.form = nil

				return m, tea.Batch(cmd, m.saveConfig())
			}
		}

		return m, cmd
	}

	return m, nil
}
