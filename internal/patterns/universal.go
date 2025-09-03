// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package patterns

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/janderssonse/karei/internal/config"
	"github.com/janderssonse/karei/internal/console"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/system"
)

// Exit codes (matching main.go).
const (
	ExitSuccess         = 0
	ExitGeneralError    = 1
	ExitUsageError      = 2
	ExitConfigError     = 3
	ExitPermissionError = 4
	ExitNotFoundError   = 5
	ExitDependencyError = 10
	ExitNetworkError    = 11
	ExitSystemError     = 12
	ExitTimeoutError    = 13
	ExitInterruptError  = 14
	ExitThemeError      = 20
	ExitFontError       = 21
	ExitAppError        = 22
	ExitBackupError     = 23
	ExitMigrationError  = 24
	ExitWarnings        = 64
)

var (
	// ErrInvalidInput indicates the provided input is malformed or invalid.
	ErrInvalidInput = errors.New("invalid input")
)

// UniversalManager consolidates ALL manager patterns into one flexible interface
// Eliminates 15+ separate manager implementations by using composition over inheritance.
type UniversalManager struct {
	Name       string
	Type       string
	KareiPath  string
	ConfigPath string
	Available  []string
	Current    string
	verbose    bool
	dryRun     bool
	handlers   map[string]func(context.Context, string) error
}

// ManagerType represents different types of managers.
type ManagerType string

const (
	// TypeTheme represents theme manager type.
	TypeTheme ManagerType = "theme"
	// TypeFont represents font manager type.
	TypeFont ManagerType = "font"
	// TypeInstall represents installation manager type.
	TypeInstall ManagerType = "install"
	// TypeSecurity represents security manager type.
	TypeSecurity ManagerType = "security"
	// TypeVerify represents verification manager type.
	TypeVerify ManagerType = "verify"
	// TypeLogs represents logs manager type.
	TypeLogs ManagerType = "logs"
	// TypeProxy represents proxy manager type.
	TypeProxy ManagerType = "proxy"
	// TypeSSH represents SSH manager type.
	TypeSSH ManagerType = "ssh"
	// TypeRestore represents restore manager type.
	TypeRestore ManagerType = "restore"
	// TypeBackup represents backup manager type.
	TypeBackup ManagerType = "backup"
	// TypeUpdate represents update manager type.
	TypeUpdate ManagerType = "update"
)

// UniversalConfig defines configuration for any manager type.
type UniversalConfig struct {
	Name      string
	Type      ManagerType
	Available []string
	Verbose   bool
	DryRun    bool
	Handlers  map[string]func(context.Context, string) error
}

// ManagerOption represents a functional option for configuring UniversalManager.
type ManagerOption func(*UniversalManager)

// WithVerbose sets the verbose flag for the UniversalManager.
func WithVerbose(verbose bool) ManagerOption {
	return func(m *UniversalManager) {
		m.verbose = verbose
	}
}

// WithDryRun sets the dry-run flag for the UniversalManager.
func WithDryRun(dryRun bool) ManagerOption {
	return func(m *UniversalManager) {
		m.dryRun = dryRun
	}
}

// WithHandlers sets the handlers map for the UniversalManager.
func WithHandlers(handlers map[string]func(context.Context, string) error) ManagerOption {
	return func(m *UniversalManager) {
		m.handlers = handlers
	}
}

// WithAvailable sets the available options for the UniversalManager.
func WithAvailable(available []string) ManagerOption {
	return func(m *UniversalManager) {
		m.Available = available
	}
}

// NewUniversalManager creates any type of manager using unified pattern
// Replaces 15+ separate NewManager() functions with one flexible constructor.
func NewUniversalManager(cfg UniversalConfig) *UniversalManager {
	return &UniversalManager{
		Name:       cfg.Name,
		Type:       string(cfg.Type),
		KareiPath:  config.GetKareiPath(),
		ConfigPath: config.GetConfigPath(string(cfg.Type)),
		Available:  cfg.Available,
		verbose:    cfg.Verbose,
		dryRun:     cfg.DryRun,
		handlers:   cfg.Handlers,
	}
}

// NewUniversalManagerWithOptions creates a new UniversalManager with functional options.
func NewUniversalManagerWithOptions(name string, managerType ManagerType, opts ...ManagerOption) *UniversalManager {
	manager := &UniversalManager{
		Name:       name,
		Type:       string(managerType),
		KareiPath:  config.GetKareiPath(),
		ConfigPath: config.GetConfigPath(string(managerType)),
		handlers:   make(map[string]func(context.Context, string) error),
	}

	// Apply options
	for _, opt := range opts {
		opt(manager)
	}

	return manager
}

