// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/janderssonse/karei/internal/domain"
)

// ThemeService manages system themes using hexagonal architecture.
type ThemeService struct {
	fileManager   domain.FileManager
	commandRunner domain.CommandRunner
	configPath    string
	themesPath    string
}

// NewThemeService creates a ThemeService.
func NewThemeService(fm domain.FileManager, cr domain.CommandRunner, configPath, themesPath string) *ThemeService {
	return &ThemeService{
		fileManager:   fm,
		commandRunner: cr,
		configPath:    configPath,
		themesPath:    themesPath,
	}
}

// ThemeConfig represents a complete theme configuration.
type ThemeConfig struct {
	Name            string `json:"name"`
	ColorScheme     string `json:"color_scheme"`
	GtkTheme        string `json:"gtk_theme"`
	IconTheme       string `json:"icon_theme"`
	CursorTheme     string `json:"cursor_theme"`
	AccentColor     string `json:"accent_color"`
	ChromeScheme    int    `json:"chrome_scheme"`
	ChromeVariant   int    `json:"chrome_variant"`
	ChromeColor     int    `json:"chrome_color"`
	VSCodeExtension string `json:"vscode_extension"`
	VSCodeTheme     string `json:"vscode_theme"`
	Background      string `json:"background"`
}

var (
	// ErrUnknownTheme is returned when an unknown theme is requested.
	ErrUnknownTheme = errors.New("unknown theme")
	// ErrInvalidPreferenceMap is returned when preference is not a map.
	ErrInvalidPreferenceMap = errors.New("preference is not a map")
	// ErrInvalidTheme is returned when theme data is invalid.
	ErrInvalidTheme = errors.New("invalid theme")
	// ErrThemeNotFound is returned when theme file is not found.
	ErrThemeNotFound = errors.New("theme not found")
)

// Chrome theme scheme constants.
const (
	ChromeSchemeAuto        = 0
	ChromeSchemeLight       = 1
	ChromeSchemeDark        = 2
	ChromeVariantTonalSpot  = 0
	ChromeVariantNeutral    = 1
	ChromeVariantVibrant    = 2
	ChromeVariantExpressive = 3
)

// GetAvailableThemes returns available themes.
func (s *ThemeService) GetAvailableThemes() map[string]ThemeConfig {
	return map[string]ThemeConfig{
		"tokyo-night": {
			Name:            "tokyo-night",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-purple-dark",
			IconTheme:       "Yaru-purple",
			CursorTheme:     "Yaru",
			AccentColor:     "purple",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     4521796,
			VSCodeExtension: "enkia.tokyo-night",
			VSCodeTheme:     "Tokyo Night",
			Background:      "background.jpg",
		},
		"catppuccin": {
			Name:            "catppuccin",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-purple-dark",
			IconTheme:       "Yaru-purple",
			CursorTheme:     "Yaru",
			AccentColor:     "purple",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     11625079,
			VSCodeExtension: "Catppuccin.catppuccin-vsc",
			VSCodeTheme:     "Catppuccin Mocha",
			Background:      "background.png",
		},
		"gruvbox": {
			Name:            "gruvbox",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-orange-dark",
			IconTheme:       "Yaru-orange",
			CursorTheme:     "Yaru",
			AccentColor:     "orange",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     2372448,
			VSCodeExtension: "jdinhlife.gruvbox",
			VSCodeTheme:     "Gruvbox Dark Medium",
			Background:      "background.jpg",
		},
		"nord": {
			Name:            "nord",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-blue-dark",
			IconTheme:       "Yaru-blue",
			CursorTheme:     "Yaru",
			AccentColor:     "blue",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     5815733,
			VSCodeExtension: "arcticicestudio.nord-visual-studio-code",
			VSCodeTheme:     "Nord",
			Background:      "background.png",
		},
		"everforest": {
			Name:            "everforest",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-green-dark",
			IconTheme:       "Yaru-green",
			CursorTheme:     "Yaru",
			AccentColor:     "green",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     5282618,
			VSCodeExtension: "sainnhe.everforest",
			VSCodeTheme:     "Everforest Dark",
			Background:      "background.jpg",
		},
		"kanagawa": {
			Name:            "kanagawa",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-dark",
			IconTheme:       "Yaru",
			CursorTheme:     "Yaru",
			AccentColor:     "orange",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     7830409,
			VSCodeExtension: "qufiwefefwoyn.kanagawa",
			VSCodeTheme:     "Kanagawa",
			Background:      "background.jpg",
		},
		"rose-pine": {
			Name:            "rose-pine",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-red-dark",
			IconTheme:       "Yaru-red",
			CursorTheme:     "Yaru",
			AccentColor:     "red",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     3291837,
			VSCodeExtension: "mvllow.rose-pine",
			VSCodeTheme:     "Rosé Pine",
			Background:      "background.jpg",
		},
		"gruvbox-light": {
			Name:          "gruvbox-light",
			ColorScheme:   "prefer-light",
			GtkTheme:      "Yaru-orange",
			IconTheme:     "Yaru-orange",
			CursorTheme:   "Yaru",
			AccentColor:   "orange",
			ChromeScheme:  ChromeSchemeLight,
			ChromeVariant: ChromeVariantTonalSpot,
			ChromeColor:   2372448,
			VSCodeTheme:   "Gruvbox Light Medium",
			Background:    "background.jpg",
		},
	}
}

