// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package mocks provides mock implementations for testing Karei components.
package mocks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	// ErrBinaryNotExecutable indicates a binary file is not executable.
	ErrBinaryNotExecutable = errors.New("binary is not executable")
)

// BinaryBehavior defines how a fake binary should behave.
type BinaryBehavior struct {
	Name        string
	Version     string
	HelpText    string
	CustomFlags map[string]string // flag -> response
	ExitCodes   map[string]int    // args -> exit code
	Outputs     map[string]string // args -> output
}

// FakeBinaryGenerator creates realistic fake binaries for testing.
type FakeBinaryGenerator struct {
	binaryDir string
	verbose   bool
}

// NewFakeBinaryGenerator creates a new fake binary generator.
func NewFakeBinaryGenerator(binaryDir string, verbose bool) *FakeBinaryGenerator {
	return &FakeBinaryGenerator{
		binaryDir: binaryDir,
		verbose:   verbose,
	}
}

// CreateFakeBinary creates a fake binary with specified behavior.
func (fbg *FakeBinaryGenerator) CreateFakeBinary(behavior BinaryBehavior) error {
	if err := os.MkdirAll(fbg.binaryDir, 0755); err != nil { //nolint:gosec
		return err
	}

	binaryPath := filepath.Join(fbg.binaryDir, behavior.Name)

	script := fbg.generateBinaryScript(behavior)

	if err := os.WriteFile(binaryPath, []byte(script), 0755); err != nil { //nolint:gosec
		return err
	}

	if fbg.verbose {
		fmt.Printf("Created fake binary: %s\n", binaryPath)
	}

	return nil
}

// CreateCommonBinaries creates the shell script content for common fake binaries.
func (fbg *FakeBinaryGenerator) CreateCommonBinaries() error {
	commonBinaries := []BinaryBehavior{
		{
			Name:     "vim",
			Version:  "8.2.4919",
			HelpText: "Vi IMproved - enhanced vi editor",
			CustomFlags: map[string]string{
				"--version": "VIM - Vi IMproved 8.2 (2019 Dec 12, compiled Apr 08 2024 01:00:00)",
			},
			Outputs: map[string]string{
				"--version": "VIM - Vi IMproved 8.2",
			},
		},
		{
			Name:     "nvim",
			Version:  "0.9.5",
			HelpText: "Neovim - hyperextensible Vim-based text editor",
			CustomFlags: map[string]string{
				"--version": "NVIM v0.9.5",
			},
		},
		{
			Name:     "btop",
			Version:  "1.2.13",
			HelpText: "Resource monitor that shows usage and stats",
			CustomFlags: map[string]string{
				"--version": "btop version: 1.2.13",
			},
		},
		{
			Name:     "git",
			Version:  "2.34.1",
			HelpText: "Fast, scalable, distributed revision control system",
			CustomFlags: map[string]string{
				"--version": "git version 2.34.1",
			},
			Outputs: map[string]string{
				"status":           "On branch main\nnothing to commit, working tree clean",
				"log --oneline -5": "abcd123 Initial commit",
			},
		},
		{
			Name:     "curl",
			Version:  "7.81.0",
			HelpText: "Command line tool for transferring data with URL syntax",
			CustomFlags: map[string]string{
				"--version": "curl 7.81.0 (x86_64-pc-linux-gnu)",
			},
		},
		{
			Name:     "wget",
			Version:  "1.21.2",
			HelpText: "Tool for retrieving files using HTTP, HTTPS, and FTP",
			CustomFlags: map[string]string{
				"--version": "GNU Wget 1.21.2 built on linux-gnu",
			},
		},
		{
			Name:     "fish",
			Version:  "3.3.1",
			HelpText: "Fish - the friendly interactive shell",
			CustomFlags: map[string]string{
				"--version": "fish, version 3.3.1",
			},
		},
		{
			Name:     "lazygit",
			Version:  "0.40.2",
			HelpText: "Simple terminal UI for git commands",
			CustomFlags: map[string]string{
				"--version": "commit=unknown, build date=unknown, build source=unknown, version=0.40.2",
			},
		},
		{
			Name:     "zellij",
			Version:  "0.39.2",
			HelpText: "Terminal multiplexer with batteries included",
			CustomFlags: map[string]string{
				"--version": "zellij 0.39.2",
			},
		},
		{
			Name:     "fastfetch",
			Version:  "2.8.10",
			HelpText: "System information fetch tool",
			CustomFlags: map[string]string{
				"--version": "fastfetch 2.8.10",
			},
			Outputs: map[string]string{
				"": "Fake system information display",
			},
		},
		{
			Name:     "gh",
			Version:  "2.40.1",
			HelpText: "GitHub CLI - GitHub on the command line",
			CustomFlags: map[string]string{
				"--version": "gh version 2.40.1 (2023-12-13)",
			},
			Outputs: map[string]string{
				"auth status": "✓ Logged in to github.com as testuser",
			},
		},
		{
			Name:     "flatpak",
			Version:  "1.12.7",
			HelpText: "Application deployment framework for desktop apps",
			CustomFlags: map[string]string{
				"--version": "Flatpak 1.12.7",
			},
			Outputs: map[string]string{
				"list": "Name                    Application ID          Version    Branch",
			},
		},
		{
			Name:     "snap",
			Version:  "2.58",
			HelpText: "Tool to interact with snaps",
			CustomFlags: map[string]string{
				"--version": "snap 2.58",
			},
			Outputs: map[string]string{
				"list": "Name     Version   Rev   Tracking      Publisher",
			},
		},
	}

	for _, binary := range commonBinaries {
		if err := fbg.CreateFakeBinary(binary); err != nil {
			return fmt.Errorf("failed to create fake binary %s: %w", binary.Name, err)
		}
	}

	return nil
}

