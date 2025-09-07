// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package ubuntu implements Ubuntu/Debian package management adapters.
package ubuntu

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/network"
)

// Static error definitions for err113 compliance.
var (
	ErrDownloadFailed        = errors.New("download failed")
	ErrMiseNotInstalled      = errors.New("mise is not installed - install mise first")
	ErrAquaNotInstalled      = errors.New("aqua is not installed - install aqua first")
	ErrPMDURLNotFound        = errors.New("could not determine latest PMD download URL")
	ErrGitHubBinaryNotImpl   = errors.New("downloadGitHubBinary not yet implemented")
	ErrGitHubReleaseNotImpl  = errors.New("downloadGitHubRelease not yet implemented")
	ErrExtractBundleNotImpl  = errors.New("extractBundle not yet implemented")
	ErrSymlinkBundleNotImpl  = errors.New("symlinkBundleExecutables not yet implemented")
	ErrGenericJavaAppNotImpl = errors.New("installGenericJavaApp not yet implemented")
)

// ReleasePattern represents a GitHub release download pattern.
type ReleasePattern struct {
	Filename  string // The filename pattern to try
	Extension string // The file extension (.zip or .tar.gz)
}

// PackageInstaller implements the PackageInstaller port for Linux systems.
type PackageInstaller struct {
	commandRunner domain.CommandRunner
	fileManager   domain.FileManager
	verbose       bool
	dryRun        bool
	tuiMode       bool // When true, suppress progress messages for TUI compatibility
}

// NewPackageInstaller creates a new Linux package installer with the provided dependencies.
func NewPackageInstaller(commandRunner domain.CommandRunner, fileManager domain.FileManager, verbose, dryRun bool) *PackageInstaller {
	return &PackageInstaller{
		commandRunner: commandRunner,
		fileManager:   fileManager,
		verbose:       verbose,
		dryRun:        dryRun,
		tuiMode:       false, // Default to CLI mode
	}
}

// NewTUIPackageInstaller creates a new Linux package installer optimized for TUI mode.
func NewTUIPackageInstaller(commandRunner domain.CommandRunner, fileManager domain.FileManager, verbose, dryRun bool) *PackageInstaller {
	return &PackageInstaller{
		commandRunner: commandRunner,
		fileManager:   fileManager,
		verbose:       verbose,
		dryRun:        dryRun,
		tuiMode:       true, // Enable TUI mode - suppress progress messages
	}
}

// Install installs a package using the appropriate method.
func (p *PackageInstaller) Install(ctx context.Context, pkg *domain.Package) (*domain.InstallationResult, error) {
	startTime := time.Now()
	result := &domain.InstallationResult{
		Package: pkg,
		Success: false,
	}

	if p.verbose && !p.tuiMode {
		fmt.Printf("Installing %s using method %s\n", pkg.Name, pkg.Method)
	}

	err := p.executeInstallMethod(ctx, pkg)

	result.Duration = time.Since(startTime).Milliseconds()
	result.Success = err == nil
	result.Error = err

	return result, err
}

// Remove removes a package using the appropriate method.
func (p *PackageInstaller) Remove(ctx context.Context, pkg *domain.Package) (*domain.InstallationResult, error) {
	startTime := time.Now()
	result := &domain.InstallationResult{
		Package: pkg,
		Success: false,
	}

	var err error

	switch pkg.Method {
	case domain.MethodAPT:
		err = p.removeAPT(ctx, pkg)
	case domain.MethodSnap:
		err = p.removeSnap(ctx, pkg)
	case domain.MethodFlatpak:
		err = p.removeFlatpak(ctx, pkg)
	case domain.MethodGitHub, domain.MethodGitHubBinary, domain.MethodGitHubBundle, domain.MethodGitHubJava:
		err = p.removeGitHub(ctx, pkg)
	case domain.MethodDEB, domain.MethodScript, domain.MethodMise, domain.MethodAqua, domain.MethodBinary:
		err = p.removeGeneric(ctx, pkg)
	default:
		err = domain.ErrUnsupportedRemoveMethod
	}

	result.Duration = time.Since(startTime).Milliseconds()
	result.Success = err == nil
	result.Error = err

	return result, err
}

// List returns a list of installed packages.
func (p *PackageInstaller) List(ctx context.Context) ([]*domain.Package, error) {
	output, err := p.commandRunner.ExecuteWithOutput(ctx, "dpkg-query", "-W", "--showformat=${Package} ${Version}\\n")
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var packages []*domain.Package

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packages = append(packages, &domain.Package{
					Name:    parts[0],
					Version: parts[1],
					Method:  domain.MethodAPT,
					Source:  parts[0], // For APT, source is the package name
				})
			}
		}
	}

	return packages, nil
}

// IsInstalled checks if a package is installed using intelligent detection.
// Detection order follows preferred installation methods:
// 1. Flatpak (GUI apps) 2. Mise (CLI tools) 3. APT/RPM 4. Snap 5. GitHub releases.
func (p *PackageInstaller) IsInstalled(ctx context.Context, name string) (bool, error) {
	// Try detection methods in priority order
	if installed, err := p.checkFlatpakFirst(ctx, name); installed || err != nil {
		return installed, err
	}

	if installed := p.checkMiseSecond(ctx, name); installed {
		return true, nil
	}

	if installed := p.checkAPTThird(ctx, name); installed {
		return true, nil
	}

	if installed := p.checkSnapFourth(ctx, name); installed {
		return true, nil
	}

	if installed := p.checkAquaFifth(ctx, name); installed {
		return true, nil
	}

	if installed := p.checkBinaryFallback(name); installed {
		return true, nil
	}

	return false, nil
}

