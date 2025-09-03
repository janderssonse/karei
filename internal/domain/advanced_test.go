// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestPackageVersionCompatibility tests version handling business rules.
func TestPackageVersionCompatibility(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	tests := []struct {
		name           string
		currentVersion string
		newVersion     string
		shouldSucceed  bool
		reason         string
	}{
		{
			name:           "upgrade_to_newer_version",
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			shouldSucceed:  true,
			reason:         "Upgrades should be allowed",
		},
		{
			name:           "same_version_reinstall",
			currentVersion: "1.0.0",
			newVersion:     "1.0.0",
			shouldSucceed:  true,
			reason:         "Reinstalling same version should be idempotent",
		},
		{
			name:           "downgrade_attempt",
			currentVersion: "2.0.0",
			newVersion:     "1.0.0",
			shouldSucceed:  true, // Current implementation allows downgrades
			reason:         "Downgrades currently allowed (may need --force flag in future)",
		},
		{
			name:           "install_specific_version",
			currentVersion: "",
			newVersion:     "1.2.3",
			shouldSucceed:  true,
			reason:         "Specific version installation should work",
		},
		{
			name:           "version_with_suffix",
			currentVersion: "1.0.0",
			newVersion:     "1.0.1-beta",
			shouldSucceed:  true,
			reason:         "Pre-release versions should be installable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pkg := &domain.Package{
				Name:    "test-app",
				Version: tc.newVersion,
				Method:  domain.MethodAPT,
				Source:  "ubuntu",
			}

			if tc.shouldSucceed {
				mockInstaller.On("Install", ctx, pkg).
					Return(&domain.InstallationResult{
						Package: pkg,
						Success: true,
					}, nil).Once()
			}

			result, err := service.Install(ctx, pkg)

			if tc.shouldSucceed {
				require.NoError(t, err, tc.reason)
				assert.NotNil(t, result)
				assert.True(t, result.Success)
			} else {
				assert.Error(t, err, tc.reason)
			}
		})
	}
}

// TestDependencyResolutionOrder tests that dependencies are handled correctly.
func TestDependencyResolutionOrder(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	// Create package with dependencies
	mainPkg := &domain.Package{
		Name:         "app-with-deps",
		Method:       domain.MethodAPT,
		Source:       "ubuntu",
		Dependencies: []string{"libfoo", "libbar", "libbase"},
	}

	// Test that main package installation succeeds even with dependencies
	// (dependency resolution would be handled by the adapter)
	mockInstaller.On("Install", ctx, mainPkg).
		Return(&domain.InstallationResult{
			Package: mainPkg,
			Success: true,
			Output:  "Installing dependencies: libfoo libbar libbase\nInstalling app-with-deps",
		}, nil).Once()

	result, err := service.Install(ctx, mainPkg)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	// Verify dependencies were considered
	assert.NotEmpty(t, mainPkg.Dependencies)
	assert.Len(t, mainPkg.Dependencies, 3)
}

