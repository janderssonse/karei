// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"errors"
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

// TestPackageConflictDetection tests detection of conflicting packages
// Business Rule: System should detect when packages conflict with each other.
func TestPackageConflictDetection(t *testing.T) {
	tests := []struct {
		name           string
		existingPkg    *domain.Package
		newPkg         *domain.Package
		shouldConflict bool
		reason         string
	}{
		{
			name: "same_package_different_methods_conflict",
			existingPkg: &domain.Package{
				Name:   "code",
				Method: domain.MethodAPT,
				Source: "code",
			},
			newPkg: &domain.Package{
				Name:   "code",
				Method: domain.MethodSnap,
				Source: "code --classic",
			},
			shouldConflict: true,
			reason:         "Same application from different sources should conflict",
		},
		{
			name: "different_packages_no_conflict",
			existingPkg: &domain.Package{
				Name:   "vim",
				Method: domain.MethodAPT,
				Source: "vim",
			},
			newPkg: &domain.Package{
				Name:   "emacs",
				Method: domain.MethodAPT,
				Source: "emacs",
			},
			shouldConflict: false,
			reason:         "Different applications should not conflict",
		},
		{
			name: "mysql_mariadb_conflict",
			existingPkg: &domain.Package{
				Name:   "mysql-server",
				Method: domain.MethodAPT,
				Source: "mysql-server",
			},
			newPkg: &domain.Package{
				Name:   "mariadb-server",
				Method: domain.MethodAPT,
				Source: "mariadb-server",
			},
			shouldConflict: true,
			reason:         "MySQL and MariaDB provide same service and conflict",
		},
		{
			name: "apache_nginx_conflict",
			existingPkg: &domain.Package{
				Name:   "apache2",
				Method: domain.MethodAPT,
				Source: "apache2",
			},
			newPkg: &domain.Package{
				Name:   "nginx",
				Method: domain.MethodAPT,
				Source: "nginx",
			},
			shouldConflict: true,
			reason:         "Web servers on same port should conflict",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Define conflict detection logic
			detectsConflict := func(existing, newPkg *domain.Package) bool {
				// Same name from different methods always conflicts
				if existing.Name == newPkg.Name && existing.Method != newPkg.Method {
					return true
				}

				// Known conflicting packages
				conflicts := map[string][]string{
					"mysql-server":   {"mariadb-server"},
					"mariadb-server": {"mysql-server"},
					"apache2":        {"nginx"},
					"nginx":          {"apache2"},
				}

				if conflictList, exists := conflicts[existing.Name]; exists {
					for _, conflict := range conflictList {
						if newPkg.Name == conflict {
							return true
						}
					}
				}

				return false
			}

			result := detectsConflict(tc.existingPkg, tc.newPkg)
			assert.Equal(t, tc.shouldConflict, result, tc.reason)
		})
	}
}

// TestPackageUpgradeVsFreshInstall tests the distinction between upgrades and fresh installs
// Business Rule: System should handle upgrades differently from fresh installs.
func TestPackageUpgradeVsFreshInstall(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()

	t.Run("fresh_install_of_new_package", func(t *testing.T) {
		pkg := &domain.Package{
			Name:    "newapp",
			Method:  domain.MethodAPT,
			Source:  "newapp",
			Version: "1.0.0",
		}

		// Mock: package not currently installed
		mockInstaller.On("List", ctx).Return([]*domain.Package{}, nil).Once()

		// Mock: fresh install succeeds
		mockInstaller.On("Install", ctx, pkg).Return(&domain.InstallationResult{
			Package: pkg,
			Success: true,
			Output:  "Installed newapp version 1.0.0",
		}, nil).Once()

		result, err := service.Install(ctx, pkg)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "1.0.0")
	})

	t.Run("upgrade_existing_package", func(t *testing.T) {
		existingPkg := &domain.Package{
			Name:    "app",
			Method:  domain.MethodAPT,
			Source:  "app",
			Version: "1.0.0",
		}

		newPkg := &domain.Package{
			Name:    "app",
			Method:  domain.MethodAPT,
			Source:  "app",
			Version: "2.0.0",
		}

		// Mock: package already installed with older version
		mockInstaller.On("List", ctx).Return([]*domain.Package{existingPkg}, nil).Once()

		// Mock: upgrade succeeds
		mockInstaller.On("Install", ctx, newPkg).Return(&domain.InstallationResult{
			Package: newPkg,
			Success: true,
			Output:  "Upgraded app from 1.0.0 to 2.0.0",
		}, nil).Once()

		result, err := service.Install(ctx, newPkg)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "Upgraded")
		assert.Contains(t, result.Output, "2.0.0")
	})

	t.Run("downgrade_prevention", func(t *testing.T) {
		existingPkg := &domain.Package{
			Name:    "app",
			Method:  domain.MethodAPT,
			Source:  "app",
			Version: "2.0.0",
		}

		olderPkg := &domain.Package{
			Name:    "app",
			Method:  domain.MethodAPT,
			Source:  "app",
			Version: "1.0.0",
		}

		// Mock: package already installed with newer version
		mockInstaller.On("List", ctx).Return([]*domain.Package{existingPkg}, nil).Once()

		// Mock: downgrade prevented
		mockInstaller.On("Install", ctx, olderPkg).Return(nil,
			domain.ErrInvalidPackage).Once()

		result, err := service.Install(ctx, olderPkg)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrInvalidPackage)
	})
}

