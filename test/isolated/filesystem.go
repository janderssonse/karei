// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package isolated provides isolated filesystem testing utilities.
package isolated

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/janderssonse/karei/test/mocks"
	"github.com/janderssonse/karei/test/offline"
)

var (
	// ErrPackageDatabaseNotLoaded indicates the package database was not loaded.
	ErrPackageDatabaseNotLoaded = errors.New("package database not loaded")
	// ErrPackageNotFound indicates a package was not found.
	ErrPackageNotFound = errors.New("package not found")
	// ErrUnsupportedInstallMethod indicates the installation method is not supported.
	ErrUnsupportedInstallMethod = errors.New("unsupported installation method")
	// ErrScriptSimulationRequiresDB indicates package database is required for script simulation.
	ErrScriptSimulationRequiresDB = errors.New("package database required for script simulation")
	// ErrCustomScriptNotFound indicates a custom script was not found.
	ErrCustomScriptNotFound = errors.New("custom script not found")
	// ErrPackageNotInstalled indicates a package is not installed.
	ErrPackageNotInstalled = errors.New("package does not appear to be installed")
	// ErrBinaryNotExecutable indicates a binary is not executable.
	ErrBinaryNotExecutable = errors.New("binary is not executable")
)

// Filesystem provides a completely isolated filesystem for testing.
type Filesystem struct {
	rootDir        string
	binDir         string
	configDir      string
	dataDir        string
	tempDir        string
	desktopDir     string
	binaryGen      *mocks.FakeBinaryGenerator
	packageDB      *offline.PackageDB
	verbose        bool
	useFakeRoot    bool
	overlaySupport bool
}

// Config configures the isolated filesystem.
type Config struct {
	RootDir        string
	Verbose        bool
	UseFakeRoot    bool
	CreateBinaries bool
	LoadPackageDB  bool
	FixtureDir     string
}

// NewFilesystem creates a new isolated filesystem.
func NewFilesystem(config Config) (*Filesystem, error) {
	filesystem := &Filesystem{
		rootDir:        config.RootDir,
		verbose:        config.Verbose,
		useFakeRoot:    config.UseFakeRoot,
		overlaySupport: supportsOverlayFS(),
	}

	// Set up directory structure
	filesystem.binDir = filepath.Join(filesystem.rootDir, "usr", "local", "bin")
	filesystem.configDir = filepath.Join(filesystem.rootDir, "etc")
	filesystem.dataDir = filepath.Join(filesystem.rootDir, "usr", "share")
	filesystem.tempDir = filepath.Join(filesystem.rootDir, "tmp")
	filesystem.desktopDir = filepath.Join(filesystem.rootDir, "usr", "share", "applications")

	// Create directory structure
	if err := filesystem.createDirectoryStructure(); err != nil {
		return nil, fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Initialize fake binary generator
	if config.CreateBinaries {
		filesystem.binaryGen = mocks.NewFakeBinaryGenerator(filesystem.binDir, config.Verbose)
		if err := filesystem.binaryGen.CreateCommonBinaries(); err != nil {
			return nil, fmt.Errorf("failed to create binaries: %w", err)
		}

		if err := filesystem.binaryGen.CreateApplicationBinaries(); err != nil {
			return nil, fmt.Errorf("failed to create app binaries: %w", err)
		}

		if err := filesystem.binaryGen.CreateDesktopEntries(filesystem.desktopDir); err != nil {
			return nil, fmt.Errorf("failed to create desktop entries: %w", err)
		}
	}

	// Initialize package database
	if config.LoadPackageDB && config.FixtureDir != "" {
		filesystem.packageDB = offline.NewPackageDB(config.Verbose)
		if err := filesystem.packageDB.LoadFromFixtures(config.FixtureDir); err != nil {
			return nil, fmt.Errorf("failed to load package database: %w", err)
		}
	}

	if filesystem.verbose {
		fmt.Printf("Created isolated filesystem at: %s\n", filesystem.rootDir)
	}

	return filesystem, nil
}

// SimulatePackageInstallation simulates installing a package.
func (fs *Filesystem) SimulatePackageInstallation(packageName string) error {
	if fs.packageDB == nil {
		return ErrPackageDatabaseNotLoaded
	}

	pkg, exists := fs.packageDB.GetPackage(packageName)
	if !exists {
		return fmt.Errorf("%w: %s", ErrPackageNotFound, packageName)
	}

	if fs.verbose {
		fmt.Printf("Simulating installation of %s (method: %s)\n", packageName, pkg.Method)
	}

	// Create fake installation files
	switch pkg.Method {
	case "apt":
		return fs.simulateAPTInstallation(pkg)
	case "flatpak":
		return fs.simulateFlatpakInstallation(pkg)
	case "github":
		return fs.simulateGitHubInstallation(pkg)
	case "deb":
		return fs.simulateDEBInstallation(pkg)
	case "script":
		return fs.simulateScriptInstallation(pkg)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedInstallMethod, pkg.Method)
	}
}