// ApplyTheme applies the specified theme.
func (s *ThemeService) ApplyTheme(ctx context.Context, themeName string) error {
	themes := s.GetAvailableThemes()

	theme, exists := themes[themeName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownTheme, themeName)
	}

	// Apply GNOME settings
	if err := s.ApplyGnomeSettings(ctx, &theme); err != nil {
		return fmt.Errorf("failed to apply GNOME settings: %w", err)
	}

	// Apply background
	if theme.Background != "" {
		if err := s.ApplyBackground(ctx, themeName, theme.Background); err != nil {
			return fmt.Errorf("failed to apply background: %w", err)
		}
	}

	// Apply terminal theme
	if err := s.ApplyTerminalTheme(ctx, themeName); err != nil {
		return fmt.Errorf("failed to apply terminal theme: %w", err)
	}

	// Apply btop theme
	if err := s.ApplyBtopTheme(ctx, themeName); err != nil {
		return fmt.Errorf("failed to apply btop theme: %w", err)
	}

	// Apply VSCode theme
	if theme.VSCodeExtension != "" {
		if err := s.ApplyVSCodeTheme(ctx, &theme); err != nil {
			return fmt.Errorf("failed to apply VSCode theme: %w", err)
		}
	}

	// Apply Chrome theme
	if err := s.ApplyChromeTheme(ctx, &theme); err != nil {
		return fmt.Errorf("failed to apply Chrome theme: %w", err)
	}

	return nil
}

// ApplyGnomeSettings applies GNOME-specific theme settings.
func (s *ThemeService) ApplyGnomeSettings(ctx context.Context, theme *ThemeConfig) error {
	settings := []struct {
		schema string
		key    string
		value  string
	}{
		{"org.gnome.desktop.interface", "color-scheme", theme.ColorScheme},
		{"org.gnome.desktop.interface", "gtk-theme", theme.GtkTheme},
		{"org.gnome.desktop.interface", "icon-theme", theme.IconTheme},
		{"org.gnome.desktop.interface", "cursor-theme", theme.CursorTheme},
		{"org.gnome.desktop.interface", "accent-color", theme.AccentColor},
	}

	for _, setting := range settings {
		if err := s.commandRunner.Execute(ctx, "gsettings", "set", setting.schema, setting.key, setting.value); err != nil {
			return fmt.Errorf("failed to set %s.%s: %w", setting.schema, setting.key, err)
		}
	}

	return nil
}

