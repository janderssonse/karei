// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package mocks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janderssonse/karei/internal/domain"
)

var (
	// ErrPackageNotFound indicates a package was not found.
	ErrPackageNotFound = errors.New("package not found")
	// ErrPackageNotInstalled indicates a package is not installed.
	ErrPackageNotInstalled = errors.New("package is not installed")
	// ErrInvalidResponseType indicates an invalid response type was received.
	ErrInvalidResponseType = errors.New("response is not a map[string]any")
)

// InstallRecord tracks installation history.
type InstallRecord struct {
	Package   string
	Method    domain.InstallMethod
	Version   string
	Timestamp time.Time
	Success   bool
	Error     error
}

// PackageInfo represents package metadata.
type PackageInfo struct {
	Name         string
	Version      string
	Available    bool
	Description  string
	Size         int64
	Dependencies []string
}

// FakePackageManager simulates package managers without system interaction.
type FakePackageManager struct {
	installedPackages map[string]PackageInfo
	availablePackages map[string]PackageInfo
	installHistory    []InstallRecord
	tempDir           string
	verbose           bool
	shouldFail        map[string]error // Packages that should fail to install
}

// NewFakePackageManager creates a fake package manager for testing.
func NewFakePackageManager(tempDir string, verbose bool) *FakePackageManager {
	return &FakePackageManager{
		installedPackages: make(map[string]PackageInfo),
		availablePackages: make(map[string]PackageInfo),
		installHistory:    []InstallRecord{},
		tempDir:           tempDir,
		verbose:           verbose,
		shouldFail:        make(map[string]error),
	}
}

// AddAvailablePackage adds a package to the available packages list.
func (fpm *FakePackageManager) AddAvailablePackage(name string, info PackageInfo) {
	info.Name = name
	fpm.availablePackages[name] = info
}

// SetPackageFailure makes a package fail installation with given error.
func (fpm *FakePackageManager) SetPackageFailure(packageName string, err error) {
	fpm.shouldFail[packageName] = err
}

// Install simulates package installation.
func (fpm *FakePackageManager) Install(packageName string, method domain.InstallMethod) error {
	record := InstallRecord{
		Package:   packageName,
		Method:    method,
		Timestamp: time.Now(),
	}

	// Check if package should fail
	if err, shouldFail := fpm.shouldFail[packageName]; shouldFail {
		record.Success = false
		record.Error = err
		fpm.installHistory = append(fpm.installHistory, record)

		return err
	}

	// Check if package is available
	packageInfo, available := fpm.availablePackages[packageName]
	if !available {
		err := fmt.Errorf("%w: %s", ErrPackageNotFound, packageName)
		record.Success = false
		record.Error = err
		fpm.installHistory = append(fpm.installHistory, record)

		return err
	}

	if fpm.verbose {
		fmt.Printf("Installing %s version %s\n", packageName, packageInfo.Version)
	}

	// Simulate installation by creating fake binary and metadata
	err := fpm.createFakeInstallation(packageName, packageInfo)
	if err != nil {
		record.Success = false
		record.Error = err
		fpm.installHistory = append(fpm.installHistory, record)

		return err
	}

	// Mark as installed
	fpm.installedPackages[packageName] = packageInfo
	record.Success = true
	record.Version = packageInfo.Version
	fpm.installHistory = append(fpm.installHistory, record)

	if fpm.verbose {
		fmt.Printf("Successfully installed %s\n", packageName)
	}

	return nil
}

// IsInstalled checks if a package is installed by checking fake installation files.
func (fpm *FakePackageManager) IsInstalled(packageName string) bool {
	_, installed := fpm.installedPackages[packageName]

	return installed
}

// GetInstalledPackages returns list of installed packages.
func (fpm *FakePackageManager) GetInstalledPackages() map[string]PackageInfo {
	result := make(map[string]PackageInfo)
	for k, v := range fpm.installedPackages {
		result[k] = v
	}

	return result
}

// GetInstallHistory returns installation history.
func (fpm *FakePackageManager) GetInstallHistory() []InstallRecord {
	return fpm.installHistory
}