// IsInstalledByMethod checks if a package is installed using a specific method.
// This is much more efficient than checking all methods.
func (p *PackageInstaller) IsInstalledByMethod(ctx context.Context, name string, method domain.InstallMethod) (bool, error) {
	switch method {
	case domain.MethodAPT:
		return p.checkAPTThird(ctx, name), nil
	case domain.MethodSnap:
		return p.checkSnapFourth(ctx, name), nil
	case domain.MethodFlatpak:
		return p.checkFlatpakFirst(ctx, name)
	case domain.MethodMise:
		return p.checkMiseSecond(ctx, name), nil
	case domain.MethodAqua:
		return p.checkAquaFifth(ctx, name), nil
	case domain.MethodDEB:
		// DEB packages are checked via APT/dpkg
		return p.checkAPTThird(ctx, name), nil
	case domain.MethodScript, domain.MethodBinary,
		domain.MethodGitHub, domain.MethodGitHubBinary,
		domain.MethodGitHubBundle, domain.MethodGitHubJava:
		// These typically install binaries, check PATH
		return p.checkBinaryFallback(name), nil
	default:
		// Fallback to checking all methods
		return p.IsInstalled(ctx, name)
	}
}

// GetBestMethod returns the best installation method for a given source.
func (p *PackageInstaller) GetBestMethod(source string) domain.InstallMethod {
	if strings.Contains(source, "github.com") {
		return domain.MethodGitHub
	}

	if strings.HasSuffix(source, ".deb") {
		return domain.MethodDEB
	}

	if strings.Contains(source, "flatpak") || strings.Contains(source, "flathub") {
		return domain.MethodFlatpak
	}

	if strings.Contains(source, "snap") {
		return domain.MethodSnap
	}
	// Default to APT for Linux
	return domain.MethodAPT
}

func (p *PackageInstaller) executeInstallMethod(ctx context.Context, pkg *domain.Package) error {
	// Use method dispatch map for reduced complexity
	return p.dispatchInstallMethod(ctx, pkg)
}

func (p *PackageInstaller) dispatchInstallMethod(ctx context.Context, pkg *domain.Package) error {
	switch pkg.Method {
	case domain.MethodAPT:
		return p.installAPT(ctx, pkg)
	case domain.MethodSnap:
		return p.installSnap(ctx, pkg)
	case domain.MethodFlatpak:
		return p.installFlatpak(ctx, pkg)
	case domain.MethodDEB:
		return p.installDEB(ctx, pkg)
	case domain.MethodScript:
		return p.installScript(ctx, pkg)
	case domain.MethodMise:
		return p.installMise(ctx, pkg)
	case domain.MethodAqua:
		return p.installAqua(ctx, pkg)
	case domain.MethodBinary:
		return p.installBinary(ctx, pkg)
	default:
		return p.dispatchGitHubMethod(ctx, pkg)
	}
}

func (p *PackageInstaller) dispatchGitHubMethod(ctx context.Context, pkg *domain.Package) error {
	switch pkg.Method {
	case domain.MethodGitHub:
		return p.installGitHub(ctx, pkg)
	case domain.MethodGitHubBinary:
		return p.installGitHubBinary(ctx, pkg)
	case domain.MethodGitHubBundle:
		return p.installGitHubBundle(ctx, pkg)
	case domain.MethodGitHubJava:
		return p.installGitHubJava(ctx, pkg)
	default:
		return domain.ErrUnsupportedInstallMethod
	}
}

// checkFlatpakFirst checks Flatpak first for GUI applications with reverse domain names.
func (p *PackageInstaller) checkFlatpakFirst(ctx context.Context, name string) (bool, error) {
	if p.isFlatpakAppID(name) {
		return p.isFlatpakInstalled(ctx, name)
	}

	return false, nil
}

// checkMiseSecond checks Mise for CLI development tools.
func (p *PackageInstaller) checkMiseSecond(ctx context.Context, name string) bool {
	if p.commandRunner.CommandExists("mise") {
		return p.isMiseInstalled(ctx, name)
	}

	return false
}

// checkAPTThird checks APT/DEB system package manager.
func (p *PackageInstaller) checkAPTThird(ctx context.Context, name string) bool {
	// Quick check using dpkg-query which is more reliable
	// dpkg-query returns exit code 1 if package not found (which is normal)
	// We use -W with format to get clean output
	output, err := p.commandRunner.ExecuteWithOutput(ctx, "dpkg-query", "-W", "-f=${Status}", name)
	if err != nil {
		// Check if it's a context cancellation/timeout (real error)
		if ctx.Err() != nil {
			// Context cancelled or timed out - this is a real error
			// But for IsInstalled, we can only return bool, so return false
			// The calling code should handle context errors separately
			return false
		}
		// Otherwise, package not found is normal - not an error
		return false
	}

	// Check if status contains "install ok installed"
	// This is the proper way to check if a package is fully installed
	return strings.Contains(output, "install ok installed")
}

// checkSnapFourth checks Snap secondary package manager.
func (p *PackageInstaller) checkSnapFourth(ctx context.Context, name string) bool {
	if p.commandRunner.CommandExists("snap") {
		return p.isSnapInstalled(ctx, name)
	}

	return false
}

// checkAquaFifth checks Aqua for GitHub releases and security tools.
func (p *PackageInstaller) checkAquaFifth(ctx context.Context, name string) bool {
	if p.commandRunner.CommandExists("aqua") {
		return p.isAquaInstalled(ctx, name)
	}

	return false
}

// checkBinaryFallback checks if binary exists in PATH as final fallback.
func (p *PackageInstaller) checkBinaryFallback(name string) bool {
	return p.commandRunner.CommandExists(name)
}

// isFlatpakAppID determines if a name is a Flatpak application ID.
func (p *PackageInstaller) isFlatpakAppID(name string) bool {
	return strings.Contains(name, ".") && (strings.HasPrefix(name, "com.") ||
		strings.HasPrefix(name, "org.") || strings.HasPrefix(name, "io.") ||
		strings.HasPrefix(name, "net.") || strings.HasPrefix(name, "de.") ||
		strings.HasPrefix(name, "fr.") || strings.HasPrefix(name, "app."))
}

// isFlatpakInstalled checks if a Flatpak package is installed.
func (p *PackageInstaller) isFlatpakInstalled(ctx context.Context, appID string) (bool, error) {
	if !p.commandRunner.CommandExists("flatpak") {
		return false, nil
	}

	// Use flatpak list to check if app is installed
	output, err := p.commandRunner.ExecuteWithOutput(ctx, "flatpak", "list", "--app", "--columns=application")
	if err != nil {
		return false, err // Return the actual error
	}

	// Check if appID is in the output
	installedApps := strings.Split(strings.TrimSpace(output), "\n")
	for _, installedApp := range installedApps {
		if strings.TrimSpace(installedApp) == appID {
			return true, nil
		}
	}

	return false, nil
}

