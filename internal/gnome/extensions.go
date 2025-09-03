// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package gnome

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

var (
	// ErrUnknownExtension is returned when the requested extension is not found.
	ErrUnknownExtension = errors.New("unknown extension")
	// ErrHomeNotSet is returned when HOME environment variable is not set.
	ErrHomeNotSet = errors.New("HOME environment variable not set")
	// ErrSchemaNotFound is returned when extension schema file is not found.
	ErrSchemaNotFound = errors.New("schema file not found")
)

// Extension represents a GNOME extension configuration.
type Extension struct {
	ID          string
	Name        string
	Description string
	SchemaPath  string
	Settings    map[string]any
}

// Extensions contains available GNOME extensions.
var Extensions = map[string]Extension{ //nolint:gochecknoglobals
	"tactile": {
		ID:          "tactile@lundal.io",
		Name:        "Tactile",
		Description: "Advanced window tiling for GNOME",
		SchemaPath:  "org.gnome.shell.extensions.tactile.gschema.xml",
		Settings: map[string]any{
			"org.gnome.shell.extensions.tactile col-0":    1,
			"org.gnome.shell.extensions.tactile col-1":    2,
			"org.gnome.shell.extensions.tactile col-2":    1,
			"org.gnome.shell.extensions.tactile col-3":    0,
			"org.gnome.shell.extensions.tactile row-0":    1,
			"org.gnome.shell.extensions.tactile row-1":    1,
			"org.gnome.shell.extensions.tactile gap-size": 32,
		},
	},
	"just-perfection": {
		ID:          "just-perfection-desktop@just-perfection",
		Name:        "Just Perfection",
		Description: "Customize GNOME Shell interface",
		SchemaPath:  "org.gnome.shell.extensions.just-perfection.gschema.xml",
		Settings: map[string]any{
			"org.gnome.shell.extensions.just-perfection animation":        2,
			"org.gnome.shell.extensions.just-perfection dash-app-running": true,
			"org.gnome.shell.extensions.just-perfection workspace":        true,
			"org.gnome.shell.extensions.just-perfection workspace-popup":  false,
		},
	},
	"blur-shell": {
		ID:          "blur-my-shell@aunetx",
		Name:        "Blur My Shell",
		Description: "Adds blur effects to GNOME Shell",
		SchemaPath:  "org.gnome.shell.extensions.blur-my-shell.gschema.xml",
		Settings: map[string]any{
			"org.gnome.shell.extensions.blur-my-shell.appfolder blur":                  false,
			"org.gnome.shell.extensions.blur-my-shell.lockscreen blur":                 false,
			"org.gnome.shell.extensions.blur-my-shell.screenshot blur":                 false,
			"org.gnome.shell.extensions.blur-my-shell.window-list blur":                false,
			"org.gnome.shell.extensions.blur-my-shell.panel blur":                      false,
			"org.gnome.shell.extensions.blur-my-shell.overview blur":                   true,
			"org.gnome.shell.extensions.blur-my-shell.overview pipeline":               "pipeline_default",
			"org.gnome.shell.extensions.blur-my-shell.dash-to-dock blur":               true,
			"org.gnome.shell.extensions.blur-my-shell.dash-to-dock brightness":         0.6,
			"org.gnome.shell.extensions.blur-my-shell.dash-to-dock sigma":              30,
			"org.gnome.shell.extensions.blur-my-shell.dash-to-dock static-blur":        true,
			"org.gnome.shell.extensions.blur-my-shell.dash-to-dock style-dash-to-dock": 0,
		},
	},
	"space-bar": {
		ID:          "space-bar@luchrioh",
		Name:        "Space Bar",
		Description: "Enhanced workspace management",
		SchemaPath:  "org.gnome.shell.extensions.space-bar.gschema.xml",
		Settings: map[string]any{
			"org.gnome.shell.extensions.space-bar.behavior smart-workspace-names":                false,
			"org.gnome.shell.extensions.space-bar.shortcuts enable-activate-workspace-shortcuts": false,
			"org.gnome.shell.extensions.space-bar.shortcuts enable-move-to-workspace-shortcuts":  true,
		},
	},
	"undecorate": {
		ID:          "undecorated-windows@tabdeveloper.github.com",
		Name:        "Undecorate",
		Description: "Remove window decorations",
		SchemaPath:  "org.gnome.shell.extensions.undecorate.gschema.xml",
		Settings:    map[string]any{},
	},
	"tophat": {
		ID:          "tophat@fflewddur.github.io",
		Name:        "TopHat",
		Description: "System monitoring in top bar",
		SchemaPath:  "org.gnome.shell.extensions.tophat.gschema.xml",
		Settings: map[string]any{
			"org.gnome.shell.extensions.tophat show-icons":         false,
			"org.gnome.shell.extensions.tophat show-cpu":           false,
			"org.gnome.shell.extensions.tophat show-disk":          false,
			"org.gnome.shell.extensions.tophat show-mem":           false,
			"org.gnome.shell.extensions.tophat show-fs":            false,
			"org.gnome.shell.extensions.tophat network-usage-unit": "bits",
		},
	},
	"alphabetical-grid": {
		ID:          "AlphabeticalAppGrid@stuarthayhurst",
		Name:        "Alphabetical App Grid",
		Description: "Sort application grid alphabetically",
		SchemaPath:  "org.gnome.shell.extensions.AlphabeticalAppGrid.gschema.xml",
		Settings: map[string]any{
			"org.gnome.shell.extensions.alphabetical-app-grid folder-order-position": "end",
		},
	},
}

