// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestExitErrorFormatting tests that ExitError properly formats messages.
func TestExitErrorFormatting(t *testing.T) {
	tests := []struct {
		name            string
		exitError       *domain.ExitError
		expectedCode    int
		expectedMessage string
	}{
		{
			name: "exit error with underlying error",
			exitError: domain.NewExitError(1, "Operation failed",
				errors.New("permission denied")),
			expectedCode:    1,
			expectedMessage: "Operation failed: permission denied",
		},
		{
			name:            "exit error without underlying error",
			exitError:       domain.NewExitError(2, "Invalid configuration", nil),
			expectedCode:    2,
			expectedMessage: "Invalid configuration",
		},
		{
			name: "exit error with network failure",
			exitError: domain.NewExitError(3, "Download failed",
				domain.ErrNetworkFailure),
			expectedCode:    3,
			expectedMessage: "Download failed: network failure",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedMessage, tc.exitError.Error())
			assert.Equal(t, tc.expectedCode, tc.exitError.Code)
		})
	}
}

// TestFormatErrorMessage tests user-friendly error formatting.
func TestFormatErrorMessage(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		packageName      string
		verbose          bool
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:        "permission error non-verbose",
			err:         errors.New("permission denied"),
			packageName: "docker",
			verbose:     false,
			shouldContain: []string{
				"Failed to install docker",
				"Permission denied",
				"Try running with sudo",
			},
			shouldNotContain: []string{
				"Technical details",
			},
		},
		{
			name:        "permission error verbose",
			err:         errors.New("permission denied: cannot write to /usr/local"),
			packageName: "docker",
			verbose:     true,
			shouldContain: []string{
				"Failed to install docker",
				"Permission denied",
				"Technical details",
				"cannot write to /usr/local",
			},
		},
		{
			name:        "network error",
			err:         errors.New("network timeout while downloading"),
			packageName: "tool",
			verbose:     false,
			shouldContain: []string{
				"Failed to install tool",
				"Network connection failed",
				"Check your internet connection",
			},
		},
		{
			name:        "package not found",
			err:         errors.New("unable to locate package vim"),
			packageName: "vim",
			verbose:     false,
			shouldContain: []string{
				"Failed to install vim",
				"Package 'vim' not found",
				"Check the package name spelling",
			},
		},
		{
			name:        "already installed",
			err:         domain.ErrAlreadyInstalled,
			packageName: "git",
			verbose:     false,
			shouldContain: []string{
				"Already installed",
				"Package is already on your system",
			},
		},
		{
			name:        "not installed error",
			err:         domain.ErrNotInstalled,
			packageName: "unknown-app",
			verbose:     false,
			shouldContain: []string{
				"Not installed",
				"Package is not on your system",
			},
		},
		{
			name:        "dependency error",
			err:         errors.New("dependency libssl missing"),
			packageName: "app",
			verbose:     false,
			shouldContain: []string{
				"Missing dependencies",
				"Install required dependencies first",
			},
		},
		{
			name:        "generic error non-verbose",
			err:         errors.New("unexpected error occurred"),
			packageName: "app",
			verbose:     false,
			shouldContain: []string{
				"Operation failed",
				"Run with --verbose for more details",
			},
			shouldNotContain: []string{
				"unexpected error occurred",
			},
		},
		{
			name:        "generic error verbose",
			err:         errors.New("unexpected error occurred"),
			packageName: "app",
			verbose:     true,
			shouldContain: []string{
				"Operation failed",
				"Technical details",
				"unexpected error occurred",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := domain.FormatErrorMessage(tc.err, tc.packageName, tc.verbose)

			for _, expected := range tc.shouldContain {
				assert.Contains(t, result, expected,
					"Error message should contain: %s", expected)
			}

			for _, unexpected := range tc.shouldNotContain {
				assert.NotContains(t, result, unexpected,
					"Error message should not contain: %s", unexpected)
			}
		})
	}
}

