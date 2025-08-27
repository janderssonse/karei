// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// NewTestAppsModel creates a test apps model with mock data to avoid real system calls.
func NewTestAppsModel(styleConfig *styles.Styles, width, height int) *AppsModel {
	// Create mock categories with test data
	mockCategories := []category{
		{
			name: "development",
			apps: []app{
				{Key: "git", Name: "Git", Description: "Version control", Source: "git", Installed: false, Selected: false},
				{Key: "vscode", Name: "VS Code", Description: "Code editor", Source: "code", Installed: true, Selected: false},
				{Key: "hadolint", Name: "Hadolint", Description: "Dockerfile linter", Source: "hadolint", Installed: true, Selected: false},
				{Key: "node", Name: "Node.js", Description: "JavaScript runtime", Source: "nodejs", Installed: false, Selected: false},
				{Key: "docker", Name: "Docker", Description: "Containerization", Source: "docker", Installed: true, Selected: false},
				{Key: "python", Name: "Python", Description: "Python interpreter", Source: "python3", Installed: true, Selected: false},
				{Key: "java", Name: "Java", Description: "Java development kit", Source: "openjdk", Installed: false, Selected: false},
			},
			selected:   make(map[string]SelectionState),
			currentApp: 0,
		},
		{
			name: "browsers",
			apps: []app{
				{Key: "firefox", Name: "Firefox", Description: "Web browser", Source: "firefox", Installed: true, Selected: false},
				{Key: "chrome", Name: "Chrome", Description: "Web browser", Source: "google-chrome-stable", Installed: false, Selected: false},
				{Key: "edge", Name: "Edge", Description: "Microsoft browser", Source: "microsoft-edge", Installed: false, Selected: false},
			},
			selected:   make(map[string]SelectionState),
			currentApp: 0,
		},
	}

	model := &AppsModel{
		styles:              styleConfig,
		width:               width,
		height:              height,
		viewport:            viewport.New(width, height),
		categories:          mockCategories,
		currentCat:          0,
		quitting:            false,
		ready:               false,
		selected:            make(map[string]SelectionState),
		searchQuery:         "",
		installStatusFilter: FilterAll,
		packageTypeFilter:   FilterAll,
		sortOption:          "Name",
	}

	return model
}