// Uninstall simulates package removal.
func (fpm *FakePackageManager) Uninstall(packageName string) error {
	if !fpm.IsInstalled(packageName) {
		return fmt.Errorf("%w: %s", ErrPackageNotInstalled, packageName)
	}

	// Remove fake binary
	binaryPath := filepath.Join(fpm.tempDir, "usr", "local", "bin", packageName)
	_ = os.Remove(binaryPath)

	// Remove fake config
	configPath := filepath.Join(fpm.tempDir, "etc", packageName, packageName+".conf")
	_ = os.Remove(configPath)
	configDir := filepath.Join(fpm.tempDir, "etc", packageName)
	_ = os.Remove(configDir)

	// Remove from installed packages
	delete(fpm.installedPackages, packageName)

	if fpm.verbose {
		fmt.Printf("Uninstalled %s\n", packageName)
	}

	return nil
}

// MockCommandExecutor simulates command execution without running real commands.
type MockCommandExecutor struct {
	commands   []ExecutedCommand
	responses  map[string]CommandResponse
	shouldFail map[string]error
	verbose    bool
}

// ExecutedCommand represents a command that was executed.
type ExecutedCommand struct {
	Name string
	Args []string
}

// CommandResponse represents the response to a command.
type CommandResponse struct {
	Output   string
	ExitCode int
	Error    error
}

// String returns string representation of command.
func (ec ExecutedCommand) String() string {
	return ec.Name + " " + strings.Join(ec.Args, " ")
}

// NewMockCommandExecutor creates a new mock command executor.
func NewMockCommandExecutor(verbose bool) *MockCommandExecutor {
	return &MockCommandExecutor{
		commands:   []ExecutedCommand{},
		responses:  make(map[string]CommandResponse),
		shouldFail: make(map[string]error),
		verbose:    verbose,
	}
}

// SetCommandResponse sets expected response for a command.
func (mce *MockCommandExecutor) SetCommandResponse(cmdStr string, response CommandResponse) {
	mce.responses[cmdStr] = response
}

// SetCommandFailure makes a command fail with given error.
func (mce *MockCommandExecutor) SetCommandFailure(cmdStr string, err error) {
	mce.shouldFail[cmdStr] = err
}

// Execute simulates command execution.
func (mce *MockCommandExecutor) Execute(name string, args ...string) error {
	cmd := ExecutedCommand{Name: name, Args: args}
	mce.commands = append(mce.commands, cmd)

	cmdStr := cmd.String()

	if mce.verbose {
		fmt.Printf("MOCK EXEC: %s\n", cmdStr)
	}

	// Check if command should fail
	if err, shouldFail := mce.shouldFail[cmdStr]; shouldFail {
		if mce.verbose {
			fmt.Printf("MOCK FAIL: %s - %v\n", cmdStr, err)
		}

		return err
	}

	// Return predefined response if available
	if response, exists := mce.responses[cmdStr]; exists {
		if mce.verbose && response.Output != "" {
			fmt.Printf("MOCK OUTPUT: %s\n", response.Output)
		}

		return response.Error
	}

	// Default success response
	if mce.verbose {
		fmt.Printf("MOCK SUCCESS: %s\n", cmdStr)
	}

	return nil
}

// GetExecutedCommands returns list of executed commands.
func (mce *MockCommandExecutor) GetExecutedCommands() []ExecutedCommand {
	return mce.commands
}

// WasCommandExecuted checks if a specific command was executed.
func (mce *MockCommandExecutor) WasCommandExecuted(name string, args ...string) bool {
	target := ExecutedCommand{Name: name, Args: args}
	for _, cmd := range mce.commands {
		if cmd.Name == target.Name && equalStringSlices(cmd.Args, target.Args) {
			return true
		}
	}

	return false
}

// Reset clears command history.
func (mce *MockCommandExecutor) Reset() {
	mce.commands = []ExecutedCommand{}
}

// MockGitHubClient simulates GitHub API interactions.
type MockGitHubClient struct {
	responses map[string]any
	failures  map[string]error
	verbose   bool
}

// NewMockGitHubClient creates a new mock GitHub client.
func NewMockGitHubClient(verbose bool) *MockGitHubClient {
	return &MockGitHubClient{
		responses: make(map[string]any),
		failures:  make(map[string]error),
		verbose:   verbose,
	}
}

// SetResponse sets mock response for API endpoint.
func (mgc *MockGitHubClient) SetResponse(endpoint string, response any) {
	mgc.responses[endpoint] = response
}

// SetFailure makes API call fail with given error.
func (mgc *MockGitHubClient) SetFailure(endpoint string, err error) {
	mgc.failures[endpoint] = err
}