// CreateApplicationBinaries creates fake binaries for Karei-managed applications.
func (fbg *FakeBinaryGenerator) CreateApplicationBinaries() error {
	appBinaries := []BinaryBehavior{
		{
			Name:     "code",
			Version:  "1.85.1",
			HelpText: "Visual Studio Code - Code editing redefined",
			CustomFlags: map[string]string{
				"--version": "1.85.1\ncommit: 0ee08df0cf4527e40edc9aa28f4b5bd38bbff2b2\nElectron: 25.9.7\nElectronBuildId: 25551756",
			},
		},
		{
			Name:     "cursor",
			Version:  "0.20.2",
			HelpText: "Cursor - AI-powered code editor",
			CustomFlags: map[string]string{
				"--version": "Cursor 0.20.2",
			},
		},
		{
			Name:     "zed",
			Version:  "0.118.0",
			HelpText: "Zed - High-performance code editor",
			CustomFlags: map[string]string{
				"--version": "zed 0.118.0",
			},
		},
		{
			Name:     "google-chrome",
			Version:  "120.0.6099.129",
			HelpText: "Google Chrome web browser",
			CustomFlags: map[string]string{
				"--version": "Google Chrome 120.0.6099.129",
			},
		},
		{
			Name:     "flameshot",
			Version:  "12.1.0",
			HelpText: "Powerful screenshot tool",
			CustomFlags: map[string]string{
				"--version": "Flameshot v12.1.0",
			},
		},
		{
			Name:     "typora",
			Version:  "1.7.6",
			HelpText: "Markdown editor and reader",
			CustomFlags: map[string]string{
				"--version": "Typora version 1.7.6",
			},
		},
		{
			Name:     "xournalpp",
			Version:  "1.1.1",
			HelpText: "Handwriting notetaking software with PDF annotation",
			CustomFlags: map[string]string{
				"--version": "Xournal++ 1.1.1",
			},
		},
	}

	for _, binary := range appBinaries {
		if err := fbg.CreateFakeBinary(binary); err != nil {
			return fmt.Errorf("failed to create fake app binary %s: %w", binary.Name, err)
		}
	}

	return nil
}

