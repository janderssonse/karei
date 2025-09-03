// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package patterns

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/janderssonse/karei/internal/adapters/platform"
	"github.com/janderssonse/karei/internal/application"
	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/config"
	"github.com/janderssonse/karei/internal/console"
	"github.com/janderssonse/karei/internal/system"
)

const (
	// ghosttyApp represents the ghostty terminal application.
	ghosttyApp = "ghostty"
)

// ManagerFactories provides simplified constructors for all manager types
// Replaces 15+ separate manager packages with unified factory pattern

// NewThemeManager creates theme manager using universal pattern
// Replaces internal/theme/manager.go, internal/theme/manager_complete.go, internal/managers/base.go ThemeManager.
func NewThemeManager(verbose bool) *UniversalManager {
	return NewThemeManagerWithDryRun(verbose, false)
}

// NewThemeManagerWithDryRun creates a theme manager with dry run option.
func NewThemeManagerWithDryRun(verbose bool, dryRun bool) *UniversalManager {
	return NewUniversalManager(UniversalConfig{
		Name:      "theme",
		Type:      TypeTheme,
		Available: []string{"tokyo-night", "catppuccin", "nord", "everforest", "gruvbox", "kanagawa", "rose-pine", "gruvbox-light"},
		Verbose:   verbose,
		DryRun:    dryRun,
		Handlers: map[string]func(context.Context, string) error{
			"default": func(ctx context.Context, theme string) error {
				return applyThemeHandlerWithDryRun(ctx, theme, dryRun)
			},
		},
	})
}

// NewFontManager creates font manager using universal pattern
// Replaces internal/font/manager.go, internal/managers/base.go FontManager.
func NewFontManager(verbose bool) *UniversalManager {
	return NewFontManagerWithDryRun(verbose, false)
}

// NewFontManagerWithDryRun creates a font manager with dry run option.
func NewFontManagerWithDryRun(verbose bool, dryRun bool) *UniversalManager {
	return NewUniversalManager(UniversalConfig{
		Name:      "font",
		Type:      TypeFont,
		Available: []string{"CaskaydiaMono", "FiraMono", "JetBrainsMono", "MesloLGS", "BerkeleyMono"},
		Verbose:   verbose,
		DryRun:    dryRun,
		Handlers: map[string]func(context.Context, string) error{
			"default": func(ctx context.Context, font string) error {
				return applyFontHandlerWithDryRun(ctx, font, dryRun)
			},
		},
	})
}

// NewSecurityManager creates security manager using universal pattern
// Replaces internal/security/manager.go.
func NewSecurityManager(verbose bool) *UniversalManager {
	return NewSecurityManagerWithDryRun(verbose, false)
}

// NewSecurityManagerWithDryRun creates a security manager with dry run option.
func NewSecurityManagerWithDryRun(verbose bool, dryRun bool) *UniversalManager {
	return NewUniversalManager(UniversalConfig{
		Name:      "security",
		Type:      TypeSecurity,
		Available: []string{"audit", "firewall", "fail2ban", "clamav", "rkhunter", "aide"},
		Verbose:   verbose,
		DryRun:    dryRun,
		Handlers: map[string]func(context.Context, string) error{
			"default": func(ctx context.Context, tool string) error {
				return runSecurityToolHandlerWithDryRun(ctx, tool, dryRun)
			},
		},
	})
}

// NewInstallManager creates install manager using universal pattern
// Replaces complex app installation scripts.
func NewInstallManager(verbose bool) *UniversalManager {
	groups := []string{"development", "browsers", "communication", "media", "productivity", "graphics", "utilities", "terminal"}

	return NewUniversalManager(UniversalConfig{
		Name:      "install",
		Type:      TypeInstall,
		Available: groups,
		Verbose:   verbose,
		Handlers: map[string]func(context.Context, string) error{
			"default": installAppHandler,
		},
	})
}

