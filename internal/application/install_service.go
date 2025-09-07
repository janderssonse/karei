// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/domain"
)

// InstallService orchestrates package installation with automatic method selection.
type InstallService struct {
	packageService *domain.PackageService
	systemDetector domain.SystemDetector
	appsManager    *apps.Manager
	verbose        bool
}

// NewInstallService creates a service with system detection capabilities.
func NewInstallService(packageService *domain.PackageService, systemDetector domain.SystemDetector) *InstallService {
	return &InstallService{
		packageService: packageService,
		systemDetector: systemDetector,
		appsManager:    apps.NewManager(false), // Non-verbose by default
		verbose:        false,
	}
}

// SetVerbose sets the verbosity level for the service.
func (s *InstallService) SetVerbose(verbose bool) {
	s.verbose = verbose
	s.appsManager = apps.NewManager(verbose)
}

// InstallApplication detects optimal method and installs via appropriate manager.
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

// InstallMultipleApplications processes batch installations with error aggregation.
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

// GetSystemInfo returns detected distribution and available package managers.
func (s *InstallService) GetSystemInfo(ctx context.Context) (*domain.SystemInfo, error) {
	return s.systemDetector.DetectSystem(ctx)
}

// ListInstalledPackages queries the package service for installed software.
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

// InstallGroup installs a predefined group of applications.
func (s *InstallService) InstallGroup(ctx context.Context, groupName string) (*domain.InstallResult, error) {
	result := &domain.InstallResult{}

	groupApps, exists := apps.Groups[groupName]
	if !exists {
		result.Failed = append(result.Failed, groupName)
		return result, fmt.Errorf("unknown group: %s", groupName)
	}

	for _, appName := range groupApps {
		if err := s.appsManager.InstallApp(ctx, appName); err != nil {
			result.Failed = append(result.Failed, appName)
		} else {
			result.Installed = append(result.Installed, appName)
		}
	}

	return result, nil
}

// InstallPackages installs multiple packages.
func (s *InstallService) InstallPackages(ctx context.Context, packages []string) (*domain.InstallResult, error) {
	result := &domain.InstallResult{}

	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		if err := s.appsManager.InstallApp(ctx, pkg); err != nil {
			result.Failed = append(result.Failed, pkg)
		} else {
			result.Installed = append(result.Installed, pkg)
		}
	}

	return result, nil
}

// GetAvailableGroups returns all available installation groups.
func (s *InstallService) GetAvailableGroups() map[string][]string {
	return apps.Groups
}

// IsAppInstalled checks if an application is installed.
func (s *InstallService) IsAppInstalled(ctx context.Context, appName string) bool {
	// Check if the app can be found via package manager query
	cmd := exec.CommandContext(ctx, "which", appName)
	if err := cmd.Run(); err == nil {
		return true
	}

	// Could also check with package managers
	// This is a simplified check
	return false
}

// GetAppDescription returns the description of an application.
func (s *InstallService) GetAppDescription(appName string) string {
	if app, exists := apps.Apps[appName]; exists {
		return app.Description
	}

	return ""
}

// GetAppVersion returns the version of an installed application.
func (s *InstallService) GetAppVersion(ctx context.Context, appName string) string {
	// Try to get version from the app itself
	cmd := exec.CommandContext(ctx, appName, "--version")

	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	return ""
}
