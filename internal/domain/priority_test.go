// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestPackageMethodPriority tests the business rules for package manager selection priority
// Business Rule: When multiple methods are available, prefer native > containerized > script.
func TestPackageMethodPriority(t *testing.T) {
	tests := []struct {
		name             string
		availableMethods []domain.InstallMethod
		systemInfo       *domain.SystemInfo
		expectedChoice   domain.InstallMethod
		reason           string
	}{
		{
			name: "native_preferred_over_containerized",
			availableMethods: []domain.InstallMethod{
				domain.MethodAPT,
				domain.MethodSnap,
				domain.MethodFlatpak,
			},
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "ubuntu",
					Family: "debian",
				},
			},
			expectedChoice: domain.MethodAPT,
			reason:         "Native package manager (APT) should be preferred over Snap/Flatpak",
		},
		{
			name: "dnf_preferred_over_yum_on_modern_fedora",
			availableMethods: []domain.InstallMethod{
				domain.MethodDNF,
				domain.MethodYum,
			},
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:      "fedora",
					Version: "39",
					Family:  "rhel",
				},
			},
			expectedChoice: domain.MethodDNF,
			reason:         "DNF should be preferred over Yum on modern Fedora",
		},
		{
			name: "flatpak_preferred_on_immutable_distros",
			availableMethods: []domain.InstallMethod{
				domain.MethodFlatpak,
				domain.MethodRPM,
			},
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "fedora-silverblue",
					Family: "rhel",
				},
			},
			expectedChoice: domain.MethodFlatpak,
			reason:         "Immutable distros should prefer containerized packages",
		},
		{
			name: "binary_as_last_resort",
			availableMethods: []domain.InstallMethod{
				domain.MethodBinary,
			},
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "alpine",
					Family: "alpine",
				},
			},
			expectedChoice: domain.MethodBinary,
			reason:         "Binary installation when no package manager available",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Find best method based on priority
			var bestMethod domain.InstallMethod

			bestPriority := 999

			for _, method := range tc.availableMethods {
				priority := domain.GetMethodPriority(method, tc.systemInfo)
				if priority < bestPriority {
					bestPriority = priority
					bestMethod = method
				}
			}

			assert.Equal(t, tc.expectedChoice, bestMethod, tc.reason)
		})
	}
}

// isMethodCompatible checks if an install method is compatible with the system.

// TestPackageMethodCompatibility tests method compatibility with system.
func TestPackageMethodCompatibility(t *testing.T) {
	tests := []struct {
		name       string
		method     domain.InstallMethod
		systemInfo *domain.SystemInfo
		compatible bool
		reason     string
	}{
		{
			name:   "apt_compatible_with_debian",
			method: domain.MethodAPT,
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					Family: "debian",
				},
			},
			compatible: true,
			reason:     "APT is compatible with Debian family",
		},
		{
			name:   "dnf_compatible_with_fedora",
			method: domain.MethodDNF,
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID: "fedora",
				},
			},
			compatible: true,
			reason:     "DNF is compatible with Fedora",
		},
		{
			name:   "pacman_not_compatible_with_ubuntu",
			method: domain.MethodPacman,
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "ubuntu",
					Family: "debian",
				},
			},
			compatible: false,
			reason:     "Pacman is not compatible with Ubuntu",
		},
		{
			name:   "snap_universal_compatibility",
			method: domain.MethodSnap,
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID: "any-distro",
				},
			},
			compatible: true,
			reason:     "Snap should work on any modern Linux with snapd",
		},
		{
			name:   "flatpak_universal_compatibility",
			method: domain.MethodFlatpak,
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID: "any-distro",
				},
			},
			compatible: true,
			reason:     "Flatpak should work on any modern Linux",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := domain.IsMethodCompatible(tc.method, tc.systemInfo)
			assert.Equal(t, tc.compatible, result, tc.reason)
		})
	}
}
