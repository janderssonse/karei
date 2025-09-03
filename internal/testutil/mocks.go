// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package testutil

import (
	"context"
	"errors"
	"time"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/mock"
)

// MockPackageInstaller mocks the PackageInstaller port for testing.
type MockPackageInstaller struct {
	mock.Mock
}

// Install mocks package installation.
func (m *MockPackageInstaller) Install(ctx context.Context, pkg *domain.Package) (*domain.InstallationResult, error) {
	args := m.Called(ctx, pkg)
	if result := args.Get(0); result != nil {
		res, ok := result.(*domain.InstallationResult)
		if !ok {
			return nil, args.Error(1)
		}

		return res, args.Error(1)
	}

	return nil, args.Error(1)
}

// Remove mocks package removal.
func (m *MockPackageInstaller) Remove(ctx context.Context, pkg *domain.Package) (*domain.InstallationResult, error) {
	args := m.Called(ctx, pkg)
	if result := args.Get(0); result != nil {
		res, ok := result.(*domain.InstallationResult)
		if !ok {
			return nil, args.Error(1)
		}

		return res, args.Error(1)
	}

	return nil, args.Error(1)
}

// List mocks listing installed packages.
func (m *MockPackageInstaller) List(ctx context.Context) ([]*domain.Package, error) {
	args := m.Called(ctx)
	if result := args.Get(0); result != nil {
		res, ok := result.([]*domain.Package)
		if !ok {
			return nil, args.Error(1)
		}

		return res, args.Error(1)
	}

	return nil, args.Error(1)
}

