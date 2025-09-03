// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"sync"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstallationIdempotency tests that installing a package multiple times is safe
// Business Rule: Installing an already installed package should not fail.
func TestInstallationIdempotency(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()
	pkg := testutil.CreateValidPackage("vim")

	// First installation succeeds
	mockInstaller.On("Install", ctx, pkg).
		Return(&domain.InstallationResult{
			Package: pkg,
			Success: true,
		}, nil).Once()

	// Second installation returns AlreadyInstalled
	mockInstaller.On("Install", ctx, pkg).
		Return(nil, domain.ErrAlreadyInstalled).Once()

	// Third installation also returns AlreadyInstalled
	mockInstaller.On("Install", ctx, pkg).
		Return(nil, domain.ErrAlreadyInstalled).Once()

	// First install should succeed
	result1, err1 := service.Install(ctx, pkg)
	require.NoError(t, err1)
	assert.True(t, result1.Success)

	// Second install should return AlreadyInstalled (not a failure)
	result2, err2 := service.Install(ctx, pkg)
	require.ErrorIs(t, err2, domain.ErrAlreadyInstalled)
	assert.Nil(t, result2)

	// Third install should also return AlreadyInstalled
	result3, err3 := service.Install(ctx, pkg)
	require.ErrorIs(t, err3, domain.ErrAlreadyInstalled)
	assert.Nil(t, result3)

	// Business rule: Multiple installs should not corrupt system
	mockInstaller.AssertExpectations(t)
}

// TestServiceThreadSafety verifies the service can be called concurrently without data races.
func TestServiceThreadSafety(_ *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()

	// Set up different packages to avoid mock collision
	packages := []*domain.Package{
		testutil.CreateValidPackage("git"),
		testutil.CreateValidPackage("vim"),
		testutil.CreateValidPackage("curl"),
	}

	// Mock each package to succeed
	for _, pkg := range packages {
		mockInstaller.On("Install", ctx, pkg).
			Return(&domain.InstallationResult{Package: pkg, Success: true}, nil).Maybe()
	}

	// Launch concurrent operations on different packages
	var wg sync.WaitGroup
	for i := range 30 {
		wg.Add(1)

		pkg := packages[i%len(packages)]
		go func(p *domain.Package) {
			defer wg.Done()
			// This tests that the service itself is thread-safe
			_, _ = service.Install(ctx, p)
		}(pkg)
	}

	wg.Wait()

	// If we get here without panic or race conditions, the service is thread-safe
	// The Go race detector will catch any data races during test execution
}

// TestRemovalIdempotency tests that removing a package multiple times is safe
// Business Rule: Removing a non-existent package should not corrupt state.
func TestRemovalIdempotency(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()
	pkg := testutil.CreateValidPackage("curl")

	// First removal succeeds
	mockInstaller.On("Remove", ctx, pkg).
		Return(&domain.InstallationResult{
			Package: pkg,
			Success: true,
		}, nil).Once()

	// Subsequent removals return NotInstalled
	mockInstaller.On("Remove", ctx, pkg).
		Return(nil, domain.ErrNotInstalled).Twice()

	// First removal should succeed
	result1, err1 := service.Remove(ctx, pkg)
	require.NoError(t, err1)
	assert.True(t, result1.Success)

	// Second removal should return NotInstalled
	result2, err2 := service.Remove(ctx, pkg)
	require.ErrorIs(t, err2, domain.ErrNotInstalled)
	assert.Nil(t, result2)

	// Third removal should also return NotInstalled
	result3, err3 := service.Remove(ctx, pkg)
	require.ErrorIs(t, err3, domain.ErrNotInstalled)
	assert.Nil(t, result3)

	mockInstaller.AssertExpectations(t)
}

