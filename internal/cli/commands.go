// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cli "github.com/urfave/cli/v3"

	"github.com/janderssonse/karei/internal/adapters/platform"
	"github.com/janderssonse/karei/internal/config"
	"github.com/janderssonse/karei/internal/console"
	"github.com/janderssonse/karei/internal/system"
)

// Constants for verification and status strings.
const (
	verifyAll       = "all"
	statusMissing   = "missing"
	statusFound     = "found"
	statusInstall   = "install"
	statusFailed    = "failed"
	statusInstalled = "installed"
)

var (
	// ErrFail2BanNotActive indicates fail2ban service is not active.
	ErrFail2BanNotActive = errors.New("fail2ban service not active")
	// ErrUnknownSecurityTool indicates the security tool is not recognized.
	ErrUnknownSecurityTool = errors.New("unknown security tool")
	// ErrFishNotInstalled indicates the fish shell is not installed.
	ErrFishNotInstalled = errors.New("fish shell not installed")
)

// createSecurityCommand creates the security command directly.
func (app *CLI) createSecurityCommand() *cli.Command {
	return &cli.Command{
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
		ArgsUsage: "[tool]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) == 0 {
				return errors.New("security tool required")
			}
			return app.runSecurityTool(ctx, args[0])
		},
	}
}

// runSecurityTool executes the specified security tool.
func (app *CLI) runSecurityTool(ctx context.Context, tool string) error {
	commandRunner := platform.NewCommandRunner(app.verbose, false)

	switch tool {
	case "audit":
		return commandRunner.ExecuteSudo(ctx, "auditctl", "-l")
	case "firewall":
		return commandRunner.ExecuteSudo(ctx, "ufw", "status", "verbose")
	case "fail2ban":
		if isServiceActive(ctx, "fail2ban") {
			return commandRunner.ExecuteSudo(ctx, "fail2ban-client", "status")
		}
		return ErrFail2BanNotActive
	case "clamav":
		return commandRunner.Execute(ctx, "clamscan", "--version")
	case "rkhunter":
		return commandRunner.ExecuteSudo(ctx, "rkhunter", "--check", "--report-warnings-only")
	case "aide":
		return commandRunner.ExecuteSudo(ctx, "aide", "--check")
	default:
		return fmt.Errorf("%w: %s", ErrUnknownSecurityTool, tool)
	}
}

// createVerifyCommand creates the verify command directly.
func (app *CLI) createVerifyCommand() *cli.Command {
	return &cli.Command{
		Name:        "verify",
		Usage:       "Verify system configuration",
		Description: "Run verification checks",
		ArgsUsage:   "[what]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			what := verifyAll
			if len(args) > 0 {
				what = args[0]
			}
			return app.runVerification(ctx, what)
		},
	}
}

// runVerification runs the specified verification.
func (app *CLI) runVerification(ctx context.Context, what string) error {
	if what == verifyAll {
		return app.runAllVerifications(ctx)
	}

	verifier, ok := app.getVerifier(what)
	if !ok {
		return fmt.Errorf("unknown verification: %s", what)
	}

	return verifier(ctx)
}

// getVerifier returns the verification function for the given type.
func (app *CLI) getVerifier(what string) (func(context.Context) error, bool) {
	verifiers := map[string]func(context.Context) error{
		"tools":        app.verifyTools,
		"integrations": app.verifyIntegrations,
		"path":         app.verifyPath,
		"fish":         app.verifyFish,
		"xdg":          app.verifyXDG,
		"versions":     app.verifyVersions,
	}

	verifier, ok := verifiers[what]
	return verifier, ok
}

// runAllVerifications runs all verification checks.
func (app *CLI) runAllVerifications(ctx context.Context) error {
	verifiers := []func(context.Context) error{
		app.verifyTools,
		app.verifyIntegrations,
		app.verifyPath,
		app.verifyFish,
		app.verifyXDG,
		app.verifyVersions,
	}

	for _, verify := range verifiers {
		if err := verify(ctx); err != nil {
			return err
		}
		if !console.DefaultOutput.Plain {
			fmt.Fprintf(os.Stderr, "\n")
		}
	}
	return nil
}