// ApplyBackground applies the background image for a theme.
func (s *ThemeService) ApplyBackground(ctx context.Context, themeName, backgroundFile string) error {
	backgroundPath := filepath.Join(s.themesPath, themeName, backgroundFile)

	if !s.fileManager.FileExists(backgroundPath) {
		return fmt.Errorf("background file not found: %s", backgroundPath)
	}

	pictureURI := "file://" + backgroundPath

	settings := []struct {
		schema string
		key    string
		value  string
	}{
		{"org.gnome.desktop.background", "picture-uri", pictureURI},
		{"org.gnome.desktop.background", "picture-uri-dark", pictureURI},
		{"org.gnome.desktop.background", "picture-options", "zoom"},
		{"org.gnome.desktop.screensaver", "picture-uri", pictureURI},
	}

	for _, setting := range settings {
		if err := s.commandRunner.Execute(ctx, "gsettings", "set", setting.schema, setting.key, setting.value); err != nil {
			return fmt.Errorf("failed to set %s.%s: %w", setting.schema, setting.key, err)
		}
	}

	return nil
}

// ApplyTerminalTheme applies the terminal theme configuration.
func (s *ThemeService) ApplyTerminalTheme(ctx context.Context, themeName string) error {
	profilePath, err := s.getTerminalProfilePath(ctx)
	if err != nil {
		return err
	}

	termTheme, err := s.loadTerminalTheme(themeName)
	if err != nil {
		return err
	}

	if len(termTheme) == 0 {
		return nil // No terminal theme to apply
	}

	return s.applyTerminalSettings(ctx, profilePath, termTheme)
}

// ApplyBtopTheme applies the btop theme configuration.
func (s *ThemeService) ApplyBtopTheme(_ context.Context, themeName string) error {
	btopConfigDir := filepath.Join(s.configPath, "btop")
	if err := s.fileManager.EnsureDir(btopConfigDir); err != nil {
		return fmt.Errorf("failed to create btop config directory: %w", err)
	}

	// Copy theme file
	src := filepath.Join(s.themesPath, themeName, "btop.theme")
	if !s.fileManager.FileExists(src) {
		return nil // Skip if no btop theme
	}

	if err := s.copyBtopTheme(src, btopConfigDir, themeName); err != nil {
		return err
	}

	return s.updateBtopConfig(btopConfigDir, themeName)
}

// ApplyVSCodeTheme applies the VSCode theme configuration.
func (s *ThemeService) ApplyVSCodeTheme(ctx context.Context, theme *ThemeConfig) error {
	// Install extension
	if err := s.commandRunner.Execute(ctx, "code", "--install-extension", theme.VSCodeExtension); err != nil {
		// Non-fatal: VSCode might not be installed, continue without error
		return nil //nolint:nilerr // VSCode is optional
	}

	// Update settings
	settingsPath := filepath.Join(s.configPath, "Code", "User", "settings.json")
	if !s.fileManager.FileExists(settingsPath) {
		if err := s.fileManager.EnsureDir(filepath.Dir(settingsPath)); err != nil {
			return fmt.Errorf("failed to create VSCode config directory: %w", err)
		}

		if err := s.fileManager.WriteFile(settingsPath, []byte("{}")); err != nil {
			return fmt.Errorf("failed to create VSCode settings: %w", err)
		}
	}

	data, err := s.fileManager.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("failed to read VSCode settings: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse VSCode settings: %w", err)
	}

	settings["workbench.colorTheme"] = theme.VSCodeTheme

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal VSCode settings: %w", err)
	}

	return s.fileManager.WriteFile(settingsPath, newData)
}

// ApplyChromeTheme applies the Chrome theme configuration.
func (s *ThemeService) ApplyChromeTheme(ctx context.Context, theme *ThemeConfig) error {
	// Chrome preferences path
	prefsPath := filepath.Join(s.configPath, "google-chrome", "Default", "Preferences")
	if !s.fileManager.FileExists(prefsPath) {
		return nil // Chrome not installed
	}

	data, err := s.fileManager.ReadFile(prefsPath)
	if err != nil {
		return fmt.Errorf("failed to read Chrome preferences: %w", err)
	}

	var prefs map[string]interface{}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return fmt.Errorf("failed to parse Chrome preferences: %w", err)
	}

	// Update theme settings
	if extensions, ok := prefs["extensions"].(map[string]interface{}); ok {
		if theme, ok := extensions["theme"].(map[string]interface{}); ok {
			theme["system_theme"] = 2 // Follow system theme
		}
	}

	// Update browser color scheme
	if browser, ok := prefs["browser"].(map[string]interface{}); ok {
		browser["theme"] = map[string]interface{}{
			"color_scheme":  theme.ChromeScheme,
			"color_variant": theme.ChromeVariant,
			"user_color":    theme.ChromeColor,
		}
	}

	newData, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("failed to marshal Chrome preferences: %w", err)
	}

	// Chrome must be closed when writing preferences
	_ = s.commandRunner.Execute(ctx, "pkill", "-SIGTERM", "chrome")

	time.Sleep(100 * time.Millisecond)

	return s.fileManager.WriteFile(prefsPath, newData)
}

