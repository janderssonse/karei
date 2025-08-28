// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package stringutil

import "testing"

func TestContainsAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text       string
		substrings []string
		expected   bool
	}{
		{"hello world", []string{"world", "foo"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"hello world", []string{}, false},
		{"", []string{"test"}, false},
		{"test", []string{""}, true},
	}

	for _, tt := range tests {
		result := ContainsAny(tt.text, tt.substrings)
		if result != tt.expected {
			t.Errorf("ContainsAny(%q, %v) = %v, want %v", tt.text, tt.substrings, result, tt.expected)
		}
	}
}
