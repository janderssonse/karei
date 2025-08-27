// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package unit

import (
	"os"
	"strings"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CommandSequence represents a sequence of commands to execute.
type CommandSequence struct {
	Name string
	Args []string
}

// CommandGenerator generates installation commands without executing them.
type CommandGenerator struct {
	verbose bool
	dryRun  bool
	tmpDir  string
	binDir  string
}

func NewCommandGenerator(verbose, dryRun bool) *CommandGenerator {
	return &CommandGenerator{
		verbose: verbose,
		dryRun:  dryRun,
		tmpDir:  os.TempDir(),
		binDir:  "/usr/local/bin",
	}
}

// NewCommandGeneratorWithPaths creates a CommandGenerator with custom paths for testing.
func NewCommandGeneratorWithPaths(verbose, dryRun bool, tmpDir, binDir string) *CommandGenerator {
	return &CommandGenerator{
		verbose: verbose,
		dryRun:  dryRun,
		tmpDir:  tmpDir,
		binDir:  binDir,
	}
}

// GenerateInstallCommands creates command sequences for package installation.
func (cg *CommandGenerator) GenerateInstallCommands(pkg *domain.Package) []CommandSequence {
	var commands []CommandSequence

	switch pkg.Method {
	case domain.MethodAPT:
		commands = append(commands, CommandSequence{
			Name: "sudo",
			Args: []string{"apt", "update", "-y"},
		})
		commands = append(commands, CommandSequence{
			Name: "sudo",
			Args: []string{"apt", "install", "-y", pkg.Source},
		})

	case domain.MethodGitHub:
		// Extract repo from source like "jesseduffield/lazygit"
		repoName := extractRepoName(pkg.Source)
		commands = append(commands, CommandSequence{
			Name: "wget",
			Args: []string{"-O", cg.tmpDir + "/" + repoName + ".tar.gz",
				"https://github.com/" + pkg.Source + "/releases/latest/download/" + repoName + "_*_Linux_x86_64.tar.gz"},
		})
		commands = append(commands, CommandSequence{
			Name: "tar",
			Args: []string{"-xzf", cg.tmpDir + "/" + repoName + ".tar.gz", "-C", cg.tmpDir},
		})
		commands = append(commands, CommandSequence{
			Name: "sudo",
			Args: []string{"mv", cg.tmpDir + "/" + repoName, cg.binDir + "/"},
		})

	case domain.MethodFlatpak:
		commands = append(commands, CommandSequence{
			Name: "flatpak",
			Args: []string{"install", "-y", "flathub", pkg.Source},
		})

	case domain.MethodSnap:
		commands = append(commands, CommandSequence{
			Name: "sudo",
			Args: []string{"snap", "install", pkg.Source},
		})

	case domain.MethodDEB:
		packageName := extractPackageName(pkg.Source)
		commands = append(commands, CommandSequence{
			Name: "wget",
			Args: []string{"-O", cg.tmpDir + "/" + packageName + ".deb", pkg.Source},
		})
		commands = append(commands, CommandSequence{
			Name: "sudo",
			Args: []string{"apt", "install", "-y", cg.tmpDir + "/" + packageName + ".deb"},
		})

	case domain.MethodScript:
		commands = append(commands, CommandSequence{
			Name: "bash",
			Args: []string{"-c", generateCustomScript(pkg.Source)},
		})
	}

	return commands
}

func extractRepoName(source string) string {
	// Extract "lazygit" from "jesseduffield/lazygit"
	parts := strings.Split(source, "/")
	if len(parts) >= 2 {
		return parts[1]
	}

	return source
}

func extractPackageName(url string) string {
	// Extract package name from URL
	if url == "" {
		return "package"
	}

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		// Remove .deb extension
		if len(filename) > 4 && filename[len(filename)-4:] == ".deb" {
			return filename[:len(filename)-4]
		}

		return filename
	}

	return "package"
}

func generateCustomScript(source string) string {
	switch source {
	case "fastfetch-install":
		return "sudo add-apt-repository -y ppa:zhangsongcui3371/fastfetch && sudo apt update -y && sudo apt install -y fastfetch"
	case "typora-install":
		return "wget -qO - https://typora.io/linux/public-key.asc | sudo tee /etc/apt/trusted.gpg.d/typora.asc && sudo add-apt-repository -y 'deb https://typora.io/linux ./' && sudo apt update -y && sudo apt install -y typora"
	default:
		return "echo 'Custom installation for " + source + "'"
	}
}