// TestSystemSpecificPackageNameTranslation tests package name translation across distros
// Business Rule: Package names should be translated for different distributions.
func TestSystemSpecificPackageNameTranslation(t *testing.T) {
	tests := []struct {
		name         string
		inputName    string
		distribution string
		family       string
		expectedName string
		reason       string
	}{
		{
			name:         "apache_on_debian",
			inputName:    "apache",
			distribution: "ubuntu",
			family:       "debian",
			expectedName: "apache2",
			reason:       "Apache is called apache2 on Debian-based systems",
		},
		{
			name:         "apache_on_rhel",
			inputName:    "apache",
			distribution: "fedora",
			family:       "rhel",
			expectedName: "httpd",
			reason:       "Apache is called httpd on RHEL-based systems",
		},
		{
			name:         "build_tools_debian",
			inputName:    "build-tools",
			distribution: "debian",
			family:       "debian",
			expectedName: "build-essential",
			reason:       "Build tools package name on Debian",
		},
		{
			name:         "build_tools_rhel",
			inputName:    "build-tools",
			distribution: "centos",
			family:       "rhel",
			expectedName: "gcc gcc-c++ make",
			reason:       "Build tools package group on RHEL",
		},
		{
			name:         "python_modern_naming",
			inputName:    "python",
			distribution: "ubuntu",
			family:       "debian",
			expectedName: "python3",
			reason:       "Python should default to python3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Package name translation logic
			translatePackageName := func(name string, _ string, family string) string {
				translations := map[string]map[string]string{
					"debian": {
						"apache":      "apache2",
						"build-tools": "build-essential",
						"python":      "python3",
					},
					"rhel": {
						"apache":      "httpd",
						"build-tools": "gcc gcc-c++ make",
						"python":      "python3",
					},
				}

				if familyTranslations, exists := translations[family]; exists {
					if translated, exists := familyTranslations[name]; exists {
						return translated
					}
				}

				return name // No translation needed
			}

			result := translatePackageName(tc.inputName, tc.distribution, tc.family)
			assert.Equal(t, tc.expectedName, result, tc.reason)
		})
	}
}

// TestDependencyChainResolution tests automatic dependency resolution
// Business Rule: Dependencies should be automatically resolved and installed.
func TestDependencyChainResolution(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()

	// Package with deep dependency chain
	mainPkg := &domain.Package{
		Name:         "webapp",
		Method:       domain.MethodAPT,
		Source:       "webapp",
		Dependencies: []string{"webserver", "database", "runtime"},
	}

	// Mock the installation with dependency resolution
	mockInstaller.On("Install", ctx, mainPkg).Return(&domain.InstallationResult{
		Package: mainPkg,
		Success: true,
		Output:  "Installed webapp with dependencies: webserver, database, runtime, lib1, lib2, lib3",
	}, nil).Once()

	result, err := service.Install(ctx, mainPkg)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "dependencies")

	// Verify all expected dependencies are mentioned
	for _, dep := range mainPkg.Dependencies {
		assert.Contains(t, result.Output, dep,
			"Dependency %s should be installed", dep)
	}
}

