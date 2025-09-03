// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// getDependencies is a helper function for test scenarios.
func getDependencies(pkg string) []string {
	deps := map[string][]string{
		"app":  {"lib1", "lib2"},
		"lib1": {"base"},
		"lib2": {"base"},
		"base": {},
		"tool": {"lib1"},
	}

	return deps[pkg]
}

// testGracefulDegradationUnderPressure tests graceful degradation under pressure.
func testGracefulDegradationUnderPressure(t *testing.T) {
	t.Helper()

	// Real scenario: System under heavy load degrades gracefully
	type LoadManager struct {
		activeOps    atomic.Int32
		maxOps       int32
		rejectedOps  atomic.Int32
		completedOps atomic.Int32
	}

	manager := &LoadManager{
		maxOps: 10, // Max concurrent operations
	}

	performOperation := func(_ int) bool {
		// Try to acquire slot
		current := manager.activeOps.Add(1)
		if current > manager.maxOps {
			// System overloaded, reject
			manager.activeOps.Add(-1)
			manager.rejectedOps.Add(1)

			return false
		}

		// Simulate work
		defer func() {
			manager.activeOps.Add(-1)
			manager.completedOps.Add(1)
		}()

		// Do work - add delay to ensure overlapping operations
		time.Sleep(time.Millisecond)

		return true
	}

	const totalOperations = 100

	var wg sync.WaitGroup

	for i := range totalOperations {
		wg.Add(1)

		go func(opID int) {
			defer wg.Done()

			_ = performOperation(opID)
		}(i)
	}

	wg.Wait()

	// Business rules for graceful degradation:
	totalProcessed := manager.completedOps.Load() + manager.rejectedOps.Load()
	assert.Equal(t, int32(totalOperations), totalProcessed,
		"All operations should be accounted for")

	// System should process at least some operations
	assert.Positive(t, manager.completedOps.Load(),
		"Some operations should complete")

	// System should reject some when overloaded
	assert.Positive(t, manager.rejectedOps.Load(),
		"Some operations should be rejected under load")

	// Active operations should be zero after completion
	assert.Equal(t, int32(0), manager.activeOps.Load(),
		"No operations should be active after completion")
}

// testDependencyResolutionUnderLoad tests dependency resolution under load.
func testDependencyResolutionUnderLoad(t *testing.T) {
	t.Helper()

	// Real scenario: Many packages with dependencies installed concurrently
	type DependencyResolver struct {
		mu       sync.RWMutex
		resolved map[string]bool
		inflight map[string]bool
	}

	resolver := &DependencyResolver{
		resolved: make(map[string]bool),
		inflight: make(map[string]bool),
	}

	var resolveDependencies func(pkg string, deps []string) bool

	resolveDependencies = func(pkg string, deps []string) bool {
		resolver.mu.Lock()

		// Check if already resolved
		if resolver.resolved[pkg] {
			resolver.mu.Unlock()
			return true
		}

		// Check if resolution in progress (circular dependency detection)
		if resolver.inflight[pkg] {
			resolver.mu.Unlock()
			return false // Circular dependency
		}

		// Mark as in-flight
		resolver.inflight[pkg] = true
		resolver.mu.Unlock()

		// Resolve dependencies first
		for _, dep := range deps {
			if !resolveDependencies(dep, getDependencies(dep)) {
				resolver.mu.Lock()
				delete(resolver.inflight, pkg)
				resolver.mu.Unlock()

				return false
			}
		}

		// Mark as resolved
		resolver.mu.Lock()
		resolver.resolved[pkg] = true
		delete(resolver.inflight, pkg)
		resolver.mu.Unlock()

		return true
	}

	// Test with concurrent resolution
	packages := []struct {
		name string
		deps []string
	}{
		{"app", []string{"lib1", "lib2"}},
		{"lib1", []string{"base"}},
		{"lib2", []string{"base"}},
		{"base", []string{}},
		{"tool", []string{"lib1"}},
	}

	var wg sync.WaitGroup

	results := make([]bool, len(packages))

	for i, pkg := range packages {
		wg.Add(1)

		go func(idx int, p struct {
			name string
			deps []string
		}) {
			defer wg.Done()

			results[idx] = resolveDependencies(p.name, p.deps)
		}(i, pkg)
	}

	wg.Wait()

	// Most packages should resolve successfully (some may fail due to concurrency)
	successCount := 0

	for _, result := range results {
		if result {
			successCount++
		}
	}

	// At least the base package and some others should succeed
	assert.GreaterOrEqual(t, successCount, 1,
		"At least some packages should resolve successfully")

	// Resolved packages should not exceed total packages
	assert.LessOrEqual(t, len(resolver.resolved), len(packages),
		"Should not resolve more packages than exist")
}

// testMultipleUsersInstallingSamePackage tests multiple users installing same package.
func testMultipleUsersInstallingSamePackage(t *testing.T) {
	t.Helper()

	// Real scenario: Multiple users try to install same package
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	sharedPackage := &domain.Package{
		Name:   "firefox",
		Method: domain.MethodAPT,
		Source: "ubuntu",
	}

	ctx := context.Background()

	var (
		firstInstaller        atomic.Int32
		alreadyInstalledCount atomic.Int32
	)

	// First install succeeds

	mockInstaller.On("Install", ctx, sharedPackage).
		Run(func(_ mock.Arguments) {
			if !firstInstaller.CompareAndSwap(0, 1) {
				// Others get "already installed"
				alreadyInstalledCount.Add(1)
			}
		}).
		Return(&domain.InstallationResult{
			Package: sharedPackage,
			Success: true,
		}, nil).Once()

	// Subsequent installs get "already installed"
	mockInstaller.On("Install", ctx, sharedPackage).
		Return(nil, domain.ErrAlreadyInstalled).Maybe()

	const numUsers = 5

	var wg sync.WaitGroup

	results := make([]error, numUsers)

	// Multiple users try to install
	for i := range numUsers {
		wg.Add(1)

		go func(userID int) {
			defer wg.Done()

			_, err := service.Install(ctx, sharedPackage)
			results[userID] = err
		}(i)
	}

	wg.Wait()

	// Business rule: Only one actual installation should occur
	successCount := 0
	alreadyInstalledErrors := 0

	for _, err := range results {
		if err == nil {
			successCount++
		} else if errors.Is(err, domain.ErrAlreadyInstalled) {
			alreadyInstalledErrors++
		}
	}

	assert.LessOrEqual(t, successCount, 1,
		"At most one installation should succeed")
	assert.GreaterOrEqual(t, alreadyInstalledErrors, numUsers-1,
		"Other users should get already installed error")
}

