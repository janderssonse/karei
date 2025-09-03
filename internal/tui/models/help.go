// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements contextual help and keyboard shortcut UI.
package models

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// HelpSection represents a help documentation section.
type HelpSection struct {
	Title   string
	Content string
}

// Help represents the help screen model.
type Help struct {
	styles         *styles.Styles
	width          int
	height         int
	sections       []HelpSection
	viewport       viewport.Model
	renderer       *glamour.TermRenderer
	currentSection int
	quitting       bool
	keyMap         HelpKeyMap
}

// HelpKeyMap defines key bindings for the help screen.
type HelpKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	Tab      key.Binding
	Back     key.Binding
	Quit     key.Binding
}

// DefaultHelpKeyMap returns the default key bindings.
func DefaultHelpKeyMap() HelpKeyMap {
	return HelpKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "previous section"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next section"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup/b", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "f"),
			key.WithHelp("pgdn/f", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to bottom"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next section"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back to menu"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// NewHelp creates a new help model.
//
//nolint:maintidx // Large help content strings are acceptable for documentation
func NewHelp(styleConfig *styles.Styles) *Help {
	// Create help sections with markdown content
	sections := []HelpSection{
		{
			Title: "Getting Started",
			Content: `# Getting Started with Karei

## Welcome to Karei! 🐟

Karei automates Linux development environment setup.

### Quick Start

1. **Launch TUI**: Run ` + "`" + `karei tui` + "`" + ` for interactive interface
2. **Install Apps**: Choose "Install Applications" from main menu
3. **Apply Theme**: Select "Configure Themes" for visual customization
4. **Configure System**: Use "System Settings" for preferences

### Key Features

- 🔧 **Automated Installation**: 40+ development tools and applications
- 🎨 **Coordinated Theming**: 5 beautiful themes across all applications  
- ⚙️ **Smart Configuration**: Intelligent defaults with customization options
- 📊 **Progress Tracking**: Real-time installation progress and logging
- 🔒 **Security Focused**: Enterprise-grade security best practices

### Navigation

| Key | Action |
|-----|--------|
| ↑/↓ or j/k | Navigate menus |
| Enter/Space | Select item |
| Tab | Switch sections |
| Esc | Go back |
| q | Quit |

Ready to transform your Linux system? Let's get started! 🚀`,
		},
		{
			Title: "Installation Guide",
			Content: `# Installation Guide

## Application Categories

### Development Tools 🔧

**Essential development applications:**

- **Git**: Distributed version control system
- **Visual Studio Code**: Feature-rich code editor with extensions  
- **Neovim**: Modern Vim-based text editor
- **Docker Desktop**: Containerization platform
- **JetBrains Toolbox**: IDE management suite
- **Postman**: API development and testing

### System Utilities 📊

**System monitoring and maintenance:**

- **htop**: Interactive process viewer
- **btop**: Beautiful resource monitor
- **Flameshot**: Powerful screenshot tool
- **Timeshift**: System backup and restore
- **Bleachbit**: System cleaner and optimizer

### Media & Graphics 🎨

**Creative and media applications:**

- **GIMP**: Advanced image editing
- **VLC**: Universal media player
- **Spotify**: Music streaming service
- **OBS Studio**: Video recording and streaming

## Installation Process

### 1. Selection Phase
- Browse categories using ↑/↓ keys
- Toggle selection with **Space** key
- Select all in category with **A** key
- Search applications with **/** key

### 2. Installation Phase
- Real-time progress bars for each application
- Download speed and ETA information
- Activity log showing detailed progress
- Pause/resume capability with **P** key

### 3. Verification Phase
- Automatic verification of installations
- Integration testing for installed applications
- Configuration validation and setup

## Tips for Success

> 💡 **Pro Tip**: Start with essential development tools first, then add specialized applications based on your workflow needs.

> ⚠️ **Important**: Ensure stable internet connection for downloading applications.

> 🔒 **Security**: All applications are downloaded from official sources and verified for integrity.`,
		},
		{
			Title: "Theme System",
			Content: `# Theme System Guide

## Coordinated Theming 🎨

Karei provides **coordinated multi-application theming** that applies consistent color schemes across your entire development environment.

## Available Themes

### 🌃 Tokyo Night
**Dark theme inspired by Tokyo's neon-lit streets**
- **Style**: Modern dark with bright accents
- **Best For**: Night coding sessions, reduced eye strain
- **Applications**: Terminal, VS Code, Neovim, GNOME
- **Variants**: Dark (default), Light, Storm

### 🐱 Catppuccin  
**Soothing pastel theme for the high-spirited**
- **Style**: Warm pastels with excellent readability
- **Best For**: Long coding sessions, comfortable viewing
- **Applications**: Terminal, VS Code, Neovim, GNOME
- **Variants**: Latte, Frappé, Macchiato, Mocha

### 🧊 Nord
**Quiet and comfortable arctic color palette**
- **Style**: Cool blues and grays, minimal contrast
- **Best For**: Focus-intensive work, clean aesthetics
- **Applications**: Terminal, VS Code, Neovim, GNOME
- **Variants**: Standard

### 🌲 Everforest
**Green based color scheme designed to be warm and soft**
- **Style**: Forest greens with warm undertones
- **Best For**: Natural feel, easy on the eyes
- **Applications**: Terminal, VS Code, Neovim, GNOME
- **Variants**: Dark, Light

### 🟤 Gruvbox
**Retro groove color scheme with warm, earthy tones**
- **Style**: Vintage-inspired warm colors
- **Best For**: Classic feel, high contrast needs
- **Applications**: Terminal, VS Code, Neovim, GNOME
- **Variants**: Dark, Light

## Theme Application Process

1. **Preview Mode**: See live preview before applying
2. **Color Demonstration**: View color palette swatches
3. **Application Testing**: See how themes look in terminal and code editor
4. **Instant Application**: Changes apply immediately across all supported applications

## Supported Applications

| Application | Tokyo Night | Catppuccin | Nord | Everforest | Gruvbox |
|-------------|-------------|------------|------|------------|---------|
| Ghostty Terminal | ✅ | ✅ | ✅ | ✅ | ✅ |
| VS Code | ✅ | ✅ | ✅ | ✅ | ✅ |
| Neovim | ✅ | ✅ | ✅ | ✅ | ✅ |
| GNOME Desktop | ✅ | ✅ | ✅ | ✅ | ✅ |
| Firefox | ✅ | ✅ | ✅ | ✅ | ✅ |

> **Note**: Restart applications after theme changes to see full effect.`,
		},
		{
			Title: "Configuration",
			Content: `# System Configuration

## Configuration Categories ⚙️

### Shell Environment 🐚

**Configure your command-line experience:**

- **Default Shell**: Choose between Fish, Zsh, or Bash
- **Terminal Emulator**: Ghostty, Alacritty, or GNOME Terminal
- **Font Selection**: JetBrains Mono, Fira Code, Cascadia Code
- **Font Size**: Customizable point size (recommended: 12-16pt)

### Development Tools 🔧

**Set up your development workflow:**

- **Primary Editor**: VS Code, Neovim, or Emacs
- **Git Configuration**: Username and email for commits
- **SSH Key Management**: Automatic key generation and setup
- **Language Environments**: Node.js, Python, Go, Rust versions

### System Preferences 🖥️

**Customize your desktop experience:**

- **Theme Selection**: Choose coordinated color scheme  
- **Wallpaper Options**: Theme default, custom image, or solid color
- **Icon Theme**: Papirus Dark, Adwaita, or Breeze
- **Window Management**: Tiling options and workspace behavior

### Privacy & Security 🔒

**Configure security and privacy settings:**

- **Telemetry Collection**: Disabled, minimal, or full data collection
- **Firewall Protection**: UFW firewall configuration
- **Automatic Updates**: Security-only, all updates, or manual
- **Data Backup**: Timeshift snapshots and scheduling

## Configuration Process

### 1. Tab Navigation
- Use **←/→** arrows to switch between categories
- Each category has its own set of configuration options
- Settings are organized logically by function

### 2. Field Editing
- Press **Enter** to edit current section using forms
- Different field types: dropdowns, text inputs, toggles
- Real-time validation and error checking

### 3. Saving Changes
- Press **S** to save current configuration
- Changes are applied immediately where possible
- Some changes require application restart or re-login

## Best Practices

> 🎯 **Recommendation**: Configure shell environment first, then development tools, then visual preferences.

> 🔄 **Updates**: Review configuration periodically as new applications are installed.

> 📝 **Backup**: Configuration changes are automatically backed up before modification.`,
		},
		{
			Title: "Troubleshooting",
			Content: `# Troubleshooting Guide

## Common Issues & Solutions 🔧

### Installation Problems

#### Application Download Fails
**Symptoms**: Download stops or fails with network error
**Solutions**:
- Check internet connection stability
- Retry installation after network issues resolve
- Use ` + "`" + `karei install <app>` + "`" + ` from CLI for detailed error info
- Check available disk space (minimum 2GB recommended)

#### Permission Errors
**Symptoms**: "Permission denied" or sudo password prompts
**Solutions**:
- Ensure user account has sudo privileges
- Run ` + "`" + `sudo -v` + "`" + ` to refresh sudo session
- Check that user is in ` + "`" + `sudo` + "`" + ` group: ` + "`" + `groups $USER` + "`" + `

#### Package Manager Conflicts
**Symptoms**: APT/Snap/Flatpak conflicts during installation
**Solutions**:
- Update package lists: ` + "`" + `sudo apt update` + "`" + `
- Clean package cache: ` + "`" + `sudo apt autoremove` + "`" + `
- Fix broken packages: ` + "`" + `sudo apt --fix-broken install` + "`" + `

### Theme Issues

#### Theme Not Applied
**Symptoms**: Applications don't reflect new theme
**Solutions**:
- Restart applications after theme change
- Log out and log back in for desktop themes
- Check application-specific theme settings
- Verify theme files exist in ` + "`" + `~/.config/karei/themes/` + "`" + `

#### Font Rendering Problems
**Symptoms**: Fonts appear incorrect or missing
**Solutions**:
- Refresh font cache: ` + "`" + `fc-cache -fv` + "`" + `
- Verify font installation: ` + "`" + `fc-list | grep -i <font-name>` + "`" + `
- Restart applications using fonts
- Check terminal font settings manually

### Configuration Issues

#### Settings Not Saved
**Symptoms**: Configuration changes don't persist
**Solutions**:
- Check file permissions in ` + "`" + `~/.config/karei/` + "`" + `
- Ensure config directory exists and is writable
- Look for error messages in logs: ` + "`" + `karei logs` + "`" + `

#### Shell Configuration Problems
**Symptoms**: Shell doesn't load correctly or commands missing
**Solutions**:
- Source configuration manually: ` + "`" + `source ~/.config/fish/config.fish` + "`" + `
- Check for syntax errors in shell config files
- Reset to defaults: ` + "`" + `karei config reset shell` + "`" + `

### Performance Issues

#### Slow TUI Response
**Symptoms**: TUI interface feels sluggish
**Solutions**:
- Increase terminal scrollback buffer
- Close other resource-intensive applications
- Check system resources: ` + "`" + `htop` + "`" + ` or ` + "`" + `btop` + "`" + `

#### High Memory Usage
**Symptoms**: System becomes slow during installation
**Solutions**:
- Install applications in smaller batches
- Close web browsers and other memory-intensive apps
- Monitor memory usage: ` + "`" + `free -h` + "`" + `

## Getting Help

### Log Analysis
- View detailed logs: ` + "`" + `karei logs` + "`" + `
- Check specific component: ` + "`" + `karei logs install` + "`" + `
- Export logs for support: ` + "`" + `karei logs --export` + "`" + `

### System Information
- Check system status: ` + "`" + `karei status` + "`" + `
- Verify installation: ` + "`" + `karei verify` + "`" + `
- Show version info: ` + "`" + `karei version` + "`" + `

### Community Support

- **GitHub Issues**: Report bugs at github.com/janderssonse/karei/issues
- **Discussions**: Ask questions at github.com/janderssonse/karei/discussions
- **Documentation**: Full docs at github.com/janderssonse/karei#readme

> 📧 **Tip**: When reporting issues, include output from ` + "`" + `karei status` + "`" + ` and relevant log entries.`,
		},
		{
			Title: "CLI Reference",
			Content: `# Command Line Interface Reference

## Core Commands 💻

### Installation Commands

` + "```bash" + `
# Interactive TUI interface (recommended)
karei tui

# Install specific applications
karei install git code neovim

# Install application groups
karei install golang rustlang javalang

# Interactive app selection
karei apps

# Update all installed applications
karei update
` + "```" + `

### Theme Commands

` + "```bash" + `
# Interactive theme selection
karei theme

# Apply specific theme
karei theme tokyo-night
karei theme catppuccin
karei theme nord

# List available themes
karei theme list

# Show current theme
karei theme current
` + "```" + `

### Configuration Commands

` + "```bash" + `
# Interactive configuration
karei config

# Configure specific components
karei font jetbrains-mono
karei font-size 14

# Shell setup
karei setup

# Desktop application entries
karei desktop
` + "```" + `

### System Commands

` + "```bash" + `
# Show system status
karei status

# Verify installation integrity
karei verify

# View logs
karei logs
karei logs install
karei logs theme

# Show version information
karei version

# Uninstall applications
karei uninstall <app-name>
` + "```" + `

### Security Commands

` + "```bash" + `
# Run security checks
karei security

# Install security tools
karei security install

# Security audit
karei security audit
` + "```" + `

## Global Options

| Option | Description |
|--------|-------------|
| ` + "`" + `--verbose` + "`" + ` | Show detailed progress messages |
| ` + "`" + `--json` + "`" + ` | Output structured JSON results |
| ` + "`" + `--plain` + "`" + ` | Plain text output for scripts |
| ` + "`" + `--help` + "`" + ` | Show help for any command |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Usage error |
| 3 | Configuration error |
| 4 | Permission error |
| 5 | Not found error |
| 10-14 | System errors |
| 20-24 | Domain-specific errors |
| 64 | Completed with warnings |

## Examples

### Complete Development Setup
` + "```bash" + `
# Launch TUI for guided setup
karei tui

# Or use CLI for automated setup
karei install golang rustlang javalang
karei theme tokyo-night
karei font jetbrains-mono
karei setup
` + "```" + `

### Theme Switching Workflow
` + "```bash" + `
# Preview themes interactively
karei tui
# Navigate to "Configure Themes"

# Or apply directly
karei theme catppuccin
karei theme nord
karei theme gruvbox
` + "```" + `

### Maintenance Tasks
` + "```bash" + `
# Regular maintenance
karei update
karei verify
karei security audit

# Check system health
karei status
karei logs
` + "```" + `

## Scripting Support

Karei is designed to be script-friendly:

` + "```bash" + `
# Machine-readable output
karei status --json
karei theme list --plain

# Silent operation
karei install git --plain > /dev/null

# Error handling
if ! karei verify --plain; then
    echo "Installation verification failed"
    exit 1
fi
` + "```" + `

> 💡 **Tip**: Use ` + "`" + `karei <command> --help` + "`" + ` for detailed help on any command.`,
		},
	}

	// Create Glamour renderer with Tokyo Night style
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		// Fallback to default renderer
		renderer, _ = glamour.NewTermRenderer()
	}

	// Create viewport for scrolling
	viewPort := viewport.New(80, 20)
	viewPort.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styleConfig.Primary).
		Padding(1)

	helpModel := &Help{
		styles:         styleConfig,
		sections:       sections,
		viewport:       viewPort,
		renderer:       renderer,
		currentSection: 0,
		keyMap:         DefaultHelpKeyMap(),
	}

	// Render initial content
	helpModel.updateContent()

	return helpModel
}

