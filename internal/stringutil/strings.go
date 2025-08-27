// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package stringutil provides string utility functions for Karei.
package stringutil

import "strings"

// Contains checks if text contains substr (case-sensitive).
// This is a wrapper around strings.Contains for consistency.
func Contains(text, substr string) bool {
	return strings.Contains(text, substr)
}

// ContainsIgnoreCase checks if text contains substr (case-insensitive).
func ContainsIgnoreCase(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}

// ContainsAny checks if text contains any of the provided substrings.
func ContainsAny(text string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(text, substr) {
			return true
		}
	}

	return false
}