// IsInstalled mocks checking if a package is installed.
func (m *MockPackageInstaller) IsInstalled(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

// GetBestMethod returns the best installation method for a source.
func (m *MockPackageInstaller) GetBestMethod(source string) domain.InstallMethod {
	args := m.Called(source)
	if method, ok := args.Get(0).(domain.InstallMethod); ok {
		return method
	}

	return domain.MethodAPT // Default to APT
}

// MockSystemDetector is a mock implementation of SystemDetector port
// for use in tests across multiple packages.
type MockSystemDetector struct {
	mock.Mock
}

// DetectSystem mocks the system detection.
func (m *MockSystemDetector) DetectSystem(ctx context.Context) (*domain.SystemInfo, error) {
	args := m.Called(ctx)
	if result := args.Get(0); result != nil {
		if info, ok := result.(*domain.SystemInfo); ok {
			return info, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

// DetectDistribution mocks distribution detection.
func (m *MockSystemDetector) DetectDistribution(ctx context.Context) (*domain.Distribution, error) {
	args := m.Called(ctx)
	if result := args.Get(0); result != nil {
		dist, ok := result.(*domain.Distribution)
		if !ok {
			return nil, args.Error(1)
		}

		return dist, args.Error(1)
	}

	return nil, args.Error(1)
}

// DetectDesktopEnvironment mocks desktop environment detection.
func (m *MockSystemDetector) DetectDesktopEnvironment(ctx context.Context) (*domain.DesktopEnvironment, error) {
	args := m.Called(ctx)
	if result := args.Get(0); result != nil {
		de, ok := result.(*domain.DesktopEnvironment)
		if !ok {
			return nil, args.Error(1)
		}

		return de, args.Error(1)
	}

	return nil, args.Error(1)
}

// DetectPackageManager mocks package manager detection.
func (m *MockSystemDetector) DetectPackageManager(ctx context.Context) (*domain.PackageManager, error) {
	args := m.Called(ctx)
	if result := args.Get(0); result != nil {
		pm, ok := result.(*domain.PackageManager)
		if !ok {
			return nil, args.Error(1)
		}

		return pm, args.Error(1)
	}

	return nil, args.Error(1)
}

// MockCommandRunner is a mock implementation of CommandRunner port.
type MockCommandRunner struct {
	mock.Mock
}

// Execute mocks command execution without output.
func (m *MockCommandRunner) Execute(ctx context.Context, name string, args ...string) error {
	// Convert variadic args to interface slice for mock.Called
	callArgs := make([]interface{}, 0, len(args)+2)

	callArgs = append(callArgs, ctx, name)
	for _, arg := range args {
		callArgs = append(callArgs, arg)
	}

	returnArgs := m.Called(callArgs...)

	return returnArgs.Error(0)
}

// ExecuteWithOutput mocks command execution with output.
func (m *MockCommandRunner) ExecuteWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	// Convert variadic args to interface slice for mock.Called
	callArgs := make([]interface{}, 0, len(args)+2)

	callArgs = append(callArgs, ctx, name)
	for _, arg := range args {
		callArgs = append(callArgs, arg)
	}

	returnArgs := m.Called(callArgs...)

	return returnArgs.String(0), returnArgs.Error(1)
}

// ExecuteSudo mocks sudo command execution.
func (m *MockCommandRunner) ExecuteSudo(ctx context.Context, name string, args ...string) error {
	callArgs := m.Called(ctx, name, args)
	return callArgs.Error(0)
}

// CommandExists mocks checking if a command exists.
func (m *MockCommandRunner) CommandExists(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

// MockNetworkClient is a mock implementation of NetworkClient port.
type MockNetworkClient struct {
	mock.Mock
}

// DownloadFile mocks file download.
func (m *MockNetworkClient) DownloadFile(ctx context.Context, url, destPath string) error {
	args := m.Called(ctx, url, destPath)
	return args.Error(0)
}

// MockFileManager is a mock implementation of FileManager port.
type MockFileManager struct {
	mock.Mock
}

// FileExists mocks checking if a file exists.
func (m *MockFileManager) FileExists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

// EnsureDir mocks ensuring a directory exists.
func (m *MockFileManager) EnsureDir(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// CopyFile mocks copying a file.
func (m *MockFileManager) CopyFile(src, dest string) error {
	args := m.Called(src, dest)
	return args.Error(0)
}

// WriteFile mocks writing data to a file.
func (m *MockFileManager) WriteFile(path string, data []byte) error {
	args := m.Called(path, data)
	return args.Error(0)
}

// ReadFile mocks reading data from a file.
func (m *MockFileManager) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	if result := args.Get(0); result != nil {
		bytes, ok := result.([]byte)
		if !ok {
			return nil, args.Error(1)
		}

		return bytes, args.Error(1)
	}

	return nil, args.Error(1)
}

// RemoveFile mocks removing a file.
func (m *MockFileManager) RemoveFile(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// ContextAwareMockInstaller checks context cancellation before operations.
// This replaces the complex SlowMockInstaller with timing dependencies.
type ContextAwareMockInstaller struct {
	MockPackageInstaller
}

// Install mocks package installation with context awareness.
func (m *ContextAwareMockInstaller) Install(ctx context.Context, pkg *domain.Package) (*domain.InstallationResult, error) {
	// Check context immediately - no timing dependencies
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return m.MockPackageInstaller.Install(ctx, pkg)
}

// Test helpers

// CreateValidPackage creates a valid test package with all required fields.
func CreateValidPackage(name string) *domain.Package {
	return &domain.Package{
		Name:   name,
		Method: domain.MethodAPT,
		Source: name,
	}
}

// CreateInvalidPackage creates an invalid test package missing required fields.
func CreateInvalidPackage() *domain.Package {
	return &domain.Package{
		Name: "missing-fields",
		// Missing Method and Source
	}
}

// CreateTestSystemInfo creates a valid SystemInfo for testing.
func CreateTestSystemInfo() *domain.SystemInfo {
	return &domain.SystemInfo{
		Distribution: &domain.Distribution{
			Name:    "Ubuntu",
			ID:      "ubuntu",
			Version: "22.04",
			Family:  "debian",
		},
		PackageManager: &domain.PackageManager{
			Name:    "apt",
			Method:  domain.MethodAPT,
			Command: "apt-get",
		},
		Architecture: "amd64",
		Kernel:       "5.15.0",
	}
}

// AssertInstallationSuccess verifies an installation result succeeded.
func AssertInstallationSuccess(t interface {
	Errorf(format string, args ...interface{})
}, result *domain.InstallationResult, err error) {
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Errorf("Expected non-nil result")
		return
	}

	if !result.Success {
		t.Errorf("Expected success=true, got false")
	}
}

// AssertInstallationFailure verifies an installation result failed with expected error.
func AssertInstallationFailure(t interface {
	Errorf(format string, args ...interface{})
}, result *domain.InstallationResult, err error, expectedErr error) {
	if err == nil {
		t.Errorf("Expected error %v, got nil", expectedErr)
	}

	if expectedErr != nil && !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	if result != nil && result.Success {
		t.Errorf("Expected success=false or nil result")
	}
}

// WaitWithTimeout runs a function with a timeout.
func WaitWithTimeout(fn func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}

		time.Sleep(10 * time.Millisecond)
	}

	return false
}

// SetupServiceMocks creates commonly used mock setup for domain tests.
func SetupServiceMocks() (*MockPackageInstaller, *MockSystemDetector, *domain.PackageService) {
	mockInstaller := new(MockPackageInstaller)
	mockDetector := new(MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)

	return mockInstaller, mockDetector, service
}
