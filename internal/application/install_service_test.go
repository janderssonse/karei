// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application

import (
	"context"
	"testing"

	"github.com/janderssonse/karei/internal/adapters/platform"
	"github.com/janderssonse/karei/internal/adapters/ubuntu"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallService_InstallApplication(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		packageName string
		sourceName  string
		expected    struct {
			success bool
			method  domain.InstallMethod
			name    string
		}
		osRelease string
	}{
		{
			name:        "successful vim installation",
			packageName: "vim",
			sourceName:  "vim",
			expected: struct {
				success bool
				method  domain.InstallMethod
				name    string
			}{
				success: true,
				method:  domain.MethodAPT,
				name:    "vim",
			},
			osRelease: `NAME="Ubuntu"
ID=ubuntu
VERSION="22.04"
VERSION_CODENAME=jammy`,
		},
		{
			name:        "successful git installation",
			packageName: "git",
			sourceName:  "git",
			expected: struct {
				success bool
				method  domain.InstallMethod
				name    string
			}{
				success: true,
				method:  domain.MethodAPT,
				name:    "git",
			},
			osRelease: `NAME="Ubuntu"
ID=ubuntu
VERSION="24.04"
VERSION_CODENAME=noble`,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Create isolated mock dependencies for each test
			commandRunner := platform.NewMockCommandRunner(false)
			fileManager := platform.NewMockFileManager(false)
			systemDetector := platform.NewSystemDetector(commandRunner, fileManager)

			// Set up mock system information
			fileManager.SetMockFile("/etc/os-release", []byte(testCase.osRelease))

			// Create adapters with dry run enabled (no system changes)
			packageInstaller := ubuntu.NewPackageInstaller(commandRunner, fileManager, false, true)
			packageService := domain.NewPackageService(packageInstaller, systemDetector)

			// Create application service
			installService := NewInstallService(packageService, systemDetector)
			require.NotNil(t, installService, "InstallService should be created successfully")

			// Test installing an application (no real system changes)
			result, err := installService.InstallApplication(context.Background(), testCase.packageName, testCase.sourceName)
			require.NoError(t, err, "InstallApplication should not error in dry run mode")
			require.NotNil(t, result, "Result should not be nil")

			// Use assert for non-critical verifications
			assert.Equal(t, testCase.expected.success, result.Success, "Installation success status should matestCaseh expected")
			assert.Equal(t, testCase.expected.name, result.Package.Name, "Package name should matestCaseh expected")
			assert.Equal(t, testCase.expected.method, result.Package.Method, "Package method should matestCaseh expected")

			// Verify no dangerous operations were attempted
			assert.NotContains(t, result.Package.Name, "/", "Package name should not contain filesystem paths")
			assert.NotEmpty(t, result.Package.Name, "Package name should not be empty")
		})
	}
}

func TestInstallService_GetSystemInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		osRelease    string
		kernelOutput string
		expected     struct {
			distributionID     string
			distributionFamily string
			packageMethod      domain.InstallMethod
		}
	}{
		{
			name: "Ubuntu 22.04 LTS detection",
			osRelease: `NAME="Ubuntu"
ID=ubuntu
VERSION="22.04"
VERSION_CODENAME=jammy`,
			kernelOutput: "5.15.0-generic",
			expected: struct {
				distributionID     string
				distributionFamily string
				packageMethod      domain.InstallMethod
			}{
				distributionID:     "ubuntu",
				distributionFamily: "debian",
				packageMethod:      domain.MethodAPT,
			},
		},
		{
			name: "Ubuntu 24.04 LTS detection",
			osRelease: `NAME="Ubuntu"
ID=ubuntu
VERSION="24.04"
VERSION_CODENAME=noble`,
			kernelOutput: "6.8.0-generic",
			expected: struct {
				distributionID     string
				distributionFamily string
				packageMethod      domain.InstallMethod
			}{
				distributionID:     "ubuntu",
				distributionFamily: "debian",
				packageMethod:      domain.MethodAPT,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Create isolated mock dependencies for each test
			commandRunner := platform.NewMockCommandRunner(false)
			fileManager := platform.NewMockFileManager(false)
			systemDetector := platform.NewSystemDetector(commandRunner, fileManager)

			// Set up mock system information
			fileManager.SetMockFile("/etc/os-release", []byte(testCase.osRelease))

			// Mock command outputs for kernel detection
			commandRunner.SetMockOutput("uname -r", testCase.kernelOutput)

			// Create application service (package installer not needed for system info)
			installService := NewInstallService(nil, systemDetector)
			require.NotNil(t, installService, "InstallService should be created successfully")

			// Test getting system info (no network calls or system changes)
			systemInfo, err := installService.GetSystemInfo(context.Background())
			require.NoError(t, err, "GetSystemInfo should not error with valid mocks")
			require.NotNil(t, systemInfo, "SystemInfo should not be nil")

			// Use assert for system information verification
			assert.Equal(t, testCase.expected.distributionID, systemInfo.Distribution.ID,
				"Distribution ID should match expected")
			assert.Equal(t, testCase.expected.distributionFamily, systemInfo.Distribution.Family,
				"Distribution family should match expected")
			assert.Equal(t, testCase.expected.packageMethod, systemInfo.PackageManager.Method,
				"Package manager method should matestCaseh expected")

			// Verify system info completeness
			assert.NotEmpty(t, systemInfo.Distribution.ID, "Distribution ID should not be empty")
			assert.NotEmpty(t, systemInfo.Distribution.Family, "Distribution family should not be empty")
		})
	}
}

// TestInstallService_ThreadSafety demonstrates thread-safe operations.
func TestInstallService_ThreadSafety(t *testing.T) {
	t.Parallel()

	// Create shared mock dependencies (read-only operations)
	commandRunner := platform.NewMockCommandRunner(false)
	fileManager := platform.NewMockFileManager(false)
	systemDetector := platform.NewSystemDetector(commandRunner, fileManager)

	// Set up mock system information
	fileManager.SetMockFile("/etc/os-release", []byte(`NAME="Ubuntu"
ID=ubuntu
VERSION="22.04"
VERSION_CODENAME=jammy`))
	commandRunner.SetMockOutput("uname -r", "5.15.0-generic")

	// Create install service
	installService := NewInstallService(nil, systemDetector)
	require.NotNil(t, installService)

	// Run concurrent operations to test thread safety
	done := make(chan bool, 10)

	for index := range 10 {
		go func(_ int) {
			defer func() { done <- true }()

			// Each goroutine performs read-only operations
			systemInfo, err := installService.GetSystemInfo(context.Background())
			assert.NoError(t, err, "GetSystemInfo should not error in concurrent access")
			assert.NotNil(t, systemInfo, "SystemInfo should not be nil")
			assert.Equal(t, "ubuntu", systemInfo.Distribution.ID,
				"Distribution ID should be consistent across concurrent calls")
		}(index)
	}

	// Wait for all operations to complete
	for range 10 {
		<-done
	}
}
