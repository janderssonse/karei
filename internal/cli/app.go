// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package cli provides command-line interface implementations.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	cliAdapter "github.com/janderssonse/karei/internal/adapters/cli"
	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/config"
	"github.com/janderssonse/karei/internal/console"
	"github.com/janderssonse/karei/internal/desktop"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/fonts"
	"github.com/janderssonse/karei/internal/patterns"
	"github.com/janderssonse/karei/internal/system"
	"github.com/janderssonse/karei/internal/tui"
	"github.com/janderssonse/karei/internal/uninstall"
	"github.com/urfave/cli/v3"
)

// Exit codes follow standard Unix conventions for better scripting support.
// Range 0-125 are safe to use (126+ have special meaning in shells).
const (
	// Standard Unix exit codes (0-10).
	ExitSuccess         = 0 // Operation completed successfully
	ExitGeneralError    = 1 // Generic failure (catch-all)
	ExitUsageError      = 2 // Invalid command line usage
	ExitConfigError     = 3 // Configuration file error
	ExitPermissionError = 4 // Permission denied
	ExitNotFoundError   = 5 // Requested resource not found

	// Network and system errors (10-19).
	ExitDependencyError = 10 // Missing dependency
	ExitNetworkError    = 11 // Network operation failed
	ExitSystemError     = 12 // System call failed
	ExitTimeoutError    = 13 // Operation timed out
	ExitInterruptError  = 14 // User interrupted (Ctrl+C)

	// Application-specific errors (20-29).
	ExitThemeError     = 20 // Theme operation failed
	ExitFontError      = 21 // Font operation failed
	ExitAppError       = 22 // App installation/removal failed
	ExitBackupError    = 23 // Backup operation failed
	ExitMigrationError = 24 // Migration failed

	// Warning (non-fatal issues occurred).
	ExitWarnings = 64 // Operation succeeded with warnings

	// CLI flags.
	HelpFlag = "--help"

	// Environment detection.
	unknownValue = "unknown"
)

var (
	// ErrNoPackagesSpecified is returned when no packages are provided for installation.
	ErrNoPackagesSpecified = errors.New("no packages specified")
	// ErrInvalidInput is returned when user input is malformed.
	ErrInvalidInput = errors.New("invalid input")
	// ErrInvalidChoice is returned when user makes an invalid selection.
	ErrInvalidChoice = errors.New("invalid choice")
	// ErrUnknownChoice is returned when user selects an unrecognized option.
	ErrUnknownChoice = errors.New("unknown choice")
	// ErrInvalidArgument is returned when a command argument is invalid.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrUnknownGroup is returned when a group is not found.
	ErrUnknownGroup = errors.New("unknown group")
	// ErrAppsFailedToInstall is returned when some apps fail to install.
	ErrAppsFailedToInstall = errors.New("apps failed to install")
	// ErrFontSizeChange is returned when font size cannot be changed.
	ErrFontSizeChange = errors.New("failed to change font size (check terminal configuration)")
	// ErrDesktopEntries is returned when desktop entries cannot be created.
	ErrDesktopEntries = errors.New("failed to create desktop entries (check ~/.local/share/applications/)")
	// ErrUpdateFailed is returned when update fails.
	ErrUpdateFailed = errors.New("failed to update (check network connection)")
)

// CLI provides a clean, composable command-line interface following hexagonal architecture.
// Eliminates 25+ separate CLI command files by using composition and factories.
type CLI struct {
	app     *cli.Command
	verbose bool
	json    bool
	quiet   bool
	plain   bool
	color   string        // "auto", "always", "never"
	timeout time.Duration // Network operation timeout
	yes     bool          // Auto-accept all prompts
}

// NewCLI creates a clean, idiomatic CLI interface
// Replaces: app.go, simplified.go, and all 20+ individual command files.
func NewCLI() *CLI {
	app := &CLI{}

	app.app = &cli.Command{
		Name:    "karei",
		Usage:   "The easiest way to set up Linux for development",
		Version: app.getVersion(),
		Suggest: true, // Enable command and flag suggestions
		Description: `Transforms fresh Linux installations into fully-configured development environments.

ESSENTIAL COMMANDS:
  install --packages git,vim     Install development tools
  theme apply --name tokyo-night  Apply coordinated themes
  verify                          Check what's installed and configured

QUICK START:
  karei install --packages git,vim     # Get essential tools
  karei theme apply --name tokyo-night  # Apply beautiful theme
  karei verify                          # Check setup

HELP & DOCS:
  karei help examples       # Complete workflows and tutorials
  karei help <command>      # Detailed command documentation  
  man karei                 # Unix manual pages
  https://docs.karei.org    # Web documentation (searchable)

SUPPORT:
  https://github.com/janderssonse/karei/issues    # Report bugs
  https://github.com/janderssonse/karei/discussions   # Ask questions`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "help",
				Usage:   "show help information",
				Aliases: []string{"h"},
			},
			&cli.BoolFlag{
				Name:        "verbose",
				Usage:       "show progress messages to stderr",
				Aliases:     []string{"v"},
				Destination: &app.verbose,
			},
			&cli.BoolFlag{
				Name:        "json",
				Usage:       "output structured JSON results",
				Aliases:     []string{"j"},
				Destination: &app.json,
			},
			&cli.BoolFlag{
				Name:        "quiet",
				Usage:       "suppress non-essential output",
				Aliases:     []string{"q"},
				Destination: &app.quiet,
			},
			&cli.BoolFlag{
				Name:        "plain",
				Usage:       "output plain text without formatting for scripts",
				Destination: &app.plain,
			},
			&cli.StringFlag{
				Name:        "color",
				Usage:       "color output mode: auto, always, never",
				Value:       "auto",
				Destination: &app.color,
			},
			&cli.DurationFlag{
				Name:        "timeout",
				Usage:       "timeout for network operations (0 = no timeout)",
				Value:       3 * time.Minute,
				Destination: &app.timeout,
			},
			&cli.BoolFlag{
				Name:        "yes",
				Aliases:     []string{"y"},
				Usage:       "automatically answer yes to all prompts",
				Destination: &app.yes,
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return app.initConfig(ctx, cmd)
		},
		Action:          app.defaultAction,
		Commands:        app.createAllCommands(),
		CommandNotFound: app.commandNotFound,
	}

	return app
}

// Run executes the CLI application.
func (app *CLI) Run(ctx context.Context, args []string) error {
	return app.app.Run(ctx, args)
}

// createAllCommands creates all CLI commands using universal factory pattern
// Replaces 20+ individual command constructor functions.
func (app *CLI) createAllCommands() []*cli.Command {
	// All commands created using universal factory - no duplicate code
	commands := []*patterns.UniversalCommand{
		patterns.NewSecurityCommand(app.verbose),
		patterns.NewVerifyCommand(app.verbose),
		patterns.NewLogsCommand(app.verbose),
	}

	// Convert universal commands to cli.Command using adapter pattern
	cliCommands := make([]*cli.Command, 0, len(commands)+5) // +5 for special commands
	for _, cmd := range commands {
		cliCommands = append(cliCommands, app.adaptUniversalCommand(cmd))
	}

	// Add remaining commands that don't fit universal pattern yet
	cliCommands = append(cliCommands, app.createSpecialCommands()...)

	return cliCommands
}

// adaptUniversalCommand converts UniversalCommand to cli.Command
// This adapter eliminates the need for separate CLI command implementations.
func (app *CLI) adaptUniversalCommand(ucmd *patterns.UniversalCommand) *cli.Command {
	return &cli.Command{
		Name:        ucmd.Name,
		Usage:       ucmd.Usage,
		Description: ucmd.Description,
		ArgsUsage:   "[option]",
		Suggest:     true, // Enable suggestions for subcommands too
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()

			return ucmd.Execute(ctx, args)
		},
	}
}

// createSpecialCommands creates commands that don't fit the universal pattern
// These would eventually be converted to universal pattern as well.
func (app *CLI) createSpecialCommands() []*cli.Command {
	return []*cli.Command{
		app.createThemeCommand(),
		app.createFontCommand(),
		app.createInstallCommand(),
		app.createUpdateCommand(),
		app.createUninstallCommand(),
		app.createListCommand(),
		app.createSetupCommand(),
		app.createAppsCommand(),
		app.createDesktopCommand(),
		app.createMenuCommand(),
		app.createVersionCommand(),
		app.createFontSizeCommand(),
		app.createHelpCommand(),
		app.createStatusCommand(),
		app.createTUICommand(),
	}
}

// createInstallCommand creates install command with flag-based interface.
func (app *CLI) createInstallCommand() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "Install development tools and applications",
		Description: `Install packages, tools, or application groups.

Groups available:
  essential    - Core tools (git, vim, curl, wget)
  development  - Development tools (docker, nodejs, python)
  productivity - Productivity apps (obsidian, notion)
  
Examples:
  karei install --packages git,vim      # Install specific packages
  karei install --group development      # Install development group
  karei install --packages git --json   # Output JSON results`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "packages",
				Aliases: []string{"p"},
				Usage:   "comma-separated list of packages to install",
			},
			&cli.StringFlag{
				Name:    "group",
				Aliases: []string{"g"},
				Usage:   "install a predefined group of packages (essential, development, productivity)",
			},
		},
		Action: app.handleInstallAction,
	}
}

func (app *CLI) handleInstallAction(ctx context.Context, cmd *cli.Command) error {
	return app.runInstall(ctx, cmd)
}

