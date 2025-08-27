// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package main demonstrates how to use the new hexagonal architecture.
// This example shows how easy it will be to add Fedora support.
package main

import (
	"context"
	"fmt"

	"github.com/janderssonse/karei/internal/adapters/platform"
	"github.com/janderssonse/karei/internal/adapters/ubuntu"
	"github.com/janderssonse/karei/internal/application"
	"github.com/janderssonse/karei/internal/domain"
)

func main() {
	ctx := context.Background()

	// Create adapters (these would be chosen based on detected system)
	commandRunner := platform.NewCommandRunner(true, true) // verbose, dry-run
	fileManager := platform.NewFileManager(true)
	systemDetector := platform.NewSystemDetector(commandRunner, fileManager)

	// For Ubuntu systems, use Ubuntu package installer
	packageInstaller := ubuntu.NewPackageInstaller(commandRunner, fileManager, true, true)

	// Create domain services
	packageService := domain.NewPackageService(packageInstaller, systemDetector)

	// Create application service
	installService := application.NewInstallService(packageService, systemDetector)

	// Demonstrate system detection
	systemInfo, err := installService.GetSystemInfo(ctx)
	if err != nil {
		fmt.Printf("Failed to detect system: %v\n", err)
	} else {
		fmt.Printf("Detected system: %s %s (%s family)\n",
			systemInfo.Distribution.Name,
			systemInfo.Distribution.Version,
			systemInfo.Distribution.Family)
		fmt.Printf("Package manager: %s\n", systemInfo.PackageManager.Name)
	}

	// Demonstrate package installation
	result, err := installService.InstallApplication(ctx, "vim", "vim")
	if err != nil {
		fmt.Printf("Installation failed: %v\n", err)

		return
	}

	fmt.Printf("Installation result for %s: success=%v, duration=%dms\n",
		result.Package.Name, result.Success, result.Duration)

	// Demonstrate multiple package installation
	apps := map[string]string{
		"git":    "git",
		"neovim": "neovim",
		"btop":   "btop",
	}

	results, err := installService.InstallMultipleApplications(ctx, apps)
	if err != nil {
		fmt.Printf("Multiple installation failed: %v\n", err)

		return
	}

	fmt.Printf("\nInstalled %d applications:\n", len(results))

	for _, result := range results {
		status := "✓"
		if !result.Success {
			status = "✗"
		}

		fmt.Printf("  %s %s (%s method)\n", status, result.Package.Name, result.Package.Method)
	}

	fmt.Println("\n--- This architecture makes adding Fedora support easy ---")
	fmt.Println("1. Create internal/adapters/fedora/package_installer.go")
	fmt.Println("2. Implement the same PackageInstaller interface with DNF/YUM")
	fmt.Println("3. Update system detection to choose the right adapter")
	fmt.Println("4. All domain logic and application services work unchanged!")
}