// IsPackageInstalled checks if a package appears to be installed.
func (fs *Filesystem) IsPackageInstalled(packageName string) bool {
	// Check for binary
	binaryPath := filepath.Join(fs.binDir, packageName)
	if _, err := os.Stat(binaryPath); err == nil {
		return true
	}

	// Check for APT status
	statusPath := filepath.Join(fs.configDir, "dpkg", "status")
	if data, err := os.ReadFile(statusPath); err == nil { //nolint:gosec
		return strings.Contains(string(data), "Package: "+packageName)
	}

	// Check for Flatpak
	flatpakPath := filepath.Join(fs.dataDir, "flatpak", "app", packageName)
	if _, err := os.Stat(flatpakPath); err == nil {
		return true
	}

	return false
}

// GetInstalledPackages returns list of apparently installed packages.
func (fs *Filesystem) GetInstalledPackages() ([]string, error) {
	var packages []string

	// Check binaries
	if binaries, err := fs.binaryGen.ListCreatedBinaries(); err == nil {
		packages = append(packages, binaries...)
	}

	// Check APT status file
	statusPath := filepath.Join(fs.configDir, "dpkg", "status")
	if data, err := os.ReadFile(statusPath); err == nil { //nolint:gosec
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Package: ") {
				pkg := strings.TrimPrefix(line, "Package: ")
				packages = append(packages, pkg)
			}
		}
	}

	return packages, nil
}

// ValidateInstallation validates that an installation looks correct.
func (fs *Filesystem) ValidateInstallation(packageName string) error {
	if !fs.IsPackageInstalled(packageName) {
		return fmt.Errorf("%w: %s", ErrPackageNotInstalled, packageName)
	}

	// Check binary exists and is executable
	binaryPath := filepath.Join(fs.binDir, packageName)
	if info, err := os.Stat(binaryPath); err == nil {
		if info.Mode()&0111 == 0 {
			return fmt.Errorf("%w: %s", ErrBinaryNotExecutable, binaryPath)
		}
	}

	return nil
}

// GetFilesystemStats returns statistics about the isolated filesystem.
func (fs *Filesystem) GetFilesystemStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["root_dir"] = fs.rootDir
	stats["overlay_support"] = fs.overlaySupport
	stats["fake_root"] = fs.useFakeRoot

	if installed, err := fs.GetInstalledPackages(); err == nil {
		stats["installed_packages"] = len(installed)
		stats["package_list"] = installed
	}

	if fs.binaryGen != nil {
		if binaries, err := fs.binaryGen.ListCreatedBinaries(); err == nil {
			stats["fake_binaries"] = len(binaries)
		}
	}

	return stats
}

// Cleanup removes the isolated filesystem.
func (fs *Filesystem) Cleanup() error {
	if fs.binaryGen != nil {
		if err := fs.binaryGen.CleanupBinaries(); err != nil {
			return fmt.Errorf("failed to cleanup binaries: %w", err)
		}
	}

	if err := os.RemoveAll(fs.rootDir); err != nil {
		return fmt.Errorf("failed to remove isolated filesystem: %w", err)
	}

	if fs.verbose {
		fmt.Printf("Cleaned up isolated filesystem: %s\n", fs.rootDir)
	}

	return nil
}