// runInstall handles the install command execution with output adapter.
func (app *CLI) runInstall(ctx context.Context, cmd *cli.Command) error {
	// Apply global timeout if specified
	if app.timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, app.timeout)
		defer cancel()
	}

	// Create output adapter based on flags
	output := cliAdapter.OutputFromContext(app.json, app.quiet)

	// Check for help flag first
	if cmd.Bool("help") {
		app.showInstallHelp(output)
		return nil
	}

	// Validate flags
	packagesFlag, groupFlag, err := app.validateInstallFlags(cmd)
	if err != nil {
		return err
	}

	// Track installation time
	startTime := time.Now()
	manager := apps.NewManager(app.verbose)

	// Prepare result tracking
	result := &domain.InstallResult{
		Installed: []string{},
		Failed:    []string{},
		Skipped:   []string{},
		Timestamp: startTime,
	}

	// Process group installation
	if groupFlag != "" {
		app.installGroupWithOutput(ctx, manager, groupFlag, result, output)
	}

	// Process individual packages
	if packagesFlag != "" {
		packages := strings.Split(packagesFlag, ",")
		for _, pkg := range packages {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" {
				app.installSingleWithOutput(ctx, manager, pkg, result, output)
			}
		}
	}

	// Calculate duration
	result.Duration = time.Since(startTime)

	// Output results
	if err := app.outputInstallResults(result, output); err != nil {
		return domain.NewExitError(ExitGeneralError, "failed to output results", err)
	}

	// Return appropriate exit code
	return app.getInstallExitCode(result)
}

// installGroupWithOutput installs a group of applications with output support.
func (app *CLI) installGroupWithOutput(ctx context.Context, manager *apps.Manager, group string, result *domain.InstallResult, output domain.OutputPort) {
	groupApps, exists := apps.Groups[group]
	if !exists {
		_ = output.Error("Unknown group: " + group)
		result.Failed = append(result.Failed, group)

		return
	}

	_ = output.Progress(fmt.Sprintf("Installing %s group (%d apps)...", group, len(groupApps)))

	groupInstalled := 0

	for _, appName := range groupApps {
		_ = output.Progress(fmt.Sprintf("  Installing %s...", appName))

		if err := manager.InstallApp(ctx, appName); err != nil {
			// Group errors are shown inline, use simpler format
			_ = output.Info("  ✗ Failed to install " + appName)
			result.Failed = append(result.Failed, appName)
		} else {
			_ = output.Info("  ✓ Installed " + appName)
			result.Installed = append(result.Installed, appName)
			groupInstalled++
		}
	}

	switch {
	case groupInstalled == len(groupApps):
		_ = output.Success(fmt.Sprintf("✓ %s group installed successfully (%d apps)", group, groupInstalled), nil)
	case groupInstalled > 0:
		_ = output.Info(fmt.Sprintf("⚠ %s group partially installed (%d/%d apps)", group, groupInstalled, len(groupApps)))
	default:
		_ = output.Error("✗ Failed to install " + group + " group")
	}
}

// installSingleWithOutput installs a single application with output support.
func (app *CLI) installSingleWithOutput(ctx context.Context, manager *apps.Manager, pkg string, result *domain.InstallResult, output domain.OutputPort) {
	// Notify about boundary crossing operations
	if !output.IsQuiet() {
		_ = output.Info("→ Installing " + pkg)
		// Keep progress simple - users don't need internal steps
		if app.verbose {
			_ = output.Progress("  Preparing installation...")
		}
	}

	if err := manager.InstallApp(ctx, pkg); err != nil {
		errorMsg := domain.FormatErrorMessage(err, pkg, app.verbose)
		_ = output.Error(errorMsg)

		result.Failed = append(result.Failed, pkg)
	} else {
		_ = output.Success(fmt.Sprintf("✓ Installed %s successfully", pkg), nil)

		// No need to mention PATH - users know what "installed" means

		result.Installed = append(result.Installed, pkg)
	}
}

// outputInstallResults outputs the installation results using the output adapter.
func (app *CLI) outputInstallResults(result *domain.InstallResult, output domain.OutputPort) error {
	// For JSON output, send the structured result
	if app.json {
		return output.Success("", result)
	}

	// For text output, provide summary
	if len(result.Installed) > 0 || len(result.Failed) > 0 {
		summary := app.buildResultSummary(
			len(result.Installed),
			len(result.Failed),
			len(result.Skipped),
			"installed",
			result.Duration,
		)

		return output.Success(summary, nil)
	}

	return nil
}

// buildResultSummary creates a summary string for operation results.
func (app *CLI) buildResultSummary(successCount, failedCount, skippedCount int, successLabel string, duration time.Duration) string {
	totalAttempted := successCount + failedCount + skippedCount

	var summary strings.Builder
	if successCount > 0 {
		summary.WriteString(fmt.Sprintf("Successfully %s %d/%d packages", successLabel, successCount, totalAttempted))
	}

	if failedCount > 0 {
		if summary.Len() > 0 {
			summary.WriteString(", ")
		}

		summary.WriteString(fmt.Sprintf("%d failed", failedCount))
	}

	if skippedCount > 0 {
		if summary.Len() > 0 {
			summary.WriteString(", ")
		}

		summary.WriteString(fmt.Sprintf("%d skipped", skippedCount))
	}

	// Add duration
	summary.WriteString(fmt.Sprintf(" (%.2fs)", duration.Seconds()))

	return summary.String()
}

// getInstallExitCode returns the appropriate exit code based on results.
func (app *CLI) getInstallExitCode(result *domain.InstallResult) error {
	if len(result.Failed) > 0 && len(result.Installed) == 0 {
		msg := "All installations failed. Common causes:\n"
		msg += "  • Network issues - check your connection\n"
		msg += "  • Permission denied - try with sudo\n"

		msg += "  • Package not found - verify package names\n"
		if !app.verbose {
			msg += "Run with --verbose for detailed errors"
		}

		return domain.NewExitError(ExitAppError, msg, nil)
	} else if len(result.Failed) > 0 {
		return domain.NewExitError(ExitWarnings, fmt.Sprintf("%d packages failed to install", len(result.Failed)), nil)
	}

	return nil
}

// isKnownGroup checks if the package name is a known application group.
// createUpdateCommand creates update command.
func (app *CLI) createUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update Karei",
		Action: func(ctx context.Context, _ *cli.Command) error {
			// Apply global timeout if specified
			if app.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, app.timeout)
				defer cancel()
			}

			executor := patterns.NewCommandExecutor(app.verbose, false)
			kareiPath := config.GetKareiPath()

			// Check if git is available
			if !system.CommandExists("git") {
				return domain.NewExitError(ExitDependencyError, "git is not installed", nil)
			}

			// Check if Karei directory exists
			if !system.IsDir(kareiPath) {
				return domain.NewExitError(ExitConfigError, "Karei installation directory not found", nil)
			}

			// Explicit boundary crossing notification
			fmt.Printf("• Connecting to remote Git repository...\n")
			fmt.Printf("  This will download updates from the internet\n")
			fmt.Printf("  Repository: https://github.com/janderssonse/karei.git\n")

			console.DefaultOutput.Progressf("Updating Karei...")
			if err := executor.Execute(ctx, "git", "-C", kareiPath, "pull"); err != nil {
				return domain.NewExitError(ExitNetworkError, "failed to update from git", err)
			}

			fmt.Printf("✓ Karei updated successfully from remote repository\n")
			fmt.Printf("  Changes have been applied to: %s\n", kareiPath)

			console.DefaultOutput.SuccessResult("updated", "Karei updated successfully")

			return nil
		},
	}
}

// createThemeCommand creates theme command with subcommands.
func (app *CLI) createThemeCommand() *cli.Command {
	return &cli.Command{
		Name:  "theme",
		Usage: "Manage system themes",
		Description: `Apply coordinated themes across all applications.

Available themes:
  tokyo-night, catppuccin, nord, everforest, gruvbox, kanagawa, rose-pine, gruvbox-light`,
		Commands: []*cli.Command{
			{
				Name:  "apply",
				Usage: "Apply a theme system-wide",
				Description: `Apply a coordinated theme across all applications including GNOME, terminal, editors, and browsers.

Examples:
  karei theme apply --name tokyo-night    # Apply tokyo-night theme
  karei theme apply -n catppuccin        # Short form`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    "name of the theme to apply",
						Required: true,
					},
				},
				Action: app.runThemeApply,
			},
			{
				Name:  "list",
				Usage: "List available themes",
				Description: `Show all available themes with their current status.

Examples:
  karei theme list           # List all themes
  karei theme list --json    # Output as JSON`,
				Action: app.runThemeList,
			},
			{
				Name:  "current",
				Usage: "Show current theme",
				Description: `Display the currently active theme.

Examples:
  karei theme current        # Show current theme
  karei theme current --json # Output as JSON`,
				Action: app.runThemeCurrent,
			},
		},
	}
}

// createUninstallCommand creates uninstall command with flag-based interface.
func (app *CLI) createUninstallCommand() *cli.Command {
	return &cli.Command{
		Name:  "uninstall",
		Usage: "Uninstall packages",
		Description: `Uninstall packages from the system.

Examples:
  karei uninstall --packages vim,git    # Uninstall specific packages
  karei uninstall -p docker,nodejs      # Short form`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "packages",
				Aliases: []string{"p"},
				Usage:   "comma-separated list of packages to uninstall",
			},
		},
		Action: app.runUninstall,
	}
}

// runUninstall handles the uninstall command execution with output adapter.
func (app *CLI) runUninstall(ctx context.Context, cmd *cli.Command) error {
	// Apply global timeout if specified
	if app.timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, app.timeout)
		defer cancel()
	}

	// Create output adapter based on flags
	output := cliAdapter.OutputFromContext(app.json, app.quiet)

	// Get packages from flag
	packagesFlag := cmd.String("packages")
	if packagesFlag == "" {
		return domain.NewExitError(ExitUsageError, "specify --packages flag with comma-separated list of packages", nil)
	}

	// Track uninstallation time
	startTime := time.Now()
	uninstaller := uninstall.NewUninstaller(app.verbose)

	// Prepare result tracking
	result := &domain.UninstallResult{
		Uninstalled: []string{},
		Failed:      []string{},
		NotFound:    []string{},
		Timestamp:   startTime,
	}

	// Process each package from the flag
	packages := strings.Split(packagesFlag, ",")
	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		_ = output.Progress(fmt.Sprintf("Uninstalling %s...", pkg))

		if err := uninstaller.UninstallSpecial(ctx, pkg); err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not installed") {
				_ = output.Info(fmt.Sprintf("⚠ %s not installed", pkg))
				result.NotFound = append(result.NotFound, pkg)
			} else {
				errorMsg := domain.FormatErrorMessage(err, pkg, app.verbose)
				_ = output.Error(errorMsg)

				result.Failed = append(result.Failed, pkg)
			}

			continue
		}

		_ = output.Success("✓ Uninstalled "+pkg, nil)
		result.Uninstalled = append(result.Uninstalled, pkg)
	}

	// Calculate duration
	result.Duration = time.Since(startTime)

	// Output results
	if err := app.outputUninstallResults(result, output); err != nil {
		return domain.NewExitError(ExitGeneralError, "failed to output results", err)
	}

	// Return appropriate exit code
	return app.getUninstallExitCode(result)
}

