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
	"github.com/mattn/go-runewidth"
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

	// Viewport for proper scrolling (no pagination hacks)
	viewport viewport.Model
	ready    bool
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

	return &Themes{
		styles:        styleConfig,
		themes:        themes,
		cursor:        0,
		selectedTheme: 0,
		showPreview:   true,
		keyMap:        DefaultThemesKeyMap(),
		// viewport initialized in handleWindowSizeMsg (idiomatic pattern)
		ready: false,
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
	var cmd tea.Cmd

	// First handle our specific messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	}

	// Delegate to viewport (idiomatic Bubble Tea pattern from glamour example)
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

// View renders the themes screen following idiomatic Bubble Tea patterns.
func (m *Themes) View() string {
	if m.quitting {
		return "Theme applied! Goodbye!"
	}

	if !m.ready {
		return "Loading themes..."
	}

	// Pure view rendering - never call SetContent in View() (idiomatic Bubble Tea)
	return m.viewport.View()
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

	// Determine which view to show based on preview state and size
	if m.showPreview && m.height >= 15 {
		// Split view: themes list + preview (content area height >= 15)
		content := m.renderSplitView()
		m.viewport.SetContent(content)
		// Reset viewport position when content changes to avoid clipping
		m.viewport.GotoTop()
	} else {
		// Full width themes list (small content area or preview off)
		content := m.renderThemesList()
		m.viewport.SetContent(content)
		m.viewport.GotoTop()
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

// renderPreview renders the theme preview panel following official examples pattern.
func (m *Themes) renderPreview(columnWidth int) string {
	if m.cursor >= len(m.themes) {
		return ""
	}

	theme := m.themes[m.cursor]

	// Calculate content width accounting for border and padding
	var contentWidth int
	if columnWidth > 0 {
		contentWidth = columnWidth - 6 // Account for border (2) + padding (2) + margin (2)
		if contentWidth < 20 {
			contentWidth = 20 // Minimum for readable content
		}
	} else {
		contentWidth = 30 // Default preview width
	}

	// Use height-aware rendering if viewport height is limited
	if m.height > 0 && m.height < 25 {
		// Use compact preview for smaller heights
		return m.renderCompactPreview(contentWidth, m.height-5) // Account for header/footer
	}

	// Complete preview content with all theme information
	sections := []string{
		m.styles.Title.Render("Theme Preview"),
		"",
		m.renderTerminalDemoWithWidth(theme, contentWidth),
		"",
		m.renderCodeDemoWithWidth(theme, contentWidth),
		"",
		m.renderColorDemoWithWidth(theme, contentWidth),
		"",
		m.renderThemeDetailsWithWidth(theme, contentWidth),
	}

	// Pure Lipgloss composition - complete preview sections
	return strings.Join(sections, "\n")
}

// renderCompactPreview renders a compact theme preview that fits within height constraint.
func (m *Themes) renderCompactPreview(width, height int) string {
	if m.cursor >= len(m.themes) {
		return ""
	}

	theme := m.themes[m.cursor]

	// Reserve space for title and spacing
	titleHeight := 2 // "Theme Preview" + blank line

	contentHeight := height - titleHeight
	if contentHeight < 4 {
		contentHeight = 4 // Minimum for any useful content
	}

	sections := []string{
		m.styles.Title.Render("Theme Preview"),
		"", // Spacing
	}

	remainingHeight := contentHeight

	// Add terminal demo if space allows
	if remainingHeight >= 4 {
		sections = append(sections, m.renderTerminalDemo(theme))

		remainingHeight -= 4 // Approximate terminal demo height
		if remainingHeight > 0 {
			sections = append(sections, "") // Spacing
			remainingHeight--
		}
	}

	// Add color demo if space allows
	if remainingHeight >= 3 {
		sections = append(sections, m.renderCompactColorDemo(theme, remainingHeight))
	}

	// Join with consistent spacing
	content := strings.Join(sections, "\n")

	// Apply width constraint to ensure consistent layout
	if width > 0 {
		contentStyle := lipgloss.NewStyle().Width(width - 4) // Account for padding
		content = contentStyle.Render(content)
	}

	return content
}

// renderCompactColorDemo renders a compact color palette preview.
func (m *Themes) renderCompactColorDemo(theme Theme, height int) string {
	if height < 2 {
		return "" // Not enough space
	}

	var builder strings.Builder
	builder.WriteString("Colors:\n")

	// Show just the most important colors
	colors := []struct {
		name  string
		value string
	}{
		{"Fg", theme.Colors.Foreground},
		{"Bg", theme.Colors.Background},
		{"Primary", theme.Colors.Primary},
	}

	linesUsed := 1 // "Colors:" line
	for _, color := range colors {
		if linesUsed >= height {
			break
		}

		colorSwatch := lipgloss.NewStyle().
			Background(lipgloss.Color(color.value)).
			Foreground(lipgloss.Color(theme.Colors.Foreground)).
			Padding(0, 1).
			Render("██")

		line := fmt.Sprintf("%s %s", colorSwatch, color.name)
		builder.WriteString(line)

		if linesUsed < height-1 {
			builder.WriteString("\n")
		}

		linesUsed++
	}

	return builder.String()
}

// renderColorDemoWithWidth renders color palette preview with responsive width.
func (m *Themes) renderColorDemoWithWidth(theme Theme, maxWidth int) string {
	var builder strings.Builder

	builder.WriteString("Color Palette:\n")

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

	// Calculate available width for color entries (maxWidth is already content width)
	availableWidth := maxWidth // No need to subtract padding again
	if availableWidth < 15 {
		availableWidth = 15 // Minimum for readable content
	}

	for _, color := range colors {
		colorSwatch := lipgloss.NewStyle().
			Background(lipgloss.Color(color.value)).
			Foreground(lipgloss.Color(theme.Colors.Foreground)).
			Padding(0, 1).
			Render("██")

		// Truncate color value if line too long (use proper Unicode width)
		colorValue := color.value

		maxLineLength := availableWidth - 2 - runewidth.StringWidth(color.name) - 3 // swatch + name + spaces
		if runewidth.StringWidth(colorValue) > maxLineLength && maxLineLength > 3 {
			colorValue = runewidth.Truncate(colorValue, maxLineLength, "...")
		}

		line := fmt.Sprintf("%s %s %s", colorSwatch, color.name, colorValue)
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	// Responsive width
	colorWidth := min(maxWidth, 30) // Max 30, but respect column width
	if colorWidth < 15 {
		colorWidth = 15 // Minimum readable width
	}

	// Simple style without border (will be in preview card)
	colorStyle := lipgloss.NewStyle().
		Padding(1).
		Width(colorWidth) // Responsive width

	return colorStyle.Render(builder.String())
}

// renderTerminalDemo renders terminal preview with consistent dimensions.
func (m *Themes) renderTerminalDemo(theme Theme) string {
	// Use reusable theme styles (idiomatic pattern)
	themeStyles := m.getThemeStyles(theme)

	content := fmt.Sprintf("%s\n%s\n%s\n%s",
		themeStyles.Primary.Render("user@karei:~$ git status"),
		themeStyles.Success.Render("✓ On branch main"),
		themeStyles.Error.Render("✗ Uncommitted changes"),
		themeStyles.Primary.Render("user@karei:~$ _"))

	terminalContent := "Terminal\n" + content

	// Consistent sizing with theme colors
	terminalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Primary)).
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Foreground)).
		Padding(1).
		Height(7). // Fixed height - idiomatic consistent sizing
		Width(30)  // Fixed width for consistency

	return terminalStyle.Render(terminalContent)
}