// NewVerifyManager creates verify manager using universal pattern
// Replaces internal/verify/manager.go.
func NewVerifyManager(verbose bool) *UniversalManager {
	return NewUniversalManager(UniversalConfig{
		Name:      "verify",
		Type:      TypeVerify,
		Available: []string{"tools", "integrations", "path", "fish", "xdg", "versions", "all"},
		Verbose:   verbose,
		Handlers: map[string]func(context.Context, string) error{
			"tools":        verifyToolsHandler,
			"integrations": verifyIntegrationsHandler,
			"path":         verifyPathHandler,
			"fish":         verifyFishHandler,
			"xdg":          verifyXDGHandler,
			"versions":     verifyVersionsHandler,
			"all":          verifyAllHandler,
			"default":      verifyAllHandler,
		},
	})
}

// NewLogsManager creates logs manager using universal pattern
// Replaces internal/logs/manager.go.
func NewLogsManager(verbose bool) *UniversalManager {
	return NewUniversalManager(UniversalConfig{
		Name:      "logs",
		Type:      TypeLogs,
		Available: []string{"install", "progress", "precheck", "errors", "all"},
		Verbose:   verbose,
		Handlers: map[string]func(context.Context, string) error{
			"install":  showInstallLogsHandler,
			"progress": showProgressLogsHandler,
			"precheck": showPrecheckLogsHandler,
			"errors":   showErrorLogsHandler,
			"all":      showAllLogsHandler,
			"default":  showAllLogsHandler,
		},
	})
}

// NewProxyManager creates proxy manager using universal pattern
// Replaces internal/proxy/manager.go.
func NewProxyManager(verbose bool) *UniversalManager {
	return NewUniversalManager(UniversalConfig{
		Name:      "proxy",
		Type:      TypeProxy,
		Available: []string{"enable", "disable", "status", "configure"},
		Verbose:   verbose,
		Handlers: map[string]func(context.Context, string) error{
			"enable":    enableProxyHandler,
			"disable":   disableProxyHandler,
			"status":    showProxyStatusHandler,
			"configure": configureProxyHandler,
			"default":   showProxyStatusHandler,
		},
	})
}

// NewSSHManager creates SSH manager using universal pattern
// Replaces internal/ssh/manager.go.
func NewSSHManager(verbose bool) *UniversalManager {
	return NewUniversalManager(UniversalConfig{
		Name:      "ssh",
		Type:      TypeSSH,
		Available: []string{"github", "gitlab", "bitbucket", "custom"},
		Verbose:   verbose,
		Handlers: map[string]func(context.Context, string) error{
			"github":    setupGitHubSSHHandler,
			"gitlab":    setupGitLabSSHHandler,
			"bitbucket": setupBitbucketSSHHandler,
			"custom":    setupCustomSSHHandler,
			"default":   setupGitHubSSHHandler,
		},
	})
}

var (
	// ErrUnknownTarget indicates an unknown app, group, or language was specified.
	ErrUnknownTarget = errors.New("unknown app, group, or language")
	// ErrFail2BanNotActive indicates fail2ban service is not active.
	ErrFail2BanNotActive = errors.New("fail2ban service not active")
	// ErrUnknownSecurityTool indicates the security tool is not recognized.
	ErrUnknownSecurityTool = errors.New("unknown security tool")
	// ErrFishNotInstalled indicates the fish shell is not installed.
	ErrFishNotInstalled = errors.New("fish shell not installed")
	// ErrProxyNotImplemented indicates proxy configuration is not implemented.
	ErrProxyNotImplemented = errors.New("proxy configuration not implemented")
	// ErrInteractiveProxyNotImpl indicates interactive proxy configuration is not implemented.
	ErrInteractiveProxyNotImpl = errors.New("interactive proxy configuration not implemented")
	// ErrCustomSSHNotImpl indicates custom SSH configuration is not implemented.
	ErrCustomSSHNotImpl = errors.New("custom SSH configuration not implemented")
)

// Handler implementations for universal managers
// These consolidate the actual implementation logic from separate manager files

func applyThemeHandler(ctx context.Context, theme string) error {
	return applyThemeHandlerWithDryRun(ctx, theme, false)
}

func applyThemeHandlerWithDryRun(ctx context.Context, theme string, dryRun bool) error {
	// Create theme service with dependencies
	fileManager := platform.NewFileManager(false)
	commandRunner := platform.NewCommandRunner(false, dryRun)
	configPath := config.GetXDGConfigHome()
	themesPath := filepath.Join(config.GetKareiPath(), "themes")

	themeService := application.NewThemeService(fileManager, commandRunner, configPath, themesPath)

	// Apply theme using the service
	if err := themeService.ApplyTheme(ctx, theme); err != nil {
		console.DefaultOutput.Warningf("Failed to apply theme: %v", err)
		return err
	}

	console.DefaultOutput.Successf("Theme '%s' applied successfully", theme)

	return nil
}

