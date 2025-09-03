// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/janderssonse/karei/internal/adapters/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunner_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     string
		args    []string
		wantErr bool
	}{
		{
			name:    "successful echo command",
			cmd:     "echo",
			args:    []string{"hello"},
			wantErr: false,
		},
		{
			name:    "non-existent command",
			cmd:     "nonexistent_command_xyz",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "command with exit code 1",
			cmd:     "sh",
			args:    []string{"-c", "exit 1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := platform.NewCommandRunner(false, false)
			err := cr.Execute(context.Background(), tt.cmd, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCommandRunner_ExecuteWithOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cmd        string
		args       []string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "capture echo output",
			cmd:        "echo",
			args:       []string{"test output"},
			wantOutput: "test output",
			wantErr:    false,
		},
		{
			name:       "capture multiline output",
			cmd:        "sh",
			args:       []string{"-c", "echo line1; echo line2"},
			wantOutput: "line1\nline2",
			wantErr:    false,
		},
		{
			name:       "command not found",
			cmd:        "nonexistent_command_xyz",
			args:       []string{},
			wantOutput: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := platform.NewCommandRunner(false, false)
			output, err := cr.ExecuteWithOutput(context.Background(), tt.cmd, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantOutput, strings.TrimSpace(output))
			}
		})
	}
}

func TestCommandRunner_ContextCancellation(t *testing.T) {
	t.Parallel()

	cr := platform.NewCommandRunner(false, false)

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start a long-running command in a goroutine
	errChan := make(chan error, 1)

	go func() {
		// Sleep for 10 seconds - should be cancelled before completion
		errChan <- cr.Execute(ctx, "sleep", "10")
	}()

	// Give the command time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for the command to return
	select {
	case err := <-errChan:
		require.Error(t, err, "cancelled command should return error")
		// The error can be either "context canceled" or "signal: killed"
		errorMsg := err.Error()
		assert.True(t, strings.Contains(errorMsg, "context canceled") ||
			strings.Contains(errorMsg, "signal: killed"),
			"error should indicate cancellation, got: %s", errorMsg)
	case <-time.After(2 * time.Second):
		t.Fatal("command did not respond to context cancellation")
	}
}

func TestCommandRunner_DryRun(t *testing.T) {
	t.Parallel()

	// Create a command runner in dry-run mode
	cr := platform.NewCommandRunner(false, true)

	// In dry-run mode, commands should not actually execute
	err := cr.Execute(context.Background(), "sh", "-c", "echo 'test' > /tmp/karei_dryrun_test.txt")
	require.NoError(t, err, "dry-run should not return error")

	// Verify the file was not created by checking with a real command runner
	realCR := platform.NewCommandRunner(false, false)
	output, err := realCR.ExecuteWithOutput(context.Background(), "ls", "/tmp/karei_dryrun_test.txt")
	// File should not exist
	require.Error(t, err, "file should not exist after dry-run")
	assert.NotContains(t, output, "karei_dryrun_test.txt")
}

func TestCommandRunner_CommandExists(t *testing.T) {
	t.Parallel()

	cr := platform.NewCommandRunner(false, false)

	tests := []struct {
		name   string
		cmd    string
		expect bool
	}{
		{"echo exists", "echo", true},
		{"sh exists", "sh", true},
		{"nonexistent command", "nonexistent_xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, cr.CommandExists(tt.cmd))
		})
	}
}
