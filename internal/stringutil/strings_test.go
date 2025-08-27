// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package stringutil

import "testing"

func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text     string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"", "", true},
		{"test", "", true},
		{"", "test", false},
	}

	for _, tt := range tests {
		result := Contains(tt.text, tt.substr)
		if result != tt.expected {
			t.Errorf("Contains(%q, %q) = %v, want %v", tt.text, tt.substr, result, tt.expected)
		}
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		text     string
		substr   string
		expected bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "foo", false},
		{"TEST", "test", true},
		{"", "", true},
	}

	for _, tt := range tests {
		result := ContainsIgnoreCase(tt.text, tt.substr)
		if result != tt.expected {
			t.Errorf("ContainsIgnoreCase(%q, %q) = %v, want %v", tt.text, tt.substr, result, tt.expected)
		}
	}
}

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
