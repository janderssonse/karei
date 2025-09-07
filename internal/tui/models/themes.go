// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements theme selection and preview UI.
package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// Theme represents a system theme.
type Theme struct {
	Name         string
	DisplayName  string
	Description  string
	Icon         string
	Colors       ThemeColors
	Variants     []string
	Applications []string
	Wallpaper    string
	Current      bool
}

// ThemeColors represents the color palette of a theme.
type ThemeColors struct {
	Primary    string
	Secondary  string
	Success    string
	Warning    string
	Error      string
	Background string
	Foreground string
}

// Themes represents the theme selection screen model.
type Themes struct {
	styles        *styles.Styles
	width         int
	height        int
	themes        []Theme
	cursor        int
	selectedTheme int
	showPreview   bool
	quitting      bool
	keyMap        ThemesKeyMap

	// Two viewports for split view (idiomatic approach)
	listViewport    viewport.Model
	previewViewport viewport.Model
	ready           bool

	// Help modal
	helpModal *HelpModal
}

// ThemesKeyMap defines key bindings for the themes screen.
type ThemesKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	Preview key.Binding
	Apply   key.Binding
	Variant key.Binding
	Back    key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// DefaultThemesKeyMap returns the default key bindings.
func DefaultThemesKeyMap() ThemesKeyMap {
	return ThemesKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select theme"),
		),
		Preview: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "toggle preview"),
		),
		Apply: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "apply theme"),
		),
		Variant: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "change variant"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
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

