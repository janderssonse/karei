// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain

import (
	"context"
	"errors"
)

// Common domain errors.
var (
	ErrNoDesktopEnvironment = errors.New("no desktop environment detected")
	ErrNoPackageManager     = errors.New("no supported package manager found")
	ErrMockFileNotFound     = errors.New("mock file not found")
)

// PackageInstaller defines the interface for package installation operations.
// Implemented by adapters for different package managers (apt, dnf, etc.).
type PackageInstaller interface {
	// Install installs a package on the system.
	Install(ctx context.Context, pkg *Package) (*InstallationResult, error)

	// Remove removes a package from the system.
	Remove(ctx context.Context, pkg *Package) (*InstallationResult, error)

	// List returns all installed packages.
	List(ctx context.Context) ([]*Package, error)

	// IsInstalled checks if a package is installed.
	IsInstalled(ctx context.Context, name string) (bool, error)

	// GetBestMethod determines the best installation method for a package.
	GetBestMethod(source string) InstallMethod
}

// SystemDetector defines the interface for system detection operations.
type SystemDetector interface {
	// DetectSystem returns system information.
	DetectSystem(ctx context.Context) (*SystemInfo, error)

	// DetectDistribution returns the Linux distribution information.
	DetectDistribution(ctx context.Context) (*Distribution, error)

	// DetectDesktopEnvironment returns the desktop environment information.
	DetectDesktopEnvironment(ctx context.Context) (*DesktopEnvironment, error)

	// DetectPackageManager returns the primary package manager for this system.
	DetectPackageManager(ctx context.Context) (*PackageManager, error)
}

// CommandRunner defines the interface for executing system commands.
type CommandRunner interface {
	// Execute runs a command and returns the result.
	Execute(ctx context.Context, name string, args ...string) error

	// ExecuteWithOutput runs a command and returns the output.
	ExecuteWithOutput(ctx context.Context, name string, args ...string) (string, error)

	// ExecuteSudo runs a command with sudo privileges.
	ExecuteSudo(ctx context.Context, name string, args ...string) error

	// CommandExists checks if a command is available on the system.
	CommandExists(name string) bool
}

// FileManager defines the interface for file operations.
type FileManager interface {
	// FileExists checks if a file exists.
	FileExists(path string) bool

	// EnsureDir creates a directory and all parent directories if they don't exist.
	EnsureDir(path string) error

	// CopyFile copies a file from source to destination.
	CopyFile(src, dest string) error

	// WriteFile writes data to a file.
	WriteFile(path string, data []byte) error

	// ReadFile reads data from a file.
	ReadFile(path string) ([]byte, error)

	// RemoveFile removes a file.
	RemoveFile(path string) error
}

// NetworkClient defines the interface for network operations.
type NetworkClient interface {
	// DownloadFile downloads a file from a URL to a destination path.
	DownloadFile(ctx context.Context, url, destPath string) error
}
