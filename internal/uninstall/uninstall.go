// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package uninstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/system"
)

var (
	// ErrUnknownApp is returned when an unknown app is requested for uninstallation.
	ErrUnknownApp = errors.New("unknown app")
	// ErrUnsupportedUninstallMethod indicates the uninstall method is not supported.
	ErrUnsupportedUninstallMethod = errors.New("unsupported uninstall method")
	// ErrUnknownGroup indicates the group is not recognized.
	ErrUnknownGroup = errors.New("unknown group")
)

// Uninstaller handles application uninstallation.
type Uninstaller struct {
	verbose  bool
	password string // For non-interactive sudo operations
	executor CommandExecutor
}

// NewUninstaller initializes an uninstaller with the specified verbosity level.
func NewUninstaller(verbose bool) *Uninstaller {
	return &Uninstaller{
		verbose:  verbose,
		executor: &RealCommandExecutor{},
	}
}

// SetPassword stores password for non-interactive sudo operations.
func (u *Uninstaller) SetPassword(password string) {
	u.password = password
}

// UninstallApp uninstalls an application by name.
//
//nolint:cyclop // Complexity from legitimate business logic (multiple uninstall methods)
func (u *Uninstaller) UninstallApp(ctx context.Context, name string) error {
	app, exists := apps.Apps[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownApp, name)
	}

	if u.verbose {
		fmt.Printf("Uninstalling %s...\n", app.Name)
	}

	// Check for special uninstall logic first
	if uninstallFunc, hasSpecial := SpecialUninstalls[name]; hasSpecial {
		if u.verbose {
			fmt.Printf("Using special uninstall for %s...\n", app.Name)
		}

		return uninstallFunc(u, ctx)
	}

	// Remove via package manager
	switch app.Method {
	case domain.MethodAPT:
		return u.uninstallAPT(ctx, name)
	case domain.MethodSnap:
		return u.uninstallSnap(ctx, name)
	case domain.MethodFlatpak:
		return u.uninstallFlatpak(ctx, app.Source)
	case domain.MethodDEB:
		return u.uninstallDEB(ctx, name)
	case domain.MethodMise:
		return u.uninstallMise(ctx, name)
	case domain.MethodGitHub, domain.MethodGitHubBinary, domain.MethodGitHubBundle, domain.MethodGitHubJava:
		return u.uninstallGitHub(ctx, name)
	case domain.MethodScript, domain.MethodBinary, domain.MethodAqua:
		return u.uninstallGeneric(ctx, name)
	default:
		return fmt.Errorf("%w for %s", ErrUnsupportedUninstallMethod, name)
	}
}

// UninstallGroup uninstalls all applications in a group.
func (u *Uninstaller) UninstallGroup(ctx context.Context, group string) error {
	appNames, exists := apps.Groups[group]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownGroup, group)
	}

	for _, appName := range appNames {
		if err := u.UninstallApp(ctx, appName); err != nil {
			if u.verbose {
				fmt.Printf("Warning: Failed to uninstall %s: %v\n", appName, err)
			} else {
				fmt.Printf("Warning: Failed to uninstall %s\n", appName)
			}
		}
	}

	return nil
}

// SpecialUninstallFunc represents a special uninstall function that needs access to uninstaller context.
type SpecialUninstallFunc func(*Uninstaller, context.Context) error

// SpecialUninstalls handles apps with custom uninstall logic (now password-aware).
var SpecialUninstalls = map[string]SpecialUninstallFunc{ //nolint:gochecknoglobals
	"chrome": func(u *Uninstaller, ctx context.Context) error {
		// Remove Chrome using dpkg to match installation method
		return u.runCommand(ctx, "sudo", "dpkg", "-r", "google-chrome-stable")
	},
	"vscode": func(uninstaller *Uninstaller, ctx context.Context) error {
		// Remove VSCode and its repository (now uses password-enabled commands)
		if err := uninstaller.runCommand(ctx, "sudo", "apt-get", "remove", "-y", "code"); err != nil {
			return err
		}
		_ = uninstaller.runCommand(ctx, "sudo", "rm", "-f", "/etc/apt/sources.list.d/vscode.list")
		_ = uninstaller.runCommand(ctx, "sudo", "rm", "-f", "/etc/apt/keyrings/packages.microsoft.gpg")

		return nil
	},
	"docker": func(uninstaller *Uninstaller, ctx context.Context) error {
		// Remove Docker completely (now uses password-enabled commands)
		if err := uninstaller.runCommand(ctx, "sudo", "apt-get", "remove", "-y", "docker", "docker-engine", "docker.io", "containerd", "runc"); err != nil {
			return err
		}
		_ = uninstaller.runCommand(ctx, "sudo", "rm", "-rf", "/var/lib/docker")
		_ = uninstaller.runCommand(ctx, "sudo", "rm", "-rf", "/var/lib/containerd")

		return nil
	},
}

