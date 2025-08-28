// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package system provides system-level utilities including file operations, command execution, and permissions for Karei.
package system

import "errors"

var (
	// ErrWrappedError indicates a generic wrapped error condition.
	ErrWrappedError = errors.New("wrapped error")
)