// renderTerminalDemoWithWidth renders terminal preview with responsive width.
func (m *Themes) renderTerminalDemoWithWidth(theme Theme, maxWidth int) string {
	// Use reusable theme styles (idiomatic pattern)
	themeStyles := m.getThemeStyles(theme)

	content := fmt.Sprintf("%s\n%s\n%s",
		themeStyles.Primary.Render("$ git status"),
		themeStyles.Success.Render("✓ Clean"),
		themeStyles.Error.Render("✗ 2 changes"))

	terminalContent := "Terminal\n" + content

	// Responsive width - fit within column
	terminalWidth := min(maxWidth, 30) // Max 30, but respect column width
	if terminalWidth < 15 {
		terminalWidth = 15 // Minimum readable width
	}

	// Consistent styling with theme colors (no border - will be in preview card)
	terminalStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Foreground)).
		Padding(1).
		Width(terminalWidth) // Responsive width

	return terminalStyle.Render(terminalContent)
}

// renderCodeDemoWithWidth renders code editor preview with responsive width.
func (m *Themes) renderCodeDemoWithWidth(theme Theme, maxWidth int) string {
	// Use reusable theme styles (idiomatic pattern)
	themeStyles := m.getThemeStyles(theme)

	codeContent := fmt.Sprintf("Editor\n%s {\n    %s\n    %s\n}",
		themeStyles.Secondary.Render("func main()"),
		themeStyles.Success.Render("fmt.Println(\"Hello!\")"),
		themeStyles.Secondary.Render("return true"))

	// Responsive width - fit within column
	codeWidth := min(maxWidth, 30) // Max 30, but respect column width
	if codeWidth < 15 {
		codeWidth = 15 // Minimum readable width
	}

	// Consistent styling with theme colors (no border - will be in preview card)
	codeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Foreground)).
		Padding(1).
		Width(codeWidth) // Responsive width

	return codeStyle.Render(codeContent)
}

