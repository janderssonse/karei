// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package fonts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janderssonse/karei/internal/stringutil"
)

func TestNewSizeManager(t *testing.T) {
	t.Parallel()

	manager := NewSizeManager(true)
	if manager == nil {
		t.Fatal("NewSizeManager returned nil")
	}

	if !manager.verbose {
		t.Error("Expected verbose to be true")
	}

	quietManager := NewSizeManager(false)
	if quietManager.verbose {
		t.Error("Expected verbose to be false")
	}
}

func TestGetAvailableSizes(t *testing.T) {
	t.Parallel()

	manager := NewSizeManager(false)
	sizes := manager.GetAvailableSizes()

	if len(sizes) == 0 {
		t.Error("Expected at least one font size to be available")
	}

	// Check that sizes are in reasonable range
	for _, size := range sizes {
		if size < 6 || size > 24 {
			t.Errorf("Font size %d is outside reasonable range (6-24)", size)
		}
	}

	// Check that sizes are sorted
	for i := 1; i < len(sizes); i++ {
		if sizes[i] <= sizes[i-1] {
			t.Error("Font sizes should be in ascending order")
		}
	}
}

func TestSetFontSizeValidation(t *testing.T) {
	t.Parallel()

	// Set up test environment
	tmpHome := t.TempDir()

	manager := NewSizeManagerWithHome(false, tmpHome)

	// Test invalid sizes
	invalidSizes := []int{5, 25, 0, -1, 100}
	for _, size := range invalidSizes {
		err := manager.SetFontSize(size)
		if err == nil {
			t.Errorf("Expected error for invalid font size %d", size)
		}
	}

	// Test valid sizes
	validSizes := []int{6, 12, 16, 24}
	for _, size := range validSizes {
		err := manager.SetFontSize(size)
		if err != nil {
			t.Errorf("Unexpected error for valid font size %d: %v", size, err)
		}
	}
}

func TestGetCurrentSizeWithoutConfig(t *testing.T) {
	t.Parallel()

	// Set up test environment with no config
	tmpHome := t.TempDir()

	manager := NewSizeManagerWithHome(false, tmpHome)

	// Should return default size when no config exists
	size, err := manager.GetCurrentSize()
	if err != nil {
		t.Errorf("GetCurrentSize should not error when no config exists: %v", err)
	}

	if size != 10 {
		t.Errorf("Expected default size 10, got %d", size)
	}
}

func TestSetAndGetFontSize(t *testing.T) {
	t.Parallel()

	// Set up test environment
	tmpHome := t.TempDir()

	manager := NewSizeManagerWithHome(false, tmpHome)

	testSize := 14

	err := manager.SetFontSize(testSize)
	if err != nil {
		t.Fatalf("Failed to set font size: %v", err)
	}

	// Verify config file was created
	configPath := filepath.Join(tmpHome, ".config", "ghostty", "font-size.conf")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify we can read it back
	currentSize, err := manager.GetCurrentSize()
	if err != nil {
		t.Fatalf("Failed to get current size: %v", err)
	}

	if currentSize != testSize {
		t.Errorf("Expected size %d, got %d", testSize, currentSize)
	}
}

func TestIncreaseFontSize(t *testing.T) {
	t.Parallel()

	// Set up test environment
	tmpHome := t.TempDir()

	manager := NewSizeManagerWithHome(false, tmpHome)

	// Set initial size
	initialSize := 12
	_ = manager.SetFontSize(initialSize)

	// Increase size
	err := manager.IncreaseFontSize()
	if err != nil {
		t.Fatalf("Failed to increase font size: %v", err)
	}

	// Verify it increased
	newSize, _ := manager.GetCurrentSize()
	if newSize <= initialSize {
		t.Errorf("Font size should have increased from %d, got %d", initialSize, newSize)
	}
}

func TestDecreaseFontSize(t *testing.T) {
	t.Parallel()

	// Set up test environment
	tmpHome := t.TempDir()

	manager := NewSizeManagerWithHome(false, tmpHome)

	// Set initial size
	initialSize := 16
	_ = manager.SetFontSize(initialSize)

	// Decrease size
	err := manager.DecreaseFontSize()
	if err != nil {
		t.Fatalf("Failed to decrease font size: %v", err)
	}

	// Verify it decreased
	newSize, _ := manager.GetCurrentSize()
	if newSize >= initialSize {
		t.Errorf("Font size should have decreased from %d, got %d", initialSize, newSize)
	}
}

func TestGetFontSizeDisplay(t *testing.T) {
	t.Parallel()

	// Set up test environment
	tmpHome := t.TempDir()

	manager := NewSizeManagerWithHome(false, tmpHome)

	// Set a specific size
	testSize := 14
	_ = manager.SetFontSize(testSize)

	display := manager.GetFontSizeDisplay()
	if display == "" {
		t.Error("GetFontSizeDisplay should not return empty string")
	}

	// Should contain the current size with arrow indicator
	if !stringutil.Contains(display, "▶ 14") {
		t.Errorf("Display should indicate current size 14, got: %s", display)
	}
}

func TestSetFontSizeForAllTerminals(t *testing.T) {
	t.Parallel()

	// Set up test environment
	tmpHome := t.TempDir()

	manager := NewSizeManagerWithHome(false, tmpHome)

	testSize := 18

	err := manager.SetFontSizeForAllTerminals(testSize)
	if err != nil {
		t.Fatalf("Failed to set font size for all terminals: %v", err)
	}

	// Verify Ghostty config was created
	ghosttyPath := filepath.Join(tmpHome, ".config", "ghostty", "font-size.conf")
	if _, err := os.Stat(ghosttyPath); os.IsNotExist(err) {
		t.Error("Ghostty config file was not created")
	}

	// Verify we can read it back
	currentSize, _ := manager.GetCurrentSize()
	if currentSize != testSize {
		t.Errorf("Expected size %d, got %d", testSize, currentSize)
	}
}