func applyFontHandler(ctx context.Context, font string) error {
	return applyFontHandlerWithDryRun(ctx, font, false)
}

func applyFontHandlerWithDryRun(ctx context.Context, font string, dryRun bool) error {
	// Create font service with dependencies
	fileManager := platform.NewFileManager(false)
	commandRunner := platform.NewCommandRunner(false, dryRun)

	home, _ := os.UserHomeDir()
	fontsDir := filepath.Join(home, ".local", "share", "fonts")
	configDir := config.GetXDGConfigHome()

	// Create network client
	networkClient := platform.NewNetworkAdapter()

	fontService := application.NewFontService(fileManager, commandRunner, networkClient, fontsDir, configDir)

	// Download and install font
	if err := fontService.DownloadAndInstallFont(ctx, font); err != nil {
		console.DefaultOutput.Warningf("Failed to install font: %v", err)
		return err
	}

	// Apply system font
	if err := fontService.ApplySystemFont(ctx, font); err != nil {
		console.DefaultOutput.Warningf("Failed to apply system font: %v", err)
		return err
	}

	console.DefaultOutput.Successf("Font '%s' applied successfully", font)

	return nil
}

func installAppHandler(ctx context.Context, target string) error {
	manager := apps.NewManager(true)

	// Check if it's a group
	if _, exists := apps.Groups[target]; exists {
		fmt.Printf("Installing group: %s\n", target)

		return manager.InstallGroup(ctx, target)
	}

	// Check if it's an app
	if _, exists := apps.Apps[target]; exists {
		fmt.Printf("Installing app: %s\n", target)

		return manager.InstallApp(ctx, target)
	}

	// Check if it's a language
	if _, exists := apps.Languages[target]; exists {
		fmt.Printf("Installing language: %s (latest)\n", target)

		return manager.InstallLanguage(ctx, target, "latest")
	}

	return fmt.Errorf("%w: %s", ErrUnknownTarget, target)
}

func runSecurityToolHandler(ctx context.Context, tool string) error {
	return runSecurityToolHandlerWithDryRun(ctx, tool, false)
}

func runSecurityToolHandlerWithDryRun(ctx context.Context, tool string, dryRun bool) error {
	executor := NewCommandExecutor(true, dryRun)
	serviceController := NewServiceController(true, dryRun)

	switch tool {
	case "audit":
		return executor.ExecuteSudo(ctx, "auditctl", "-l")
	case "firewall":
		return executor.ExecuteSudo(ctx, "ufw", "status", "verbose")
	case "fail2ban":
		if serviceController.IsActive(ctx, "fail2ban") {
			return executor.ExecuteSudo(ctx, "fail2ban-client", "status")
		}

		return ErrFail2BanNotActive
	case "clamav":
		return executor.Execute(ctx, "clamscan", "--version")
	case "rkhunter":
		return executor.ExecuteSudo(ctx, "rkhunter", "--check", "--report-warnings-only")
	case "aide":
		return executor.ExecuteSudo(ctx, "aide", "--check")
	default:
		return fmt.Errorf("%w: %s", ErrUnknownSecurityTool, tool)
	}
}

func verifyToolsHandler(ctx context.Context, _ string) error {
	// Core tools and their installation methods
	tools := map[string]string{
		"git":      "apt",
		"fish":     "apt",
		"starship": "aqua",
		"zellij":   "aqua",
		"btop":     "apt",
		"neovim":   "apt",
		"lazygit":  "aqua",
	}

	executor := NewCommandExecutor(false, false)

	console.DefaultOutput.Progressf("Verifying tools...")

	for tool, method := range tools {
		var isInstalled bool

		// Check installation based on method
		switch method {
		case "aqua":
			// Check if aqua is available and tool is installed via aqua with proper AQUA_ROOT_DIR
			if executor.CommandExists("aqua") {
				userLocal := filepath.Dir(config.GetUserBinDir())
				cmd := exec.CommandContext(ctx, "aqua", "which", tool) //nolint:gosec
				cmd.Env = os.Environ()
				cmd.Env = append(cmd.Env, "AQUA_ROOT_DIR="+userLocal)
				isInstalled = cmd.Run() == nil
			} else {
				// Fallback to checking if command exists in PATH
				isInstalled = executor.CommandExists(tool)
			}
		default:
			isInstalled = executor.CommandExists(tool)
		}

		printVerificationStatus(tool, isInstalled)
	}

	return nil
}

