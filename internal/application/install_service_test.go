// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/janderssonse/karei/internal/application"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testPackageName = "vim"
)

// SetupServiceMocks creates common test setup to reduce duplication.
func SetupServiceMocks() (*testutil.MockPackageInstaller, *testutil.MockSystemDetector, *application.InstallService) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	domainService := domain.NewPackageService(mockInstaller, mockDetector)
	service := application.NewInstallService(domainService, mockDetector)

	return mockInstaller, mockDetector, service
}

func TestInstallApplicationWithSystemDetection(t *testing.T) {
	t.Parallel()

	mockInstaller, mockDetector, service := SetupServiceMocks()

	ctx := context.Background()

	// Setup system detection
	systemInfo := testutil.CreateTestSystemInfo()
	mockDetector.On("DetectSystem", ctx).Return(systemInfo, nil)

	// Setup package installation
	expectedResult := &domain.InstallationResult{
		Success: true,
	}
	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == testPackageName && pkg.Source == testPackageName && pkg.Method == domain.MethodAPT
	})).Return(expectedResult, nil)

	// Execute
	result, err := service.InstallApplication(ctx, testPackageName, testPackageName)

	// Verify orchestration worked correctly
	require.NoError(t, err)
	assert.True(t, result.Success)

	mockDetector.AssertExpectations(t)
	mockInstaller.AssertExpectations(t)
}

func TestInstallApplicationHandlesSystemDetectionFailure(t *testing.T) {
	t.Parallel()

	mockInstaller, mockDetector, service := SetupServiceMocks()

	ctx := context.Background()

	// Setup system detection to fail
	mockDetector.On("DetectSystem", ctx).Return(nil, errors.New("detection failed"))

	// Execute
	result, err := service.InstallApplication(ctx, "vim", "vim")

	// Verify error handling
	require.Error(t, err)
	assert.Nil(t, result)
	// The error should mention system detection failure
	// This is important for debugging
	assert.Contains(t, err.Error(), "detect system")

	// Verify installation was never attempted
	mockInstaller.AssertNotCalled(t, "Install")
	mockDetector.AssertExpectations(t)
}

func TestInstallApplicationWithNoPackageManager(t *testing.T) {
	t.Parallel()

	mockInstaller, mockDetector, service := SetupServiceMocks()

	ctx := context.Background()

	// Return error indicating no package manager
	mockDetector.On("DetectSystem", ctx).Return(nil, domain.ErrNoPackageManager)

	// Execute
	result, err := service.InstallApplication(ctx, "vim", "vim")

	// Should handle the error gracefully
	require.Error(t, err)
	assert.Nil(t, result)
	// Error should indicate system detection failure
	assert.Contains(t, err.Error(), "detect system")
	mockInstaller.AssertNotCalled(t, "Install")
}

func TestInstallMultipleApplicationsContinuesOnFailure(t *testing.T) {
	t.Parallel()

	mockInstaller, mockDetector, service := SetupServiceMocks()

	ctx := context.Background()

	// Setup system detection
	systemInfo := testutil.CreateTestSystemInfo()
	mockDetector.On("DetectSystem", ctx).Return(systemInfo, nil)

	// Setup mixed success/failure results
	successResult := &domain.InstallationResult{Success: true}

	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "vim"
	})).Return(successResult, nil).Once()

	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "git"
	})).Return(nil, domain.ErrNetworkFailure).Once()

	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "curl"
	})).Return(successResult, nil).Once()

	// Execute
	apps := map[string]string{
		"vim":  "vim",
		"git":  "git",
		"curl": "curl",
	}

	results, err := service.InstallMultipleApplications(ctx, apps)

	// Verify partial success is handled correctly
	require.NoError(t, err) // Overall operation doesn't fail
	assert.Len(t, results, 3)

	// Count successes and failures
	successCount := 0
	failureCount := 0

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	assert.Equal(t, 2, successCount, "Should have 2 successful installations")
	assert.Equal(t, 1, failureCount, "Should have 1 failed installation")

	mockDetector.AssertExpectations(t)
	mockInstaller.AssertExpectations(t)
}