// Init initializes the help model.
func (m *Help) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the Help model.
func (m *Help) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	}

	return m, nil
}

// View renders the help screen.
func (m *Help) View() string {
	if m.quitting {
		return GoodbyeMessage
	}

	var builder strings.Builder

	// Header with navigation
	header := m.renderHeader()
	builder.WriteString(header)
	builder.WriteString("\n\n")

	// Main content viewport
	builder.WriteString(m.viewport.View())
	builder.WriteString("\n\n")

	// Footer with keybindings
	footer := m.renderFooter()
	builder.WriteString(footer)

	return builder.String()
}

// handleKeyMsg processes keyboard input for the help screen.
//

func (m *Help) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Quit):
		m.quitting = true

		return m, tea.Quit
	case key.Matches(msg, m.keyMap.Back):
		return m, func() tea.Msg {
			return NavigateMsg{Screen: MenuScreen}
		}
	case key.Matches(msg, m.keyMap.Left):
		return m.handleSectionNavigation(-1)
	case key.Matches(msg, m.keyMap.Right), key.Matches(msg, m.keyMap.Tab):
		return m.handleSectionNavigation(1)
	case key.Matches(msg, m.keyMap.Home):
		m.viewport.GotoTop()

		return m, nil
	case key.Matches(msg, m.keyMap.End):
		m.viewport.GotoBottom()

		return m, nil
	default:
		var cmd tea.Cmd

		m.viewport, cmd = m.viewport.Update(msg)

		return m, cmd
	}
}

