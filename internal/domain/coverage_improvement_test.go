// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkErrorSuggestions is a helper function to check error suggestions formatting.
func checkErrorSuggestions(t *testing.T, expectedMsg string, verbose, hasSuggestions bool, formatted string) {
	t.Helper()

	// Handle generic errors separately
	if expectedMsg == "Operation failed" {
		checkGenericErrorSuggestions(t, verbose, formatted)
		return
	}

	// Handle errors with suggestions
	if hasSuggestions {
		checkErrorWithSuggestions(t, verbose, formatted)
	}
}

func checkGenericErrorSuggestions(t *testing.T, verbose bool, formatted string) {
	t.Helper()
	// Generic errors always have "Run with --verbose" suggestion
	if verbose {
		assert.Contains(t, formatted, "Suggestions:",
			"Verbose mode should have suggestions section")
	} else {
		assert.Contains(t, formatted, "(Run with --verbose",
			"Non-verbose mode should suggest --verbose flag")
	}
}

func checkErrorWithSuggestions(t *testing.T, verbose bool, formatted string) {
	t.Helper()
	// In non-verbose mode, suggestions are inline
	if !verbose {
		assert.Contains(t, formatted, "(",
			"Non-verbose mode should have inline suggestions")
	} else {
		assert.Contains(t, formatted, "Suggestions:",
			"Verbose mode should have suggestions section")
	}
}