// renderThemeDetailsWithWidth renders theme details with responsive width.
func (m *Themes) renderThemeDetailsWithWidth(theme Theme, maxWidth int) string {
	// Calculate available width for content (maxWidth is already content width)
	availableWidth := maxWidth // No need to subtract padding again
	if availableWidth < 20 {
		availableWidth = 20 // Minimum for readable content
	}

	// Truncate description to fit available width (use proper Unicode width)
	description := theme.Description
	if runewidth.StringWidth(description) > availableWidth-10 { // Leave space for bullet
		description = runewidth.Truncate(description, availableWidth-13, "...")
	}

	// Format applications list to fit width (use proper Unicode width)
	apps := strings.Join(theme.Applications[:min(3, len(theme.Applications))], ", ")
	if runewidth.StringWidth(apps) > availableWidth-8 { // Leave space for "Apps: "
		apps = runewidth.Truncate(apps, availableWidth-11, "...")
	} else if len(theme.Applications) > 3 {
		apps += "..."
	}

	// Create width-aware details content
	details := fmt.Sprintf(`Theme Details:
• %s
• Apps: %s  
• Variants: %s
• Style: %s`,
		description,
		apps,
		strings.Join(theme.Variants[:min(2, len(theme.Variants))], ", "),
		runewidth.Truncate(theme.Wallpaper, availableWidth/2, "..."))

	// Responsive width
	detailsWidth := min(maxWidth, 30) // Max 30, but respect column width
	if detailsWidth < 20 {
		detailsWidth = 20 // Minimum readable width
	}

	// Apply simple styling (no border - will be in preview card)
	detailsStyle := lipgloss.NewStyle().
		Padding(1).
		Width(detailsWidth). // Responsive width
		Render(details)

	return detailsStyle
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

func (m *Themes) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	// Delegate unhandled keys to viewport (idiomatic pattern for scrolling support)
	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
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

	// Use the content height directly since main app already calculated content area
	contentHeight := msg.Height
	if contentHeight < 5 {
		contentHeight = 5 // Minimum viable height
	}

	if !m.ready {
		// Initialize viewport with content area size
		m.viewport = viewport.New(msg.Width, contentHeight)
		m.ready = true
		// Set initial content after viewport is ready
		m.updateViewportContent()
	} else {
		// Update viewport size to content area
		m.viewport.Width = msg.Width
		m.viewport.Height = contentHeight
		// Refresh content for new size
		m.updateViewportContent()
	}

	return m, nil
}

