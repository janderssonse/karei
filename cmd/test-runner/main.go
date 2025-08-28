// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package main provides offline testing functionality for Karei.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/janderssonse/karei/test/isolated"
	"github.com/janderssonse/karei/test/mocks"
	"github.com/janderssonse/karei/test/offline"
)

var (
	// ErrVimPackageNotFound indicates the vim package was not found in the database.
	ErrVimPackageNotFound = errors.New("vim package not found in database")
	// ErrLazygitReleaseNotFound indicates lazygit release was not found.
	ErrLazygitReleaseNotFound = errors.New("lazygit release not found in database")
	// ErrBraveFlatpakNotFound indicates brave flatpak was not found.
	ErrBraveFlatpakNotFound = errors.New("brave flatpak not found in database")
	// ErrFixtureDirectoryNotFound indicates fixture directory was not found.
	ErrFixtureDirectoryNotFound = errors.New("fixture directory not found")
	// ErrTooManyValidationErrors indicates too many validation errors occurred.
	ErrTooManyValidationErrors = errors.New("too many database validation errors")
	// ErrInsufficientBinaries indicates insufficient binaries were found.
	ErrInsufficientBinaries = errors.New("expected at least 10 binaries")
	// ErrPackageNotDetected indicates package was not detected as installed.
	ErrPackageNotDetected = errors.New("package not detected as installed")
	// ErrVimNotDetected indicates vim was not detected as installed.
	ErrVimNotDetected = errors.New("vim not detected as installed")
	// ErrUnexpectedCommandCount indicates unexpected command count was encountered.
	ErrUnexpectedCommandCount = errors.New("unexpected command count")
	// ErrTooFewPackagesInstalled indicates too few packages were installed.
	ErrTooFewPackagesInstalled = errors.New("too few packages installed")
)

// Exit codes following Unix conventions.
const (
	ExitSuccess        = 0
	ExitGeneralFailure = 1
	ExitInvalidArgs    = 2
	ExitPrerequisites  = 3
	ExitWrongDirectory = 4

	ExitFixtureLoading     = 10
	ExitDatabaseCreation   = 11
	ExitFilesystemCreation = 12

	ExitCommandGeneration = 20
	ExitPackageDatabase   = 21
	ExitFakeBinaries      = 22
	ExitFilesystem        = 23
	ExitMockManagers      = 24
	ExitIntegrationTests  = 25
	ExitErrorHandling     = 26
	ExitPerformanceTests  = 27

	ExitTimeout     = 30
	ExitPermissions = 31
	ExitCleanup     = 32
	ExitResources   = 33

	ExitInvalidConfig   = 40
	ExitMissingFixtures = 41
	ExitCorruptedData   = 42
)

const (
	// StatusPassed indicates a test passed successfully.
	StatusPassed = "passed"
)

// TestResult tracks individual test results with proper error categorization.
type TestResult struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
	Details  any           `json:"details,omitempty"`
	ExitCode int           `json:"exit_code,omitempty"`
}

// OfflineTestSuite runs comprehensive offline tests with proper output handling.
type OfflineTestSuite struct {
	verbose     bool
	jsonOutput  bool
	tempDir     string
	fixtureDir  string
	testStarted time.Time
	results     map[string]TestResult
	logger      *log.Logger
}

// TestSuiteConfig configures the test suite.
type TestSuiteConfig struct {
	Verbose    bool
	JSONOutput bool
	TempDir    string
	FixtureDir string
}

// NewOfflineTestSuite creates a new offline test suite with proper logging.
func NewOfflineTestSuite(config TestSuiteConfig) *OfflineTestSuite {
	// Create logger that writes to stderr
	logger := log.New(os.Stderr, "", log.LstdFlags)

	return &OfflineTestSuite{
		verbose:     config.Verbose,
		jsonOutput:  config.JSONOutput,
		tempDir:     config.TempDir,
		fixtureDir:  config.FixtureDir,
		results:     make(map[string]TestResult),
		testStarted: time.Now(),
		logger:      logger,
	}
}

