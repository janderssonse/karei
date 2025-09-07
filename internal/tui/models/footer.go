// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements TUI screen models using Bubble Tea.
package models

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// FooterAction represents a key-action pair for footer display.
type FooterAction struct {
	Key    string
	Action string
}

// RenderFooter creates a standardized footer with the given actions.
func RenderFooter(styleConfig *styles.Styles, width int, actions []FooterAction, includeHelp bool) string {
	// Styles for different parts
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styleConfig.Primary) // Keys in primary color

	bracketStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styleConfig.Primary) // Brackets also in primary color

	actionStyle := lipgloss.NewStyle().
		Foreground(styleConfig.Muted) // Actions in muted color

	// Helper function to format action
	formatAction := func(key, action string) string {
		return bracketStyle.Render("[") +
			keyStyle.Render(key) +
			bracketStyle.Render("]") +
			" " +
			actionStyle.Render(action)
	}

	// Build action strings
	actionStrings := make([]string, 0, len(actions)+1) // +1 for potential help
	for _, action := range actions {
		actionStrings = append(actionStrings, formatAction(action.Key, action.Action))
	}

	// Add help if requested
	if includeHelp {
		helpKey := bracketStyle.Render("[") +
			lipgloss.NewStyle().Bold(true).Foreground(styleConfig.Warning).Render("?") +
			bracketStyle.Render("]")
		actionStrings = append(actionStrings, helpKey+" "+actionStyle.Render("Help"))
	}

	// Join actions with spacing
	footerText := strings.Join(actionStrings, "   ")

	// Style the footer container
	return lipgloss.NewStyle().
		Padding(0, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("240")).
		Width(width).
		Render(footerText)
}