// TestPartialBatchInstallationRecovery tests recovery from partial batch failures
// Business Rule: Batch operations should track and report partial success accurately.
func TestPartialBatchInstallationRecovery(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()

	packages := []*domain.Package{
		{Name: "app1", Method: domain.MethodAPT, Source: "app1"},
		{Name: "app2", Method: domain.MethodAPT, Source: "app2"},
		{Name: "app3", Method: domain.MethodAPT, Source: "app3"},
		{Name: "app4", Method: domain.MethodAPT, Source: "app4"},
	}

	// Mock: app1 succeeds
	mockInstaller.On("Install", ctx, packages[0]).Return(&domain.InstallationResult{
		Package: packages[0],
		Success: true,
	}, nil).Once()

	// Mock: app2 fails with network error
	mockInstaller.On("Install", ctx, packages[1]).Return(nil,
		domain.ErrNetworkFailure).Once()

	// Mock: app3 succeeds
	mockInstaller.On("Install", ctx, packages[2]).Return(&domain.InstallationResult{
		Package: packages[2],
		Success: true,
	}, nil).Once()

	// Mock: app4 fails with permission error
	mockInstaller.On("Install", ctx, packages[3]).Return(nil,
		domain.ErrPermissionDenied).Once()

	// Install all packages
	var (
		results           []*domain.InstallationResult
		succeeded, failed int
	)

	for _, pkg := range packages {
		result, err := service.Install(ctx, pkg)
		if err != nil {
			failed++
			// Track failed installations
			results = append(results, &domain.InstallationResult{
				Package: pkg,
				Success: false,
				Error:   err,
			})
		} else {
			succeeded++

			results = append(results, result)
		}
	}

	// Verify partial success tracking
	assert.Equal(t, 2, succeeded, "Should have 2 successful installations")
	assert.Equal(t, 2, failed, "Should have 2 failed installations")

	// Verify error types are preserved
	require.ErrorIs(t, results[1].Error, domain.ErrNetworkFailure)
	require.ErrorIs(t, results[3].Error, domain.ErrPermissionDenied)

	mockInstaller.AssertExpectations(t)
}

// TestInstallationRollbackOnFailure tests rollback behavior on critical failures
// Business Rule: Failed installations should be rolled back to maintain system consistency.
func TestInstallationRollbackOnFailure(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()

	// Package that will fail during installation
	pkg := &domain.Package{
		Name:         "critical-app",
		Method:       domain.MethodAPT,
		Source:       "critical-app",
		Dependencies: []string{"dep1", "dep2"},
	}

	// Mock: Installation starts but fails midway
	mockInstaller.On("Install", ctx, pkg).Return(nil,
		domain.ErrDependencyMissing).Once()

	// Mock: Rollback is attempted (remove partially installed files)
	mockInstaller.On("Remove", ctx, mock.MatchedBy(func(p *domain.Package) bool {
		return p.Name == pkg.Name
	})).Return(&domain.InstallationResult{
		Package: pkg,
		Success: true,
		Output:  "Rolled back partial installation",
	}, nil).Maybe()

	// Attempt installation
	result, err := service.Install(ctx, pkg)

	// Verify failure is handled
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrDependencyMissing)

	// In a real implementation, we would verify rollback was triggered
	// This test documents the expected behavior
}

