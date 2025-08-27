// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package uninstall

import (
	"context"
	"testing"
)

func TestNewUninstaller(t *testing.T) {
	t.Parallel()

	uninstaller := NewUninstaller(true)
	if uninstaller == nil {
		t.Fatal("NewUninstaller returned nil")
	}

	if !uninstaller.verbose {
		t.Error("Expected verbose to be true")
	}

	quietUninstaller := NewUninstaller(false)
	if quietUninstaller.verbose {
		t.Error("Expected verbose to be false")
	}
}

func TestSpecialUninstalls(t *testing.T) {
	t.Parallel()
	// Test that special uninstalls are defined
	expectedSpecial := []string{"chrome", "vscode", "docker"}

	for _, app := range expectedSpecial {
		if _, exists := SpecialUninstalls[app]; !exists {
			t.Errorf("Expected special uninstall for %s not found", app)
		}
	}
}

func TestUninstallSpecialWithUnknownApp(t *testing.T) {
	t.Parallel()

	uninstaller := NewUninstaller(false)

	// Test with unknown app (should fallback to regular uninstall)
	err := uninstaller.UninstallSpecial(context.Background(), "nonexistent-app")
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestUninstallGroup(t *testing.T) {
	t.Parallel()

	uninstaller := NewUninstaller(false)

	// Test with unknown group
	err := uninstaller.UninstallGroup(context.Background(), "nonexistent-group")
	if err == nil {
		t.Error("Expected error for nonexistent group")
	}
}

func TestRunCommand(t *testing.T) {
	t.Parallel()

	uninstaller := NewUninstaller(true)

	// Test with a safe command that should work
	err := uninstaller.runCommand(context.Background(), "echo", "test")
	if err != nil {
		t.Errorf("runCommand failed with safe command: %v", err)
	}

	// Test with command that should fail
	err = uninstaller.runCommand(context.Background(), "nonexistent-command-12345")
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}
}
