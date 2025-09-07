// SPDX-FileCopyrightText: 2024 Josef Andersson
//
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpModal represents a modal overlay showing all available commands.
type HelpModal struct {
	visible  bool
	screen   string // Current screen context
	commands []HelpModalSection
	width    int
	height   int
}

// HelpModalSection groups related commands.
type HelpModalSection struct {
	Title    string
	Commands []HelpModalCommand
}

// HelpModalCommand represents a single keyboard command.
type HelpModalCommand struct {
	Keys        string
	Description string
}

// NewHelpModal creates a new help modal.
func NewHelpModal() *HelpModal {
	return &HelpModal{
		visible: false,
	}
}

// SetScreen updates the help content based on current screen.
func (h *HelpModal) SetScreen(screen string) {
	h.screen = screen
	h.commands = h.getCommandsForScreen(screen)
}

// Toggle shows/hides the modal.
func (h *HelpModal) Toggle() {
	h.visible = !h.visible
}

// Show displays the modal.
func (h *HelpModal) Show() {
	h.visible = true
}

// Hide closes the modal.
func (h *HelpModal) Hide() {
	h.visible = false
}

// IsVisible returns whether the modal is shown.
func (h *HelpModal) IsVisible() bool {
	return h.visible
}

// SetSize updates the modal dimensions.
func (h *HelpModal) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// Update handles key events for the modal.
func (h *HelpModal) Update(msg tea.Msg) tea.Cmd {
	if !h.visible {
		return nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		keys := getHelpKeys()
		if key.Matches(msg, keys.Help) || key.Matches(msg, keys.Quit) {
			h.Hide()
			return nil
		}
	}

	return nil
}

// View renders the help modal.
func (h *HelpModal) View() string {
	if !h.visible {
		return ""
	}

	// Modal styling
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		MaxWidth(60)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Bold(true).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Width(15)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	// Build content
	var content strings.Builder
	content.WriteString(titleStyle.Render("All Commands"))
	content.WriteString("\n")

	for i, section := range h.commands {
		if i > 0 {
			content.WriteString("\n")
		}

		content.WriteString(sectionStyle.Render(section.Title))
		content.WriteString("\n")

		for _, cmd := range section.Commands {
			line := fmt.Sprintf("%s %s\n",
				keyStyle.Render(cmd.Keys),
				descStyle.Render(cmd.Description))
			content.WriteString(line)
		}
	}

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("Press ? or Esc to close"))

	// Just return the modal content - centering will be handled by RenderModalOverlay
	return modalStyle.Render(content.String())
}

// getCommandsForScreen returns appropriate help content for each screen.
func (h *HelpModal) getCommandsForScreen(screen string) []HelpModalSection {
	switch screen {
	case "apps", "packages":
		return []HelpModalSection{
			{
				Title: "Navigation",
				Commands: []HelpModalCommand{
					{"j/k or ↑↓", "Navigate items"},
					{"h/l or ←→", "Switch categories"},
					{"g/G", "Go to top/bottom"},
					{"PgUp/PgDn", "Page up/down"},
				},
			},
			{
				Title: "Selection",
				Commands: []HelpModalCommand{
					{"Space", "Toggle selection"},
					{"a", "Select all in category"},
					{"A", "Deselect all"},
				},
			},
			{
				Title: "Actions",
				Commands: []HelpModalCommand{
					{"Enter", "Install selected packages"},
					{"/", "Search packages"},
					{"p", "Preview package details"},
					{"v", "Show package variants"},
					{"r", "Refresh package list"},
				},
			},
			{
				Title: "General",
				Commands: []HelpModalCommand{
					{"Tab", "Switch sections"},
					{"Esc", "Go back"},
					{"q", "Quit application"},
				},
			},
		}

	case "themes":
		return []HelpModalSection{
			{
				Title: "Navigation",
				Commands: []HelpModalCommand{
					{"←→ or h/l", "Browse themes"},
					{"j/k or ↑↓", "Navigate theme list"},
				},
			},
			{
				Title: "Actions",
				Commands: []HelpModalCommand{
					{"Enter", "Apply selected theme"},
					{"p or Space", "Preview theme"},
					{"r", "Reset to default"},
				},
			},
			{
				Title: "General",
				Commands: []HelpModalCommand{
					{"Tab", "Switch sections"},
					{"Esc", "Go back"},
					{"q", "Quit application"},
				},
			},
		}

	case "settings", "preferences":
		return []HelpModalSection{
			{
				Title: "Navigation",
				Commands: []HelpModalCommand{
					{"Tab", "Switch setting sections"},
					{"j/k or ↑↓", "Navigate settings"},
				},
			},
			{
				Title: "Actions",
				Commands: []HelpModalCommand{
					{"Enter", "Edit setting value"},
					{"Space", "Toggle boolean setting"},
					{"s", "Save all changes"},
					{"r", "Reset to defaults"},
					{"d", "Discard changes"},
				},
			},
			{
				Title: "General",
				Commands: []HelpModalCommand{
					{"Esc", "Go back"},
					{"q", "Quit application"},
				},
			},
		}

	case "status":
		return []HelpModalSection{
			{
				Title: "Navigation",
				Commands: []HelpModalCommand{
					{"j/k or ↑↓", "Scroll log"},
					{"g/G", "Go to top/bottom"},
				},
			},
			{
				Title: "Actions",
				Commands: []HelpModalCommand{
					{"r", "Refresh status"},
					{"c", "Clear log"},
					{"Enter", "View details"},
				},
			},
			{
				Title: "General",
				Commands: []HelpModalCommand{
					{"Esc", "Go back"},
					{"q", "Quit application"},
				},
			},
		}

	default: // menu or unknown
		return []HelpModalSection{
			{
				Title: "Navigation",
				Commands: []HelpModalCommand{
					{"j/k or ↑↓", "Navigate menu"},
					{"Enter", "Select option"},
				},
			},
			{
				Title: "General",
				Commands: []HelpModalCommand{
					{"q", "Quit application"},
				},
			},
		}
	}
}

// getHelpKeys returns common key bindings for the help modal.
func getHelpKeys() struct {
	Help key.Binding
	Quit key.Binding
} {
	return struct {
		Help key.Binding
		Quit key.Binding
	}{
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close help"),
		),
	}
}