func verifyIntegrationsHandler(_ context.Context, _ string) error {
	console.DefaultOutput.Progressf("Verifying integrations...")
	// Check if Karei configs are properly linked
	configs := map[string]string{
		"fish":     filepath.Join(config.GetXDGConfigHome(), "fish", "config.fish"),
		ghosttyApp: filepath.Join(config.GetXDGConfigHome(), ghosttyApp, "config"),
		"btop":     filepath.Join(config.GetXDGConfigHome(), "btop", "btop.conf"),
	}

	for name, path := range configs {
		exists := system.FileExists(path)
		printConfigStatus(name, exists)
	}

	return nil
}

func verifyPathHandler(_ context.Context, _ string) error {
	console.DefaultOutput.Progressf("Verifying PATH...")

	userBin := config.GetUserBinDir()

	pathEnv := os.Getenv("PATH")
	inPath := strings.Contains(pathEnv, userBin)
	printPathStatus(userBin, inPath)

	return nil
}

func verifyFishHandler(_ context.Context, _ string) error {
	console.DefaultOutput.Progressf("Verifying Fish shell...")

	executor := NewCommandExecutor(false, false)

	if !executor.CommandExists("fish") {
		return ErrFishNotInstalled
	}

	fishConfig := filepath.Join(config.GetXDGConfigHome(), "fish", "config.fish")
	configExists := system.FileExists(fishConfig)
	printFishConfigStatus(configExists)

	return nil
}

func verifyXDGHandler(_ context.Context, _ string) error {
	console.DefaultOutput.Progressf("Verifying XDG directories...")

	dirs := map[string]string{
		"CONFIG": config.GetXDGConfigHome(),
		"DATA":   config.GetXDGDataHome(),
	}

	for name, dir := range dirs {
		outputXDGDirStatus(name, dir, system.IsDir(dir))
	}

	return nil
}

func outputXDGDirStatus(name, dir string, exists bool) {
	keyName := "xdg-" + strings.ToLower(name) + "-home"

	if console.DefaultOutput.JSON {
		// Handle JSON mode in verifyAllHandler
		return
	}

	if exists {
		outputXDGSuccess(name, dir, keyName)
	} else {
		outputXDGFailure(name, dir, keyName)
	}
}

func outputXDGSuccess(name, dir, keyName string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainKeyValue(keyName, dir)
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✓ XDG_%s_HOME: %s", name, dir))
	}
}

func outputXDGFailure(name, dir, keyName string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus(keyName, "missing")
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✗ XDG_%s_HOME: %s (not found)", name, dir))
	}
}

func verifyVersionsHandler(ctx context.Context, _ string) error {
	console.DefaultOutput.Progressf("Verifying versions...")

	executor := NewCommandExecutor(false, false)

	tools := map[string][]string{
		"Git":      {"git", "--version"},
		"Fish":     {"fish", "--version"},
		"Starship": {"starship", "--version"},
		"Neovim":   {"nvim", "--version"},
	}

	for name, cmd := range tools {
		output, err := executor.ExecuteWithOutput(ctx, cmd[0], cmd[1:]...)
		outputVersionStatus(name, output, err)
	}

	return nil
}

func outputVersionStatus(name, output string, err error) {
	keyName := strings.ToLower(name) + "-version"

	if console.DefaultOutput.JSON {
		// Handle JSON mode in verifyAllHandler
		return
	}

	if err == nil {
		version := strings.Split(output, "\n")[0]
		outputVersionSuccess(name, version, keyName)
	} else {
		outputVersionFailure(name, keyName)
	}
}

func outputVersionSuccess(name, version, keyName string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainKeyValue(keyName, version)
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✓ %s: %s", name, version))
	}
}

