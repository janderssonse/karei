// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package desktop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janderssonse/karei/internal/stringutil"
)

func TestDesktopApps(t *testing.T) {
	t.Parallel()
	// Test that we have the expected desktop apps
	expectedApps := []string{"about", "activity", "karei"}

	for _, appName := range expectedApps {
		if _, exists := DesktopApps[appName]; !exists {
			t.Errorf("Expected desktop app %s not found", appName)
		}
	}
}

func TestCreateDesktopEntry(t *testing.T) {
	t.Parallel()

	// Set up test environment
	tmpHome := t.TempDir()

	// Test creating about desktop entry
	err := CreateDesktopEntryWithEnv("about", tmpHome, "testuser")
	if err != nil {
		t.Fatalf("CreateDesktopEntry failed: %v", err)
	}

	// Check that file was created
	desktopFile := filepath.Join(tmpHome, ".local/share/applications", "About.desktop")
	if _, err := os.Stat(desktopFile); os.IsNotExist(err) {
		t.Error("Desktop file was not created")
	}

	// Read and check content
	content, err := os.ReadFile(desktopFile) //nolint:gosec
	if err != nil {
		t.Fatalf("Failed to read desktop file: %v", err)
	}

	contentStr := string(content)
	if !stringutil.Contains(contentStr, "Name=About") {
		t.Error("Desktop file missing Name field")
	}

	if !stringutil.Contains(contentStr, "Exec=") {
		t.Error("Desktop file missing Exec field")
	}

	if !stringutil.Contains(contentStr, "[Desktop Entry]") {
		t.Error("Desktop file missing Desktop Entry header")
	}
}

func TestCreateDesktopEntryUnknown(t *testing.T) {
	t.Parallel()

	err := CreateDesktopEntry("nonexistent")
	if err == nil {
		t.Error("Expected error for unknown desktop app")
	}
}

func TestRemoveDesktopEntry(t *testing.T) {
	t.Parallel()

	// Set up test environment
	tmpHome := t.TempDir()

	// Create then remove
	err := CreateDesktopEntryWithEnv("about", tmpHome, "testuser")
	if err != nil {
		t.Fatalf("CreateDesktopEntry failed: %v", err)
	}

	err = RemoveDesktopEntryWithEnv("about", tmpHome)
	if err != nil {
		t.Fatalf("RemoveDesktopEntry failed: %v", err)
	}

	// Check that file was removed
	desktopFile := filepath.Join(tmpHome, ".local/share/applications", "About.desktop")
	if _, err := os.Stat(desktopFile); !os.IsNotExist(err) {
		t.Error("Desktop file was not removed")
	}
}

func TestCreateAllDesktopEntries(t *testing.T) {
	t.Parallel()

	// Set up test environment
	tmpHome := t.TempDir()

	err := CreateAllDesktopEntriesWithEnv(tmpHome, "testuser")
	if err != nil {
		t.Fatalf("CreateAllDesktopEntries failed: %v", err)
	}

	// Check that all files were created
	appsDir := filepath.Join(tmpHome, ".local/share/applications")
	for _, app := range DesktopApps {
		desktopFile := filepath.Join(appsDir, app.Name+".desktop")
		if _, err := os.Stat(desktopFile); os.IsNotExist(err) {
			t.Errorf("Desktop file %s was not created", app.Name)
		}
	}
}