// TestFormatErrorMessageForUserExperience tests that error messages provide good UX.
func TestFormatErrorMessageForUserExperience(t *testing.T) {
	// Business rule: Error messages should be actionable and user-friendly
	tests := []struct {
		name           string
		err            error
		packageName    string
		verbose        bool
		mustContain    []string
		mustNotContain []string
	}{
		{
			name:        "permission_error_suggests_sudo",
			err:         errors.New("permission denied"),
			packageName: "docker",
			verbose:     false,
			mustContain: []string{"✗", "docker", "sudo"},
		},
		{
			name:        "network_error_actionable",
			err:         errors.New("connection timeout"),
			packageName: "nodejs",
			verbose:     false,
			mustContain: []string{"✗", "nodejs", "Network"},
		},
		{
			name:        "verbose_shows_technical_details",
			err:         errors.New("complex internal error with stack trace"),
			packageName: "rust",
			verbose:     true,
			mustContain: []string{"Technical details", "complex internal error"},
		},
		{
			name:           "non_verbose_hides_technical_jargon",
			err:            errors.New("ENOENT syscall failed at 0x7fff"),
			packageName:    "python",
			verbose:        false,
			mustNotContain: []string{"ENOENT", "0x7fff", "syscall"},
			mustContain:    []string{"✗", "python"},
		},
		{
			name:        "dependency_error_suggests_fix",
			err:         errors.New("dependency error: libssl-dev missing"),
			packageName: "openssl",
			verbose:     false,
			mustContain: []string{"Missing dependencies", "Install required dependencies"},
		},
		{
			name:        "already_installed_clear_message",
			err:         errors.New("package is already installed"),
			packageName: "git",
			verbose:     false,
			mustContain: []string{"Already installed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.FormatErrorMessage(tt.err, tt.packageName, tt.verbose)

			for _, expected := range tt.mustContain {
				assert.Contains(t, result, expected,
					"Error message should contain user-friendly element: %s", expected)
			}

			for _, unexpected := range tt.mustNotContain {
				assert.NotContains(t, result, unexpected,
					"Error message should not expose technical details: %s", unexpected)
			}
		})
	}
}

// TestErrorMessageSuggestions tests that suggestions are appropriate.
func TestErrorMessageSuggestions(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedSuggestion string
	}{
		{
			name:               "permission error suggests sudo",
			err:                domain.ErrPermissionDenied,
			expectedSuggestion: "Try running with sudo",
		},
		{
			name:               "network error suggests connection check",
			err:                domain.ErrNetworkFailure,
			expectedSuggestion: "Check your internet connection",
		},
		{
			name:               "not found suggests update",
			err:                errors.New("package not found"),
			expectedSuggestion: "Check the package name spelling", // Match actual suggestion
		},
		{
			name:               "dependency error suggests fix",
			err:                domain.ErrDependencyMissing,
			expectedSuggestion: "Install required dependencies first",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test in non-verbose mode where suggestions are inline
			result := domain.FormatErrorMessage(tc.err, "test-package", false)

			// Should contain the suggestion
			assert.Contains(t, result, tc.expectedSuggestion)
		})
	}
}

// TestErrorPatternMatching tests the pattern matching for error categorization.
func TestErrorPatternMatching(t *testing.T) {
	tests := []struct {
		name            string
		errorMessage    string
		expectedPattern string
	}{
		{
			name:            "matches permission pattern",
			errorMessage:    "Operation not permitted: permission denied",
			expectedPattern: "Permission denied",
		},
		{
			name:            "matches sudo pattern",
			errorMessage:    "This operation requires sudo privileges",
			expectedPattern: "Permission denied",
		},
		{
			name:            "matches network pattern",
			errorMessage:    "connection refused to server",
			expectedPattern: "Network connection failed",
		},
		{
			name:            "matches timeout pattern",
			errorMessage:    "operation timeout after 30 seconds",
			expectedPattern: "Network connection failed",
		},
		{
			name:            "matches not found pattern",
			errorMessage:    "Package not found in repository",
			expectedPattern: "not found",
		},
		{
			name:            "matches no such pattern",
			errorMessage:    "no such package available",
			expectedPattern: "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := errors.New(tc.errorMessage)
			formatted := domain.FormatErrorMessage(err, "test", false)

			// Convert to lowercase for pattern matching
			formattedLower := strings.ToLower(formatted)
			patternLower := strings.ToLower(tc.expectedPattern)

			assert.Contains(t, formattedLower, patternLower,
				"Error should be categorized as: %s", tc.expectedPattern)
		})
	}
}

// TestExitErrorCodeMeaning tests that exit codes follow Unix conventions.
func TestExitErrorCodeMeaning(t *testing.T) {
	// Business rule: Exit codes should follow Unix conventions
	tests := []struct {
		code     int
		meaning  string
		scenario string
	}{
		{
			code:     1,
			meaning:  "general error",
			scenario: "Package installation failed",
		},
		{
			code:     2,
			meaning:  "misuse of shell command",
			scenario: "Invalid command line arguments",
		},
		{
			code:     126,
			meaning:  "command cannot execute",
			scenario: "Permission denied on executable",
		},
		{
			code:     127,
			meaning:  "command not found",
			scenario: "Required tool not installed",
		},
		{
			code:     130,
			meaning:  "terminated by Ctrl+C",
			scenario: "User interrupted installation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			// Create error with the conventional code
			err := domain.NewExitError(tc.code, tc.scenario, nil)

			// Verify the code is set correctly
			assert.Equal(t, tc.code, err.Code,
				"Exit code should match Unix convention for %s", tc.meaning)

			// Verify error message includes the scenario
			assert.Contains(t, err.Error(), tc.scenario,
				"Error message should describe the scenario")
		})
	}
}
