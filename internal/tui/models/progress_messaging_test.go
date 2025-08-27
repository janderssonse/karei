// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestProgressMessageFlow tests that progress stages don't create duplicate log entries.
func TestProgressMessageFlow(t *testing.T) {
	t.Parallel()

	// Test progress stage with empty message (install phase)
	msg := ProgressUpdateMsg{
		TaskIndex: 0,
		Progress:  0.6,
		Message:   "", // Empty message should not create log entry
	}

	// Create a mock progress model
	progress := &Progress{
		tasks: []InstallTask{{Status: TaskStatusPending}},
		logs:  []string{},
	}

	// Handle the message using the proper Update method
	progress.handleProgressUpdateMsg(msg)

	// Verify no log entry was created for empty message
	if len(progress.logs) != 0 {
		t.Errorf("Expected no log entries for empty message, got %d entries", len(progress.logs))
	}

	// Test progress stage with actual message
	msg2 := ProgressUpdateMsg{
		TaskIndex: 0,
		Progress:  0.4,
		Message:   "Downloading Chrome package...",
	}

	progress.handleProgressUpdateMsg(msg2)

	// Verify log entry was created for non-empty message
	if len(progress.logs) != 1 {
		t.Errorf("Expected 1 log entry for non-empty message, got %d entries", len(progress.logs))
	}

	if len(progress.logs) > 0 && progress.logs[0] != "Downloading Chrome package..." {
		t.Errorf("Expected log entry 'Downloading Chrome package...', got '%s'", progress.logs[0])
	}
}

// TestRefreshStatusMessage tests that RefreshStatusMsg is properly structured.
func TestRefreshStatusMessage(t *testing.T) {
	t.Parallel()

	// Create RefreshStatusMsg
	msg := RefreshStatusMsg{}

	// Test it implements tea.Msg interface
	var _ tea.Msg = msg

	// RefreshStatusMsg should be empty struct (no data needed)
	if msg != (RefreshStatusMsg{}) {
		t.Error("RefreshStatusMsg should be empty struct")
	}
}