// verifyTools verifies tool installations.
func (app *CLI) verifyTools(ctx context.Context) error {
	tools := map[string]string{
		"git":      "apt",
		"fish":     "apt",
		"starship": "aqua",
		"zellij":   "aqua",
		"btop":     "apt",
		"neovim":   "apt",
		"lazygit":  "aqua",
	}

	console.DefaultOutput.Progressf("Verifying tools...")

	for tool, method := range tools {
		var isInstalled bool

		switch method {
		case "aqua":
			if commandExists("aqua") {
				userLocal := filepath.Dir(config.GetUserBinDir())
				// #nosec G204 - tool is from a predefined map, not user input
				cmd := exec.CommandContext(ctx, "aqua", "which", tool)
				cmd.Env = os.Environ()
				cmd.Env = append(cmd.Env, "AQUA_ROOT_DIR="+userLocal)
				isInstalled = cmd.Run() == nil
			} else {
				isInstalled = commandExists(tool)
			}
		default:
			isInstalled = commandExists(tool)
		}

		if console.DefaultOutput.Plain {
			status := statusMissing
			if isInstalled {
				status = statusInstalled
			}
			console.DefaultOutput.PlainStatus(tool, status)
		} else {
			if isInstalled {
				console.DefaultOutput.Result("✓ " + tool)
			} else {
				console.DefaultOutput.Result(fmt.Sprintf("✗ %s - not found", tool))
			}
		}
	}

	return nil
}

// verifyIntegrations verifies config integrations.
func (app *CLI) verifyIntegrations(_ context.Context) error {
	console.DefaultOutput.Progressf("Verifying integrations...")

	configs := map[string]string{
		"fish":    filepath.Join(config.GetXDGConfigHome(), "fish", "config.fish"),
		"ghostty": filepath.Join(config.GetXDGConfigHome(), "ghostty", "config"),
		"btop":    filepath.Join(config.GetXDGConfigHome(), "btop", "btop.conf"),
	}

	for name, path := range configs {
		exists := system.FileExists(path)

		if console.DefaultOutput.Plain {
			status := statusMissing
			if exists {
				status = statusFound
			}
			console.DefaultOutput.PlainStatus(name+"-config", status)
		} else {
			if exists {
				console.DefaultOutput.Result(fmt.Sprintf("✓ %s config", name))
			} else {
				console.DefaultOutput.Result(fmt.Sprintf("✗ %s config - not found", name))
			}
		}
	}

	return nil
}

// verifyPath verifies PATH configuration.
func (app *CLI) verifyPath(_ context.Context) error {
	console.DefaultOutput.Progressf("Verifying PATH...")

	userBin := config.GetUserBinDir()
	pathEnv := os.Getenv("PATH")
	inPath := strings.Contains(pathEnv, userBin)

	if console.DefaultOutput.Plain {
		status := statusMissing
		if inPath {
			status = statusFound
		}
		console.DefaultOutput.PlainStatus("user-bin-path", status)
	} else {
		if inPath {
			console.DefaultOutput.Result("✓ User bin directory in PATH")
		} else {
			console.DefaultOutput.Result("✗ User bin directory not in PATH: " + userBin)
		}
	}

	return nil
}

// verifyFish verifies Fish shell configuration.
func (app *CLI) verifyFish(_ context.Context) error {
	console.DefaultOutput.Progressf("Verifying Fish shell...")

	if !commandExists("fish") {
		return ErrFishNotInstalled
	}

	fishConfig := filepath.Join(config.GetXDGConfigHome(), "fish", "config.fish")
	configExists := system.FileExists(fishConfig)

	if console.DefaultOutput.Plain {
		status := statusMissing
		if configExists {
			status = statusFound
		}
		console.DefaultOutput.PlainStatus("fish-config", status)
	} else {
		if configExists {
			console.DefaultOutput.Result("✓ Fish configuration found")
		} else {
			console.DefaultOutput.Result("✗ Fish configuration not found")
		}
	}

	return nil
}

// verifyXDG verifies XDG directories.
func (app *CLI) verifyXDG(_ context.Context) error {
	console.DefaultOutput.Progressf("Verifying XDG directories...")

	dirs := map[string]string{
		"CONFIG": config.GetXDGConfigHome(),
		"DATA":   config.GetXDGDataHome(),
	}

	for name, dir := range dirs {
		exists := system.IsDir(dir)
		app.reportXDGDirectory(name, dir, exists)
	}

	return nil
}

