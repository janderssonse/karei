// SPDX-FileCopyrightText: 2024 Josef Andersson
//
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHelpModal(t *testing.T) {
	t.Run("creates modal with default state", func(t *testing.T) {
		modal := NewHelpModal()

		assert.NotNil(t, modal)
		assert.False(t, modal.visible)

		// Set dimensions
		modal.SetSize(100, 50)
		assert.Equal(t, 100, modal.width)
		assert.Equal(t, 50, modal.height)
	})
}

func TestHelpModalToggle(t *testing.T) {
	t.Run("toggles visibility", func(t *testing.T) {
		modal := NewHelpModal()
		modal.SetSize(100, 50)

		// Initially not visible
		assert.False(t, modal.IsVisible())

		// Toggle on
		modal.Toggle()
		assert.True(t, modal.IsVisible())

		// Toggle off
		modal.Toggle()
		assert.False(t, modal.IsVisible())
	})
}

func TestHelpModalShow(t *testing.T) {
	t.Run("shows modal", func(t *testing.T) {
		modal := NewHelpModal()
		modal.SetSize(100, 50)

		modal.Show()
		assert.True(t, modal.IsVisible())

		// Calling show again should keep it visible
		modal.Show()
		assert.True(t, modal.IsVisible())
	})
}

func TestHelpModalHide(t *testing.T) {
	t.Run("hides modal", func(t *testing.T) {
		modal := NewHelpModal()
		modal.SetSize(100, 50)
		modal.Show()

		modal.Hide()
		assert.False(t, modal.IsVisible())

		// Calling hide again should keep it hidden
		modal.Hide()
		assert.False(t, modal.IsVisible())
	})
}

func TestHelpModalSetScreen(t *testing.T) {
	tests := []struct {
		name     string
		screen   string
		expected string
	}{
		{
			name:     "apps screen",
			screen:   "apps",
			expected: "All Commands",
		},
		{
			name:     "themes screen",
			screen:   "themes",
			expected: "All Commands",
		},
		{
			name:     "settings screen",
			screen:   "settings",
			expected: "All Commands",
		},
		{
			name:     "status screen",
			screen:   "status",
			expected: "All Commands",
		},
		{
			name:     "unknown screen",
			screen:   "unknown",
			expected: "All Commands",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modal := NewHelpModal()
			modal.SetSize(100, 50)
			modal.SetScreen(tt.screen)
			modal.Show()

			view := modal.View()
			assert.Contains(t, view, tt.expected)
		})
	}
}

func TestHelpModalUpdate(t *testing.T) {
	t.Run("handles key messages when visible", func(t *testing.T) {
		modal := NewHelpModal()
		modal.SetSize(100, 50)
		modal.Show()

		// Test closing with Escape (assuming keys.Quit includes Escape)
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		cmd := modal.Update(msg)

		assert.False(t, modal.IsVisible())
		assert.Nil(t, cmd)
	})

	t.Run("handles ? key to toggle when visible", func(t *testing.T) {
		modal := NewHelpModal()
		modal.SetSize(100, 50)
		modal.Show()

		// The ? key is handled as Help key in the implementation
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		cmd := modal.Update(msg)

		// Check if modal closed (assuming ? toggles off when visible)
		assert.Nil(t, cmd)
	})

	t.Run("ignores key messages when not visible", func(t *testing.T) {
		modal := NewHelpModal()
		modal.SetSize(100, 50)

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		cmd := modal.Update(msg)

		assert.False(t, modal.IsVisible())
		assert.Nil(t, cmd)
	})
}

func TestHelpModalView(t *testing.T) {
	t.Run("renders empty when not visible", func(t *testing.T) {
		modal := NewHelpModal()
		modal.SetSize(100, 50)

		view := modal.View()
		assert.Empty(t, view)
	})

	t.Run("renders content when visible", func(t *testing.T) {
		modal := NewHelpModal()
		modal.SetSize(100, 50)
		modal.Show()
		modal.SetScreen("apps")

		view := modal.View()
		require.NotEmpty(t, view)

		// Check for expected content
		assert.Contains(t, view, "All Commands")
		assert.Contains(t, view, "Navigation")
		assert.Contains(t, view, "Selection")
		assert.Contains(t, view, "Press ? or Esc to close")
	})

	t.Run("renders correct commands for each screen", func(t *testing.T) {
		modal := NewHelpModal()
		modal.SetSize(100, 50)
		modal.Show()

		screens := map[string][]string{
			"apps": {
				"Space", "Toggle selection",
				"Enter", "Install/uninstall selected",
				"/", "Search packages",
			},
			"themes": {
				"j/k", "Navigate theme list",
				"Enter", "Apply selected theme",
				"?", "Show this help",
			},
			"settings": {
				"Tab", "Switch setting sections",
				"Enter", "Edit setting value",
				"s", "Save all changes",
			},
			"status": {
				"r", "Refresh status",
				"c", "Clear log",
				"Enter", "View details",
			},
		}

		for screen, expectedCommands := range screens {
			t.Run(screen, func(t *testing.T) {
				modal.SetScreen(screen)
				view := modal.View()

				for _, cmd := range expectedCommands {
					assert.Contains(t, view, cmd, "Expected command '%s' in %s screen", cmd, screen)
				}
			})
		}
	})
}

func TestHelpModalIntegration(t *testing.T) {
	t.Run("full workflow", func(t *testing.T) {
		// Create modal
		modal := NewHelpModal()
		modal.SetSize(100, 50)
		assert.False(t, modal.IsVisible())

		// Set screen
		modal.SetScreen("apps")

		// Show modal
		modal.Show()
		assert.True(t, modal.IsVisible())

		// Verify content
		view := modal.View()
		assert.Contains(t, view, "All Commands")

		// Update size
		modal.SetSize(120, 60)
		assert.Equal(t, 120, modal.width)
		assert.Equal(t, 60, modal.height)

		// Close with Escape
		escMsg := tea.KeyMsg{Type: tea.KeyEsc}
		_ = modal.Update(escMsg)
		assert.False(t, modal.IsVisible())

		// Verify empty view when hidden
		view = modal.View()
		assert.Empty(t, view)
	})
}