// TestInstallRemoveInstallCycle tests that packages can be reinstalled after removal
// Business Rule: Install -> Remove -> Install should work correctly.
func TestInstallRemoveInstallCycle(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()
	pkg := testutil.CreateValidPackage("htop")

	// Setup expectations for the cycle
	// First install
	mockInstaller.On("Install", ctx, pkg).
		Return(&domain.InstallationResult{
			Package: pkg,
			Success: true,
		}, nil).Once()

	// Remove
	mockInstaller.On("Remove", ctx, pkg).
		Return(&domain.InstallationResult{
			Package: pkg,
			Success: true,
		}, nil).Once()

	// Second install
	mockInstaller.On("Install", ctx, pkg).
		Return(&domain.InstallationResult{
			Package: pkg,
			Success: true,
		}, nil).Once()

	// Execute the cycle
	// Install
	result1, err1 := service.Install(ctx, pkg)
	require.NoError(t, err1)
	assert.True(t, result1.Success)

	// Remove
	result2, err2 := service.Remove(ctx, pkg)
	require.NoError(t, err2)
	assert.True(t, result2.Success)

	// Install again
	result3, err3 := service.Install(ctx, pkg)
	require.NoError(t, err3)
	assert.True(t, result3.Success)

	mockInstaller.AssertExpectations(t)
}

// TestConcurrentDifferentPackages tests installing different packages concurrently
// Business Rule: Different packages should be installable in parallel without interference.
func TestConcurrentDifferentPackages(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()

	packages := []*domain.Package{
		testutil.CreateValidPackage("vim"),
		testutil.CreateValidPackage("git"),
		testutil.CreateValidPackage("curl"),
		testutil.CreateValidPackage("htop"),
		testutil.CreateValidPackage("tmux"),
	}

	// Setup mock expectations for each package
	for _, pkg := range packages {
		mockInstaller.On("Install", ctx, pkg).
			Return(&domain.InstallationResult{
				Package: pkg,
				Success: true,
			}, nil).Once()
	}

	// Install all packages concurrently
	var wg sync.WaitGroup

	errors := make(chan error, len(packages))

	for _, pkg := range packages {
		wg.Add(1)

		go func(p *domain.Package) {
			defer wg.Done()

			result, err := service.Install(ctx, p)
			switch {
			case err != nil:
				errors <- err
			case !result.Success:
				errors <- assert.AnError
			default:
				errors <- nil
			}
		}(pkg)
	}

	wg.Wait()
	close(errors)

	// All installations should succeed
	errorCount := 0

	for err := range errors {
		if err != nil {
			errorCount++

			t.Errorf("Installation failed: %v", err)
		}
	}

	assert.Equal(t, 0, errorCount, "All concurrent installations should succeed")
	mockInstaller.AssertExpectations(t)
}

// TestBatchOperationPartialFailure tests that batch operations handle partial failures correctly
// Business Rule: If some packages fail, others should still be attempted.
func TestBatchOperationPartialFailure(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()

	packages := []*domain.Package{
		testutil.CreateValidPackage("success1"),
		testutil.CreateValidPackage("fail1"),
		testutil.CreateValidPackage("success2"),
		testutil.CreateValidPackage("fail2"),
		testutil.CreateValidPackage("success3"),
	}

	// Setup mixed success/failure expectations
	for i, pkg := range packages {
		if i%2 == 0 {
			// Even indices succeed
			mockInstaller.On("Install", ctx, pkg).
				Return(&domain.InstallationResult{
					Package: pkg,
					Success: true,
				}, nil).Once()
		} else {
			// Odd indices fail
			mockInstaller.On("Install", ctx, pkg).
				Return(nil, domain.ErrNetworkFailure).Once()
		}
	}

	// Install all packages
	successCount := 0
	failCount := 0

	for _, pkg := range packages {
		result, err := service.Install(ctx, pkg)
		if err != nil {
			failCount++

			require.ErrorIs(t, err, domain.ErrNetworkFailure)
		} else {
			successCount++

			assert.True(t, result.Success)
		}
	}

	// Business rule: Partial failures don't stop other installations
	assert.Equal(t, 3, successCount, "Three packages should succeed")
	assert.Equal(t, 2, failCount, "Two packages should fail")
	mockInstaller.AssertExpectations(t)
}
