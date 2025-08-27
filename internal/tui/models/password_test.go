// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/styles"
)

func TestPasswordPrompt_Creation(t *testing.T) {
	t.Parallel()

	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
		{AppKey: "brave", Operation: StateInstall, AppName: "Brave Browser"},
	}

	prompt := NewPasswordPrompt(context.Background(), styles.New(), operations)

	if prompt == nil {
		t.Fatal("Password prompt should not be nil")
	}

	if len(prompt.operations) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(prompt.operations))
	}

	if !strings.Contains(prompt.message, "Installing 2 applications") {
		t.Errorf("Message should mention installing 2 applications, got: %s", prompt.message)
	}
}

func TestPasswordPrompt_MixedOperations(t *testing.T) {
	t.Parallel()

	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
		{AppKey: "firefox", Operation: StateUninstall, AppName: "Firefox"},
	}

	prompt := NewPasswordPrompt(context.Background(), styles.New(), operations)

	if !strings.Contains(prompt.message, "Installing 1 applications and uninstalling 1 applications") {
		t.Errorf("Message should mention mixed operations, got: %s", prompt.message)
	}
}

func TestPasswordPrompt_KeyHandling(t *testing.T) {
	t.Parallel()

	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
	}

	prompt := NewPasswordPrompt(context.Background(), styles.New(), operations)

	var isCorrectType bool

	// Test typing characters
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	model, _ := prompt.Update(keyMsg)

	prompt, isCorrectType = model.(*PasswordPrompt)
	if !isCorrectType {
		t.Fatal("Expected model to be *PasswordPrompt")
	}

	if prompt.password != "a" {
		t.Errorf("Expected password 'a', got '%s'", prompt.password)
	}

	// Test backspace
	keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
	model, _ = prompt.Update(keyMsg)

	prompt, isCorrectType = model.(*PasswordPrompt)
	if !isCorrectType {
		t.Fatal("Expected model to be *PasswordPrompt")
	}

	if prompt.password != "" {
		t.Errorf("Expected empty password after backspace, got '%s'", prompt.password)
	}
}

func TestPasswordPrompt_Cancellation(t *testing.T) {
	t.Parallel()

	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
	}

	prompt := NewPasswordPrompt(context.Background(), styles.New(), operations)

	// Test ESC key (cancel)
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := prompt.Update(keyMsg)

	if cmd == nil {
		t.Error("ESC should return a command")

		return
	}

	msg := cmd()

	result, ok := msg.(PasswordPromptResult)
	if !ok {
		t.Error("Should return PasswordPromptResult")

		return
	}

	if !result.Cancelled {
		t.Error("Result should indicate cancellation")
	}

	if result.Password != "" {
		t.Error("Cancelled result should have empty password")
	}
}

func TestPasswordPrompt_Confirmation(t *testing.T) {
	t.Parallel()

	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
	}

	prompt := NewPasswordPrompt(context.Background(), styles.New(), operations)

	// Type password and get updated model
	prompt = enterPasswordForTest(t, prompt, "testpass")

	// Test confirmation flow
	validationMsg := testPasswordConfirmation(t, prompt)

	// Verify results
	verifyPasswordValidation(t, validationMsg, "testpass", 1)
}

// enterPasswordForTest types a password into the prompt and returns updated model.
func enterPasswordForTest(t *testing.T, prompt *PasswordPrompt, password string) *PasswordPrompt {
	t.Helper()

	for _, char := range password {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		model, _ := prompt.Update(keyMsg)

		var isCorrectType bool

		prompt, isCorrectType = model.(*PasswordPrompt)
		if !isCorrectType {
			t.Fatal("Expected model to be *PasswordPrompt")
		}
	}

	return prompt
}

// testPasswordConfirmation tests the confirmation flow and returns validation message.
func testPasswordConfirmation(t *testing.T, prompt *PasswordPrompt) PasswordValidationMsg {
	t.Helper()

	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := prompt.Update(keyMsg)

	if cmd == nil {
		t.Fatal("Enter should return a validation command")
	}

	msg := cmd()

	validationMsg, isValid := msg.(PasswordValidationMsg)
	if !isValid {
		t.Fatal("Should return PasswordValidationMsg")
	}

	return validationMsg
}

// verifyPasswordValidation verifies the password validation results.
func verifyPasswordValidation(t *testing.T, validationMsg PasswordValidationMsg, expectedPassword string, expectedOperations int) {
	t.Helper()

	if validationMsg.Password != expectedPassword {
		t.Errorf("Expected validation password '%s', got '%s'", expectedPassword, validationMsg.Password)
	}

	if len(validationMsg.Operations) != expectedOperations {
		t.Errorf("Expected %d operation, got %d", expectedOperations, len(validationMsg.Operations))
	}
}

func TestPasswordPrompt_EmptyPasswordValidation(t *testing.T) {
	t.Parallel()

	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
	}

	prompt := NewPasswordPrompt(context.Background(), styles.New(), operations)

	// Try to confirm with empty password
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	model, cmd := prompt.Update(keyMsg)

	var isCorrectType bool

	prompt, isCorrectType = model.(*PasswordPrompt)
	if !isCorrectType {
		t.Fatal("Expected model to be *PasswordPrompt")
	}

	if cmd != nil {
		t.Error("Enter with empty password should not return a command")
	}

	if prompt.error == "" {
		t.Error("Should show error for empty password")
	}
}
