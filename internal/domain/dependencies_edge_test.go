// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDependencyGraphEdgeCases tests edge cases and error conditions.
func TestDependencyGraphEdgeCases(t *testing.T) {
	// These tests ensure 100% coverage of dependencies.go
	t.Run("empty_graph", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Empty graph has no cycles
		hasCycle, cyclePath := graph.HasCircularDependency()
		assert.False(t, hasCycle, "Empty graph should have no cycles")
		assert.Nil(t, cyclePath, "Empty graph should return nil cycle path")

		// Resolving non-existent package
		order, err := graph.ResolveDependencies("non-existent")
		require.NoError(t, err)
		assert.Len(t, order, 1)
		assert.Equal(t, "non-existent", order[0])
	})

	t.Run("single_package_no_deps", func(t *testing.T) {
		graph := domain.NewDependencyGraph()
		graph.AddPackage(&domain.Package{
			Name:         "standalone",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		hasCycle, _ := graph.HasCircularDependency()
		assert.False(t, hasCycle, "Single package should have no cycles")

		order, err := graph.ResolveDependencies("standalone")
		require.NoError(t, err)
		assert.Equal(t, []string{"standalone"}, order)
	})

	t.Run("package_with_empty_string_dependency", func(t *testing.T) {
		graph := domain.NewDependencyGraph()
		graph.AddPackage(&domain.Package{
			Name:         "pkg-with-empty-dep",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"", "real-dep", ""},
		})
		graph.AddPackage(&domain.Package{
			Name:         "real-dep",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		// Should handle empty string dependencies gracefully
		order, err := graph.ResolveDependencies("pkg-with-empty-dep")
		require.NoError(t, err)
		assert.Contains(t, order, "real-dep")
		assert.Contains(t, order, "pkg-with-empty-dep")
	})

	t.Run("get_all_deps_with_cycles", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Create a cycle
		graph.AddPackage(&domain.Package{
			Name:         "a",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"b"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "b",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"a"},
		})

		// GetAllDependencies should handle cycles without infinite loop
		deps := graph.GetAllDependencies("a")
		assert.Contains(t, deps, "b")
		assert.Len(t, deps, 1) // Should only return b once
	})

	t.Run("deep_dependency_chain", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Create deep chain: a -> b -> c -> d -> e -> f
		chain := []string{"a", "b", "c", "d", "e", "f"}
		for i := range len(chain) - 1 {
			graph.AddPackage(&domain.Package{
				Name:         chain[i],
				Method:       domain.MethodAPT,
				Source:       "ubuntu",
				Dependencies: []string{chain[i+1]},
			})
		}
		// Last package has no dependencies
		graph.AddPackage(&domain.Package{
			Name:         chain[len(chain)-1],
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		order, err := graph.ResolveDependencies("a")
		require.NoError(t, err)
		assert.Len(t, order, 6)
		// Should be in reverse order: f, e, d, c, b, a
		for i := range chain {
			assert.Equal(t, chain[len(chain)-1-i], order[i])
		}
	})

	t.Run("multiple_paths_to_same_dep", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Multiple paths to 'base':
		// app -> lib1 -> base
		// app -> lib2 -> base
		// app -> lib3 -> lib4 -> base
		graph.AddPackage(&domain.Package{
			Name:         "app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"lib1", "lib2", "lib3"},
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
			Name:         "lib3",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"lib4"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "lib4",
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

		// Base should appear only once and before all libs
		baseCount := 0
		baseIndex := -1

		for i, pkg := range order {
			if pkg == "base" {
				baseCount++
				baseIndex = i
			}
		}

		assert.Equal(t, 1, baseCount, "Base should appear exactly once")
		assert.Equal(t, 0, baseIndex, "Base should be first")
	})

	t.Run("partial_graph_resolution", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Add some packages
		graph.AddPackage(&domain.Package{
			Name:         "group1-a",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"group1-b"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "group1-b",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})
		graph.AddPackage(&domain.Package{
			Name:         "group2-a",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"group2-b"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "group2-b",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{},
		})

		// Resolve only group1
		order, err := graph.ResolveDependencies("group1-a")
		require.NoError(t, err)
		assert.Len(t, order, 2)
		assert.NotContains(t, order, "group2-a")
		assert.NotContains(t, order, "group2-b")
	})

	t.Run("cycle_detection_path_accuracy", func(t *testing.T) {
		graph := domain.NewDependencyGraph()

		// Create a specific cycle: a -> b -> c -> b (b and c form cycle)
		graph.AddPackage(&domain.Package{
			Name:         "a",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"b"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "b",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"c"},
		})
		graph.AddPackage(&domain.Package{
			Name:         "c",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"b"}, // Creates cycle with b
		})

		hasCycle, cyclePath := graph.HasCircularDependency()
		assert.True(t, hasCycle)
		assert.NotNil(t, cyclePath)
		// The cycle path should contain b and c
		assert.Contains(t, cyclePath, "b")
		assert.Contains(t, cyclePath, "c")
		// The last element should be the same as first (showing the cycle)
		if len(cyclePath) > 0 {
			assert.Equal(t, cyclePath[len(cyclePath)-1], cyclePath[0],
				"Cycle path should end where it started")
		}
	})
}

// TestDependencyCollectionCompleteness tests GetAllDependencies thoroughly.
func TestDependencyCollectionCompleteness(t *testing.T) {
	graph := domain.NewDependencyGraph()

	// Build a complex graph:
	//        app
	//       / | \
	//      /  |  \
	//    ui  api  db
	//    |    |    |
	//   react |  postgres
	//     \   |   /
	//      \  |  /
	//      nodejs
	graph.AddPackage(&domain.Package{
		Name:         "app",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"ui", "api", "db"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "ui",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"react"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "api",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"nodejs"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "db",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"postgres"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "react",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"nodejs"},
	})
	graph.AddPackage(&domain.Package{
		Name:         "postgres",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"nodejs"}, // Contrived but tests shared deps
	})
	graph.AddPackage(&domain.Package{
		Name:         "nodejs",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{},
	})

	allDeps := graph.GetAllDependencies("app")

	// Should include all transitive dependencies exactly once
	expectedDeps := []string{"ui", "api", "db", "react", "nodejs", "postgres"}
	assert.Len(t, allDeps, len(expectedDeps))

	for _, expected := range expectedDeps {
		assert.Contains(t, allDeps, expected)
	}

	// Test from middle of graph
	reactDeps := graph.GetAllDependencies("react")
	assert.Equal(t, []string{"nodejs"}, reactDeps)

	// Test leaf node
	nodejsDeps := graph.GetAllDependencies("nodejs")
	assert.Empty(t, nodejsDeps)
}