// isSnapInstalled checks if a Snap package is installed.
func (p *PackageInstaller) isSnapInstalled(ctx context.Context, name string) bool {
	if !p.commandRunner.CommandExists("snap") {
		return false
	}

	// Use snap list to check if package is installed
	err := p.commandRunner.Execute(ctx, "snap", "list", name)

	return err == nil
}

// Private installation methods

func (p *PackageInstaller) installAPT(ctx context.Context, pkg *domain.Package) error {
	// Check if already installed
	installed, err := p.IsInstalled(ctx, pkg.Source)
	if err == nil && installed {
		if p.verbose && !p.tuiMode {
			fmt.Printf("Package %s already installed\n", pkg.Source)
		}

		return nil
	}

	if p.dryRun {
		if !p.tuiMode {
			fmt.Printf("DRY RUN: sudo apt update && sudo apt install -y %s\n", pkg.Source)
		}

		return nil
	}

	if !p.tuiMode {
		fmt.Printf("Installing %s via APT...\n", pkg.Source)
	}

	// Update package lists with proxy settings
	updateArgs := append([]string{"apt-get"}, network.ConfigureAPTProxy()...)

	updateArgs = append(updateArgs, "update")
	if err := p.commandRunner.ExecuteSudo(ctx, updateArgs[0], updateArgs[1:]...); err != nil {
		return fmt.Errorf("failed to update package lists: %w", err)
	}

	// Install package with proxy settings
	installArgs := append([]string{"apt-get"}, network.ConfigureAPTProxy()...)
	installArgs = append(installArgs, "install", "-y", pkg.Source)

	return p.commandRunner.ExecuteSudo(ctx, installArgs[0], installArgs[1:]...)
}

func (p *PackageInstaller) installSnap(ctx context.Context, pkg *domain.Package) error {
	// Check if already installed
	if p.isSnapInstalled(ctx, pkg.Source) {
		if p.verbose && !p.tuiMode {
			fmt.Printf("Snap %s already installed\n", pkg.Source)
		}

		return nil
	}

	if p.dryRun {
		if !p.tuiMode {
			fmt.Printf("DRY RUN: sudo snap install %s\n", pkg.Source)
		}

		return nil
	}

	if !p.tuiMode {
		fmt.Printf("Installing %s via Snap...\n", pkg.Source)
	}

	// Parse options from Source if they exist (e.g., "package --classic")
	parts := strings.Fields(pkg.Source)
	args := []string{"install"}

	// Add any options that came with the source
	if len(parts) > 1 {
		args = append(args, parts[1:]...)
		args = append(args, parts[0]) // Package name goes last
	} else {
		args = append(args, pkg.Source)
	}

	return p.commandRunner.ExecuteSudo(ctx, "snap", args...)
}

func (p *PackageInstaller) installFlatpak(ctx context.Context, pkg *domain.Package) error {
	// Check if already installed
	if installed, err := p.isFlatpakInstalled(ctx, pkg.Source); err == nil && installed {
		if p.verbose && !p.tuiMode {
			fmt.Printf("Flatpak %s already installed\n", pkg.Source)
		}

		return nil
	}

	if p.dryRun {
		if !p.tuiMode {
			fmt.Printf("DRY RUN: flatpak install -y --user flathub %s\n", pkg.Source)
		}

		return nil
	}

	// Ensure Flathub remote is added
	if err := p.ensureFlathubRemote(ctx); err != nil {
		return fmt.Errorf("failed to ensure Flathub remote: %w", err)
	}

	if !p.tuiMode {
		fmt.Printf("⬛ Installing %s via Flatpak from Flathub...\n", pkg.Source)
	}

	// Build install command with appropriate flags
	args := []string{"install", "-y", "--user"}
	// Note: --noninteractive is not a valid Flatpak flag, removed
	args = append(args, "flathub", pkg.Source)

	return p.commandRunner.Execute(ctx, "flatpak", args...)
}

func (p *PackageInstaller) installDEB(ctx context.Context, pkg *domain.Package) error {
	if p.dryRun {
		// DRY RUN: download and install would happen here - TUI handles display
		return nil
	}

	if p.verbose {
		fmt.Printf("Installing DEB package: %s\n", pkg.Source)
	}

	debPath := pkg.Source

	// Download if URL
	if strings.HasPrefix(pkg.Source, "http") {
		if p.verbose {
			fmt.Printf("◦ DEB package will be downloaded from: %s\n", pkg.Source)
		}

		tempFile, err := p.downloadDEBFile(ctx, pkg.Source)
		if err != nil {
			return fmt.Errorf("failed to download DEB: %w", err)
		}

		debPath = tempFile
	}

	// Install using dpkg with sudo
	if err := p.commandRunner.ExecuteSudo(ctx, "dpkg", "-i", debPath); err != nil {
		// Return the dpkg error without attempting automatic fixes
		// User should manually resolve dependency issues
		return fmt.Errorf("dpkg installation failed: %w", err)
	}

	if p.verbose {
		fmt.Printf("✓ DEB package installed successfully\n")
	}

	return nil
}

// installGitHub provides fallback to the old generic method.
func (p *PackageInstaller) installGitHub(ctx context.Context, pkg *domain.Package) error {
	if !p.tuiMode {
		fmt.Printf("⚠ Using legacy GitHub installation method for %s\n", pkg.Name)
		fmt.Printf("  Consider updating to: github-binary, github-bundle, or github-java\n")
	}

	// Fallback to binary installation for compatibility
	return p.installGitHubBinary(ctx, pkg)
}

// installGitHubBinary installs a single binary from GitHub releases.
func (p *PackageInstaller) installGitHubBinary(ctx context.Context, pkg *domain.Package) error {
	if p.commandRunner.CommandExists(pkg.Name) {
		if p.verbose && !p.tuiMode {
			fmt.Printf("Binary %s already available\n", pkg.Name)
		}

		return nil
	}

	if p.dryRun {
		if !p.tuiMode {
			fmt.Printf("DRY RUN: install GitHub binary from %s\n", pkg.Source)
		}

		return nil
	}

	if !p.tuiMode {
		fmt.Printf("Installing %s binary from GitHub...\n", pkg.Name)
	}

	// Handle direct URLs (like mise)
	if strings.HasPrefix(pkg.Source, "http") {
		return p.downloadDirectBinary(ctx, pkg)
	}

	// Handle repository/name format
	return p.downloadGitHubBinary(ctx, pkg)
}

