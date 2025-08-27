// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package fonts provides font management functionality.
package fonts

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/janderssonse/karei/internal/platform"
)

var (
	// ErrUnknownFont is returned when the requested font is not found.
	ErrUnknownFont = errors.New("unknown font")
	// ErrBadStatus is returned when HTTP request fails.
	ErrBadStatus = errors.New("bad HTTP status")
	// ErrInvalidFontSize is returned when font size is outside valid range.
	ErrInvalidFontSize = errors.New("font size must be between 6 and 24")
	// ErrMaxFontSize is returned when trying to increase beyond maximum.
	ErrMaxFontSize = errors.New("already at maximum font size")
	// ErrMinFontSize is returned when trying to decrease beyond minimum.
	ErrMinFontSize = errors.New("already at minimum font size")
	// ErrInvalidSizeFormat is returned when font size string is not a valid number.
	ErrInvalidSizeFormat = errors.New("invalid font size format")
)

// FontConfig represents a font configuration.
type FontConfig struct {
	Name     string
	URL      string
	FileType string
	FullName string
}

func getFonts() map[string]FontConfig {
	return map[string]FontConfig{
		"CaskaydiaMono": {
			Name:     "CaskaydiaMono",
			URL:      "https://github.com/ryanoasis/nerd-fonts/releases/latest/download/CascadiaMono.zip",
			FileType: "ttf",
			FullName: "CaskaydiaMono Nerd Font",
		},
		"FiraMono": {
			Name:     "FiraMono",
			URL:      "https://github.com/ryanoasis/nerd-fonts/releases/latest/download/FiraMono.zip",
			FileType: "otf",
			FullName: "FiraMono Nerd Font",
		},
		"JetBrainsMono": {
			Name:     "JetBrainsMono",
			URL:      "https://github.com/ryanoasis/nerd-fonts/releases/latest/download/JetBrainsMono.zip",
			FileType: "ttf",
			FullName: "JetBrainsMono Nerd Font",
		},
		"MesloLGS": {
			Name:     "MesloLGS",
			URL:      "https://github.com/ryanoasis/nerd-fonts/releases/latest/download/Meslo.zip",
			FileType: "ttf",
			FullName: "MesloLGS Nerd Font",
		},
		"BerkeleyMono": {
			Name:     "BerkeleyMono",
			URL:      "",
			FileType: "ttf",
			FullName: "Berkeley Mono",
		},
	}
}

// DownloadAndInstallFont downloads and installs a font.
func DownloadAndInstallFont(ctx context.Context, fontName string) error {
	return DownloadAndInstallFontWithOptions(ctx, fontName, false)
}

// DownloadAndInstallFontWithOptions downloads and installs a font with options.
func DownloadAndInstallFontWithOptions(ctx context.Context, fontName string, dryRun bool) error {
	font, exists := getFonts()[fontName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownFont, fontName)
	}

	// Berkeley Mono is local only
	if font.URL == "" {
		return nil
	}

	// In dry run mode, just return success
	if dryRun {
		return nil
	}

	// Check if font already installed
	if isFontInstalled(ctx, font.FullName) {
		return nil
	}

	fontsDir := filepath.Join(os.Getenv("HOME"), ".local/share/fonts")
	if err := os.MkdirAll(fontsDir, 0755); err != nil { //nolint:gosec
		return err
	}

	tmpDir := os.TempDir()
	zipPath := filepath.Join(tmpDir, fontName+".zip")
	extractDir := filepath.Join(tmpDir, fontName)

	// Download font from GitHub
	fmt.Printf("• Downloading %s font from %s\n", fontName, font.URL)

	if err := downloadFontFile(ctx, font.URL, zipPath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("✓ Downloaded %s font successfully\n", fontName)

	defer func() { _ = os.Remove(zipPath) }()

	// Extract
	if err := extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	defer func() { _ = os.RemoveAll(extractDir) }()

	// Copy font files
	if err := copyFontFiles(extractDir, fontsDir, font.FileType); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	// Refresh font cache
	return exec.CommandContext(ctx, "fc-cache").Run()
}