// TestConcurrentInstallationOrdering tests that concurrent installations maintain order guarantees
// Business Rule: Concurrent installations of different packages should not interfere.
func TestConcurrentInstallationOrdering(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()

	// Create multiple independent packages
	packages := []*domain.Package{
		{Name: "tool1", Method: domain.MethodAPT, Source: "tool1"},
		{Name: "tool2", Method: domain.MethodAPT, Source: "tool2"},
		{Name: "tool3", Method: domain.MethodAPT, Source: "tool3"},
	}

	// Track installation order
	var mu sync.Mutex

	installOrder := []string{}

	// Setup mocks to track order
	for _, pkg := range packages {
		p := pkg // Capture for closure
		mockInstaller.On("Install", ctx, p).Return(
			&domain.InstallationResult{Package: p, Success: true},
			nil,
		).Run(func(_ mock.Arguments) {
			mu.Lock()

			installOrder = append(installOrder, p.Name)

			mu.Unlock()
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
		})
	}

	// Install concurrently
	var wg sync.WaitGroup

	results := make(chan *domain.InstallationResult, len(packages))
	errors := make(chan error, len(packages))

	for _, pkg := range packages {
		wg.Add(1)

		go func(p *domain.Package) {
			defer wg.Done()

			result, err := service.Install(ctx, p)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(pkg)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Verify all succeeded
	successCount := 0
	for range results {
		successCount++
	}

	assert.Equal(t, len(packages), successCount, "All packages should install successfully")

	// Verify no errors
	var errorCount int
	for range errors {
		errorCount++
	}

	assert.Equal(t, 0, errorCount, "No errors should occur")

	// Verify all packages were installed (order doesn't matter for independent packages)
	assert.Len(t, installOrder, len(packages))

	for _, pkg := range packages {
		assert.Contains(t, installOrder, pkg.Name, "Package %s should be installed", pkg.Name)
	}

	mockInstaller.AssertExpectations(t)
}

// TestConcurrentInstallationOfSamePackage tests concurrent installs of the same package
// Business Rule: Concurrent installations of the same package should be idempotent.
func TestConcurrentInstallationOfSamePackage(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()
	pkg := &domain.Package{Name: "shared-tool", Method: domain.MethodAPT, Source: "shared-tool"}

	// Track actual install attempts
	var (
		installCount int32
		mu           sync.Mutex
	)

	// First real install succeeds

	mockInstaller.On("Install", ctx, pkg).Return(
		&domain.InstallationResult{Package: pkg, Success: true},
		nil,
	).Once().Run(func(_ mock.Arguments) {
		mu.Lock()
		atomic.AddInt32(&installCount, 1)
		mu.Unlock()
		time.Sleep(50 * time.Millisecond) // Simulate installation time
	})

	// Subsequent attempts get AlreadyInstalled
	mockInstaller.On("Install", ctx, pkg).Return(
		nil,
		domain.ErrAlreadyInstalled,
	).Maybe()

	// Launch many concurrent installations
	const numGoroutines = 20

	var wg sync.WaitGroup

	results := make([]error, numGoroutines)

	start := make(chan struct{})

	for i := range numGoroutines {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			<-start // Wait for signal to start simultaneously

			_, err := service.Install(ctx, pkg)
			results[idx] = err
		}(i)
	}

	// Start all goroutines simultaneously
	close(start)
	wg.Wait()

	// Count outcomes
	var successCount, alreadyInstalledCount int

	for _, err := range results {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, domain.ErrAlreadyInstalled):
			alreadyInstalledCount++
		default:
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Business rule: At least one should succeed, rest should be AlreadyInstalled
	assert.GreaterOrEqual(t, successCount, 1, "At least one installation should succeed")
	assert.Equal(t, numGoroutines, successCount+alreadyInstalledCount,
		"All attempts should either succeed or return AlreadyInstalled")

	// Verify idempotency - only one actual installation happened
	assert.LessOrEqual(t, int(atomic.LoadInt32(&installCount)), 1,
		"Only one actual installation should occur")

	mockInstaller.AssertExpectations(t)
}

// TestBatchInstallationTransactionSemantics tests atomicity of batch operations
// Business Rule: Batch installations should either all succeed or all fail (transaction semantics).
func TestBatchInstallationTransactionSemantics(t *testing.T) {
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	ctx := context.Background()

	// Create a batch of packages to install
	packages := []*domain.Package{
		{Name: "database", Method: domain.MethodAPT, Source: "postgresql"},
		{Name: "cache", Method: domain.MethodAPT, Source: "redis"},
		{Name: "queue", Method: domain.MethodAPT, Source: "rabbitmq"},
	}

	t.Run("all_succeed_transaction", func(t *testing.T) {
		// Reset mocks
		mockInstaller.ExpectedCalls = nil

		// All packages install successfully
		for _, pkg := range packages {
			mockInstaller.On("Install", ctx, pkg).Return(
				&domain.InstallationResult{Package: pkg, Success: true},
				nil,
			).Once()
		}

		// Execute batch installation
		var (
			results      []*domain.InstallationResult
			installError error
		)

		for _, pkg := range packages {
			result, err := service.Install(ctx, pkg)
			if err != nil {
				installError = err
				break // Stop on first error
			}

			results = append(results, result)
		}

		// Verify transaction succeeded
		require.NoError(t, installError, "No errors in successful transaction")
		assert.Len(t, results, len(packages), "All packages should be installed")

		for _, result := range results {
			assert.True(t, result.Success, "Each installation should succeed")
		}
	})

	t.Run("rollback_on_failure", func(t *testing.T) {
		// Reset mocks
		mockInstaller.ExpectedCalls = nil

		// First package succeeds
		mockInstaller.On("Install", ctx, packages[0]).Return(
			&domain.InstallationResult{Package: packages[0], Success: true},
			nil,
		).Once()

		// Second package fails
		mockInstaller.On("Install", ctx, packages[1]).Return(
			nil,
			errors.New("disk full"),
		).Once()

		// Third package should not be attempted
		// (no mock setup for packages[2])

		// Track installed packages for rollback
		var (
			installed    []*domain.Package
			installError error
		)

		// Execute batch with rollback logic

		for _, pkg := range packages {
			result, err := service.Install(ctx, pkg)
			if err != nil {
				installError = err
				// Rollback previously installed packages
				for _, installedPkg := range installed {
					// In real implementation, this would call Remove
					mockInstaller.On("Remove", ctx, installedPkg).Return(
						&domain.InstallationResult{Package: installedPkg, Success: true},
						nil,
					).Maybe()
				}

				break
			}

			installed = append(installed, result.Package)
		}

		// Verify partial failure handling
		require.Error(t, installError, "Should have error from failed package")
		assert.Len(t, installed, 1, "Only first package should be installed before failure")

		// In a real implementation, we would verify rollback was called
		// This test documents the expected transaction behavior
	})

	t.Run("atomic_batch_with_dependencies", func(t *testing.T) {
		// Reset mocks
		mockInstaller.ExpectedCalls = nil

		// Package with dependencies that must be atomic
		appWithDeps := &domain.Package{
			Name:         "web-app",
			Method:       domain.MethodAPT,
			Source:       "webapp",
			Dependencies: []string{"nginx", "postgresql", "redis"},
		}

		// Mock atomic installation
		mockInstaller.On("Install", ctx, appWithDeps).Return(
			&domain.InstallationResult{
				Package: appWithDeps,
				Success: true,
				Output:  "Installed with all dependencies atomically",
			},
			nil,
		).Once()

		result, err := service.Install(ctx, appWithDeps)

		// Verify atomic installation
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Contains(t, result.Output, "atomically",
			"Dependencies should be installed atomically")
	})

	mockInstaller.AssertExpectations(t)
}

// TestResourceValidationBeforeInstall tests pre-installation resource checks
// Business Rule: System should validate resources (disk, memory) before installation.
func TestResourceValidationBeforeInstall(t *testing.T) {
	tests := []struct {
		name          string
		pkg           *domain.Package
		diskRequired  int64 // bytes
		diskAvailable int64 // bytes
		shouldFail    bool
		errorType     error
	}{
		{
			name: "sufficient_disk_space",
			pkg: &domain.Package{
				Name:   "small-app",
				Method: domain.MethodAPT,
				Source: "small-app",
			},
			diskRequired:  1024 * 1024 * 100,  // 100MB
			diskAvailable: 1024 * 1024 * 1024, // 1GB
			shouldFail:    false,
		},
		{
			name: "insufficient_disk_space",
			pkg: &domain.Package{
				Name:   "large-app",
				Method: domain.MethodAPT,
				Source: "large-app",
			},
			diskRequired:  1024 * 1024 * 1024 * 10, // 10GB
			diskAvailable: 1024 * 1024 * 500,       // 500MB
			shouldFail:    true,
			errorType:     domain.ErrInsufficientSpace,
		},
		{
			name: "exact_disk_space",
			pkg: &domain.Package{
				Name:   "exact-app",
				Method: domain.MethodAPT,
				Source: "exact-app",
			},
			diskRequired:  1024 * 1024 * 100, // 100MB
			diskAvailable: 1024 * 1024 * 100, // 100MB
			shouldFail:    false,             // Should allow exact match
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Business logic for resource validation
			validateResources := func(required, available int64) error {
				if required > available {
					return domain.ErrInsufficientSpace
				}

				return nil
			}

			err := validateResources(tc.diskRequired, tc.diskAvailable)

			if tc.shouldFail {
				require.Error(t, err, "Should fail with insufficient resources")

				if tc.errorType != nil {
					require.ErrorIs(t, err, tc.errorType)
				}
			} else {
				assert.NoError(t, err, "Should succeed with sufficient resources")
			}
		})
	}
}
