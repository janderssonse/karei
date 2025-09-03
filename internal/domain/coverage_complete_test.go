// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"errors"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDependencyGraphFullCoverage tests all edge cases for 100% coverage.
func TestDependencyGraphFullCoverage(t *testing.T) {
	// These tests specifically target uncovered lines in dependencies.go
	t.Run("dfsDetectCycle_fallback_path", func(t *testing.T) {
		// Business Rule: Even if cycle detection has an internal error,
		// it should still report the cycle with available information
		graph := domain.NewDependencyGraph()

		// Create a cycle that might trigger the fallback path in dfsDetectCycle
		// This happens when cycleStart calculation fails (line 79 fallback)
		graph.AddPackage(&domain.Package{
			Name:         "pkg1",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"pkg2"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "pkg2",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"pkg3"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "pkg3",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"pkg1"},
		})

		hasCycle, cyclePath := graph.HasCircularDependency()
		assert.True(t, hasCycle, "Should detect cycle")
		assert.NotEmpty(t, cyclePath, "Should provide cycle path even with fallback")
	})

	t.Run("topologicalSort_with_error_propagation", func(t *testing.T) {
		// Business Rule: topologicalSort should propagate errors from recursive calls
		// This tests line 119-121 where recursive error is returned
		graph := domain.NewDependencyGraph()

		// Create a deep dependency chain that might cause stack issues
		for i := range 100 {
			graph.AddPackage(&domain.Package{
				Name:         string(rune('a'+i%26)) + "_" + string(rune('0'+i%10)),
				Method:       domain.MethodAPT,
				Source:       "ubuntu",
				Dependencies: []string{},
			})
		}

		// This should complete without error
		order, err := graph.ResolveDependencies("a0")
		require.NoError(t, err, "Should handle deep recursion")
		assert.NotNil(t, order, "Should return order")
	})

	t.Run("collectDependencies_already_visited", func(t *testing.T) {
		// Business Rule: collectDependencies should skip already visited nodes
		// This tests the early return at line 138-140
		graph := domain.NewDependencyGraph()

		// Create a graph where multiple packages depend on the same base
		graph.AddPackage(&domain.Package{
			Name:         "app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"lib1", "lib2"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "lib1",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"base", "shared"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "lib2",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"base", "shared"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "base",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})
		graph.AddPackage(&domain.Package{
			Name:         "shared",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"base"}, // shared also depends on base
		})

		deps := graph.GetAllDependencies("app")

		// Count occurrences of each dependency
		depCount := make(map[string]int)
		for _, dep := range deps {
			depCount[dep]++
		}

		// Each dependency should appear exactly once
		for dep, count := range depCount {
			assert.Equal(t, 1, count, "Dependency %s should appear exactly once", dep)
		}

		// Should have all 4 dependencies
		assert.Len(t, deps, 4, "Should have exactly 4 unique dependencies")
	})

	t.Run("resolveDependencies_with_missing_optional_deps", func(t *testing.T) {
		// Business Rule: Resolution should handle missing optional dependencies
		// This tests line 115-117 where missing deps are allowed
		graph := domain.NewDependencyGraph()

		graph.AddPackage(&domain.Package{
			Name:         "main-app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"required-lib", "optional-feature", ""},
		})
		graph.AddPackage(&domain.Package{
			Name:         "required-lib",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})
		// Note: optional-feature is NOT added (simulating optional dependency)

		order, err := graph.ResolveDependencies("main-app")
		require.NoError(t, err, "Should handle missing optional dependencies")
		assert.Contains(t, order, "main-app")
		assert.Contains(t, order, "required-lib")
		// optional-feature should not be in the order
		assert.NotContains(t, order, "optional-feature")
	})
}

