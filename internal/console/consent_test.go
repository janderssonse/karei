// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package console

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddConfigMarker(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		format   string
		expected string
	}{
		{
			name:     "shell format adds comment",
			content:  "export PATH=/usr/bin",
			format:   "shell",
			expected: "export PATH=/usr/bin # Modified by Karei on",
		},
		{
			name:     "conf format adds comment",
			content:  "option=value",
			format:   "conf",
			expected: "option=value # Modified by Karei on",
		},
		{
			name:     "json format returns unchanged",
			content:  `{"key": "value"}`,
			format:   "json",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddConfigMarker(tt.content, tt.format)

			if tt.format == "json" {
				assert.Equal(t, tt.expected, result)
			} else {
				assert.True(t, strings.HasPrefix(result, tt.expected))
				// Check it contains a date
				assert.Contains(t, result, time.Now().Format("2006"))
			}
		})
	}
}

func TestGetTimestamp(t *testing.T) {
	timestamp := GetTimestamp()

	// Parse the timestamp to verify format
	_, err := time.Parse("2006-01-02", timestamp)
	require.NoError(t, err, "Timestamp should be in YYYY-MM-DD format")

	// Check it's today's date
	today := time.Now().Format("2006-01-02")
	assert.Equal(t, today, timestamp)
}

func TestConsentConstants(t *testing.T) {
	// Verify constants are defined correctly
	assert.Equal(t, "yes", ConsentYes)
	assert.Equal(t, "y", ConsentY)
}

func TestAutoYesFlag(t *testing.T) {
	// Save original value
	original := AutoYes

	defer func() { AutoYes = original }()

	// Test setting the flag
	AutoYes = true
	assert.True(t, AutoYes)

	AutoYes = false
	assert.False(t, AutoYes)
}
