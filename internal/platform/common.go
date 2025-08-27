// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package platform provides platform utilities for Karei.
package platform

import "errors"

var (
	// ErrWrappedError indicates a generic wrapped error condition.
	ErrWrappedError = errors.New("wrapped error")
)
