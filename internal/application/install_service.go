// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application

import (
	"context"
	"fmt"

	"github.com/janderssonse/karei/internal/domain"
)

// InstallService installs packages with automatic system detection.
type InstallService struct {
	packageService *domain.PackageService
	systemDetector domain.SystemDetector
}

// NewInstallService creates an InstallService.
func NewInstallService(packageService *domain.PackageService, systemDetector domain.SystemDetector) *InstallService {
	return &InstallService{
		packageService: packageService,
		systemDetector: systemDetector,
	}
}

// InstallApplication installs an application, automatically detecting the best method.
func (s *InstallService) InstallApplication(ctx context.Context, name, source string) (*domain.InstallationResult, error) {
	// Detect system information
	systemInfo, err := s.systemDetector.DetectSystem(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect system: %w", err)
	}

	// Create package with system-appropriate method
	pkg := &domain.Package{
		Name:   name,
		Source: source,
		Method: s.getBestMethodForSystem(source, systemInfo),
	}

	// Install the package
	return s.packageService.Install(ctx, pkg)
}

// InstallMultipleApplications installs multiple applications in sequence.
func (s *InstallService) InstallMultipleApplications(ctx context.Context, apps map[string]string) ([]*domain.InstallationResult, error) {
	results := make([]*domain.InstallationResult, 0, len(apps))

	for name, source := range apps {
		result, err := s.InstallApplication(ctx, name, source)
		if err != nil {
			// Continue with other installations even if one fails
			result = &domain.InstallationResult{
				Package: &domain.Package{Name: name, Source: source},
				Success: false,
				Error:   err,
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// GetSystemInfo detects the current system's distribution and package manager.
func (s *InstallService) GetSystemInfo(ctx context.Context) (*domain.SystemInfo, error) {
	return s.systemDetector.DetectSystem(ctx)
}

// ListInstalledPackages returns all installed packages.
func (s *InstallService) ListInstalledPackages(ctx context.Context) ([]*domain.Package, error) {
	return s.packageService.List(ctx)
}

// getBestMethodForSystem determines the best installation method based on source and system.
func (s *InstallService) getBestMethodForSystem(_ string, systemInfo *domain.SystemInfo) domain.InstallMethod {
	// For Ubuntu/Debian systems
	if systemInfo.IsDebianBased() {
		if systemInfo.PackageManager.Method == domain.MethodAPT {
			return domain.MethodAPT
		}
	}

	// For Fedora/RHEL systems
	if systemInfo.IsFedora() {
		if systemInfo.PackageManager.Method == domain.MethodDNF {
			return domain.MethodDNF
		}

		if systemInfo.PackageManager.Method == domain.MethodYum {
			return domain.MethodYum
		}
	}

	// For Arch systems
	if systemInfo.IsArch() {
		return domain.MethodPacman
	}

	// Default to the system's package manager
	return systemInfo.PackageManager.Method
}
