// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package styles defines consistent visual styling for TUI components.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles contains all the styles used in the TUI.
type Styles struct {
	// Color palette
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	Info      lipgloss.Color
	Muted     lipgloss.Color

	// Component styles
	Header     lipgloss.Style
	Footer     lipgloss.Style
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Card       lipgloss.Style
	Button     lipgloss.Style
	Selected   lipgloss.Style
	Unselected lipgloss.Style
	Border     lipgloss.Style

	// Text styles (cached for performance)
	MutedText   lipgloss.Style
	PrimaryText lipgloss.Style
	SuccessText lipgloss.Style
	ErrorText   lipgloss.Style
	WarningText lipgloss.Style

	// Layout styles
	Container lipgloss.Style
	Content   lipgloss.Style
	Sidebar   lipgloss.Style
}

// New creates a new Styles instance with default Tokyo Night theme.
func New() *Styles {
	// Tokyo Night color palette
	primary := lipgloss.Color("#7aa2f7")    // Blue
	secondary := lipgloss.Color("#bb9af7")  // Purple
	success := lipgloss.Color("#9ece6a")    // Green
	warning := lipgloss.Color("#e0af68")    // Yellow
	errorColor := lipgloss.Color("#f7768e") // Red
	info := lipgloss.Color("#7dcfff")       // Cyan
	muted := lipgloss.Color("#565f89")      // Gray

	background := lipgloss.Color("#1a1b26") // Dark background
	foreground := lipgloss.Color("#c0caf5") // Light foreground

	return &Styles{
		Primary:   primary,
		Secondary: secondary,
		Success:   success,
		Warning:   warning,
		Error:     errorColor,
		Info:      info,
		Muted:     muted,

		Header: lipgloss.NewStyle().
			Background(primary).
			Foreground(background).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1),

		Footer: lipgloss.NewStyle().
			Background(muted).
			Foreground(foreground).
			Padding(0, 1).
			MarginTop(1),

		Title: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(secondary).
			Italic(true),

		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(muted).
			Padding(1, 2).
			MarginBottom(1),

		Button: lipgloss.NewStyle().
			Background(primary).
			Foreground(background).
			Bold(true).
			Padding(0, 2).
			MarginRight(1),

		Selected: lipgloss.NewStyle().
			Background(primary).
			Foreground(background).
			Padding(0, 1),

		Unselected: lipgloss.NewStyle().
			Foreground(foreground).
			Padding(0, 1),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primary),

		// Cached text styles
		MutedText: lipgloss.NewStyle().
			Foreground(muted),

		PrimaryText: lipgloss.NewStyle().
			Foreground(primary),

		SuccessText: lipgloss.NewStyle().
			Foreground(success),

		ErrorText: lipgloss.NewStyle().
			Foreground(errorColor),

		WarningText: lipgloss.NewStyle().
			Foreground(warning),

		Container: lipgloss.NewStyle().
			Padding(1, 2),

		Content: lipgloss.NewStyle().
			Padding(0, 1),

		Sidebar: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(muted).
			Padding(1).
			Width(25),
	}
}

// Logo returns the styled Karei ASCII logo.
func (s *Styles) Logo() string {
	logo := `
  ██╗  ██╗ █████╗ ██████╗ ███████╗██╗
  ██║ ██╔╝██╔══██╗██╔══██╗██╔════╝██║
  █████╔╝ ███████║██████╔╝█████╗  ██║
  ██╔═██╗ ██╔══██║██╔══██╗██╔══╝  ██║
  ██║  ██╗██║  ██║██║  ██║███████╗██║
  ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚═╝`

	return s.Title.Render(logo)
}

// StatusIcon returns styled status icons.
func (s *Styles) StatusIcon(status string) string {
	style := s.Unselected

	var icon string

	switch status {
	case "success", "completed", "installed":
		style = lipgloss.NewStyle().Foreground(s.Success)
		icon = "✓"
	case "error", "failed":
		style = lipgloss.NewStyle().Foreground(s.Error)
		icon = "✗"
	case "warning":
		style = lipgloss.NewStyle().Foreground(s.Warning)
		icon = "!"
	case "info":
		style = lipgloss.NewStyle().Foreground(s.Info)
		icon = "i"
	case "progress", "installing", "uninstalling", "downloading":
		style = lipgloss.NewStyle().Foreground(s.Primary)
		icon = "⚬"
	case "pending":
		style = lipgloss.NewStyle().Foreground(s.Muted)
		icon = "○"
	default:
		icon = "•"
	}

	return style.Render(icon)
}

// ProgressBar creates a styled progress bar.
func (s *Styles) ProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))

	bar := ""

	for i := range width {
		if i < filled {
			bar += "█"
		} else {
			bar += "▓"
		}
	}

	return lipgloss.NewStyle().
		Foreground(s.Primary).
		Render(bar)
}

// ContextualProgressBar creates a styled progress bar with contextual colors.
func (s *Styles) ContextualProgressBar(current, total int, width int, hasErrors, isCompleted bool) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))

	bar := ""

	for i := range width {
		if i < filled {
			bar += "█"
		} else {
			bar += "▓"
		}
	}

	// Choose color based on status
	var color lipgloss.Color

	switch {
	case hasErrors:
		color = s.Error // Red for failures
	case isCompleted:
		color = s.Success // Green for completed
	default:
		color = s.Warning // Yellow for in-progress
	}

	return lipgloss.NewStyle().
		Foreground(color).
		Render(bar)
}

// Keybinding returns styled keybinding text.
func (s *Styles) Keybinding(key, desc string) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(s.Primary).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(s.Muted)

	return keyStyle.Render("["+key+"]") + " " + descStyle.Render(desc)
}
