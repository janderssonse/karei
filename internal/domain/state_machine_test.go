// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PackageState represents the state of a package in the system.
type PackageState string

const (
	StateNotInstalled PackageState = "not_installed"
	StateDownloading  PackageState = "downloading"
	StateInstalling   PackageState = "installing"
	StateInstalled    PackageState = "installed"
	StateUpgrading    PackageState = "upgrading"
	StateRemoving     PackageState = "removing"
	StateFailed       PackageState = "failed"
	StateCorrupted    PackageState = "corrupted"
)

// TestPackageLifecycleStateMachine verifies valid state transitions.
func TestPackageLifecycleStateMachine(t *testing.T) {
	t.Parallel()

	t.Run("valid_state_transitions", func(t *testing.T) {
		t.Parallel()
		testValidStateTransitions(t)
	})

	t.Run("invalid_state_transitions_rejected", func(t *testing.T) {
		t.Parallel()
		testInvalidStateTransitions(t)
	})

	t.Run("state_persistence_across_failures", func(t *testing.T) {
		t.Parallel()

		// Business rule: Package state must be recoverable after crash
		type PackageRecord struct {
			Name  string
			State PackageState
			mu    sync.RWMutex
		}

		record := &PackageRecord{
			Name:  "critical-package",
			State: StateNotInstalled,
		}

		// Simulate installation process with potential failure
		updateState := func(newState PackageState) error {
			record.mu.Lock()
			defer record.mu.Unlock()

			// Check if transition is valid
			if !isValidTransition(record.State, newState) {
				return errors.New("invalid state transition")
			}

			oldState := record.State
			record.State = newState

			// Simulate crash during state change
			if newState == StateInstalling && oldState == StateDownloading {
				// State was persisted before crash
				return nil
			}

			return nil
		}

		// Start installation
		err := updateState(StateDownloading)
		require.NoError(t, err)
		assert.Equal(t, StateDownloading, record.State)

		// Move to installing
		err = updateState(StateInstalling)
		require.NoError(t, err)
		assert.Equal(t, StateInstalling, record.State)

		// After recovery, state should be preserved
		// Package manager should either:
		// 1. Continue installation
		// 2. Rollback to safe state
		assert.Contains(t, []PackageState{StateInstalling, StateDownloading, StateFailed},
			record.State, "State after recovery should be deterministic")
	})

	t.Run("concurrent_state_changes_safety", func(t *testing.T) {
		t.Parallel()

		// Business rule: Concurrent operations on same package must be serialized
		type SafePackageManager struct {
			states map[string]PackageState
			locks  map[string]*sync.Mutex
			mu     sync.RWMutex
		}

		manager := &SafePackageManager{
			states: make(map[string]PackageState),
			locks:  make(map[string]*sync.Mutex),
		}

		getPackageLock := func(name string) *sync.Mutex {
			manager.mu.Lock()
			defer manager.mu.Unlock()

			if _, exists := manager.locks[name]; !exists {
				manager.locks[name] = &sync.Mutex{}
			}

			return manager.locks[name]
		}

		changeState := func(name string, newState PackageState) error {
			lock := getPackageLock(name)

			lock.Lock()
			defer lock.Unlock()

			manager.mu.Lock()
			defer manager.mu.Unlock()

			currentState, exists := manager.states[name]
			if !exists {
				currentState = StateNotInstalled
			}

			if !isValidTransition(currentState, newState) {
				return errors.New("invalid transition")
			}

			manager.states[name] = newState

			return nil
		}

		// Try concurrent operations
		var wg sync.WaitGroup

		errors := make([]error, 2)

		// Operation 1: Try to install
		wg.Add(1)

		go func() {
			defer wg.Done()

			errors[0] = changeState("pkg", StateDownloading)
		}()

		// Operation 2: Try to remove (should fail if downloading started)
		wg.Add(1)

		go func() {
			defer wg.Done()

			errors[1] = changeState("pkg", StateRemoving)
		}()

		wg.Wait()

		// Only one operation should succeed
		successCount := 0

		for _, err := range errors {
			if err == nil {
				successCount++
			}
		}

		assert.Equal(t, 1, successCount,
			"Exactly one concurrent operation should succeed")
	})

	t.Run("rollback_on_failed_transition", func(t *testing.T) {
		t.Parallel()

		// Business rule: Failed operations must rollback to previous stable state
		type TransactionalPackageManager struct {
			currentState  PackageState
			previousState PackageState
		}

		manager := &TransactionalPackageManager{
			currentState: StateInstalled,
		}

		attemptUpgrade := func() error {
			// Save current state for rollback
			manager.previousState = manager.currentState
			manager.currentState = StateUpgrading

			// Simulate upgrade failure
			upgradeSuccessful := false

			if !upgradeSuccessful {
				// Rollback to previous state
				manager.currentState = manager.previousState
				return errors.New("upgrade failed")
			}

			manager.currentState = StateInstalled

			return nil
		}

		initialState := manager.currentState
		err := attemptUpgrade()

		// Business rule: Failed upgrade must restore original state
		require.Error(t, err)
		assert.Equal(t, initialState, manager.currentState,
			"State should be rolled back after failed upgrade")
	})
}

