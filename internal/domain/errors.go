// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain

import (
	"errors"
	"strings"
)

// Common domain errors.
var (
	ErrPermissionDenied  = errors.New("permission denied")
	ErrNetworkFailure    = errors.New("network failure")
	ErrAlreadyInstalled  = errors.New("already installed")
	ErrNotInstalled      = errors.New("not installed")
	ErrDependencyMissing = errors.New("dependency missing")
)

// ErrorInfo provides user-friendly error information.
type ErrorInfo struct {
	Message     string   // User-friendly message
	Suggestions []string // Actionable suggestions
	ShowDetails bool     // Whether to show technical details
}

// getErrorMatchers returns error patterns and their corresponding info.
func getErrorMatchers() []struct {
	patterns []string
	getInfo  func(string, bool) ErrorInfo
} {
	return []struct {
		patterns []string
		getInfo  func(string, bool) ErrorInfo
	}{
		{
			patterns: []string{"permission", "denied", "sudo", "root"},
			getInfo: func(_ string, verbose bool) ErrorInfo {
				return ErrorInfo{
					Message:     "Permission denied",
					Suggestions: []string{"Try running with sudo", "Check that your user has admin privileges"},
					ShowDetails: verbose,
				}
			},
		},
		{
			patterns: []string{"network", "connection", "timeout", "no such host"},
			getInfo: func(_ string, verbose bool) ErrorInfo {
				return ErrorInfo{
					Message:     "Network connection failed",
					Suggestions: []string{"Check your internet connection", "Try again in a few moments"},
					ShowDetails: verbose,
				}
			},
		},
		{
			patterns: []string{"not found", "no such", "unable to locate"},
			getInfo: func(pkg string, verbose bool) ErrorInfo {
				if pkg != "" {
					return ErrorInfo{
						Message:     "Package '" + pkg + "' not found",
						Suggestions: []string{"Check the package name spelling", "Update package lists: sudo apt update"},
						ShowDetails: verbose,
					}
				}
				return ErrorInfo{
					Message:     "Package not found",
					Suggestions: []string{"Verify the package name", "Update your package lists"},
					ShowDetails: verbose,
				}
			},
		},
		{
			patterns: []string{"already installed", "is installed"},
			getInfo: func(_ string, verbose bool) ErrorInfo {
				return ErrorInfo{
					Message:     "Already installed",
					Suggestions: []string{"Package is already on your system"},
					ShowDetails: verbose,
				}
			},
		},
		{
			patterns: []string{"not installed", "is not installed"},
			getInfo: func(_ string, verbose bool) ErrorInfo {
				return ErrorInfo{
					Message:     "Not installed",
					Suggestions: []string{"Package is not on your system", "Use 'karei list' to see installed packages"},
					ShowDetails: verbose,
				}
			},
		},
		{
			patterns: []string{"dependency", "depends", "requires"},
			getInfo: func(_ string, verbose bool) ErrorInfo {
				return ErrorInfo{
					Message:     "Missing dependencies",
					Suggestions: []string{"Install required dependencies first", "Try: sudo apt --fix-broken install"},
					ShowDetails: verbose,
				}
			},
		},
	}
}

// GetErrorInfo analyzes an error and returns user-friendly information.
func GetErrorInfo(err error, packageName string, verbose bool) ErrorInfo {
	if err == nil {
		return ErrorInfo{}
	}

	errStr := strings.ToLower(err.Error())

	// Check each error pattern
	for _, matcher := range getErrorMatchers() {
		for _, pattern := range matcher.patterns {
			if strings.Contains(errStr, pattern) {
				return matcher.getInfo(packageName, verbose)
			}
		}
	}

	// Generic error - show details in verbose mode
	return ErrorInfo{
		Message:     "Operation failed",
		Suggestions: []string{"Run with --verbose for more details"},
		ShowDetails: verbose,
	}
}

// FormatErrorMessage formats an error for display.
func FormatErrorMessage(err error, packageName string, verbose bool) string {
	info := GetErrorInfo(err, packageName, verbose)

	var result strings.Builder

	// Main message
	if packageName != "" {
		result.WriteString("✗ Failed to install ")
		result.WriteString(packageName)

		if info.Message != "" {
			result.WriteString(": ")
			result.WriteString(info.Message)
		}
	} else {
		result.WriteString("✗ ")
		result.WriteString(info.Message)
	}

	// Add technical details if verbose
	if info.ShowDetails && err != nil {
		result.WriteString("\n  Technical details: ")
		result.WriteString(err.Error())
	}

	// Add suggestions
	if len(info.Suggestions) > 0 && !verbose {
		// In non-verbose mode, just show the first suggestion inline
		result.WriteString(" (")
		result.WriteString(info.Suggestions[0])
		result.WriteString(")")
	} else if len(info.Suggestions) > 0 && verbose {
		// In verbose mode, show all suggestions
		result.WriteString("\n  Suggestions:")

		for _, suggestion := range info.Suggestions {
			result.WriteString("\n    • ")
			result.WriteString(suggestion)
		}
	}

	return result.String()
}