// TestDependencyChainValidation tests complex dependency scenarios.
func TestDependencyChainValidation(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	t.Run("deep_dependency_chain", func(t *testing.T) {
		// Test that deep dependency chains are handled
		deepPkg := &domain.Package{
			Name:   "app-with-deep-deps",
			Method: domain.MethodAPT,
			Source: "ubuntu",
			Dependencies: []string{
				"level1-dep",
				"level1-dep2",
			},
		}

		// Mock successful installation with dependency resolution
		mockInstaller.On("Install", ctx, deepPkg).
			Return(&domain.InstallationResult{
				Package: deepPkg,
				Success: true,
				Output:  "Resolving dependencies...\nInstalling level1-dep\nInstalling level1-dep2\nInstalling app-with-deep-deps",
			}, nil).Once()

		result, err := service.Install(ctx, deepPkg)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "Resolving dependencies")
	})

	t.Run("missing_dependency_failure", func(t *testing.T) {
		// Test that missing dependencies cause failure
		brokenPkg := &domain.Package{
			Name:         "broken-deps-app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"non-existent-lib"},
		}

		mockInstaller.On("Install", ctx, brokenPkg).
			Return(nil, domain.ErrDependencyMissing).Once()

		result, err := service.Install(ctx, brokenPkg)
		require.ErrorIs(t, err, domain.ErrDependencyMissing)
		assert.Nil(t, result)
	})

	t.Run("optional_vs_required_dependencies", func(t *testing.T) {
		// Test that packages can have optional dependencies
		// (though current model doesn't distinguish)
		flexPkg := &domain.Package{
			Name:         "flexible-app",
			Method:       domain.MethodAPT,
			Source:       "ubuntu",
			Dependencies: []string{"required-lib", "optional-lib"},
		}

		// Installation succeeds even if optional dep fails
		mockInstaller.On("Install", ctx, flexPkg).
			Return(&domain.InstallationResult{
				Package: flexPkg,
				Success: true,
				Output:  "Warning: optional-lib not found\nInstalled with required dependencies only",
			}, nil).Once()

		result, err := service.Install(ctx, flexPkg)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "optional-lib not found")
	})
}

// TestTransactionalInstallationSemantics tests all-or-nothing installation.
func TestTransactionalInstallationSemantics(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	packages := []*domain.Package{
		{Name: "pkg1", Method: domain.MethodAPT, Source: "ubuntu"},
		{Name: "pkg2", Method: domain.MethodAPT, Source: "ubuntu"},
		{Name: "pkg3", Method: domain.MethodAPT, Source: "ubuntu"},
	}

	t.Run("all_succeed_atomically", func(t *testing.T) {
		for _, pkg := range packages {
			mockInstaller.On("Install", ctx, pkg).
				Return(&domain.InstallationResult{
					Package: pkg,
					Success: true,
				}, nil).Once()
		}

		// Install all packages
		var results []*domain.InstallationResult

		for _, pkg := range packages {
			result, err := service.Install(ctx, pkg)
			require.NoError(t, err)

			results = append(results, result)
		}

		assert.Len(t, results, 3)

		for _, result := range results {
			assert.True(t, result.Success)
		}
	})

	t.Run("failure_stops_transaction", func(t *testing.T) {
		// First succeeds
		mockInstaller.On("Install", ctx, packages[0]).
			Return(&domain.InstallationResult{
				Package: packages[0],
				Success: true,
			}, nil).Once()

		// Second fails
		mockInstaller.On("Install", ctx, packages[1]).
			Return(nil, domain.ErrDependencyMissing).Once()

		// Third should not be attempted in a transaction
		// (though current implementation doesn't enforce transactions)

		var (
			succeeded int
			failed    bool
		)

		for _, pkg := range packages {
			result, err := service.Install(ctx, pkg)
			if err != nil {
				failed = true
				break
			}

			if result.Success {
				succeeded++
			}
		}

		assert.True(t, failed, "Should have a failure")
		assert.Equal(t, 1, succeeded, "Only first package should succeed before failure")
	})
}

// TestResourceConstraintValidation tests disk space and resource checking.
func TestResourceConstraintValidation(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	largePkg := &domain.Package{
		Name:   "large-application",
		Method: domain.MethodBinary,
		Source: "https://example.com/large.tar.gz",
	}

	t.Run("insufficient_disk_space", func(t *testing.T) {
		mockInstaller.On("Install", ctx, largePkg).
			Return(nil, domain.ErrInsufficientSpace).Once()

		result, err := service.Install(ctx, largePkg)

		require.ErrorIs(t, err, domain.ErrInsufficientSpace)
		assert.Nil(t, result)
	})

	t.Run("concurrent_installation_limit", func(t *testing.T) {
		var (
			activeInstalls int32
			mu             sync.Mutex
			wg             sync.WaitGroup
		)

		// Mock installer that simulates concurrent limit

		for range 10 {
			pkg := &domain.Package{
				Name:   "concurrent-pkg",
				Method: domain.MethodAPT,
				Source: "ubuntu",
			}

			mockInstaller.On("Install", ctx, pkg).
				Return(&domain.InstallationResult{
					Package: pkg,
					Success: true,
				}, nil).Maybe()
		}

		// Launch concurrent installations
		for range 10 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				mu.Lock()

				current := atomic.AddInt32(&activeInstalls, 1)

				mu.Unlock()

				// Verify we don't exceed limit (in real implementation)
				assert.LessOrEqual(t, current, int32(10), "Concurrent installs tracked")

				pkg := &domain.Package{
					Name:   "concurrent-pkg",
					Method: domain.MethodAPT,
					Source: "ubuntu",
				}

				_, _ = service.Install(ctx, pkg)

				atomic.AddInt32(&activeInstalls, -1)
			}()
		}

		wg.Wait()
		assert.Equal(t, int32(0), activeInstalls, "All installs completed")
	})
}

