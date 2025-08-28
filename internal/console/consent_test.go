// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package console

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptConsent(t *testing.T) {
	tests := []struct {
		name        string
		prompt      string
		autoYes     bool
		userInput   string
		expected    bool
		expectError bool
	}{
		{
			name:        "auto yes flag returns true",
			prompt:      "Install packages?",
			autoYes:     true,
			userInput:   "", // ignored when autoYes is true
			expected:    true,
			expectError: false,
		},
		{
			name:        "user enters y",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "y\n",
			expected:    true,
			expectError: false,
		},
		{
			name:        "user enters Y",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "Y\n",
			expected:    true,
			expectError: false,
		},
		{
			name:        "user enters yes",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "yes\n",
			expected:    true,
			expectError: false,
		},
		{
			name:        "user enters YES",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "YES\n",
			expected:    true,
			expectError: false,
		},
		{
			name:        "user enters n",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "n\n",
			expected:    false,
			expectError: false,
		},
		{
			name:        "user enters N",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "N\n",
			expected:    false,
			expectError: false,
		},
		{
			name:        "user enters no",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "no\n",
			expected:    false,
			expectError: false,
		},
		{
			name:        "user enters NO",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "NO\n",
			expected:    false,
			expectError: false,
		},
		{
			name:        "user enters empty (default no)",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "\n",
			expected:    false,
			expectError: false,
		},
		{
			name:        "user enters invalid input",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "maybe\n",
			expected:    false,
			expectError: false, // Falls back to no
		},
		{
			name:        "EOF returns error",
			prompt:      "Continue?",
			autoYes:     false,
			userInput:   "", // EOF
			expected:    false,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a reader with test input
			reader := strings.NewReader(tc.userInput)

			var output bytes.Buffer

			result, err := PromptConsentWithReader(tc.prompt, tc.autoYes, reader, &output)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}

			// Verify prompt was shown (unless auto-yes)
			if !tc.autoYes && tc.userInput != "" {
				assert.Contains(t, output.String(), tc.prompt)
			}
		})
	}
}

func TestPromptConsentAutoYes(t *testing.T) {
	// Test that auto-yes doesn't read from stdin at all
	result, err := PromptConsentWithReader("Test prompt", true, nil, &bytes.Buffer{})

	require.NoError(t, err)
	assert.True(t, result)
}

// promptConsentWithReader is a testable version of prompt consent.
// It accepts custom reader and writer for testing.
func PromptConsentWithReader(prompt string, autoYes bool, reader io.Reader, writer io.Writer) (bool, error) {
	// If auto-yes is set, immediately return true
	if autoYes {
		_, _ = fmt.Fprintf(writer, "Auto-accepting: %s\n", prompt)
		return true, nil
	}

	// Show prompt
	_, _ = fmt.Fprintf(writer, "%s [y/N]: ", prompt)

	// Read response
	bufReader := bufio.NewReader(reader)

	response, err := bufReader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))

	return response == ConsentY || response == ConsentYes, nil
}