// ApplySystemFont applies a font to system settings.
func ApplySystemFont(ctx context.Context, fontName string) error {
	return ApplySystemFontWithOptions(ctx, fontName, false)
}

// ApplySystemFontWithOptions applies a font to system settings with options.
func ApplySystemFontWithOptions(ctx context.Context, fontName string, dryRun bool) error {
	font, exists := getFonts()[fontName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownFont, fontName)
	}

	// In dry run mode, just return success
	if dryRun {
		return nil
	}

	// Set GNOME monospace font
	cmd := exec.CommandContext(ctx, "gsettings", "set", "org.gnome.desktop.interface", "monospace-font-name", font.FullName+" 10") //nolint:gosec
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set GNOME font: %w", err)
	}

	// Set VSCode font
	settingsPath := filepath.Join(os.Getenv("HOME"), ".config/Code/User/settings.json")
	if _, err := os.Stat(settingsPath); err == nil {
		// Read, modify, write settings.json - simplified approach
		cmd := exec.CommandContext(ctx, "sed", "-i", //nolint:gosec
			fmt.Sprintf(`s/"editor.fontFamily": ".*"/"editor.fontFamily": "%s"/g`, font.FullName),
			settingsPath)
		_ = cmd.Run() // Ignore errors
	}

	return nil
}

func isFontInstalled(ctx context.Context, fontName string) bool {
	cmd := exec.CommandContext(ctx, "fc-list")

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), fontName)
}

func downloadFontFile(ctx context.Context, url, filepath string) error {
	fmt.Printf("↓ Connecting to %s...\n", url)

	// Use proxy-aware HTTP client
	client := platform.GetHTTPClient()
	client.Timeout = 5 * time.Minute

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s", ErrBadStatus, resp.Status)
	}

	out, err := os.Create(filepath) //nolint:gosec
	if err != nil {
		return err
	}

	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)

	return err
}

func extractZip(src, dest string) error {
	zipReader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}

	defer func() { _ = zipReader.Close() }()

	if err := os.MkdirAll(dest, 0755); err != nil { //nolint:gosec
		return err
	}

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		reader, err := file.Open()
		if err != nil {
			return err
		}

		path := filepath.Join(dest, file.Name)                        //nolint:gosec
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { //nolint:gosec
			_ = reader.Close()

			return err
		}

		outFile, err := os.Create(path) //nolint:gosec
		if err != nil {
			_ = reader.Close()

			return err
		}

		_, err = io.Copy(outFile, reader) //nolint:gosec
		_ = outFile.Close()
		_ = reader.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func copyFontFiles(srcDir, dstDir, fileType string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), "."+fileType) {
			dst := filepath.Join(dstDir, info.Name())

			return platform.CopyFile(path, dst)
		}

		return nil
	})
}

// SizeManager handles font size operations.
type SizeManager struct {
	verbose bool
	homeDir string
}

// NewSizeManager creates a new font size manager.
func NewSizeManager(verbose bool) *SizeManager {
	return &SizeManager{verbose: verbose, homeDir: ""}
}

// NewSizeManagerWithHome creates a new font size manager with custom home directory for testing.
func NewSizeManagerWithHome(verbose bool, homeDir string) *SizeManager {
	return &SizeManager{verbose: verbose, homeDir: homeDir}
}

// GetCurrentSize returns the current font size.
func (m *SizeManager) GetCurrentSize() (int, error) {
	configPath := filepath.Join(m.getHomeDir(), ".config", "ghostty", "font-size.conf")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return 10, nil // Default size
	}

	content, err := os.ReadFile(configPath) //nolint:gosec
	if err != nil {
		return 10, err
	}

	// Parse font-size = 12 format
	re := regexp.MustCompile(`font-size\s*=\s*(\d+)`)
	matches := re.FindStringSubmatch(string(content))

	if len(matches) > 1 {
		size, err := strconv.Atoi(matches[1])
		if err != nil {
			return 10, err
		}

		return size, nil
	}

	return 10, nil
}

