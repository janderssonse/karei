// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package uninstall

import (
	"context"
	"github.com/janderssonse/karei/internal/adapters/system"
)

// CommandExecutor abstracts system command execution for testing.
type CommandExecutor interface {
	Run(ctx context.Context, verbose bool, name string, args ...string) error
	RunWithPassword(ctx context.Context, verbose bool, password string, args ...string) error
}

// RealCommandExecutor executes actual system commands.
type RealCommandExecutor struct{}

// Run executes a system command with the given arguments.
func (r *RealCommandExecutor) Run(ctx context.Context, verbose bool, name string, args ...string) error {
	return system.Run(ctx, verbose, name, args...)
}

// RunWithPassword executes a system command with sudo using the provided password.
func (r *RealCommandExecutor) RunWithPassword(ctx context.Context, verbose bool, password string, args ...string) error {
	return system.RunWithPassword(ctx, verbose, password, args...)
}
