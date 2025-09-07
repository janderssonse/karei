// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/domain"
)

const (
	// homeDir is a placeholder for user home directory.
	homeDir = "/home/"
)

// UninstallService handles application removal using hexagonal architecture.
type UninstallService struct {
	fileManager   domain.FileManager
	commandRunner domain.CommandRunner
	installer     domain.PackageInstaller
	verbose       bool
}

// NewUninstallService creates a service for removing applications and configurations.
func NewUninstallService(fm domain.FileManager, cr domain.CommandRunner, pi domain.PackageInstaller, verbose bool) *UninstallService {
	return &UninstallService{
		fileManager:   fm,
		commandRunner: cr,
		installer:     pi,
		verbose:       verbose,
	}
}

var (
	// ErrUnknownApp is returned when an unknown app is requested for uninstallation.
	ErrUnknownApp = errors.New("unknown app")
	// ErrUnsupportedUninstallMethod indicates the uninstall method is not supported.
	ErrUnsupportedUninstallMethod = errors.New("unsupported uninstall method")
	// ErrUnknownGroup indicates the group is not recognized.
	ErrUnknownGroup = errors.New("unknown group")
)

// UninstallApp uninstalls an application by name.
func (s *UninstallService) UninstallApp(ctx context.Context, name string) error {
	app, exists := apps.Apps[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownApp, name)
	}

	if s.verbose {
		fmt.Printf("Uninstalling %s...\n", app.Name)
	}

	// Check for special uninstall logic first
	if hasSpecialUninstall(name) {
		return s.specialUninstall(ctx, name)
	}

	// Convert to domain Package for uninstallation
	pkg := &domain.Package{
		Name:   app.Name,
		Method: app.Method,
		Source: app.Source,
	}

	// Use the PackageInstaller port for uninstallation
	result, err := s.installer.Remove(ctx, pkg)
	if err != nil {
		return fmt.Errorf("failed to uninstall %s: %w", name, err)
	}

	if !result.Success {
		return fmt.Errorf("uninstallation failed for %s", name)
	}

	// Clean up any remaining files
	return s.cleanupAppFiles(ctx, name)
}

// UninstallGroup uninstalls all applications in a group.
func (s *UninstallService) UninstallGroup(ctx context.Context, group string) error {
	appNames, exists := apps.Groups[group]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownGroup, group)
	}

	var failedApps []string

	for _, appName := range appNames {
		if err := s.UninstallApp(ctx, appName); err != nil {
			if s.verbose {
				fmt.Printf("Warning: Failed to uninstall %s: %v\n", appName, err)
			}

			failedApps = append(failedApps, appName)
		}
	}

	if len(failedApps) > 0 {
		return fmt.Errorf("failed to uninstall: %s", strings.Join(failedApps, ", "))
	}

	return nil
}

// ListInstalledApps returns a list of installed applications.
func (s *UninstallService) ListInstalledApps(ctx context.Context) ([]string, error) {
	packages, err := s.installer.List(ctx)
	if err != nil {
		return nil, err
	}

	appNames := make([]string, 0, len(packages))
	for _, pkg := range packages {
		appNames = append(appNames, pkg.Name)
	}

	return appNames, nil
}

// IsAppInstalled checks if an application is installed.
func (s *UninstallService) IsAppInstalled(ctx context.Context, appName string) (bool, error) {
	return s.installer.IsInstalled(ctx, appName)
}

func (s *UninstallService) cleanupAppFiles(_ context.Context, appName string) error {
	// Common cleanup paths
	home := homeDir
	cleanupPaths := []string{
		filepath.Join(home, ".config", appName),
		filepath.Join(home, ".local", "share", appName),
		filepath.Join(home, ".cache", appName),
		filepath.Join("/opt", appName),
		filepath.Join("/usr/local/bin", appName),
	}

	for _, path := range cleanupPaths {
		if s.fileManager.FileExists(path) {
			if err := s.fileManager.RemoveFile(path); err != nil {
				if s.verbose {
					fmt.Printf("Warning: Could not remove %s: %v\n", path, err)
				}
			}
		}
	}

	return nil
}

// hasSpecialUninstall checks if an app requires special uninstallation.
func hasSpecialUninstall(appName string) bool {
	specialApps := []string{
		"chrome", "docker", "vscode", "postman", "obsidian",
		"spotify", "discord", "slack", "teams", "zoom",
	}

	return slices.Contains(specialApps, appName)
}

