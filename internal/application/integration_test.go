// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application_test

import (
	"context"
	"testing"

	"github.com/janderssonse/karei/internal/application"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestUserJourneyInstallMultipleAppsOnFreshSystem tests complete flow from fresh system to multiple apps installed.
func TestUserJourneyInstallMultipleAppsOnFreshSystem(t *testing.T) {
	// Setup: Fresh Ubuntu system with no packages installed
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)

	domainService := domain.NewPackageService(mockInstaller, mockDetector)
	service := application.NewInstallService(domainService, mockDetector)

	ctx := context.Background()

	// Step 1: System detection for fresh Ubuntu system
	freshUbuntuSystem := &domain.SystemInfo{
		Distribution: &domain.Distribution{
			ID:      "ubuntu",
			Version: "22.04",
			Family:  "debian",
		},
		PackageManager: &domain.PackageManager{
			Method: domain.MethodAPT,
		},
	}

	mockDetector.On("DetectSystem", ctx).Return(freshUbuntuSystem, nil)

	// Step 2: Initially no packages installed
	mockInstaller.On("List", ctx).Return([]*domain.Package{}, nil).Once()

	// Verify empty system
	packages, err := service.ListInstalledPackages(ctx)
	require.NoError(t, err)
	assert.Empty(t, packages)

	// Step 3: Install essential developer tools
	essentialApps := map[string]string{
		"vim":             "vim",
		"git":             "git",
		"curl":            "curl",
		"build-essential": "build-essential",
	}

	// Setup expectations for each installation
	for appName, source := range essentialApps {
		mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
			return pkg.Name == appName && pkg.Source == source && pkg.Method == domain.MethodAPT
		})).Return(&domain.InstallationResult{
			Package: &domain.Package{Name: appName, Method: domain.MethodAPT, Source: source},
			Success: true,
			Output:  "Successfully installed " + appName,
		}, nil).Once()
	}

	// Execute batch installation
	results, err := service.InstallMultipleApplications(ctx, essentialApps)

	// Step 4: Verify all installations succeeded
	require.NoError(t, err)
	assert.Len(t, results, 4)

	for _, result := range results {
		assert.True(t, result.Success, "All essential apps should install successfully")
		require.NoError(t, result.Error)
	}

	// Step 5: Verify packages now show as installed
	installedPackages := []*domain.Package{
		{Name: "vim", Method: domain.MethodAPT, Source: "vim"},
		{Name: "git", Method: domain.MethodAPT, Source: "git"},
		{Name: "curl", Method: domain.MethodAPT, Source: "curl"},
		{Name: "build-essential", Method: domain.MethodAPT, Source: "build-essential"},
	}
	mockInstaller.On("List", ctx).Return(installedPackages, nil).Once()

	packages, err = service.ListInstalledPackages(ctx)
	require.NoError(t, err)
	assert.Len(t, packages, 4)

	// Verify all expectations met
	mockDetector.AssertExpectations(t)
	mockInstaller.AssertExpectations(t)
}

// TestUserJourneyMigrateFromAPTToSnap tests migration scenario.
func TestUserJourneyMigrateFromAPTToSnap(t *testing.T) {
	// Scenario: User wants to migrate an app from APT to Snap for better sandboxing
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)

	domainService := domain.NewPackageService(mockInstaller, mockDetector)
	service := application.NewInstallService(domainService, mockDetector)

	ctx := context.Background()

	// System supports both APT and Snap
	hybridSystem := &domain.SystemInfo{
		Distribution: &domain.Distribution{
			ID:      "ubuntu",
			Version: "22.04",
			Family:  "debian",
		},
		PackageManager: &domain.PackageManager{
			Method: domain.MethodAPT, // Primary method
		},
	}

	// DetectSystem may be called multiple times during the test
	mockDetector.On("DetectSystem", ctx).Return(hybridSystem, nil).Maybe()

	// Step 1: App currently installed via APT
	currentPackages := []*domain.Package{
		{Name: "code", Method: domain.MethodAPT, Source: "code"},
	}
	mockInstaller.On("List", ctx).Return(currentPackages, nil).Once()

	packages, err := service.ListInstalledPackages(ctx)
	require.NoError(t, err)
	assert.Len(t, packages, 1)
	assert.Equal(t, domain.MethodAPT, packages[0].Method)

	// Step 2: Remove APT version
	mockInstaller.On("Remove", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "code" && pkg.Method == domain.MethodAPT
	})).Return(&domain.InstallationResult{
		Package: &domain.Package{Name: "code", Method: domain.MethodAPT, Source: "code"},
		Success: true,
		Output:  "Removed code from APT",
	}, nil).Once()

	// Note: InstallService doesn't have RemoveApplication, so we call domain service directly
	removePkg := &domain.Package{Name: "code", Method: domain.MethodAPT, Source: "code"}
	removeResult, err := domainService.Remove(ctx, removePkg)
	require.NoError(t, err)
	assert.True(t, removeResult.Success)

	// Step 3: Install Snap version
	// In real scenario, user would specify snap method explicitly
	snapPkg := &domain.Package{
		Name:   "code",
		Method: domain.MethodSnap,
		Source: "code --classic",
	}

	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "code" && pkg.Method == domain.MethodSnap
	})).Return(&domain.InstallationResult{
		Package: snapPkg,
		Success: true,
		Output:  "Installed code from Snap Store",
	}, nil).Once()

	installResult, err := domainService.Install(ctx, snapPkg)
	require.NoError(t, err)
	assert.True(t, installResult.Success)

	// Step 4: Verify migration complete
	newPackages := []*domain.Package{
		{Name: "code", Method: domain.MethodSnap, Source: "code --classic"},
	}
	mockInstaller.On("List", ctx).Return(newPackages, nil).Once()

	packages, err = service.ListInstalledPackages(ctx)
	require.NoError(t, err)
	assert.Len(t, packages, 1)
	assert.Equal(t, domain.MethodSnap, packages[0].Method, "App should now be installed via Snap")

	mockDetector.AssertExpectations(t)
	mockInstaller.AssertExpectations(t)
}