// NewThemes creates a new theme selection model.
func NewThemes(styleConfig *styles.Styles) *Themes {
	// Define available themes (alphabetically sorted by DisplayName)
	themes := []Theme{
		{
			Name:        "catppuccin-latte",
			DisplayName: "☀️ Catppuccin Latte",
			Description: "Soothing light pastel theme for the high-spirited",
			Icon:        "☀️",
			Colors: ThemeColors{
				Primary:    "#1e66f5",
				Secondary:  "#8839ef",
				Success:    "#40a02b",
				Warning:    "#df8e1d",
				Error:      "#d20f39",
				Background: "#eff1f5",
				Foreground: "#4c4f69",
			},
			Variants:     []string{"Light"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Light minimalist cat silhouettes",
		},
		{
			Name:        "catppuccin",
			DisplayName: "🐱 Catppuccin Mocha",
			Description: "Soothing dark pastel theme for the high-spirited",
			Icon:        "🐱",
			Colors: ThemeColors{
				Primary:    "#89b4fa",
				Secondary:  "#cba6f7",
				Success:    "#a6e3a1",
				Warning:    "#f9e2af",
				Error:      "#f38ba8",
				Background: "#1e1e2e",
				Foreground: "#cdd6f4",
			},
			Variants:     []string{"Latte", "Frappé", "Macchiato", "Mocha"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Minimalist cat silhouettes",
		},
		{
			Name:        "dracula",
			DisplayName: "🧛 Dracula",
			Description: "Dark theme for hackers - over 300 apps supported",
			Icon:        "🧛",
			Colors: ThemeColors{
				Primary:    "#8be9fd",
				Secondary:  "#bd93f9",
				Success:    "#50fa7b",
				Warning:    "#f1fa8c",
				Error:      "#ff5555",
				Background: "#282a36",
				Foreground: "#f8f8f2",
			},
			Variants:     []string{"Dark"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME", "300+ apps"},
			Wallpaper:    "Gothic vampire aesthetic",
		},
		{
			Name:        "everforest",
			DisplayName: "🌲 Everforest",
			Description: "Green based color scheme designed to be warm and soft",
			Icon:        "🌲",
			Colors: ThemeColors{
				Primary:    "#a7c080",
				Secondary:  "#d699b6",
				Success:    "#a7c080",
				Warning:    "#dbbc7f",
				Error:      "#e67e80",
				Background: "#2d353b",
				Foreground: "#d3c6aa",
			},
			Variants:     []string{"Dark", "Light"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Forest landscapes",
		},
		{
			Name:        "gruvbox-light",
			DisplayName: "☀️ Gruvbox Light",
			Description: "Light variant of the retro groove color scheme",
			Icon:        "☀️",
			Colors: ThemeColors{
				Primary:    "#3c8588",
				Secondary:  "#b16286",
				Success:    "#98971a",
				Warning:    "#d79921",
				Error:      "#cc241d",
				Background: "#fbf1c7",
				Foreground: "#282828",
			},
			Variants:     []string{"Light"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Light rustic textures",
		},
		{
			Name:        "gruvbox",
			DisplayName: "🟤 Gruvbox",
			Description: "Retro groove color scheme with warm, earthy tones",
			Icon:        "🟤",
			Colors: ThemeColors{
				Primary:    "#83a598",
				Secondary:  "#d3869b",
				Success:    "#b8bb26",
				Warning:    "#fabd2f",
				Error:      "#fb4934",
				Background: "#282828",
				Foreground: "#ebdbb2",
			},
			Variants:     []string{"Dark", "Light"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Rustic textures",
		},
		{
			Name:        "kanagawa",
			DisplayName: "🌊 Kanagawa",
			Description: "Dark theme inspired by Kanagawa's famous wave painting",
			Icon:        "🌊",
			Colors: ThemeColors{
				Primary:    "#7E9CD8",
				Secondary:  "#957FB8",
				Success:    "#76946A",
				Warning:    "#FF9E3B",
				Error:      "#C34043",
				Background: "#1F1F28",
				Foreground: "#DCD7BA",
			},
			Variants:     []string{"Dark", "Light"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Japanese wave art",
		},
		{
			Name:        "monokai",
			DisplayName: "🎨 Monokai Pro",
			Description: "The iconic theme that inspired millions of developers",
			Icon:        "🎨",
			Colors: ThemeColors{
				Primary:    "#66d9ef",
				Secondary:  "#ae81ff",
				Success:    "#a6e22e",
				Warning:    "#e6db74",
				Error:      "#f92672",
				Background: "#2d2a2e",
				Foreground: "#fcfcfa",
			},
			Variants:     []string{"Classic", "Pro", "Machine", "Octagon"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Vibrant creative energy",
		},
		{
			Name:        "nord",
			DisplayName: "🧊 Nord",
			Description: "Quiet and comfortable arctic color palette",
			Icon:        "🧊",
			Colors: ThemeColors{
				Primary:    "#88c0d0",
				Secondary:  "#b48ead",
				Success:    "#a3be8c",
				Warning:    "#ebcb8b",
				Error:      "#bf616a",
				Background: "#2e3440",
				Foreground: "#d8dee9",
			},
			Variants:     []string{"Standard"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Minimal nordic landscapes",
		},
		{
			Name:        "one-dark",
			DisplayName: "⚡ One Dark Pro",
			Description: "Atom's iconic One Dark theme - beloved by millions",
			Icon:        "⚡",
			Colors: ThemeColors{
				Primary:    "#61afef",
				Secondary:  "#c678dd",
				Success:    "#98c379",
				Warning:    "#e5c07b",
				Error:      "#e06c75",
				Background: "#282c34",
				Foreground: "#abb2bf",
			},
			Variants:     []string{"Dark", "Vivid"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Modern minimalist dark",
		},
		{
			Name:        "rose-pine",
			DisplayName: "🌹 Rose Pine",
			Description: "All natural pine, faux fur and a bit of soho vibes",
			Icon:        "🌹",
			Colors: ThemeColors{
				Primary:    "#56949f",
				Secondary:  "#907aa9",
				Success:    "#286983",
				Warning:    "#ea9d34",
				Error:      "#b4637a",
				Background: "#faf4ed",
				Foreground: "#575279",
			},
			Variants:     []string{"Dawn", "Moon"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Elegant botanical patterns",
		},
		{
			Name:        "solarized-dark",
			DisplayName: "🌚 Solarized Dark",
			Description: "Precision colors for machines and people",
			Icon:        "🌚",
			Colors: ThemeColors{
				Primary:    "#268bd2",
				Secondary:  "#d33682",
				Success:    "#859900",
				Warning:    "#b58900",
				Error:      "#dc322f",
				Background: "#002b36",
				Foreground: "#839496",
			},
			Variants:     []string{"Dark", "Light"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Scientific precision",
		},
		{
			Name:        "solarized-light",
			DisplayName: "🌞 Solarized Light",
			Description: "Precision colors for machines and people (light)",
			Icon:        "🌞",
			Colors: ThemeColors{
				Primary:    "#268bd2",
				Secondary:  "#d33682",
				Success:    "#859900",
				Warning:    "#b58900",
				Error:      "#dc322f",
				Background: "#fdf6e3",
				Foreground: "#657b83",
			},
			Variants:     []string{"Light"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Warm light precision",
		},
		{
			Name:        "tokyo-night",
			DisplayName: "🌃 Tokyo Night",
			Description: "A dark theme inspired by Tokyo's neon-lit streets",
			Icon:        "🌃",
			Colors: ThemeColors{
				Primary:    "#7aa2f7",
				Secondary:  "#bb9af7",
				Success:    "#9ece6a",
				Warning:    "#e0af68",
				Error:      "#f7768e",
				Background: "#1a1b26",
				Foreground: "#a9b1d6",
			},
			Variants:     []string{"Dark", "Light", "Storm"},
			Applications: []string{"Terminal", "VS Code", "Neovim", "GNOME"},
			Wallpaper:    "Tokyo cityscape (4K)",
			Current:      true, // This would come from system detection
		},
	}

	// Create help modal
	helpModal := NewHelpModal()
	helpModal.SetScreen("themes")

	return &Themes{
		styles:        styleConfig,
		themes:        themes,
		cursor:        0,
		selectedTheme: 0,
		showPreview:   true,
		keyMap:        DefaultThemesKeyMap(),
		// viewport initialized in handleWindowSizeMsg (idiomatic pattern)
		ready:     false,
		helpModal: helpModal,
	}
}

// Init initializes the themes model.
func (m *Themes) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the Themes model.
func (m *Themes) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var cmd tea.Cmd

	// Handle window size first as it affects viewports

	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		return m.handleWindowSizeMsg(msg)
	}

	// Update viewports BEFORE handling keys - this ensures they process all messages
	// Always update list viewport (it's always visible)
	m.listViewport, cmd = m.listViewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update preview viewport if visible
	if m.showPreview && m.height >= 15 && m.ready {
		m.previewViewport, cmd = m.previewViewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Now handle our keyboard input
	if msg, ok := msg.(tea.KeyMsg); ok {
		model, cmd := m.handleKeyMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		return model, tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}

// View renders the themes screen following idiomatic Bubble Tea patterns.
func (m *Themes) View() string {
	if m.quitting {
		return "Theme applied! Goodbye!"
	}

	if !m.ready {
		return "Loading themes..."
	}

	// If help modal is visible, show it as an overlay
	if m.helpModal != nil && m.helpModal.IsVisible() {
		return m.renderWithModal()
	}

	return m.renderBaseView()
}

// renderWithModal renders the view with modal overlay.
func (m *Themes) renderWithModal() string {
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

// renderBaseView renders the main themes view without overlays.
func (m *Themes) renderBaseView() string {
	// Build view components
	var components []string

	// Add clean header
	header := m.renderCleanHeader()
	components = append(components, header)

	// Calculate remaining height for viewports
	headerHeight := lipgloss.Height(header)
	footerHeight := 3 // Footer height (consistent with apps page)
	viewportHeight := m.height - headerHeight - footerHeight - 1

	// Render main content (split view or single column)
	var mainContent string

	if m.showPreview && viewportHeight >= 15 {
		// Get viewport content
		leftContent := m.listViewport.View()
		rightContent := m.previewViewport.View()

		// Use lipgloss styles with explicit dimensions to force clean rendering
		// This prevents viewport artifacts (see bubbles#454)
		leftStyle := lipgloss.NewStyle().
			Width(35).
			Height(viewportHeight).
			MaxHeight(viewportHeight).
			Margin(0). // Critical: margin 0 prevents artifacts
			Padding(0)

		rightStyle := lipgloss.NewStyle().
			Width(m.width - 36).
			Height(viewportHeight).
			MaxHeight(viewportHeight).
			Margin(0). // Critical: margin 0 prevents artifacts
			Padding(0)

		// Apply styles to ensure consistent rendering
		leftStyled := leftStyle.Render(leftContent)
		rightStyled := rightStyle.Render(rightContent)

		// Join horizontally with top alignment
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)
	} else {
		// Single column view
		mainContent = m.listViewport.View()
	}

	components = append(components, mainContent)

	// Add clean footer
	footer := m.renderCleanFooter()
	components = append(components, footer)

	// Compose all components
	return lipgloss.JoinVertical(lipgloss.Top, components...)
}

// renderCleanHeader renders the new simplified header format.
func (m *Themes) renderCleanHeader() string {
	// Left side: App name » Current location
	location := "Karei » Theme Selection"
	leftSide := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary).
		Render(location)

	// Right side: Current theme name
	status := ""

	if m.cursor >= 0 && m.cursor < len(m.themes) {
		currentTheme := m.themes[m.cursor]
		if currentTheme.Current {
			status = currentTheme.DisplayName + " (current)"
		} else {
			status = currentTheme.DisplayName
		}
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
func (m *Themes) renderCleanFooter() string {
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

	// Theme selection actions
	actions = []string{
		formatAction("←→", "Choose"),
		formatAction("Enter", "Apply"),
		formatAction("Space", "Preview"),
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

// GetSelectedTheme returns the theme at the current selection index.
func (m *Themes) GetSelectedTheme() Theme {
	if m.selectedTheme >= 0 && m.selectedTheme < len(m.themes) {
		return m.themes[m.selectedTheme]
	}

	return Theme{}
}

// updateViewportContent updates the viewport content based on current state.
// This should be called when content needs to be refreshed (idiomatic pattern).
func (m *Themes) updateViewportContent() {
	if !m.ready {
		return
	}

	// Update list viewport with themes
	themesContent := m.renderAllThemes()
	m.listViewport.SetContent(themesContent)

	// Update preview viewport if showing preview
	if m.showPreview && m.height >= 15 {
		previewContent := m.renderPreviewContent()
		m.previewViewport.SetContent(previewContent)
		// Reset preview viewport to top for clean rendering
		m.previewViewport.GotoTop()
	}
}

// getThemeStyles returns reusable styles for a theme (idiomatic pattern).
func (m *Themes) getThemeStyles(theme Theme) struct {
	Primary   lipgloss.Style
	Secondary lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Error     lipgloss.Style
} {
	return struct {
		Primary   lipgloss.Style
		Secondary lipgloss.Style
		Success   lipgloss.Style
		Warning   lipgloss.Style
		Error     lipgloss.Style
	}{
		Primary:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Primary)),
		Secondary: lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Secondary)),
		Success:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Success)),
		Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Warning)),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Error)),
	}
}

// renderPreviewContent renders the preview content for the preview viewport.
func (m *Themes) renderPreviewContent() string {
	if m.cursor >= len(m.themes) {
		return ""
	}

	theme := m.themes[m.cursor]

	// Build preview sections without worrying about width constraints
	sections := []string{
		m.styles.Title.Render("Theme Preview - " + theme.DisplayName),
		"",
		m.renderTerminalDemoWithWidth(theme, 0),
		"",
		m.renderCodeDemoWithWidth(theme, 0),
		"",
		m.renderColorDemoWithWidth(theme, 0),
		"",
		m.renderThemeDetailsWithWidth(theme, 0),
	}

	return strings.Join(sections, "\n")
}

// renderCompactPreview renders a compact theme preview that fits within height constraint.
func (m *Themes) renderColorDemoWithWidth(theme Theme, _ int) string {
	// Color swatches
	colors := []struct {
		name  string
		value string
	}{
		{"Primary", theme.Colors.Primary},
		{"Secondary", theme.Colors.Secondary},
		{"Success", theme.Colors.Success},
		{"Warning", theme.Colors.Warning},
		{"Error", theme.Colors.Error},
		{"Background", theme.Colors.Background},
		{"Foreground", theme.Colors.Foreground},
	}

	// Build color palette with larger, clearer hex values
	rows := make([]string, 0, 9) // 7 colors + 2 headers
	rows = append(rows, lipgloss.NewStyle().Bold(true).Render("Color Palette"))
	rows = append(rows, "")

	for _, color := range colors {
		// Create color swatch (wider for better visibility)
		swatch := lipgloss.NewStyle().
			Background(lipgloss.Color(color.value)).
			Render("      ") // 6 spaces for wider swatch

		// Format name and hex value with better spacing and emphasis
		// Make hex value bold for better readability
		nameAndValue := fmt.Sprintf(" %-10s %s",
			color.name,
			lipgloss.NewStyle().Bold(true).Render(color.value))

		// Combine swatch with formatted text
		row := swatch + nameAndValue

		rows = append(rows, row)
	}

	// Join all rows
	content := strings.Join(rows, "\n")

	// Create bordered box with margin 0 to prevent viewport artifacts (see bubbles#454)
	colorStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Success)).
		Padding(1).
		Margin(0) // Critical: margin 0 prevents viewport artifacts

	return colorStyle.Render(content)
}

// renderTerminalDemoWithWidth renders terminal preview with responsive width.
func (m *Themes) renderTerminalDemoWithWidth(theme Theme, _ int) string {
	// Use reusable theme styles (idiomatic pattern)
	themeStyles := m.getThemeStyles(theme)

	// Build lines applying style to ENTIRE line to avoid alignment issues
	// Use ASCII characters instead of Unicode to avoid width calculation issues
	lines := []string{
		"Terminal Preview",
		"",
		themeStyles.Primary.Render("$ git status"),
		themeStyles.Success.Render("[OK] On branch main"),
		themeStyles.Warning.Render("[!] Your branch is up to date"),
		themeStyles.Error.Render("[X] Untracked files present"),
		themeStyles.Secondary.Render("$ _"),
	}

	terminalContent := strings.Join(lines, "\n")

	// Create bordered box with margin 0 to prevent viewport artifacts
	terminalStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Primary)).
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Foreground)).
		Padding(1).
		Margin(0) // Critical: margin 0 prevents viewport artifacts

	return terminalStyle.Render(terminalContent)
}

