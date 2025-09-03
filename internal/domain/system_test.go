// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
)

// distributionTestCase represents a test case for distribution detection.
type distributionTestCase struct {
	name       string
	systemInfo *domain.SystemInfo
	expected   bool
	reason     string
}

// runDistributionTests runs a set of distribution detection tests.
func runDistributionTests(t *testing.T, tests []distributionTestCase, checkFunc func(*domain.SystemInfo) bool) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := checkFunc(tc.systemInfo)
			assert.Equal(t, tc.expected, result, tc.reason)
		})
	}
}

// TestSystemInfoIsLinux tests Linux distribution detection logic
// Business rule: IsLinux() returns true for Ubuntu and Debian-family distros.
func TestSystemInfoIsLinux(t *testing.T) {
	tests := []distributionTestCase{
		{
			name: "ubuntu_identified_as_linux",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "ubuntu",
					Family: "debian",
				},
			},
			expected: true,
			reason:   "Ubuntu (ID=ubuntu) should be identified as Linux",
		},
		{
			name: "debian_family_identified_as_linux",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "linuxmint",
					Family: "debian",
				},
			},
			expected: true,
			reason:   "Any debian family distro should be identified as Linux",
		},
		{
			name: "non_debian_not_identified_as_linux",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "fedora",
					Family: "rhel",
				},
			},
			expected: false,
			reason:   "Non-Debian distros return false (method name is misleading)",
		},
		{
			name:       "nil_distribution_returns_false",
			systemInfo: &domain.SystemInfo{},
			expected:   false,
			reason:     "Nil distribution should safely return false",
		},
	}

	runDistributionTests(t, tests, func(info *domain.SystemInfo) bool {
		return info.IsLinux()
	})
}

// TestSystemInfoIsFedora tests Fedora/RHEL family detection
// Business rule: IsFedora() returns true for ID=fedora OR family=rhel.
func TestSystemInfoIsFedora(t *testing.T) {
	tests := []distributionTestCase{
		{
			name: "fedora_id_matches",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "fedora",
					Family: "rhel",
				},
			},
			expected: true,
			reason:   "ID=fedora should match",
		},
		{
			name: "rhel_family_matches",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "centos", // Any ID
					Family: "rhel",
				},
			},
			expected: true,
			reason:   "Family=rhel should match regardless of ID",
		},
		{
			name: "non_rhel_family",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "ubuntu",
					Family: "debian",
				},
			},
			expected: false,
			reason:   "Non-RHEL family should not match",
		},
		{
			name:       "nil_distribution",
			systemInfo: &domain.SystemInfo{},
			expected:   false,
			reason:     "Nil distribution should safely return false",
		},
	}

	runDistributionTests(t, tests, func(info *domain.SystemInfo) bool {
		return info.IsFedora()
	})
}

// TestSystemInfoIsArch tests Arch Linux family detection
// Business rule: IsArch() returns true for ID=arch OR family=arch.
func TestSystemInfoIsArch(t *testing.T) {
	tests := []distributionTestCase{
		{
			name: "arch_id_matches",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "arch",
					Family: "arch",
				},
			},
			expected: true,
			reason:   "ID=arch should match",
		},
		{
			name: "arch_family_matches",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "manjaro", // Derivative
					Family: "arch",
				},
			},
			expected: true,
			reason:   "Family=arch should match for derivatives",
		},
		{
			name: "non_arch_family",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "ubuntu",
					Family: "debian",
				},
			},
			expected: false,
			reason:   "Non-Arch family should not match",
		},
		{
			name:       "nil_distribution",
			systemInfo: &domain.SystemInfo{},
			expected:   false,
			reason:     "Nil distribution should safely return false",
		},
	}

	runDistributionTests(t, tests, func(info *domain.SystemInfo) bool {
		return info.IsArch()
	})
}

// TestSystemInfoIsGNOME tests GNOME desktop environment detection.
func TestSystemInfoIsGNOME(t *testing.T) {
	tests := []struct {
		name       string
		systemInfo *domain.SystemInfo
		expected   bool
		reason     string
	}{
		{
			name: "gnome_desktop",
			systemInfo: &domain.SystemInfo{
				DesktopEnvironment: &domain.DesktopEnvironment{
					Name:    "GNOME",
					Session: "gnome",
					Version: "45.0",
				},
			},
			expected: true,
			reason:   "GNOME desktop should be identified",
		},
		{
			name: "kde_not_gnome",
			systemInfo: &domain.SystemInfo{
				DesktopEnvironment: &domain.DesktopEnvironment{
					Name:    "KDE",
					Session: "plasma",
					Version: "5.27",
				},
			},
			expected: false,
			reason:   "KDE is not GNOME",
		},
		{
			name: "xfce_not_gnome",
			systemInfo: &domain.SystemInfo{
				DesktopEnvironment: &domain.DesktopEnvironment{
					Name:    "XFCE",
					Session: "xfce",
					Version: "4.18",
				},
			},
			expected: false,
			reason:   "XFCE is not GNOME",
		},
		{
			name:       "nil_desktop_environment",
			systemInfo: &domain.SystemInfo{},
			expected:   false,
			reason:     "Nil desktop environment should return false",
		},
		{
			name: "server_no_desktop",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "ubuntu-server",
					Family: "debian",
				},
				// No DesktopEnvironment set
			},
			expected: false,
			reason:   "Server without desktop should return false",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.systemInfo.IsGNOME()
			assert.Equal(t, tc.expected, result, tc.reason)
		})
	}
}

// TestSystemInfoMethodCombinations tests complex scenarios with multiple conditions.
func TestSystemInfoMethodCombinations(t *testing.T) {
	tests := []struct {
		name        string
		systemInfo  *domain.SystemInfo
		isLinux     bool
		isFedora    bool
		isArch      bool
		isGNOME     bool
		description string
	}{
		{
			name: "ubuntu_gnome_desktop",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "ubuntu",
					Family: "debian",
				},
				DesktopEnvironment: &domain.DesktopEnvironment{
					Name: "GNOME",
				},
			},
			isLinux:     true,
			isFedora:    false,
			isArch:      false,
			isGNOME:     true,
			description: "Ubuntu with GNOME should be Linux and GNOME",
		},
		{
			name: "fedora_kde_desktop",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "fedora",
					Family: "rhel",
				},
				DesktopEnvironment: &domain.DesktopEnvironment{
					Name: "KDE",
				},
			},
			isLinux:     false, // Not Debian-based
			isFedora:    true,
			isArch:      false,
			isGNOME:     false,
			description: "Fedora with KDE should be Fedora but not GNOME or Debian-Linux",
		},
		{
			name: "arch_gnome_desktop",
			systemInfo: &domain.SystemInfo{
				Distribution: &domain.Distribution{
					ID:     "arch",
					Family: "arch",
				},
				DesktopEnvironment: &domain.DesktopEnvironment{
					Name: "GNOME",
				},
			},
			isLinux:     false, // Not Debian-based
			isFedora:    false,
			isArch:      true,
			isGNOME:     true,
			description: "Arch with GNOME should be Arch and GNOME",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.isLinux, tc.systemInfo.IsLinux(),
				"IsLinux check failed: %s", tc.description)
			assert.Equal(t, tc.isFedora, tc.systemInfo.IsFedora(),
				"IsFedora check failed: %s", tc.description)
			assert.Equal(t, tc.isArch, tc.systemInfo.IsArch(),
				"IsArch check failed: %s", tc.description)
			assert.Equal(t, tc.isGNOME, tc.systemInfo.IsGNOME(),
				"IsGNOME check failed: %s", tc.description)
		})
	}
}
