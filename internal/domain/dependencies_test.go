// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCircularDependencyDetection tests real circular dependency detection.
func TestCircularDependencyDetection(t *testing.T) {
	// Business Rule: System must detect and prevent circular dependencies
	t.Run("simple_circular_dependency", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// A -> B -> C -> A (circular)
		graph.AddPackage(&domain.Package{
			Name:         "package-a",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"package-b"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "package-b",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"package-c"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "package-c",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"package-a"},
		})

		hasCycle, cyclePath := graph.HasCircularDependency()
		assert.True(t, hasCycle, "Should detect circular dependency")
		assert.NotEmpty(t, cyclePath, "Should provide cycle path")
		assert.Contains(t, cyclePath, "package-a")
		assert.Contains(t, cyclePath, "package-b")
		assert.Contains(t, cyclePath, "package-c")
	})

	t.Run("self_dependency", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Package depends on itself
		graph.AddPackage(&domain.Package{
			Name:         "self-referential",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"self-referential"},
		})

		hasCycle, cyclePath := graph.HasCircularDependency()
		assert.True(t, hasCycle, "Should detect self-dependency as circular")
		assert.Contains(t, cyclePath, "self-referential")
	})

	t.Run("complex_circular_dependency", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// A -> B -> C -> D -> B (circular with extra node)
		graph.AddPackage(&domain.Package{
			Name:         "app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"framework"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "framework",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"runtime"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "runtime",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"compiler"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "compiler",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"framework"}, // Creates cycle
		})

		hasCycle, cyclePath := graph.HasCircularDependency()
		assert.True(t, hasCycle, "Should detect complex circular dependency")
		assert.NotEmpty(t, cyclePath, "Should provide cycle path")
		// The cycle is: framework -> runtime -> compiler -> framework
		assert.Contains(t, cyclePath, "framework")
		assert.Contains(t, cyclePath, "runtime")
		assert.Contains(t, cyclePath, "compiler")
	})

	t.Run("no_circular_dependency", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Valid dependency tree
		graph.AddPackage(&domain.Package{
			Name:         "app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"lib-a", "lib-b"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "lib-a",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"lib-base"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "lib-b",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"lib-base"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "lib-base",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		hasCycle, cyclePath := graph.HasCircularDependency()
		assert.False(t, hasCycle, "Should not detect circular dependency in valid tree")
		assert.Empty(t, cyclePath, "Should not provide cycle path when no cycle exists")
	})
}

// TestDependencyResolution tests dependency resolution and ordering.
func TestDependencyResolution(t *testing.T) {
	// Business Rule: Dependencies must be installed in correct order
	t.Run("simple_dependency_chain", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// app -> lib -> base
		graph.AddPackage(&domain.Package{
			Name:         "app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"lib"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "lib",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"base"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "base",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		order, err := graph.ResolveDependencies("app")
		require.NoError(t, err)
		require.Len(t, order, 3)

		// Verify installation order: base, lib, app
		assert.Equal(t, "base", order[0], "Base should be installed first")
		assert.Equal(t, "lib", order[1], "Lib should be installed second")
		assert.Equal(t, "app", order[2], "App should be installed last")
	})

	t.Run("diamond_dependency", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Diamond pattern:
		//     app
		//    /   \
		//   lib1 lib2
		//    \   /
		//     base
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
			Dependencies: []string{"base"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "lib2",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"base"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "base",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		order, err := graph.ResolveDependencies("app")
		require.NoError(t, err)
		require.Len(t, order, 4)

		// Base must come before lib1 and lib2
		baseIndex := indexOf(order, "base")
		lib1Index := indexOf(order, "lib1")
		lib2Index := indexOf(order, "lib2")
		appIndex := indexOf(order, "app")

		assert.Less(t, baseIndex, lib1Index, "Base must be installed before lib1")
		assert.Less(t, baseIndex, lib2Index, "Base must be installed before lib2")
		assert.Less(t, lib1Index, appIndex, "Lib1 must be installed before app")
		assert.Less(t, lib2Index, appIndex, "Lib2 must be installed before app")
	})

	t.Run("missing_optional_dependency", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Package with missing optional dependency
		graph.AddPackage(&domain.Package{
			Name:         "app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"required-lib", "optional-lib"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "required-lib",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})
		// Note: optional-lib is not added to graph (simulating missing package)

		order, err := graph.ResolveDependencies("app")
		require.NoError(t, err)
		assert.Contains(t, order, "app")
		assert.Contains(t, order, "required-lib")
		// Optional dependency is skipped, not an error
	})

	t.Run("circular_dependency_prevents_resolution", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Create circular dependency
		graph.AddPackage(&domain.Package{
			Name:         "pkg-a",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"pkg-b"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "pkg-b",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"pkg-a"},
		})

		order, err := graph.ResolveDependencies("pkg-a")
		require.Error(t, err, "Should fail to resolve circular dependencies")
		require.ErrorIs(t, err, domain.ErrCircularDependency)
		assert.Nil(t, order, "Should not return order when circular dependency exists")
	})
}

// TestGetAllDependencies tests recursive dependency collection.
func TestGetAllDependencies(t *testing.T) {
	// Business Rule: System must track all transitive dependencies
	graph := domain.NewDependencyGraph()

	// Complex dependency tree
	graph.AddPackage(&domain.Package{
		Name:         "webapp",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"frontend", "backend"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "frontend",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"react", "webpack"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "backend",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"nodejs", "express"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "react",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"nodejs"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "webpack",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"nodejs"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "express",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"nodejs"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "nodejs",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{},
	})

	allDeps := graph.GetAllDependencies("webapp")

	// Should include all transitive dependencies
	assert.Contains(t, allDeps, "frontend")
	assert.Contains(t, allDeps, "backend")
	assert.Contains(t, allDeps, "react")
	assert.Contains(t, allDeps, "webpack")
	assert.Contains(t, allDeps, "nodejs")
	assert.Contains(t, allDeps, "express")

	// Should not include webapp itself
	assert.NotContains(t, allDeps, "webapp")
}

// Helper function.
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}

	return -1
}