// UninstallSpecial handles applications with custom uninstall logic.
func (u *Uninstaller) UninstallSpecial(ctx context.Context, appName string) error {
	uninstallFunc, exists := SpecialUninstalls[appName]
	if !exists {
		return u.UninstallApp(ctx, appName)
	}

	if u.verbose {
		fmt.Printf("Running special uninstall for %s...\n", appName)
	}

	return uninstallFunc(u, ctx)
}

// Private methods (unexported - placed after public methods per funcorder)

func (u *Uninstaller) uninstallAPT(ctx context.Context, packageName string) error {
	return u.runCommand(ctx, "sudo", "apt-get", "remove", "-y", packageName)
}

func (u *Uninstaller) uninstallSnap(ctx context.Context, packageName string) error {
	return u.runCommand(ctx, "sudo", "snap", "remove", packageName)
}

func (u *Uninstaller) uninstallFlatpak(ctx context.Context, packageName string) error {
	// Build command with appropriate flags for TUI/CLI mode (user-level to match installation)
	args := []string{"uninstall", "--user", "-y"}
	if !u.verbose {
		// In TUI mode, use minimal output to prevent progress bar conflicts
		args = append(args, "--noninteractive")
	}

	args = append(args, packageName)

	return u.runCommand(ctx, "flatpak", args...)
}

func (u *Uninstaller) uninstallDEB(ctx context.Context, packageName string) error {
	// Map app keys to actual DEB package names for proper uninstallation
	actualPackageName := mapToDebPackageName(packageName)

	// For DEB packages, try to find package name and remove via APT
	return u.runCommand(ctx, "sudo", "apt-get", "remove", "-y", actualPackageName)
}

func (u *Uninstaller) uninstallMise(ctx context.Context, packageName string) error {
	// Mise can track packages in multiple formats:
	// 1. Plain name: "hadolint", "node", "python"
	// 2. Aqua backend: "aqua:hadolint/hadolint", "aqua:koalaman/shellcheck"
	// We need to detect the actual installed name to uninstall correctly
	actualPackageName := u.detectMisePackageName(ctx, packageName)
	if u.verbose && actualPackageName != packageName {
		fmt.Printf("Detected mise package name: %s -> %s\n", packageName, actualPackageName)
	}

	return u.runCommand(ctx, "mise", "uninstall", actualPackageName)
}

// detectMisePackageName finds the actual package name mise is tracking.
//
//nolint:cyclop // Complexity from multiple package name matching strategies
func (u *Uninstaller) detectMisePackageName(ctx context.Context, packageName string) string {
	// Get list of installed mise packages
	output, err := system.RunWithOutput(ctx, "mise", "list")
	if err != nil {
		// If we can't get the list, return the original name
		return packageName
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		fields := strings.Fields(trimmedLine)
		if len(fields) == 0 {
			continue
		}

		installedName := fields[0]

		// Check for exact match (e.g., "hadolint" == "hadolint")
		if installedName == packageName {
			return installedName
		}

		// Check for aqua backend match (e.g., "aqua:hadolint/hadolint" contains "/hadolint")
		if strings.HasPrefix(installedName, "aqua:") {
			// Extract the tool name from aqua format
			// "aqua:owner/tool" -> "tool"
			parts := strings.Split(installedName, "/")
			if len(parts) == 2 {
				toolName := parts[1]
				if toolName == packageName {
					return installedName
				}
			}
		}

		// Check for other backend formats if they exist in the future
		// e.g., "cargo:packagename", "npm:packagename", etc.
		if strings.Contains(installedName, ":") {
			parts := strings.Split(installedName, ":")
			if len(parts) == 2 {
				// Check if the part after : contains our package name
				backendPart := parts[1]
				if backendPart == packageName || strings.HasSuffix(backendPart, "/"+packageName) {
					return installedName
				}
			}
		}
	}

	// If no match found, return the original name
	// mise will handle the error if the package doesn't exist
	return packageName
}

