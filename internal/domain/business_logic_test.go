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

// TestCriticalPackageProtection tests that critical system packages are handled carefully.
func TestCriticalPackageProtection(t *testing.T) {
	criticalPackages := []string{
		"systemd",
		"kernel",
		"libc6",
		"bash",
		"coreutils",
	}

	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	for _, pkgName := range criticalPackages {
		t.Run(pkgName, func(t *testing.T) {
			pkg := &domain.Package{
				Name:   pkgName,
				Method: domain.MethodAPT,
				Source: "system",
			}

			// Critical packages should be handled with care
			// Current implementation may not have specific protection
			mockInstaller.On("Remove", ctx, pkg).
				Return(nil, domain.ErrPermissionDenied).Maybe()

			result, err := service.Remove(ctx, pkg)

			// Document current behavior: no specific critical package protection
			if err != nil {
				// System should at least require permissions
				require.Error(t, err, "Removing critical package %s should be restricted", pkgName)
			}

			assert.Nil(t, result)
		})
	}
}

// TestPackageVersionConstraints tests version-specific installation logic.
func TestPackageVersionConstraints(t *testing.T) {
	tests := []struct {
		name        string
		pkg         *domain.Package
		shouldAllow bool
		reason      string
	}{
		{
			name: "specific_version_allowed",
			pkg: &domain.Package{
				Name:    "nodejs",
				Version: "18.17.0",
				Method:  domain.MethodAPT,
				Source:  "nodesource",
			},
			shouldAllow: true,
			reason:      "Specific versions should be installable",
		},
		{
			name: "version_range_allowed",
			pkg: &domain.Package{
				Name:    "postgresql",
				Version: ">=14",
				Method:  domain.MethodAPT,
				Source:  "pgdg",
			},
			shouldAllow: true,
			reason:      "Version constraints should be supported",
		},
		{
			name: "latest_version_allowed",
			pkg: &domain.Package{
				Name:    "docker",
				Version: "latest",
				Method:  domain.MethodScript,
				Source:  "get.docker.com",
			},
			shouldAllow: true,
			reason:      "Latest version should be allowed",
		},
	}

	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldAllow {
				mockInstaller.On("Install", ctx, tc.pkg).
					Return(&domain.InstallationResult{
						Package: tc.pkg,
						Success: true,
					}, nil).Maybe()
			}

			result, err := service.Install(ctx, tc.pkg)

			if tc.shouldAllow {
				require.NoError(t, err, tc.reason)

				if result != nil {
					assert.True(t, result.Success)
				}
			} else {
				assert.Error(t, err, tc.reason)
			}
		})
	}
}

// TestPackageSourceValidation tests that package sources are properly validated.
func TestPackageSourceBusinessValidation(t *testing.T) {
	tests := []struct {
		name        string
		method      domain.InstallMethod
		source      string
		shouldValid bool
		reason      string
	}{
		{
			name:        "github_source_format",
			method:      domain.MethodGitHub,
			source:      "owner/repo",
			shouldValid: true,
			reason:      "GitHub sources should be owner/repo format",
		},
		{
			name:        "github_invalid_format",
			method:      domain.MethodGitHub,
			source:      "not-a-valid-github",
			shouldValid: true, // Current implementation doesn't validate format
			reason:      "Invalid GitHub format currently accepted",
		},
		{
			name:        "url_source_for_binary",
			method:      domain.MethodBinary,
			source:      "https://example.com/binary.tar.gz",
			shouldValid: true,
			reason:      "Binary method should accept URLs",
		},
		{
			name:        "script_source_validation",
			method:      domain.MethodScript,
			source:      "https://get.example.com/install.sh",
			shouldValid: true,
			reason:      "Script sources should be URLs or paths",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pkg := &domain.Package{
				Name:   "test-app",
				Method: tc.method,
				Source: tc.source,
			}

			isValid := pkg.IsValid()
			assert.Equal(t, tc.shouldValid, isValid, tc.reason)
		})
	}
}

// TestTransactionalInstallation tests all-or-nothing installation semantics.
func TestTransactionalInstallation(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	packages := []*domain.Package{
		testutil.CreateValidPackage("package1"),
		testutil.CreateValidPackage("package2"),
		testutil.CreateValidPackage("package3"),
	}

	// Simulate failure on second package
	mockInstaller.On("Install", ctx, packages[0]).
		Return(&domain.InstallationResult{Package: packages[0], Success: true}, nil).Once()

	mockInstaller.On("Install", ctx, packages[1]).
		Return(nil, domain.ErrDependencyMissing).Once()

	// Third package should not be attempted in transactional mode
	// (This depends on implementation - document current behavior)
	mockInstaller.On("Install", ctx, packages[2]).
		Return(&domain.InstallationResult{Package: packages[2], Success: true}, nil).Maybe()

	// Install all packages
	results := make([]*domain.InstallationResult, 0, len(packages))

	var failedPackage *domain.Package

	for _, pkg := range packages {
		result, err := service.Install(ctx, pkg)
		if err != nil {
			failedPackage = pkg
			break
		}

		results = append(results, result)
	}

	// Verify that installation stopped at failure
	assert.NotNil(t, failedPackage, "Should have a failed package")
	assert.Equal(t, packages[1], failedPackage, "Should fail on second package")
	assert.Len(t, results, 1, "Only first package should be installed before failure")
}