// Apply applies configuration using the appropriate handler
// Consolidates Apply/Install/Configure/Execute methods from all managers.
//
//nolint:cyclop // Complex branching for different manager types
func (um *UniversalManager) Apply(ctx context.Context, target string) error {
	if !um.IsValid(target) {
		// Return specific exit code based on manager type
		code := ExitNotFoundError

		switch um.Type {
		case "theme":
			code = ExitThemeError
		case "font":
			code = ExitFontError
		case "install":
			code = ExitAppError
		}

		return domain.NewExitError(code, fmt.Sprintf("invalid %s: %s", um.Type, target), nil)
	}

	console.DefaultOutput.Progressf("Applying %s: %s", um.Type, target)

	// Use specific handler if available
	var err error
	if handler, exists := um.handlers[target]; exists {
		err = handler(ctx, target)
	} else if defaultHandler, exists := um.handlers["default"]; exists {
		// Fall back to default handler
		err = defaultHandler(ctx, target)
	} else {
		return domain.NewExitError(ExitConfigError, "no handler available for "+target, nil)
	}

	// Handle error case first to reduce nesting
	if err != nil {
		if um.verbose {
			console.DefaultOutput.Errorf("Failed to apply %s %s: %v", um.Type, target, err)
		} else {
			console.DefaultOutput.Errorf("Failed to apply %s %s", um.Type, target)
		}

		return err
	}

	// Success: update current state and save it
	um.Current = target
	if saveErr := um.SaveCurrent(target); saveErr != nil {
		msg := "Failed to save configuration"
		if um.verbose {
			console.DefaultOutput.Warningf("%s: %v", msg, saveErr)
		} else {
			console.DefaultOutput.Warningf("%s", msg)
		}
	}

	console.DefaultOutput.Successf("%s applied successfully: %s", um.Type, target)

	return err
}

// IsValid checks if a choice is valid for the manager.
func (um *UniversalManager) IsValid(choice string) bool {
	return slices.Contains(um.Available, choice)
}

// GetCurrent detects and caches the current selection if not already set.
func (um *UniversalManager) GetCurrent() string {
	if um.Current == "" {
		um.Current = um.detectCurrent()
	}

	return um.Current
}

// GetAvailable lists all configured options.
func (um *UniversalManager) GetAvailable() []string {
	return um.Available
}

// SaveCurrent saves the current selection to the configuration file.
func (um *UniversalManager) SaveCurrent(choice string) error {
	if err := um.SetCurrent(choice); err != nil {
		return err
	}

	content := fmt.Sprintf("KAREI_%s=%s\n", strings.ToUpper(um.Type), choice)

	return system.SafeWriteFile(um.ConfigPath, []byte(content))
}

// SetCurrent validates and updates the current selection.
func (um *UniversalManager) SetCurrent(choice string) error {
	if !um.IsValid(choice) {
		return fmt.Errorf("%w for %s: %s", ErrInvalidInput, um.Type, choice)
	}

	um.Current = choice

	return nil
}

// Status returns the current status of the manager.
func (um *UniversalManager) Status() map[string]any {
	return map[string]any{
		"type":      um.Type,
		"current":   um.GetCurrent(),
		"available": um.Available,
		"config":    um.ConfigPath,
	}
}

// Private methods (unexported - placed after public methods per funcorder)

func (um *UniversalManager) detectCurrent() string {
	if !system.FileExists(um.ConfigPath) {
		return um.getDefault()
	}

	content, err := os.ReadFile(um.ConfigPath)
	if err != nil {
		return um.getDefault()
	}

	// Look for pattern: TYPE=value
	pattern := fmt.Sprintf("KAREI_%s=", strings.ToUpper(um.Type))
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, pattern) {
			value := strings.TrimPrefix(line, pattern)

			value = strings.Trim(value, `"'`)
			if um.IsValid(value) {
				return value
			}
		}
	}

	return um.getDefault()
}

func (um *UniversalManager) getDefault() string {
	if len(um.Available) > 0 {
		return um.Available[0]
	}

	return ""
}