// runThemeApply handles the theme apply subcommand.
func (app *CLI) runThemeApply(ctx context.Context, cmd *cli.Command) error {
	themeName := cmd.String("name")

	// Create theme manager
	themeCmd := patterns.NewThemeCommand(app.verbose)

	// Apply the theme
	return themeCmd.Execute(ctx, []string{themeName})
}

// runThemeList handles the theme list subcommand.
func (app *CLI) runThemeList(ctx context.Context, _ *cli.Command) error {
	// Create theme manager
	themeCmd := patterns.NewThemeCommand(app.verbose)

	// List themes
	return themeCmd.Execute(ctx, []string{"list"})
}

// runThemeCurrent handles the theme current subcommand.
func (app *CLI) runThemeCurrent(ctx context.Context, _ *cli.Command) error {
	// Create theme manager
	themeCmd := patterns.NewThemeCommand(app.verbose)

	// Show current theme (empty args shows status)
	return themeCmd.Execute(ctx, []string{})
}

// createFontCommand creates font command with subcommands.
func (app *CLI) createFontCommand() *cli.Command {
	return &cli.Command{
		Name:  "font",
		Usage: "Manage system fonts",
		Description: `Install and configure programming fonts across terminal and editor applications.

Available fonts:
  CaskaydiaMono, FiraMono, JetBrainsMono, MesloLGS, BerkeleyMono`,
		Commands: []*cli.Command{
			{
				Name:  "install",
				Usage: "Install and apply a font",
				Description: `Install and configure a programming font system-wide.

Examples:
  karei font install --name JetBrainsMono  # Install JetBrains Mono
  karei font install -n FiraMono           # Short form`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    "name of the font to install",
						Required: true,
					},
				},
				Action: app.runFontInstall,
			},
			{
				Name:  "list",
				Usage: "List available fonts",
				Description: `Show all available fonts with their installation status.

Examples:
  karei font list           # List all fonts
  karei font list --json    # Output as JSON`,
				Action: app.runFontList,
			},
			{
				Name:  "current",
				Usage: "Show current font",
				Description: `Display the currently active font.

Examples:
  karei font current        # Show current font
  karei font current --json # Output as JSON`,
				Action: app.runFontCurrent,
			},
		},
	}
}

// runFontInstall handles the font install subcommand.
func (app *CLI) runFontInstall(ctx context.Context, cmd *cli.Command) error {
	fontName := cmd.String("name")

	// Create font manager
	fontCmd := patterns.NewFontCommand(app.verbose)

	// Install the font
	return fontCmd.Execute(ctx, []string{fontName})
}

// runFontList handles the font list subcommand.
func (app *CLI) runFontList(ctx context.Context, _ *cli.Command) error {
	// Create font manager
	fontCmd := patterns.NewFontCommand(app.verbose)

	// List fonts
	return fontCmd.Execute(ctx, []string{"list"})
}

// runFontCurrent handles the font current subcommand.
func (app *CLI) runFontCurrent(ctx context.Context, _ *cli.Command) error {
	// Create font manager
	fontCmd := patterns.NewFontCommand(app.verbose)

	// Show current font (empty args shows status)
	return fontCmd.Execute(ctx, []string{})
}

// outputUninstallResults outputs the uninstallation results using the output adapter.
func (app *CLI) outputUninstallResults(result *domain.UninstallResult, output domain.OutputPort) error {
	// For JSON output, send the structured result
	if app.json {
		return output.Success("", result)
	}

	// For text output, provide summary
	if len(result.Uninstalled) > 0 || len(result.Failed) > 0 || len(result.NotFound) > 0 {
		// Use NotFound as the "skipped" count for uninstall
		summary := app.buildResultSummary(
			len(result.Uninstalled),
			len(result.Failed),
			len(result.NotFound),
			"uninstalled",
			result.Duration,
		)
		// Override the "skipped" text with "not found" for clarity
		summary = strings.Replace(summary, "skipped", "not found", 1)

		return output.Success(summary, nil)
	}

	return nil
}

// getUninstallExitCode returns the appropriate exit code based on results.
func (app *CLI) getUninstallExitCode(result *domain.UninstallResult) error {
	if len(result.Failed) > 0 && len(result.Uninstalled) == 0 {
		msg := "All uninstalls failed. Try:\n"
		msg += "  • Check if packages are actually installed\n"

		msg += "  • Run with sudo for system packages\n"
		if !app.verbose {
			msg += "  • Use --verbose for detailed errors"
		}

		return domain.NewExitError(ExitAppError, msg, nil)
	} else if len(result.Failed) > 0 {
		return domain.NewExitError(ExitWarnings, fmt.Sprintf("%d packages failed to uninstall", len(result.Failed)), nil)
	}

	return nil
}

// createListCommand creates list command to show installed packages.
func (app *CLI) createListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List installed packages",
		Action: app.runList,
	}
}

// runList handles the list command execution with output adapter.
func (app *CLI) runList(ctx context.Context, _ *cli.Command) error {
	// Create output adapter based on flags
	output := cliAdapter.OutputFromContext(app.json, app.quiet)

	// Get installed packages information
	result := &domain.ListResult{
		Packages:  []domain.PackageInfo{},
		Timestamp: time.Now(),
	}

	// Check installed applications
	installedApps := app.getInstalledApps()
	for _, appName := range installedApps {
		pkg := domain.PackageInfo{
			Name:        appName,
			Type:        "app",
			Installed:   time.Now(), // Would need to get actual install time from metadata
			Description: app.getAppDescription(appName),
		}

		// Try to get version info
		if version := app.getAppVersion(ctx, appName); version != "" {
			pkg.Version = version
		}

		result.Packages = append(result.Packages, pkg)
	}

	// Check installed themes
	if currentTheme := app.getCurrentTheme(); currentTheme != "" {
		result.Packages = append(result.Packages, domain.PackageInfo{
			Name:        currentTheme,
			Type:        "theme",
			Installed:   time.Now(),
			Description: "Active theme",
		})
	}

	// Check installed fonts
	if currentFont := app.getCurrentFont(); currentFont != "" {
		result.Packages = append(result.Packages, domain.PackageInfo{
			Name:        currentFont,
			Type:        "font",
			Installed:   time.Now(),
			Description: "Active font",
		})
	}

	result.Total = len(result.Packages)

	// Output results
	if app.json {
		return output.Success("", result)
	}

	// Text output as table
	if len(result.Packages) > 0 {
		headers := []string{"Name", "Type", "Version", "Description"}
		rows := make([][]string, 0, len(result.Packages))

		for _, pkg := range result.Packages {
			version := pkg.Version
			if version == "" {
				version = "-"
			}

			rows = append(rows, []string{pkg.Name, pkg.Type, version, pkg.Description})
		}

		_ = output.Table(headers, rows)
		_ = output.Info(fmt.Sprintf("\nTotal: %d packages installed", result.Total))
	} else {
		_ = output.Info("No packages installed")
	}

	return nil
}

// Helper methods for list command.
func (app *CLI) getInstalledApps() []string {
	// Check common installed apps from the system
	commonApps := []string{"git", "vim", "docker", "go", "rust", "node", "python"}

	var installed []string

	for _, name := range commonApps {
		// Check if app is actually installed
		if system.CommandExists(name) {
			installed = append(installed, name)
		}
	}

	return installed
}

func (app *CLI) getAppDescription(appName string) string {
	// Return basic descriptions for known apps
	descriptions := map[string]string{
		"git":    "Version control system",
		"vim":    "Text editor",
		"docker": "Container platform",
		"go":     "Go programming language",
		"rust":   "Rust programming language",
		"node":   "Node.js runtime",
		"python": "Python programming language",
	}

	if desc, exists := descriptions[appName]; exists {
		return desc
	}

	return ""
}

func (app *CLI) getAppVersion(ctx context.Context, appName string) string {
	// Try to get version from the app itself
	executor := patterns.NewCommandExecutor(false, false)

	// Common version flags
	versionFlags := []string{"--version", "-v", "version"}
	for _, flag := range versionFlags {
		output, err := executor.ExecuteWithOutput(ctx, appName, flag)
		if err == nil && output != "" {
			// Extract version number from output (simplified)
			lines := strings.Split(output, "\n")
			if len(lines) > 0 {
				// Look for version pattern
				for _, line := range lines {
					if strings.Contains(strings.ToLower(line), "version") {
						return strings.TrimSpace(line)
					}
				}

				return strings.TrimSpace(lines[0])
			}
		}
	}

	return ""
}

func (app *CLI) getCurrentTheme() string {
	// Would check actual theme configuration
	// For now, returning empty
	return ""
}

func (app *CLI) getCurrentFont() string {
	// Would check actual font configuration
	// For now, returning empty
	return ""
}

// createSetupCommand creates first-time setup command.
func (app *CLI) createSetupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Run first-time interactive setup",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return app.runFirstTimeSetup(ctx)
		},
	}
}

// createAppsCommand creates app selection command.
func (app *CLI) createAppsCommand() *cli.Command {
	return &cli.Command{
		Name:  "apps",
		Usage: "Interactive app selection and installation",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return app.runAppSelector(ctx)
		},
	}
}