// specialUninstall handles special uninstallation cases.
func (s *UninstallService) specialUninstall(ctx context.Context, appName string) error {
	uninstallers := map[string]func(context.Context) error{
		"chrome":   s.uninstallChrome,
		"docker":   s.uninstallDocker,
		"vscode":   s.uninstallVSCode,
		"postman":  s.uninstallPostman,
		"obsidian": s.uninstallObsidian,
		"spotify":  s.uninstallSpotify,
		"discord":  s.uninstallDiscord,
		"slack":    s.uninstallSlack,
		"teams":    s.uninstallTeams,
		"zoom":     s.uninstallZoom,
	}

	if uninstaller, ok := uninstallers[appName]; ok {
		return uninstaller(ctx)
	}

	return fmt.Errorf("%w: %s", ErrUnsupportedUninstallMethod, appName)
}

// Special uninstall implementations

func (s *UninstallService) uninstallChrome(ctx context.Context) error {
	// Stop Chrome processes - non-critical if fails
	// pkill returns error if no processes found, which is fine
	_ = s.commandRunner.Execute(ctx, "pkill", "-f", "chrome")

	// Remove Chrome
	if err := s.commandRunner.ExecuteSudo(ctx, "apt", "remove", "--purge", "-y", "google-chrome-stable"); err != nil {
		return err
	}

	// Clean up Chrome directories
	home := homeDir
	chromeDirs := []string{
		filepath.Join(home, ".config", "google-chrome"),
		filepath.Join(home, ".cache", "google-chrome"),
	}

	var cleanupErrors []error

	for _, dir := range chromeDirs {
		if s.fileManager.FileExists(dir) {
			if err := s.fileManager.RemoveFile(dir); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to remove %s: %w", dir, err))
			}
		}
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("chrome uninstall completed with cleanup errors: %v", cleanupErrors)
	}

	return nil
}

func (s *UninstallService) uninstallDocker(ctx context.Context) error {
	// Stop Docker service - log but don't fail if service doesn't exist
	if err := s.commandRunner.ExecuteSudo(ctx, "systemctl", "stop", "docker"); err != nil {
		// Service might not be running or not exist, continue with uninstall
		if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "not loaded") {
			return fmt.Errorf("failed to stop docker service: %w", err)
		}
	}

	if err := s.commandRunner.ExecuteSudo(ctx, "systemctl", "disable", "docker"); err != nil {
		// Service might not be enabled, continue
		if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "not loaded") {
			return fmt.Errorf("failed to disable docker service: %w", err)
		}
	}

	// Remove Docker packages
	dockerPackages := []string{
		"docker-ce", "docker-ce-cli", "containerd.io",
		"docker-compose-plugin", "docker-buildx-plugin",
	}

	args := append([]string{"remove", "--purge", "-y"}, dockerPackages...)
	if err := s.commandRunner.ExecuteSudo(ctx, "apt", args...); err != nil {
		return err
	}

	// Clean up Docker data
	dockerDirs := []string{
		"/var/lib/docker",
		"/var/lib/containerd",
		"/etc/docker",
	}

	for _, dir := range dockerDirs {
		if s.fileManager.FileExists(dir) {
			_ = s.commandRunner.ExecuteSudo(ctx, "rm", "-rf", dir)
		}
	}

	return nil
}

func (s *UninstallService) uninstallVSCode(ctx context.Context) error {
	// Remove VSCode
	if err := s.commandRunner.ExecuteSudo(ctx, "apt", "remove", "--purge", "-y", "code"); err != nil {
		// Try snap if apt fails
		_ = s.commandRunner.Execute(ctx, "snap", "remove", "code")
	}

	// Clean up VSCode directories
	home := homeDir
	vscodeDirs := []string{
		filepath.Join(home, ".config", "Code"),
		filepath.Join(home, ".vscode"),
	}

	for _, dir := range vscodeDirs {
		if s.fileManager.FileExists(dir) {
			_ = s.fileManager.RemoveFile(dir)
		}
	}

	return nil
}

