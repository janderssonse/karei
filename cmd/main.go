// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package main provides the CLI entry point for Karei.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"github.com/janderssonse/karei/internal/cli"
)

// Exit codes following Unix conventions.
const (
	ExitSuccess         = 0  // Command completed successfully
	ExitGeneralError    = 1  // General errors
	ExitUsageError      = 2  // Invalid arguments/usage
	ExitConfigError     = 3  // Configuration issues
	ExitPermissionError = 4  // Permission denied, need sudo
	ExitNotFoundError   = 5  // Theme/font/app not found
	ExitDependencyError = 10 // Missing dependencies (gum, git, curl)
	ExitNetworkError    = 11 // Download/network failures
	ExitSystemError     = 12 // Disk space, filesystem issues
	ExitTimeoutError    = 13 // Interactive timeout
	ExitInterruptError  = 14 // User Ctrl+C interrupt
	ExitThemeError      = 20 // Theme application failed
	ExitFontError       = 21 // Font installation failed
	ExitAppError        = 22 // Application installation failed
	ExitBackupError     = 23 // Backup/restore failed
	ExitMigrationError  = 24 // Migration failed
	ExitWarnings        = 64 // Completed with warnings
)

// ExitError provides specific exit codes for different failure modes.
type ExitError struct {
	Code    int
	Message string
	Err     error
}

// NewExitError creates an ExitError with the specified code and message.
func NewExitError(code int, message string, err error) *ExitError {
	return &ExitError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}

	return e.Message
}

func main() {
	os.Exit(run())
}

func run() int {
	// Acquire process lock to prevent multiple karei instances
	lockPath := filepath.Join(os.TempDir(), "karei.lock")
	lock := flock.New(lockPath)

	locked, err := lock.TryLock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to acquire process lock: %v\n", err)

		return ExitSystemError
	}

	if !locked {
		fmt.Fprintf(os.Stderr, "Another karei instance is already running\n")

		return ExitGeneralError
	}

	defer func() {
		if unlockErr := lock.Unlock(); unlockErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to release process lock: %v\n", unlockErr)
		}
	}()

	app := cli.App()

	ctx := context.Background()
	if err := app.Run(ctx, os.Args); err != nil {
		// All errors now must be ExitError with specific codes
		exitErr := &ExitError{}
		if errors.As(err, &exitErr) {
			// Error message to stderr only
			fmt.Fprintf(os.Stderr, "%s\n", exitErr.Message)

			return exitErr.Code
		}
		// Fallback for unexpected errors (should not happen)
		fmt.Fprintf(os.Stderr, "Unexpected error: %v\n", err)

		return ExitGeneralError
	}

	return ExitSuccess
}
