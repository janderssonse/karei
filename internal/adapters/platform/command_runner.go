// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package platform provides shared command execution functionality.
package platform

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/janderssonse/karei/internal/network"
)

// CommandRunner implements the CommandRunner port for real system commands.
type CommandRunner struct {
	verbose bool
	dryRun  bool
	tuiMode bool // When true, suppress direct terminal output for TUI compatibility
}

// NewCommandRunner creates a new command runner.
func NewCommandRunner(verbose, dryRun bool) *CommandRunner {
	return &CommandRunner{
		verbose: verbose,
		dryRun:  dryRun,
		tuiMode: false, // Default to CLI mode
	}
}

// NewTUICommandRunner creates a new command runner optimized for TUI mode.
func NewTUICommandRunner(verbose, dryRun bool) *CommandRunner {
	return &CommandRunner{
		verbose: verbose,
		dryRun:  dryRun,
		tuiMode: true, // Enable TUI mode - suppress direct terminal output
	}
}

// Execute runs a command and returns the result.
func (r *CommandRunner) Execute(ctx context.Context, name string, args ...string) error {
	if r.verbose && !r.tuiMode {
		fmt.Printf("Executing: %s %s\n", name, strings.Join(args, " "))
	}

	if r.dryRun {
		if !r.tuiMode {
			fmt.Printf("DRY RUN: %s %s\n", name, strings.Join(args, " "))
		}

		return nil
	}

	cmd := exec.CommandContext(ctx, name, args...)

	// Propagate proxy environment variables
	cmd.Env = append(os.Environ(), network.GetProxyEnv()...)

	if r.tuiMode {
		return r.executeTUIMode(cmd)
	}

	return r.executeCLIMode(cmd)
}

// ExecuteWithOutput runs a command and returns the output.
func (r *CommandRunner) ExecuteWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	if r.verbose && !r.tuiMode {
		fmt.Printf("Executing (with output): %s %s\n", name, strings.Join(args, " "))
	}

	if r.dryRun {
		if !r.tuiMode {
			fmt.Printf("DRY RUN (with output): %s %s\n", name, strings.Join(args, " "))
		}

		return "", nil
	}

	cmd := exec.CommandContext(ctx, name, args...)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// ExecuteSudo runs a command with sudo privileges.
func (r *CommandRunner) ExecuteSudo(ctx context.Context, name string, args ...string) error {
	if r.verbose && !r.tuiMode {
		fmt.Printf("Executing with sudo: %s %s\n", name, strings.Join(args, " "))
	}

	if r.dryRun {
		if !r.tuiMode {
			fmt.Printf("DRY RUN (sudo): %s %s\n", name, strings.Join(args, " "))
		}

		return nil
	}

	// Prepend sudo to the command
	allArgs := append([]string{name}, args...)
	// #nosec G204 - This is intentional command execution with validated input
	cmd := exec.CommandContext(ctx, "sudo", allArgs...)

	// Propagate proxy environment variables to sudo command
	cmd.Env = append(os.Environ(), network.GetProxyEnv()...)

	if r.tuiMode {
		return r.executeTUIMode(cmd)
	}

	return r.executeCLIMode(cmd)
}

// CommandExists checks if a command is available on the system.
func (r *CommandRunner) CommandExists(name string) bool {
	_, err := exec.LookPath(name)

	return err == nil
}

// Removed GetProxyEnv - use network.GetProxyEnv() instead for consistency

// executeTUIMode handles command execution in TUI mode with output capture.
func (r *CommandRunner) executeTUIMode(cmd *exec.Cmd) error {
	// TUI mode: Capture all output to prevent terminal interference
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Read both stdout and stderr concurrently
	stdoutBytes, _ := io.ReadAll(stdout)
	stderrBytes, _ := io.ReadAll(stderr)

	// Wait for command completion
	if err := cmd.Wait(); err != nil {
		stderrOutput := strings.TrimSpace(string(stderrBytes))
		if stderrOutput != "" {
			return fmt.Errorf("command failed: %w (stderr: %s)", err, stderrOutput)
		}

		return fmt.Errorf("command failed: %w", err)
	}

	// In TUI mode, output is captured but not displayed
	// TUI framework will handle progress display
	_ = stdoutBytes // Prevent unused variable warning

	return nil
}

// executeCLIMode handles command execution in CLI mode with normal terminal output.
func (r *CommandRunner) executeCLIMode(cmd *exec.Cmd) error {
	// CLI mode: Allow normal terminal output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// MockCommandRunner implements the CommandRunner port for testing.
type MockCommandRunner struct {
	commands map[string]string // command -> expected output
	verbose  bool
}

// NewMockCommandRunner creates a new mock command runner for testing.
func NewMockCommandRunner(verbose bool) *MockCommandRunner {
	return &MockCommandRunner{
		commands: make(map[string]string),
		verbose:  verbose,
	}
}

// SetMockOutput sets the expected output for a command.
func (r *MockCommandRunner) SetMockOutput(command, output string) {
	r.commands[command] = output
}

// Execute runs a mock command.
func (r *MockCommandRunner) Execute(_ context.Context, name string, args ...string) error {
	fullCommand := name + " " + strings.Join(args, " ")
	if r.verbose {
		fmt.Printf("MOCK: Executing %s\n", fullCommand)
	}

	// Always succeed in mock mode
	return nil
}

// ExecuteWithOutput runs a mock command and returns preset output.
func (r *MockCommandRunner) ExecuteWithOutput(_ context.Context, name string, args ...string) (string, error) {
	fullCommand := name + " " + strings.Join(args, " ")
	if r.verbose {
		fmt.Printf("MOCK: Executing %s\n", fullCommand)
	}

	if output, exists := r.commands[fullCommand]; exists {
		return output, nil
	}

	// Return empty string by default
	return "", nil
}

// ExecuteSudo runs a mock sudo command.
func (r *MockCommandRunner) ExecuteSudo(_ context.Context, name string, args ...string) error {
	fullCommand := "sudo " + name + " " + strings.Join(args, " ")
	if r.verbose {
		fmt.Printf("MOCK: Executing %s\n", fullCommand)
	}

	// Always succeed in mock mode
	return nil
}

// CommandExists always returns true in mock mode.
func (r *MockCommandRunner) CommandExists(_ string) bool {
	return true
}
