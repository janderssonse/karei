// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package themes provides theme management and application for Karei.
package themes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

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
	// ChromeSchemeAuto uses system color scheme.
	ChromeSchemeAuto = 0
	// ChromeSchemeLight forces light theme.
	ChromeSchemeLight = 1
	// ChromeSchemeDark forces dark theme.
	ChromeSchemeDark = 2
)

// Chrome theme variant constants.
const (
	// ChromeVariantTonalSpot uses tonal spot color variant.
	ChromeVariantTonalSpot = 0
	// ChromeVariantNeutral uses neutral color variant.
	ChromeVariantNeutral = 1
	// ChromeVariantVibrant uses vibrant color variant.
	ChromeVariantVibrant = 2
	// ChromeVariantExpressive uses expressive color variant.
	ChromeVariantExpressive = 3
)

// ThemeConfig represents a complete theme configuration.
type ThemeConfig struct {
	Name            string `json:"name"`
	ColorScheme     string `json:"color_scheme"` // prefer-dark or prefer-light
	GtkTheme        string `json:"gtk_theme"`
	IconTheme       string `json:"icon_theme"`
	CursorTheme     string `json:"cursor_theme"`
	AccentColor     string `json:"accent_color"`
	ChromeScheme    int    `json:"chrome_scheme"`  // 0=auto, 1=light, 2=dark
	ChromeVariant   int    `json:"chrome_variant"` // 0=tonal_spot, 1=neutral, 2=vibrant, 3=expressive
	ChromeColor     int    `json:"chrome_color"`   // RGB color as int
	VSCodeExtension string `json:"vscode_extension"`
	VSCodeTheme     string `json:"vscode_theme"`
	Background      string `json:"background"` // filename in theme dir
}

// IsValid validates the theme has required fields.
func (t *ThemeConfig) IsValid() bool {
	return t.Name != ""
}

// GetName returns the name of the theme.
func (t *ThemeConfig) GetName() string {
	return t.Name
}

// Theme is an alias for ThemeConfig for compatibility.
type Theme = ThemeConfig

func getThemes() map[string]ThemeConfig {
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
			ChromeColor:     9699539,
			VSCodeExtension: "catppuccin.catppuccin-vsc",
			VSCodeTheme:     "Catppuccin Mocha",
			Background:      "background.png",
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
			ChromeColor:     5951037,
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
			ChromeColor:     7384391,
			VSCodeExtension: "sainnhe.everforest",
			VSCodeTheme:     "Everforest Dark",
			Background:      "background.jpg",
		},
		"gruvbox": {
			Name:            "gruvbox",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-bark-dark",
			IconTheme:       "Yaru-bark",
			CursorTheme:     "Yaru",
			AccentColor:     "orange",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     13395456,
			VSCodeExtension: "jdinhlife.gruvbox",
			VSCodeTheme:     "Gruvbox Dark Medium",
			Background:      "background.jpg",
		},
		"gruvbox-light": {
			Name:            "gruvbox-light",
			ColorScheme:     "prefer-light",
			GtkTheme:        "Yaru",
			IconTheme:       "Yaru",
			CursorTheme:     "Yaru",
			AccentColor:     "orange",
			ChromeScheme:    1,
			ChromeVariant:   0,
			ChromeColor:     13395456,
			VSCodeExtension: "jdinhlife.gruvbox",
			VSCodeTheme:     "Gruvbox Light Medium",
			Background:      "background.jpg",
		},
		"kanagawa": {
			Name:            "kanagawa",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-red-dark",
			IconTheme:       "Yaru-red",
			CursorTheme:     "Yaru",
			AccentColor:     "red",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     8947848,
			VSCodeExtension: "qufiwefefwoyn.kanagawa",
			VSCodeTheme:     "Kanagawa",
			Background:      "background.jpg",
		},
		"rose-pine": {
			Name:            "rose-pine",
			ColorScheme:     "prefer-dark",
			GtkTheme:        "Yaru-pink-dark",
			IconTheme:       "Yaru-pink",
			CursorTheme:     "Yaru",
			AccentColor:     "pink",
			ChromeScheme:    ChromeSchemeDark,
			ChromeVariant:   ChromeVariantTonalSpot,
			ChromeColor:     12171705,
			VSCodeExtension: "mvllow.rose-pine",
			VSCodeTheme:     "Rosé Pine",
			Background:      "background.jpg",
		},
	}
}

