// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package platform provides shared adapters that work across distributions.
package platform

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/janderssonse/karei/internal/domain"
)

// SystemDetector implements the SystemDetector port for Linux systems.
type SystemDetector struct {
	commandRunner domain.CommandRunner
	fileManager   domain.FileManager
}

// NewSystemDetector creates a new system detector.
func NewSystemDetector(commandRunner domain.CommandRunner, fileManager domain.FileManager) *SystemDetector {
	return &SystemDetector{
		commandRunner: commandRunner,
		fileManager:   fileManager,
	}
}

// DetectSystem returns comprehensive system information.
func (d *SystemDetector) DetectSystem(ctx context.Context) (*domain.SystemInfo, error) {
	distribution, err := d.DetectDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect distribution: %w", err)
	}

	desktopEnv, _ := d.DetectDesktopEnvironment(ctx) // Optional, may fail

	packageManager, err := d.DetectPackageManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package manager: %w", err)
	}

	return &domain.SystemInfo{
		Distribution:       distribution,
		DesktopEnvironment: desktopEnv,
		PackageManager:     packageManager,
		Architecture:       runtime.GOARCH,
		Kernel:             d.getKernelVersion(ctx),
	}, nil
}

// DetectDistribution returns the Linux distribution information.
func (d *SystemDetector) DetectDistribution(_ context.Context) (*domain.Distribution, error) {
	// Try to read /etc/os-release first (standard)
	if d.fileManager.FileExists("/etc/os-release") {
		data, err := d.fileManager.ReadFile("/etc/os-release")
		if err == nil {
			return d.parseOSRelease(string(data)), nil
		}
	}

	// Fallback to other methods
	if d.fileManager.FileExists("/etc/lsb-release") {
		data, err := d.fileManager.ReadFile("/etc/lsb-release")
		if err == nil {
			return d.parseLSBRelease(string(data)), nil
		}
	}

	return &domain.Distribution{
		Name:   "Unknown",
		ID:     "unknown",
		Family: "unknown",
	}, nil
}

// DetectDesktopEnvironment returns the desktop environment information.
func (d *SystemDetector) DetectDesktopEnvironment(_ context.Context) (*domain.DesktopEnvironment, error) {
	// Check environment variables
	if session := os.Getenv("XDG_CURRENT_DESKTOP"); session != "" {
		return &domain.DesktopEnvironment{
			Name:    session,
			Session: os.Getenv("XDG_SESSION_DESKTOP"),
		}, nil
	}

	if session := os.Getenv("DESKTOP_SESSION"); session != "" {
		return &domain.DesktopEnvironment{
			Name:    session,
			Session: session,
		}, nil
	}

	return nil, domain.ErrNoDesktopEnvironment
}

// DetectPackageManager returns the primary package manager for this system.
func (d *SystemDetector) DetectPackageManager(_ context.Context) (*domain.PackageManager, error) {
	// Check for various package managers in order of preference
	packageManagers := []struct {
		name    string
		command string
		method  domain.InstallMethod
	}{
		{"APT", "apt", domain.MethodAPT},
		{"DNF", "dnf", domain.MethodDNF},
		{"YUM", "yum", domain.MethodYum},
		{"Pacman", "pacman", domain.MethodPacman},
		{"Zypper", "zypper", domain.MethodZypper},
	}

	for _, pm := range packageManagers {
		if d.commandRunner.CommandExists(pm.command) {
			return &domain.PackageManager{
				Name:    pm.name,
				Method:  pm.method,
				Command: pm.command,
			}, nil
		}
	}

	return nil, domain.ErrNoPackageManager
}

// Helper methods

func (d *SystemDetector) parseOSRelease(content string) *domain.Distribution {
	fields := make(map[string]string)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
				fields[key] = value
			}
		}
	}

	family := d.determineFamily(fields["ID"])

	return &domain.Distribution{
		Name:     fields["NAME"],
		ID:       fields["ID"],
		Version:  fields["VERSION"],
		Codename: fields["VERSION_CODENAME"],
		Family:   family,
	}
}

func (d *SystemDetector) parseLSBRelease(content string) *domain.Distribution {
	fields := make(map[string]string)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				fields[key] = value
			}
		}
	}

	id := strings.ToLower(fields["DISTRIB_ID"])
	family := d.determineFamily(id)

	return &domain.Distribution{
		Name:     fields["DISTRIB_DESCRIPTION"],
		ID:       id,
		Version:  fields["DISTRIB_RELEASE"],
		Codename: fields["DISTRIB_CODENAME"],
		Family:   family,
	}
}

func (d *SystemDetector) determineFamily(distributionID string) string {
	distributionID = strings.ToLower(distributionID)

	// Use maps to reduce cyclomatic complexity
	familyMap := map[string]string{
		"ubuntu":   "debian",
		"debian":   "debian",
		"mint":     "debian",
		"fedora":   "rhel",
		"rhel":     "rhel",
		"centos":   "rhel",
		"rocky":    "rhel",
		"arch":     "arch",
		"manjaro":  "arch",
		"opensuse": "suse",
		"suse":     "suse",
	}

	for distro, family := range familyMap {
		if strings.Contains(distributionID, distro) {
			return family
		}
	}

	return "unknown"
}

func (d *SystemDetector) getKernelVersion(ctx context.Context) string {
	output, err := d.commandRunner.ExecuteWithOutput(ctx, "uname", "-r")
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(output)
}