// ExtensionManager manages GNOME shell extensions.
type ExtensionManager struct {
	verbose bool
}

// NewExtensionManager creates a new extension manager instance.
func NewExtensionManager(verbose bool) *ExtensionManager {
	return &ExtensionManager{verbose: verbose}
}

// InstallExtensionManager installs the GNOME extension management tools.
func (m *ExtensionManager) InstallExtensionManager(ctx context.Context) error {
	if m.verbose {
		fmt.Println("Installing GNOME extension management tools...")
	}

	// Install gnome-shell-extension-manager via apt
	if err := exec.CommandContext(ctx, "sudo", "apt", "-y", "install", "gnome-shell-extension-manager").Run(); err != nil {
		return fmt.Errorf("failed to install extension manager: %w", err)
	}

	// Install pipx if not available
	if !m.isCommandAvailable(ctx, "pipx") {
		if err := exec.CommandContext(ctx, "sudo", "apt", "-y", "install", "pipx").Run(); err != nil {
			return fmt.Errorf("failed to install pipx: %w", err)
		}
	}

	// Install gnome-extensions-cli via pipx
	if err := exec.CommandContext(ctx, "pipx", "install", "gnome-extensions-cli", "--system-site-packages").Run(); err != nil {
		return fmt.Errorf("failed to install gnome-extensions-cli: %w", err)
	}

	if m.verbose {
		fmt.Println("✓ GNOME extension management tools installed")
	}

	return nil
}

// InstallExtension installs a specific GNOME shell extension.
func (m *ExtensionManager) InstallExtension(ctx context.Context, extensionName string) error {
	ext, exists := Extensions[extensionName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownExtension, extensionName)
	}

	if m.verbose {
		fmt.Printf("Installing %s GNOME extension...\n", ext.Name)
	}

	// Install extension using gext (gnome-extensions-cli)
	if err := exec.CommandContext(ctx, "gext", "install", ext.ID).Run(); err != nil { //nolint:gosec
		return fmt.Errorf("failed to install extension %s: %w", ext.Name, err)
	}

	// Copy schema file to system location
	if ext.SchemaPath != "" {
		if err := m.copyExtensionSchema(ctx, ext); err != nil {
			if m.verbose {
				fmt.Printf("Warning: Failed to copy schema for %s: %v\n", ext.Name, err)
			} else {
				fmt.Printf("Warning: Failed to copy schema for %s\n", ext.Name)
			}
		}
	}

	if m.verbose {
		fmt.Printf("✓ %s extension installed\n", ext.Name)
	}

	return nil
}

// InstallAllExtensions installs all configured GNOME shell extensions.
func (m *ExtensionManager) InstallAllExtensions(ctx context.Context) error {
	// First install extension manager
	if err := m.InstallExtensionManager(ctx); err != nil {
		return err
	}

	// Disable default Ubuntu extensions
	if err := m.DisableDefaultExtensions(ctx); err != nil {
		if m.verbose {
			fmt.Printf("Warning: Failed to disable default extensions: %v\n", err)
		}
	}

	// Install individual extensions
	for name := range Extensions {
		if err := m.InstallExtension(ctx, name); err != nil {
			if m.verbose {
				fmt.Printf("Warning: Failed to install %s: %v\n", name, err)
			} else {
				fmt.Printf("Warning: Failed to install %s\n", name)
			}
		}
	}

	// Compile schemas
	if err := m.CompileSchemas(ctx); err != nil {
		return fmt.Errorf("failed to compile schemas: %w", err)
	}

	// Configure extension settings
	if err := m.ConfigureExtensions(ctx); err != nil {
		return fmt.Errorf("failed to configure extensions: %w", err)
	}

	fmt.Println("✓ All GNOME extensions installed and configured")

	return nil
}