// Test cases for command generation.
func TestCommandGeneration_APT(t *testing.T) {
	t.Parallel()

	// Use isolated temporary directories for complete test isolation
	tmpDir := t.TempDir()
	binDir := t.TempDir()

	generator := NewCommandGeneratorWithPaths(false, true, tmpDir, binDir)
	require.NotNil(t, generator, "Generator should be created successfully")

	pkg := &domain.Package{
		Method: domain.MethodAPT,
		Source: "vim",
	}

	commands := generator.GenerateInstallCommands(pkg)

	// Use require for critical assertions that tests depend on
	require.NotEmpty(t, commands, "Commands should not be empty")
	require.Len(t, commands, 2, "APT installation should generate exactly 2 commands")

	expected := []CommandSequence{
		{Name: "sudo", Args: []string{"apt", "update", "-y"}},
		{Name: "sudo", Args: []string{"apt", "install", "-y", "vim"}},
	}

	// Use assert for comparison since test can continue if this fails
	assert.Equal(t, expected, commands)
}

func TestCommandGeneration_GitHub(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	binDir := t.TempDir()

	generator := NewCommandGeneratorWithPaths(false, true, tmpDir, binDir)
	pkg := &domain.Package{
		Method: domain.MethodGitHub,
		Source: "jesseduffield/lazygit",
	}

	commands := generator.GenerateInstallCommands(pkg)

	expected := []CommandSequence{
		{
			Name: "wget",
			Args: []string{"-O", tmpDir + "/lazygit.tar.gz",
				"https://github.com/jesseduffield/lazygit/releases/latest/download/lazygit_*_Linux_x86_64.tar.gz"},
		},
		{Name: "tar", Args: []string{"-xzf", tmpDir + "/lazygit.tar.gz", "-C", tmpDir}},
		{Name: "sudo", Args: []string{"mv", tmpDir + "/lazygit", binDir + "/"}},
	}

	assert.Equal(t, expected, commands)
}

func TestCommandGeneration_Flatpak(t *testing.T) {
	t.Parallel()

	generator := NewCommandGenerator(false, true)
	pkg := &domain.Package{
		Method: domain.MethodFlatpak,
		Source: "com.brave.Browser",
	}

	commands := generator.GenerateInstallCommands(pkg)

	expected := []CommandSequence{
		{Name: "flatpak", Args: []string{"install", "-y", "flathub", "com.brave.Browser"}},
	}

	assert.Equal(t, expected, commands)
}

func TestCommandGeneration_DEB(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	binDir := t.TempDir()

	generator := NewCommandGeneratorWithPaths(false, true, tmpDir, binDir)
	pkg := &domain.Package{
		Method: domain.MethodDEB,
		Source: "https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb",
	}

	commands := generator.GenerateInstallCommands(pkg)

	expected := []CommandSequence{
		{
			Name: "wget",
			Args: []string{"-O", tmpDir + "/google-chrome-stable_current_amd64.deb",
				"https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb"},
		},
		{Name: "sudo", Args: []string{"apt", "install", "-y", tmpDir + "/google-chrome-stable_current_amd64.deb"}},
	}

	assert.Equal(t, expected, commands)
}

func TestCommandGeneration_CustomScript(t *testing.T) {
	t.Parallel()

	generator := NewCommandGenerator(false, true)
	pkg := &domain.Package{
		Method: domain.MethodScript,
		Source: "fastfetch-install",
	}

	commands := generator.GenerateInstallCommands(pkg)

	expected := []CommandSequence{
		{
			Name: "bash",
			Args: []string{"-c",
				"sudo add-apt-repository -y ppa:zhangsongcui3371/fastfetch && sudo apt update -y && sudo apt install -y fastfetch"},
		},
	}

	assert.Equal(t, expected, commands)
}

func TestCommandGeneration_AllMethods(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	binDir := t.TempDir()

	generator := NewCommandGeneratorWithPaths(false, true, tmpDir, binDir)

	testCases := []struct {
		name    string
		pkg     *domain.Package
		minCmds int // Minimum expected commands
	}{
		{
			name:    "APT package",
			pkg:     &domain.Package{Method: domain.MethodAPT, Source: "btop"},
			minCmds: 2,
		},
		{
			name:    "GitHub release",
			pkg:     &domain.Package{Method: domain.MethodGitHub, Source: "zellij-org/zellij"},
			minCmds: 3,
		},
		{
			name:    "Flatpak app",
			pkg:     &domain.Package{Method: domain.MethodFlatpak, Source: "org.signal.Signal"},
			minCmds: 1,
		},
		{
			name:    "DEB package",
			pkg:     &domain.Package{Method: domain.MethodDEB, Source: "https://example.com/package.deb"},
			minCmds: 2,
		},
		{
			name:    "Custom script",
			pkg:     &domain.Package{Method: domain.MethodScript, Source: "typora-install"},
			minCmds: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			commands := generator.GenerateInstallCommands(testCase.pkg)
			assert.GreaterOrEqual(t, len(commands), testCase.minCmds,
				"Expected at least %d commands for %s", testCase.minCmds, testCase.name)

			// Verify all commands have names
			for i, cmd := range commands {
				assert.NotEmpty(t, cmd.Name, "Command %d should have a name", i)
			}
		})
	}
}

func TestPackageValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		pkg     *domain.Package
		isValid bool
	}{
		{
			name:    "Valid APT package",
			pkg:     &domain.Package{Method: domain.MethodAPT, Source: "vim"},
			isValid: true,
		},
		{
			name:    "Valid GitHub package",
			pkg:     &domain.Package{Method: domain.MethodGitHub, Source: "user/repo"},
			isValid: true,
		},
		{
			name:    "Empty source",
			pkg:     &domain.Package{Method: domain.MethodAPT, Source: ""},
			isValid: false,
		},
		{
			name:    "Invalid GitHub format",
			pkg:     &domain.Package{Method: domain.MethodGitHub, Source: "invalid"},
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			isValid := validatePackage(tc.pkg)
			assert.Equal(t, tc.isValid, isValid, "Package validation mismatch")
		})
	}
}

func validatePackage(pkg *domain.Package) bool {
	if pkg.Source == "" {
		return false
	}

	switch pkg.Method {
	case domain.MethodGitHub:
		parts := strings.Split(pkg.Source, "/")

		return len(parts) >= 2 && parts[0] != "" && parts[1] != ""
	default:
		return true
	}
}

func TestExtractRepoName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    string
		expected string
	}{
		{"jesseduffield/lazygit", "lazygit"},
		{"zellij-org/zellij", "zellij"},
		{"single", "single"},
		{"user/repo/extra", "repo"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := extractRepoName(tc.input)
		assert.Equal(t, tc.expected, result, "Failed to extract repo name from %s", tc.input)
	}
}

func TestExtractPackageName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    string
		expected string
	}{
		{"https://example.com/package.deb", "package"},
		{"https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb", "google-chrome-stable_current_amd64"},
		{"simple.deb", "simple"},
		{"no-extension", "no-extension"},
		{"", "package"},
	}

	for _, tc := range testCases {
		result := extractPackageName(tc.input)
		assert.Equal(t, tc.expected, result, "Failed to extract package name from %s", tc.input)
	}
}

// TestCommandGeneration_TableDriven_Functional demonstrates functional patterns: table-driven testing,
// isolated temp dirs, require for critical checks, no network calls, thread-safe.
func TestCommandGeneration_TableDriven_Functional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		method           domain.InstallMethod
		source           string
		expectedCount    int
		expectedBinary   string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:             "APT package installation",
			method:           domain.MethodAPT,
			source:           "curl",
			expectedCount:    2,
			expectedBinary:   "sudo",
			shouldContain:    []string{"apt", "update", "install", "curl"},
			shouldNotContain: []string{"wget", "http", "github"},
		},
		{
			name:             "Flatpak application",
			method:           domain.MethodFlatpak,
			source:           "org.gimp.GIMP",
			expectedCount:    1,
			expectedBinary:   "flatpak",
			shouldContain:    []string{"flatpak", "install", "flathub", "org.gimp.GIMP"},
			shouldNotContain: []string{"sudo", "apt", "wget"},
		},
		{
			name:             "Snap package",
			method:           domain.MethodSnap,
			source:           "code",
			expectedCount:    1,
			expectedBinary:   "sudo",
			shouldContain:    []string{"sudo", "snap", "install", "code"},
			shouldNotContain: []string{"apt", "flatpak", "wget"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Each test gets isolated temporary directories
			tmpDir := t.TempDir()
			binDir := t.TempDir()

			// Create fresh generator for complete isolation
			generator := NewCommandGeneratorWithPaths(false, true, tmpDir, binDir)
			require.NotNil(t, generator, "Generator creation should never fail")

			pkg := &domain.Package{
				Method: testCase.method,
				Source: testCase.source,
			}
			require.NotNil(t, pkg, "Package should be valid")

			// Generate commands with no network access
			commands := generator.GenerateInstallCommands(pkg)

			// Use require for fundamental assertions
			require.Len(t, commands, testCase.expectedCount,
				"Method %s should generate exactly %d commands", testCase.method, testCase.expectedCount)
			require.NotEmpty(t, commands, "Commands should never be empty")

			// Verify first command uses expected binary
			require.Equal(t, testCase.expectedBinary, commands[0].Name,
				"First command should use %s binary", testCase.expectedBinary)

			// Convert commands to string for content verification
			commandStr := ""
			for _, cmd := range commands {
				commandStr += cmd.Name + " " + strings.Join(cmd.Args, " ") + " "
			}

			// Verify expected content is present
			for _, expected := range testCase.shouldContain {
				assert.Contains(t, commandStr, expected,
					"Commands should contain '%s' for method %s", expected, testCase.method)
			}

			// Verify forbidden content is absent (security check)
			for _, forbidden := range testCase.shouldNotContain {
				assert.NotContains(t, commandStr, forbidden,
					"Commands should NOT contain '%s' for method %s", forbidden, testCase.method)
			}

			// Verify no dangerous or harmful commands
			assert.NotContains(t, commandStr, "rm -rf /",
				"Commands should never contain dangerous filesystem operations")
			assert.NotContains(t, commandStr, "curl | sh",
				"Commands should never pipe downloads directly to shell")
		})
	}
}