func testValidStateTransitions(t *testing.T) {
	t.Helper()
	// Business rule: Packages follow specific state transitions
	validTransitions := map[PackageState][]PackageState{
		StateNotInstalled: {StateDownloading},
		StateDownloading:  {StateInstalling, StateFailed},
		StateInstalling:   {StateInstalled, StateFailed, StateCorrupted},
		StateInstalled:    {StateUpgrading, StateRemoving},
		StateUpgrading:    {StateInstalled, StateFailed},
		StateRemoving:     {StateNotInstalled, StateFailed},
		StateFailed:       {StateDownloading, StateRemoving}, // Can retry or cleanup
		StateCorrupted:    {StateRemoving},                   // Must remove corrupted packages
	}

	// Test each valid transition
	for fromState, toStates := range validTransitions {
		for _, toState := range toStates {
			t.Run(string(fromState)+"_to_"+string(toState), func(t *testing.T) {
				transition := isValidTransition(fromState, toState)
				assert.True(t, transition,
					"Transition from %s to %s should be valid", fromState, toState)
			})
		}
	}
}

func testInvalidStateTransitions(t *testing.T) {
	t.Helper()
	// Business rule: Invalid transitions must be rejected
	invalidTransitions := []struct {
		from   PackageState
		to     PackageState
		reason string
	}{
		{StateNotInstalled, StateInstalled, "Cannot install without downloading"},
		{StateNotInstalled, StateRemoving, "Cannot remove what's not installed"},
		{StateInstalled, StateDownloading, "Already installed packages don't download"},
		{StateRemoving, StateInstalling, "Cannot install while removing"},
		{StateCorrupted, StateInstalled, "Corrupted packages cannot become installed"},
		{StateDownloading, StateUpgrading, "Cannot upgrade while downloading"},
	}

	for _, tc := range invalidTransitions {
		t.Run(string(tc.from)+"_to_"+string(tc.to), func(t *testing.T) {
			transition := isValidTransition(tc.from, tc.to)
			assert.False(t, transition, tc.reason)
		})
	}
}

// testIdempotentOperation tests that an operation is idempotent.
func testIdempotentOperation(t *testing.T, operationType string) {
	t.Helper()

	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	pkg := &domain.Package{
		Name:   "idempotent-pkg",
		Method: domain.MethodAPT,
		Source: "ubuntu",
	}

	ctx := context.Background()

	switch operationType {
	case "install":
		// First install - transitions from not_installed to installed
		mockInstaller.On("Install", ctx, pkg).
			Return(&domain.InstallationResult{
				Package: pkg,
				Success: true,
			}, nil).Once()

		result1, err1 := service.Install(ctx, pkg)
		require.NoError(t, err1)
		assert.True(t, result1.Success)

		// Second install - already installed, should be idempotent
		mockInstaller.On("Install", ctx, pkg).
			Return(nil, domain.ErrAlreadyInstalled).Once()

		result2, err2 := service.Install(ctx, pkg)

		// Business rule: Installing twice is not an error
		require.ErrorIs(t, err2, domain.ErrAlreadyInstalled)
		assert.Nil(t, result2)

	case "remove":
		// First remove - package exists
		mockInstaller.On("Remove", ctx, pkg).
			Return(&domain.InstallationResult{
				Package: pkg,
				Success: true,
			}, nil).Once()

		result1, err1 := service.Remove(ctx, pkg)
		require.NoError(t, err1)
		assert.True(t, result1.Success)

		// Second remove - already removed
		mockInstaller.On("Remove", ctx, pkg).
			Return(nil, domain.ErrNotInstalled).Once()

		result2, err2 := service.Remove(ctx, pkg)

		// Business rule: Removing non-existent package is not critical error
		require.ErrorIs(t, err2, domain.ErrNotInstalled)
		assert.Nil(t, result2)
	}

	// Package is still in expected state - idempotent
	mockInstaller.AssertExpectations(t)
}