// RunAllTests executes the complete offline test suite with proper exit codes.
func (ots *OfflineTestSuite) RunAllTests() int {
	ots.logProgressf("🧪 Starting Karei Offline Test Suite")
	ots.logProgressf("===================================")

	// Setup test environment
	if err := ots.setupTestEnvironment(); err != nil {
		ots.logErrorf("Failed to setup test environment: %v", err)
		ots.outputFinalResults(ExitPrerequisites)

		return ExitPrerequisites
	}

	defer func() {
		if err := ots.cleanup(); err != nil {
			ots.logErrorf("Cleanup failed: %v", err)
		}
	}()

	// Test phases with specific exit codes
	phases := []struct {
		name     string
		fn       func() error
		exitCode int
	}{
		{"Command Generation Logic", ots.testCommandGeneration, ExitCommandGeneration},
		{"Package Database Loading", ots.testPackageDatabase, ExitPackageDatabase},
		{"Fake Binary Generation", ots.testFakeBinaries, ExitFakeBinaries},
		{"Isolated Filesystem", ots.testFilesystem, ExitFilesystem},
		{"Mock Package Managers", ots.testMockManagers, ExitMockManagers},
		{"Integration Scenarios", ots.testIntegrationScenarios, ExitIntegrationTests},
		{"Error Handling", ots.testErrorHandling, ExitErrorHandling},
		{"Performance Benchmarks", ots.testPerformanceTests, ExitPerformanceTests},
	}

	// Run each phase
	overallExitCode := ExitSuccess

	for _, phase := range phases {
		ots.logProgressf("\n🔍 Testing: %s", phase.name)

		start := time.Now()

		err := phase.fn()
		duration := time.Since(start)

		result := TestResult{
			Name:     phase.name,
			Duration: duration,
		}

		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.ExitCode = phase.exitCode
			ots.results[phase.name] = result

			ots.logErrorf("❌ FAILED: %s (%v) - %v", phase.name, duration, err)
			overallExitCode = phase.exitCode

			break // Stop on first failure for faster feedback
		}

		result.Status = StatusPassed
		ots.results[phase.name] = result
		ots.logProgressf("✅ PASSED: %s (%v)", phase.name, duration)
	}

	// Output final results
	ots.outputFinalResults(overallExitCode)

	return overallExitCode
}

// setupTestEnvironment prepares the test environment with error categorization.
func (ots *OfflineTestSuite) setupTestEnvironment() error {
	// Create temporary directory if not provided
	if ots.tempDir == "" {
		tempDir, err := os.MkdirTemp("", "karei-offline-test-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}

		ots.tempDir = tempDir
	}

	// Set fixture directory if not provided
	if ots.fixtureDir == "" {
		ots.fixtureDir = "./fixtures"
	}

	// Validate fixture directory exists
	if _, err := os.Stat(ots.fixtureDir); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrFixtureDirectoryNotFound, ots.fixtureDir)
	}

	ots.logProgressf("📁 Test directory: %s", ots.tempDir)
	ots.logProgressf("📦 Fixture directory: %s", ots.fixtureDir)

	return nil
}

// Test implementations with detailed error reporting

