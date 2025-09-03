// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestPackageValidation tests the business rule that packages must have required fields.
func TestPackageValidation(t *testing.T) {
	tests := []struct {
		name     string
		pkg      *domain.Package
		expected bool
	}{
		{
			name: "valid package with all required fields",
			pkg: &domain.Package{
				Name:   "test-app",
				Method: domain.MethodAPT,
				Source: "test-source",
			},
			expected: true,
		},
		{
			name: "invalid package missing name",
			pkg: &domain.Package{
				Method: domain.MethodAPT,
				Source: "test-source",
			},
			expected: false,
		},
		{
			name: "invalid package missing method",
			pkg: &domain.Package{
				Name:   "test-app",
				Source: "test-source",
			},
			expected: false,
		},
		{
			name: "invalid package missing source",
			pkg: &domain.Package{
				Name:   "test-app",
				Method: domain.MethodAPT,
			},
			expected: false,
		},
		{
			name:     "empty package is invalid",
			pkg:      &domain.Package{},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// This tests actual business logic: validation rules
			assert.Equal(t, tc.expected, tc.pkg.IsValid())
		})
	}
}

// TestPackageValidationForDifferentMethods validates method-specific requirements.
func TestPackageValidationForDifferentMethods(t *testing.T) {
	tests := []struct {
		name        string
		pkg         *domain.Package
		shouldValid bool
		description string
	}{
		{
			name: "APT package with standard source",
			pkg: &domain.Package{
				Name:   "vim",
				Method: domain.MethodAPT,
				Source: "vim",
			},
			shouldValid: true,
			description: "Standard APT packages should be valid",
		},
		{
			name: "GitHub package with repo format source",
			pkg: &domain.Package{
				Name:   "tool",
				Method: domain.MethodGitHub,
				Source: "owner/repo",
			},
			shouldValid: true,
			description: "GitHub packages need owner/repo format",
		},
		{
			name: "Snap package with snap name",
			pkg: &domain.Package{
				Name:   "code",
				Method: domain.MethodSnap,
				Source: "code",
			},
			shouldValid: true,
			description: "Snap packages use snap store names",
		},
		{
			name: "Script package with URL",
			pkg: &domain.Package{
				Name:   "custom-install",
				Method: domain.MethodScript,
				Source: "https://example.com/install.sh",
			},
			shouldValid: true,
			description: "Script packages should have script location",
		},
		{
			name: "Binary package with download URL",
			pkg: &domain.Package{
				Name:   "binary-tool",
				Method: domain.MethodBinary,
				Source: "https://releases.example.com/tool.tar.gz",
			},
			shouldValid: true,
			description: "Binary packages need download URLs",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.shouldValid, tc.pkg.IsValid(), tc.description)
		})
	}
}

// TestPackageServiceInstallRejectsInvalidPackages tests the business rule that invalid packages are rejected.
func TestPackageServiceInstallRejectsInvalidPackages(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	invalidPackage := testutil.CreateInvalidPackage()

	ctx := context.Background()
	result, err := service.Install(ctx, invalidPackage)

	// Business rule: Invalid packages must be rejected with specific error
	assert.Nil(t, result)
	require.ErrorIs(t, err, domain.ErrInvalidPackage)

	// Verify installer was never called (port boundary not crossed for invalid input)
	mockInstaller.AssertNotCalled(t, "Install", mock.Anything, mock.Anything)
}

// TestPackageServiceInstallDelegatesValidPackagesToPort tests that valid packages are passed to the port.
func TestPackageServiceInstallDelegatesValidPackagesToPort(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	validPackage := testutil.CreateValidPackage("test-app")

	expectedResult := &domain.InstallationResult{
		Package: validPackage,
		Success: true,
	}

	// Setup mock expectation at port boundary
	mockInstaller.On("Install", mock.Anything, validPackage).Return(expectedResult, nil)

	ctx := context.Background()
	result, err := service.Install(ctx, validPackage)

	// Verify business logic: valid package crosses port boundary
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Same(t, validPackage, result.Package)
	mockInstaller.AssertExpectations(t)
}

