// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain

const (
	distroArch = "arch"
	distroRHEL = "rhel"
)

// Distribution represents a Linux distribution.
type Distribution struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Version  string `json:"version"`
	Codename string `json:"codename"`
	Family   string `json:"family"` // debian, rhel, arch, suse
}

// DesktopEnvironment represents the desktop environment.
type DesktopEnvironment struct {
	Name    string `json:"name"`
	Session string `json:"session"`
	Version string `json:"version"`
}

// PackageManager represents the package manager type.
type PackageManager struct {
	Name    string        `json:"name"`
	Method  InstallMethod `json:"method"`
	Command string        `json:"command"`
}

// SystemInfo contains system information.
type SystemInfo struct {
	Distribution       *Distribution       `json:"distribution"`
	DesktopEnvironment *DesktopEnvironment `json:"desktop_environment"`
	PackageManager     *PackageManager     `json:"package_manager"`
	Architecture       string              `json:"architecture"`
	Kernel             string              `json:"kernel"`
}

// IsDebianBased checks if the system is Debian/Ubuntu-based.
func (s *SystemInfo) IsDebianBased() bool {
	return s.Distribution != nil &&
		(s.Distribution.ID == "ubuntu" || s.Distribution.Family == "debian")
}

// IsLinux checks if the system is Linux-based.
// Deprecated: Use IsDebianBased() for Debian/Ubuntu systems.
func (s *SystemInfo) IsLinux() bool {
	return s.IsDebianBased()
}

// IsFedora checks if the system is Fedora-based.
func (s *SystemInfo) IsFedora() bool {
	return s.Distribution != nil &&
		(s.Distribution.ID == "fedora" || s.Distribution.Family == distroRHEL)
}

// IsArch checks if the system is Arch-based.
func (s *SystemInfo) IsArch() bool {
	return s.Distribution != nil &&
		(s.Distribution.ID == distroArch || s.Distribution.Family == distroArch)
}

// IsGNOME checks if the desktop environment is GNOME.
func (s *SystemInfo) IsGNOME() bool {
	return s.DesktopEnvironment != nil && s.DesktopEnvironment.Name == "GNOME"
}