// installGitHubBundle installs applications with directory structure to ~/.local/share.
func (p *PackageInstaller) installGitHubBundle(ctx context.Context, pkg *domain.Package) error {
	if p.dryRun {
		if !p.tuiMode {
			fmt.Printf("DRY RUN: install GitHub bundle from %s\n", pkg.Source)
		}

		return nil
	}

	if !p.tuiMode {
		fmt.Printf("Installing %s bundle from GitHub...\n", pkg.Name)
	}

	// Download and extract to ~/.local/share/toolname/
	userShare := filepath.Join(filepath.Dir(p.getUserBinDir()), "share", pkg.Name)
	if err := p.fileManager.EnsureDir(userShare); err != nil {
		return fmt.Errorf("failed to create share directory: %w", err)
	}

	// Download bundle
	tempFile, err := p.downloadGitHubRelease(ctx, pkg, []string{".tar.gz", ".zip"})
	if err != nil {
		return err
	}

	// Extract bundle
	if err := p.extractBundle(tempFile, userShare); err != nil {
		return err
	}

	// Create symlinks for executables in ~/.local/bin/
	return p.symlinkBundleExecutables(userShare, pkg.Name)
}

// installGitHubJava installs Java applications with proper structure and wrapper.
func (p *PackageInstaller) installGitHubJava(ctx context.Context, pkg *domain.Package) error {
	if p.dryRun {
		if !p.tuiMode {
			fmt.Printf("DRY RUN: install GitHub Java app from %s\n", pkg.Source)
		}

		return nil
	}

	if !p.tuiMode {
		fmt.Printf("Installing %s Java application from GitHub...\n", pkg.Name)
	}

	// Special handling for known Java tools
	if pkg.Name == "pmd" {
		return p.installPMDFromGitHub(ctx, pkg)
	}

	// Generic Java app installation
	return p.installGenericJavaApp(ctx, pkg)
}

func (p *PackageInstaller) installScript(ctx context.Context, pkg *domain.Package) error {
	if p.dryRun {
		if !p.tuiMode {
			fmt.Printf("DRY RUN: run custom script %s\n", pkg.Source)
		}

		return nil
	}

	if !p.tuiMode {
		fmt.Printf("Running install script: %s\n", pkg.Source)
	}

	scriptPath := pkg.Source

	// Download if URL
	if strings.HasPrefix(scriptPath, "http") {
		if !p.tuiMode {
			fmt.Printf("⚠ Install script will be downloaded and executed from: %s\n", scriptPath)
			fmt.Printf("⬢ Security notice: Only run scripts from trusted sources\n")
		}

		tempFile := filepath.Join(os.TempDir(), "install.sh")
		if err := p.downloadFile(ctx, scriptPath, tempFile); err != nil {
			return fmt.Errorf("failed to download script: %w", err)
		}

		scriptPath = tempFile
	}

	return p.commandRunner.Execute(ctx, "bash", scriptPath)
}

func (p *PackageInstaller) removeAPT(ctx context.Context, pkg *domain.Package) error {
	if p.dryRun {
		// DRY RUN: APT removal would happen here - TUI handles display
		return nil
	}

	fmt.Printf("Uninstalling %s...\n", pkg.Source)

	return p.commandRunner.ExecuteSudo(ctx, "apt-get", "remove", "-y", pkg.Source)
}

func (p *PackageInstaller) removeSnap(ctx context.Context, pkg *domain.Package) error {
	if p.dryRun {
		// DRY RUN: Snap removal would happen here - TUI handles display
		return nil
	}

	return p.commandRunner.ExecuteSudo(ctx, "snap", "remove", pkg.Source)
}

func (p *PackageInstaller) removeFlatpak(ctx context.Context, pkg *domain.Package) error {
	if p.dryRun {
		// DRY RUN: Flatpak removal would happen here - TUI handles display
		return nil
	}

	if p.verbose {
		fmt.Printf("Removing Flatpak %s...\n", pkg.Source)
	}

	// Build uninstall command with appropriate flags
	args := []string{"uninstall", "-y", "--user"}
	// Note: --noninteractive is not a valid Flatpak flag, removed
	args = append(args, pkg.Source)

	return p.commandRunner.Execute(ctx, "flatpak", args...)
}

// ensureFlathubRemote adds the Flathub remote if not present.
func (p *PackageInstaller) ensureFlathubRemote(ctx context.Context) error {
	if !p.tuiMode {
		fmt.Printf("• Connecting to Flathub repository...\n")
	}

	// Build remote-add command with appropriate flags (user-level)
	remoteArgs := []string{"remote-add", "--if-not-exists", "--user"}
	// Note: --noninteractive is not a valid Flatpak flag, removed
	remoteArgs = append(remoteArgs, "flathub", "https://dl.flathub.org/repo/flathub.flatpakrepo")

	return p.commandRunner.Execute(ctx, "flatpak", remoteArgs...)
}

// downloadDEBFile downloads a DEB file from URL to temp directory.
func (p *PackageInstaller) downloadDEBFile(ctx context.Context, url string) (string, error) {
	// Download progress handled by TUI

	// Create HTTP client with timeout and proxy support
	client := network.GetHTTPClient()
	client.Timeout = 15 * time.Minute // Allow up to 15 minutes for large downloads like Chrome DEB

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "karei/1.0")

	// Download execution - progress handled by TUI

	resp, err := client.Do(req)
	if err != nil {
		// Download error handled by error return - TUI displays errors
		return "", fmt.Errorf("failed to download %s: %w", url, err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			if p.verbose {
				fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
			}
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w with status %d: %s", ErrDownloadFailed, resp.StatusCode, resp.Status)
	}

	// Download size logging suppressed - TUI handles progress display

	// Create temp file
	tempFile := filepath.Join(os.TempDir(), "package.deb")

	outFile, err := os.Create(tempFile) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %w", tempFile, err)
	}

	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			if p.verbose {
				fmt.Printf("Warning: failed to close output file: %v\n", closeErr)
			}
		}
	}()

	// Copy response body to file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", tempFile, err)
	}

	// Download completion logging suppressed - TUI handles success display

	return tempFile, nil
}

