// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application_test

import (
	"context"
	"testing"

	"github.com/janderssonse/karei/internal/application"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPackageInstaller for testing.
type MockPackageInstaller struct {
	mock.Mock
}

func (m *MockPackageInstaller) Install(ctx context.Context, pkg *domain.Package) (*domain.InstallationResult, error) {
	args := m.Called(ctx, pkg)

	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}

	res, ok := result.(*domain.InstallationResult)
	if !ok {
		return nil, args.Error(1)
	}

	return res, args.Error(1)
}

func (m *MockPackageInstaller) Remove(ctx context.Context, pkg *domain.Package) (*domain.InstallationResult, error) {
	args := m.Called(ctx, pkg)

	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}

	res, ok := result.(*domain.InstallationResult)
	if !ok {
		return nil, args.Error(1)
	}

	return res, args.Error(1)
}

func (m *MockPackageInstaller) List(ctx context.Context) ([]*domain.Package, error) {
	args := m.Called(ctx)

	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}

	res, ok := result.([]*domain.Package)
	if !ok {
		return nil, args.Error(1)
	}

	return res, args.Error(1)
}

func (m *MockPackageInstaller) IsInstalled(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockPackageInstaller) GetBestMethod(source string) domain.InstallMethod {
	args := m.Called(source)
	result := args.Get(0)

	method, ok := result.(domain.InstallMethod)
	if !ok {
		return domain.MethodAPT // Default fallback
	}

	return method
}

// MockCommandRunner for testing.
type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Execute(ctx context.Context, name string, args ...string) error {
	mockArgs := m.Called(ctx, name, args)
	return mockArgs.Error(0)
}

func (m *MockCommandRunner) ExecuteWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	mockArgs := m.Called(ctx, name, args)
	return mockArgs.String(0), mockArgs.Error(1)
}

func (m *MockCommandRunner) ExecuteSudo(ctx context.Context, name string, args ...string) error {
	mockArgs := m.Called(ctx, name, args)
	return mockArgs.Error(0)
}

func (m *MockCommandRunner) CommandExists(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func TestStatusService_GetSystemStatus(t *testing.T) {
	ctx := context.Background()

	// Create mocks
	mockInstaller := new(MockPackageInstaller)
	mockRunner := new(MockCommandRunner)

	// Setup expectations
	packages := []*domain.Package{
		{Name: "git"},
		{Name: "vim"},
		{Name: "docker"},
	}
	mockInstaller.On("List", ctx).Return(packages, nil)

	// Mock df output
	dfOutput := `Filesystem     1G-blocks  Used Available Use% Mounted on
/dev/sda1           234G   45G      177G  21% /`
	mockRunner.On("ExecuteWithOutput", ctx, "df", []string{"-BG", "/"}).Return(dfOutput, nil)

	// Mock uptime output
	uptimeOutput := "up 2 days, 3 hours, 15 minutes"
	mockRunner.On("ExecuteWithOutput", ctx, "uptime", []string{"-p"}).Return(uptimeOutput, nil)

	// Create service and test
	service := application.NewStatusService(mockInstaller, mockRunner)
	status, err := service.GetSystemStatus(ctx)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, 3, status.InstalledApps)
	assert.Equal(t, 126, status.AvailableApps) // Hardcoded for now
	assert.InDelta(t, 45.0, status.DiskUsageGB, 0.01)
	assert.InDelta(t, 177.0, status.DiskAvailGB, 0.01)
	assert.InDelta(t, 51.0, status.UptimeHours, 0.01) // 2*24 + 3

	// Verify mock expectations
	mockInstaller.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestFormatDiskSpace(t *testing.T) {
	tests := []struct {
		gb   float64
		want string
	}{
		{0.5, "512 MB"},
		{1.0, "1.0 GB"},
		{45.3, "45.3 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := application.FormatDiskSpace(tt.gb)
			assert.Equal(t, tt.want, got)
		})
	}
}