// ApplyGnomeTheme applies a theme to the GNOME desktop environment.
func ApplyGnomeTheme(ctx context.Context, themeName string) error {
	return ApplyGnomeThemeWithOptions(ctx, themeName, false)
}

// ApplyGnomeThemeWithOptions applies a GNOME theme with additional options.
func ApplyGnomeThemeWithOptions(ctx context.Context, themeName string, dryRun bool) error {
	theme, exists := getThemes()[themeName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownTheme, themeName)
	}

	settings := map[string]string{
		"org.gnome.desktop.interface color-scheme": theme.ColorScheme,
		"org.gnome.desktop.interface cursor-theme": theme.CursorTheme,
		"org.gnome.desktop.interface gtk-theme":    theme.GtkTheme,
		"org.gnome.desktop.interface icon-theme":   theme.IconTheme,
		"org.gnome.desktop.interface accent-color": theme.AccentColor,
	}

	for key, value := range settings {
		if dryRun {
			// In dry run mode, just log what would be done
			continue
		}

		cmd := exec.CommandContext(ctx, "gsettings", "set", key, value) //nolint:gosec
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	// Set background if exists
	if theme.Background != "" && !dryRun {
		backgroundPath := fmt.Sprintf("file:///home/%s/.local/share/karei/themes/%s/%s",
			os.Getenv("USER"), themeName, theme.Background)
		cmd := exec.CommandContext(ctx, "gsettings", "set", "org.gnome.desktop.background", "picture-uri", backgroundPath) //nolint:gosec
		_ = cmd.Run()
		cmd = exec.CommandContext(ctx, "gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", backgroundPath) //nolint:gosec
		_ = cmd.Run()
	}

	return nil
}

// ChromePreferences represents Chrome's preferences structure.
type ChromePreferences struct {
	Extensions map[string]any             `json:"extensions,omitempty"`
	Browser    map[string]any             `json:"browser,omitempty"`
	NTP        map[string]any             `json:"ntp,omitempty"`
	Extra      map[string]json.RawMessage `json:"-"` // Preserve unknown fields
}

// MarshalJSON implements custom marshaling to preserve unknown fields.
func (c *ChromePreferences) MarshalJSON() ([]byte, error) {
	result := make(map[string]any)

	// Add known fields
	if c.Extensions != nil {
		result["extensions"] = c.Extensions
	}

	if c.Browser != nil {
		result["browser"] = c.Browser
	}

	if c.NTP != nil {
		result["ntp"] = c.NTP
	}

	// Add preserved unknown fields
	for k, v := range c.Extra {
		result[k] = v
	}

	return json.Marshal(result)
}

// ApplyChromeTheme applies a theme to Google Chrome browser.
func ApplyChromeTheme(ctx context.Context, themeName string) error { //nolint:cyclop
	theme, exists := getThemes()[themeName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownTheme, themeName)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	prefsPath := filepath.Join(configDir, "google-chrome/Default/Preferences")

	if _, err := os.Stat(prefsPath); errors.Is(err, fs.ErrNotExist) {
		return nil // Chrome not installed, skip
	}

	// Kill Chrome
	_ = exec.CommandContext(ctx, "pkill", "-f", "chrome").Run()

	// Wait for Chrome to close with timeout
	if err := waitForProcessExit(ctx, "chrome", 5*time.Second); err != nil {
		// Chrome didn't exit in time, but continue anyway
		_ = err
	}

	// Read preferences
	data, err := os.ReadFile(prefsPath) //nolint:gosec
	if err != nil {
		return err
	}

	// Parse into structured type with preserved unknown fields
	var prefs ChromePreferences

	// First unmarshal to preserve all fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract known fields
	prefs.Extra = make(map[string]json.RawMessage)

	for k, v := range raw {
		switch k {
		case "extensions":
			if err := json.Unmarshal(v, &prefs.Extensions); err != nil {
				prefs.Extensions = make(map[string]any)
			}
		case "browser":
			if err := json.Unmarshal(v, &prefs.Browser); err != nil {
				prefs.Browser = make(map[string]any)
			}
		case "ntp":
			if err := json.Unmarshal(v, &prefs.NTP); err != nil {
				prefs.NTP = make(map[string]any)
			}
		default:
			prefs.Extra[k] = v
		}
	}

	// Ensure maps are initialized
	if prefs.Extensions == nil {
		prefs.Extensions = make(map[string]any)
	}

	if prefs.Browser == nil {
		prefs.Browser = make(map[string]any)
	}

	if prefs.NTP == nil {
		prefs.NTP = make(map[string]any)
	}

	// Set theme values without type assertions
	prefs.Extensions["theme"] = map[string]any{
		"id":           "user_color_theme_id",
		"system_theme": 0,
	}

	prefs.Browser["theme"] = map[string]any{
		"color_scheme":  theme.ChromeScheme,
		"color_variant": theme.ChromeVariant,
		"user_color":    theme.ChromeColor,
	}

	prefs.NTP["custom_background_dict"] = map[string]any{
		"background_url": fmt.Sprintf("https://github.com/janderssonse/karei/blob/master/themes/%s/%s?raw=true", themeName, theme.Background),
	}

	// Write back
	newData, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	return os.WriteFile(prefsPath, newData, 0644) //nolint:gosec
}