func outputVersionFailure(name, keyName string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus(keyName, "failed")
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✗ %s: version check failed", name))
	}
}

func verifyAllHandler(ctx context.Context, _ string) error {
	if console.DefaultOutput.JSON {
		// Collect all verification data for JSON output
		result := map[string]any{
			"tools":        collectToolStatus(ctx),
			"integrations": collectIntegrationStatus(),
			"path":         collectPathStatus(),
			"fish":         collectFishStatus(),
			"xdg":          collectXDGStatus(),
			"versions":     collectVersionStatus(ctx),
		}
		console.DefaultOutput.JSONResult("success", result)

		return nil
	}

	handlers := []func(context.Context, string) error{
		verifyToolsHandler,
		verifyIntegrationsHandler,
		verifyPathHandler,
		verifyFishHandler,
		verifyXDGHandler,
		verifyVersionsHandler,
	}

	for _, handler := range handlers {
		if err := handler(ctx, ""); err != nil {
			return err
		}

		if !console.DefaultOutput.Plain {
			fmt.Fprintf(os.Stderr, "\n") // Add spacing only in human mode
		}
	}

	return nil
}

// Helper functions for JSON verification data collection.
func collectToolStatus(ctx context.Context) map[string]string {
	// Use same tool list as verifyToolsHandler
	tools := map[string]string{
		"git":      "apt",
		"fish":     "apt",
		"starship": "aqua",
		"zellij":   "aqua",
		"btop":     "apt",
		"neovim":   "apt",
		"lazygit":  "aqua",
	}

	executor := NewCommandExecutor(false, false)
	status := make(map[string]string)

	for tool, method := range tools {
		var isInstalled bool

		switch method {
		case "aqua":
			if executor.CommandExists("aqua") {
				userLocal := filepath.Dir(config.GetUserBinDir())
				cmd := exec.CommandContext(ctx, "aqua", "which", tool) //nolint:gosec
				cmd.Env = os.Environ()
				cmd.Env = append(cmd.Env, "AQUA_ROOT_DIR="+userLocal)
				isInstalled = cmd.Run() == nil
			} else {
				isInstalled = executor.CommandExists(tool)
			}
		default:
			isInstalled = executor.CommandExists(tool)
		}

		if isInstalled {
			status[tool] = "installed"
		} else {
			status[tool] = "missing"
		}
	}

	return status
}

func collectIntegrationStatus() map[string]string {
	configs := map[string]string{
		"fish":     filepath.Join(config.GetXDGConfigHome(), "fish", "config.fish"),
		ghosttyApp: filepath.Join(config.GetXDGConfigHome(), ghosttyApp, "config"),
		"btop":     filepath.Join(config.GetXDGConfigHome(), "btop", "btop.conf"),
	}
	status := make(map[string]string)

	for name, path := range configs {
		if system.FileExists(path) {
			status[name] = "found"
		} else {
			status[name] = "missing"
		}
	}

	return status
}

func collectPathStatus() map[string]any {
	userBin := config.GetUserBinDir()
	pathEnv := os.Getenv("PATH")

	return map[string]any{
		"user_bin_dir": userBin,
		"in_path":      strings.Contains(pathEnv, userBin),
	}
}

func collectFishStatus() map[string]any {
	executor := NewCommandExecutor(false, false)
	fishConfig := filepath.Join(config.GetXDGConfigHome(), "fish", "config.fish")

	return map[string]any{
		"installed":     executor.CommandExists("fish"),
		"config_exists": system.FileExists(fishConfig),
	}
}

func collectXDGStatus() map[string]string {
	dirs := map[string]string{
		"config": config.GetXDGConfigHome(),
		"data":   config.GetXDGDataHome(),
	}
	status := make(map[string]string)

	for name, dir := range dirs {
		status[name] = dir
	}

	return status
}

func collectVersionStatus(ctx context.Context) map[string]string {
	tools := map[string][]string{
		"git":      {"git", "--version"},
		"fish":     {"fish", "--version"},
		"starship": {"starship", "--version"},
		"neovim":   {"nvim", "--version"},
	}
	executor := NewCommandExecutor(false, false)
	status := make(map[string]string)

	for name, cmd := range tools {
		if output, err := executor.ExecuteWithOutput(ctx, cmd[0], cmd[1:]...); err == nil {
			version := strings.Split(output, "\n")[0]
			status[name] = version
		} else {
			status[name] = "check_failed"
		}
	}

	return status
}