// ListThemes lists all available theme names.
func (s *ThemeService) ListThemes() []string {
	themes := s.GetAvailableThemes()

	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}

	return names
}

// GetTheme retrieves configuration for the named theme.
func (s *ThemeService) GetTheme(name string) (*ThemeConfig, error) {
	themes := s.GetAvailableThemes()
	if theme, ok := themes[name]; ok {
		return &theme, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrUnknownTheme, name)
}

func (s *ThemeService) copyBtopTheme(src, btopConfigDir, themeName string) error {
	dst := filepath.Join(btopConfigDir, "themes", themeName+".theme")
	if err := s.fileManager.EnsureDir(filepath.Dir(dst)); err != nil {
		return fmt.Errorf("failed to create btop themes directory: %w", err)
	}

	if err := s.fileManager.CopyFile(src, dst); err != nil {
		return fmt.Errorf("failed to copy btop theme: %w", err)
	}

	return nil
}

func (s *ThemeService) updateBtopConfig(btopConfigDir, themeName string) error {
	configFile := filepath.Join(btopConfigDir, "btop.conf")

	var config []byte

	if s.fileManager.FileExists(configFile) {
		config, _ = s.fileManager.ReadFile(configFile)
	}

	configStr := s.updateBtopConfigContent(string(config), themeName)

	return s.fileManager.WriteFile(configFile, []byte(configStr))
}

func (s *ThemeService) updateBtopConfigContent(configStr, themeName string) string {
	themeLineNew := fmt.Sprintf("color_theme = \"%s\"", themeName)

	if strings.Contains(configStr, "color_theme") {
		lines := strings.Split(configStr, "\n")
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "color_theme") {
				lines[i] = themeLineNew
				break
			}
		}

		return strings.Join(lines, "\n")
	}

	if configStr != "" && !strings.HasSuffix(configStr, "\n") {
		configStr += "\n"
	}

	return configStr + themeLineNew + "\n"
}

func (s *ThemeService) getTerminalProfilePath(ctx context.Context) (string, error) {
	output, err := s.commandRunner.ExecuteWithOutput(ctx, "gsettings", "get", "org.gnome.Terminal.ProfilesList", "default")
	if err != nil {
		return "", fmt.Errorf("failed to get default terminal profile: %w", err)
	}

	profileID := strings.Trim(strings.TrimSpace(output), "'")

	return fmt.Sprintf("org.gnome.Terminal.Legacy.Profile:/org/gnome/terminal/legacy/profiles:/:%s/", profileID), nil
}

func (s *ThemeService) loadTerminalTheme(themeName string) (map[string]interface{}, error) {
	themeFile := filepath.Join(s.themesPath, themeName, "gnome-terminal.json")
	if !s.fileManager.FileExists(themeFile) {
		// Return empty map instead of nil to avoid nilnil
		return make(map[string]interface{}), nil
	}

	data, err := s.fileManager.ReadFile(themeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read theme file: %w", err)
	}

	var termTheme map[string]interface{}
	if err := json.Unmarshal(data, &termTheme); err != nil {
		return nil, fmt.Errorf("failed to parse theme: %w", err)
	}

	return termTheme, nil
}

func (s *ThemeService) applyTerminalSettings(ctx context.Context, profilePath string, termTheme map[string]interface{}) error {
	for key, value := range termTheme {
		valueStr := s.formatTerminalValue(value)
		if err := s.commandRunner.Execute(ctx, "gsettings", "set", profilePath, key, valueStr); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	return nil
}

func (s *ThemeService) formatTerminalValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case float64:
		if v == float64(int(v)) {
			return strconv.Itoa(int(v))
		}

		return fmt.Sprintf("%f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