// ThemeApplication represents a theme application operation.
type ThemeApplication struct {
	Name    string
	Applied bool
	Error   error
}

// ApplyThemeToAllApplications applies a theme to all supported applications using functional patterns.
func ApplyThemeToAllApplications(ctx context.Context, themeName string) []ThemeApplication {
	applications := []struct {
		name      string
		applyFunc func(context.Context, string) error
	}{
		{"GNOME", ApplyGnomeTheme},
		{"Chrome", ApplyChromeTheme},
		{"VSCode", ApplyVSCodeTheme},
	}

	result := make([]ThemeApplication, len(applications))
	for i, app := range applications {
		err := app.applyFunc(ctx, themeName)
		result[i] = ThemeApplication{
			Name:    app.name,
			Applied: err == nil,
			Error:   err,
		}
	}

	return result
}

// GetSuccessfulApplications filters theme applications to show only successful ones.
func GetSuccessfulApplications(applications []ThemeApplication) []string {
	var successful []string

	for _, app := range applications {
		if app.Applied {
			successful = append(successful, app.Name)
		}
	}

	return successful
}

// GetFailedApplications filters theme applications to show only failed ones.
func GetFailedApplications(applications []ThemeApplication) []ThemeApplication {
	var failed []ThemeApplication

	for _, app := range applications {
		if !app.Applied {
			failed = append(failed, app)
		}
	}

	return failed
}

// ApplyVSCodeTheme applies a theme to Visual Studio Code.
func ApplyVSCodeTheme(ctx context.Context, themeName string) error {
	theme, exists := getThemes()[themeName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownTheme, themeName)
	}

	// Check if code command exists
	if _, err := exec.LookPath("code"); err != nil {
		// VSCode not installed, skip - this is not an error condition
		return nil //nolint:nilerr
	}

	// Install extension
	cmd := exec.CommandContext(ctx, "code", "--install-extension", theme.VSCodeExtension) //nolint:gosec
	_ = cmd.Run()                                                                         // Ignore errors

	// Update settings
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	settingsPath := filepath.Join(configDir, "Code/User/settings.json")

	var settings map[string]any
	if data, err := os.ReadFile(settingsPath); err == nil { //nolint:gosec
		_ = json.Unmarshal(data, &settings)
	}

	if settings == nil {
		settings = make(map[string]any)
	}

	settings["workbench.colorTheme"] = theme.VSCodeTheme

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	_ = os.MkdirAll(filepath.Dir(settingsPath), 0755) //nolint:gosec

	return os.WriteFile(settingsPath, data, 0644) //nolint:gosec
}

// waitForProcessExit waits for a process to exit with a timeout.
func waitForProcessExit(ctx context.Context, processName string, timeout time.Duration) error {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("timeout waiting for %s to exit: %w", processName, waitCtx.Err())
		case <-ticker.C:
			if exec.CommandContext(ctx, "pgrep", "-f", processName).Run() != nil {
				// Process has exited
				return nil
			}
		}
	}
}