func showInstallLogsHandler(ctx context.Context, _ string) error {
	logPath := filepath.Join(config.GetXDGDataHome(), "karei", "install.log")

	return showLogFile(ctx, logPath, "Installation")
}

func showProgressLogsHandler(ctx context.Context, _ string) error {
	logPath := filepath.Join(config.GetXDGDataHome(), "karei", "progress.log")

	return showLogFile(ctx, logPath, "Progress")
}

func showPrecheckLogsHandler(ctx context.Context, _ string) error {
	logPath := filepath.Join(config.GetXDGDataHome(), "karei", "precheck.log")

	return showLogFile(ctx, logPath, "Precheck")
}

func showErrorLogsHandler(ctx context.Context, _ string) error {
	logPath := filepath.Join(config.GetXDGDataHome(), "karei", "errors.log")

	return showLogFile(ctx, logPath, "Errors")
}

func showAllLogsHandler(ctx context.Context, _ string) error { //nolint:unparam
	handlers := []func(context.Context, string) error{
		showInstallLogsHandler,
		showProgressLogsHandler,
		showPrecheckLogsHandler,
		showErrorLogsHandler,
	}

	for _, handler := range handlers {
		_ = handler(ctx, "")

		fmt.Println()
	}

	return nil
}

func showLogFile(ctx context.Context, path, name string) error {
	fmt.Printf("▸ %s Logs (%s):\n", name, path)

	if !system.FileExists(path) {
		fmt.Printf("No %s logs found\n", strings.ToLower(name))

		return nil
	}

	executor := NewCommandExecutor(false, false)

	output, err := executor.ExecuteWithOutput(ctx, "tail", "-n", "20", path)
	if err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	fmt.Println(output)

	return nil
}

func enableProxyHandler(_ context.Context, _ string) error {
	fmt.Println("▸ Enabling proxy configuration...")
	// Implementation would set proxy environment variables and configs
	return ErrProxyNotImplemented
}

func disableProxyHandler(_ context.Context, _ string) error {
	fmt.Println("▸ Disabling proxy configuration...")
	// Implementation would remove proxy environment variables and configs
	return ErrProxyNotImplemented
}

func showProxyStatusHandler(_ context.Context, _ string) error { //nolint:unparam
	envVars := []string{"http_proxy", "https_proxy", "ftp_proxy", "no_proxy"}

	if console.DefaultOutput.JSON {
		status := make(map[string]string)

		for _, env := range envVars {
			if value := os.Getenv(env); value != "" {
				status[env] = value
			} else {
				status[env] = "unset"
			}
		}

		console.DefaultOutput.JSONResult("success", map[string]any{"proxy": status})

		return nil
	}

	console.DefaultOutput.Progressf("Proxy Status:")

	for _, env := range envVars {
		value := os.Getenv(env)
		outputProxyStatus(env, value)
	}

	return nil
}

func outputProxyStatus(env, value string) {
	if value != "" {
		outputProxySet(env, value)
	} else {
		outputProxyUnset(env)
	}
}

func outputProxySet(env, value string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainKeyValue(env, value)
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✓ %s: %s", env, value))
	}
}

func outputProxyUnset(env string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainKeyValue(env, "unset")
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✗ %s: not set", env))
	}
}

func configureProxyHandler(_ context.Context, _ string) error {
	fmt.Println("▸ Interactive proxy configuration...")
	// Implementation would prompt for proxy settings
	return ErrInteractiveProxyNotImpl
}

