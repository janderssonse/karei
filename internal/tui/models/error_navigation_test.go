// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/styles"
)

func TestErrorScreen_Navigation(t *testing.T) {
	t.Parallel()

	errorDetails := CreateInstallationError("brave", "Installation failed", "Permission denied")
	errorScreen := NewErrorScreen(styles.New(), errorDetails)

	// Test ESC key navigation
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := errorScreen.Update(keyMsg)

	if cmd == nil {
		t.Error("ESC key should return navigation command")

		return
	}

	msg := cmd()

	navMsg, ok := msg.(NavigateMsg)
	if !ok {
		t.Error("Should return NavigateMsg")

		return
	}

	if navMsg.Screen != AppsScreen {
		t.Errorf("ESC should navigate to AppsScreen, got %d", navMsg.Screen)
	}
}

func TestErrorScreen_QuitHandling(t *testing.T) {
	t.Parallel()

	errorDetails := CreateInstallationError("brave", "Installation failed", "Permission denied")
	errorScreen := NewErrorScreen(styles.New(), errorDetails)

	// Test quit key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	model, cmd := errorScreen.Update(keyMsg)

	var ok bool

	errorScreen, ok = model.(*ErrorScreen)
	if !ok {
		t.Fatal("Expected model to be *ErrorScreen")
	}

	if !errorScreen.quitting {
		t.Error("Should set quitting flag")
	}

	if cmd == nil {
		t.Error("Should return quit command")
	}
}

func TestErrorScreen_RetryHandling(t *testing.T) {
	t.Parallel()

	// Create recoverable error
	errorDetails := CreatePermissionError("Permission denied", "Sudo required")
	errorScreen := NewErrorScreen(styles.New(), errorDetails)

	// Test retry key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := errorScreen.Update(keyMsg)

	if cmd == nil {
		t.Error("Retry key should return command for recoverable error")
	}
}

func TestErrorScreen_ErrorTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		errorFn  func() ErrorDetails
		expected string
	}{
		{
			name:     "Installation Error",
			errorFn:  func() ErrorDetails { return CreateInstallationError("brave", "Failed", "Details") },
			expected: "Installation Error",
		},
		{
			name:     "Permission Error",
			errorFn:  func() ErrorDetails { return CreatePermissionError("Access denied", "Details") },
			expected: "Permission Error",
		},
		{
			name:     "Network Error",
			errorFn:  func() ErrorDetails { return CreateNetworkError("Connection failed", "Details") },
			expected: "Network Error",
		},
		{
			name:     "Configuration Error",
			errorFn:  func() ErrorDetails { return CreateConfigurationError("Config invalid", "Details") },
			expected: "Configuration Error",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			errorDetails := testCase.errorFn()
			actual := errorDetails.GetErrorType()

			if actual != testCase.expected {
				t.Errorf("Expected error type '%s', got '%s'", testCase.expected, actual)
			}
		})
	}
}