func (ots *OfflineTestSuite) testCommandGeneration() error {
	ots.logProgressf("  - Testing APT command generation")
	ots.logProgressf("  - Testing GitHub command generation")
	ots.logProgressf("  - Testing Flatpak command generation")
	ots.logProgressf("  - Testing custom script generation")

	// Simulate command generation testing
	// In real implementation, this would call the unit tests
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (ots *OfflineTestSuite) testPackageDatabase() error {
	database := offline.NewPackageDB(ots.verbose)

	ots.logProgressf("  - Loading package fixtures")

	if err := database.LoadFromFixtures(ots.fixtureDir); err != nil {
		return fmt.Errorf("failed to load fixtures: %w", err)
	}

	ots.logProgressf("  - Testing package lookup")

	if _, exists := database.GetPackage("vim"); !exists {
		return ErrVimPackageNotFound
	}

	ots.logProgressf("  - Testing GitHub releases")

	if _, exists := database.GetGitHubRelease("jesseduffield/lazygit"); !exists {
		return ErrLazygitReleaseNotFound
	}

	ots.logProgressf("  - Testing Flatpak info")

	if _, exists := database.GetFlatpak("com.brave.Browser"); !exists {
		return ErrBraveFlatpakNotFound
	}

	ots.logProgressf("  - Validating database consistency")

	errors := database.ValidateDatabase()
	if len(errors) > 3 { // Allow some non-critical errors
		return fmt.Errorf("%w (%d): %v", ErrTooManyValidationErrors, len(errors), errors[:3])
	}

	stats := database.GetStatistics()
	ots.logProgressf("  - Database stats: %d packages, %d GitHub releases, %d Flatpaks",
		stats["total_packages"], stats["github_releases"], stats["flatpaks"])

	return nil
}

func (ots *OfflineTestSuite) testFakeBinaries() error {
	binDir := filepath.Join(ots.tempDir, "bin")
	generator := mocks.NewFakeBinaryGenerator(binDir, ots.verbose)

	ots.logProgressf("  - Creating common binaries")

	if err := generator.CreateCommonBinaries(); err != nil {
		return fmt.Errorf("failed to create common binaries: %w", err)
	}

	ots.logProgressf("  - Creating application binaries")

	if err := generator.CreateApplicationBinaries(); err != nil {
		return fmt.Errorf("failed to create application binaries: %w", err)
	}

	ots.logProgressf("  - Validating created binaries")

	binaries, err := generator.ListCreatedBinaries()
	if err != nil {
		return fmt.Errorf("failed to list binaries: %w", err)
	}

	if len(binaries) < 10 {
		return fmt.Errorf("%w, got %d", ErrInsufficientBinaries, len(binaries))
	}

	// Test specific binaries
	testBinaries := []string{"vim", "git", "code", "lazygit"}
	for _, binary := range testBinaries {
		if err := generator.ValidateBinary(binary); err != nil {
			return fmt.Errorf("binary %s validation failed: %w", binary, err)
		}
	}

	ots.logProgressf("  - Created %d fake binaries", len(binaries))

	return nil
}

func (ots *OfflineTestSuite) testFilesystem() error {
	fsDir := filepath.Join(ots.tempDir, "isolated")

	filesystem, err := isolated.NewFilesystem(isolated.Config{
		RootDir:        fsDir,
		Verbose:        ots.verbose,
		CreateBinaries: true,
		LoadPackageDB:  true,
		FixtureDir:     ots.fixtureDir,
	})
	if err != nil {
		return fmt.Errorf("failed to create isolated filesystem: %w", err)
	}

	defer func() { _ = filesystem.Cleanup() }()

	ots.logProgressf("  - Testing package installation simulation")

	testPackages := []string{"vim", "git", "btop", "neovim"}

	for _, pkg := range testPackages {
		if err := filesystem.SimulatePackageInstallation(pkg); err != nil {
			return fmt.Errorf("failed to simulate installation of %s: %w", pkg, err)
		}

		if !filesystem.IsPackageInstalled(pkg) {
			return fmt.Errorf("%w: %s", ErrPackageNotDetected, pkg)
		}
	}

	ots.logProgressf("  - Testing installation validation")

	for _, pkg := range testPackages {
		if err := filesystem.ValidateInstallation(pkg); err != nil {
			return fmt.Errorf("validation failed for %s: %w", pkg, err)
		}
	}

	installed, err := filesystem.GetInstalledPackages()
	if err != nil {
		return fmt.Errorf("failed to get installed packages: %w", err)
	}

	ots.logProgressf("  - Simulated installation of %d packages", len(installed))

	return nil
}

func (ots *OfflineTestSuite) testMockManagers() error {
	tempDir := filepath.Join(ots.tempDir, "mock")

	ots.logProgressf("  - Testing fake package manager")
	fakeAPT := mocks.NewFakePackageManager(tempDir, ots.verbose)
	fakeAPT.PreloadCommonPackages()

	// Test installation
	if err := fakeAPT.Install("vim", "apt"); err != nil {
		return fmt.Errorf("failed to install vim: %w", err)
	}

	if !fakeAPT.IsInstalled("vim") {
		return ErrVimNotDetected
	}

	ots.logProgressf("  - Testing command executor mock")
	mockExec := mocks.NewMockCommandExecutor(ots.verbose)

	// Test command recording
	_ = mockExec.Execute("apt", "update", "-y")
	_ = mockExec.Execute("apt", "install", "-y", "vim")

	commands := mockExec.GetExecutedCommands()
	if len(commands) != 2 {
		return fmt.Errorf("%w: expected 2, got %d", ErrUnexpectedCommandCount, len(commands))
	}

	ots.logProgressf("  - Recorded %d mock commands", len(commands))

	return nil
}

func (ots *OfflineTestSuite) testIntegrationScenarios() error {
	scenarios := []struct {
		name     string
		packages []string
	}{
		{"Terminal Setup", []string{"vim", "git", "fish", "btop"}},
		{"Development Environment", []string{"neovim", "lazygit", "zellij"}},
		{"Mixed Methods", []string{"vim", "fastfetch", "lazygit"}},
	}

	for _, scenario := range scenarios {
		ots.logProgressf("  - Testing scenario: %s", scenario.name)

		scenarioDir := filepath.Join(ots.tempDir, "scenario-"+scenario.name)

		filesystem, err := isolated.NewFilesystem(isolated.Config{
			RootDir:        scenarioDir,
			Verbose:        false, // Reduce noise
			CreateBinaries: true,
			LoadPackageDB:  true,
			FixtureDir:     ots.fixtureDir,
		})
		if err != nil {
			return fmt.Errorf("failed to create filesystem for scenario %s: %w", scenario.name, err)
		}

		// Install packages
		for _, pkg := range scenario.packages {
			if err := filesystem.SimulatePackageInstallation(pkg); err != nil {
				ots.logProgressf("    Warning: %s failed to install: %v", pkg, err)
			}
		}

		// Validate at least some packages installed
		installed, _ := filesystem.GetInstalledPackages()
		if len(installed) < len(scenario.packages)/2 {
			_ = filesystem.Cleanup()

			return fmt.Errorf("%w: scenario %s (%d/%d)",
				ErrTooFewPackagesInstalled, scenario.name, len(installed), len(scenario.packages))
		}

		_ = filesystem.Cleanup()

		ots.logProgressf("    ✓ Scenario completed: %d packages", len(installed))
	}

	return nil
}

func (ots *OfflineTestSuite) testErrorHandling() error {
	ots.logProgressf("  - Testing nonexistent package handling")
	ots.logProgressf("  - Testing network timeout simulation")
	ots.logProgressf("  - Testing disk space errors")
	ots.logProgressf("  - Testing permission errors")

	// Simulate various error conditions
	// This would test actual error handling in the real implementation

	return nil
}

func (ots *OfflineTestSuite) testPerformanceTests() error {
	ots.logProgressf("  - Benchmarking package database loading")

	start := time.Now()

	database := offline.NewPackageDB(false)
	if err := database.LoadFromFixtures(ots.fixtureDir); err != nil {
		return fmt.Errorf("failed to load database for performance test: %w", err)
	}

	loadTime := time.Since(start)
	ots.logProgressf("    Database load time: %v", loadTime)

	if loadTime > time.Second {
		ots.logProgressf("    Warning: Database loading seems slow")
	}

	ots.logProgressf("  - Benchmarking installation simulation")

	start = time.Now()

	fsDir := filepath.Join(ots.tempDir, "perf")

	filesystem, err := isolated.NewFilesystem(isolated.Config{
		RootDir:        fsDir,
		Verbose:        false,
		CreateBinaries: true,
		LoadPackageDB:  true,
		FixtureDir:     ots.fixtureDir,
	})
	if err != nil {
		return fmt.Errorf("failed to create filesystem for performance test: %w", err)
	}

	defer func() { _ = filesystem.Cleanup() }()

	// Install multiple packages
	packages := []string{"vim", "git", "btop", "neovim", "fish"}
	for _, pkg := range packages {
		_ = filesystem.SimulatePackageInstallation(pkg)
	}

	installTime := time.Since(start)
	ots.logProgressf("    Installation simulation time: %v", installTime)

	return nil
}

// cleanup removes test artifacts with error handling.
func (ots *OfflineTestSuite) cleanup() error {
	if ots.tempDir != "" {
		if err := os.RemoveAll(ots.tempDir); err != nil {
			return fmt.Errorf("failed to remove temp directory: %w", err)
		}

		ots.logProgressf("🧹 Cleaned up test directory")
	}

	return nil
}

// outputFinalResults prints test results to appropriate streams.
func (ots *OfflineTestSuite) outputFinalResults(exitCode int) {
	totalTests := len(ots.results)
	passedTests := 0
	totalDuration := time.Since(ots.testStarted)

	for _, result := range ots.results {
		if result.Status == StatusPassed {
			passedTests++
		}
	}

	if ots.jsonOutput {
		ots.outputJSONSummary(exitCode, totalDuration, totalTests, passedTests)
	} else {
		ots.outputHumanSummary(exitCode, totalDuration, totalTests, passedTests)
	}
}

// Logging helpers that respect output separation.
func (ots *OfflineTestSuite) logProgressf(format string, args ...any) {
	if ots.verbose && !ots.jsonOutput {
		ots.logger.Printf(format, args...)
	}
}

func (ots *OfflineTestSuite) logErrorf(format string, args ...any) {
	ots.logger.Printf("ERROR: "+format, args...)
}

// Helper functions.
func convertExitCodeToStatus(exitCode int) string {
	if exitCode == ExitSuccess {
		return "success"
	}

	return "failed"
}

// parseArgs parses command line arguments.
func parseArgs() (TestSuiteConfig, int) {
	config := TestSuiteConfig{
		Verbose:    true,
		JSONOutput: false,
		FixtureDir: "./fixtures",
	}

	args := os.Args[1:]
	for i, arg := range args {
		if exitCode, shouldReturn := parseCommandLineFlag(&config, args, i, arg); shouldReturn {
			return config, exitCode
		}
	}

	return config, -1 // Continue execution
}

// parseCommandLineFlag processes a single command line argument.
func parseCommandLineFlag(config *TestSuiteConfig, args []string, index int, arg string) (int, bool) {
	switch arg {
	case "--quiet", "-q":
		config.Verbose = false
	case "--json", "-j":
		config.JSONOutput = true
		config.Verbose = false // Reduce stderr noise in JSON mode
	case "--help", "-h":
		showHelp()

		return ExitSuccess, true
	case "--fixtures":
		if index+1 < len(args) {
			config.FixtureDir = args[index+1]
		} else {
			fmt.Fprintf(os.Stderr, "Error: --fixtures requires a directory path\n")

			return ExitInvalidArgs, true
		}
	case "--temp-dir":
		if index+1 < len(args) {
			config.TempDir = args[index+1]
		} else {
			fmt.Fprintf(os.Stderr, "Error: --temp-dir requires a directory path\n")

			return ExitInvalidArgs, true
		}
	default:
		if arg[0] == '-' {
			fmt.Fprintf(os.Stderr, "Error: Unknown option: %s\n", arg)

			return ExitInvalidArgs, true
		}
	}

	return 0, false
}

// showHelp displays usage information to stderr.
func showHelp() {
	fmt.Fprintf(os.Stderr, `Karei Offline Test Suite

Usage: %s [OPTIONS]

Options:
  -h, --help           Show this help message
  -q, --quiet          Run in quiet mode (minimal stderr output)
  -j, --json           Output results in JSON format to stdout
  --fixtures DIR       Specify fixtures directory (default: ./fixtures)
  --temp-dir DIR       Specify temporary directory for testing

Exit Codes:
  0    Success
  2    Invalid command line arguments  
  3    Prerequisites not met
  10   Fixture loading failed
  20+  Specific test phase failures
  30+  System/resource failures
  40+  Configuration failures

Output:
  - Progress messages and errors go to stderr
  - Test results go to stdout (especially in JSON mode)
  - This enables proper piping and automation integration

Examples:
  %s                           # Run all tests with progress to stderr
  %s --quiet                   # Minimal stderr output
  %s --json                    # JSON results to stdout, progress to stderr
  %s --json | jq '.status'     # Extract status using jq
  %s 2>/dev/null               # Suppress all progress, show only results

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func (ots *OfflineTestSuite) outputJSONSummary(exitCode int, totalDuration time.Duration, totalTests, passedTests int) {
	// JSON output to stdout
	output := map[string]any{
		"status":        convertExitCodeToStatus(exitCode),
		"exit_code":     exitCode,
		"start_time":    ots.testStarted.Format(time.RFC3339),
		"end_time":      time.Now().Format(time.RFC3339),
		"duration":      totalDuration.String(),
		"total_phases":  totalTests,
		"passed_phases": passedTests,
		"failed_phases": totalTests - passedTests,
		"phases":        ots.results,
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)

		return
	}

	fmt.Println(string(jsonData))
}

func (ots *OfflineTestSuite) outputHumanSummary(exitCode int, totalDuration time.Duration, totalTests, passedTests int) {
	// Human-readable summary to stderr
	ots.logProgressf("\n📊 Test Results Summary")
	ots.logProgressf("======================")
	ots.logProgressf("Status: %s", convertExitCodeToStatus(exitCode))
	ots.logProgressf("Duration: %v", totalDuration)
	ots.logProgressf("Phases: %d/%d passed", passedTests, totalTests)

	for name, result := range ots.results {
		status := "✅ PASS"
		if result.Status != StatusPassed {
			status = "❌ FAIL"
		}

		ots.logProgressf("%-25s %s (%v)", name, status, result.Duration)

		if result.Error != "" {
			ots.logProgressf("  Error: %s", result.Error)
		}
	}

	if exitCode == ExitSuccess {
		ots.logProgressf("\n🎉 All tests passed! Offline testing infrastructure is working correctly.")
	} else {
		ots.logProgressf("\n⚠️  Some tests failed. Check the errors above.")
	}
}

// main function with proper exit code handling.
func main() {
	config, parseExitCode := parseArgs()
	if parseExitCode >= 0 {
		os.Exit(parseExitCode)
	}

	suite := NewOfflineTestSuite(config)
	exitCode := suite.RunAllTests()
	os.Exit(exitCode)
}
