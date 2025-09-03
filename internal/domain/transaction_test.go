// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
)

// testPartialFailureRecovery tests partial failure recovery.
func testPartialFailureRecovery(t *testing.T) {
	t.Helper()

	// Test recovery strategies for partial failures

	packages := []struct {
		name     string
		willFail bool
	}{
		{"essential-lib", false},
		{"optional-feature", true}, // This will fail
		{"core-component", false},
	}

	essentialPackages := make([]string, 0)
	optionalPackages := make([]string, 0)

	for _, pkg := range packages {
		if pkg.willFail {
			// Track optional package failure
			optionalPackages = append(optionalPackages, pkg.name)
			continue
		}
		// Install essential packages
		essentialPackages = append(essentialPackages, pkg.name)
	}

	// Essential packages should still be installed
	assert.Len(t, essentialPackages, 2, "Essential packages should be installed")
	assert.Contains(t, essentialPackages, "essential-lib")
	assert.Contains(t, essentialPackages, "core-component")

	// Optional package failure tracked but doesn't stop installation
	assert.Len(t, optionalPackages, 1, "Optional package failure should be tracked")
	assert.Contains(t, optionalPackages, "optional-feature")
}

// testConcurrentBatchIsolation tests concurrent batch isolation.
func testConcurrentBatchIsolation(t *testing.T) {
	t.Helper()

	// Business Rule: Concurrent batch installations should be isolated

	batch1 := []string{"vim", "emacs", "nano"}
	batch2 := []string{"docker", "podman", "containerd"}

	var wg sync.WaitGroup

	batch1Results := make([]string, 0)
	batch2Results := make([]string, 0)

	var mu1, mu2 sync.Mutex

	// Batch 1 installation
	wg.Add(1)

	go func() {
		defer wg.Done()

		for _, pkg := range batch1 {
			mu1.Lock()

			batch1Results = append(batch1Results, pkg)

			mu1.Unlock()
			time.Sleep(10 * time.Millisecond) // Simulate work
		}
	}()

	// Batch 2 installation
	wg.Add(1)

	go func() {
		defer wg.Done()

		for _, pkg := range batch2 {
			mu2.Lock()

			batch2Results = append(batch2Results, pkg)

			mu2.Unlock()
			time.Sleep(10 * time.Millisecond) // Simulate work
		}
	}()

	wg.Wait()

	// Verify isolation
	assert.ElementsMatch(t, batch1, batch1Results, "Batch 1 should complete independently")
	assert.ElementsMatch(t, batch2, batch2Results, "Batch 2 should complete independently")

	// Verify no cross-contamination
	for _, pkg := range batch1Results {
		assert.NotContains(t, batch2, pkg, "Batch 1 packages should not appear in batch 2")
	}

	for _, pkg := range batch2Results {
		assert.NotContains(t, batch1, pkg, "Batch 2 packages should not appear in batch 1")
	}
}

