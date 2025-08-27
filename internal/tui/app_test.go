// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package tui

import (
	"testing"

	"github.com/janderssonse/karei/internal/tui/models"
	"github.com/janderssonse/karei/internal/tui/styles"
)

func TestNewApp(t *testing.T) {
	t.Parallel()

	app := NewApp()
	if app == nil {
		t.Fatal("NewApp() returned nil")
	}

	if app.styles == nil {
		t.Error("App styles not initialized")
	}
}

func TestAppInitialization(t *testing.T) {
	t.Parallel()

	app := NewApp()
	if app == nil {
		t.Fatal("NewApp() returned nil")
	}

	if app.currentScreen != Screen(models.MenuScreen) {
		t.Errorf("Expected initial screen to be MenuScreen, got %v", app.currentScreen)
	}

	if app.contentModel == nil {
		t.Error("App contentModel not initialized")
	}
}

func TestAppModels(t *testing.T) {
	t.Parallel()

	app := NewApp()

	// Test that models cache is initialized
	if app.models == nil {
		t.Error("App models cache not initialized")
	}

	// Test that menu model is cached
	if _, exists := app.models[Screen(models.MenuScreen)]; !exists {
		t.Error("Menu model not cached during initialization")
	}
}

func TestMenuModel(t *testing.T) {
	t.Parallel()

	s := styles.New()
	menu := models.NewMenu(s)

	if menu == nil {
		t.Fatal("NewMenu() returned nil")
	}

	// Test that the menu can be initialized
	cmd := menu.Init()
	if cmd != nil {
		t.Error("Expected Init() to return nil command")
	}
}

func TestStyles(t *testing.T) {
	t.Parallel()

	testStyles := styles.New()
	if testStyles == nil {
		t.Fatal("styles.New() returned nil")
	}

	// Test that styles are initialized
	if testStyles.Primary == "" {
		t.Error("Primary color not initialized")
	}

	// Test status icons
	successIcon := testStyles.StatusIcon("success")
	if successIcon == "" {
		t.Error("StatusIcon should return non-empty string")
	}

	// Test progress bar
	progressBar := testStyles.ProgressBar(50, 100, 20)
	if progressBar == "" {
		t.Error("ProgressBar should return non-empty string")
	}

	// Test keybinding formatting
	keybinding := testStyles.Keybinding("q", "quit")
	if keybinding == "" {
		t.Error("Keybinding should return non-empty string")
	}
}