// CreateDesktopEntries creates fake .desktop files for applications.
func (fbg *FakeBinaryGenerator) CreateDesktopEntries(desktopDir string) error {
	if err := os.MkdirAll(desktopDir, 0755); err != nil { //nolint:gosec
		return err
	}

	desktopEntries := map[string]string{
		"code.desktop": `[Desktop Entry]
Version=1.0
Type=Application
Name=Visual Studio Code
GenericName=Text Editor
Comment=Code Editing. Redefined.
Exec=/usr/bin/code --unity-launch %F
Icon=code
Terminal=false
MimeType=text/plain;inode/directory;
Categories=TextEditor;Development;IDE;
`,
		"google-chrome.desktop": `[Desktop Entry]
Version=1.0
Name=Google Chrome
GenericName=Web Browser
Comment=Access the Internet
Exec=/usr/bin/google-chrome-stable %U
Terminal=false
Icon=google-chrome
Type=Application
Categories=Network;WebBrowser;
MimeType=application/pdf;application/rdf+xml;application/rss+xml;application/xhtml+xml;application/xhtml_xml;application/xml;image/gif;image/jpeg;image/png;image/webp;text/html;text/xml;x-scheme-handler/ftp;x-scheme-handler/http;x-scheme-handler/https;
`,
		"nvim.desktop": `[Desktop Entry]
Name=Neovim
GenericName=Text Editor
Comment=Edit text files
TryExec=nvim
Exec=nvim %F
Terminal=true
Type=Application
Keywords=Text;editor;
Icon=nvim
Categories=Utility;TextEditor;
`,
		"flameshot.desktop": `[Desktop Entry]
Name=Flameshot
Comment=Powerful yet simple-to-use screenshot software
GenericName=Screenshot software
Exec=flameshot
Icon=flameshot
Terminal=false
Type=Application
Categories=Graphics;Photography;
`,
	}

	for filename, content := range desktopEntries {
		desktopPath := filepath.Join(desktopDir, filename)
		if err := os.WriteFile(desktopPath, []byte(content), 0644); err != nil { //nolint:gosec
			return fmt.Errorf("failed to create desktop entry %s: %w", filename, err)
		}

		if fbg.verbose {
			fmt.Printf("Created desktop entry: %s\n", desktopPath)
		}
	}

	return nil
}

// ValidateBinary checks if a fake binary works correctly.
func (fbg *FakeBinaryGenerator) ValidateBinary(binaryName string) error {
	binaryPath := filepath.Join(fbg.binaryDir, binaryName)

	// Check if binary exists and is executable
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("binary %s does not exist: %w", binaryName, err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("%w: %s", ErrBinaryNotExecutable, binaryName)
	}

	if fbg.verbose {
		fmt.Printf("Validated binary: %s (size: %d bytes)\n", binaryPath, info.Size())
	}

	return nil
}

// CleanupBinaries removes all fake binaries.
func (fbg *FakeBinaryGenerator) CleanupBinaries() error {
	if err := os.RemoveAll(fbg.binaryDir); err != nil {
		return fmt.Errorf("failed to cleanup binaries: %w", err)
	}

	if fbg.verbose {
		fmt.Printf("Cleaned up binary directory: %s\n", fbg.binaryDir)
	}

	return nil
}

// GetBinaryPath returns the full path to a fake binary.
func (fbg *FakeBinaryGenerator) GetBinaryPath(binaryName string) string {
	return filepath.Join(fbg.binaryDir, binaryName)
}

// ListCreatedBinaries returns list of created fake binaries.
func (fbg *FakeBinaryGenerator) ListCreatedBinaries() ([]string, error) {
	files, err := os.ReadDir(fbg.binaryDir)
	if err != nil {
		return nil, err
	}

	var binaries []string

	for _, file := range files {
		if !file.IsDir() && isExecutable(file) {
			binaries = append(binaries, file.Name())
		}
	}

	return binaries, nil
}

// generateBinaryScript creates the shell script content for a fake binary.
func (fbg *FakeBinaryGenerator) generateBinaryScript(behavior BinaryBehavior) string {
	script := fmt.Sprintf(`#!/bin/bash
# Fake %s binary
echo "Fake %s version %s"
`, behavior.Name, behavior.Name, behavior.Version)

	// Add standard command handling
	script += `case "$1" in
`
	for flag, output := range behavior.CustomFlags {
		script += fmt.Sprintf(`    %s) echo "%s" ;;
`, flag, output)
	}

	if behavior.HelpText != "" {
		script += fmt.Sprintf(`    --help) echo "%s" ;;
`, behavior.HelpText)
	}

	script += `    *) echo "Unknown option: $1" ;;
`
	script += `esac
`

	// Add exit behavior for special cases
	if len(behavior.ExitCodes) > 0 {
		script += `# Handle special exit codes
`
		for args, code := range behavior.ExitCodes {
			script += fmt.Sprintf(`if [ "$*" = "%s" ]; then exit %d; fi
`, args, code)
		}
	}

	return script
}

// isExecutable checks if a file is executable.
func isExecutable(file os.DirEntry) bool {
	info, err := file.Info()
	if err != nil {
		return false
	}

	return info.Mode()&0111 != 0
}