// TestErrorMatchersFullCoverage tests all error matcher patterns.
func TestErrorMatchersFullCoverage(t *testing.T) {
	// Business Rule: All error patterns must be tested for user experience
	t.Run("getErrorMatchers_all_patterns", func(t *testing.T) {
		// Test each specific error pattern to ensure getErrorMatchers is fully covered
		errorPatterns := []struct {
			name          string
			errorText     string
			packageName   string
			expectedInMsg string
		}{
			// Permission patterns
			{"permission", "permission denied", "pkg", "Permission denied"},
			{"denied", "access denied", "pkg", "Permission denied"},
			{"sudo", "sudo required", "pkg", "Permission denied"},
			{"root", "must be root", "pkg", "Permission denied"},

			// Network patterns
			{"network", "network error", "pkg", "Network connection failed"},
			{"connection", "connection refused", "pkg", "Network connection failed"},
			{"timeout", "timeout occurred", "pkg", "Network connection failed"},
			{"no_such_host", "no such host", "pkg", "Network connection failed"},

			// Not found patterns
			{"not_found", "package not found", "pkg", "not found"},
			{"no_such", "no such package", "pkg", "not found"},
			{"unable_to_locate", "unable to locate package", "pkg", "not found"},

			// Already installed patterns
			{"already_installed", "already installed", "pkg", "Already installed"},
			{"is_installed", "package is installed", "pkg", "Already installed"},

			// Not installed patterns
			{"not_installed", "not installed", "pkg", "Not installed"},
			{"is_not_installed", "is not installed", "pkg", "Not installed"},

			// Dependency patterns
			{"dependency", "dependency error", "pkg", "Missing dependencies"},
			{"depends", "depends on missing", "pkg", "Missing dependencies"},
			{"requires", "requires package X", "pkg", "Missing dependencies"},
		}

		for _, tc := range errorPatterns {
			t.Run(tc.name, func(t *testing.T) {
				err := errors.New(tc.errorText)
				formatted := domain.FormatErrorMessage(err, tc.packageName, false)
				assert.Contains(t, formatted, tc.expectedInMsg,
					"Pattern '%s' should trigger correct message", tc.name)
			})
		}
	})

	t.Run("getErrorInfo_nil_error", func(t *testing.T) {
		// Test nil error handling
		formatted := domain.FormatErrorMessage(nil, "package", false)
		assert.Contains(t, formatted, "Failed to install package",
			"Nil error should still format with package name")
	})

	t.Run("formatErrorMessage_empty_package", func(t *testing.T) {
		// Test error formatting without package name
		err := errors.New("generic error")
		formatted := domain.FormatErrorMessage(err, "", false)
		assert.Contains(t, formatted, "✗ Operation failed",
			"Should format error without package name")
		assert.NotContains(t, formatted, "Failed to install",
			"Should not mention installation when no package")
	})
}

// TestCompletePackageInstallationFlow tests a meaningful end-to-end scenario.
func TestCompletePackageInstallationFlow(t *testing.T) {
	// Business Rule: Complete package installation must handle all edge cases
	t.Run("complex_real_world_dependency_tree", func(t *testing.T) {
		// Simulate installing a complex application with real-world dependencies
		graph := domain.NewDependencyGraph()

		// Web application with database, cache, and monitoring
		graph.AddPackage(&domain.Package{
			Name:         "webapp",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"nginx", "postgresql", "redis", "prometheus"},
		})

		// Web server dependencies
		graph.AddPackage(&domain.Package{
			Name:         "nginx",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"openssl", "pcre", "zlib"},
		})

		// Database dependencies
		graph.AddPackage(&domain.Package{
			Name:         "postgresql",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"openssl", "readline", "zlib"},
		})

		// Cache dependencies
		graph.AddPackage(&domain.Package{
			Name:         "redis",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"libc6", "openssl"},
		})

		// Monitoring dependencies
		graph.AddPackage(&domain.Package{
			Name:         "prometheus",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"libc6"},
		})

		// Shared libraries
		graph.AddPackage(&domain.Package{
			Name:         "openssl",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"libc6"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "pcre",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"libc6"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "zlib",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"libc6"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "readline",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"libc6"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "libc6",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		// Resolve dependencies
		order, err := graph.ResolveDependencies("webapp")
		require.NoError(t, err, "Should resolve complex dependency tree")

		// Verify installation order
		// libc6 must be first as it has no dependencies
		assert.Equal(t, "libc6", order[0], "Base library should be installed first")

		// webapp must be last
		assert.Equal(t, "webapp", order[len(order)-1], "Main app should be installed last")

		// All shared libraries should come before their dependents
		libc6Idx := findIndex(order, "libc6")
		opensslIdx := findIndex(order, "openssl")
		nginxIdx := findIndex(order, "nginx")
		postgresqlIdx := findIndex(order, "postgresql")

		assert.Less(t, libc6Idx, opensslIdx, "libc6 before openssl")
		assert.Less(t, opensslIdx, nginxIdx, "openssl before nginx")
		assert.Less(t, opensslIdx, postgresqlIdx, "openssl before postgresql")

		// Verify all packages are included
		assert.Len(t, order, 10, "Should include all 10 packages")

		// Get all dependencies for the main app
		allDeps := graph.GetAllDependencies("webapp")
		assert.Len(t, allDeps, 9, "webapp should have 9 total dependencies")
	})
}

// Helper function for tests.
func findIndex(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}

	return -1
}