// installMise installs a package using mise (development environment manager).
func (p *PackageInstaller) installMise(ctx context.Context, pkg *domain.Package) error {
	if err := p.validateMisePrerequisites(ctx, pkg); err != nil {
		return err
	}

	if p.dryRun {
		return p.handleMiseDryRun(pkg)
	}

	return p.executeMiseInstall(ctx, pkg)
}

func (p *PackageInstaller) validateMisePrerequisites(ctx context.Context, pkg *domain.Package) error {
	// Check if mise is available
	if !p.commandRunner.CommandExists("mise") {
		return ErrMiseNotInstalled
	}

	// Check if package is already installed via mise
	if p.isMiseInstalled(ctx, pkg.Name) {
		if p.verbose && !p.tuiMode {
			fmt.Printf("Package %s already installed via mise\n", pkg.Name)
		}

		return nil
	}

	return nil
}

func (p *PackageInstaller) handleMiseDryRun(pkg *domain.Package) error {
	if !p.tuiMode {
		fmt.Printf("DRY RUN: mise use -g %s\n", pkg.Source)
	}

	return nil
}

func (p *PackageInstaller) executeMiseInstall(ctx context.Context, pkg *domain.Package) error {
	if !p.tuiMode {
		fmt.Printf("Installing %s via mise (development environment manager)...\n", pkg.Name)
		fmt.Printf("⬛ Tool will be downloaded and managed by mise\n")
	}

	// Setup mise configuration if not already done
	if err := p.setupMiseConfig(); err != nil {
		return fmt.Errorf("failed to setup mise config: %w", err)
	}

	return p.runMiseCommand(ctx, pkg)
}

func (p *PackageInstaller) runMiseCommand(ctx context.Context, pkg *domain.Package) error {
	tool := pkg.Source
	if tool == "" {
		tool = pkg.Name
	}

	version := pkg.Version
	if version == "" || version == "latest" {
		// For latest, let mise determine the version
		return p.commandRunner.Execute(ctx, "mise", "use", "-g", tool)
	}

	// For specific version, use tool@version format
	return p.commandRunner.Execute(ctx, "mise", "use", "-g", tool+"@"+version)
}

// installAqua installs a package using aqua (declarative CLI version manager).
func (p *PackageInstaller) installAqua(ctx context.Context, pkg *domain.Package) error {
	if err := p.validateAquaPrerequisites(ctx, pkg); err != nil {
		return err
	}

	if p.dryRun {
		return p.handleAquaDryRun(pkg)
	}

	return p.executeAquaInstall(ctx, pkg)
}

func (p *PackageInstaller) validateAquaPrerequisites(ctx context.Context, pkg *domain.Package) error {
	// Check if aqua is available
	if !p.commandRunner.CommandExists("aqua") {
		return ErrAquaNotInstalled
	}

	// Check if package is already installed via aqua
	if p.isAquaInstalled(ctx, pkg.Name) {
		if p.verbose && !p.tuiMode {
			fmt.Printf("Package %s already installed via aqua\n", pkg.Name)
		}

		return nil
	}

	return nil
}

func (p *PackageInstaller) handleAquaDryRun(pkg *domain.Package) error {
	if !p.tuiMode {
		fmt.Printf("DRY RUN: aqua i -c ~/.config/aqua/aqua.yaml %s\n", pkg.Source)
	}

	return nil
}

func (p *PackageInstaller) executeAquaInstall(ctx context.Context, pkg *domain.Package) error {
	if !p.tuiMode {
		fmt.Printf("Installing %s via aqua (declarative CLI version manager)...\n", pkg.Name)
		fmt.Printf("• Connecting to aqua registry for package information...\n")
		fmt.Printf("⬛ Package will be downloaded from GitHub releases\n")
	}

	aquaConfig, err := p.setupAquaConfig()
	if err != nil {
		return err
	}

	aquaPackage := p.getAquaPackageName(pkg)

	if err := p.addPackageToAquaConfig(aquaConfig, aquaPackage); err != nil {
		return fmt.Errorf("failed to add package to aqua config: %w", err)
	}

	return p.runAquaInstall(ctx, aquaConfig, aquaPackage)
}

func (p *PackageInstaller) setupAquaConfig() (string, error) {
	aquaConfig := p.getXDGConfigHome() + "/aqua/aqua.yaml"
	if !p.fileManager.FileExists(aquaConfig) {
		if err := p.createAquaConfig(aquaConfig); err != nil {
			return "", fmt.Errorf("failed to create aqua config: %w", err)
		}
	}

	return aquaConfig, nil
}

func (p *PackageInstaller) getAquaPackageName(pkg *domain.Package) string {
	aquaPackage := pkg.Source
	if aquaPackage == "" {
		aquaPackage = pkg.Name
	}

	return aquaPackage
}

func (p *PackageInstaller) runAquaInstall(ctx context.Context, aquaConfig, aquaPackage string) error {
	// Install using aqua with project config and proper AQUA_ROOT_DIR
	// Set AQUA_ROOT_DIR to user's .local directory so binaries install to .local/bin
	userLocal := filepath.Dir(p.getUserBinDir()) // This gives us ~/.local

	// Use ExecuteWithEnv if available, otherwise use Execute
	// For now, we'll set the environment variable before execution
	if err := os.Setenv("AQUA_ROOT_DIR", userLocal); err != nil {
		return fmt.Errorf("failed to set AQUA_ROOT_DIR: %w", err)
	}

	defer func() { _ = os.Unsetenv("AQUA_ROOT_DIR") }()

	return p.commandRunner.Execute(ctx, "aqua", "i", "-c", aquaConfig, aquaPackage)
}

// installBinary installs a pre-compiled binary directly.
func (p *PackageInstaller) installBinary(ctx context.Context, pkg *domain.Package) error {
	if p.commandRunner.CommandExists(pkg.Name) {
		return p.handleBinaryAlreadyExists(pkg)
	}

	if p.dryRun {
		return p.handleBinaryDryRun(pkg)
	}

	return p.executeBinaryInstall(ctx, pkg)
}