func setupGitHubSSHHandler(ctx context.Context, _ string) error {
	fmt.Println("▸ Setting up GitHub SSH key...")

	executor := NewCommandExecutor(true, false)

	// Check if SSH key already exists
	sshDir := filepath.Join(config.GetUserBinDir(), "..", "..", ".ssh")
	keyPath := filepath.Join(sshDir, "id_ed25519")

	if system.FileExists(keyPath) {
		fmt.Println("✓ SSH key already exists")

		return nil
	}

	// Generate new SSH key
	email := "user@example.com" // Would prompt user for email

	return executor.Execute(ctx, "ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
}

func setupGitLabSSHHandler(ctx context.Context, _ string) error {
	fmt.Println("▸ Setting up GitLab SSH key...")

	return setupGitHubSSHHandler(ctx, "") // Same implementation for now
}

func setupBitbucketSSHHandler(ctx context.Context, _ string) error {
	fmt.Println("▸ Setting up Bitbucket SSH key...")

	return setupGitHubSSHHandler(ctx, "") // Same implementation for now
}

func setupCustomSSHHandler(_ context.Context, _ string) error {
	fmt.Println("▸ Setting up custom SSH configuration...")

	return ErrCustomSSHNotImpl
}

// CommandFactories provides simplified constructors for all CLI commands
// Replaces 20+ separate CLI command files with unified factory pattern

// NewThemeCommand creates theme CLI command using universal pattern
// Replaces internal/cli/theme_native.go and internal/cli/theme_urfave.go.
func NewThemeCommand(verbose bool) *UniversalCommand {
	return NewUniversalCommand(CommandConfig{
		Name:  "theme",
		Usage: "Manage system themes",
		Description: `Apply coordinated themes across all applications including GNOME, terminal, editors, and browsers.

AVAILABLE THEMES:
  tokyo-night, catppuccin, nord, everforest, gruvbox, kanagawa, rose-pine, gruvbox-light

EXAMPLES:
  karei theme tokyo-night    Apply tokyo-night theme system-wide
  karei theme list           Show available themes with previews

DOCUMENTATION:
  https://github.com/janderssonse/karei/blob/master/docs/themes.md

TROUBLESHOOTING:
  If theme doesn't apply: https://github.com/janderssonse/karei/blob/master/docs/themes.md#troubleshooting`,
		Type:        TypeTheme,
		Available:   []string{"tokyo-night", "catppuccin", "nord", "everforest", "gruvbox", "kanagawa", "rose-pine", "gruvbox-light"},
		Interactive: true,
		Verbose:     verbose,
		Handlers: map[string]func(context.Context, string) error{
			"default": applyThemeHandler,
		},
	})
}

// NewFontCommand creates font CLI command using universal pattern
// Replaces internal/cli/font_native.go and internal/cli/font_urfave.go.
func NewFontCommand(verbose bool) *UniversalCommand {
	return NewUniversalCommand(CommandConfig{
		Name:  "font",
		Usage: "Manage system fonts",
		Description: `Install and configure programming fonts across terminal and editor applications.

AVAILABLE FONTS:
  CaskaydiaMono, FiraMono, JetBrainsMono, MesloLGS, BerkeleyMono

EXAMPLES:
  karei font JetBrainsMono    Install and apply JetBrains Mono font
  karei font list             Show available fonts with previews

DOCUMENTATION:
  https://github.com/janderssonse/karei/blob/master/docs/fonts.md

TROUBLESHOOTING:
  Font issues: https://github.com/janderssonse/karei/blob/master/docs/fonts.md#troubleshooting`,
		Type:        TypeFont,
		Available:   []string{"CaskaydiaMono", "FiraMono", "JetBrainsMono", "MesloLGS", "BerkeleyMono"},
		Interactive: true,
		Verbose:     verbose,
		Handlers: map[string]func(context.Context, string) error{
			"default": applyFontHandler,
		},
	})
}

// NewSecurityCommand creates security CLI command
// Replaces internal/cli/security_native.go.
func NewSecurityCommand(verbose bool) *UniversalCommand {
	return NewUniversalCommand(CommandConfig{
		Name:  "security",
		Usage: "Run security checks and tools",
		Description: `Execute security audits and configure monitoring tools.

AVAILABLE TOOLS:
  audit      Run system security audit
  firewall   Configure UFW firewall rules
  fail2ban   Setup intrusion detection
  clamav     Install antivirus scanner
  rkhunter   Rootkit detection tool
  aide       File integrity monitoring

EXAMPLES:
  karei security audit      Run security audit
  karei security firewall   Configure basic firewall protection

DOCUMENTATION:
  https://github.com/janderssonse/karei/blob/master/docs/security.md

WARNING:
  These tools modify system security settings. Review documentation before use.`,
		Type:        TypeSecurity,
		Available:   []string{"audit", "firewall", "fail2ban", "clamav", "rkhunter", "aide"},
		Interactive: true,
		Verbose:     verbose,
		Handlers: map[string]func(context.Context, string) error{
			"default": runSecurityToolHandler,
		},
	})
}

// NewVerifyCommand creates verify CLI command
// Replaces internal/cli/verify_native.go.
func NewVerifyCommand(verbose bool) *UniversalCommand {
	return NewUniversalCommand(CommandConfig{
		Name:        "verify",
		Usage:       "Verify system configuration",
		Description: "Run verification checks",
		Type:        TypeVerify,
		Available:   []string{"tools", "integrations", "path", "fish", "xdg", "versions", "all"},
		Interactive: true,
		Verbose:     verbose,
		Handlers: map[string]func(context.Context, string) error{
			"tools":        verifyToolsHandler,
			"integrations": verifyIntegrationsHandler,
			"path":         verifyPathHandler,
			"fish":         verifyFishHandler,
			"xdg":          verifyXDGHandler,
			"versions":     verifyVersionsHandler,
			"all":          verifyAllHandler,
			"default":      verifyAllHandler,
		},
	})
}

// NewLogsCommand creates logs CLI command
// Replaces internal/cli/logs_native.go.
func NewLogsCommand(verbose bool) *UniversalCommand {
	return NewUniversalCommand(CommandConfig{
		Name:        "logs",
		Usage:       "View system logs",
		Description: "Display Karei installation and operation logs",
		Type:        TypeLogs,
		Available:   []string{"install", "progress", "precheck", "errors", "all"},
		Interactive: true,
		Verbose:     verbose,
		Handlers: map[string]func(context.Context, string) error{
			"install":  showInstallLogsHandler,
			"progress": showProgressLogsHandler,
			"precheck": showPrecheckLogsHandler,
			"errors":   showErrorLogsHandler,
			"all":      showAllLogsHandler,
			"default":  showAllLogsHandler,
		},
	})
}

func printVerificationStatus(tool string, isInstalled bool) {
	if console.DefaultOutput.JSON {
		// Handle JSON mode in verifyAllHandler
		return
	}

	if isInstalled {
		printToolInstalled(tool)
	} else {
		printToolMissing(tool)
	}
}

func printToolInstalled(tool string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus(tool, "installed")
	} else {
		console.DefaultOutput.Result("✓ " + tool)
	}
}