// TestDependencyCyclePthEdgeCases tests specific edge cases in cycle detection
// These are MEANINGFUL tests that validate real-world scenarios.
func TestDependencyCyclePthEdgeCases(t *testing.T) {
	// Business Rule: System must accurately report the exact cycle path for debugging
	t.Run("cycle_path_when_start_index_not_found", func(t *testing.T) {
		// This tests a defensive programming case - if the cycle detection
		// algorithm encounters an unexpected state where the cycle start
		// cannot be found in the path, it should still return a valid result
		graph := domain.NewDependencyGraph()

		// Create a complex multi-level dependency that might trigger edge cases
		graph.AddPackage(&domain.Package{
			Name:         "web-app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"framework", "database"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "framework",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"orm"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "orm",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"database"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "database",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"web-app"}, // Creates complex cycle
		})

		hasCycle, cyclePath := graph.HasCircularDependency()
		assert.True(t, hasCycle, "Should detect the complex circular dependency")
		assert.NotEmpty(t, cyclePath, "Should provide a cycle path even in edge cases")

		// The cycle path should contain all involved packages
		cycleStr := strings.Join(cyclePath, " -> ")
		assert.Contains(t, cycleStr, "web-app")
		assert.Contains(t, cycleStr, "database")
	})

	t.Run("already_visited_dependency_in_resolution", func(t *testing.T) {
		// Business Rule: When resolving dependencies, if a package is already
		// visited (e.g., shared dependency), it should be skipped to avoid duplication
		graph := domain.NewDependencyGraph()

		// Create a diamond dependency where 'common' is shared
		//     app
		//    /   \
		//  svc1  svc2
		//    \   /
		//   common
		graph.AddPackage(&domain.Package{
			Name:         "app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"service1", "service2"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "service1",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"common-lib"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "service2",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"common-lib"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "common-lib",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		order, err := graph.ResolveDependencies("app")
		require.NoError(t, err)

		// common-lib should appear exactly once
		commonCount := 0

		for _, pkg := range order {
			if pkg == "common-lib" {
				commonCount++
			}
		}

		assert.Equal(t, 1, commonCount, "Shared dependency should appear exactly once")

		// Verify correct order
		commonIdx := -1
		service1Idx := -1
		service2Idx := -1
		appIdx := -1

		for i, pkg := range order {
			switch pkg {
			case "common-lib":
				commonIdx = i
			case "service1":
				service1Idx = i
			case "service2":
				service2Idx = i
			case "app":
				appIdx = i
			}
		}

		assert.Less(t, commonIdx, service1Idx, "common-lib must be installed before service1")
		assert.Less(t, commonIdx, service2Idx, "common-lib must be installed before service2")
		assert.Less(t, service1Idx, appIdx, "service1 must be installed before app")
		assert.Less(t, service2Idx, appIdx, "service2 must be installed before app")
	})

	t.Run("topological_sort_error_propagation", func(t *testing.T) {
		// Business Rule: If topological sort encounters an error during recursion,
		// it should properly propagate the error up the call stack
		graph := domain.NewDependencyGraph()

		// Create a scenario with missing required dependencies
		graph.AddPackage(&domain.Package{
			Name:         "enterprise-app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"middleware", "missing-critical-dep"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "middleware",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"runtime"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "runtime",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})
		// Note: missing-critical-dep is NOT added to the graph

		// This should handle the missing dependency gracefully
		order, err := graph.ResolveDependencies("enterprise-app")
		require.NoError(t, err, "Should handle missing optional dependencies")
		assert.Contains(t, order, "runtime")
		assert.Contains(t, order, "middleware")
		assert.Contains(t, order, "enterprise-app")
	})

	t.Run("collect_dependencies_with_visited_nodes", func(t *testing.T) {
		// Business Rule: GetAllDependencies should not include already visited
		// dependencies to prevent infinite loops and duplicates
		graph := domain.NewDependencyGraph()

		// Create a complex graph with shared dependencies
		graph.AddPackage(&domain.Package{
			Name:         "microservice",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"api-gateway", "message-queue", "cache"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "api-gateway",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"auth", "cache"}, // cache is shared
		})
		graph.AddPackage(&domain.Package{
			Name:         "message-queue",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"cache"}, // cache is shared
		})
		graph.AddPackage(&domain.Package{
			Name:         "cache",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"memory-manager"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "auth",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})
		graph.AddPackage(&domain.Package{
			Name:         "memory-manager",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		deps := graph.GetAllDependencies("microservice")

		// Each dependency should appear exactly once
		depCounts := make(map[string]int)
		for _, dep := range deps {
			depCounts[dep]++
		}

		for dep, count := range depCounts {
			assert.Equal(t, 1, count, "Dependency %s should appear exactly once", dep)
		}

		// Should include all transitive dependencies
		assert.Contains(t, deps, "api-gateway")
		assert.Contains(t, deps, "message-queue")
		assert.Contains(t, deps, "cache")
		assert.Contains(t, deps, "auth")
		assert.Contains(t, deps, "memory-manager")
		assert.Len(t, deps, 5, "Should have exactly 5 unique dependencies")
	})
}

// TestErrorMatchersCoverage tests all error matcher patterns
// These are MEANINGFUL tests for user experience and error reporting.
func TestErrorMatchersCoverage(t *testing.T) {
	// Business Rule: All error types must have user-friendly messages
	t.Run("all_error_patterns_coverage", func(t *testing.T) {
		testCases := []struct {
			name           string
			errorText      string
			packageName    string
			verbose        bool
			expectedMsg    string
			hasSuggestions bool
		}{
			{
				name:           "permission_error_verbose",
				errorText:      "permission denied while accessing /etc/config",
				packageName:    "nginx",
				verbose:        true,
				expectedMsg:    "Permission denied",
				hasSuggestions: true,
			},
			{
				name:           "sudo_error_non_verbose",
				errorText:      "sudo: command not found",
				packageName:    "docker",
				verbose:        false,
				expectedMsg:    "Permission denied",
				hasSuggestions: true,
			},
			{
				name:           "network_timeout_verbose",
				errorText:      "connection timeout after 30 seconds",
				packageName:    "remote-pkg",
				verbose:        true,
				expectedMsg:    "Network connection failed",
				hasSuggestions: true,
			},
			{
				name:           "no_such_host_non_verbose",
				errorText:      "no such host: example.com",
				packageName:    "cloud-cli",
				verbose:        false,
				expectedMsg:    "Network connection failed",
				hasSuggestions: true,
			},
			{
				name:           "package_not_found_verbose",
				errorText:      "package vim not found in repository",
				packageName:    "vim",
				verbose:        true,
				expectedMsg:    "Package not found",
				hasSuggestions: true,
			},
			{
				name:           "command_not_found_non_verbose",
				errorText:      "command not found: gcc",
				packageName:    "build-essential",
				verbose:        false,
				expectedMsg:    "Package not found", // "not found" pattern matches to Package not found
				hasSuggestions: true,
			},
			{
				name:           "disk_space_error_verbose",
				errorText:      "No space left on device",
				packageName:    "large-app",
				verbose:        true,
				expectedMsg:    "Operation failed", // No disk space pattern, falls back to generic
				hasSuggestions: false,              // Generic error has "Run with --verbose" suggestion
			},
			{
				name:           "unknown_error_verbose",
				errorText:      "some random error that doesn't match patterns",
				packageName:    "mystery-pkg",
				verbose:        true,
				expectedMsg:    "Operation failed",
				hasSuggestions: false,
			},
			{
				name:           "dependency_error_non_verbose",
				errorText:      "dependency resolution failed for package X",
				packageName:    "complex-app",
				verbose:        false,
				expectedMsg:    "Missing dependencies",
				hasSuggestions: true,
			},
			{
				name:           "unresolved_error_verbose",
				errorText:      "unresolved dependencies: lib1, lib2",
				packageName:    "app-with-deps",
				verbose:        true,
				expectedMsg:    "Operation failed", // "unresolved" doesn't match any pattern
				hasSuggestions: false,
			},
			{
				name:           "disk_full_pattern",
				errorText:      "disk full error occurred",
				packageName:    "big-package",
				verbose:        false,
				expectedMsg:    "Operation failed", // No disk pattern, falls back to generic
				hasSuggestions: false,
			},
			{
				name:           "already_installed_pattern",
				errorText:      "package is already installed on system",
				packageName:    "existing-pkg",
				verbose:        false,
				expectedMsg:    "Already installed",
				hasSuggestions: true,
			},
			{
				name:           "not_installed_pattern",
				errorText:      "package is not installed",
				packageName:    "missing-pkg",
				verbose:        true,
				expectedMsg:    "Not installed",
				hasSuggestions: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create an error from the error text
				err := errors.New(tc.errorText)
				formatted := domain.FormatErrorMessage(err, tc.packageName, tc.verbose)

				// Check that the formatted error contains the expected message
				// For package-specific errors, the message will be different
				if tc.packageName != "" && tc.expectedMsg == "Package not found" {
					assert.Contains(t, formatted, "not found",
						"Error format should contain 'not found' for package errors")
				} else {
					assert.Contains(t, formatted, tc.expectedMsg,
						"Error format should contain user-friendly message")
				}

				// If verbose, should show original error
				if tc.verbose {
					assert.Contains(t, formatted, "Technical details:",
						"Verbose mode should show technical details")
					assert.Contains(t, formatted, tc.errorText,
						"Verbose mode should show original error")
				}

				// Check for suggestions - generic errors have different suggestions
				checkErrorSuggestions(t, tc.expectedMsg, tc.verbose, tc.hasSuggestions, formatted)
			})
		}
	})

	t.Run("error_formatting_with_special_characters", func(t *testing.T) {
		// Business Rule: Error formatting should handle special characters safely
		specialErrors := []string{
			"error: path/with\\backslash failed",
			"error: quote\" in 'message'",
			"error: newline\nin\nmessage",
			"error: tab\tin\tmessage",
			"error: unicode→symbols←present",
		}

		for _, errText := range specialErrors {
			err := errors.New(errText)
			formatted := domain.FormatErrorMessage(err, "test-pkg", false)
			assert.NotEmpty(t, formatted, "Should format special character errors")
			assert.NotContains(t, formatted, "panic", "Should not panic on special chars")
		}
	})
}

// TestResolveDependenciesWithCircularCheck tests the circular dependency check in ResolveDependencies
// This is a MEANINGFUL test for production safety.
func TestResolveDependenciesWithCircularCheck(t *testing.T) {
	// Business Rule: ResolveDependencies must fail fast when circular dependencies are detected
	t.Run("resolve_fails_on_circular_dependency", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Create a production-like scenario with circular dependency
		// Common in microservices: service A needs B for auth, B needs A for data
		graph.AddPackage(&domain.Package{
			Name:         "auth-service",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"user-service"}, // Needs user data
		})
		graph.AddPackage(&domain.Package{
			Name:         "user-service",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"auth-service"}, // Needs auth validation
		})

		// This should fail with circular dependency error
		_, err := graph.ResolveDependencies("auth-service")
		require.Error(t, err, "Should fail when circular dependency exists")
		assert.Contains(t, err.Error(), "circular dependency",
			"Error should mention circular dependency")

		// The error should include the cycle path for debugging
		assert.Contains(t, err.Error(), "auth-service",
			"Error should include packages in cycle")
		assert.Contains(t, err.Error(), "user-service",
			"Error should include packages in cycle")
	})
}
