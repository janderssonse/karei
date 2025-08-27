// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform

import (
	"os"
	"testing"
)

func TestBoldNoColorCompliance(t *testing.T) {
	// Save and restore original environment
	origNoColor := os.Getenv("NO_COLOR")
	origTerm := os.Getenv("TERM")

	t.Cleanup(func() {
		if origNoColor != "" {
			t.Setenv("NO_COLOR", origNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}

		t.Setenv("TERM", origTerm)
	})

	testCases := []struct {
		name        string
		setupEnv    func(*testing.T)
		setupOutput func(*OutputState)
		input       string
		expectColor bool
	}{
		{
			name: "NO_COLOR environment variable set",
			setupEnv: func(t *testing.T) {
				t.Helper()
				t.Setenv("NO_COLOR", "1")
				t.Setenv("TERM", "xterm-256color")
			},
			setupOutput: func(o *OutputState) {
				o.SetMode(false, false, false)
			},
			input:       "TEST",
			expectColor: false,
		},
		{
			name: "TERM=dumb",
			setupEnv: func(t *testing.T) {
				t.Helper()
				_ = os.Unsetenv("NO_COLOR")
				t.Setenv("TERM", "dumb")
			},
			setupOutput: func(o *OutputState) {
				o.SetMode(false, false, false)
			},
			input:       "TEST",
			expectColor: false,
		},
		{
			name: "Plain mode enabled",
			setupEnv: func(t *testing.T) {
				t.Helper()
				_ = os.Unsetenv("NO_COLOR")
				t.Setenv("TERM", "xterm-256color")
			},
			setupOutput: func(o *OutputState) {
				o.SetMode(false, false, true) // plain=true
			},
			input:       "TEST",
			expectColor: false,
		},
		{
			name: "JSON mode enabled",
			setupEnv: func(t *testing.T) {
				t.Helper()
				_ = os.Unsetenv("NO_COLOR")
				t.Setenv("TERM", "xterm-256color")
			},
			setupOutput: func(o *OutputState) {
				o.SetMode(false, true, false) // json=true
			},
			input:       "TEST",
			expectColor: false,
		},
	}

	for _, testCase := range testCases { //nolint:paralleltest
		t.Run(testCase.name, func(t *testing.T) {
			// Setup environment
			testCase.setupEnv(t)

			// Create fresh OutputState instance
			output := &OutputState{}
			testCase.setupOutput(output)

			// Test Bold method
			result := output.Bold(testCase.input)

			// Check if color formatting is disabled
			if testCase.expectColor {
				// Should contain ANSI codes (when TTY)
				if result == testCase.input {
					t.Errorf("Expected color formatting, but got plain text: %q", result)
				}
			} else {
				// Should NOT contain ANSI codes
				if result != testCase.input {
					t.Errorf("Expected no color formatting, but got: %q (wanted: %q)", result, testCase.input)
				}
			}
		})
	}
}