func (p *PackageInstaller) handleBinaryAlreadyExists(pkg *domain.Package) error {
	if p.verbose && !p.tuiMode {
		fmt.Printf("Binary %s already available\n", pkg.Name)
	}

	return nil
}

func (p *PackageInstaller) handleBinaryDryRun(pkg *domain.Package) error {
	if !p.tuiMode {
		fmt.Printf("DRY RUN: download and install binary %s\n", pkg.Source)
	}

	return nil
}

func (p *PackageInstaller) executeBinaryInstall(ctx context.Context, pkg *domain.Package) error {
	if !p.tuiMode {
		fmt.Printf("Installing %s binary from %s...\n", pkg.Name, pkg.Source)
	}

	tempFile, err := p.downloadBinaryToTemp(ctx, pkg)
	if err != nil {
		return err
	}

	return p.installBinaryFromTemp(tempFile, pkg)
}

func (p *PackageInstaller) downloadBinaryToTemp(ctx context.Context, pkg *domain.Package) (string, error) {
	// Download binary to temp file
	tempFile := filepath.Join(os.TempDir(), pkg.Name)
	if err := p.downloadFile(ctx, pkg.Source, tempFile); err != nil {
		return "", fmt.Errorf("failed to download binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(tempFile, 0755); err != nil { //nolint:gosec // G302: Executable files need 0755 permissions
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	return tempFile, nil
}

func (p *PackageInstaller) installBinaryFromTemp(tempFile string, pkg *domain.Package) error {
	// Move to user bin directory
	binDir := p.getUserBinDir()
	if err := p.fileManager.EnsureDir(binDir); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	targetPath := filepath.Join(binDir, pkg.Name)
	if err := os.Rename(tempFile, targetPath); err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}

	if p.verbose && !p.tuiMode {
		fmt.Printf("✅ %s installed successfully to %s\n", pkg.Name, targetPath)
	}

	return nil
}

// Helper methods for checking if packages are installed

func (p *PackageInstaller) isMiseInstalled(ctx context.Context, name string) bool {
	// First try the exact name
	if err := p.commandRunner.Execute(ctx, "mise", "which", name); err == nil {
		return true
	}

	// Try common binary name mappings for packages where package name != binary name
	binaryMappings := map[string][]string{
		"neovim":     {"nvim"},
		"maven":      {"mvn"},
		"lazygit":    {"lazygit"},
		"lazydocker": {"lazydocker"},
		"starship":   {"starship"},
		"ripgrep":    {"rg"},
		"fd-find":    {"fd"},
		"bat":        {"bat"},
		"eza":        {"eza"},
		"zoxide":     {"zoxide"},
		"delta":      {"delta"},
		"hyperfine":  {"hyperfine"},
		"bottom":     {"btm"},
		"fzf":        {"fzf"},
		"yq":         {"yq"}, // yq might not be active but let's try
	}

	if binaries, exists := binaryMappings[name]; exists {
		for _, binary := range binaries {
			if err := p.commandRunner.Execute(ctx, "mise", "which", binary); err == nil {
				return true
			}
		}
	}

	return false
}

func (p *PackageInstaller) isAquaInstalled(ctx context.Context, name string) bool {
	// Check if aqua has the package installed with proper AQUA_ROOT_DIR
	userLocal := filepath.Dir(p.getUserBinDir())

	_ = os.Setenv("AQUA_ROOT_DIR", userLocal)

	defer func() { _ = os.Unsetenv("AQUA_ROOT_DIR") }()

	err := p.commandRunner.Execute(ctx, "aqua", "which", name)

	return err == nil
}

// Helper methods for configuration

func (p *PackageInstaller) setupMiseConfig() error {
	configDir := p.getXDGConfigHome() + "/mise"

	// Ensure config directory exists
	if err := p.fileManager.EnsureDir(configDir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Copy main config if template exists and config doesn't
	sourcePath := "configs/mise.toml"
	destPath := configDir + "/config.toml"

	if !p.fileManager.FileExists(destPath) && p.fileManager.FileExists(sourcePath) {
		data, err := p.fileManager.ReadFile(sourcePath)
		if err == nil {
			if err := p.fileManager.WriteFile(destPath, data); err != nil {
				return fmt.Errorf("failed to write mise config: %w", err)
			}
		}
	}

	return nil
}

func (p *PackageInstaller) createAquaConfig(configPath string) error {
	// Ensure directory exists
	if err := p.fileManager.EnsureDir(filepath.Dir(configPath)); err != nil {
		return err
	}

	aquaTemplate := `# Karei Aqua Configuration
# Declarative CLI Version Manager
registries:
  - type: standard
    ref: v4.155.1

packages:
  # Packages will be added here automatically
`

	return p.fileManager.WriteFile(configPath, []byte(aquaTemplate))
}

func (p *PackageInstaller) addPackageToAquaConfig(configPath, aquaPackage string) error {
	// Read existing config
	content, err := p.fileManager.ReadFile(configPath)
	if err != nil {
		return err
	}

	configStr := string(content)

	// Check if package already exists
	packageLine := "  - name: " + aquaPackage
	if strings.Contains(configStr, packageLine) {
		return nil // Already present
	}

	// Add package to the end of packages section
	if strings.Contains(configStr, "packages:") {
		configStr = strings.Replace(configStr,
			"  # Packages will be added here automatically",
			packageLine+"\n  # Packages will be added here automatically",
			1)
	}

	return p.fileManager.WriteFile(configPath, []byte(configStr))
}

// Helper methods for system information

func (p *PackageInstaller) getUserBinDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to environment variable
		home = os.Getenv("HOME")
		if home == "" {
			home = "/home/" + os.Getenv("USER")
		}
	}

	return filepath.Join(home, ".local", "bin")
}

func (p *PackageInstaller) getXDGConfigHome() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to environment variable
		home = os.Getenv("HOME")
		if home == "" {
			home = "/home/" + os.Getenv("USER")
		}
	}

	return filepath.Join(home, ".config")
}