// GetLatestRelease simulates getting latest release info.
func (mgc *MockGitHubClient) GetLatestRelease(repo string) (map[string]any, error) {
	endpoint := "repos/" + repo + "/releases/latest"

	if mgc.verbose {
		fmt.Printf("MOCK GITHUB: %s\n", endpoint)
	}

	if err, shouldFail := mgc.failures[endpoint]; shouldFail {
		return nil, err
	}

	if response, exists := mgc.responses[endpoint]; exists {
		if typedResponse, ok := response.(map[string]any); ok {
			return typedResponse, nil
		}

		return nil, fmt.Errorf("%w for %s", ErrInvalidResponseType, endpoint)
	}

	// Default mock response
	return map[string]any{
		"tag_name": "v1.0.0",
		"assets": []any{
			map[string]any{
				"name":                 extractRepoName(repo) + "_1.0.0_Linux_x86_64.tar.gz",
				"browser_download_url": "https://github.com/" + repo + "/releases/download/v1.0.0/" + extractRepoName(repo) + "_1.0.0_Linux_x86_64.tar.gz",
			},
		},
	}, nil
}

// Helper functions.
func equalStringSlices(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}

	for i, v := range first {
		if v != second[i] {
			return false
		}
	}

	return true
}

func extractRepoName(repo string) string {
	parts := strings.Split(repo, "/")
	if len(parts) >= 2 {
		return parts[1]
	}

	return repo
}

// PreloadCommonPackages adds commonly used packages to fake package manager.
func (fpm *FakePackageManager) PreloadCommonPackages() {
	commonPackages := map[string]PackageInfo{
		"vim": {
			Version:     "8.2.0",
			Available:   true,
			Description: "Vi IMproved - enhanced vi editor",
			Size:        2048000,
		},
		"btop": {
			Version:     "1.2.13",
			Available:   true,
			Description: "Resource monitor",
			Size:        1024000,
		},
		"neovim": {
			Version:     "0.9.5",
			Available:   true,
			Description: "Hyperextensible Vim-based text editor",
			Size:        4096000,
		},
		"git": {
			Version:     "2.34.1",
			Available:   true,
			Description: "Fast, scalable, distributed revision control system",
			Size:        8192000,
		},
		"curl": {
			Version:     "7.81.0",
			Available:   true,
			Description: "Command line tool for transferring data",
			Size:        512000,
		},
		"wget": {
			Version:     "1.21.2",
			Available:   true,
			Description: "Tool for retrieving files using HTTP, HTTPS and FTP",
			Size:        1024000,
		},
		"fish": {
			Version:     "3.3.1",
			Available:   true,
			Description: "Friendly interactive shell",
			Size:        2048000,
		},
		"zellij": {
			Version:     "0.39.2",
			Available:   true,
			Description: "Terminal multiplexer",
			Size:        4096000,
		},
		"lazygit": {
			Version:     "0.40.2",
			Available:   true,
			Description: "Simple terminal UI for git commands",
			Size:        8192000,
		},
		"fastfetch": {
			Version:     "2.8.10",
			Available:   true,
			Description: "System information display tool",
			Size:        1024000,
		},
	}

	for name, info := range commonPackages {
		fpm.AddAvailablePackage(name, info)
	}
}
func (fpm *FakePackageManager) createFakeInstallation(packageName string, info PackageInfo) error {
	// Create binary directory
	binDir := filepath.Join(fpm.tempDir, "usr", "local", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil { //nolint:gosec
		return err
	}

	// Create fake binary
	binaryPath := filepath.Join(binDir, packageName)
	binaryContent := fmt.Sprintf(`#!/bin/bash
# Fake %s binary for testing
echo "Fake %s version %s"
case "$1" in
    --version) echo "%s version %s" ;;
    --help) echo "Usage: %s [options]" ;;
    *) echo "Running fake %s" ;;
esac
`, packageName, packageName, info.Version, packageName, info.Version, packageName, packageName)

	if err := os.WriteFile(binaryPath, []byte(binaryContent), 0755); err != nil { //nolint:gosec
		return err
	}

	// Create config directory if needed
	configDir := filepath.Join(fpm.tempDir, "etc", packageName)
	if err := os.MkdirAll(configDir, 0755); err != nil { //nolint:gosec
		return err
	}

	// Create fake config file
	configPath := filepath.Join(configDir, packageName+".conf")

	configContent := fmt.Sprintf("# Configuration for %s\nversion=%s\n", packageName, info.Version)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil { //nolint:gosec
		return err
	}

	return nil
}

// IsInstalled checks if a package is installed.