// UniversalCommand consolidates ALL CLI command patterns into one flexible interface
// Eliminates 20+ separate CLI command implementations.
type UniversalCommand struct {
	Name        string
	Usage       string
	Description string
	Manager     *UniversalManager
	Interactive bool
}

// CommandConfig defines configuration for any CLI command.
type CommandConfig struct {
	Name        string
	Usage       string
	Description string
	Type        ManagerType
	Available   []string
	Interactive bool
	Verbose     bool
	Handlers    map[string]func(context.Context, string) error
}

// NewUniversalCommand creates a new universal command with the given configuration.
func NewUniversalCommand(config CommandConfig) *UniversalCommand {
	manager := NewUniversalManager(UniversalConfig{
		Name:      config.Name,
		Type:      config.Type,
		Available: config.Available,
		Verbose:   config.Verbose,
		Handlers:  config.Handlers,
	})

	return &UniversalCommand{
		Name:        config.Name,
		Usage:       config.Usage,
		Description: config.Description,
		Manager:     manager,
		Interactive: config.Interactive,
	}
}

// Execute handles command execution with both direct and interactive modes
// Consolidates 20+ separate command Action functions.
func (uc *UniversalCommand) Execute(ctx context.Context, args []string) error {
	// Direct mode: command arg
	if len(args) > 0 && args[0] != "" {
		target := args[0]
		if target == "list" {
			uc.showAvailable()

			return nil
		}

		return uc.Manager.Apply(ctx, target)
	}

	// Interactive mode if enabled
	if uc.Interactive {
		uc.showConciseHelp()

		return uc.runInteractive(ctx)
	}

	// Show current status
	uc.showStatus()

	return nil
}

func (uc *UniversalCommand) showAvailable() {
	current := uc.Manager.GetCurrent()

	switch {
	case console.DefaultOutput.JSON:
		console.DefaultOutput.JSONResult("success", map[string]any{
			"type":      uc.Manager.Type,
			"current":   current,
			"available": uc.Manager.Available,
		})
	case console.DefaultOutput.Plain:
		// Plain mode: output each option with current status
		for _, option := range uc.Manager.Available {
			if option == current {
				console.DefaultOutput.PlainStatus(option, "current")
			} else {
				console.DefaultOutput.PlainStatus(option, "available")
			}
		}
	default:
		// Human-readable mode with visual markers
		console.DefaultOutput.Progressf("Available %s options:", uc.Manager.Type)

		for _, option := range uc.Manager.Available {
			marker := "  "
			if option == current {
				marker = "▶ "
			}

			console.DefaultOutput.Result(fmt.Sprintf("%s%s", marker, option))
		}
	}
}

func (uc *UniversalCommand) showStatus() {
	status := uc.Manager.Status()

	switch {
	case console.DefaultOutput.JSON:
		console.DefaultOutput.JSONResult("success", status)
	case console.DefaultOutput.Plain:
		// Plain mode: just output the current value
		console.DefaultOutput.PlainValue(fmt.Sprintf("%s", status["current"]))
	default:
		// Human-readable mode with additional info
		console.DefaultOutput.Result(fmt.Sprintf("%s", status["current"]))
		console.DefaultOutput.Progressf("Available: %v", status["available"])
	}
}

func (uc *UniversalCommand) showConciseHelp() {
	fmt.Fprintf(os.Stderr, "%s - %s\n\n", uc.Name, uc.Usage)
	fmt.Fprintf(os.Stderr, "Usage: karei %s [option]\n\n", uc.Name)

	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  karei %s list      # Show available options\n", uc.Name)

	available := uc.Manager.GetAvailable()
	if len(available) > 0 {
		fmt.Fprintf(os.Stderr, "  karei %s %s   # Apply %s\n", uc.Name, available[0], available[0])
	}

	fmt.Fprintf(os.Stderr, "\nFor more options, use: karei %s --help\n\n", uc.Name)
}