func (u *Uninstaller) runCommand(ctx context.Context, name string, args ...string) error {
	if u.verbose {
		fmt.Printf("Running: %s %v\n", name, args)
	}

	// If it's a sudo command and we have a password, use the password-enabled method
	if name == "sudo" && u.password != "" {
		return u.executor.RunWithPassword(ctx, u.verbose, u.password, args...)
	}

	// For non-sudo commands or when no password is available, use regular execution
	return u.executor.Run(ctx, u.verbose, name, args...)
}

// mapToDebPackageName maps app keys to their actual DEB package names for uninstallation.
func mapToDebPackageName(appKey string) string {
	// Map app keys to actual package names where they differ
	debPackageMap := map[string]string{
		"chrome": "google-chrome-stable", // Google Chrome DEB package name
		"vscode": "code",                 // Visual Studio Code
		"brave":  "brave-browser",        // Brave Browser
		// Add more mappings as needed
	}

	if packageName, exists := debPackageMap[appKey]; exists {
		return packageName
	}

	// If no mapping exists, use the app key as-is
	return appKey
}

// uninstallGitHub removes GitHub-installed packages (all subcategories).
func (u *Uninstaller) uninstallGitHub(_ context.Context, name string) error {
	homeDir := u.getUserHomeDir()
	userBinDir := filepath.Join(homeDir, ".local", "bin")
	userShareDir := filepath.Join(homeDir, ".local", "share")

	// Remove binary from ~/.local/bin/
	binPath := filepath.Join(userBinDir, name)
	u.removeFile(binPath, "binary")

	// For bundles and Java apps, also remove from ~/.local/share/
	sharePath := filepath.Join(userShareDir, name)
	u.removeDirectory(sharePath, "application directory")

	if u.verbose {
		fmt.Printf("✓ %s removed successfully\n", name)
	}

	return nil
}

// getUserHomeDir gets the user's home directory with fallback.
func (u *Uninstaller) getUserHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to environment variable
		homeDir = os.Getenv("HOME")
	}

	return homeDir
}

// removeFile removes a single file with logging.
func (u *Uninstaller) removeFile(path string, description string) {
	if _, err := os.Stat(path); err != nil {
		return // File doesn't exist, nothing to do
	}

	if err := os.Remove(path); err != nil {
		u.logRemovalError(description, path, err)
	} else if u.verbose {
		fmt.Printf("✓ Removed %s: %s\n", description, path)
	}
}

// removeDirectory removes a directory and its contents with logging.
func (u *Uninstaller) removeDirectory(path string, description string) {
	if _, err := os.Stat(path); err != nil {
		return // Directory doesn't exist, nothing to do
	}

	if err := os.RemoveAll(path); err != nil {
		u.logRemovalError(description, path, err)
	} else if u.verbose {
		fmt.Printf("✓ Removed %s: %s\n", description, path)
	}
}

// logRemovalError logs removal errors based on verbosity.
func (u *Uninstaller) logRemovalError(description string, path string, err error) {
	if u.verbose {
		fmt.Printf("⚠ Failed to remove %s %s: %v\n", description, path, err)
	} else {
		fmt.Printf("⚠ Failed to remove %s %s\n", description, path)
	}
}

// uninstallGeneric removes generically installed packages (binary, script, etc.).
func (u *Uninstaller) uninstallGeneric(_ context.Context, name string) error {
	homeDir := u.getUserHomeDir()
	userBinDir := filepath.Join(homeDir, ".local", "bin")

	// For most generic installations, just remove the binary
	binPath := filepath.Join(userBinDir, name)
	if _, err := os.Stat(binPath); err == nil {
		if err := os.Remove(binPath); err != nil {
			return fmt.Errorf("failed to remove binary %s: %w", binPath, err)
		}

		if u.verbose {
			fmt.Printf("✓ Removed binary: %s\n", binPath)
		}
	} else if u.verbose {
		fmt.Printf("⚠ Binary %s not found (may already be removed)\n", binPath)
	}

	if u.verbose {
		fmt.Printf("✓ %s removed successfully\n", name)
	}

	return nil
}
