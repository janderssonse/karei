// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestInstallRespectsContextCancellation tests that operations respect context cancellation
// without timing dependencies.
func TestInstallRespectsContextCancellation(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	validPackage := testutil.CreateValidPackage("test-app")

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Mock should check context and return cancelled error
	mockInstaller.On("Install", mock.Anything, validPackage).
		Return(nil, context.Canceled)

	result, err := service.Install(ctx, validPackage)

	// Business rule: cancelled context prevents operation
	assert.Nil(t, result)
	require.ErrorIs(t, err, context.Canceled)
	mockInstaller.AssertExpectations(t)
}

// TestRemoveRespectsContextCancellation tests remove operation respects context.
func TestRemoveRespectsContextCancellation(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	validPackage := testutil.CreateValidPackage("test-app")

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Setup mock to handle the call if reached
	mockInstaller.On("Remove", mock.Anything, validPackage).
		Return(nil, context.Canceled).Maybe()

	result, err := service.Remove(ctx, validPackage)

	// Business rule: operations should handle cancelled context
	assert.Nil(t, result)
	assert.Error(t, err)
}

// TestListPackagesRespectsContext tests that list operations respect context.
func TestListPackagesRespectsContext(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Mock should return context error
	mockInstaller.On("List", mock.Anything).
		Return(nil, context.Canceled)

	packages, err := service.List(ctx)

	assert.Nil(t, packages)
	require.ErrorIs(t, err, context.Canceled)
	mockInstaller.AssertExpectations(t)
}

// TestMultiplePackageOperations tests handling multiple packages
// This replaces the concurrent test with a simpler, more focused test.
func TestMultiplePackageOperations(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	packages := []*domain.Package{
		{Name: "app1", Method: domain.MethodAPT, Source: "source1"},
		{Name: "app2", Method: domain.MethodSnap, Source: "source2"},
		{Name: "app3", Method: domain.MethodFlatpak, Source: "source3"},
	}

	ctx := context.Background()

	// Test that service can handle multiple packages sequentially
	for _, pkg := range packages {
		result := &domain.InstallationResult{
			Package: pkg,
			Success: true,
		}
		mockInstaller.On("Install", ctx, pkg).
			Return(result, nil).Once()

		// Install package
		actualResult, err := service.Install(ctx, pkg)
		require.NoError(t, err)
		assert.True(t, actualResult.Success)
		assert.Equal(t, pkg, actualResult.Package)
	}

	mockInstaller.AssertExpectations(t)
}

// TestOperationWithContextValues tests that context values are preserved.
func TestOperationWithContextValues(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	validPackage := testutil.CreateValidPackage("test-app")

	// Create context with values
	type contextKey string

	const userKey contextKey = "user"

	ctx := context.WithValue(context.Background(), userKey, "test-user")

	expectedResult := &domain.InstallationResult{
		Package: validPackage,
		Success: true,
	}

	// Verify context is passed through
	mockInstaller.On("Install", mock.MatchedBy(func(c context.Context) bool {
		return c.Value(userKey) == "test-user"
	}), validPackage).Return(expectedResult, nil)

	result, err := service.Install(ctx, validPackage)

	require.NoError(t, err)
	assert.True(t, result.Success) // Verify operation succeeded with context values
	mockInstaller.AssertExpectations(t)
}

// TestInstallWithDependencyChain tests handling packages with dependencies.
func TestInstallWithDependencyChain(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	// Package with dependencies
	mainPackage := &domain.Package{
		Name:         "main-app",
		Method:       domain.MethodAPT,
		Source:       "main-app",
		Dependencies: []string{"lib1", "lib2"},
	}

	ctx := context.Background()

	// The port should handle dependency resolution
	expectedResult := &domain.InstallationResult{
		Package: mainPackage,
		Success: true,
		Output:  "Installed with dependencies",
	}

	mockInstaller.On("Install", ctx, mainPackage).Return(expectedResult, nil)

	result, err := service.Install(ctx, mainPackage)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "dependencies")
	mockInstaller.AssertExpectations(t)
}

// TestRemoveHandlesNotInstalledError tests removing non-existent packages.
func TestRemoveHandlesNotInstalledError(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	validPackage := testutil.CreateValidPackage("not-installed-app")

	ctx := context.Background()

	// Mock returns not installed error
	mockInstaller.On("Remove", ctx, validPackage).
		Return(nil, domain.ErrNotInstalled)

	result, err := service.Remove(ctx, validPackage)

	// Business rule: removing non-installed package is an error
	assert.Nil(t, result)
	require.ErrorIs(t, err, domain.ErrNotInstalled)
	mockInstaller.AssertExpectations(t)
}