// renderCodeDemoWithWidth renders code editor preview with responsive width.
func (m *Themes) renderCodeDemoWithWidth(theme Theme, _ int) string {
	// Use reusable theme styles (idiomatic pattern)
	themeStyles := m.getThemeStyles(theme)

	// Build code lines
	codeLines := []string{
		"Code Editor",
		"",
		themeStyles.Warning.Render("// Apply theme to system"),
		themeStyles.Secondary.Render("func applyTheme(name string) error {"),
		"    " + themeStyles.Primary.Render("theme") + " := " + themeStyles.Success.Render("\"catppuccin\""),
		"    " + themeStyles.Error.Render("return fmt.Errorf(\"not found\")"),
		"    " + themeStyles.Secondary.Render("return nil"),
		"}",
	}

	codeContent := strings.Join(codeLines, "\n")

	// Create bordered box with margin 0 to prevent viewport artifacts
	codeStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Secondary)).
		Padding(1).
		Margin(0) // Critical: margin 0 prevents viewport artifacts

	return codeStyle.Render(codeContent)
}

// renderThemeDetailsWithWidth renders theme details with responsive width.
func (m *Themes) renderThemeDetailsWithWidth(theme Theme, _ int) string {
	// Build comprehensive theme details
	var builder strings.Builder
	builder.WriteString("Theme Information\n\n")

	// Description
	builder.WriteString("📝 Description:\n")
	builder.WriteString(fmt.Sprintf("   %s\n\n", theme.Description))

	// Applications in a nice list
	builder.WriteString("📦 Supported Applications:\n")

	for i, app := range theme.Applications {
		if i < 4 { // Show first 4 apps
			builder.WriteString(fmt.Sprintf("   • %s\n", app))
		}
	}

	if len(theme.Applications) > 4 {
		builder.WriteString(fmt.Sprintf("   • ... and %d more\n", len(theme.Applications)-4))
	}

	builder.WriteString("\n")

	// Variants
	builder.WriteString("🎨 Available Variants:\n")

	variantList := strings.Join(theme.Variants, " | ")
	builder.WriteString(fmt.Sprintf("   %s\n\n", variantList))

	// Wallpaper/Style info
	builder.WriteString("🖼️ Wallpaper Style:\n")
	builder.WriteString(fmt.Sprintf("   %s\n", theme.Wallpaper))

	// Current theme indicator
	if theme.Current {
		builder.WriteString("\n✅ This is your current theme")
	}

	// Create bordered box with margin 0 to prevent viewport artifacts
	detailsStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Warning)).
		Padding(1).
		Margin(0) // Critical: margin 0 prevents viewport artifacts

	return detailsStyle.Render(builder.String())
}