func (s *UninstallService) uninstallPostman(ctx context.Context) error {
	// Remove Postman
	_ = s.commandRunner.Execute(ctx, "snap", "remove", "postman")

	// Remove from /opt if installed there
	if s.fileManager.FileExists("/opt/Postman") {
		_ = s.commandRunner.ExecuteSudo(ctx, "rm", "-rf", "/opt/Postman")
	}

	// Remove desktop entry
	desktopFile := "/usr/share/applications/postman.desktop"
	if s.fileManager.FileExists(desktopFile) {
		_ = s.commandRunner.ExecuteSudo(ctx, "rm", desktopFile)
	}

	return nil
}

func (s *UninstallService) uninstallObsidian(_ context.Context) error {
	// Remove Obsidian AppImage
	home := homeDir
	obsidianPaths := []string{
		filepath.Join(home, ".local", "bin", "Obsidian.AppImage"),
		filepath.Join(home, "Applications", "Obsidian.AppImage"),
		"/opt/Obsidian.AppImage",
	}

	for _, path := range obsidianPaths {
		if s.fileManager.FileExists(path) {
			_ = s.fileManager.RemoveFile(path)
		}
	}

	// Remove config
	configPath := filepath.Join(home, ".config", "obsidian")
	if s.fileManager.FileExists(configPath) {
		_ = s.fileManager.RemoveFile(configPath)
	}

	return nil
}

func (s *UninstallService) uninstallSpotify(ctx context.Context) error {
	// Remove Spotify
	_ = s.commandRunner.Execute(ctx, "snap", "remove", "spotify")

	// Try Flatpak if snap fails
	_ = s.commandRunner.Execute(ctx, "flatpak", "uninstall", "-y", "com.spotify.Client")

	return nil
}

func (s *UninstallService) uninstallDiscord(ctx context.Context) error {
	// Remove Discord
	_ = s.commandRunner.ExecuteSudo(ctx, "apt", "remove", "--purge", "-y", "discord")

	// Try snap if apt fails
	_ = s.commandRunner.Execute(ctx, "snap", "remove", "discord")

	// Clean up Discord directories
	home := homeDir
	discordDirs := []string{
		filepath.Join(home, ".config", "discord"),
		filepath.Join(home, ".cache", "discord"),
	}

	for _, dir := range discordDirs {
		if s.fileManager.FileExists(dir) {
			_ = s.fileManager.RemoveFile(dir)
		}
	}

	return nil
}

func (s *UninstallService) uninstallSlack(ctx context.Context) error {
	// Remove Slack
	_ = s.commandRunner.Execute(ctx, "snap", "remove", "slack")

	// Try APT if snap fails
	_ = s.commandRunner.ExecuteSudo(ctx, "apt", "remove", "--purge", "-y", "slack-desktop")

	return nil
}

func (s *UninstallService) uninstallTeams(ctx context.Context) error {
	// Remove Teams
	_ = s.commandRunner.ExecuteSudo(ctx, "apt", "remove", "--purge", "-y", "teams")

	// Clean up Teams directories
	home := homeDir

	teamsDir := filepath.Join(home, ".config", "Microsoft", "Microsoft Teams")
	if s.fileManager.FileExists(teamsDir) {
		_ = s.fileManager.RemoveFile(teamsDir)
	}

	return nil
}

func (s *UninstallService) uninstallZoom(ctx context.Context) error {
	// Remove Zoom
	_ = s.commandRunner.ExecuteSudo(ctx, "apt", "remove", "--purge", "-y", "zoom")

	// Clean up Zoom directories
	home := homeDir
	zoomDirs := []string{
		filepath.Join(home, ".zoom"),
		filepath.Join(home, ".config", "zoomus.conf"),
	}

	for _, dir := range zoomDirs {
		if s.fileManager.FileExists(dir) {
			_ = s.fileManager.RemoveFile(dir)
		}
	}

	return nil
}

// UninstallPackages uninstalls multiple packages.
func (s *UninstallService) UninstallPackages(ctx context.Context, packages []string) (*domain.UninstallResult, error) {
	result := &domain.UninstallResult{}

	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		if err := s.UninstallApp(ctx, pkg); err != nil {
			result.Failed = append(result.Failed, pkg)
			if errors.Is(err, ErrUnknownApp) {
				result.NotFound = append(result.NotFound, pkg)
			}
		} else {
			result.Uninstalled = append(result.Uninstalled, pkg)
		}
	}

	return result, nil
}