// handleSectionNavigation moves between help sections.
//

func (m *Help) handleSectionNavigation(direction int) (tea.Model, tea.Cmd) {
	newSection := m.currentSection + direction
	if newSection >= 0 && newSection < len(m.sections) {
		m.currentSection = newSection
		m.updateContent()
	}

	return m, nil
}

// handleWindowSizeMsg processes window resize messages.
//

func (m *Help) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Update viewport size
	header := m.renderHeader()
	footer := m.renderFooter()
	verticalMargins := lipgloss.Height(header) + lipgloss.Height(footer)

	m.viewport.Width = msg.Width
	m.viewport.Height = msg.Height - verticalMargins

	// Update content with new dimensions
	m.updateContent()

	return m, nil
}

// renderHeader creates the header with section navigation.
func (m *Help) renderHeader() string {
	var builder strings.Builder

	// Title
	title := m.styles.Title.Render("❓ Help & Documentation")
	builder.WriteString(title)
	builder.WriteString("\n")

	// Section tabs
	tabs := make([]string, 0, len(m.sections))

	for i, section := range m.sections {
		var style lipgloss.Style
		if i == m.currentSection {
			style = m.styles.Selected.
				Padding(0, 1).
				MarginRight(1)
		} else {
			style = m.styles.Unselected.
				Padding(0, 1).
				MarginRight(1).
				Faint(true)
		}

		tabs = append(tabs, style.Render(section.Title))
	}

	tabsLine := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	builder.WriteString(tabsLine)

	return builder.String()
}

// renderFooter creates the footer with keybindings.
func (m *Help) renderFooter() string {
	var keybindings []string

	keybindings = append(keybindings, m.styles.Keybinding("↑↓/jk", "scroll"))
	keybindings = append(keybindings, m.styles.Keybinding("←→/hl", "sections"))
	keybindings = append(keybindings, m.styles.Keybinding("tab", "next section"))
	keybindings = append(keybindings, m.styles.Keybinding("g/G", "top/bottom"))
	keybindings = append(keybindings, m.styles.Keybinding("esc", "back"))
	keybindings = append(keybindings, m.styles.Keybinding("q", "quit"))

	footer := strings.Join(keybindings, "  ")

	return m.styles.Footer.Render(footer)
}

// updateContent renders the current section content and updates the viewport.
func (m *Help) updateContent() {
	if m.currentSection >= len(m.sections) {
		return
	}

	section := m.sections[m.currentSection]

	// Render markdown content using Glamour
	rendered, err := m.renderer.Render(section.Content)
	if err != nil {
		// Fallback to plain text if rendering fails
		rendered = section.Content
	}

	// Set content in viewport
	m.viewport.SetContent(rendered)
}