func (uc *UniversalCommand) runInteractive(ctx context.Context) error {
	// Skip interactive mode in JSON mode
	if console.DefaultOutput.JSON {
		return domain.NewExitError(ExitUsageError, "interactive mode not available in JSON output mode", nil)
	}

	// Check if stdin is a terminal - if not, show help instead of hanging
	if !console.DefaultOutput.IsTTY(os.Stdin.Fd()) {
		console.DefaultOutput.Errorf("Interactive mode requires a terminal")
		fmt.Fprintf(os.Stderr, "Use: karei %s <option> or karei %s list\n", uc.Name, uc.Name)

		return domain.NewExitError(ExitUsageError, "stdin is not a terminal", nil)
	}

	available := uc.Manager.GetAvailable()
	current := uc.Manager.GetCurrent()

	console.DefaultOutput.Progressf("Current %s: %s", uc.Manager.Type, current)
	console.DefaultOutput.Progressf("Available %s options:", uc.Manager.Type)

	for index, option := range available {
		marker := "  "
		if option == current {
			marker = "▶ "
		}

		console.DefaultOutput.Progressf("%s%d. %s", marker, index+1, option)
	}

	fmt.Fprintf(os.Stderr, "\nSelect %s (1-%d): ", uc.Manager.Type, len(available))

	var choice int
	if _, err := fmt.Scanln(&choice); err != nil {
		return domain.NewExitError(ExitUsageError, "invalid input", err)
	}

	if choice < 1 || choice > len(available) {
		return domain.NewExitError(ExitUsageError, "invalid choice", nil)
	}

	selected := available[choice-1]

	return uc.Manager.Apply(ctx, selected)
}

// CommandExecutor consolidates ALL command execution patterns
// Eliminates 30+ duplicate exec.Command usages across files.
type CommandExecutor struct {
	verbose bool
	dryRun  bool
}

// NewCommandExecutor creates a new command executor.
func NewCommandExecutor(verbose, dryRun bool) *CommandExecutor {
	return &CommandExecutor{
		verbose: verbose,
		dryRun:  dryRun,
	}
}

// Execute runs command with unified patterns
// Replaces all exec.Command usages with consistent interface.
func (ce *CommandExecutor) Execute(ctx context.Context, name string, args ...string) error {
	if ce.dryRun {
		fmt.Printf("DRY RUN: %s %s\n", name, strings.Join(args, " "))

		return nil
	}

	return system.Run(ctx, ce.verbose, name, args...)
}

// ExecuteSudo runs a command with sudo privileges.
func (ce *CommandExecutor) ExecuteSudo(ctx context.Context, name string, args ...string) error {
	sudoArgs := append([]string{name}, args...)

	return ce.Execute(ctx, "sudo", sudoArgs...)
}

// ExecuteWithOutput runs a command and returns its output.
func (ce *CommandExecutor) ExecuteWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	if ce.dryRun {
		fmt.Printf("DRY RUN: %s %s\n", name, strings.Join(args, " "))

		return "", nil
	}

	return system.RunWithOutput(ctx, name, args...)
}

// ExecuteSilent runs a command silently without output.
func (ce *CommandExecutor) ExecuteSilent(ctx context.Context, name string, args ...string) error {
	if ce.dryRun {
		return nil
	}

	return system.RunSilent(ctx, name, args...)
}

// CommandExists checks if a command is available in the system.
func (ce *CommandExecutor) CommandExists(name string) bool {
	return system.CommandExists(name)
}

// ServiceController consolidates systemctl operations
// Replaces duplicate systemctl patterns across managers.
type ServiceController struct {
	executor *CommandExecutor
}

// NewServiceController creates a new service controller for systemctl operations.
func NewServiceController(verbose, dryRun bool) *ServiceController {
	return &ServiceController{
		executor: NewCommandExecutor(verbose, dryRun),
	}
}

// IsActive checks if a systemd service is active.
func (sc *ServiceController) IsActive(ctx context.Context, serviceName string) bool {
	err := sc.executor.ExecuteSilent(ctx, "systemctl", "is-active", "--quiet", serviceName)

	return err == nil
}

// Enable marks a systemd service to start at boot.
func (sc *ServiceController) Enable(ctx context.Context, serviceName string) error {
	return sc.executor.ExecuteSudo(ctx, "systemctl", "enable", serviceName)
}

// Start starts a systemd service.
func (sc *ServiceController) Start(ctx context.Context, serviceName string) error {
	return sc.executor.ExecuteSudo(ctx, "systemctl", "start", serviceName)
}

// Status gets the status of a systemd service.
func (sc *ServiceController) Status(ctx context.Context, serviceName string) (string, error) {
	return sc.executor.ExecuteWithOutput(ctx, "systemctl", "status", serviceName)
}

// GetProperty gets a specific property of a systemd service.
func (sc *ServiceController) GetProperty(ctx context.Context, serviceName, property string) (string, error) {
	return sc.executor.ExecuteWithOutput(ctx, "systemctl", "show", serviceName, "--property="+property, "--value")
}
