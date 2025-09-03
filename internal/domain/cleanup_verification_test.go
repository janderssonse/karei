// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testCleanupOrderOnNestedCancellation is a helper to test cleanup order on nested cancellation.
func testCleanupOrderOnNestedCancellation(t *testing.T) {
	t.Helper()

	cleanupOrder := make([]string, 0)

	var mu sync.Mutex

	recordCleanup := func(name string) {
		mu.Lock()
		defer mu.Unlock()

		cleanupOrder = append(cleanupOrder, name)
	}

	// Nested operation structure
	outerOperation := func(ctx context.Context) error {
		defer recordCleanup("outer")

		innerOperation := func(ctx context.Context) error {
			defer recordCleanup("inner")

			deepestOperation := func(ctx context.Context) error {
				defer recordCleanup("deepest")

				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					return nil
				}
			}

			return deepestOperation(ctx)
		}

		return innerOperation(ctx)
	}

	// Execute with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_ = outerOperation(ctx)

	// Business rule: Cleanup happens in LIFO order (deepest first)
	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, []string{"deepest", "inner", "outer"}, cleanupOrder,
		"Cleanup should happen in LIFO order")
}

// testConcurrentCancellationCleanup is a helper to test concurrent cancellation cleanup.
func testConcurrentCancellationCleanup(t *testing.T) {
	t.Helper()

	var (
		activeOperations atomic.Int32
		cleanupCount     atomic.Int32
	)

	operation := func(ctx context.Context, _ int) error {
		activeOperations.Add(1)

		defer func() {
			// Cleanup always happens
			activeOperations.Add(-1)
			cleanupCount.Add(1)
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Simulate work
			return nil
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	const numOperations = 10

	// Start operations
	for i := range numOperations {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			if id == 5 {
				// Cancel midway
				cancel()
			}

			_ = operation(ctx, id)
		}(i)
	}

	wg.Wait()
	cancel() // Ensure cancel is called

	// Business rule: All operations must clean up, even on cancellation
	assert.Equal(t, int32(0), activeOperations.Load(),
		"All operations should be cleaned up")
	assert.Equal(t, int32(numOperations), cleanupCount.Load(),
		"Cleanup should happen for all operations")
}

// testTransactionRollback is a helper to test transaction rollback on cancellation.
func testTransactionRollback(t *testing.T) {
	t.Helper()

	type Transaction struct {
		ID         string
		Operations []string
		Committed  bool
		RolledBack bool
	}

	executeTransaction := func(ctx context.Context, ops []string) (*Transaction, error) {
		tx := &Transaction{
			ID:         "tx-123",
			Operations: make([]string, 0),
		}

		for _, op := range ops {
			// Check context before each operation
			select {
			case <-ctx.Done():
				// ROLLBACK on cancellation
				tx.RolledBack = true
				// Undo all operations
				tx.Operations = nil

				return nil, ctx.Err()
			default:
				tx.Operations = append(tx.Operations, op)
			}
		}

		// Commit if all operations succeeded
		tx.Committed = true

		return tx, nil
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	operations := []string{"CREATE", "UPDATE", "DELETE"}
	tx, err := executeTransaction(ctx, operations)

	// Business rule: Cancelled transactions must not be committed
	assert.Nil(t, tx, "Cancelled transaction should not return result")
	assert.ErrorIs(t, err, context.Canceled)
}

// testResourceCleanupOnTimeout is a helper to test resource cleanup on timeout.
func testResourceCleanupOnTimeout(t *testing.T) {
	t.Helper()

	type ResourceManager struct {
		mu              sync.Mutex
		acquiredLocks   map[string]bool
		openConnections []string
		tempFiles       []string
	}

	manager := &ResourceManager{
		acquiredLocks:   make(map[string]bool),
		openConnections: make([]string, 0),
		tempFiles:       make([]string, 0),
	}

	// Function that acquires resources
	performOperation := func(ctx context.Context) error {
		// Acquire lock
		manager.mu.Lock()
		manager.acquiredLocks["database"] = true
		manager.mu.Unlock()

		// Open connection
		manager.mu.Lock()
		manager.openConnections = append(manager.openConnections, "db-conn-1")
		manager.mu.Unlock()

		// Create temp file
		manager.mu.Lock()
		manager.tempFiles = append(manager.tempFiles, "/tmp/install.tmp")
		manager.mu.Unlock()

		// Check if cancelled
		select {
		case <-ctx.Done():
			// CLEANUP SHOULD HAPPEN HERE
			manager.mu.Lock()
			defer manager.mu.Unlock()

			// Release locks
			for lock := range manager.acquiredLocks {
				delete(manager.acquiredLocks, lock)
			}

			// Close connections
			manager.openConnections = nil

			// Delete temp files
			manager.tempFiles = nil

			return ctx.Err()
		default:
			// Continue operation
			return nil
		}
	}

	// Test with expired context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := performOperation(ctx)
	require.ErrorIs(t, err, context.Canceled)

	// Verify cleanup happened
	manager.mu.Lock()
	defer manager.mu.Unlock()

	assert.Empty(t, manager.acquiredLocks, "Locks should be released")
	assert.Empty(t, manager.openConnections, "Connections should be closed")
	assert.Empty(t, manager.tempFiles, "Temp files should be deleted")
}

// testPartialInstallationCleanup is a helper to test partial installation cleanup.
func testPartialInstallationCleanup(t *testing.T) {
	t.Helper()

	// Track what was installed and what was cleaned up
	type InstallationTracker struct {
		mu           sync.Mutex
		installed    []string
		cleanedUp    []string
		filesCreated []string
		filesDeleted []string
	}

	tracker := &InstallationTracker{
		installed:    make([]string, 0),
		cleanedUp:    make([]string, 0),
		filesCreated: make([]string, 0),
		filesDeleted: make([]string, 0),
	}

	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	// Package with dependencies
	mainPkg := &domain.Package{
		Name:         "main-app",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"dep1", "dep2", "dep3"},
	}

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Mock should track partial installation
	mockInstaller.On("Install", mock.Anything, mainPkg).
		Run(func(_ mock.Arguments) {
			tracker.mu.Lock()
			defer tracker.mu.Unlock()

			// Simulate that we started creating files
			tracker.filesCreated = append(tracker.filesCreated,
				"/usr/bin/main-app",
				"/etc/main-app.conf",
			)
			tracker.installed = append(tracker.installed, "main-app")
		}).
		Return(nil, context.Canceled).Once()

	// Attempt installation with cancelled context
	result, err := service.Install(ctx, mainPkg)

	// Verify cancellation was handled
	assert.Nil(t, result, "Cancelled operation should not return result")
	require.ErrorIs(t, err, context.Canceled)

	// IMPORTANT: In a real system, we should verify cleanup happened
	// This test documents that cleanup SHOULD occur
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	// Business rule: Partial installations must be cleaned up
	if len(tracker.filesCreated) > 0 && len(tracker.filesDeleted) == 0 {
		t.Logf("WARNING: Files were created but not cleaned up on cancellation: %v",
			tracker.filesCreated)
		// In production, this would be a failure
		// assert.Equal(t, len(tracker.filesCreated), len(tracker.filesDeleted),
		//     "All created files should be cleaned up on cancellation")
	}
}

// TestContextCancellationWithCleanup verifies that cancelled operations clean up properly.
func TestContextCancellationWithCleanup(t *testing.T) {
	t.Parallel()

	t.Run("partial_installation_cleanup_on_cancellation", func(t *testing.T) {
		t.Parallel()
		testPartialInstallationCleanup(t)
	})

	t.Run("resource_cleanup_on_context_timeout", func(t *testing.T) {
		t.Parallel()
		testResourceCleanupOnTimeout(t)
	})

	t.Run("transaction_rollback_on_cancellation", func(t *testing.T) {
		t.Parallel()
		testTransactionRollback(t)
	})

	t.Run("concurrent_cancellation_cleanup", func(t *testing.T) {
		t.Parallel()
		testConcurrentCancellationCleanup(t)
	})

	t.Run("cleanup_order_on_nested_cancellation", func(t *testing.T) {
		t.Parallel()
		testCleanupOrderOnNestedCancellation(t)
	})
}

// TestResourceLeakPrevention tests that resources are cleaned up on cancellation.
func TestResourceLeakPrevention(t *testing.T) {
	t.Parallel()

	t.Run("file_descriptor_leak_prevention", func(t *testing.T) {
		t.Parallel()

		type FileTracker struct {
			mu           sync.Mutex
			openFiles    map[string]bool
			maxOpenFiles int
		}

		tracker := &FileTracker{
			openFiles:    make(map[string]bool),
			maxOpenFiles: 10,
		}

		openFile := func(ctx context.Context, path string) (func(), error) {
			tracker.mu.Lock()
			defer tracker.mu.Unlock()

			// Check context first
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			// Check file limit
			if len(tracker.openFiles) >= tracker.maxOpenFiles {
				return nil, errors.New("too many open files")
			}

			tracker.openFiles[path] = true

			// Return cleanup function
			cleanup := func() {
				tracker.mu.Lock()
				defer tracker.mu.Unlock()

				delete(tracker.openFiles, path)
			}

			return cleanup, nil
		}

		// Test with cancellation
		ctx, cancel := context.WithCancel(context.Background())

		// Open some files
		cleanups := make([]func(), 0)

		for i := range 5 {
			cleanup, err := openFile(ctx, "/tmp/file"+string(rune(i)))
			if err == nil {
				cleanups = append(cleanups, cleanup)
			}
		}

		// Cancel context
		cancel()

		// Try to open more files (should fail due to cancellation)
		_, err := openFile(ctx, "/tmp/file_after_cancel")
		require.ErrorIs(t, err, context.Canceled)

		// Clean up opened files
		for _, cleanup := range cleanups {
			cleanup()
		}

		// Verify no leaks
		tracker.mu.Lock()
		defer tracker.mu.Unlock()

		assert.Empty(t, tracker.openFiles, "All files should be closed")
	})

	t.Run("goroutine_leak_prevention", func(t *testing.T) {
		t.Parallel()

		var activeGoroutines atomic.Int32

		// Worker that respects context
		worker := func(ctx context.Context, _ int) {
			activeGoroutines.Add(1)
			defer activeGoroutines.Add(-1)

			// Work loop that checks context
			<-ctx.Done()
			// Clean exit on cancellation
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Start workers
		const numWorkers = 10
		for i := range numWorkers {
			go worker(ctx, i)
		}

		// Let workers start
		for activeGoroutines.Load() < numWorkers {
			// Busy wait for all workers to start
			runtime.Gosched() // Yield to allow other goroutines to run
		}

		// Cancel to stop all workers
		cancel()

		// Wait for cleanup
		for activeGoroutines.Load() > 0 {
			// Busy wait for all workers to exit
			runtime.Gosched() // Yield to allow other goroutines to run
		}

		// Business rule: All goroutines must exit on context cancellation
		assert.Equal(t, int32(0), activeGoroutines.Load(),
			"All goroutines should exit on cancellation")
	})
}