// SetFontSize sets the font size to the specified value.
func (m *SizeManager) SetFontSize(size int) error {
	if size < 6 || size > 24 {
		return ErrInvalidFontSize
	}

	configDir := filepath.Join(m.getHomeDir(), ".config", "ghostty")
	if err := os.MkdirAll(configDir, 0755); err != nil { //nolint:gosec
		return err
	}

	configPath := filepath.Join(configDir, "font-size.conf")
	content := fmt.Sprintf("font-size = %d\n", size)

	if m.verbose {
		fmt.Printf("Setting font size to %d\n", size)
	}

	return os.WriteFile(configPath, []byte(content), 0644) //nolint:gosec
}

// GetAvailableSizes returns all supported font sizes.
func (m *SizeManager) GetAvailableSizes() []int {
	return []int{7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 18, 20, 22, 24}
}

// IncreaseFontSize increases the font size to the next available size.
func (m *SizeManager) IncreaseFontSize() error {
	current, err := m.GetCurrentSize()
	if err != nil {
		return err
	}

	sizes := m.GetAvailableSizes()
	for _, size := range sizes {
		if size > current {
			return m.SetFontSize(size)
		}
	}

	return ErrMaxFontSize
}

// DecreaseFontSize decreases the font size to the next smaller available size.
func (m *SizeManager) DecreaseFontSize() error {
	current, err := m.GetCurrentSize()
	if err != nil {
		return err
	}

	sizes := m.GetAvailableSizes()
	for i := len(sizes) - 1; i >= 0; i-- {
		if sizes[i] < current {
			return m.SetFontSize(sizes[i])
		}
	}

	return ErrMinFontSize
}

// UpdateAlacrittyConfig updates the Alacritty terminal font size configuration.
func (m *SizeManager) UpdateAlacrittyConfig(size int) error {
	alacrittyPath := filepath.Join(m.getHomeDir(), ".config", "alacritty", "font-size.toml")

	if _, err := os.Stat(alacrittyPath); os.IsNotExist(err) {
		// Create if doesn't exist
		content := fmt.Sprintf("size = %d\n", size)

		return os.WriteFile(alacrittyPath, []byte(content), 0644) //nolint:gosec
	}

	// Read and update existing
	content, err := os.ReadFile(alacrittyPath) //nolint:gosec
	if err != nil {
		return err
	}

	// Replace size line
	re := regexp.MustCompile(`size\s*=\s*\d+`)
	newContent := re.ReplaceAllString(string(content), fmt.Sprintf("size = %d", size))

	return os.WriteFile(alacrittyPath, []byte(newContent), 0644) //nolint:gosec
}

// SetFontSizeForAllTerminals sets the font size for all supported terminal applications.
func (m *SizeManager) SetFontSizeForAllTerminals(size int) error {
	// Set for Ghostty (primary)
	if err := m.SetFontSize(size); err != nil {
		return err
	}

	// Also set for Alacritty if config exists
	if err := m.UpdateAlacrittyConfig(size); err != nil && m.verbose {
		fmt.Printf("Warning: Could not update Alacritty config: %v\n", err)
	}

	return nil
}

// GetFontSizeDisplay returns a formatted string showing the current font size.
func (m *SizeManager) GetFontSizeDisplay() string {
	current, err := m.GetCurrentSize()
	if err != nil {
		return "unknown"
	}

	sizes := m.GetAvailableSizes()
	display := make([]string, len(sizes))

	for i, size := range sizes {
		if size == current {
			display[i] = fmt.Sprintf("▶ %d", size)
		} else {
			display[i] = fmt.Sprintf("  %d", size)
		}
	}

	return strings.Join(display, "\n")
}

func (m *SizeManager) getHomeDir() string {
	if m.homeDir != "" {
		return m.homeDir
	}

	return os.Getenv("HOME")
}

// SetFontSize is a convenience function for setting font size.
func SetFontSize(_ context.Context, size string, verbose bool) error {
	sizeInt, err := strconv.Atoi(size)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidSizeFormat, size)
	}

	manager := NewSizeManager(verbose)

	return manager.SetFontSize(sizeInt)
}