// createDesktopCommand creates desktop entries command.
func (app *CLI) createDesktopCommand() *cli.Command {
	return &cli.Command{
		Name:  "desktop",
		Usage: "Create desktop application entries",
		Action: func(_ context.Context, _ *cli.Command) error {
			if err := desktop.CreateAllDesktopEntries(); err != nil {
				if app.verbose {
					return fmt.Errorf("failed to create desktop entries: %w", err)
				}
				return ErrDesktopEntries
			}
			fmt.Println("✓ Desktop entries created successfully")

			return nil
		},
	}
}

// createMenuCommand creates interactive menu.
func (app *CLI) createMenuCommand() *cli.Command {
	return &cli.Command{
		Name:  "menu",
		Usage: "Show interactive menu",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return app.runInteractiveMenu(ctx)
		},
	}
}

// createVersionCommand creates version command.
func (app *CLI) createVersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show version information",
		Action: func(_ context.Context, _ *cli.Command) error {
			version := app.getVersion()
			console.DefaultOutput.SuccessResult(version, "")

			return nil
		},
	}
}

// createFontSizeCommand creates font size management command.
func (app *CLI) createFontSizeCommand() *cli.Command {
	return &cli.Command{
		Name:      "font-size",
		Usage:     "Manage terminal font size",
		ArgsUsage: "[size|increase|decrease|show]",
		Action:    app.handleFontSizeCommand,
	}
}

// createHelpCommand creates git-style help command.
func (app *CLI) createHelpCommand() *cli.Command {
	return &cli.Command{
		Name:      "help",
		Usage:     "Show help for commands",
		ArgsUsage: "[command|examples]",
		Description: `Display help information for karei commands.

USAGE:
  karei help              Show main help
  karei help examples     Show comprehensive examples
  karei help <command>    Show detailed command documentation
  karei help tutorial     Interactive tutorial guide
  karei help troubleshoot Common problems and solutions
  karei help faq          Frequently asked questions

This works the same as using --help or -h flags.`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()

			if len(args) == 0 {
				// Show concise help like the default action
				app.showConciseHelp()
				fmt.Printf("\nFor complete help, use: karei --help\n")

				return nil
			}

			subcommand := args[0]

			// Handle special help topics
			switch subcommand {
			case "examples":
				app.showExamples()

				return nil
			case "tutorial":
				app.showTutorial()

				return nil
			case "troubleshoot", "troubleshooting":
				app.showTroubleshooting()

				return nil
			case "faq":
				app.showFAQ()

				return nil
			case "man", "manual":
				app.showManPage(ctx)

				return nil
			}

			// Show detailed command documentation
			if app.showDetailedCommandHelp(subcommand) {
				return nil
			}

			// Command not found
			return domain.NewExitError(ExitNotFoundError, "unknown help topic: "+subcommand, nil)
		},
	}
}

// runInteractiveMenu provides simplified interactive menu
// Replaces complex menu implementations with universal pattern.
func (app *CLI) runInteractiveMenu(ctx context.Context) error {
	app.showHeader()

	for {
		choice, err := app.showMainMenu()
		if err != nil {
			return err
		}

		if choice == "exit" {
			break
		}

		if err := app.handleMenuChoice(ctx, choice); err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println("Press Enter to continue...")
			_, _ = fmt.Scanln()
		}
	}

	return nil
}

// showMainMenu displays simplified main menu options.
func (app *CLI) showMainMenu() (string, error) {
	// Check if stdin is a terminal - if not, show help instead of hanging
	if !console.DefaultOutput.IsTTY(os.Stdin.Fd()) {
		console.DefaultOutput.Errorf("Interactive menu requires a terminal")
		fmt.Fprintf(os.Stderr, "Use: karei <command> or karei --help\n")

		return "", domain.NewExitError(ExitUsageError, "stdin is not a terminal", nil)
	}

	options := []string{
		"theme - Manage themes",
		"font - Manage fonts",
		"install - Install packages",
		"security - Security tools",
		"verify - Verify system",
		"logs - View logs",
		"update - Update Karei",
		"exit - Exit",
	}

	fmt.Println("\n▸ Karei Menu")
	fmt.Println("================")

	for i, option := range options {
		fmt.Printf("  %d. %s\n", i+1, option)
	}

	fmt.Print("\nChoice (1-8): ")

	var choice int
	if _, err := fmt.Scanln(&choice); err != nil {
		return "", ErrInvalidInput
	}

	if choice < 1 || choice > len(options) {
		return "", ErrInvalidChoice
	}

	// Extract command from option
	parts := strings.Split(options[choice-1], " - ")

	return parts[0], nil
}

// handleMenuChoice handles menu selections using universal commands.
func (app *CLI) handleMenuChoice(ctx context.Context, choice string) error {
	// Create universal command for the choice and execute interactively
	var universalCmd *patterns.UniversalCommand

	switch choice {
	case "theme":
		universalCmd = patterns.NewThemeCommand(app.verbose)
	case "font":
		universalCmd = patterns.NewFontCommand(app.verbose)
	case "security":
		universalCmd = patterns.NewSecurityCommand(app.verbose)
	case "verify":
		universalCmd = patterns.NewVerifyCommand(app.verbose)
	case "logs":
		universalCmd = patterns.NewLogsCommand(app.verbose)
	case "install":
		return app.interactiveInstall(ctx)
	case "update":
		return app.updateKarei(ctx)
	default:
		return fmt.Errorf("%w: %s", ErrUnknownChoice, choice)
	}

	// Execute universal command interactively
	return universalCmd.Execute(ctx, []string{})
}

// interactiveInstall provides simplified interactive installation.
func (app *CLI) interactiveInstall(ctx context.Context) error {
	commonPackages := []string{
		"git", "curl", "wget", "vim", "htop", "tree", "jq", "unzip",
	}

	fmt.Println("\nCommon packages:")

	for i, pkg := range commonPackages {
		fmt.Printf("  %d. %s\n", i+1, pkg)
	}

	fmt.Print("\nSelect package (1-8) or type custom name: ")

	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return ErrInvalidInput
	}

	// Check if it's a number (selecting from list)
	if choice := app.parseChoice(input, len(commonPackages)); choice > 0 {
		input = commonPackages[choice-1]
	}

	executor := patterns.NewCommandExecutor(app.verbose, false)

	return executor.ExecuteSudo(ctx, "apt-get", "install", "-y", input)
}

// updateKarei performs Karei update.
func (app *CLI) updateKarei(ctx context.Context) error {
	executor := patterns.NewCommandExecutor(app.verbose, false)
	kareiPath := config.GetKareiPath()

	fmt.Println("• Connecting to Git repository for update...")
	fmt.Println("↻ Updating Karei...")

	if err := executor.Execute(ctx, "git", "-C", kareiPath, "pull"); err != nil {
		if app.verbose {
			return fmt.Errorf("failed to update: %w", err)
		}

		return ErrUpdateFailed
	}

	fmt.Println("✓ Git update completed successfully")
	fmt.Println("✓ Karei updated successfully")

	return nil
}

// parseChoice parses user input as number choice.
func (app *CLI) parseChoice(input string, maxVal int) int {
	var choice int
	if n, err := fmt.Sscanf(input, "%d", &choice); n == 1 && err == nil && choice >= 1 && choice <= maxVal {
		return choice
	}

	return 0
}

// defaultAction runs when no command is provided.
func (app *CLI) defaultAction(ctx context.Context, _ *cli.Command) error {
	// Check if help flags are present anywhere in arguments
	args := os.Args[1:] // Skip program name
	for _, arg := range args {
		if arg == "-h" || arg == HelpFlag {
			// Show help and exit successfully
			app.showConciseHelp()

			return nil
		}
	}

	// If any arguments provided but no valid command found, show help instead of TUI
	if len(args) > 0 {
		app.showConciseHelp()
		fmt.Fprintf(os.Stderr, "\nFor complete help, use: karei --help\n")

		return nil
	}

	// Launch TUI only when no arguments provided
	if err := tui.LaunchInteractive(ctx); err != nil {
		// TUI errors are usually terminal-related
		if app.verbose {
			return domain.NewExitError(ExitGeneralError, fmt.Sprintf("Failed to launch TUI: %v", err), nil)
		}

		return domain.NewExitError(ExitGeneralError, "Failed to launch interactive interface (terminal required)", nil)
	}

	return nil
}

// initConfig initializes configuration and output settings.
func (app *CLI) initConfig(ctx context.Context, _ *cli.Command) (context.Context, error) {
	// Validate conflicting flags
	if app.json && app.plain {
		return ctx, domain.NewExitError(ExitUsageError, "cannot use both --json and --plain flags simultaneously", nil)
	}

	// Validate color flag
	switch app.color {
	case "auto", "always", "never":
		// Valid values
	default:
		return ctx, domain.NewExitError(ExitUsageError, "invalid --color value: must be auto, always, or never", nil)
	}

	// Apply color override to environment
	switch app.color {
	case "never":
		_ = os.Setenv("NO_COLOR", "1")
	case "always":
		_ = os.Unsetenv("NO_COLOR")
		// Color output handled by TTY detection in platform/output.go
	}
	// "auto" uses default TTY detection

	// Configure output utilities based on flags
	console.DefaultOutput.SetMode(app.verbose, app.json, app.plain)

	// Set global auto-yes flag
	console.AutoYes = app.yes

	return ctx, nil
}

// getVersion returns current version.
func (app *CLI) getVersion() string {
	versionFile := filepath.Join(config.GetKareiPath(), "version")
	if content, err := os.ReadFile(versionFile); err == nil { //nolint:gosec
		return strings.TrimSpace(string(content))
	}

	return "dev"
}

// getVersionWithPath returns current version with custom path for testing.
func (app *CLI) getVersionWithPath(customPath string) string {
	kareiPath := customPath
	if kareiPath == "" {
		kareiPath = config.GetKareiPath()
	}

	versionFile := filepath.Join(kareiPath, "version")
	if content, err := os.ReadFile(versionFile); err == nil { //nolint:gosec
		return strings.TrimSpace(string(content))
	}

	return "dev"
}