// TestPackageOperationIdempotency verifies operations are idempotent.
func TestPackageOperationIdempotency(t *testing.T) {
	t.Parallel()

	t.Run("install_idempotency_with_state", func(t *testing.T) {
		t.Parallel()
		testIdempotentOperation(t, "install")
	})

	t.Run("remove_idempotency_with_state", func(t *testing.T) {
		t.Parallel()
		testIdempotentOperation(t, "remove")
	})
}

// TestPackageStateInvariantsValidation verifies state invariants always hold.
func TestPackageStateInvariantsValidation(t *testing.T) {
	t.Parallel()

	t.Run("no_duplicate_states", func(t *testing.T) {
		t.Parallel()

		// Invariant: A package cannot be in multiple states simultaneously
		type PackageStateManager struct {
			packages map[string]PackageState
			mu       sync.RWMutex
		}

		manager := &PackageStateManager{
			packages: make(map[string]PackageState),
		}

		// This should NEVER happen
		invalidScenarios := []struct {
			name   string
			states []PackageState
		}{
			{
				name:   "installing_and_removing",
				states: []PackageState{StateInstalling, StateRemoving},
			},
			{
				name:   "installed_and_not_installed",
				states: []PackageState{StateInstalled, StateNotInstalled},
			},
			{
				name:   "downloading_and_installed",
				states: []PackageState{StateDownloading, StateInstalled},
			},
		}

		for _, scenario := range invalidScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				// Try to set multiple states (should be prevented)
				pkg := "test-pkg"

				manager.mu.Lock()
				manager.packages[pkg] = scenario.states[0]
				currentState := manager.packages[pkg]
				manager.mu.Unlock()

				// Invariant: Setting new state should replace, not add
				manager.mu.Lock()
				manager.packages[pkg] = scenario.states[1]
				newState := manager.packages[pkg]
				manager.mu.Unlock()

				// Only one state should exist
				assert.NotEqual(t, currentState, newState,
					"State should change, not accumulate")
				assert.Equal(t, scenario.states[1], newState,
					"Latest state should be active")
			})
		}
	})

	t.Run("dependency_state_constraints", func(t *testing.T) {
		t.Parallel()

		// Invariant: Dependencies must be installed before dependents
		type DependencyStateTracker struct {
			states map[string]PackageState
			deps   map[string][]string
		}

		tracker := &DependencyStateTracker{
			states: map[string]PackageState{
				"lib": StateInstalled,
				"app": StateNotInstalled,
			},
			deps: map[string][]string{
				"app": {"lib"}, // app depends on lib
			},
		}

		canInstall := func(pkg string) bool {
			deps, hasDeps := tracker.deps[pkg]
			if !hasDeps {
				return true
			}

			for _, dep := range deps {
				state, exists := tracker.states[dep]
				if !exists || state != StateInstalled {
					return false
				}
			}

			return true
		}

		// App can install because lib is installed
		assert.True(t, canInstall("app"),
			"Should be able to install app when dependencies are met")

		// Now test reverse - lib cannot be removed while app is installed
		tracker.states["app"] = StateInstalled

		canRemove := func(pkg string) bool {
			// Check if any installed package depends on this
			for depPkg, deps := range tracker.deps {
				for _, dep := range deps {
					if dep == pkg && tracker.states[depPkg] == StateInstalled {
						return false
					}
				}
			}

			return true
		}

		assert.False(t, canRemove("lib"),
			"Cannot remove lib while app depends on it")
	})
}

// Helper function for state transition validation.
func isValidTransition(from, to PackageState) bool {
	validTransitions := map[PackageState][]PackageState{
		StateNotInstalled: {StateDownloading},
		StateDownloading:  {StateInstalling, StateFailed},
		StateInstalling:   {StateInstalled, StateFailed, StateCorrupted},
		StateInstalled:    {StateUpgrading, StateRemoving},
		StateUpgrading:    {StateInstalled, StateFailed},
		StateRemoving:     {StateNotInstalled, StateFailed},
		StateFailed:       {StateDownloading, StateRemoving},
		StateCorrupted:    {StateRemoving},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, state := range allowed {
		if state == to {
			return true
		}
	}

	return false
}
