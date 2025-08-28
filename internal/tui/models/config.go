// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models provides configuration screen for the TUI interface.
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

	return &Config{
		styles:     styleConfig,
		sections:   sections,
		tabs:       tabs,
		currentTab: 0,
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

// View renders the config screen.
func (m *Config) View() string {
	if m.quitting {
		return "Configuration saved! Goodbye!\n"
	}

	// If form is active, show form
	if m.form != nil {
		return m.form.View()
	}

	var builder strings.Builder

	// Header
	header := m.renderHeader()
	builder.WriteString(header)
	builder.WriteString("\n\n")

	// Tabs
	tabs := m.renderTabs()
	builder.WriteString(tabs)
	builder.WriteString("\n\n")

	// Current section content
	content := m.renderSection(m.sections[m.currentTab])
	builder.WriteString(content)
	builder.WriteString("\n\n")

	// Footer
	footer := m.renderFooter()
	builder.WriteString(footer)

	return builder.String()
}

// GetConfiguration returns the current configuration values.
func (m *Config) GetConfiguration() map[string]any {
	config := make(map[string]any)

	for _, section := range m.sections {
		for _, field := range section.Fields {
			config[field.Name] = field.Value
		}
	}

	return config
}

// renderHeader creates the header.
func (m *Config) renderHeader() string {
	title := m.styles.Title.Render("⚙️ System Configuration")
	subtitle := m.styles.Subtitle.Render("Configure your development environment preferences")

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle)
}

// renderTabs creates the tab navigation.
func (m *Config) renderTabs() string {
	tabs := make([]string, 0, len(m.tabs))

	for i, tab := range m.tabs {
		var style lipgloss.Style
		if i == m.currentTab {
			style = m.styles.Selected.
				Padding(0, 2).
				MarginRight(1)
		} else {
			style = m.styles.Unselected.
				Padding(0, 2).
				MarginRight(1).
				Faint(true)
		}

		tabs = append(tabs, style.Render(tab))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// renderSection renders a configuration section.
func (m *Config) renderSection(section ConfigSection) string {
	var builder strings.Builder

	// Calculate available space dynamically
	header := m.renderHeader()
	footer := m.renderFooter()
	tabs := m.renderTabs()

	// Section container with dynamic sizing
	availableWidth := m.width
	availableHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - lipgloss.Height(tabs)

	sectionStyle := m.styles.Card.
		Width(availableWidth).
		Height(availableHeight)

	var content strings.Builder
	content.WriteString(m.styles.Title.Render(section.Title))
	content.WriteString("\n\n")

	// Render fields
	for _, field := range section.Fields {
		fieldView := m.renderField(field)
		content.WriteString(fieldView)
		content.WriteString("\n")
	}

	builder.WriteString(sectionStyle.Render(content.String()))

	return builder.String()
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
func (m *Config) renderFooter() string {
	var keybindings []string

	keybindings = append(keybindings, m.styles.Keybinding("←→", "switch tabs"))
	keybindings = append(keybindings, m.styles.Keybinding("enter", "edit section"))
	keybindings = append(keybindings, m.styles.Keybinding("s", "save config"))
	keybindings = append(keybindings, m.styles.Keybinding("esc", "back"))
	keybindings = append(keybindings, m.styles.Keybinding("q", "quit"))

	footer := strings.Join(keybindings, "  ")

	return m.styles.Footer.Render(footer)
}

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

func (m *Config) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