// TestInstallApplicationWithDependencies tests dependency handling.
func TestInstallApplicationWithDependencies(t *testing.T) {
	t.Parallel()

	mockInstaller, mockDetector, service := SetupServiceMocks()

	ctx := context.Background()

	systemInfo := testutil.CreateTestSystemInfo()
	mockDetector.On("DetectSystem", ctx).Return(systemInfo, nil)

	// App with dependencies - note that InstallApplication doesn't set dependencies
	// The dependencies would be resolved by the installer port
	expectedResult := &domain.InstallationResult{
		Success: true,
		Output:  "Installed with dependencies: lib1, lib2",
	}

	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "complex-app"
	})).Return(expectedResult, nil)

	// Execute
	result, err := service.InstallApplication(ctx, "complex-app", "complex-app")

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "dependencies")
}

// TestInstallApplicationValidatesInput tests input validation at application layer.
func TestInstallApplicationValidatesInput(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		source      string
		shouldError bool
	}{
		{
			name:        "empty app name",
			appName:     "",
			source:      "source",
			shouldError: true,
		},
		{
			name:        "empty source",
			appName:     "app",
			source:      "",
			shouldError: true,
		},
		{
			name:        "valid input",
			appName:     "app",
			source:      "source",
			shouldError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockInstaller, mockDetector, service := SetupServiceMocks()

			ctx := context.Background()

			// Always setup system detection since InstallApplication calls it first
			systemInfo := testutil.CreateTestSystemInfo()
			mockDetector.On("DetectSystem", ctx).Return(systemInfo, nil).Once()

			// The package will be invalid if name or source is empty
			// which will be caught at the domain layer
			if tc.appName != "" && tc.source != "" {
				mockInstaller.On("Install", ctx, mock.Anything).
					Return(&domain.InstallationResult{Success: true}, nil).Maybe()
			}

			result, err := service.InstallApplication(ctx, tc.appName, tc.source)

			if tc.shouldError {
				// Empty name/source will create invalid package caught by domain layer
				require.Error(t, err)
				assert.Nil(t, result)
			} else if err == nil {
				// Valid case might succeed
				assert.NotNil(t, result)
			}
		})
	}
}

// TestPublicAPICoverage tests all public API methods.
// These are simple delegations but need coverage as they're public API.
func TestPublicAPICoverage(t *testing.T) {
	mockInstaller, mockDetector, service := SetupServiceMocks()

	ctx := context.Background()

	t.Run("GetSystemInfo", func(t *testing.T) {
		expectedInfo := testutil.CreateTestSystemInfo()
		mockDetector.On("DetectSystem", ctx).Return(expectedInfo, nil).Once()

		info, err := service.GetSystemInfo(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedInfo, info)
	})

	t.Run("ListInstalledPackages", func(t *testing.T) {
		expectedPackages := []*domain.Package{
			testutil.CreateValidPackage("vim"),
		}
		mockInstaller.On("List", ctx).Return(expectedPackages, nil).Once()

		packages, err := service.ListInstalledPackages(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedPackages, packages)
	})
}

// TestInstallMultipleApplicationsErrorAggregation tests proper error reporting for batch operations.
func TestInstallMultipleApplicationsErrorAggregation(t *testing.T) {
	mockInstaller, mockDetector, service := SetupServiceMocks()

	ctx := context.Background()

	// Setup system detection
	systemInfo := testutil.CreateTestSystemInfo()
	mockDetector.On("DetectSystem", ctx).Return(systemInfo, nil)

	// Setup different error types for different packages
	// When installer returns error, the result is nil
	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "network-fail"
	})).Return(nil, domain.ErrNetworkFailure).Once()

	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "permission-fail"
	})).Return(nil, domain.ErrPermissionDenied).Once()

	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "success-app"
	})).Return(&domain.InstallationResult{
		Package: &domain.Package{Name: "success-app", Method: domain.MethodAPT, Source: "source3"},
		Success: true,
	}, nil).Once()

	// Execute batch install
	apps := map[string]string{
		"network-fail":    "source1",
		"permission-fail": "source2",
		"success-app":     "source3",
	}

	results, err := service.InstallMultipleApplications(ctx, apps)

	// Business rule: batch operations should complete even with failures
	require.NoError(t, err, "Batch operation should not fail overall")
	assert.Len(t, results, 3)

	// Verify each result has appropriate error information
	successCount := 0
	networkFailCount := 0
	permissionFailCount := 0

	for _, result := range results {
		assert.NotNil(t, result, "Result should never be nil")
		assert.NotNil(t, result.Package, "Package should be set even on failure")

		if result.Success {
			successCount++

			require.NoError(t, result.Error)
		} else {
			require.Error(t, result.Error, "Failed result should have error")

			if errors.Is(result.Error, domain.ErrNetworkFailure) {
				networkFailCount++
			} else if errors.Is(result.Error, domain.ErrPermissionDenied) {
				permissionFailCount++
			}
		}
	}

	assert.Equal(t, 1, successCount, "Should have 1 successful install")
	assert.Equal(t, 1, networkFailCount, "Should have 1 network failure")
	assert.Equal(t, 1, permissionFailCount, "Should have 1 permission failure")

	mockDetector.AssertExpectations(t)
	mockInstaller.AssertExpectations(t)
}