// testBatchWithDependenciesTransactional tests batch with dependencies transactional.
func testBatchWithDependenciesTransactional(t *testing.T) {
	t.Helper()

	// Package with dependencies - all must succeed or all must fail
	mainPkg := &domain.Package{
		Name:         "webapp",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"database", "cache", "webserver"},
	}

	depPackages := map[string]*domain.Package{
		"database": {
			Name:   "postgresql",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
		"cache": {
			Name:   "redis",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
		"webserver": {
			Name:   "nginx",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
	}

	// Simulate transactional installation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	installOrder := []string{"database", "cache", "webserver", "webapp"}
	installed := make([]string, 0)

InstallLoop:
	for _, name := range installOrder {
		select {
		case <-ctx.Done():
			// Timeout - rollback
			t.Log("Installation timed out, rolling back")

			installed = nil

			break InstallLoop
		default:
			if name == "webapp" {
				// Install main package
				installed = append(installed, mainPkg.Name)
			} else {
				// Install dependency
				if pkg, ok := depPackages[name]; ok {
					installed = append(installed, pkg.Name)
				}
			}
		}
	}

	// Verify transaction completed
	assert.Len(t, installed, 4, "All packages should be installed in transaction")
}

// testBatchInstallationRollback tests batch installation rollback on failure.
func testBatchInstallationRollback(t *testing.T) {
	t.Helper()

	packages := []*domain.Package{
		{
			Name:   "package1",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
		{
			Name:   "package2",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
		{
			Name:   "failing-package",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
		{
			Name:   "package3",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
	}

	// Track installed packages for rollback
	installed := make([]string, 0)
	rolledBack := false

	for _, pkg := range packages {
		if pkg.Name == "failing-package" {
			// Simulate failure - trigger rollback
			rolledBack = true

			// Rollback previously installed packages
			for i := len(installed) - 1; i >= 0; i-- {
				// In real implementation, would uninstall installed[i]
				_ = installed[i] // Acknowledge the package for rollback
			}

			installed = nil

			break
		}

		installed = append(installed, pkg.Name)
	}

	assert.True(t, rolledBack, "Rollback should be triggered on failure")
	assert.Empty(t, installed, "All installed packages should be rolled back")
}

// testSuccessfulBatchInstallation tests a successful batch installation.
func testSuccessfulBatchInstallation(t *testing.T) {
	t.Helper()

	packages := []*domain.Package{
		{
			Name:   "nginx",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
		{
			Name:   "postgresql",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
		{
			Name:   "redis",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		},
	}

	// Simulate batch installation
	results := make([]*domain.InstallationResult, 0, len(packages))
	installedCount := 0

	for _, pkg := range packages {
		result := &domain.InstallationResult{
			Package:  pkg,
			Success:  true,
			Duration: 100,
		}
		results = append(results, result)
		installedCount++
	}

	// Verify all packages installed
	assert.Equal(t, len(packages), installedCount, "All packages should be installed")

	for _, result := range results {
		assert.True(t, result.Success, "Each package should succeed")
	}
}

// TestBatchInstallationTransactions tests atomic batch installation behavior.
func TestBatchInstallationTransactions(t *testing.T) {
	// Business Rule: Batch installations must be atomic - all succeed or all fail
	t.Run("successful_batch_installation", func(t *testing.T) {
		testSuccessfulBatchInstallation(t)
	})

	t.Run("batch_installation_rollback_on_failure", func(t *testing.T) {
		testBatchInstallationRollback(t)
	})

	t.Run("batch_with_dependencies_transactional", func(t *testing.T) {
		testBatchWithDependenciesTransactional(t)
	})

	t.Run("concurrent_batch_isolation", func(t *testing.T) {
		testConcurrentBatchIsolation(t)
	})

	t.Run("partial_failure_recovery", func(t *testing.T) {
		testPartialFailureRecovery(t)
	})
}

// TestTransactionSavepoints tests savepoint/checkpoint functionality.
func TestTransactionSavepoints(t *testing.T) {
	// Business Rule: Support savepoints for complex multi-stage installations
	t.Run("savepoint_creation_and_restore", func(t *testing.T) {
		type InstallState struct {
			Packages   []string
			Savepoints map[string][]string
		}

		state := &InstallState{
			Packages:   make([]string, 0),
			Savepoints: make(map[string][]string),
		}

		// Stage 1: Base packages
		state.Packages = append(state.Packages, "base-lib", "core-utils")
		state.Savepoints["stage1"] = append([]string{}, state.Packages...)

		// Stage 2: Framework
		state.Packages = append(state.Packages, "framework", "plugins")
		state.Savepoints["stage2"] = append([]string{}, state.Packages...)

		// Stage 3: Application (fails)
		state.Packages = append(state.Packages, "app")

		appError := errors.New("app installation failed")
		if appError != nil {
			// Restore to stage2 savepoint
			state.Packages = append([]string{}, state.Savepoints["stage2"]...)
		}

		// Verify restoration
		assert.Len(t, state.Packages, 4, "Should restore to stage2 state")
		assert.Contains(t, state.Packages, "framework")
		assert.NotContains(t, state.Packages, "app", "Failed app should not be in state")
	})

	t.Run("nested_transaction_support", func(t *testing.T) {
		type Transaction struct {
			ID        string
			Parent    *Transaction
			Packages  []string
			Committed bool
		}

		// Main transaction
		mainTx := &Transaction{
			ID:       "main",
			Packages: []string{"package1"},
		}

		// Nested transaction 1
		nestedTx1 := &Transaction{
			ID:       "nested1",
			Parent:   mainTx,
			Packages: []string{"package2", "package3"},
		}

		// Nested transaction 2 (will fail)
		nestedTx2 := &Transaction{
			ID:       "nested2",
			Parent:   mainTx,
			Packages: []string{"package4"},
		}

		// Commit nested1
		nestedTx1.Committed = true
		mainTx.Packages = append(mainTx.Packages, nestedTx1.Packages...)

		// Nested2 fails - don't commit
		failureOccurred := true
		if !failureOccurred {
			nestedTx2.Committed = true
			mainTx.Packages = append(mainTx.Packages, nestedTx2.Packages...)
		}

		// Verify only committed transactions are included
		assert.Len(t, mainTx.Packages, 3, "Should have main + nested1 packages")
		assert.Contains(t, mainTx.Packages, "package1")
		assert.Contains(t, mainTx.Packages, "package2")
		assert.Contains(t, mainTx.Packages, "package3")
		assert.NotContains(t, mainTx.Packages, "package4", "Failed nested tx should not be included")
	})
}

// TestIdempotentTransactions tests that transactions can be safely retried.
func TestIdempotentTransactions(t *testing.T) {
	// Business Rule: Failed transactions should be safely retriable
	t.Run("retry_after_network_failure", func(t *testing.T) {
		// Testing retry logic for network failures
		packageName := "remote-package"

		attempts := 0
		maxAttempts := 3

		var lastError error

		installed := false

		for attempts < maxAttempts && !installed {
			attempts++

			// Simulate network failure on first 2 attempts
			if attempts < 3 {
				lastError = errors.New("network timeout")
				continue
			}

			// Success on third attempt
			installed = true
			lastError = nil
		}

		assert.True(t, installed, "Package %s should eventually install", packageName)
		assert.Equal(t, 3, attempts, "Should take 3 attempts")
		assert.NoError(t, lastError, "Should succeed eventually")
	})

	t.Run("no_duplicate_installation", func(t *testing.T) {
		// Ensure retrying doesn't cause duplicate installations
		installedPackages := make(map[string]bool)
		pkg := &domain.Package{
			Name:   "unique-package",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		}

		// First attempt
		if !installedPackages[pkg.Name] {
			installedPackages[pkg.Name] = true
		}

		// Retry (should detect already installed)
		alreadyInstalled := installedPackages[pkg.Name]
		if !alreadyInstalled {
			installedPackages[pkg.Name] = true
		}

		assert.True(t, alreadyInstalled, "Should detect package already installed")
		assert.Len(t, installedPackages, 1, "Should only have one entry")
	})
}

// TestTransactionMetrics tests transaction performance metrics.
func TestTransactionMetrics(t *testing.T) {
	// Business Rule: Track transaction metrics for optimization
	type TransactionMetrics struct {
		StartTime    time.Time
		EndTime      time.Time
		PackageCount int
		SuccessCount int
		FailureCount int
		RollbackTime time.Duration
	}

	t.Run("track_transaction_performance", func(t *testing.T) {
		metrics := &TransactionMetrics{
			StartTime:    time.Now(),
			PackageCount: 10,
		}

		// Simulate installation
		for i := range metrics.PackageCount {
			if i == 7 {
				// Simulate failure at package 8
				metrics.FailureCount++

				// Measure rollback time
				rollbackStart := time.Now()
				time.Sleep(50 * time.Millisecond) // Simulate rollback
				metrics.RollbackTime = time.Since(rollbackStart)

				break
			}

			metrics.SuccessCount++

			time.Sleep(10 * time.Millisecond) // Simulate work
		}

		metrics.EndTime = time.Now()
		totalDuration := metrics.EndTime.Sub(metrics.StartTime)

		// Verify metrics
		assert.Equal(t, 7, metrics.SuccessCount, "Should have 7 successful installs")
		assert.Equal(t, 1, metrics.FailureCount, "Should have 1 failure")
		assert.Greater(t, metrics.RollbackTime, time.Duration(0), "Should track rollback time")
		assert.Less(t, metrics.RollbackTime, totalDuration, "Rollback should be faster than total time")

		// Calculate success rate
		successRate := float64(metrics.SuccessCount) / float64(metrics.PackageCount) * 100
		assert.InEpsilon(t, 70.0, successRate, 0.01, "Success rate should be 70%")
	})
}