// applyTheme applies the selected theme.
func (m *Themes) applyTheme() tea.Cmd {
	// Integrate with hexagonal architecture to actually apply theme (requires theme service port)
	return nil
}

// navigateToMenuCmd returns a command to navigate to the menu screen (idiomatic pattern).
func (m *Themes) navigateToMenuCmd() tea.Cmd {
	return func() tea.Msg {
		return NavigateMsg{Screen: MenuScreen}
	}
}

// handleKeyMsg processes keyboard input for the themes screen.
//

//nolint:cyclop // Complex but necessary for handling various UI interactions
func (m *Themes) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		return m, m.navigateToMenuCmd()
	case key.Matches(msg, m.keyMap.Up):
		return m.handleCursorMovement(-1)
	case key.Matches(msg, m.keyMap.Down):
		return m.handleCursorMovement(1)
	case key.Matches(msg, m.keyMap.Select):
		return m.handleThemeSelection()
	case key.Matches(msg, m.keyMap.Preview):
		return m.handlePreviewToggle()
	case key.Matches(msg, m.keyMap.Apply):
		return m, m.applyTheme()
	case key.Matches(msg, m.keyMap.Variant):
		// Cycle through theme variants (requires state tracking for current variant)
		return m, nil
	}

	// Don't forward key messages to viewports here - they're already handled in Update()
	return m, nil
}

