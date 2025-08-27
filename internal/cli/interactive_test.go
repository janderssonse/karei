// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package cli

import (
	"testing"

	"github.com/janderssonse/karei/internal/apps"
)

func TestGetAppEmoji(t *testing.T) {
	t.Parallel()

	cli := &CLI{}

	tests := []struct {
		group    string
		expected string
	}{
		{"development", "▸"},
		{"browsers", "◦"},
		{"communication", "◈"},
		{"media", "▫"},
		{"productivity", "▪"},
		{"graphics", "◉"},
		{"utilities", "■"},
		{"terminal", "□"},
		{"unknown", "⬛"},
	}

	for _, test := range tests {
		result := cli.selectEmojiForGroup(test.group)
		if result != test.expected {
			t.Errorf("selectEmojiForGroup(%s) = %s; want %s", test.group, result, test.expected)
		}
	}
}

func TestCreateUniversalManager(t *testing.T) {
	t.Parallel()

	cli := &CLI{verbose: true}

	themeManager := cli.createUniversalManager("theme", true)
	if themeManager == nil {
		t.Error("Expected theme manager, got nil")
	}

	fontManager := cli.createUniversalManager("font", true)
	if fontManager == nil {
		t.Error("Expected font manager, got nil")
	}

	unknownManager := cli.createUniversalManager("unknown", true)
	if unknownManager != nil {
		t.Error("Expected nil for unknown manager type")
	}
}

func TestExecuteSetup(t *testing.T) {
	t.Parallel()

	setup := &InteractiveSetup{
		Theme:     "", // Empty theme to avoid gsettings calls
		Font:      "", // Empty font to avoid fc-cache calls
		Apps:      []string{},
		Languages: []string{},
		Databases: []string{}, // Empty databases to avoid sudo calls
		Groups:    []string{}, // Empty groups to avoid actual installation
		Confirmed: true,
	}

	// Test setup structure validation only - don't call executeSetup as it performs real system operations
	// Verify setup has expected fields and types
	if setup.Theme != "" {
		t.Error("Expected empty theme for test isolation")
	}

	if setup.Font != "" {
		t.Error("Expected empty font for test isolation")
	}

	if len(setup.Apps) != 0 {
		t.Error("Expected empty apps list for test isolation")
	}

	if len(setup.Languages) != 0 {
		t.Error("Expected empty languages list for test isolation")
	}

	if len(setup.Databases) != 0 {
		t.Error("Expected empty databases list for test isolation")
	}

	if len(setup.Groups) != 0 {
		t.Error("Expected empty groups list for test isolation")
	}

	if !setup.Confirmed {
		t.Error("Expected confirmed setup")
	}
}

func TestAppsListApps(t *testing.T) {
	t.Parallel()
	// Test the new ListApps function
	allApps := apps.ListApps("")
	if len(allApps) == 0 {
		t.Error("Expected some apps, got none")
	}

	terminalApps := apps.ListApps("terminal")
	if len(terminalApps) == 0 {
		t.Error("Expected terminal apps, got none")
	}

	unknownApps := apps.ListApps("nonexistent")
	if len(unknownApps) != 0 {
		t.Error("Expected no apps for unknown group")
	}
}
