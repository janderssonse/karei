// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/janderssonse/karei/internal/domain"
)

// FontService manages system fonts using hexagonal architecture.
type FontService struct {
	fileManager   domain.FileManager
	commandRunner domain.CommandRunner
	networkClient domain.NetworkClient
	fontsDir      string
	configDir     string
}

// NewFontService creates a FontService.
func NewFontService(fm domain.FileManager, cr domain.CommandRunner, nc domain.NetworkClient, fontsDir, configDir string) *FontService {
	return &FontService{
		fileManager:   fm,
		commandRunner: cr,
		networkClient: nc,
		fontsDir:      fontsDir,
		configDir:     configDir,
	}
}

// FontConfig represents a font configuration.
type FontConfig struct {
	Name     string
	URL      string
	FileType string
	FullName string
}

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

// GetAvailableFonts returns available fonts.
func (s *FontService) GetAvailableFonts() map[string]FontConfig {
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
func (s *FontService) DownloadAndInstallFont(ctx context.Context, fontName string) error {
	font, exists := s.GetAvailableFonts()[fontName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownFont, fontName)
	}

	// Berkeley Mono is local only
	if font.URL == "" {
		return nil
	}

	// Create temp directory for download
	tempDir := filepath.Join("/tmp", fmt.Sprintf("font-%s-%d", fontName, time.Now().Unix()))
	if err := s.fileManager.EnsureDir(tempDir); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	defer func() {
		_ = s.commandRunner.Execute(ctx, "rm", "-rf", tempDir)
	}()

	// Download font
	zipPath := filepath.Join(tempDir, fontName+".zip")
	if err := s.networkClient.DownloadFile(ctx, font.URL, zipPath); err != nil {
		return fmt.Errorf("failed to download font: %w", err)
	}

	// Extract and install
	if err := s.extractAndInstallFont(ctx, zipPath, font); err != nil {
		return fmt.Errorf("failed to install font: %w", err)
	}

	// Update font cache
	return s.commandRunner.Execute(ctx, "fc-cache", "-f")
}

// ApplySystemFont applies a font system-wide.
func (s *FontService) ApplySystemFont(ctx context.Context, fontName string) error {
	font, exists := s.GetAvailableFonts()[fontName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownFont, fontName)
	}

	// Apply to GNOME Terminal
	if err := s.applyTerminalFont(ctx, font.FullName); err != nil {
		return fmt.Errorf("failed to apply terminal font: %w", err)
	}

	// Apply to text editor (gedit)
	if err := s.applyEditorFont(ctx, font.FullName); err != nil {
		// Non-fatal: editor might not be installed
		_ = err
	}

	// Apply to system monospace font
	return s.commandRunner.Execute(ctx, "gsettings", "set",
		"org.gnome.desktop.interface", "monospace-font-name",
		fmt.Sprintf("'%s 11'", font.FullName))
}

// IncreaseFontSize increases the system font size.
func (s *FontService) IncreaseFontSize(ctx context.Context) error {
	currentSize, err := s.getCurrentFontSize(ctx)
	if err != nil {
		return err
	}

	if currentSize >= 24 {
		return ErrMaxFontSize
	}

	newSize := currentSize + 1

	return s.setSystemFontSize(ctx, newSize)
}

// DecreaseFontSize decreases the system font size.
func (s *FontService) DecreaseFontSize(ctx context.Context) error {
	currentSize, err := s.getCurrentFontSize(ctx)
	if err != nil {
		return err
	}

	if currentSize <= 6 {
		return ErrMinFontSize
	}

	newSize := currentSize - 1

	return s.setSystemFontSize(ctx, newSize)
}

// ListFonts lists all downloadable Nerd Font names.
func (s *FontService) ListFonts() []string {
	fonts := s.GetAvailableFonts()

	names := make([]string, 0, len(fonts))
	for name := range fonts {
		names = append(names, name)
	}

	return names
}