// handleCursorMovement moves the cursor up or down.
//

func (m *Themes) handleCursorMovement(direction int) (tea.Model, tea.Cmd) {
	newCursor := m.cursor + direction
	if newCursor >= 0 && newCursor < len(m.themes) {
		m.cursor = newCursor
		// Update preview content when cursor changes (if preview is showing)
		if m.showPreview {
			m.updateViewportContent()
		}
		// Use Bubble Tea viewport's natural scrolling - no arithmetic
		m.ensureSelectionVisible()
	}

	return m, nil
}

// handleThemeSelection selects the current theme.
//

func (m *Themes) handleThemeSelection() (tea.Model, tea.Cmd) {
	m.selectedTheme = m.cursor

	return m, nil
}

// handlePreviewToggle toggles the preview display.
//

func (m *Themes) handlePreviewToggle() (tea.Model, tea.Cmd) {
	m.showPreview = !m.showPreview
	// Update viewport content when preview state changes (idiomatic pattern)
	m.updateViewportContent()

	return m, nil
}

// handleWindowSizeMsg processes window resize messages.
//

func (m *Themes) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Update help modal size
	if m.helpModal != nil {
		m.helpModal.SetSize(msg.Width, msg.Height)
	}

	// Calculate content height accounting for header and footer
	headerHeight := 3 // Header height (consistent across pages)
	footerHeight := 3 // Footer height (consistent across pages)

	contentHeight := msg.Height - headerHeight - footerHeight
	if contentHeight < 5 {
		contentHeight = 5 // Minimum viable height
	}

	if !m.ready {
		// Initialize both viewports
		// List viewport takes fixed width
		m.listViewport = viewport.New(35, contentHeight)
		// Preview viewport takes remaining width (accounting for space separator)
		m.previewViewport = viewport.New(msg.Width-36, contentHeight)
		m.ready = true
		// Set initial content after viewports are ready
		m.updateViewportContent()
	} else {
		// Update viewport sizes
		m.listViewport.Width = 35
		m.listViewport.Height = contentHeight
		m.previewViewport.Width = msg.Width - 36
		m.previewViewport.Height = contentHeight
		// Refresh content for new size
		m.updateViewportContent()
	}

	return m, nil
}

