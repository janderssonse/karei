// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package uninstall

import (
	"context"
	"fmt"
)

// MockCommandExecutor records commands for testing without execution.
type MockCommandExecutor struct {
	Commands []string
	Results  map[string]error
}

// Run records the command and returns a mocked result.
func (m *MockCommandExecutor) Run(_ context.Context, _ bool, name string, args ...string) error {
	fullCmd := fmt.Sprintf("%s %v", name, args)
	m.Commands = append(m.Commands, fullCmd)

	if err, ok := m.Results[fullCmd]; ok {
		return err
	}

	return nil
}

// RunWithPassword records the command with sudo prefix and returns a mocked result.
func (m *MockCommandExecutor) RunWithPassword(_ context.Context, _ bool, _ string, args ...string) error {
	fullCmd := fmt.Sprintf("sudo %v", args)
	m.Commands = append(m.Commands, fullCmd)

	if err, ok := m.Results[fullCmd]; ok {
		return err
	}

	return nil
}

// NewTestUninstaller creates an Uninstaller with a mock executor for testing.
func NewTestUninstaller(verbose bool) (*Uninstaller, *MockCommandExecutor) {
	mock := &MockCommandExecutor{
		Commands: []string{},
		Results:  make(map[string]error),
	}

	return &Uninstaller{
		verbose:  verbose,
		executor: mock,
	}, mock
}
