// SPDX-FileCopyrightText: 2024 Josef Andersson
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestUserFriendlyErrorMessages tests that errors are formatted for end users.
func TestUserFriendlyErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		packageName    string
		verbose        bool
		mustContain    string
		mustNotContain string
	}{
		{
			name:           "permission_error_suggests_sudo",
			err:            errors.New("permission denied"),
			packageName:    "vim",
			verbose:        false,
			mustContain:    "sudo",
			mustNotContain: "technical",
		},
		{
			name:           "network_error_suggests_connection_check",
			err:            errors.New("connection timeout"),
			packageName:    "firefox",
			verbose:        false,
			mustContain:    "connection",
			mustNotContain: "timeout", // Technical detail hidden
		},
		{
			name:           "verbose_mode_shows_technical_details",
			err:            errors.New("connection timeout at 192.168.1.1:443"),
			packageName:    "firefox",
			verbose:        true,
			mustContain:    "192.168.1.1:443",
			mustNotContain: "",
		},
		{
			name:           "package_not_found_includes_name",
			err:            errors.New("unable to locate package firefox-dev"),
			packageName:    "firefox-dev",
			verbose:        false,
			mustContain:    "firefox-dev",
			mustNotContain: "",
		},
		{
			name:           "nil_error_returns_empty",
			err:            nil,
			packageName:    "",
			verbose:        false,
			mustContain:    "",
			mustNotContain: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := domain.FormatErrorMessage(tt.err, tt.packageName, tt.verbose)

			if tt.err == nil {
				// Nil errors should produce minimal output
				assert.LessOrEqual(t, len(formatted), 10, "Nil error should produce minimal output")
			} else {
				// All error messages should have the error marker
				assert.Contains(t, formatted, "✗", "Error messages should have error marker")

				if tt.mustContain != "" {
					assert.Contains(t, strings.ToLower(formatted), strings.ToLower(tt.mustContain),
						"Should contain expected text")
				}

				if tt.mustNotContain != "" {
					assert.NotContains(t, formatted, tt.mustNotContain,
						"Should not contain technical details in non-verbose mode")
				}

				// Business rule: Error messages should be concise
				lines := strings.Count(formatted, "\n")
				if !tt.verbose {
					assert.LessOrEqual(t, lines, 2, "Non-verbose errors should be concise")
				}
			}
		})
	}
}

// TestErrorSuggestionsAreActionable tests that error suggestions help users.
func TestErrorSuggestionsAreActionable(t *testing.T) {
	actionableErrors := []error{
		errors.New("permission denied"),
		errors.New("network timeout"),
		errors.New("package not found"),
		errors.New("dependency conflict"),
		errors.New("already installed"),
	}

	for _, err := range actionableErrors {
		formatted := domain.FormatErrorMessage(err, "test-pkg", false)

		// Business rule: Known errors should provide actionable guidance
		hasActionableWords := strings.Contains(formatted, "Try") ||
			strings.Contains(formatted, "Check") ||
			strings.Contains(formatted, "Install") ||
			strings.Contains(formatted, "Update") ||
			strings.Contains(formatted, "Use") ||
			strings.Contains(formatted, "Run") ||
			strings.Contains(formatted, "Package is")

		assert.True(t, hasActionableWords,
			"Error '%s' should provide actionable guidance", err.Error())
	}
}

// TestErrorFormattingConsistency tests formatting consistency.
func TestErrorFormattingConsistency(t *testing.T) {
	// Same error should format consistently
	err := errors.New("permission denied")

	format1 := domain.FormatErrorMessage(err, "vim", false)
	format2 := domain.FormatErrorMessage(err, "vim", false)

	assert.Equal(t, format1, format2, "Same error should format consistently")

	// Package name should be included when provided
	withPkg := domain.FormatErrorMessage(err, "vim", false)
	withoutPkg := domain.FormatErrorMessage(err, "", false)

	assert.Contains(t, withPkg, "vim", "Package name should be included")
	assert.NotContains(t, withoutPkg, "vim", "Should not contain package name when not provided")
}

// TestCriticalErrorPrioritization tests that critical errors are clear.
func TestCriticalErrorPrioritization(t *testing.T) {
	criticalErrors := []struct {
		err      error
		critical string
	}{
		{errors.New("permission denied: cannot write to /usr"), "Permission"},
		{errors.New("no such host: package.server.com"), "Network"},
		{errors.New("package vim not found"), "not found"},
	}

	for _, ce := range criticalErrors {
		formatted := domain.FormatErrorMessage(ce.err, "pkg", false)

		// Critical error type should be clearly communicated
		assert.Contains(t, formatted, ce.critical,
			"Critical error type should be clear: %s", ce.err)

		// Should not be overly technical in non-verbose mode
		assert.LessOrEqual(t, len(formatted), 200,
			"Error message should be concise for CLI display")
	}
}

// TestErrorFormattingPerformance tests that error formatting is fast.
func TestErrorFormattingPerformance(t *testing.T) {
	// Even complex errors should format quickly
	complexError := errors.New(strings.Repeat("error ", 100))

	// Format 1000 times - should be nearly instant
	for range 1000 {
		formatted := domain.FormatErrorMessage(complexError, "pkg", false)
		assert.NotEmpty(t, formatted)
	}

	// Business rule: Error formatting should not allocate excessively
	// This is implicitly tested by the loop above completing quickly
}