// renderAllThemes renders all themes for viewport (following apps screen pattern).
func (m *Themes) renderAllThemes() string {
	var builder strings.Builder

	builder.WriteString(m.styles.Title.Render("Available Themes"))
	builder.WriteString("\n\n")

	// Render ALL themes - viewport handles what's visible
	for themeIndex, theme := range m.themes {
		var (
			style  lipgloss.Style
			prefix string
		)

		if themeIndex == m.cursor {
			style = m.styles.Selected
			prefix = "❯ "
		} else {
			style = m.styles.Unselected
			prefix = "  "
		}

		// Current theme indicator
		currentIndicator := ""
		if theme.Current {
			currentIndicator = " (current)"
		}

		line := fmt.Sprintf("%s%s%s", prefix, theme.DisplayName, currentIndicator)
		builder.WriteString(style.Render(line))
		builder.WriteString("\n")

		// Show description for selected theme
		if themeIndex == m.cursor {
			descStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)
			desc := descStyle.Render("    " + theme.Description)
			builder.WriteString(desc)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// ensureSelectionVisible calculates exact line position of selection in rendered content.
// Follows the same pattern as apps screen for proper viewport scrolling.
func (m *Themes) ensureSelectionVisible() {
	if !m.ready || len(m.themes) == 0 {
		return
	}

	// Calculate EXACT line position of current selection in rendered content
	selectionLine := m.calculateThemeSelectionLine()

	// Get current list viewport window
	viewportTop := m.listViewport.YOffset
	viewportBottom := viewportTop + m.listViewport.Height - 1

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

		m.listViewport.SetYOffset(newOffset)
	} else if selectionLine >= viewportBottom-bottomBuffer {
		// Selection too close to bottom - scroll down to maintain buffer
		newOffset := selectionLine - m.listViewport.Height + bottomBuffer + 1
		totalLines := m.listViewport.TotalLineCount()

		maxOffset := totalLines - m.listViewport.Height
		if maxOffset < 0 {
			maxOffset = 0
		}

		if newOffset > maxOffset {
			newOffset = maxOffset
		}

		m.listViewport.SetYOffset(newOffset)
	}
	// Selection is comfortably visible - no scroll needed
}

// calculateThemeSelectionLine calculates exact line position of theme selection.
func (m *Themes) calculateThemeSelectionLine() int {
	// Start with title and spacing
	line := 2 // "Available Themes" + blank line

	// Count lines before current selection
	for range m.cursor {
		line++ // Theme name line
		// Note: we don't count description lines for non-selected themes
	}

	// Add the current selection line
	line++ // Current theme name line

	return line
}

// GetNavigationHints returns screen-specific navigation hints for the footer.
func (m *Themes) GetNavigationHints() []string {
	return []string{
		"[j/k] Navigate",
		"[Space] Select",
		"[p] Preview",
		"[Enter] Apply",
		"[v] Variant",
	}
}
