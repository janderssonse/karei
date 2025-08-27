// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janderssonse/karei/internal/tui/styles"
	"github.com/janderssonse/karei/internal/uninstall"
)

func TestProgressModel_EndToEndInstallFlow(t *testing.T) {
	// Setup isolated test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Create mixed operations for testing
	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
		{AppKey: "brave", Operation: StateInstall, AppName: "Brave Browser"},
	}

	// Create progress model
	model := NewProgressWithOperations(context.Background(), styles.New(), operations)

	if model == nil {
		t.Fatal("Progress model should not be nil")
	}

	// Note: installer and uninstaller are now initialized internally in NewProgress
	// using the hexagonal PackageInstaller architecture
	// No need to replace them for testing as they use TUI-optimized versions

	if model.uninstaller == nil {
		t.Error("Uninstaller should be initialized")
	}

	// Verify tasks were created
	if len(model.tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(model.tasks))
	}

	// Verify task details
	for taskIndex, task := range model.tasks {
		if task.Status != TaskStatusPending {
			t.Errorf("Task %d should start with pending status, got %s", taskIndex, task.Status)
		}

		if task.Operation != OperationInstall {
			t.Errorf("Task %d should be install operation, got %s", taskIndex, task.Operation)
		}

		expectedName := operations[taskIndex].AppKey
		if task.Name != expectedName {
			t.Errorf("Task %d should have name %s, got %s", taskIndex, expectedName, task.Name)
		}
	}
}

func TestProgressModel_NavigationFlow(t *testing.T) {
	// Setup isolated test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Create a simple install operation
	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
	}

	model := NewProgressWithOperations(context.Background(), styles.New(), operations)

	// Use dry-run mode for testing
	// Installer is now initialized internally with hexagonal PackageInstaller

	// Simulate completion
	model.completed = true

	// Test ESC key behavior
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}

	_, cmd := model.handleKeyInput(keyMsg)

	if cmd == nil {
		t.Error("ESC key should return navigation command when completed")

		return
	}

	// For idiomatic Bubble Tea testing, we test that the command exists
	// The framework handles command execution - we don't need to test internal mechanics
	msg := cmd()

	// Should return NavigateMsg with CompletedOperationsMsg (improved idiomatic pattern)
	navigateMsg, isNavigateMsg := msg.(NavigateMsg)
	if !isNavigateMsg {
		t.Errorf("Expected NavigateMsg, got %T", msg)

		return
	}

	// Verify navigation includes completed operations for immediate sync
	completedMsg, isCompletedMsg := navigateMsg.Data.(CompletedOperationsMsg)
	if !isCompletedMsg {
		t.Errorf("Expected CompletedOperationsMsg in Data, got %T", navigateMsg.Data)

		return
	}

	if len(completedMsg.Operations) != 1 || completedMsg.Operations[0].AppKey != "git" {
		t.Errorf("Expected CompletedOperationsMsg with git operation, got %v", completedMsg.Operations)
	}
}

func TestProgressModel_TaskExecution(t *testing.T) {
	// Setup isolated test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Create install operation
	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
	}

	model := NewProgressWithOperations(context.Background(), styles.New(), operations)

	// Use dry-run mode for testing
	// Installer is now initialized internally with hexagonal PackageInstaller

	// Test executeNextTask finds the first pending task
	cmd := model.executeNextTask()

	if cmd == nil {
		t.Error("executeNextTask should return command for pending task")
	}

	// Verify task state after execution setup
	if model.currentTask != 0 {
		t.Errorf("Current task should be 0, got %d", model.currentTask)
	}
}

func TestProgressModel_MixedOperations_InstallAndUninstall(t *testing.T) {
	// Setup isolated test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Create mixed operations
	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
		{AppKey: "firefox", Operation: StateUninstall, AppName: "Firefox"},
		{AppKey: "vscode", Operation: StateInstall, AppName: "Visual Studio Code"},
	}

	model := NewProgressWithOperations(context.Background(), styles.New(), operations)

	// Use dry-run mode for testing
	// Installer is now initialized internally with hexagonal PackageInstaller
	model.uninstaller = uninstall.NewUninstaller(false) // verbose=false for tests

	// Verify tasks were created correctly
	if len(model.tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(model.tasks))
	}

	// Check task operations
	expectedOps := []string{"install", "uninstall", "install"}
	for i, task := range model.tasks {
		if task.Operation != expectedOps[i] {
			t.Errorf("Task %d should have operation %s, got %s", i, expectedOps[i], task.Operation)
		}
	}

	// Verify all start as pending
	for i, task := range model.tasks {
		if task.Status != TaskStatusPending {
			t.Errorf("Task %d should start pending, got %s", i, task.Status)
		}
	}
}

func TestProgressModel_CompletionFlow(t *testing.T) {
	// Setup isolated test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	operations := []SelectedOperation{
		{AppKey: "git", Operation: StateInstall, AppName: "Git"},
	}

	model := NewProgressWithOperations(context.Background(), styles.New(), operations)

	// Use dry-run mode for testing
	// Installer is now initialized internally with hexagonal PackageInstaller

	// Simulate task completion
	completedMsg := CompletedMsg{
		TaskName: "git",
		Success:  true,
		Duration: time.Second * 2,
		Error:    "",
	}

	// Process the completion
	_ = model.handleCompletedTask(completedMsg)

	// Verify task was marked as completed
	if model.tasks[0].Status != TaskStatusCompleted {
		t.Errorf("Task should be completed, got %s", model.tasks[0].Status)
	}

	if model.tasks[0].Duration != time.Second*2 {
		t.Errorf("Task duration should be 2s, got %v", model.tasks[0].Duration)
	}

	// Verify overall completion
	if !model.completed {
		t.Error("Model should be marked as completed")
	}

	// Verify log entry was added
	if len(model.logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(model.logs))
	}

	if !strings.Contains(model.logs[0], "git") {
		t.Errorf("Log should contain app name, got: %s", model.logs[0])
	}

	if !strings.Contains(model.logs[0], "completed") {
		t.Errorf("Log should indicate completion, got: %s", model.logs[0])
	}
}

func TestProgressModel_ErrorHandling(t *testing.T) {
	// Setup isolated test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	operations := []SelectedOperation{
		{AppKey: "nonexistent", Operation: StateInstall, AppName: "Non-existent App"},
	}

	model := NewProgressWithOperations(context.Background(), styles.New(), operations)

	// Use dry-run mode for testing
	// Installer is now initialized internally with hexagonal PackageInstaller

	// Simulate task failure
	failedMsg := CompletedMsg{
		TaskName: "nonexistent",
		Success:  false,
		Duration: time.Second,
		Error:    "CRITICAL: App nonexistent not found in catalog",
	}

	// Process the failure
	errorModel := model.handleCompletedTask(failedMsg)

	// Verify task was marked as failed
	if model.tasks[0].Status != TaskStatusFailed {
		t.Errorf("Task should be failed, got %s", model.tasks[0].Status)
	}

	if model.tasks[0].Error != "CRITICAL: App nonexistent not found in catalog" {
		t.Errorf("Task should have error message, got: %s", model.tasks[0].Error)
	}

	// For long error messages, should return error screen
	if len(failedMsg.Error) > 10 && errorModel == nil {
		t.Error("Long error should return error screen model")
	}
}
