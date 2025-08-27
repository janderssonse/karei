// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// AppDelegate implements the list.ItemDelegate interface for Application items.
type AppDelegate struct {
	styles   *styles.Styles
	selected map[string]bool
}

// NewAppDelegate creates a new delegate for Application items.
func NewAppDelegate(s *styles.Styles, selected map[string]bool) *AppDelegate {
	return &AppDelegate{
		styles:   s,
		selected: selected,
	}
}

// Height returns the height of each list item.
func (d *AppDelegate) Height() int {
	return 2
}

// Spacing returns the spacing between list items.
func (d *AppDelegate) Spacing() int {
	return 0 // No spacing to maximize items shown
}

// Update handles item-specific updates.
func (d *AppDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

// Render renders a list item.
func (d *AppDelegate) Render(writer io.Writer, listModel list.Model, index int, item list.Item) {
	app, ok := item.(Application)
	if !ok {
		return
	}

	// Get item state
	isSelected := index == listModel.Index()
	isToggled := d.selected[app.Name]

	// Combined status and selection indicator using circles only
	var indicator string

	switch {
	case app.Installed && isToggled:
		indicator = "⬤" // Installed + selected
	case app.Installed && !isToggled:
		indicator = "●" // Installed + not selected
	case !app.Installed && isToggled:
		indicator = "◉" // Not installed + selected for installation
	default:
		indicator = "○" // Not installed + not selected
	}

	// Clean main line: [indicator] [icon] [name]
	mainLine := fmt.Sprintf("%s %s %-12s", indicator, app.Icon, app.Name)

	// Description line with proper indentation
	descLine := fmt.Sprintf("   %s • %s • %s", app.Description, app.Size, app.Source)

	// Apply styling based on state
	mainStyle, descStyle := d.getItemStyles(isSelected, isToggled)

	// Render both lines
	styledMain := mainStyle.Render(mainLine)
	styledDesc := descStyle.Render(descLine)

	// Output with proper line breaks
	_, _ = fmt.Fprintf(writer, "%s\n%s", styledMain, styledDesc)
}

// getItemStyles returns the appropriate main and description styles based on item state.
func (d *AppDelegate) getItemStyles(isSelected, isToggled bool) (lipgloss.Style, lipgloss.Style) {
	var mainStyle, descStyle lipgloss.Style

	if isSelected {
		mainStyle = d.styles.Selected.Bold(true)
		descStyle = lipgloss.NewStyle().
			Foreground(d.styles.Primary).
			Faint(false)
	} else {
		mainStyle = d.styles.Unselected
		descStyle = lipgloss.NewStyle().
			Foreground(d.styles.Muted).
			Faint(true)
	}

	// Special styling for toggled items
	if isToggled {
		mainStyle = mainStyle.Foreground(d.styles.Success)
		descStyle = descStyle.Foreground(d.styles.Success)
	}

	return mainStyle, descStyle
}