func printToolMissing(tool string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus(tool, "missing")
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✗ %s - not found", tool))
	}
}

func printConfigStatus(name string, exists bool) {
	if console.DefaultOutput.JSON {
		// Handle JSON mode in verifyAllHandler
		return
	}

	if exists {
		printConfigFound(name)
	} else {
		printConfigMissing(name)
	}
}

func printConfigFound(name string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus(name+"-config", "found")
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✓ %s config", name))
	}
}

func printConfigMissing(name string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus(name+"-config", "missing")
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✗ %s config - not found", name))
	}
}

func printPathStatus(userBin string, inPath bool) {
	if console.DefaultOutput.JSON {
		// Handle JSON mode in verifyAllHandler
		return
	}

	if inPath {
		printPathFound()
	} else {
		printPathMissing(userBin)
	}
}

func printPathFound() {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus("user-bin-path", "found")
	} else {
		console.DefaultOutput.Result("✓ User bin directory in PATH")
	}
}

func printPathMissing(userBin string) {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus("user-bin-path", "missing")
	} else {
		console.DefaultOutput.Result("✗ User bin directory not in PATH: " + userBin)
	}
}

func printFishConfigStatus(configExists bool) {
	if console.DefaultOutput.JSON {
		// Handle JSON mode in verifyAllHandler
		return
	}

	if configExists {
		printFishConfigFound()
	} else {
		printFishConfigMissing()
	}
}

func printFishConfigFound() {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus("fish-config", "found")
	} else {
		console.DefaultOutput.Result("✓ Fish configuration found")
	}
}

func printFishConfigMissing() {
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainStatus("fish-config", "missing")
	} else {
		console.DefaultOutput.Result("✗ Fish configuration not found")
	}
}
