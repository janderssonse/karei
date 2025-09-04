// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain

import (
	"context"
	"errors"
	"strings"
)

var (
	// ErrInvalidPackage indicates the package is malformed or invalid.
	ErrInvalidPackage = errors.New("invalid package")
	// ErrPackageNotFound indicates the package was not found.
	ErrPackageNotFound = errors.New("package not found")
	// ErrUnsupportedInstallMethod indicates the installation method is not supported.
	ErrUnsupportedInstallMethod = errors.New("unsupported installation method")
	// ErrUnsupportedRemoveMethod indicates the removal method is not supported.
	ErrUnsupportedRemoveMethod = errors.New("unsupported removal method")
	// ErrInsufficientSpace indicates there is not enough disk space for installation.
	ErrInsufficientSpace = errors.New("insufficient disk space")
)

// InstallMethod represents different installation methods.
type InstallMethod string

// Installation methods supported across all distributions.
const (
	MethodAPT          InstallMethod = "apt"
	MethodDNF          InstallMethod = "dnf"    // Fedora
	MethodYum          InstallMethod = "yum"    // RHEL/CentOS
	MethodPacman       InstallMethod = "pacman" // Arch
	MethodZypper       InstallMethod = "zypper" // openSUSE
	MethodSnap         InstallMethod = "snap"
	MethodFlatpak      InstallMethod = "flatpak"
	MethodGitHub       InstallMethod = "github"
	MethodGitHubBinary InstallMethod = "github-binary"
	MethodGitHubBundle InstallMethod = "github-bundle"
	MethodGitHubJava   InstallMethod = "github-java"
	MethodDEB          InstallMethod = "deb"
	MethodRPM          InstallMethod = "rpm"
	MethodScript       InstallMethod = "script"
	MethodBinary       InstallMethod = "binary"
	MethodAqua         InstallMethod = "aqua"
	MethodMise         InstallMethod = "mise"
)

// Package represents a software package to be installed.
type Package struct {
	Name         string        `json:"name"`
	Group        string        `json:"group"`
	Description  string        `json:"description"`
	Method       InstallMethod `json:"method"`
	Source       string        `json:"source"`
	Version      string        `json:"version,omitempty"`
	Dependencies []string      `json:"dependencies,omitempty"`
}

// IsValid validates the package has required fields.
func (p *Package) IsValid() bool {
	// Trim whitespace to check for actual content
	name := strings.TrimSpace(p.Name)
	method := strings.TrimSpace(string(p.Method))
	source := strings.TrimSpace(p.Source)

	return name != "" && method != "" && source != ""
}

// InstallationResult represents the result of a package installation.
type InstallationResult struct {
	Package  *Package `json:"package"`
	Success  bool     `json:"success"`
	Error    error    `json:"error,omitempty"`
	Duration int64    `json:"duration_ms"`
	Output   string   `json:"output,omitempty"`
}

// PackageService provides core package management operations.
type PackageService struct {
	installer PackageInstaller
	detector  SystemDetector
}

// NewPackageService creates a service with installer and detector ports.
func NewPackageService(installer PackageInstaller, detector SystemDetector) *PackageService {
	return &PackageService{
		installer: installer,
		detector:  detector,
	}
}

// Install installs a package using the appropriate method for the current system.
func (s *PackageService) Install(ctx context.Context, pkg *Package) (*InstallationResult, error) {
	if !pkg.IsValid() {
		return nil, ErrInvalidPackage
	}

	// Let the installer handle the installation
	return s.installer.Install(ctx, pkg)
}

// Remove removes a package from the system.
func (s *PackageService) Remove(ctx context.Context, pkg *Package) (*InstallationResult, error) {
	if !pkg.IsValid() {
		return nil, ErrInvalidPackage
	}

	return s.installer.Remove(ctx, pkg)
}

// List returns all installed packages.
func (s *PackageService) List(ctx context.Context) ([]*Package, error) {
	return s.installer.List(ctx)
}
