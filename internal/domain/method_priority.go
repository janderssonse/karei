// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain

// GetMethodPriority returns the priority for a given install method.
// Lower values indicate higher priority.
func GetMethodPriority(method InstallMethod, systemInfo *SystemInfo) int {
	// Special case for immutable distros
	if systemInfo != nil && systemInfo.Distribution != nil &&
		systemInfo.Distribution.ID == "fedora-silverblue" &&
		method == MethodFlatpak {
		return 0 // Highest for immutable
	}

	priorities := map[InstallMethod]int{
		// Native package managers - highest priority
		MethodAPT:    1,
		MethodDNF:    1,
		MethodPacman: 1,
		MethodZypper: 1,
		MethodYum:    2, // Older, lower priority than DNF
		// Containerized - medium priority
		MethodFlatpak: 3,
		MethodSnap:    4,
		// Direct installation - lower priority
		MethodDEB: 5,
		MethodRPM: 5,
		// Script/Binary - lowest priority
		MethodScript: 6,
		MethodBinary: 6,
	}

	if priority, exists := priorities[method]; exists {
		return priority
	}

	return 999
}

// IsMethodCompatible checks if an install method is compatible with the system.
func IsMethodCompatible(method InstallMethod, info *SystemInfo) bool {
	if info.Distribution == nil {
		return false
	}

	// Universal methods
	universalMethods := map[InstallMethod]bool{
		MethodSnap:    true,
		MethodFlatpak: true,
		MethodBinary:  true,
		MethodScript:  true,
	}
	if universalMethods[method] {
		return true
	}

	// Distribution-specific methods
	distro := info.Distribution
	compatibilityRules := map[InstallMethod]bool{
		MethodAPT:    distro.Family == "debian",
		MethodDNF:    distro.ID == "fedora" || (distro.Family == "rhel" && distro.Version >= "8"),
		MethodYum:    distro.Family == "rhel",
		MethodPacman: distro.Family == "arch",
		MethodZypper: distro.Family == "suse",
	}

	return compatibilityRules[method]
}
