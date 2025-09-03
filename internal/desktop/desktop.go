// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package desktop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	// ErrUnknownDesktopApp is returned when the requested desktop app is not found.
	ErrUnknownDesktopApp = errors.New("unknown desktop app")
)

// DesktopApp represents a desktop application entry.
type DesktopApp struct { //nolint:revive
	Name          string
	Comment       string
	Exec          string
	Icon          string
	Categories    string
	Terminal      bool
	StartupNotify bool
}

// DesktopApps contains desktop application definitions.
var DesktopApps = map[string]DesktopApp{ //nolint:gochecknoglobals
	"about": {
		Name:          "About",
		Comment:       "System information from Fastfetch",
		Exec:          "ghostty --class=About --title=About -e bash -c 'fastfetch; read -n 1 -s'",
		Icon:          "/home/%s/.local/share/karei/applications/icons/Ubuntu.png",
		Categories:    "GTK;",
		Terminal:      false,
		StartupNotify: false,
	},
	"activity": {
		Name:          "Activity",
		Comment:       "System activity from btop",
		Exec:          "ghostty --class=Activity --title=Activity -e btop",
		Icon:          "/home/%s/.local/share/karei/applications/icons/Activity.png",
		Categories:    "GTK;",
		Terminal:      false,
		StartupNotify: false,
	},
	"karei": {
		Name:          "Karei",
		Comment:       "Ubuntu desktop setup system",
		Exec:          "ghostty --class=Karei --title=Karei -e karei menu",
		Icon:          "/home/%s/.local/share/karei/applications/icons/Karei.png",
		Categories:    "GTK;System;",
		Terminal:      false,
		StartupNotify: false,
	},
}

// CreateDesktopEntry creates a desktop entry file for the given application.
func CreateDesktopEntry(appName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to environment variable
		homeDir = os.Getenv("HOME")
	}

	return CreateDesktopEntryWithEnv(appName, homeDir, os.Getenv("USER"))
}

// CreateDesktopEntryWithEnv creates a desktop entry with custom environment variables for testing.
func CreateDesktopEntryWithEnv(appName, homeDir, username string) error {
	app, exists := DesktopApps[appName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownDesktopApp, appName)
	}

	if username == "" {
		username = "user"
	}

	// Create applications directory
	appsDir := filepath.Join(homeDir, ".local/share/applications")
	if err := os.MkdirAll(appsDir, 0755); err != nil { //nolint:gosec
		return err
	}

	// Generate desktop file content
	content := fmt.Sprintf(`[Desktop Entry]
Version=1.0
Name=%s
Comment=%s
Exec=%s
Terminal=%t
Type=Application
Icon=%s
Categories=%s
StartupNotify=%t
`,
		app.Name,
		app.Comment,
		app.Exec,
		app.Terminal,
		fmt.Sprintf(app.Icon, username),
		app.Categories,
		app.StartupNotify,
	)

	// Write desktop file
	desktopFile := filepath.Join(appsDir, app.Name+".desktop")

	return os.WriteFile(desktopFile, []byte(content), 0644) //nolint:gosec
}

// CreateAllDesktopEntries creates desktop entries for all defined applications.
func CreateAllDesktopEntries() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to environment variable
		homeDir = os.Getenv("HOME")
	}

	return CreateAllDesktopEntriesWithEnv(homeDir, os.Getenv("USER"))
}

// CreateAllDesktopEntriesWithEnv creates desktop entries with custom environment variables for testing.
func CreateAllDesktopEntriesWithEnv(homeDir, username string) error {
	for name := range DesktopApps {
		if err := CreateDesktopEntryWithEnv(name, homeDir, username); err != nil {
			fmt.Printf("Warning: Failed to create desktop entry for %s: %v\n", name, err)
		}
	}

	return nil
}

// RemoveDesktopEntry removes a desktop entry file for the given application.
func RemoveDesktopEntry(appName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to environment variable
		homeDir = os.Getenv("HOME")
	}

	return RemoveDesktopEntryWithEnv(appName, homeDir)
}

// RemoveDesktopEntryWithEnv removes a desktop entry with custom environment variables for testing.
func RemoveDesktopEntryWithEnv(appName, homeDir string) error {
	app, exists := DesktopApps[appName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownDesktopApp, appName)
	}

	desktopFile := filepath.Join(homeDir, ".local/share/applications", app.Name+".desktop")

	return os.Remove(desktopFile)
}