// GetFont retrieves configuration for the named font.
func (s *FontService) GetFont(name string) (*FontConfig, error) {
	fonts := s.GetAvailableFonts()
	if font, ok := fonts[name]; ok {
		return &font, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrUnknownFont, name)
}

func (s *FontService) extractAndInstallFont(_ context.Context, zipPath string, font FontConfig) error {
	// Read zip file
	data, err := s.fileManager.ReadFile(zipPath)
	if err != nil {
		return fmt.Errorf("reading font zip %s: %w", zipPath, err)
	}

	// Open zip archive from bytes
	reader, err := zip.NewReader(strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		return fmt.Errorf("opening font archive %s: %w", font.Name, err)
	}

	// Extract font files
	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, "."+font.FileType) {
			continue
		}

		// Skip Windows-specific fonts
		if strings.Contains(file.Name, "Windows") {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("opening font file %s: %w", file.Name, err)
		}

		defer func() {
			_ = rc.Close()
		}()

		fontData, err := io.ReadAll(rc)
		if err != nil {
			return fmt.Errorf("reading font file %s: %w", file.Name, err)
		}

		// Install to user fonts directory
		destPath := filepath.Join(s.fontsDir, filepath.Base(file.Name))
		if err := s.fileManager.WriteFile(destPath, fontData); err != nil {
			return fmt.Errorf("writing font %s to %s: %w", font.Name, destPath, err)
		}
	}

	return nil
}

func (s *FontService) getCurrentFontSize(ctx context.Context) (int, error) {
	output, err := s.commandRunner.ExecuteWithOutput(ctx, "gsettings", "get",
		"org.gnome.desktop.interface", "font-name")
	if err != nil {
		return 0, fmt.Errorf("getting current font size: %w", err)
	}

	// Extract size from font string like "'Ubuntu 11'"
	re := regexp.MustCompile(`\d+`)

	matches := re.FindString(output)
	if matches == "" {
		return 11, nil // Default size
	}

	size, err := strconv.Atoi(matches)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidSizeFormat, matches)
	}

	return size, nil
}

func (s *FontService) setSystemFontSize(ctx context.Context, size int) error {
	if size < 6 || size > 24 {
		return ErrInvalidFontSize
	}

	// Get current font name
	output, err := s.commandRunner.ExecuteWithOutput(ctx, "gsettings", "get",
		"org.gnome.desktop.interface", "font-name")
	if err != nil {
		return fmt.Errorf("getting font name for size update: %w", err)
	}

	// Replace size in font string
	re := regexp.MustCompile(`(\d+)`)
	newFont := re.ReplaceAllString(output, strconv.Itoa(size))

	// Apply new font size
	return s.commandRunner.Execute(ctx, "gsettings", "set",
		"org.gnome.desktop.interface", "font-name", newFont)
}

func (s *FontService) applyTerminalFont(ctx context.Context, fontName string) error {
	// Get current profile
	output, err := s.commandRunner.ExecuteWithOutput(ctx, "gsettings", "get",
		"org.gnome.Terminal.ProfilesList", "default")
	if err != nil {
		return err
	}

	profileID := strings.Trim(strings.TrimSpace(output), "'")
	profilePath := fmt.Sprintf("org.gnome.Terminal.Legacy.Profile:/org/gnome/terminal/legacy/profiles:/:%s/", profileID)

	// Set font
	return s.commandRunner.Execute(ctx, "gsettings", "set", profilePath, "font",
		fmt.Sprintf("'%s 11'", fontName))
}

func (s *FontService) applyEditorFont(ctx context.Context, fontName string) error {
	// Apply to gedit
	if err := s.commandRunner.Execute(ctx, "gsettings", "set",
		"org.gnome.gedit.preferences.editor", "editor-font",
		fmt.Sprintf("'%s 11'", fontName)); err != nil {
		return err
	}

	return s.commandRunner.Execute(ctx, "gsettings", "set",
		"org.gnome.gedit.preferences.editor", "use-default-font", "false")
}
