// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

//go:build integration
// +build integration

package system

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		args    []string
		verbose bool
		wantErr bool
	}{
		{
			name:    "successful echo command",
			command: "echo",
			args:    []string{"hello"},
			verbose: false,
			wantErr: false,
		},
		{
			name:    "successful echo with verbose",
			command: "echo",
			args:    []string{"hello", "world"},
			verbose: true,
			wantErr: false,
		},
		{
			name:    "nonexistent command",
			command: "/nonexistent/command",
			args:    []string{},
			verbose: false,
			wantErr: true,
		},
		{
			name:    "command with exit code 1",
			command: "sh",
			args:    []string{"-c", "exit 1"},
			verbose: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := Run(ctx, tt.verbose, tt.command, tt.args...)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunWithOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		command    string
		args       []string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "capture echo output",
			command:    "echo",
			args:       []string{"hello world"},
			wantOutput: "hello world",
			wantErr:    false,
		},
		{
			name:       "capture multiple lines",
			command:    "sh",
			args:       []string{"-c", "echo 'line1'; echo 'line2'"},
			wantOutput: "line1\nline2",
			wantErr:    false,
		},
		{
			name:       "capture error output",
			command:    "sh",
			args:       []string{"-c", "echo 'error' >&2; exit 1"},
			wantOutput: "error",
			wantErr:    true,
		},
		{
			name:       "nonexistent command",
			command:    "/nonexistent/command",
			args:       []string{},
			wantOutput: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			output, err := RunWithOutput(ctx, tt.command, tt.args...)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.wantOutput != "" {
				assert.Contains(t, strings.TrimSpace(output), strings.TrimSpace(tt.wantOutput))
			}
		})
	}
}

func TestRunSilent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "successful silent command",
			command: "echo",
			args:    []string{"silent"},
			wantErr: false,
		},
		{
			name:    "silent command with error",
			command: "sh",
			args:    []string{"-c", "exit 1"},
			wantErr: true,
		},
		{
			name:    "nonexistent command",
			command: "/nonexistent/command",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := RunSilent(ctx, tt.command, tt.args...)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunWithContext(t *testing.T) {
	t.Parallel()

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		// Start a long-running command
		done := make(chan error, 1)

		go func() {
			done <- Run(ctx, false, "sleep", "10")
		}()

		// Cancel the context after a short delay
		time.Sleep(50 * time.Millisecond)
		cancel()

		// Wait for the command to finish
		select {
		case err := <-done:
			require.Error(t, err)
			assert.Contains(t, err.Error(), "signal: killed")
		case <-time.After(2 * time.Second):
			t.Fatal("Command did not terminate after context cancellation")
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := Run(ctx, false, "sleep", "10")
		require.Error(t, err)
	})
}

func TestCommandExistence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "echo exists",
			command: "echo",
			want:    true,
		},
		{
			name:    "sh exists",
			command: "sh",
			want:    true,
		},
		{
			name:    "nonexistent command",
			command: "nonexistent_command_xyz123",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := exec.LookPath(tt.command)
			exists := err == nil
			assert.Equal(t, tt.want, exists)
		})
	}
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty command", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		err := Run(ctx, false, "")
		require.Error(t, err)
	})

	t.Run("command with many arguments", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		args := make([]string, 100)
		for i := range args {
			args[i] = fmt.Sprintf("arg%d", i)
		}

		err := Run(ctx, false, "echo", args...)
		assert.NoError(t, err)
	})

	t.Run("command with special characters", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		output, err := RunWithOutput(ctx, "echo", "hello $USER & | > < \"quoted\" 'single'")
		require.NoError(t, err)
		assert.Contains(t, output, "hello $USER")
		assert.Contains(t, output, "quoted")
	})

	t.Run("command with newlines in arguments", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		output, err := RunWithOutput(ctx, "echo", "line1\nline2\nline3")
		require.NoError(t, err)
		assert.Contains(t, output, "line1\nline2\nline3")
	})
}