// TestUserJourneyHandleFailedInstallationWithRetry tests error recovery flow.
func TestUserJourneyHandleFailedInstallationWithRetry(t *testing.T) {
	// Scenario: Network failure during installation, user retries after fixing network
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)

	domainService := domain.NewPackageService(mockInstaller, mockDetector)
	service := application.NewInstallService(domainService, mockDetector)

	ctx := context.Background()

	// Setup system
	systemInfo := testutil.CreateTestSystemInfo()
	mockDetector.On("DetectSystem", ctx).Return(systemInfo, nil)

	// Step 1: First attempt fails due to network
	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "docker"
	})).Return(nil, domain.ErrNetworkFailure).Once()

	result, err := service.InstallApplication(ctx, "docker", "docker.io")

	// Verify failure is handled gracefully
	require.Error(t, err)
	require.ErrorIs(t, err, domain.ErrNetworkFailure)
	assert.Nil(t, result)

	// Step 2: User fixes network and retries
	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "docker"
	})).Return(&domain.InstallationResult{
		Package: &domain.Package{Name: "docker", Method: domain.MethodAPT, Source: "docker.io"},
		Success: true,
		Output:  "Docker installed successfully",
	}, nil).Once()

	result, err = service.InstallApplication(ctx, "docker", "docker.io")

	// Verify retry succeeds
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	mockDetector.AssertExpectations(t)
	mockInstaller.AssertExpectations(t)
}

// TestUserJourneyDetectAndInstallForDifferentDistros tests cross-distro compatibility.
func TestUserJourneyDetectAndInstallForDifferentDistros(t *testing.T) {
	testCases := []struct {
		name           string
		systemInfo     *domain.SystemInfo
		expectedMethod domain.InstallMethod
		appName        string
		source         string
	}{
		{
			name: "Ubuntu with APT",
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
			appName:        "nginx",
			source:         "nginx",
		},
		{
			name: "Fedora with DNF",
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
			appName:        "nginx",
			source:         "nginx",
		},
		{
			name: "Arch with Pacman",
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
			appName:        "nginx",
			source:         "nginx",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockInstaller := new(testutil.MockPackageInstaller)
			mockDetector := new(testutil.MockSystemDetector)

			domainService := domain.NewPackageService(mockInstaller, mockDetector)
			service := application.NewInstallService(domainService, mockDetector)

			ctx := context.Background()

			// Setup system detection
			mockDetector.On("DetectSystem", ctx).Return(tc.systemInfo, nil).Once()

			// Setup installation expectation with correct method
			mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
				return pkg.Name == tc.appName &&
					pkg.Method == tc.expectedMethod &&
					pkg.Source == tc.source
			})).Return(&domain.InstallationResult{
				Package: &domain.Package{
					Name:   tc.appName,
					Method: tc.expectedMethod,
					Source: tc.source,
				},
				Success: true,
			}, nil).Once()

			// Execute installation
			result, err := service.InstallApplication(ctx, tc.appName, tc.source)

			// Verify correct method was selected for the distro
			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.Equal(t, tc.expectedMethod, result.Package.Method)

			mockDetector.AssertExpectations(t)
			mockInstaller.AssertExpectations(t)
		})
	}
}

// TestUserJourneyInstallWithDependencyResolution tests complex dependency scenarios.
func TestUserJourneyInstallWithDependencyResolution(t *testing.T) {
	// Scenario: Installing a package that requires multiple dependencies
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)

	domainService := domain.NewPackageService(mockInstaller, mockDetector)
	service := application.NewInstallService(domainService, mockDetector)

	ctx := context.Background()

	systemInfo := testutil.CreateTestSystemInfo()
	mockDetector.On("DetectSystem", ctx).Return(systemInfo, nil)

	// Installing a complex app triggers dependency installation
	// The port layer handles dependency resolution
	mockInstaller.On("Install", ctx, mock.MatchedBy(func(pkg *domain.Package) bool {
		return pkg.Name == "postgresql"
	})).Return(&domain.InstallationResult{
		Package: &domain.Package{Name: "postgresql", Method: domain.MethodAPT, Source: "postgresql"},
		Success: true,
		Output:  "Installed postgresql with dependencies: libpq5, postgresql-client, postgresql-common",
	}, nil).Once()

	result, err := service.InstallApplication(ctx, "postgresql", "postgresql")

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "dependencies")

	// Verify the system now has all components
	installedPackages := []*domain.Package{
		{Name: "postgresql", Method: domain.MethodAPT, Source: "postgresql"},
		{Name: "libpq5", Method: domain.MethodAPT, Source: "libpq5"},
		{Name: "postgresql-client", Method: domain.MethodAPT, Source: "postgresql-client"},
		{Name: "postgresql-common", Method: domain.MethodAPT, Source: "postgresql-common"},
	}
	mockInstaller.On("List", ctx).Return(installedPackages, nil).Once()

	packages, err := service.ListInstalledPackages(ctx)
	require.NoError(t, err)
	assert.Len(t, packages, 4, "Main package plus dependencies should be installed")

	mockDetector.AssertExpectations(t)
	mockInstaller.AssertExpectations(t)
}