// supportsOverlayFS checks if the system supports OverlayFS.
func supportsOverlayFS() bool {
	return runtime.GOOS == "linux"
}

// GetBinaryPath returns path to a binary in the isolated filesystem.
func (fs *Filesystem) GetBinaryPath(binaryName string) string {
	return filepath.Join(fs.binDir, binaryName)
}

// GetConfigPath returns path to config directory.
func (fs *Filesystem) GetConfigPath() string {
	return fs.configDir
}

// GetDataPath returns path to data directory.
func (fs *Filesystem) GetDataPath() string {
	return fs.dataDir
}

// GetTempPath returns path to temp directory.
func (fs *Filesystem) GetTempPath() string {
	return fs.tempDir
}

// createDirectoryStructure creates the basic filesystem structure.
func (fs *Filesystem) createDirectoryStructure() error {
	dirs := []string{
		"usr/local/bin",
		"usr/bin",
		"usr/share/applications",
		"usr/share/man",
		"etc",
		"home/testuser/.config",
		"home/testuser/.local/share",
		"home/testuser/.local/bin",
		"tmp",
		"var/log",
		"var/cache",
		"opt",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(fs.rootDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil { //nolint:gosec
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}

	if fs.verbose {
		fmt.Printf("Created %d directories in isolated filesystem\n", len(dirs))
	}

	return nil
}

// simulateAPTInstallation simulates APT package installation.
func (fs *Filesystem) simulateAPTInstallation(pkg offline.PackageMetadata) error {
	// Create binary
	binaryPath := filepath.Join(fs.binDir, pkg.Name)
	binaryContent := fmt.Sprintf(`#!/bin/bash
# Fake %s binary (APT installed)
echo "Fake %s version %s (APT)"
case "$1" in
    --version) echo "%s %s" ;;
    --help) echo "%s - %s" ;;
esac
`, pkg.Name, pkg.Name, pkg.Version, pkg.Name, pkg.Version, pkg.Name, pkg.Description)

	if err := os.WriteFile(binaryPath, []byte(binaryContent), 0755); err != nil { //nolint:gosec
		return err
	}

	// Create package status file
	statusPath := filepath.Join(fs.configDir, "dpkg", "status")
	_ = os.MkdirAll(filepath.Dir(statusPath), 0755) //nolint:gosec

	statusEntry := fmt.Sprintf(`Package: %s
Status: install ok installed
Priority: %s
Section: %s
Maintainer: %s
Architecture: %s
Version: %s
Description: %s
 %s

`, pkg.Name, pkg.Priority, pkg.Section, pkg.Maintainer, pkg.Architecture, pkg.Version, pkg.Description, pkg.Description)

	statusFile, err := os.OpenFile(statusPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //nolint:gosec
	if err != nil {
		return err
	}

	defer func() { _ = statusFile.Close() }()

	_, err = statusFile.WriteString(statusEntry)

	return err
}

// simulateFlatpakInstallation simulates Flatpak installation.
func (fs *Filesystem) simulateFlatpakInstallation(pkg offline.PackageMetadata) error {
	flatpakDir := filepath.Join(fs.dataDir, "flatpak", "app", pkg.Source)
	if err := os.MkdirAll(flatpakDir, 0755); err != nil { //nolint:gosec
		return err
	}

	// Create app metadata
	metadataPath := filepath.Join(flatpakDir, "current", "active", "metadata")
	_ = os.MkdirAll(filepath.Dir(metadataPath), 0755) //nolint:gosec

	metadata := fmt.Sprintf(`[Application]
name=%s
runtime=org.freedesktop.Platform/x86_64/22.08
sdk=org.freedesktop.Sdk/x86_64/22.08

[Context]
shared=ipc;
sockets=x11;wayland;
`, pkg.Source)

	if err := os.WriteFile(metadataPath, []byte(metadata), 0644); err != nil { //nolint:gosec
		return err
	}

	// Create desktop entry
	desktopName := strings.ReplaceAll(pkg.Source, ".", "_") + ".desktop"
	desktopPath := filepath.Join(fs.desktopDir, desktopName)

	desktopContent := fmt.Sprintf(`[Desktop Entry]
Version=1.0
Type=Application
Name=%s
Comment=%s
Exec=flatpak run %s
Icon=%s
Terminal=false
Categories=Application;
`, pkg.Name, pkg.Description, pkg.Source, pkg.Source)

	return os.WriteFile(desktopPath, []byte(desktopContent), 0644) //nolint:gosec
}

// simulateGitHubInstallation simulates GitHub release installation.
func (fs *Filesystem) simulateGitHubInstallation(pkg offline.PackageMetadata) error {
	// Create binary
	binaryPath := filepath.Join(fs.binDir, pkg.Name)
	binaryContent := fmt.Sprintf(`#!/bin/bash
# Fake %s binary (GitHub release)
echo "Fake %s version %s (GitHub)"
case "$1" in
    --version) echo "%s %s" ;;
    --help) echo "%s - GitHub release binary" ;;
esac
`, pkg.Name, pkg.Name, pkg.Version, pkg.Name, pkg.Version, pkg.Name)

	if err := os.WriteFile(binaryPath, []byte(binaryContent), 0755); err != nil { //nolint:gosec
		return err
	}

	// Create installation record
	recordPath := filepath.Join(fs.dataDir, "karei", "installed", pkg.Name+".json")
	_ = os.MkdirAll(filepath.Dir(recordPath), 0755) //nolint:gosec

	record := fmt.Sprintf(`{
  "name": "%s",
  "version": "%s",
  "method": "github",
  "source": "%s",
  "installed_at": "%s"
}
`, pkg.Name, pkg.Version, pkg.Source, fs.getCurrentTimestamp())

	return os.WriteFile(recordPath, []byte(record), 0644) //nolint:gosec
}

// simulateDEBInstallation simulates DEB package installation.
func (fs *Filesystem) simulateDEBInstallation(pkg offline.PackageMetadata) error {
	// Similar to APT but with DEB-specific metadata
	return fs.simulateAPTInstallation(pkg)
}

// simulateScriptInstallation simulates custom script installation.
func (fs *Filesystem) simulateScriptInstallation(pkg offline.PackageMetadata) error {
	if fs.packageDB == nil {
		return ErrScriptSimulationRequiresDB
	}

	script, exists := fs.packageDB.GetCustomScript(pkg.Source)
	if !exists {
		return fmt.Errorf("%w: %s", ErrCustomScriptNotFound, pkg.Source)
	}

	// Create binary
	binaryPath := filepath.Join(fs.binDir, script.Name)
	binaryContent := fmt.Sprintf(`#!/bin/bash
# Fake %s binary (custom script)
echo "Fake %s (custom installation)"
case "$1" in
    --version) echo "%s custom" ;;
    --help) echo "%s - %s" ;;
esac
`, script.Name, script.Name, script.Name, script.Name, script.Description)

	if err := os.WriteFile(binaryPath, []byte(binaryContent), 0755); err != nil { //nolint:gosec
		return err
	}

	// Execute post-install steps
	for _, postCmd := range script.PostInstall {
		if err := fs.simulateCommand(postCmd); err != nil {
			return fmt.Errorf("post-install failed: %w", err)
		}
	}

	return nil
}

// simulateCommand simulates executing a command in the isolated environment.
func (fs *Filesystem) simulateCommand(command string) error {
	if strings.Contains(command, "mkdir -p") {
		// Extract directory path and create it
		parts := strings.Fields(command)
		for i, part := range parts {
			if part == "-p" && i+1 < len(parts) {
				dirPath := parts[i+1]
				// Convert to isolated path
				if strings.HasPrefix(dirPath, "~/") {
					dirPath = filepath.Join(fs.rootDir, "home/testuser", dirPath[2:])
				} else if strings.HasPrefix(dirPath, "/") {
					dirPath = filepath.Join(fs.rootDir, dirPath[1:])
				}

				return os.MkdirAll(dirPath, 0755) //nolint:gosec
			}
		}
	}

	if fs.verbose {
		fmt.Printf("Simulating command: %s\n", command)
	}

	return nil
}

// getCurrentTimestamp returns current timestamp in ISO format.
func (fs *Filesystem) getCurrentTimestamp() string {
	return "2024-03-15T10:30:00Z"
}