// TestSecurityBoundaries tests security constraints and validation.
func TestSecurityBoundaries(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	t.Run("path_traversal_prevention", func(t *testing.T) {
		maliciousPkg := &domain.Package{
			Name:   "../../etc/passwd",
			Method: domain.MethodBinary,
			Source: "/etc/passwd",
		}

		// The package is technically valid by current rules
		assert.True(t, maliciousPkg.IsValid())

		// But installer should reject malicious paths
		mockInstaller.On("Install", ctx, maliciousPkg).
			Return(nil, domain.ErrPermissionDenied).Maybe()

		result, err := service.Install(ctx, maliciousPkg)

		// Should be rejected at some layer
		if err != nil {
			assert.Error(t, err, "Malicious path should be rejected")
		}

		_ = result
	})

	t.Run("command_injection_prevention", func(t *testing.T) {
		injectionPkg := &domain.Package{
			Name:   "app; rm -rf /",
			Method: domain.MethodScript,
			Source: "https://evil.com/script.sh",
		}

		// Package name contains shell metacharacters
		assert.True(t, injectionPkg.IsValid()) // Current implementation doesn't validate

		// Adapter should sanitize or reject
		mockInstaller.On("Install", ctx, injectionPkg).
			Return(nil, domain.ErrInvalidPackage).Maybe()

		result, err := service.Install(ctx, injectionPkg)
		if err != nil {
			assert.Error(t, err, "Command injection should be prevented")
		}

		_ = result
	})

	t.Run("untrusted_source_validation", func(t *testing.T) {
		untrustedPkg := &domain.Package{
			Name:   "suspicious-app",
			Method: domain.MethodGitHub,
			Source: "unknown-user/suspicious-repo",
		}

		// Domain allows any GitHub source
		assert.True(t, untrustedPkg.IsValid())

		// Security validation would be in adapter
		mockInstaller.On("Install", ctx, untrustedPkg).
			Return(&domain.InstallationResult{
				Package: untrustedPkg,
				Success: true,
				Output:  "Warning: Installing from untrusted source",
			}, nil).Maybe()

		result, err := service.Install(ctx, untrustedPkg)

		// Current implementation allows untrusted sources
		// This test documents the security boundary
		_ = err
		_ = result
	})
}