// reportXDGDirectory reports the status of an XDG directory.
func (app *CLI) reportXDGDirectory(name, dir string, exists bool) {
	if console.DefaultOutput.Plain {
		keyName := "xdg-" + strings.ToLower(name) + "-home"
		if exists {
			console.DefaultOutput.PlainKeyValue(keyName, dir)
		} else {
			console.DefaultOutput.PlainStatus(keyName, statusMissing)
		}
		return
	}

	if exists {
		console.DefaultOutput.Result(fmt.Sprintf("✓ XDG_%s_HOME: %s", name, dir))
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✗ XDG_%s_HOME: %s (not found)", name, dir))
	}
}

// verifyVersions verifies tool versions.
func (app *CLI) verifyVersions(ctx context.Context) error {
	console.DefaultOutput.Progressf("Verifying versions...")

	commandRunner := platform.NewCommandRunner(false, false)

	tools := map[string][]string{
		"Git":      {"git", "--version"},
		"Fish":     {"fish", "--version"},
		"Starship": {"starship", "--version"},
		"Neovim":   {"nvim", "--version"},
	}

	for name, cmd := range tools {
		output, err := commandRunner.ExecuteWithOutput(ctx, cmd[0], cmd[1:]...)
		app.reportToolVersion(name, output, err)
	}

	return nil
}

// reportToolVersion reports the version check result for a tool.
func (app *CLI) reportToolVersion(name, output string, err error) {
	keyName := strings.ToLower(name) + "-version"

	if err != nil {
		if console.DefaultOutput.Plain {
			console.DefaultOutput.PlainStatus(keyName, statusFailed)
		} else {
			console.DefaultOutput.Result(fmt.Sprintf("✗ %s: version check failed", name))
		}
		return
	}

	version := strings.Split(output, "\n")[0]
	if console.DefaultOutput.Plain {
		console.DefaultOutput.PlainKeyValue(keyName, version)
	} else {
		console.DefaultOutput.Result(fmt.Sprintf("✓ %s: %s", name, version))
	}
}

// createLogsCommand creates the logs command directly.
func (app *CLI) createLogsCommand() *cli.Command {
	return &cli.Command{
		Name:        "logs",
		Usage:       "View system logs",
		Description: "Display Karei installation and operation logs",
		ArgsUsage:   "[type]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			logType := verifyAll
			if len(args) > 0 {
				logType = args[0]
			}
			return app.showLogs(ctx, logType)
		},
	}
}

// showLogs displays the specified log type.
func (app *CLI) showLogs(ctx context.Context, logType string) error {
	logDir := filepath.Join(config.GetXDGDataHome(), "karei")

	switch logType {
	case statusInstall:
		return showLogFile(ctx, filepath.Join(logDir, "install.log"), "Installation")
	case "progress":
		return showLogFile(ctx, filepath.Join(logDir, "progress.log"), "Progress")
	case "precheck":
		return showLogFile(ctx, filepath.Join(logDir, "precheck.log"), "Precheck")
	case "errors":
		return showLogFile(ctx, filepath.Join(logDir, "errors.log"), "Errors")
	case verifyAll:
		logTypes := []struct {
			file string
			name string
		}{
			{"install.log", "Installation"},
			{"progress.log", "Progress"},
			{"precheck.log", "Precheck"},
			{"errors.log", "Errors"},
		}
		for _, lt := range logTypes {
			if err := showLogFile(ctx, filepath.Join(logDir, lt.file), lt.name); err != nil {
				return err
			}
			fmt.Println()
		}
		return nil
	default:
		return fmt.Errorf("unknown log type: %s", logType)
	}
}

// showLogFile displays the contents of a log file.
func showLogFile(ctx context.Context, path, name string) error {
	fmt.Printf("▸ %s Logs (%s):\n", name, path)

	if !system.FileExists(path) {
		fmt.Printf("No %s logs found\n", strings.ToLower(name))
		return nil
	}

	commandRunner := platform.NewCommandRunner(false, false)
	output, err := commandRunner.ExecuteWithOutput(ctx, "tail", "-n", "20", path)
	if err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	fmt.Println(output)
	return nil
}

// Helper functions

// commandExists checks if a command is available in PATH.
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// isServiceActive checks if a systemd service is active.
func isServiceActive(ctx context.Context, service string) bool {
	commandRunner := platform.NewCommandRunner(false, false)
	err := commandRunner.Execute(ctx, "systemctl", "is-active", "--quiet", service)
	return err == nil
}