// downloadFile downloads a file from URL to local path.
func (p *PackageInstaller) downloadFile(ctx context.Context, url, destPath string) error {
	if p.verbose && !p.tuiMode {
		fmt.Printf("• Downloading from %s...\n", url)
	}

	// Create HTTP client with proxy support - let context handle cancellation
	client := network.GetHTTPClient()

	// Create request with context (context handles timeout and cancellation)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "karei/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}

	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w with status %d: %s", ErrDownloadFailed, resp.StatusCode, resp.Status)
	}

	// Create destination file
	outFile, err := os.Create(destPath) //nolint:gosec // G304: destPath is validated and controlled by the application
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}

	defer func() { _ = outFile.Close() }()

	// Copy response body to file with context cancellation support
	// Use io.CopyBuffer with a smaller buffer to allow context cancellation checks
	buffer := make([]byte, 32*1024) // 32KB buffer for responsive cancellation

	_, err = io.CopyBuffer(outFile, resp.Body, buffer)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", destPath, err)
	}

	if p.verbose && !p.tuiMode {
		fmt.Printf("✓ Download completed successfully\n")
	}

	return nil
}

// installPMDFromGitHub handles PMD installation using dynamic version detection.
func (p *PackageInstaller) installPMDFromGitHub(ctx context.Context, _ *domain.Package) error {
	if !p.tuiMode {
		fmt.Printf("Installing PMD from GitHub releases...\n")
	}

	// Add a reasonable timeout for PMD installation (5 minutes max)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Get the latest PMD download URL dynamically
	downloadURL := p.getPMDLatestURL(ctx)
	if downloadURL == "" {
		return ErrPMDURLNotFound
	}

	if !p.tuiMode {
		fmt.Printf("• Downloading PMD from: %s\n", downloadURL)
	}

	// Download PMD ZIP file
	tempFile := filepath.Join(os.TempDir(), "pmd-bin.zip")

	if !p.tuiMode {
		fmt.Printf("• Starting download to: %s\n", tempFile)
	}

	if err := p.downloadFile(ctx, downloadURL, tempFile); err != nil {
		return fmt.Errorf("failed to download PMD from %s: %w", downloadURL, err)
	}

	if !p.tuiMode {
		fmt.Printf("• Download completed, file size: %d bytes\n", getFileSize(tempFile))
	}

	// Extract PMD to user's local directory (not just bin)
	userLocal := filepath.Dir(p.getUserBinDir()) // ~/.local
	if err := p.fileManager.EnsureDir(userLocal); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	return p.extractPMDZIP(ctx, tempFile, userLocal)
}

// getFileSize returns file size or 0 if file doesn't exist.
func getFileSize(filePath string) int64 {
	if info, err := os.Stat(filePath); err == nil {
		return info.Size()
	}

	return 0
}

// getPMDLatestURL attempts to get the latest PMD download URL, returns empty string if failed.
func (p *PackageInstaller) getPMDLatestURL(ctx context.Context) string {
	// Try using curl command which we know works
	output, err := p.commandRunner.ExecuteWithOutput(ctx, "curl", "-s", "https://api.github.com/repos/pmd/pmd/releases/latest")
	if err != nil {
		return ""
	}

	// Look for the bin.zip URL in the response
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "browser_download_url") &&
			strings.Contains(trimmed, "pmd-dist-") &&
			strings.Contains(trimmed, "-bin.zip") &&
			!strings.Contains(trimmed, ".asc") {
			// Extract URL: "browser_download_url": "URL"
			urlStart := strings.Index(trimmed, "https://")
			if urlStart == -1 {
				continue
			}

			urlEnd := strings.Index(trimmed[urlStart:], "\"")
			if urlEnd == -1 {
				continue
			}

			return trimmed[urlStart : urlStart+urlEnd]
		}
	}

	return ""
}

// Helper functions for GitHub installation methods

// downloadDirectBinary downloads a binary from a direct URL.
func (p *PackageInstaller) downloadDirectBinary(ctx context.Context, pkg *domain.Package) error {
	tempFile := filepath.Join(os.TempDir(), pkg.Name)
	if err := p.downloadFile(ctx, pkg.Source, tempFile); err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}

	// Make executable and move to bin directory
	if err := os.Chmod(tempFile, 0755); err != nil { //nolint:gosec // G302: Executable files need 0755 permissions
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	binDir := p.getUserBinDir()
	if err := p.fileManager.EnsureDir(binDir); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	targetPath := filepath.Join(binDir, pkg.Name)

	return os.Rename(tempFile, targetPath)
}

// downloadGitHubBinary downloads a binary from GitHub releases using common patterns.
func (p *PackageInstaller) downloadGitHubBinary(_ context.Context, pkg *domain.Package) error {
	return fmt.Errorf("%w for %s", ErrGitHubBinaryNotImpl, pkg.Name)
}

// downloadGitHubRelease downloads a release file trying multiple extensions.
func (p *PackageInstaller) downloadGitHubRelease(_ context.Context, pkg *domain.Package, _ []string) (string, error) {
	return "", fmt.Errorf("%w for %s", ErrGitHubReleaseNotImpl, pkg.Name)
}

// extractBundle extracts an archive preserving directory structure.
func (p *PackageInstaller) extractBundle(_, _ string) error {
	return ErrExtractBundleNotImpl
}

// symlinkBundleExecutables creates symlinks for executables in a bundle.
func (p *PackageInstaller) symlinkBundleExecutables(_, _ string) error {
	return ErrSymlinkBundleNotImpl
}

// installGenericJavaApp installs Java applications using generic patterns.
func (p *PackageInstaller) installGenericJavaApp(_ context.Context, pkg *domain.Package) error {
	return fmt.Errorf("%w for %s", ErrGenericJavaAppNotImpl, pkg.Name)
}

// extractPMDZIP extracts PMD with proper directory structure to ~/.local/share/pmd.
func (p *PackageInstaller) extractPMDZIP(ctx context.Context, zipPath, targetDir string) error {
	if !p.tuiMode {
		fmt.Printf("• Extracting PMD to %s...\n", targetDir)
	}

	pmdDir, err := p.setupPMDDirectory(targetDir)
	if err != nil {
		return err
	}

	if err := p.extractPMDFiles(ctx, zipPath, pmdDir); err != nil {
		return err
	}

	return p.createPMDSymlink(pmdDir)
}