// showHeader displays ASCII art header.
func (app *CLI) showHeader() {
	// Simplified header - reuses existing ASCII art logic
	asciiArt := []string{
		"________      _____      _____  ___________.___  _________ ___ ___ ",
		"\\_____  \\    /     \\    /  _  \\ \\_   _____/|   |/   _____//   |   \\",
		" /   |   \\  /  \\ /  \\  /  /_\\  \\ |    __)  |   |\\_____  \\/    ~    \\",
		"/    |    \\/    Y    \\/    |    \\|     \\   |   |/        \\    Y    /",
		"\\_______  /\\____|__  /\\____|__  /\\___  /   |___/_______  /\\___|_  /",
		"        \\/         \\/         \\/     \\/                \\/       \\/",
	}

	// Color gradient
	colors := []string{
		"\033[38;5;81m", "\033[38;5;75m", "\033[38;5;69m",
		"\033[38;5;63m", "\033[38;5;57m", "\033[38;5;51m",
	}
	reset := "\033[0m"

	for i, line := range asciiArt {
		colorIndex := i % len(colors)
		fmt.Printf("%s%s%s\n", colors[colorIndex], line, reset)
	}

	fmt.Printf("\n▸ Karei %s\n", app.getVersion())
}

// commandNotFound handles unknown commands.
func (app *CLI) commandNotFound(_ context.Context, _ *cli.Command, command string) {
	// Check if help flags are present anywhere in arguments
	args := os.Args[1:] // Skip program name
	for _, arg := range args {
		if arg == "-h" || arg == HelpFlag {
			// Show help instead of error for unknown commands with help flags
			app.showConciseHelp()
			os.Exit(ExitSuccess)
		}
	}

	console.DefaultOutput.Errorf("'%s' is not a command.", command)
	fmt.Fprintf(os.Stderr, "\nRun 'karei --help' to see available commands.\n")

	// Exit with error code
	os.Exit(ExitNotFoundError)
}

// showConciseHelp displays user-friendly help when no command is provided.
func (app *CLI) showConciseHelp() {
	version := app.getVersion()

	// Output goes to stdout (helpful information, not an error)
	if app.json {
		// JSON mode - provide structured help data
		console.DefaultOutput.JSONResult("success", map[string]any{
			"name":    "karei",
			"version": version,
			"usage":   "karei <command> [args...]",
			"help":    "use 'karei --help' for complete documentation",
		})
	} else {
		// Brief help - immediate sense of what this tool does
		fmt.Printf("karei %s - Transform Linux into a development environment\n\n", version)

		fmt.Printf("%s\n", console.DefaultOutput.Header("ESSENTIAL COMMANDS"))
		fmt.Printf("  install <pkg>     Install tools (git, vim, docker, go, rust)\n")
		fmt.Printf("  theme <name>      Apply themes (tokyo-night, catppuccin, nord)\n")
		fmt.Printf("  verify            Check what's installed\n\n")

		fmt.Printf("%s\n", console.DefaultOutput.Header("GET STARTED"))
		fmt.Printf("  karei install git vim\n")
		fmt.Printf("  karei theme tokyo-night\n\n")

		fmt.Printf("Complete help:       karei --help\n")
		fmt.Printf("Detailed examples:   karei help examples\n")
		fmt.Printf("Command docs:        karei help <command>\n")
	}
}

