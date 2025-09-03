// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain

import (
	"errors"
	"fmt"
)

var (
	// ErrCircularDependency indicates a circular dependency was detected.
	ErrCircularDependency = errors.New("circular dependency detected")
	// ErrMissingDependency indicates a required dependency is not available.
	ErrMissingDependency = errors.New("missing required dependency")
)

// DependencyGraph represents package dependencies.
type DependencyGraph struct {
	packages map[string]*Package
	edges    map[string][]string // package -> dependencies
}

// NewDependencyGraph creates a new dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		packages: make(map[string]*Package),
		edges:    make(map[string][]string),
	}
}

// AddPackage adds a package to the dependency graph.
func (g *DependencyGraph) AddPackage(pkg *Package) {
	g.packages[pkg.Name] = pkg
	if pkg.Dependencies != nil {
		g.edges[pkg.Name] = pkg.Dependencies
	} else {
		g.edges[pkg.Name] = []string{}
	}
}

// HasCircularDependency checks if the graph has circular dependencies.
func (g *DependencyGraph) HasCircularDependency() (bool, []string) {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for pkg := range g.packages {
		if !visited[pkg] {
			if hasCycle, path := g.dfsDetectCycle(pkg, visited, recStack, []string{}); hasCycle {
				return true, path
			}
		}
	}

	return false, nil
}

// ResolveDependencies returns packages in installation order (topological sort).
func (g *DependencyGraph) ResolveDependencies(pkg string) ([]string, error) {
	// First check for cycles
	if hasCycle, cyclePath := g.HasCircularDependency(); hasCycle {
		return nil, fmt.Errorf("%w: %v", ErrCircularDependency, cyclePath)
	}

	visited := make(map[string]bool)
	result := make([]string, 0)

	if err := g.topologicalSort(pkg, visited, &result); err != nil {
		return nil, err
	}

	// The topological sort already gives us the correct order:
	// dependencies first, then the packages that depend on them
	return result, nil
}

// GetAllDependencies recursively gets all dependencies for a package.
func (g *DependencyGraph) GetAllDependencies(pkg string) []string {
	visited := make(map[string]bool)
	deps := make([]string, 0)
	g.collectDependencies(pkg, visited, &deps)

	return deps
}

// dfsDetectCycle performs depth-first search to detect cycles.
func (g *DependencyGraph) dfsDetectCycle(pkg string, visited, recStack map[string]bool, path []string) (bool, []string) {
	visited[pkg] = true
	recStack[pkg] = true
	path = append(path, pkg)

	for _, dep := range g.edges[pkg] {
		if !visited[dep] {
			if cycle, cyclePath := g.dfsDetectCycle(dep, visited, recStack, path); cycle {
				return true, cyclePath
			}
		} else if recStack[dep] {
			// Found a cycle - build the cycle path
			cycleStart := -1

			for i, p := range path {
				if p == dep {
					cycleStart = i
					break
				}
			}

			if cycleStart >= 0 {
				cyclePath := make([]string, 0, len(path)-cycleStart+1)
				cyclePath = append(cyclePath, path[cycleStart:]...)
				cyclePath = append(cyclePath, dep)

				return true, cyclePath
			}

			return true, append(path, dep)
		}
	}

	recStack[pkg] = false

	return false, nil
}

// topologicalSort performs topological sorting using DFS.
func (g *DependencyGraph) topologicalSort(pkg string, visited map[string]bool, result *[]string) error {
	if visited[pkg] {
		return nil
	}

	visited[pkg] = true

	for _, dep := range g.edges[pkg] {
		if _, exists := g.packages[dep]; !exists && dep != "" {
			// Allow missing optional dependencies but track them
			continue
		}

		if err := g.topologicalSort(dep, visited, result); err != nil {
			return err
		}
	}

	*result = append(*result, pkg)

	return nil
}

// collectDependencies recursively collects dependencies.
func (g *DependencyGraph) collectDependencies(pkg string, visited map[string]bool, deps *[]string) {
	if visited[pkg] {
		return
	}

	visited[pkg] = true

	for _, dep := range g.edges[pkg] {
		if !visited[dep] {
			*deps = append(*deps, dep)
			g.collectDependencies(dep, visited, deps)
		}
	}
}
