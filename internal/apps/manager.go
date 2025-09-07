// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package apps

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/janderssonse/karei/internal/adapters/platform"
	"github.com/janderssonse/karei/internal/adapters/ubuntu"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/versions"
)

var (
	// ErrUnknownApp is returned when the requested application is not found.
	ErrUnknownApp = errors.New("unknown app")
	// ErrUnknownGroup is returned when the requested group is not found.
	ErrUnknownGroup = errors.New("unknown group")
)

// Manager handles installation and management of applications.
type Manager struct {
	packageInstaller domain.PackageInstaller
	versionManager   *versions.VersionManager
}

// NewManager creates a new application manager with default version manager.
func NewManager(verbose bool) *Manager {
	return NewManagerWithVersionManager(verbose, versions.NewVersionManager(versions.GetVersionsConfigPath()))
}

// NewManagerWithVersionManager creates a new application manager with a custom version manager.
func NewManagerWithVersionManager(verbose bool, versionManager *versions.VersionManager) *Manager {
	// Create platform adapters
	commandRunner := platform.NewCommandRunner(verbose, false) // dryRun=false for real installation
	fileManager := platform.NewFileManager(verbose)

	// Create PackageInstaller with hexagonal architecture
	packageInstaller := ubuntu.NewPackageInstaller(commandRunner, fileManager, verbose, false) // tuiMode=false for CLI

	return &Manager{
		packageInstaller: packageInstaller,
		versionManager:   versionManager,
	}
}

// NewTUIManager creates a new application manager optimized for TUI usage.
// This version suppresses command output to prevent terminal interference.
func NewTUIManager(verbose bool) *Manager {
	return NewTUIManagerWithVersionManager(verbose, versions.NewVersionManager(versions.GetVersionsConfigPath()))
}

// NewTUIManagerWithVersionManager creates a new TUI-optimized application manager with a custom version manager.
func NewTUIManagerWithVersionManager(verbose bool, versionManager *versions.VersionManager) *Manager {
	// Create TUI-optimized platform adapters that suppress output
	commandRunner := platform.NewTUICommandRunner(verbose, false) // dryRun=false, tuiMode=true
	fileManager := platform.NewFileManager(verbose)

	// Create PackageInstaller with TUI mode enabled
	packageInstaller := ubuntu.NewTUIPackageInstaller(commandRunner, fileManager, verbose, false) // tuiMode=true

	return &Manager{
		packageInstaller: packageInstaller,
		versionManager:   versionManager,
	}
}

// InstallApp installs a single application by name.
func (m *Manager) InstallApp(ctx context.Context, name string) error {
	app, exists := Apps[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownApp, name)
	}

	pkg := &domain.Package{
		Name:        name,
		Group:       app.Group,
		Description: app.Description,
		Method:      app.Method,
		Source:      app.Source,
	}

	_, err := m.packageInstaller.Install(ctx, pkg)
	if err != nil {
		return err
	}

	if app.PostInstall != nil {
		return app.PostInstall()
	}

	return nil
}

// InstallGroup installs all applications in the specified group.
func (m *Manager) InstallGroup(ctx context.Context, group string) error {
	apps, exists := Groups[group]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownGroup, group)
	}

	for _, appName := range apps {
		if err := m.InstallApp(ctx, appName); err != nil {
			fmt.Printf("Warning: Failed to install %s: %v\n", appName, err)
		}
	}

	return nil
}

// InstallGroupFunctional installs all applications in the specified group with functional error handling.
func (m *Manager) InstallGroupFunctional(ctx context.Context, group string) error {
	apps, exists := Groups[group]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownGroup, group)
	}

	// Install applications with simple error handling
	var successful []string

	for _, appName := range apps {
		if err := m.InstallApp(ctx, appName); err != nil {
			fmt.Printf("Warning: Failed to install %s: %v\n", appName, err)
		} else {
			successful = append(successful, appName)
		}
	}

	fmt.Printf("Successfully installed %d/%d applications in group %s\n",
		len(successful), len(apps), group)

	return nil
}

// InstallMultipleApps installs multiple applications and returns a map of errors.
func (m *Manager) InstallMultipleApps(ctx context.Context, appNames []string) map[string]error {
	results := make(map[string]error)

	// Install each app with simple error handling
	for _, appName := range appNames {
		if err := m.InstallApp(ctx, appName); err != nil {
			results[appName] = err
		}
	}

	return results
}