// showExamples displays comprehensive examples grouped by use case.
func (app *CLI) showExamples() {
	version := app.getVersion()

	fmt.Printf("karei examples - Learn by doing [version %s]\n\n", version)

	// Start with essentials - you need tools before beautification
	fmt.Printf("%s\n", console.DefaultOutput.Header("STORY 1: ESSENTIAL TOOLS"))
	fmt.Printf("First things first - get the tools every developer needs:\n\n")

	fmt.Printf("  $ karei install git vim curl\n")
	fmt.Printf("  ▸ Installing git...\n")
	fmt.Printf("  ✓ git installed successfully\n")
	fmt.Printf("  ▸ Installing vim...\n")
	fmt.Printf("  ✓ vim installed successfully\n")
	fmt.Printf("  ▸ Installing curl...\n")
	fmt.Printf("  ✓ curl installed successfully\n\n")

	fmt.Printf("  $ karei verify\n")
	fmt.Printf("  ▸ Verifying tools...\n")
	fmt.Printf("  ✓ git\n")
	fmt.Printf("  ✓ vim\n")
	fmt.Printf("  ✓ curl\n")
	fmt.Printf("  ✓ fish\n")
	fmt.Printf("  ✗ docker - not found\n\n")

	// Now make it beautiful
	fmt.Printf("%s\n", console.DefaultOutput.Header("STORY 2: INSTANT TRANSFORMATION"))
	fmt.Printf("Now make your dev environment beautiful:\n\n")

	fmt.Printf("  $ karei theme tokyo-night\n")
	fmt.Printf("  ▸ Applying tokyo-night theme to 6 applications...\n")
	fmt.Printf("  ✓ GNOME theme applied\n")
	fmt.Printf("  ✓ Terminal theme applied\n")
	fmt.Printf("  ✓ VS Code theme applied\n")
	fmt.Printf("  ✓ Theme 'tokyo-night' application complete\n\n")

	fmt.Printf("  Your entire desktop just transformed! Try switching windows to see.\n\n")

	// More complex - language setup
	fmt.Printf("%s\n", console.DefaultOutput.Header("STORY 3: LANGUAGE SETUP"))
	fmt.Printf("Set up a complete Go development environment:\n\n")

	fmt.Printf("  $ karei install go\n")
	fmt.Printf("  ▸ Installing language: go (latest)\n")
	fmt.Printf("  ✓ Go 1.21.5 installed\n")
	fmt.Printf("  ✓ GOPATH configured\n")
	fmt.Printf("  ✓ Added to PATH\n\n")

	fmt.Printf("  $ go version\n")
	fmt.Printf("  go version go1.21.5 linux/amd64\n\n")

	// Complex workflow combining multiple commands
	fmt.Printf("%s\n", console.DefaultOutput.Header("STORY 4: COMPLETE SETUP"))
	fmt.Printf("Set up everything for a new project:\n\n")

	fmt.Printf("  # Pick your style\n")
	fmt.Printf("  $ karei theme catppuccin\n")
	fmt.Printf("  ✓ Theme 'catppuccin' applied to all applications\n\n")

	fmt.Printf("  # Get your tools\n")
	fmt.Printf("  $ karei install development      # Install entire dev group\n")
	fmt.Printf("  ▸ Installing development group...\n")
	fmt.Printf("  ✓ git installed\n")
	fmt.Printf("  ✓ docker installed\n")
	fmt.Printf("  ✓ neovim installed\n")
	fmt.Printf("  ✓ lazygit installed\n")
	fmt.Printf("  ✓ development group installed successfully\n\n")

	fmt.Printf("  # Set your font\n")
	fmt.Printf("  $ karei font JetBrainsMono\n")
	fmt.Printf("  ✓ Font 'JetBrainsMono' applied successfully\n\n")

	fmt.Printf("  # Check everything\n")
	fmt.Printf("  $ karei status\n")
	fmt.Printf("  Current Configuration:\n")
	fmt.Printf("  ✓ Theme: catppuccin\n")
	fmt.Printf("  ✓ Font: JetBrainsMono\n")
	fmt.Printf("  ✓ 15/15 common tools available\n\n")

	// Advanced workflows
	fmt.Printf("%s\n", console.DefaultOutput.Header("ADVANCED EXAMPLES"))

	fmt.Printf("Interactive app browser:\n")
	fmt.Printf("  $ karei apps\n")
	fmt.Printf("  [Interactive TUI launches with categories]\n\n")

	fmt.Printf("JSON output for scripting:\n")
	fmt.Printf("  $ karei --json verify | jq '.tools'\n")
	fmt.Printf("  {\"git\":\"installed\",\"vim\":\"installed\",\"docker\":\"missing\"}\n\n")

	fmt.Printf("Batch theme switching:\n")
	fmt.Printf("  $ for theme in tokyo-night nord catppuccin; do\n")
	fmt.Printf("      karei theme $theme && sleep 5\n")
	fmt.Printf("    done\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("DOCUMENTATION"))
	fmt.Printf("Terminal Documentation:\n")
	fmt.Printf("  karei help <command>      # Command-specific help\n")
	fmt.Printf("  karei help man            # Unix manual page\n")
	fmt.Printf("  man karei                 # System man page (if installed)\n\n")

	fmt.Printf("Web Documentation:\n")
	fmt.Printf("  https://docs.karei.org                 # Searchable documentation\n")
	fmt.Printf("  https://docs.karei.org/commands/theme  # Theme command guide\n")
	fmt.Printf("  https://docs.karei.org/workflows       # Complete workflows\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("SUPPORT"))
	fmt.Printf("• Report bugs:     https://github.com/janderssonse/karei/issues\n")
	fmt.Printf("• Ask questions:   https://github.com/janderssonse/karei/discussions\n")
	fmt.Printf("• Source code:     https://github.com/janderssonse/karei\n")
}

// showManPage displays the manual page in terminal.
func (app *CLI) showManPage(ctx context.Context) {
	// Try to use system man first, fall back to embedded
	if err := system.Run(ctx, false, "man", "karei"); err == nil {
		return
	}

	// Fall back to embedded man page content
	fmt.Printf("KAREI(1)                         KAREI MANUAL                         KAREI(1)\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("NAME"))
	fmt.Printf("       karei - Linux development environment automation\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("SYNOPSIS"))
	fmt.Printf("       karei [global-options] command [args...]\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("DESCRIPTION"))
	fmt.Printf("       Karei transforms fresh Linux installations into fully-configured\n")
	fmt.Printf("       development environments with modern tools, beautiful themes, and\n")
	fmt.Printf("       everything developers need to get started quickly.\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("GLOBAL OPTIONS"))
	fmt.Printf("       --help, -h     Show help information\n")
	fmt.Printf("       --verbose      Show progress messages to stderr\n")
	fmt.Printf("       --json         Output structured JSON results\n")
	fmt.Printf("       --plain        Output plain text without formatting\n")
	fmt.Printf("       --version      Show version information\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("ESSENTIAL COMMANDS"))
	fmt.Printf("       install <pkg>  Install development tools and packages\n")
	fmt.Printf("       theme <name>   Apply coordinated themes across applications\n")
	fmt.Printf("       verify         Check system configuration and installed tools\n")
	fmt.Printf("       font <name>    Install and configure programming fonts\n")
	fmt.Printf("       help <topic>   Show detailed help for commands or topics\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("EXAMPLES"))
	fmt.Printf("       Install essential tools:\n")
	fmt.Printf("           karei install git vim curl\n\n")
	fmt.Printf("       Apply a beautiful theme:\n")
	fmt.Printf("           karei theme tokyo-night\n\n")
	fmt.Printf("       Check your setup:\n")
	fmt.Printf("           karei verify\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("EXIT CODES"))
	fmt.Printf("       0      Success\n")
	fmt.Printf("       2      Usage error\n")
	fmt.Printf("       5      Not found (theme/font/app)\n")
	fmt.Printf("       10     Dependencies missing\n")
	fmt.Printf("       20-24  Domain-specific errors\n")
	fmt.Printf("       64     Completed with warnings\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("SEE ALSO"))
	fmt.Printf("       karei help examples    Complete workflows and tutorials\n")
	fmt.Printf("       karei help <command>   Detailed command documentation\n")
	fmt.Printf("       https://docs.karei.org Web documentation\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("BUGS"))
	fmt.Printf("       Report bugs: https://github.com/janderssonse/karei/issues\n\n")
}

// showDetailedCommandHelp displays comprehensive documentation for specific commands.
func (app *CLI) showDetailedCommandHelp(command string) bool {
	switch command {
	case "theme":
		app.showThemeDocumentation()
	case "font":
		app.showFontDocumentation()
	case "install":
		app.showInstallDocumentation()
	case "security":
		app.showSecurityDocumentation()
	case "verify":
		app.showVerifyDocumentation()
	case "logs":
		app.showLogsDocumentation()
	case "update":
		app.showUpdateDocumentation()
	default:
		return false
	}

	return true
}

// showThemeDocumentation displays comprehensive theme documentation.
func (app *CLI) showThemeDocumentation() {
	fmt.Printf("karei-theme(1)                      KAREI MANUAL                      karei-theme(1)\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("NAME"))
	fmt.Printf("       karei theme - Apply coordinated themes across all applications\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("SYNOPSIS"))
	fmt.Printf("       karei theme [theme-name]\n")
	fmt.Printf("       karei theme list\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("DESCRIPTION"))
	fmt.Printf("       The theme command applies coordinated color schemes across GNOME desktop,\n")
	fmt.Printf("       terminal emulators, editors, and browsers. All applications using the\n")
	fmt.Printf("       selected theme will have consistent colors and styling.\n\n")

	fmt.Printf("       Themes are applied system-wide and affect:\n")
	fmt.Printf("         • GNOME desktop environment (if available)\n")
	fmt.Printf("         • Terminal applications (ghostty, btop, zellij)\n")
	fmt.Printf("         • Text editors (neovim, vscode)\n")
	fmt.Printf("         • Web browsers (chrome extensions)\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("OPTIONS"))
	fmt.Printf("       theme-name    Apply the specified theme\n")
	fmt.Printf("       list          Show available themes with current selection\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("AVAILABLE THEMES"))
	fmt.Printf("       tokyo-night   Dark theme with bright accent colors\n")
	fmt.Printf("       catppuccin    Warm, pastel color palette\n")
	fmt.Printf("       nord          Arctic, blue-tinted theme\n")
	fmt.Printf("       everforest    Green-based, forest-inspired colors\n")
	fmt.Printf("       gruvbox       Retro groove colors with warm tones\n")
	fmt.Printf("       kanagawa      Traditional Japanese color palette\n")
	fmt.Printf("       rose-pine     Subtle, elegant rose-tinted colors\n")
	fmt.Printf("       gruvbox-light Light variant of gruvbox theme\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("EXAMPLES"))
	fmt.Printf("       Apply tokyo-night theme:\n")
	fmt.Printf("         $ karei theme tokyo-night\n")
	fmt.Printf("         tokyo-night\n\n")

	fmt.Printf("       List available themes:\n")
	fmt.Printf("         $ karei theme list\n")
	fmt.Printf("         ▶ tokyo-night\n")
	fmt.Printf("           catppuccin\n")
	fmt.Printf("           nord\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("TROUBLESHOOTING"))
	fmt.Printf("       Theme not applied to application:\n")
	fmt.Printf("         1. Restart the application\n")
	fmt.Printf("         2. Check if application supports theming\n")
	fmt.Printf("         3. Run: karei verify integrations\n\n")

	fmt.Printf("       GNOME theme not changing:\n")
	fmt.Printf("         1. Ensure gnome-tweaks is installed\n")
	fmt.Printf("         2. Log out and back in\n")
	fmt.Printf("         3. Check GNOME extensions\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("SEE ALSO"))
	fmt.Printf("       karei-font(1), karei-verify(1)\n")
	fmt.Printf("       https://docs.karei.org/themes\n\n")
}

// showInstallDocumentation displays comprehensive install documentation.
func (app *CLI) showInstallDocumentation() {
	fmt.Printf("karei-install(1)                    KAREI MANUAL                    karei-install(1)\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("NAME"))
	fmt.Printf("       karei install - Install development packages and tools\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("SYNOPSIS"))
	fmt.Printf("       karei install <packages...>\n")
	fmt.Printf("       karei install <group>\n")
	fmt.Printf("       karei install <language>\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("DESCRIPTION"))
	fmt.Printf("       The install command manages software installation from multiple sources:\n")
	fmt.Printf("       system packages (APT), GitHub releases, and language toolchains.\n\n")

	fmt.Printf("       Installation sources are automatically detected:\n")
	fmt.Printf("         • APT packages (vim, git, curl)\n")
	fmt.Printf("         • GitHub repositories (user/repo format)\n")
	fmt.Printf("         • Language toolchains (go, rust, python)\n")
	fmt.Printf("         • Application groups (development, browsers)\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("PACKAGE TYPES"))
	fmt.Printf("       Individual packages:\n")
	fmt.Printf("         vim, git, curl, htop, tree, jq, unzip\n\n")

	fmt.Printf("       Language toolchains:\n")
	fmt.Printf("         go, rust, python, node, java\n\n")

	fmt.Printf("       Application groups:\n")
	fmt.Printf("         development     Essential dev tools\n")
	fmt.Printf("         browsers        Web browsers\n")
	fmt.Printf("         communication   Chat and email\n")
	fmt.Printf("         media           Audio/video tools\n")
	fmt.Printf("         productivity    Office applications\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("EXAMPLES"))
	fmt.Printf("       Install individual packages:\n")
	fmt.Printf("         $ karei install vim git curl\n")
	fmt.Printf("         vim\n")
	fmt.Printf("         git\n")
	fmt.Printf("         curl\n\n")

	fmt.Printf("       Install development group:\n")
	fmt.Printf("         $ karei install development\n")
	fmt.Printf("         Installing group: development\n\n")

	fmt.Printf("       Install from GitHub:\n")
	fmt.Printf("         $ karei install username/repository\n")
	fmt.Printf("         Cloning repository...\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("EXIT CODES"))
	fmt.Printf("       0     All packages installed successfully\n")
	fmt.Printf("       22    Package installation failed\n")
	fmt.Printf("       64    Some packages failed, others succeeded\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("SEE ALSO"))
	fmt.Printf("       karei-verify(1), karei-update(1)\n")
	fmt.Printf("       https://docs.karei.org/install\n\n")
}

// showTutorial displays interactive tutorial.
func (app *CLI) showTutorial() {
	fmt.Printf("karei tutorial - Interactive setup guide\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("WELCOME TO KAREI"))
	fmt.Printf("This tutorial will guide you through setting up your Linux development environment.\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("STEP 1: SYSTEM VERIFICATION"))
	fmt.Printf("First, let's check your system:\n")
	fmt.Printf("  $ karei verify\n\n")
	fmt.Printf("This command checks for required tools and dependencies.\n")
	fmt.Printf("Press Enter to continue or Ctrl+C to exit...\n")

	// Wait for user input
	_, _ = fmt.Scanln()

	fmt.Printf("\n%s\n", console.DefaultOutput.Header("STEP 2: CHOOSE A THEME"))
	fmt.Printf("Karei provides coordinated themes for your entire desktop:\n")
	fmt.Printf("  $ karei theme list          # See available themes\n")
	fmt.Printf("  $ karei theme tokyo-night   # Apply tokyo-night theme\n\n")
	fmt.Printf("Recommended themes for beginners: tokyo-night, catppuccin, nord\n")
	fmt.Printf("Press Enter to continue...\n")

	_, _ = fmt.Scanln()

	fmt.Printf("\n%s\n", console.DefaultOutput.Header("STEP 3: INSTALL DEVELOPMENT TOOLS"))
	fmt.Printf("Install essential development packages:\n")
	fmt.Printf("  $ karei install development  # Install development group\n")
	fmt.Printf("  $ karei install vim git      # Install specific packages\n\n")
	fmt.Printf("Press Enter to continue...\n")

	_, _ = fmt.Scanln()

	fmt.Printf("\n%s\n", console.DefaultOutput.Header("NEXT STEPS"))
	fmt.Printf("You're ready to start! Try these commands:\n\n")
	fmt.Printf("  karei help examples     # See comprehensive examples\n")
	fmt.Printf("  karei menu              # Interactive menu\n")
	fmt.Printf("  karei help <command>    # Detailed command help\n\n")
	fmt.Printf("Tutorial complete!\n")
}

// showTroubleshooting displays common problems and solutions.
func (app *CLI) showTroubleshooting() {
	fmt.Printf("karei troubleshooting - Common problems and solutions\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("INSTALLATION ISSUES"))
	fmt.Printf("Problem: Command not found after installation\n")
	fmt.Printf("Solution:\n")
	fmt.Printf("  1. Check PATH: echo $PATH\n")
	fmt.Printf("  2. Restart terminal or run: source ~/.bashrc\n")
	fmt.Printf("  3. Verify installation: karei verify path\n\n")

	fmt.Printf("Problem: Permission denied errors\n")
	fmt.Printf("Solution:\n")
	fmt.Printf("  1. Some operations require sudo (system packages)\n")
	fmt.Printf("  2. User installs go to ~/.local/bin\n")
	fmt.Printf("  3. Check file permissions: ls -la ~/.local/bin\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("THEME ISSUES"))
	fmt.Printf("Problem: Theme not applied to all applications\n")
	fmt.Printf("Solution:\n")
	fmt.Printf("  1. Restart affected applications\n")
	fmt.Printf("  2. Check theme support: karei verify integrations\n")
	fmt.Printf("  3. Some apps need manual configuration\n\n")

	fmt.Printf("Problem: GNOME theme not changing\n")
	fmt.Printf("Solution:\n")
	fmt.Printf("  1. Install gnome-tweaks: sudo apt install gnome-tweaks\n")
	fmt.Printf("  2. Log out and back in\n")
	fmt.Printf("  3. Check for conflicting extensions\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("FONT ISSUES"))
	fmt.Printf("Problem: Font not applied to terminal\n")
	fmt.Printf("Solution:\n")
	fmt.Printf("  1. Restart terminal application\n")
	fmt.Printf("  2. Check font installation: fc-list | grep FontName\n")
	fmt.Printf("  3. Manually set in terminal preferences\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("GETTING HELP"))
	fmt.Printf("Still having issues? Here's how to get help:\n\n")
	fmt.Printf("  1. Check logs: karei logs errors\n")
	fmt.Printf("  2. Run verification: karei verify all\n")
	fmt.Printf("  3. Check documentation: https://docs.karei.org\n")
	fmt.Printf("  4. File an issue: https://github.com/janderssonse/karei/issues\n\n")
	fmt.Printf("When reporting issues, include:\n")
	fmt.Printf("  • Your OS version: lsb_release -a\n")
	fmt.Printf("  • Karei version: karei version\n")
	fmt.Printf("  • Error logs: karei logs errors\n")
}

// showFAQ displays frequently asked questions.
func (app *CLI) showFAQ() {
	fmt.Printf("karei FAQ - Frequently Asked Questions\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("GENERAL QUESTIONS"))
	fmt.Printf("Q: What is Karei?\n")
	fmt.Printf("A: Karei is the easiest way to set up Linux for development. It transforms\n")
	fmt.Printf("   fresh Linux installations into fully-configured development environments\n")
	fmt.Printf("   with modern tools, beautiful themes, and everything developers need.\n\n")

	fmt.Printf("Q: Is Karei safe to use?\n")
	fmt.Printf("A: Yes. Karei follows security best practices:\n")
	fmt.Printf("   • Creates backups before making changes\n")
	fmt.Printf("   • Uses HTTPS for all downloads\n")
	fmt.Printf("   • Minimal sudo usage with clear escalation points\n")
	fmt.Printf("   • All code is open source and auditable\n\n")

	fmt.Printf("Q: Can I uninstall things installed by Karei?\n")
	fmt.Printf("A: Yes. Use 'karei uninstall <package>' to safely remove packages\n")
	fmt.Printf("   and clean up their configurations.\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("COMPATIBILITY"))
	fmt.Printf("Q: Which Linux distributions are supported?\n")
	fmt.Printf("A: Currently focused on Ubuntu 20.04 LTS and newer, with planned support\n")
	fmt.Printf("   for additional distributions. Ubuntu 22.04+ recommended for best compatibility.\n\n")

	fmt.Printf("Q: Will it support other distributions?\n")
	fmt.Printf("A: Yes! Multi-distro support is planned. Many features already work on\n")
	fmt.Printf("   Debian-based distributions, with broader Linux support coming soon.\n\n")

	fmt.Printf("Q: Can I use it on servers without GUI?\n")
	fmt.Printf("A: Yes! Terminal-only installations work fine. GUI features are\n")
	fmt.Printf("   automatically skipped on headless systems.\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("CUSTOMIZATION"))
	fmt.Printf("Q: Can I create custom themes?\n")
	fmt.Printf("A: Not yet directly, but you can:\n")
	fmt.Printf("   • Modify themes in ~/.local/share/karei/themes/\n")
	fmt.Printf("   • Submit new themes via GitHub\n")
	fmt.Printf("   • Use existing themes as templates\n\n")

	fmt.Printf("Q: How do I add custom applications?\n")
	fmt.Printf("A: Create scripts in ~/.local/share/karei/install/custom/\n")
	fmt.Printf("   Following the pattern of existing installers.\n\n")

	fmt.Printf("%s\n", console.DefaultOutput.Header("ADVANCED USAGE"))
	fmt.Printf("Q: Can I use Karei in scripts?\n")
	fmt.Printf("A: Yes! Use the --json flag for machine-readable output:\n")
	fmt.Printf("   karei --json theme tokyo-night | jq '.status'\n\n")

	fmt.Printf("Q: How do I contribute?\n")
	fmt.Printf("A: • Report bugs: GitHub issues\n")
	fmt.Printf("   • Add themes: Submit pull requests\n")
	fmt.Printf("   • Add applications: Create installer scripts\n")
	fmt.Printf("   • Documentation: Improve docs and examples\n\n")

	fmt.Printf("For more questions: https://github.com/janderssonse/karei/discussions\n")
}

// Placeholder documentation functions (to be expanded).
func (app *CLI) showFontDocumentation() {
	fmt.Printf("See: karei help tutorial for font management guide\n")
	fmt.Printf("Documentation: https://docs.karei.org/fonts\n")
}

func (app *CLI) showSecurityDocumentation() {
	fmt.Printf("See: karei help tutorial for security tools guide\n")
	fmt.Printf("Documentation: https://docs.karei.org/security\n")
}

func (app *CLI) showVerifyDocumentation() {
	fmt.Printf("See: karei help troubleshoot for verification help\n")
	fmt.Printf("Documentation: https://docs.karei.org/verify\n")
}

func (app *CLI) showLogsDocumentation() {
	fmt.Printf("See: karei help troubleshoot for log analysis\n")
	fmt.Printf("Documentation: https://docs.karei.org/logs\n")
}

func (app *CLI) showUpdateDocumentation() {
	fmt.Printf("See: karei help tutorial for update procedures\n")
	fmt.Printf("Documentation: https://docs.karei.org/update\n")
}

func (app *CLI) handleFontSizeCommand(_ context.Context, cmd *cli.Command) error {
	fontManager := fonts.NewSizeManager(app.verbose)

	if cmd.Args().Len() == 0 {
		return app.showFontSizeInfo(fontManager)
	}

	arg := cmd.Args().Get(0)

	return app.processFontSizeArg(fontManager, arg)
}

func (app *CLI) showFontSizeInfo(fontManager *fonts.SizeManager) error {
	current, _ := fontManager.GetCurrentSize()
	fmt.Printf("Current font size: %d\n", current)
	fmt.Println("\nAvailable sizes:")
	fmt.Println(fontManager.GetFontSizeDisplay())

	return nil
}

func (app *CLI) processFontSizeArg(fontManager *fonts.SizeManager, arg string) error {
	switch arg {
	case "show":
		return app.showFontSizeInfo(fontManager)
	case "increase":
		if err := fontManager.IncreaseFontSize(); err != nil {
			if app.verbose {
				return fmt.Errorf("failed to increase font size: %w", err)
			}

			return ErrFontSizeChange
		}

		fmt.Println("✓ Font size increased")
	case "decrease":
		if err := fontManager.DecreaseFontSize(); err != nil {
			if app.verbose {
				return fmt.Errorf("failed to decrease font size: %w", err)
			}

			return ErrFontSizeChange
		}

		fmt.Println("✓ Font size decreased")
	default:
		return app.setFontSizeFromString(fontManager, arg)
	}

	return nil
}

func (app *CLI) setFontSizeFromString(fontManager *fonts.SizeManager, arg string) error {
	var size int
	if n, err := fmt.Sscanf(arg, "%d", &size); n == 1 && err == nil {
		if err := fontManager.SetFontSizeForAllTerminals(size); err != nil {
			if app.verbose {
				return fmt.Errorf("failed to set font size: %w", err)
			}

			return ErrFontSizeChange
		}

		fmt.Printf("✓ Font size set to %d\n", size)

		return nil
	}

	return fmt.Errorf("%w: %s. Use 'increase', 'decrease', 'show', or a number", ErrInvalidArgument, arg)
}

// createStatusCommand creates a status command to show current system state.
func (app *CLI) createStatusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "Show current system state",
		ArgsUsage: "",
		Description: `Display the current state of your karei installation.

Shows:
- Installed language groups and individual tools
- Current theme and font settings
- System configuration status
- Helpful next steps

This command helps you understand what's currently installed and suggests actions.`,
		Action: app.handleStatusAction,
	}
}

func (app *CLI) handleStatusAction(_ context.Context, _ *cli.Command) error {
	// Create output adapter based on flags
	output := cliAdapter.OutputFromContext(app.json, app.quiet)

	// Gather comprehensive status information
	result := app.gatherSystemStatus()

	// Output status
	if app.json {
		return output.Success("", result)
	}

	// Enhanced text output with state and suggestions
	return app.displayDetailedStatus(output, result)
}

// gatherSystemStatus collects comprehensive system state information.
func (app *CLI) gatherSystemStatus() *domain.StatusResult {
	result := &domain.StatusResult{
		Version:      app.getVersion(),
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Timestamp:    time.Now(),
		Environment:  make(map[string]string),
	}

	// Count installed packages by type
	installedApps := app.getInstalledApps()
	result.Installed = len(installedApps)

	// Get current theme and font
	result.Theme = app.getCurrentTheme()
	result.Font = app.getCurrentFont()

	// Check environment state
	result.Environment["shell"] = app.detectShell()
	result.Environment["terminal"] = app.detectTerminal()
	result.Environment["package_manager"] = app.detectPackageManager()

	return result
}

// displayDetailedStatus shows comprehensive system state with suggestions.
func (app *CLI) displayDetailedStatus(output domain.OutputPort, result *domain.StatusResult) error {
	if output.IsQuiet() {
		return nil
	}

	// Header with git-like styling
	_ = output.Info("Karei Development Environment")
	_ = output.Info("On platform " + result.Platform + "/" + result.Architecture)
	_ = output.Info("Version " + result.Version + " (use 'karei update' to check for updates)")
	_ = output.Info("")

	// Package status with detailed breakdown
	app.displayPackageStatus(output, result)

	// Configuration status
	app.displayConfigurationStatus(output, result)

	// Development environment status
	app.displayDevelopmentStatus(output)

	// Suggested actions based on current state
	app.displaySuggestedActions(output, result)

	return nil
}

// displayPackageStatus shows detailed package information.
func (app *CLI) displayPackageStatus(output domain.OutputPort, result *domain.StatusResult) {
	if result.Installed == 0 {
		_ = output.Info("No packages installed")
		_ = output.Info("  (use 'karei install <package>' to install development tools)")
		_ = output.Info("  (use 'karei install development' to install a curated set)")
	} else {
		_ = output.Info(fmt.Sprintf("Packages installed: %d", result.Installed))

		// Show categorized packages for better overview
		installedApps := app.getInstalledApps()
		categories := app.categorizeApps(installedApps)

		for category, apps := range categories {
			_ = output.Info(fmt.Sprintf("  %s: %s", category, strings.Join(apps, ", ")))
		}

		_ = output.Info("  (use 'karei list' for detailed package information)")
	}

	_ = output.Info("")
}

// displayConfigurationStatus shows theme and font configuration.
func (app *CLI) displayConfigurationStatus(output domain.OutputPort, result *domain.StatusResult) {
	_ = output.Info("Configuration:")

	if result.Theme != "" {
		_ = output.Info("  Theme: " + result.Theme + " (active)")
	} else {
		_ = output.Info("  Theme: none configured")
		_ = output.Info("    (use 'karei theme <name>' to apply a coordinated theme)")
		_ = output.Info("    (use 'karei theme list' to see available themes)")
	}

	if result.Font != "" {
		_ = output.Info("  Font: " + result.Font + " (active)")
	} else {
		_ = output.Info("  Font: system default")
		_ = output.Info("    (use 'karei font <name>' to configure terminal fonts)")
	}

	_ = output.Info("")
}

// displayDevelopmentStatus shows development tool status.
func (app *CLI) displayDevelopmentStatus(output domain.OutputPort) {
	_ = output.Info("Essential development tools:")

	// Check essential development tools
	essentialTools := []struct {
		name, description string
	}{
		{"git", "Version control system"},
		{"vim", "Text editor"},
		{"docker", "Container platform"},
		{"go", "Go programming language"},
		{"node", "Node.js runtime"},
	}

	installedCount := 0
	missingTools := []string{}

	for _, tool := range essentialTools {
		if system.CommandExists(tool.name) {
			_ = output.Info(fmt.Sprintf("  ✓ %s (%s)", tool.name, tool.description))

			installedCount++
		} else {
			_ = output.Info(fmt.Sprintf("  ✗ %s (%s) - not installed", tool.name, tool.description))
			missingTools = append(missingTools, tool.name)
		}
	}

	if len(missingTools) > 0 {
		_ = output.Info(fmt.Sprintf("  Missing %d essential tools: %s", len(missingTools), strings.Join(missingTools, ", ")))
	}

	_ = output.Info("")
}

// displaySuggestedActions provides context-aware command suggestions.
func (app *CLI) displaySuggestedActions(output domain.OutputPort, result *domain.StatusResult) {
	suggestions := app.generateSuggestions(result)

	if len(suggestions) == 0 {
		_ = output.Info("✓ Your development environment is well configured!")
		_ = output.Info("  Run 'karei update' to check for updates")
		_ = output.Info("  Run 'karei apps' to discover new tools")
	} else {
		_ = output.Info("Suggested next steps:")

		for _, suggestion := range suggestions {
			_ = output.Info("  " + suggestion)
		}
	}

	_ = output.Info("")
}

// createTUICommand creates the interactive TUI command.
func (app *CLI) createTUICommand() *cli.Command {
	return &cli.Command{
		Name:      "tui",
		Usage:     "Launch interactive TUI interface",
		ArgsUsage: "",
		Description: `Launch the interactive Terminal User Interface (TUI) for Karei.
The TUI provides a menu-driven interface for:
- Installing applications interactively
- Configuring themes with live preview
- Managing system settings
- Viewing installation status
- Accessing help and documentation

Navigation:
- Use arrow keys or j/k to navigate
- Press Enter to select
- Press q or Ctrl+C to quit`,
		Action: app.handleTUIAction,
	}
}

// handleTUIAction handles the TUI command.
func (app *CLI) handleTUIAction(ctx context.Context, _ *cli.Command) error {
	if err := tui.LaunchInteractive(ctx); err != nil {
		if app.verbose {
			return domain.NewExitError(ExitGeneralError, fmt.Sprintf("Failed to launch TUI: %v", err), nil)
		}

		return domain.NewExitError(ExitGeneralError, "Failed to launch interactive interface (terminal required)", nil)
	}

	return nil
}

// generateSuggestions creates context-aware command suggestions.
func (app *CLI) generateSuggestions(result *domain.StatusResult) []string {
	var suggestions []string

	// Suggest based on installation count
	if result.Installed == 0 {
		suggestions = append(suggestions, "Install essential tools: karei install git vim")
		suggestions = append(suggestions, "Install development group: karei install development")
	} else if result.Installed < 3 {
		suggestions = append(suggestions, "Add more development tools: karei install docker go")
	}

	// Suggest configuration
	if result.Theme == "" {
		suggestions = append(suggestions, "Apply a coordinated theme: karei theme tokyo-night")
		suggestions = append(suggestions, "Browse available themes: karei theme list")
	}

	if result.Font == "" {
		suggestions = append(suggestions, "Configure terminal fonts: karei font CaskaydiaMono")
	}

	// Check for missing essential tools and suggest installation
	installedApps := app.getInstalledApps()
	essentials := []string{"git", "vim"}
	missingEssentials := []string{}

	for _, tool := range essentials {
		found := false

		for _, installed := range installedApps {
			if installed == tool {
				found = true

				break
			}
		}

		if !found {
			missingEssentials = append(missingEssentials, tool)
		}
	}

	if len(missingEssentials) > 0 {
		suggestions = append(suggestions, "Install missing essentials: karei install "+strings.Join(missingEssentials, " "))
	}

	return suggestions
}

// Helper methods for environment detection.
func (app *CLI) detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return unknownValue
	}

	return filepath.Base(shell)
}

func (app *CLI) detectTerminal() string {
	terms := []string{"TERM_PROGRAM", "TERMINAL_EMULATOR", "TERM"}
	for _, env := range terms {
		if value := os.Getenv(env); value != "" {
			return value
		}
	}

	return unknownValue
}

func (app *CLI) detectPackageManager() string {
	managers := []string{"apt", "yum", "dnf", "pacman", "brew"}
	for _, mgr := range managers {
		if system.CommandExists(mgr) {
			return mgr
		}
	}

	return unknownValue
}

func (app *CLI) categorizeApps(apps []string) map[string][]string {
	categories := map[string][]string{
		"Development": {},
		"Tools":       {},
		"Languages":   {},
	}

	// Categorize known apps
	devTools := map[string]bool{"git": true, "vim": true, "docker": true}
	languages := map[string]bool{"go": true, "node": true, "python": true, "rust": true}

	for _, app := range apps {
		switch {
		case devTools[app]:
			categories["Development"] = append(categories["Development"], app)
		case languages[app]:
			categories["Languages"] = append(categories["Languages"], app)
		default:
			categories["Tools"] = append(categories["Tools"], app)
		}
	}

	// Remove empty categories
	for category, items := range categories {
		if len(items) == 0 {
			delete(categories, category)
		}
	}

	return categories
}

// App provides a clean app constructor following hexagonal architecture.
// Replaces multiple app creation patterns with single entry point.
func App() *cli.Command {
	app := NewCLI()

	return app.app
}

// showInstallHelp displays install command help.
func (app *CLI) showInstallHelp(output domain.OutputPort) {
	if !output.IsQuiet() {
		fmt.Printf("Usage: karei install --packages <packages> OR --group <group>\n\n")
		fmt.Printf("Examples:\n")
		fmt.Printf("  karei install --packages vim,git     # Install specific packages\n")
		fmt.Printf("  karei install --group development     # Install development group\n")
		fmt.Printf("  karei install -p git --json          # Output JSON results\n")
		fmt.Printf("  karei install -g essential -q        # Quiet mode\n\n")
		fmt.Printf("For more help: karei help install\n")
	}
}

// validateInstallFlags validates and returns install command flags.
func (app *CLI) validateInstallFlags(cmd *cli.Command) (string, string, error) {
	packagesFlag := cmd.String("packages")
	groupFlag := cmd.String("group")

	// Validate that at least one flag is provided
	if packagesFlag == "" && groupFlag == "" {
		return "", "", domain.NewExitError(ExitUsageError, "specify either --packages or --group", nil)
	}

	// Validate that both flags are not provided
	if packagesFlag != "" && groupFlag != "" {
		return "", "", domain.NewExitError(ExitUsageError, "specify either --packages or --group, not both", nil)
	}

	return packagesFlag, groupFlag, nil
}