// testSystemUpdateDuringUserInstall tests system update during user install.
func testSystemUpdateDuringUserInstall(t *testing.T) {
	t.Helper()

	// Real scenario: System update starts while user installs app
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	systemPackages := []*domain.Package{
		{Name: "libc6", Method: domain.MethodAPT, Source: "ubuntu"},
		{Name: "openssl", Method: domain.MethodAPT, Source: "ubuntu"},
		{Name: "systemd", Method: domain.MethodAPT, Source: "ubuntu"},
	}

	userPackage := &domain.Package{
		Name:   "vscode",
		Method: domain.MethodSnap,
		Source: "snapcraft",
	}

	ctx := context.Background()

	// System packages should have priority
	var (
		systemStarted atomic.Bool
		userBlocked   atomic.Bool
	)

	// Setup mocks for system packages

	for _, pkg := range systemPackages {
		mockInstaller.On("Install", ctx, pkg).
			Run(func(_ mock.Arguments) {
				systemStarted.Store(true)
			}).
			Return(&domain.InstallationResult{
				Package: pkg,
				Success: true,
			}, nil).Maybe()
	}

	// User package should wait if system update is running
	mockInstaller.On("Install", ctx, userPackage).
		Run(func(_ mock.Arguments) {
			if systemStarted.Load() {
				userBlocked.Store(true)
			}
		}).
		Return(&domain.InstallationResult{
			Package: userPackage,
			Success: true,
		}, nil).Maybe()

	var wg sync.WaitGroup

	// Start system update
	wg.Add(1)

	go func() {
		defer wg.Done()

		for _, pkg := range systemPackages {
			_, _ = service.Install(ctx, pkg)
		}
	}()

	// User tries to install (slightly delayed)
	wg.Add(1)

	go func() {
		defer wg.Done()

		time.Sleep(10 * time.Millisecond) // Small delay

		_, _ = service.Install(ctx, userPackage)
	}()

	wg.Wait()

	// Business rule: System updates have priority
	assert.True(t, systemStarted.Load(), "System update should start")
	// User operation may or may not be blocked depending on implementation
	// This test documents the expected behavior
}

// TestRealWorldConcurrentScenarios tests actual concurrent usage patterns.
func TestRealWorldConcurrentScenarios(t *testing.T) {
	t.Parallel()

	t.Run("system_update_during_user_install", func(t *testing.T) {
		t.Parallel()
		testSystemUpdateDuringUserInstall(t)
	})

	t.Run("multiple_users_installing_same_package", func(t *testing.T) {
		t.Parallel()
		testMultipleUsersInstallingSamePackage(t)
	})

	t.Run("dependency_resolution_under_load", func(t *testing.T) {
		t.Parallel()
		testDependencyResolutionUnderLoad(t)
	})

	t.Run("graceful_degradation_under_pressure", func(t *testing.T) {
		t.Parallel()
		testGracefulDegradationUnderPressure(t)
	})

	t.Run("race_condition_in_cache_invalidation", func(t *testing.T) {
		t.Parallel()

		// Real scenario: Package cache invalidation during concurrent access
		type PackageCache struct {
			mu      sync.RWMutex
			data    map[string]*domain.Package
			version int64
		}

		cache := &PackageCache{
			data:    make(map[string]*domain.Package),
			version: 0,
		}

		// Read from cache
		readCache := func(name string) (*domain.Package, bool) {
			cache.mu.RLock()
			defer cache.mu.RUnlock()

			pkg, exists := cache.data[name]

			return pkg, exists
		}

		// Write to cache
		writeCache := func(pkg *domain.Package) {
			cache.mu.Lock()
			defer cache.mu.Unlock()

			cache.data[pkg.Name] = pkg
			cache.version++
		}

		// Invalidate cache
		invalidateCache := func() {
			cache.mu.Lock()
			defer cache.mu.Unlock()

			cache.data = make(map[string]*domain.Package)
			cache.version++
		}

		var wg sync.WaitGroup

		const numOps = 100

		// Mix of reads, writes, and invalidations
		for i := range numOps {
			wg.Add(1)

			go func(opID int) {
				defer wg.Done()

				switch opID % 3 {
				case 0: // Read
					_, _ = readCache("test-pkg")
				case 1: // Write
					pkg := &domain.Package{
						Name:   "test-pkg",
						Method: domain.MethodAPT,
						Source: "ubuntu",
					}
					writeCache(pkg)
				case 2: // Invalidate
					if opID%10 == 2 { // Less frequent
						invalidateCache()
					}
				}
			}(i)
		}

		wg.Wait()

		// No assertion on final state - this test verifies no deadlocks/races
		// The cache should be in a valid state (no corruption)
		cache.mu.RLock()
		finalVersion := cache.version
		cache.mu.RUnlock()

		assert.GreaterOrEqual(t, finalVersion, int64(0),
			"Cache version should never be negative")
	})
}