// DisableDefaultExtensions disables all default GNOME shell extensions.
func (m *ExtensionManager) DisableDefaultExtensions(ctx context.Context) error {
	if m.verbose {
		fmt.Println("Disabling default Ubuntu extensions...")
	}

	defaultExtensions := []string{
		"ubuntu-dock@ubuntu.com",
		"ubuntu-appindicators@ubuntu.com",
		"desktop-icons-ng@rastersoft.com",
	}

	for _, extID := range defaultExtensions {
		_ = exec.CommandContext(ctx, "gnome-extensions", "disable", extID).Run() //nolint:gosec
	}

	return nil
}

// CompileSchemas compiles the GNOME extension schemas.
func (m *ExtensionManager) CompileSchemas(ctx context.Context) error {
	if m.verbose {
		fmt.Println("Compiling GNOME schemas...")
	}

	return exec.CommandContext(ctx, "sudo", "glib-compile-schemas", "/usr/share/glib-2.0/schemas/").Run()
}

// ConfigureExtensions configures all installed GNOME shell extensions.
func (m *ExtensionManager) ConfigureExtensions(ctx context.Context) error {
	if m.verbose {
		fmt.Println("Configuring GNOME extension settings...")
	}

	for _, ext := range Extensions {
		for setting, value := range ext.Settings {
			if err := m.setGSetting(ctx, setting, value); err != nil {
				fmt.Printf("Warning: Failed to set %s: %v\n", setting, err)
			}
		}
	}

	return nil
}

// EnableExtension activates the specified GNOME shell extension.
func (m *ExtensionManager) EnableExtension(ctx context.Context, extensionName string) error {
	ext, exists := Extensions[extensionName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownExtension, extensionName)
	}

	return exec.CommandContext(ctx, "gnome-extensions", "enable", ext.ID).Run() //nolint:gosec
}

// DisableExtension disables a specific GNOME shell extension.
func (m *ExtensionManager) DisableExtension(ctx context.Context, extensionName string) error {
	ext, exists := Extensions[extensionName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownExtension, extensionName)
	}

	return exec.CommandContext(ctx, "gnome-extensions", "disable", ext.ID).Run() //nolint:gosec
}

// ListInstalledExtensions returns a list of all installed GNOME shell extensions.
func (m *ExtensionManager) ListInstalledExtensions(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "gnome-extensions", "list")

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse output to get list of installed extensions
	// This is a simplified implementation
	return []string{string(output)}, nil
}

// GetExtensionStatus returns the status of a specific GNOME shell extension.
func (m *ExtensionManager) GetExtensionStatus(ctx context.Context, extensionName string) (bool, error) {
	ext, exists := Extensions[extensionName]
	if !exists {
		return false, fmt.Errorf("%w: %s", ErrUnknownExtension, extensionName)
	}

	cmd := exec.CommandContext(ctx, "gnome-extensions", "info", ext.ID) //nolint:gosec
	err := cmd.Run()

	return err == nil, nil
}

func (m *ExtensionManager) copyExtensionSchema(ctx context.Context, ext Extension) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		xdgDataHome = filepath.Join(home, ".local", "share")
	}

	sourcePath := filepath.Join(xdgDataHome, "gnome-shell", "extensions", ext.ID, "schemas", ext.SchemaPath)
	destPath := filepath.Join("/usr/share/glib-2.0/schemas/", ext.SchemaPath)

	// Check if source exists
	if _, err := os.Stat(sourcePath); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrSchemaNotFound, sourcePath)
	}

	// Copy schema file
	return exec.CommandContext(ctx, "sudo", "cp", sourcePath, destPath).Run() //nolint:gosec
}

func (m *ExtensionManager) setGSetting(ctx context.Context, setting string, value any) error {
	var valueStr string

	switch settingValue := value.(type) {
	case string:
		valueStr = fmt.Sprintf("'%s'", settingValue)
	case bool:
		if settingValue {
			valueStr = "true"
		} else {
			valueStr = "false"
		}
	case int:
		valueStr = strconv.Itoa(settingValue)
	case float64:
		valueStr = fmt.Sprintf("%.1f", settingValue)
	default:
		valueStr = fmt.Sprintf("%v", settingValue)
	}

	return exec.CommandContext(ctx, "gsettings", "set", setting, valueStr).Run() //nolint:gosec
}

func (m *ExtensionManager) isCommandAvailable(_ context.Context, command string) bool {
	_, err := exec.LookPath(command)

	return err == nil
}