// TestPackageServiceInstallHandlesPortErrors tests error propagation from port.
func TestPackageServiceInstallHandlesPortErrors(t *testing.T) {
	tests := []struct {
		name          string
		portError     error
		expectedError error
		description   string
	}{
		{
			name:          "network failure propagated",
			portError:     domain.ErrNetworkFailure,
			expectedError: domain.ErrNetworkFailure,
			description:   "Network errors should be propagated",
		},
		{
			name:          "permission denied propagated",
			portError:     domain.ErrPermissionDenied,
			expectedError: domain.ErrPermissionDenied,
			description:   "Permission errors should be propagated",
		},
		{
			name:          "package not found propagated",
			portError:     domain.ErrPackageNotFound,
			expectedError: domain.ErrPackageNotFound,
			description:   "Package not found errors should be propagated",
		},
		{
			name:          "already installed handled",
			portError:     domain.ErrAlreadyInstalled,
			expectedError: domain.ErrAlreadyInstalled,
			description:   "Already installed is not a fatal error",
		},
		{
			name:          "generic errors propagated",
			portError:     errors.New("unexpected error"),
			expectedError: nil, // We check that an error exists, not its type
			description:   "Unexpected errors should be propagated",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockInstaller := new(testutil.MockPackageInstaller)
			mockDetector := new(testutil.MockSystemDetector)
			service := domain.NewPackageService(mockInstaller, mockDetector)

			validPackage := testutil.CreateValidPackage("test-app")

			// Setup mock to return error at port boundary
			mockInstaller.On("Install", mock.Anything, validPackage).Return(nil, tc.portError)

			ctx := context.Background()
			result, err := service.Install(ctx, validPackage)

			// Business logic: errors from port must be propagated
			assert.Nil(t, result)
			require.Error(t, err, tc.description)

			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			}

			mockInstaller.AssertExpectations(t)
		})
	}
}

// TestPackageServiceRemoveRejectsInvalidPackages tests removal validation.
func TestPackageServiceRemoveRejectsInvalidPackages(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	invalidPackage := testutil.CreateInvalidPackage()

	ctx := context.Background()
	result, err := service.Remove(ctx, invalidPackage)

	// Business rule: Invalid packages must be rejected for removal too
	assert.Nil(t, result)
	require.ErrorIs(t, err, domain.ErrInvalidPackage)
	mockInstaller.AssertNotCalled(t, "Remove", mock.Anything, mock.Anything)
}

// TestPackageServiceRemoveDelegatesValidPackagesToPort tests valid removal.
func TestPackageServiceRemoveDelegatesValidPackagesToPort(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	validPackage := testutil.CreateValidPackage("test-app")

	expectedResult := &domain.InstallationResult{
		Package: validPackage,
		Success: true,
	}

	// Setup mock expectation
	mockInstaller.On("Remove", mock.Anything, validPackage).Return(expectedResult, nil)

	ctx := context.Background()
	result, err := service.Remove(ctx, validPackage)

	// Verify removal succeeds for valid package
	require.NoError(t, err)
	assert.True(t, result.Success)
	mockInstaller.AssertExpectations(t)
}

// TestPackageListingScenarios tests various package listing scenarios.
func TestPackageListingScenarios(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	t.Run("list_multiple_packages", func(t *testing.T) {
		installedPackages := []*domain.Package{
			testutil.CreateValidPackage("vim"),
			testutil.CreateValidPackage("git"),
			testutil.CreateValidPackage("curl"),
		}

		mockInstaller.On("List", ctx).Return(installedPackages, nil).Once()
		packages, err := service.List(ctx)

		require.NoError(t, err)
		assert.Len(t, packages, 3)
		// Verify all packages are valid
		for _, pkg := range packages {
			assert.True(t, pkg.IsValid())
		}
	})

	t.Run("empty_system_no_packages", func(t *testing.T) {
		mockInstaller.On("List", ctx).Return([]*domain.Package{}, nil).Once()
		packages, err := service.List(ctx)

		require.NoError(t, err)
		assert.Empty(t, packages)
	})

	t.Run("list_error_propagation", func(t *testing.T) {
		// Test that errors from port are properly handled
		mockInstaller.On("List", ctx).Return(nil, domain.ErrPermissionDenied).Once()
		packages, err := service.List(ctx)

		assert.Nil(t, packages)
		assert.ErrorIs(t, err, domain.ErrPermissionDenied)
	})
}

// TestPackageServiceListHandlesErrors tests error handling in List operation.
func TestPackageServiceListHandlesErrors(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()
	expectedError := errors.New("failed to list packages")
	mockInstaller.On("List", ctx).Return(nil, expectedError)

	packages, err := service.List(ctx)

	assert.Nil(t, packages)
	assert.Equal(t, expectedError, err)
	mockInstaller.AssertExpectations(t)
}

// TestPackageVersionSpecification tests packages with version constraints.
func TestPackageVersionSpecification(t *testing.T) {
	tests := []struct {
		name    string
		pkg     *domain.Package
		isValid bool
	}{
		{
			name: "package with specific version",
			pkg: &domain.Package{
				Name:    "tool",
				Method:  domain.MethodAPT,
				Source:  "tool",
				Version: "1.2.3",
			},
			isValid: true,
		},
		{
			name: "package with version range",
			pkg: &domain.Package{
				Name:    "tool",
				Method:  domain.MethodAPT,
				Source:  "tool",
				Version: ">=1.0.0",
			},
			isValid: true,
		},
		{
			name: "package without version (latest)",
			pkg: &domain.Package{
				Name:   "tool",
				Method: domain.MethodAPT,
				Source: "tool",
			},
			isValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.isValid, tc.pkg.IsValid())
		})
	}
}