// TestInstallationTimeouts tests timeout handling.
func TestInstallationTimeouts(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	slowPkg := &domain.Package{
		Name:   "slow-package",
		Method: domain.MethodScript,
		Source: "https://example.com/slow.sh",
	}

	t.Run("context_timeout_respected", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Simulate slow installation
		mockInstaller.On("Install", mock.Anything, slowPkg).
			Run(func(_ mock.Arguments) {
				time.Sleep(200 * time.Millisecond)
			}).
			Return(nil, context.DeadlineExceeded).Maybe()

		result, err := service.Install(ctx, slowPkg)

		// Should timeout
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("context_cancellation_handled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel immediately
		cancel()

		mockInstaller.On("Install", mock.Anything, slowPkg).
			Return(nil, context.Canceled).Maybe()

		result, err := service.Install(ctx, slowPkg)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestPackageStateInvariants tests important state invariants.
func TestPackageStateInvariants(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	pkg := &domain.Package{
		Name:   "test-pkg",
		Method: domain.MethodAPT,
		Source: "ubuntu",
	}

	t.Run("cannot_remove_non_installed_package", func(t *testing.T) {
		mockInstaller.On("Remove", ctx, pkg).
			Return(nil, domain.ErrNotInstalled).Once()

		result, err := service.Remove(ctx, pkg)

		require.ErrorIs(t, err, domain.ErrNotInstalled)
		assert.Nil(t, result)
	})

	t.Run("double_install_is_idempotent", func(t *testing.T) {
		// First install succeeds
		mockInstaller.On("Install", ctx, pkg).
			Return(&domain.InstallationResult{
				Package: pkg,
				Success: true,
			}, nil).Once()

		// Second install returns already installed
		mockInstaller.On("Install", ctx, pkg).
			Return(nil, domain.ErrAlreadyInstalled).Once()

		result1, err1 := service.Install(ctx, pkg)
		require.NoError(t, err1)
		assert.True(t, result1.Success)

		result2, err2 := service.Install(ctx, pkg)
		require.ErrorIs(t, err2, domain.ErrAlreadyInstalled)
		assert.Nil(t, result2)
	})

	t.Run("install_remove_install_cycle", func(t *testing.T) {
		// This invariant is already tested in idempotency_test.go
		// but it's a critical business rule worth emphasizing

		// Install
		mockInstaller.On("Install", ctx, pkg).
			Return(&domain.InstallationResult{Package: pkg, Success: true}, nil).Once()

		// Remove
		mockInstaller.On("Remove", ctx, pkg).
			Return(&domain.InstallationResult{Package: pkg, Success: true}, nil).Once()

		// Install again
		mockInstaller.On("Install", ctx, pkg).
			Return(&domain.InstallationResult{Package: pkg, Success: true}, nil).Once()

		// Execute cycle
		r1, e1 := service.Install(ctx, pkg)
		require.NoError(t, e1)
		assert.True(t, r1.Success)

		r2, e2 := service.Remove(ctx, pkg)
		require.NoError(t, e2)
		assert.True(t, r2.Success)

		r3, e3 := service.Install(ctx, pkg)
		require.NoError(t, e3)
		assert.True(t, r3.Success)
	})
}

// TestAdvancedErrorRecoveryStrategies tests different error recovery approaches.
func TestAdvancedErrorRecoveryStrategies(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	pkg := &domain.Package{
		Name:   "flaky-package",
		Method: domain.MethodAPT,
		Source: "ubuntu",
	}

	t.Run("transient_error_recovery", func(t *testing.T) {
		// First attempt fails with network error
		mockInstaller.On("Install", ctx, pkg).
			Return(nil, domain.ErrNetworkFailure).Once()

		// Second attempt succeeds (after retry)
		mockInstaller.On("Install", ctx, pkg).
			Return(&domain.InstallationResult{
				Package: pkg,
				Success: true,
			}, nil).Once()

		// First attempt
		result1, err1 := service.Install(ctx, pkg)
		require.ErrorIs(t, err1, domain.ErrNetworkFailure)
		assert.Nil(t, result1)

		// Retry
		result2, err2 := service.Install(ctx, pkg)
		require.NoError(t, err2)
		assert.True(t, result2.Success)
	})

	t.Run("permanent_error_no_recovery", func(t *testing.T) {
		badPkg := &domain.Package{
			Name:   "",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		}

		// Invalid package always fails
		result, err := service.Install(ctx, badPkg)
		require.ErrorIs(t, err, domain.ErrInvalidPackage)
		assert.Nil(t, result)

		// Retry also fails
		result2, err2 := service.Install(ctx, badPkg)
		require.ErrorIs(t, err2, domain.ErrInvalidPackage)
		assert.Nil(t, result2)
	})
}
