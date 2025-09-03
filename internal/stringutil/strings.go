// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package stringutil

import "strings"

// ContainsAny checks if text contains any of the provided substrings.
func ContainsAny(text string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(text, substr) {
			return true
		}
	}

	return false
}