// InstallLanguage installs a programming language with the specified version.
func (m *Manager) InstallLanguage(ctx context.Context, lang, version string) error {
	// Install mise if not present
	installed, err := m.packageInstaller.IsInstalled(ctx, "mise")
	if err != nil {
		return fmt.Errorf("failed to check if mise is installed: %w", err)
	}

	if !installed {
		if err := m.InstallApp(ctx, "mise"); err != nil {
			return fmt.Errorf("failed to install mise: %w", err)
		}
	}

	// Use version from config if not provided
	if version == "" {
		configVersion, err := m.versionManager.GetVersion(lang)
		if err != nil {
			// If can't get version from config, use "latest"
			version = "latest"
		} else {
			version = configVersion
		}
	}

	switch lang {
	case "ruby":
		return m.installRuby(ctx, version)
	case "nodejs":
		return m.installNodeJS(ctx, version)
	default:
		return m.installMiseLanguage(ctx, lang, version)
	}
}

// ListApps returns a list of available applications, optionally filtered by group.
func (m *Manager) ListApps(group string) []App {
	var apps []App

	if group == "" {
		for _, app := range Apps {
			apps = append(apps, app)
		}

		return apps
	}

	appNames, exists := Groups[group]
	if !exists {
		return apps
	}

	for _, name := range appNames {
		if app, exists := Apps[name]; exists {
			apps = append(apps, app)
		}
	}

	return apps
}

// ListGroups returns a list of all available application groups.
func (m *Manager) ListGroups() []string {
	groups := make([]string, 0, len(Groups))
	for name := range Groups {
		groups = append(groups, name)
	}

	return groups
}

// IsAppInstalled checks if an application is installed.
func (m *Manager) IsAppInstalled(ctx context.Context, name string) bool {
	app, exists := Apps[name]
	if !exists {
		return false
	}

	// Use the appropriate identifier for each method
	identifier := name

	switch app.Method {
	case domain.MethodFlatpak:
		// For Flatpak, use the Source field which contains the Flatpak ID
		identifier = app.Source
	case domain.MethodMise:
		// For Mise, use the lowercase key since mise commands are case-sensitive
		// The catalog uses capitalized Names but mise needs lowercase
		identifier = name // 'name' here is the key from the catalog which is lowercase
	}

	// Use the more efficient IsInstalledByMethod that only checks the specific method
	// This is MUCH faster than checking all methods sequentially
	type methodChecker interface {
		IsInstalledByMethod(ctx context.Context, name string, method domain.InstallMethod) (bool, error)
	}

	if checker, ok := m.packageInstaller.(methodChecker); ok {
		installed, err := checker.IsInstalledByMethod(ctx, identifier, app.Method)
		if err != nil {
			// This should rarely happen - only for real errors
			return false
		}

		return installed
	}

	// Fallback to the old method if IsInstalledByMethod is not available
	installed, err := m.packageInstaller.IsInstalled(ctx, identifier)
	if err != nil {
		return false
	}

	return installed
}

func (m *Manager) installRuby(ctx context.Context, version string) error {
	// Install Ruby via mise
	if err := m.installMiseLanguage(ctx, "ruby", version); err != nil {
		return err
	}

	// Install Rails - mise automatically shims gem command
	return exec.CommandContext(ctx, "gem", "install", "rails", "--no-document").Run()
}

func (m *Manager) installNodeJS(ctx context.Context, version string) error {
	if version == "latest" {
		version = "lts"
	}

	return m.installMiseLanguage(ctx, "nodejs", version)
}

func (m *Manager) installMiseLanguage(ctx context.Context, lang, version string) error {
	// Map language name to mise tool name
	miseToolName := Languages[lang]
	if miseToolName == "" {
		miseToolName = lang // fallback to original name
	}

	// Install language version with mise (no plugin needed)
	if err := exec.CommandContext(ctx, "mise", "install", miseToolName+"@"+version).Run(); err != nil { //nolint:gosec
		return fmt.Errorf("failed to install %s %s: %w", lang, version, err)
	}

	// Set global version
	return exec.CommandContext(ctx, "mise", "use", "--global", miseToolName+"@"+version).Run() //nolint:gosec
}