// renderSplitView renders the split view following official split-editors pattern.
func (m *Themes) renderSplitView() string {
	if m.width <= 0 {
		m.width = 80 // Fallback for initialization
	}

	// Check if terminal is too narrow for split view
	if m.width < 40 {
		// Terminal too narrow - fall back to single column
		return m.renderThemesList()
	}

	// Following split-editors pattern: dynamic sizing with simple division
	columnWidth := m.width / 2

	// Create content with MaxWidth constraint (idiomatic Lipgloss pattern)
	themesListContent := m.renderThemesForColumn(columnWidth)
	previewContent := m.renderPreview(columnWidth)

	// Create styled columns following official examples pattern
	leftColumn := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Muted).
		Padding(1).
		MaxWidth(columnWidth). // Use MaxWidth instead of Width
		Render(themesListContent)

	rightColumn := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Muted).
		Padding(1).
		MaxWidth(columnWidth). // Use MaxWidth instead of Width
		Render(previewContent)

	// Pure Lipgloss composition following official pattern
	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)
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

// renderThemesForColumn renders themes following official examples pattern.
func (m *Themes) renderThemesForColumn(columnWidth int) string {
	var builder strings.Builder

	title := m.getColumnTitle(columnWidth)
	builder.WriteString(m.styles.Title.Render(title))
	builder.WriteString("\n\n")

	// Render themes with width-aware truncation when needed
	for themeIndex, theme := range m.themes {
		themeEntry := m.renderSingleTheme(themeIndex, theme, columnWidth)
		builder.WriteString(themeEntry)
		builder.WriteString("\n")
	}

	return builder.String()
}

func (m *Themes) getColumnTitle(columnWidth int) string {
	title := "Available Themes"
	if columnWidth > 0 && runewidth.StringWidth(title) > columnWidth-6 { // Account for border+padding
		title = "Themes"
	}

	return title
}

func (m *Themes) renderSingleTheme(themeIndex int, theme Theme, columnWidth int) string {
	style, prefix := m.getThemeStyle(themeIndex)
	themeName, currentIndicator := m.formatThemeName(theme, columnWidth, prefix)

	line := prefix + themeName + currentIndicator

	return style.Render(line)
}

func (m *Themes) getThemeStyle(themeIndex int) (lipgloss.Style, string) {
	if themeIndex == m.cursor {
		return m.styles.Selected, "❯ "
	}

	return m.styles.Unselected, "  "
}

func (m *Themes) formatThemeName(theme Theme, columnWidth int, prefix string) (string, string) {
	themeName := theme.DisplayName

	currentIndicator := ""
	if theme.Current {
		currentIndicator = " (current)"
	}

	// Apply width constraints if specified
	if columnWidth > 0 {
		themeName, currentIndicator = m.applyWidthConstraints(themeName, currentIndicator, columnWidth, prefix)
	}

	return themeName, currentIndicator
}

func (m *Themes) applyWidthConstraints(themeName, currentIndicator string, columnWidth int, prefix string) (string, string) {
	// Account for border (2) + padding (2) + prefix width
	availableWidth := columnWidth - 6 - runewidth.StringWidth(prefix)
	if availableWidth < 10 {
		availableWidth = 10 // Minimum readable width
	}

	// Truncate if needed using Unicode-safe operations
	fullLine := themeName + currentIndicator
	if runewidth.StringWidth(fullLine) > availableWidth {
		if runewidth.StringWidth(themeName) > availableWidth-3 {
			themeName = runewidth.Truncate(themeName, availableWidth-3, "...")
		}
		// Skip current indicator if no space
		if runewidth.StringWidth(themeName)+runewidth.StringWidth(currentIndicator) > availableWidth {
			currentIndicator = ""
		}
	}

	return themeName, currentIndicator
}

// ensureSelectionVisible calculates exact line position of selection in rendered content.
// Follows the same pattern as apps screen for proper viewport scrolling.
func (m *Themes) ensureSelectionVisible() {
	if !m.ready || len(m.themes) == 0 {
		return
	}

	// Calculate EXACT line position of current selection in rendered content
	selectionLine := m.calculateThemeSelectionLine()

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

// renderThemesList renders the full width themes list using idiomatic Lipgloss.
func (m *Themes) renderThemesList() string {
	if !m.ready {
		return "Loading themes..."
	}

	// Just return all themes content - main View() handles viewport setup
	return m.renderAllThemes()
}