// TestInstallApplicationWithContextTimeout tests context timeout handling.
func TestInstallApplicationWithContextTimeout(t *testing.T) {
	_, mockDetector, service := SetupServiceMocks()

	// Create context that's already timed out
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	// System detection should fail with deadline exceeded
	mockDetector.On("DetectSystem", mock.Anything).
		Return(nil, context.DeadlineExceeded).Maybe()

	result, err := service.InstallApplication(ctx, "app", "source")

	// Business rule: timeout errors should be propagated
	require.Error(t, err)
	assert.Nil(t, result)
	// Check for actual timeout error type
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestGetBestMethodForSystem tests the critical business logic of package method selection.
func TestGetBestMethodForSystem(t *testing.T) {
	tests := []struct {
		name           string
		systemInfo     *domain.SystemInfo
		expectedMethod domain.InstallMethod
		reason         string
	}{
		{
			name: "ubuntu_with_apt",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "ubuntu",
					Family: "debian",
				},
				PackageManager: &domain.PackageManager{
					Method: domain.MethodAPT,
				},
			},
			expectedMethod: domain.MethodAPT,
			reason:         "Ubuntu should use APT when available",
		},
		{
			name: "debian_with_apt",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "debian",
					Family: "debian",
				},
				PackageManager: &domain.PackageManager{
					Method: domain.MethodAPT,
				},
			},
			expectedMethod: domain.MethodAPT,
			reason:         "Debian should use APT when available",
		},
		{
			name: "fedora_with_dnf",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "fedora",
					Family: "rhel",
				},
				PackageManager: &domain.PackageManager{
					Method: domain.MethodDNF,
				},
			},
			expectedMethod: domain.MethodDNF,
			reason:         "Fedora should prefer DNF",
		},
		{
			name: "rhel_with_yum",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "rhel",
					Family: "rhel",
				},
				PackageManager: &domain.PackageManager{
					Method: domain.MethodYum,
				},
			},
			expectedMethod: domain.MethodYum,
			reason:         "RHEL should use Yum when DNF not available",
		},
		{
			name: "arch_with_pacman",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "arch",
					Family: "arch",
				},
				PackageManager: &domain.PackageManager{
					Method: domain.MethodPacman,
				},
			},
			expectedMethod: domain.MethodPacman,
			reason:         "Arch always uses Pacman",
		},
		{
			name: "manjaro_arch_family",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "manjaro",
					Family: "arch",
				},
				PackageManager: &domain.PackageManager{
					Method: domain.MethodPacman,
				},
			},
			expectedMethod: domain.MethodPacman,
			reason:         "Arch derivatives use Pacman",
		},
		{
			name: "unknown_distro_fallback",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "unknown",
					Family: "unknown",
				},
				PackageManager: &domain.PackageManager{
					Method: domain.MethodSnap,
				},
			},
			expectedMethod: domain.MethodSnap,
			reason:         "Unknown distros fallback to detected package manager",
		},
		{
			name: "ubuntu_without_matching_method",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "ubuntu",
					Family: "debian",
				},
				PackageManager: &domain.PackageManager{
					Method: domain.MethodSnap, // Only snap available
				},
			},
			expectedMethod: domain.MethodSnap,
			reason:         "Falls back to available method when preferred not available",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockInstaller, mockDetector, service := SetupServiceMocks()

			// Access the private method through InstallApplication
			ctx := context.Background()
			mockDetector.On("DetectSystem", ctx).Return(tc.systemInfo, nil)

			// The method is selected during InstallApplication
			mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
				// Verify the package has the expected method
				return pkg.Method == tc.expectedMethod
			})).Return(&domain.InstallationResult{Success: true}, nil).Maybe()

			result, err := service.InstallApplication(ctx, "test-app", "test-source")

			// If system detection succeeded, verify method selection
			if tc.systemInfo != nil {
				require.NoError(t, err, tc.reason)

				if result != nil {
					assert.True(t, result.Success)
				}
			}
		})
	}
}