func (p *PackageInstaller) setupPMDDirectory(targetDir string) (string, error) {
	// Create PMD directory in ~/.local/share/pmd/
	pmdDir := filepath.Join(targetDir, "share", "pmd")
	if err := p.fileManager.EnsureDir(pmdDir); err != nil {
		return "", fmt.Errorf("failed to create PMD directory: %w", err)
	}

	return pmdDir, nil
}

func (p *PackageInstaller) extractPMDFiles(ctx context.Context, zipPath, pmdDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open PMD ZIP file: %w", err)
	}

	defer func() { _ = reader.Close() }()

	// Extract all files, stripping the top-level directory (pmd-bin-X.Y.Z)
	for _, file := range reader.File {
		// Check for context cancellation during extraction
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := p.processPMDFile(file, pmdDir); err != nil {
			return err
		}
	}

	return nil
}

func (p *PackageInstaller) processPMDFile(file *zip.File, pmdDir string) error {
	if file.FileInfo().IsDir() {
		return nil
	}

	// Strip the top-level directory from the path
	// pmd-bin-7.16.0/bin/pmd -> bin/pmd
	parts := strings.Split(file.Name, "/")
	if len(parts) < 2 {
		return nil // Skip files in root
	}

	relativePath := strings.Join(parts[1:], "/")
	targetPath := filepath.Join(pmdDir, relativePath)

	// Ensure target directory exists
	targetDirPath := filepath.Dir(targetPath)
	if err := p.fileManager.EnsureDir(targetDirPath); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDirPath, err)
	}

	// Extract the file
	if err := p.extractPMDFile(file, targetPath); err != nil {
		return fmt.Errorf("failed to extract %s: %w", file.Name, err)
	}

	return nil
}

func (p *PackageInstaller) createPMDSymlink(pmdDir string) error {
	// Create symlink in ~/.local/bin/pmd -> ~/.local/share/pmd/bin/pmd
	binDir := p.getUserBinDir()
	if err := p.fileManager.EnsureDir(binDir); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	symlinkSource := filepath.Join(pmdDir, "bin", "pmd")
	symlinkTarget := filepath.Join(binDir, "pmd")

	// Remove existing symlink if it exists
	if p.fileManager.FileExists(symlinkTarget) {
		if err := os.Remove(symlinkTarget); err != nil {
			return fmt.Errorf("failed to remove existing PMD symlink: %w", err)
		}
	}

	if err := os.Symlink(symlinkSource, symlinkTarget); err != nil {
		return fmt.Errorf("failed to create PMD symlink: %w", err)
	}

	if !p.tuiMode {
		fmt.Printf("✓ PMD extracted successfully\n")
		fmt.Printf("✓ Created symlink: %s -> %s\n", symlinkTarget, symlinkSource)
	}

	return nil
}

// extractPMDFile extracts a single file from PMD ZIP archive.
func (p *PackageInstaller) extractPMDFile(file *zip.File, targetPath string) error {
	// Open file in ZIP
	reader, err := file.Open()
	if err != nil {
		return err
	}

	defer func() { _ = reader.Close() }()

	// Create target file
	outFile, err := os.Create(targetPath) //nolint:gosec // G304: targetPath is validated and controlled by the application
	if err != nil {
		return err
	}

	defer func() { _ = outFile.Close() }()

	// Copy content
	_, err = io.Copy(outFile, reader) //nolint:gosec // G110: ZIP extraction from trusted sources (official releases)
	if err != nil {
		return err
	}

	// Preserve permissions - make scripts executable
	if strings.Contains(targetPath, "/bin/") && !strings.HasSuffix(targetPath, ".bat") {
		return os.Chmod(targetPath, 0755) //nolint:gosec // G302: Executable files need 0755 permissions
	}

	return os.Chmod(targetPath, 0644) //nolint:gosec // G302: Regular files need 0644 permissions for proper operation
}

// Removal methods

// removeGitHub removes GitHub-installed packages (all subcategories).
//
//nolint:unparam // Intentionally returns nil - uninstall operations should be forgiving
func (p *PackageInstaller) removeGitHub(_ context.Context, pkg *domain.Package) error {
	if p.dryRun {
		fmt.Printf("DRY RUN: remove GitHub package %s\n", pkg.Name)

		return nil
	}

	fmt.Printf("Removing %s...\n", pkg.Name)

	// Remove binary from ~/.local/bin/
	binPath := filepath.Join(p.getUserBinDir(), pkg.Name)
	if p.fileManager.FileExists(binPath) {
		if err := os.Remove(binPath); err != nil {
			fmt.Printf("⚠ Failed to remove binary %s: %v\n", binPath, err)
		} else {
			fmt.Printf("✓ Removed binary: %s\n", binPath)
		}
	}

	// For bundles and Java apps, also remove from ~/.local/share/
	sharePath := filepath.Join(filepath.Dir(p.getUserBinDir()), "share", pkg.Name)
	if p.fileManager.FileExists(sharePath) {
		if err := os.RemoveAll(sharePath); err != nil {
			fmt.Printf("⚠ Failed to remove application directory %s: %v\n", sharePath, err)
		} else {
			fmt.Printf("✓ Removed application directory: %s\n", sharePath)
		}
	}

	fmt.Printf("✓ %s removed successfully\n", pkg.Name)

	return nil
}

// removeGeneric removes generically installed packages (binary, script, etc.).
func (p *PackageInstaller) removeGeneric(_ context.Context, pkg *domain.Package) error {
	if p.dryRun {
		fmt.Printf("DRY RUN: remove %s package %s\n", pkg.Method, pkg.Name)

		return nil
	}

	fmt.Printf("Removing %s...\n", pkg.Name)

	// For most generic installations, just remove the binary
	binPath := filepath.Join(p.getUserBinDir(), pkg.Name)
	if p.fileManager.FileExists(binPath) {
		if err := os.Remove(binPath); err != nil {
			return fmt.Errorf("failed to remove binary %s: %w", binPath, err)
		}

		fmt.Printf("✓ Removed binary: %s\n", binPath)
	} else {
		fmt.Printf("⚠ Binary %s not found (may already be removed)\n", binPath)
	}

	fmt.Printf("✓ %s removed successfully\n", pkg.Name)

	return nil
}