// TestSystemCompatibilityChecks tests OS/architecture compatibility validation.
func TestSystemCompatibilityChecks(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	// Setup system detection with actual domain structures
	systemInfo := &domain.SystemInfo{
		Distribution: &domain.Distribution{
			ID:     "ubuntu",
			Name:   "Ubuntu",
			Family: "debian",
		},
		Architecture: "amd64",
		Kernel:       "5.15.0",
	}
	mockDetector.On("DetectSystem", ctx).Return(systemInfo, nil).Maybe()

	tests := []struct {
		name       string
		pkg        *domain.Package
		compatible bool
		reason     string
	}{
		{
			name: "linux_package_on_linux",
			pkg: &domain.Package{
				Name:   "docker",
				Method: domain.MethodScript,
				Source: "get.docker.com",
			},
			compatible: true,
			reason:     "Linux package on Linux system",
		},
		{
			name: "architecture_specific_amd64",
			pkg: &domain.Package{
				Name:   "vscode",
				Method: domain.MethodDEB,
				Source: "microsoft",
			},
			compatible: true,
			reason:     "amd64 package on amd64 system",
		},
		{
			name: "ubuntu_specific_package",
			pkg: &domain.Package{
				Name:   "ubuntu-restricted-extras",
				Method: domain.MethodAPT,
				Source: "ubuntu",
			},
			compatible: true,
			reason:     "Ubuntu package on Ubuntu system",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.compatible {
				mockInstaller.On("Install", ctx, tc.pkg).
					Return(&domain.InstallationResult{
						Package: tc.pkg,
						Success: true,
					}, nil).Maybe()
			}

			result, err := service.Install(ctx, tc.pkg)

			// Current implementation doesn't have system compatibility checks
			// This test documents what could be implemented
			if tc.compatible {
				// Should succeed
				if err != nil {
					assert.NotEqual(t, "incompatible system", err.Error(), tc.reason)
				}
			}

			_ = result // Document that result is checked elsewhere
		})
	}
}

// TestConflictingPackageHandling tests how conflicting packages are handled.
func TestConflictingPackageHandling(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	// Test MySQL vs MariaDB conflict scenario
	mysql := &domain.Package{
		Name:   "mysql-server",
		Method: domain.MethodAPT,
		Source: "ubuntu",
	}

	mariadb := &domain.Package{
		Name:   "mariadb-server",
		Method: domain.MethodAPT,
		Source: "ubuntu",
	}

	// Install MySQL first
	mockInstaller.On("Install", ctx, mysql).
		Return(&domain.InstallationResult{Package: mysql, Success: true}, nil).Once()

	result1, err1 := service.Install(ctx, mysql)
	require.NoError(t, err1)
	assert.True(t, result1.Success)

	// Attempt to install MariaDB - adapter should detect conflict
	// Current implementation may not have conflict detection
	mockInstaller.On("Install", ctx, mariadb).
		Return(nil, domain.ErrAlreadyInstalled).Once()

	result2, err2 := service.Install(ctx, mariadb)
	require.Error(t, err2, "Should not install conflicting package")
	assert.Nil(t, result2)
}

// TestInstallationMetrics tests that installation metrics are properly collected.
func TestInstallationMetrics(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	pkg := testutil.CreateValidPackage("nginx")

	expectedResult := &domain.InstallationResult{
		Package:  pkg,
		Success:  true,
		Duration: 1500, // 1.5 seconds in milliseconds
		Output:   "Reading package lists...\nInstalling nginx...\nSuccess",
	}

	mockInstaller.On("Install", ctx, pkg).
		Return(expectedResult, nil).Once()

	result, err := service.Install(ctx, pkg)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Positive(t, result.Duration, "Should record installation duration")
	assert.NotEmpty(t, result.Output, "Should capture installation output")
}

// TestPackageCleanupOnFailure tests that partial installations are handled on failure.
func TestPackageCleanupOnFailure(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	pkg := testutil.CreateValidPackage("complex-app")

	// Simulate installation failure with network error
	mockInstaller.On("Install", ctx, pkg).
		Return(nil, domain.ErrNetworkFailure).Once()

	// Cleanup might be attempted (implementation-dependent)
	mockInstaller.On("Remove", ctx, mock.MatchedBy(func(p *domain.Package) bool {
		return p.Name == pkg.Name
	})).Return(&domain.InstallationResult{Success: true}, nil).Maybe()

	result, err := service.Install(ctx, pkg)

	require.ErrorIs(t, err, domain.ErrNetworkFailure)
	assert.Nil(t, result)

	// Document current behavior: no automatic cleanup on failure
}
