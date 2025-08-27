// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Run executes a command with optional verbose output
// Consolidates logic repeated 10+ times across files.
func Run(ctx context.Context, verbose bool, name string, args ...string) error {
	if verbose {
		fmt.Printf("Running: %s %s\n", name, strings.Join(args, " "))
	}

	cmd := exec.CommandContext(ctx, name, args...)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	// Enable interactive sudo password prompts
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// RunWithOutput executes command and captures output.
func RunWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()

	return string(output), err
}

// RunSilent executes command with no output.
func RunSilent(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)

	return cmd.Run()
}

// RunWithEnv executes a command with environment variables.
func RunWithEnv(ctx context.Context, verbose bool, env map[string]string, name string, args ...string) error {
	if verbose {
		fmt.Printf("Running: %s %s\n", name, strings.Join(args, " "))

		for key, value := range env {
			fmt.Printf("  %s=%s\n", key, value)
		}
	}

	cmd := exec.CommandContext(ctx, name, args...)

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	// Enable interactive sudo password prompts
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// RunWithPassword executes a sudo command with password provided via stdin.
// This avoids interactive password prompts by using sudo's -S flag.
func RunWithPassword(ctx context.Context, verbose bool, password string, args ...string) error {
	if verbose {
		// Don't print password in verbose mode!
		fmt.Printf("Running: sudo %s\n", strings.Join(args, " "))
	}

	// Build command: sudo -S [args...]
	sudoArgs := append([]string{"-S"}, args...)
	//nolint:gosec // G204: Intentional subprocess execution with variables for sudo command automation
	cmd := exec.CommandContext(ctx, "sudo", sudoArgs...)

	// Provide password via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Write password to stdin
	if _, err := fmt.Fprintf(stdin, "%s\n", password); err != nil {
		return fmt.Errorf("failed to write password: %w", err)
	}

	_ = stdin.Close()

	// Wait for command to complete
	return cmd.Wait()
}

// CommandExists checks if command is available.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)

	return err == nil
}
