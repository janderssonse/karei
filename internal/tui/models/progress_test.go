// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package models

import (
	"context"
	"testing"

	"github.com/janderssonse/karei/internal/tui/styles"
	"github.com/janderssonse/karei/internal/uninstall"
)

func TestParseDpkgProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		dpkgOutput    string
		appName       string
		expectedProg  float64
		expectedMsg   string
		expectedFound bool
	}{
		// Real dpkg output phases (using LC_ALL=C) - UPDATED for better timing
		{
			name:          "Selecting package",
			dpkgOutput:    "Selecting previously unselected package google-chrome-stable.",
			appName:       "Chrome",
			expectedProg:  0.62,
			expectedMsg:   "Selecting Chrome package",
			expectedFound: true,
		},
		{
			name:          "Reading database",
			dpkgOutput:    "Reading database ... 285432 files and directories currently installed.)",
			appName:       "Chrome",
			expectedProg:  0.65,
			expectedMsg:   "Reading package database",
			expectedFound: true,
		},
		{
			name:          "Preparing to unpack",
			dpkgOutput:    "Preparing to unpack /tmp/google-chrome-stable.deb ...",
			appName:       "Chrome",
			expectedProg:  0.68,
			expectedMsg:   "Preparing to unpack Chrome",
			expectedFound: true,
		},
		{
			name:          "Unpacking",
			dpkgOutput:    "Unpacking google-chrome-stable (122.0.6261.57-1) ...",
			appName:       "Chrome",
			expectedProg:  0.72,
			expectedMsg:   "Unpacking Chrome package",
			expectedFound: true,
		},
		{
			name:          "Setting up",
			dpkgOutput:    "Setting up google-chrome-stable (122.0.6261.57-1) ...",
			appName:       "Chrome",
			expectedProg:  0.75,
			expectedMsg:   "Setting up Chrome",
			expectedFound: true,
		},
		{
			name:          "Update alternatives",
			dpkgOutput:    "update-alternatives: using /usr/bin/google-chrome-stable to provide /usr/bin/x-www-browser",
			appName:       "Chrome",
			expectedProg:  0.92,
			expectedMsg:   "Configuring Chrome alternatives",
			expectedFound: true,
		},
		{
			name:          "Processing mailcap triggers",
			dpkgOutput:    "Processing triggers for mailcap (3.70+nmu1ubuntu1) ...",
			appName:       "Chrome",
			expectedProg:  0.96,
			expectedMsg:   "Processing MIME type triggers",
			expectedFound: true,
		},
		{
			name:          "Processing gnome-menus triggers",
			dpkgOutput:    "Processing triggers for gnome-menus (3.36.0-1ubuntu3) ...",
			appName:       "Chrome",
			expectedProg:  0.97,
			expectedMsg:   "Processing GNOME menu triggers",
			expectedFound: true,
		},
		{
			name:          "Processing desktop-file-utils triggers",
			dpkgOutput:    "Processing triggers for desktop-file-utils (0.26-1ubuntu3) ...",
			appName:       "Chrome",
			expectedProg:  0.98,
			expectedMsg:   "Processing desktop file triggers",
			expectedFound: true,
		},
		{
			name:          "Processing man-db triggers",
			dpkgOutput:    "Processing triggers for man-db (2.10.2-1) ...",
			appName:       "Chrome",
			expectedProg:  0.99,
			expectedMsg:   "Processing manual page triggers",
			expectedFound: true,
		},
		{
			name:          "Processing menu triggers - completion",
			dpkgOutput:    "Processing triggers for menu (2.1.49) ...",
			appName:       "Chrome",
			expectedProg:  1.0,
			expectedMsg:   "Processing menu triggers",
			expectedFound: true,
		},

		// Test with different package names to ensure genericity
		{
			name:          "VSCode selecting package",
			dpkgOutput:    "Selecting previously unselected package code.",
			appName:       "VSCode",
			expectedProg:  0.62,
			expectedMsg:   "Selecting VSCode package",
			expectedFound: true,
		},
		{
			name:          "VSCode setting up",
			dpkgOutput:    "Setting up code (1.85.2-1705473751) ...",
			appName:       "VSCode",
			expectedProg:  0.75,
			expectedMsg:   "Setting up VSCode",
			expectedFound: true,
		},
		{
			name:          "Firefox unpacking",
			dpkgOutput:    "Unpacking firefox (122.0+build2-0ubuntu0.22.04.1) ...",
			appName:       "Firefox",
			expectedProg:  0.72,
			expectedMsg:   "Unpacking Firefox package",
			expectedFound: true,
		},

		// Test whitespace handling
		{
			name:          "Output with leading whitespace",
			dpkgOutput:    "   Setting up google-chrome-stable (122.0.6261.57-1) ...   ",
			appName:       "Chrome",
			expectedProg:  0.75,
			expectedMsg:   "Setting up Chrome",
			expectedFound: true,
		},

		// Test non-matching lines (should return false)
		{
			name:          "Random dpkg output",
			dpkgOutput:    "dpkg: warning: files list file for package 'libgtk-3-common' missing",
			appName:       "Chrome",
			expectedProg:  0,
			expectedMsg:   "",
			expectedFound: false,
		},
		{
			name:          "Empty line",
			dpkgOutput:    "",
			appName:       "Chrome",
			expectedProg:  0,
			expectedMsg:   "",
			expectedFound: false,
		},
		{
			name:          "Non-installation output",
			dpkgOutput:    "dpkg-query: no packages found matching something",
			appName:       "Chrome",
			expectedProg:  0,
			expectedMsg:   "",
			expectedFound: false,
		},

		// Test edge cases with unusual app names
		{
			name:          "App name with spaces",
			dpkgOutput:    "Setting up my-complex-app-name (1.0.0) ...",
			appName:       "My Complex App",
			expectedProg:  0.75,
			expectedMsg:   "Setting up My Complex App",
			expectedFound: true,
		},
		{
			name:          "Single character app name",
			dpkgOutput:    "Unpacking x (1.0) ...",
			appName:       "X",
			expectedProg:  0.72,
			expectedMsg:   "Unpacking X package",
			expectedFound: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			progress, message, found := parseDpkgProgress(testCase.dpkgOutput, testCase.appName)

			if found != testCase.expectedFound {
				t.Errorf("parseDpkgProgress() found = %v, expected %v", found, testCase.expectedFound)
			}

			if !testCase.expectedFound {
				return // Skip progress/message checks if we expect no match
			}

			if progress != testCase.expectedProg {
				t.Errorf("parseDpkgProgress() progress = %v, expected %v", progress, testCase.expectedProg)
			}

			if message != testCase.expectedMsg {
				t.Errorf("parseDpkgProgress() message = %q, expected %q", message, testCase.expectedMsg)
			}
		})
	}
}

func TestParseDpkgProgress_ProgressSequence(t *testing.T) {
	t.Parallel()

	// Test that progress values are in ascending order (realistic installation flow)
	dpkgSequence := []struct {
		output string
		phase  string
	}{
		{"Selecting previously unselected package test-app.", "select"},
		{"Reading database ... 285432 files", "reading"},
		{"Preparing to unpack /tmp/test-app.deb ...", "prepare"},
		{"Unpacking test-app (1.0.0) ...", "unpack"},
		{"Setting up test-app (1.0.0) ...", "setup"},
		{"update-alternatives: using /usr/bin/test-app", "alternatives"},
		{"Processing triggers for mailcap", "triggers"},
		{"Processing triggers for menu", "completion"},
	}

	var lastProgress float64

	appName := "TestApp"

	for stepIndex, step := range dpkgSequence {
		progress, _, found := parseDpkgProgress(step.output, appName)

		if !found {
			t.Errorf("Step %d (%s): Expected to find match but didn't", stepIndex, step.phase)

			continue
		}

		if progress <= lastProgress {
			t.Errorf("Step %d (%s): Progress %v should be > previous %v",
				stepIndex, step.phase, progress, lastProgress)
		}

		lastProgress = progress
	}

	// Final progress should be 1.0 (100%)
	if lastProgress != 1.0 {
		t.Errorf("Final progress should be 1.0, got %v", lastProgress)
	}
}

func TestParseDpkgProgress_RealWorldExamples(t *testing.T) {
	t.Parallel()

	// Real dpkg output captured from Chrome installation with LC_ALL=C
	realOutput := []string{
		"(Reading database ... 285432 files and directories currently installed.)",
		"Preparing to unpack /tmp/google-chrome-stable_current_amd64.deb ...",
		"Unpacking google-chrome-stable (122.0.6261.57-1) ...",
		"Setting up google-chrome-stable (122.0.6261.57-1) ...",
		"update-alternatives: using /usr/bin/google-chrome-stable to provide /usr/bin/x-www-browser (x-www-browser) in auto mode",
		"update-alternatives: using /usr/bin/google-chrome-stable to provide /usr/bin/gnome-www-browser (gnome-www-browser) in auto mode",
		"Processing triggers for mailcap (3.70+nmu1ubuntu1) ...",
		"Processing triggers for gnome-menus (3.36.0-1ubuntu3) ...",
		"Processing triggers for desktop-file-utils (0.26-1ubuntu3) ...",
		"Processing triggers for man-db (2.10.2-1) ...",
	}

	matchCount := 0

	for _, line := range realOutput {
		_, _, found := parseDpkgProgress(line, "Chrome")
		if found {
			matchCount++
		}
	}

	// We should match most of the important phases
	expectedMatches := 7 // reading, preparing, unpacking, setting up, alternatives, triggers
	if matchCount < expectedMatches {
		t.Errorf("Expected at least %d matches from real output, got %d", expectedMatches, matchCount)
	}
}

func TestParseDpkgProgress_EmptyAppName(t *testing.T) {
	t.Parallel()

	progress, message, found := parseDpkgProgress("Setting up test-package (1.0.0) ...", "")

	if !found {
		t.Error("Expected to find match even with empty app name")
	}

	if progress != 0.75 {
		t.Errorf("Expected progress 0.75, got %v", progress)
	}

	if message != "Setting up " {
		t.Errorf("Expected message 'Setting up ', got %q", message)
	}
}

func TestParseDpkgProgress_CaseSensitivity(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"Setting up google-chrome-stable (122.0.6261.57-1) ...",
		"SETTING UP google-chrome-stable (122.0.6261.57-1) ...", // Should not match - dpkg doesn't output uppercase
		"setting up google-chrome-stable (122.0.6261.57-1) ...", // Should not match - dpkg uses proper case
	}

	expectedResults := []bool{true, false, false}

	for i, testCase := range testCases {
		_, _, found := parseDpkgProgress(testCase, "Chrome")
		if found != expectedResults[i] {
			t.Errorf("Case %d: expected %v, got %v for input %q", i, expectedResults[i], found, testCase)
		}
	}
}

func TestParseDpkgUninstallProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		output        string
		appName       string
		wantProgress  float64
		wantMessage   string
		wantHasResult bool
	}{
		// Reading package lists
		{
			name:          "Reading package lists",
			output:        "Reading package lists...",
			appName:       "chrome",
			wantProgress:  0.25,
			wantMessage:   "Reading package lists",
			wantHasResult: true,
		},
		// Building dependency tree
		{
			name:          "Building dependency tree",
			output:        "Building dependency tree",
			appName:       "chrome",
			wantProgress:  0.35,
			wantMessage:   "Building dependency tree",
			wantHasResult: true,
		},
		// Reading state information
		{
			name:          "Reading state information",
			output:        "Reading state information...",
			appName:       "chrome",
			wantProgress:  0.45,
			wantMessage:   "Reading state information",
			wantHasResult: true,
		},
		// Preparing to remove
		{
			name:          "Preparing to remove",
			output:        "Preparing to remove google-chrome-stable",
			appName:       "chrome",
			wantProgress:  0.55,
			wantMessage:   "Preparing to remove chrome",
			wantHasResult: true,
		},
		// Removing package
		{
			name:          "Removing package",
			output:        "Removing google-chrome-stable",
			appName:       "chrome",
			wantProgress:  0.65,
			wantMessage:   "Removing chrome",
			wantHasResult: true,
		},
		// Processing triggers - man-db
		{
			name:          "Processing man-db triggers",
			output:        "Processing triggers for man-db (2.8.3-2ubuntu0.1)...",
			appName:       "chrome",
			wantProgress:  0.75,
			wantMessage:   "Processing manual page triggers",
			wantHasResult: true,
		},
		// Processing triggers - desktop-file-utils
		{
			name:          "Processing desktop-file-utils triggers",
			output:        "Processing triggers for desktop-file-utils (0.23-1ubuntu3.18.04.2)...",
			appName:       "chrome",
			wantProgress:  0.78,
			wantMessage:   "Processing desktop file triggers",
			wantHasResult: true,
		},
		// Processing triggers - gnome-menus
		{
			name:          "Processing gnome-menus triggers",
			output:        "Processing triggers for gnome-menus (3.13.3-11ubuntu1.1)...",
			appName:       "chrome",
			wantProgress:  0.80,
			wantMessage:   "Processing GNOME menu triggers",
			wantHasResult: true,
		},
		// Processing triggers - mailcap
		{
			name:          "Processing mailcap triggers",
			output:        "Processing triggers for mailcap (3.60ubuntu1)...",
			appName:       "chrome",
			wantProgress:  0.82,
			wantMessage:   "Processing MIME type triggers",
			wantHasResult: true,
		},
		// Generic processing triggers
		{
			name:          "Processing generic triggers",
			output:        "Processing triggers for shared-mime-info (1.9-2)...",
			appName:       "chrome",
			wantProgress:  0.75,
			wantMessage:   "Processing system triggers",
			wantHasResult: true,
		},
		// Purging configuration files
		{
			name:          "Purging configuration files",
			output:        "Purging configuration files for google-chrome-stable (103.0.5060.53-1)...",
			appName:       "chrome",
			wantProgress:  0.90,
			wantMessage:   "Purging configuration files",
			wantHasResult: true,
		},
		// dpkg warnings
		{
			name:          "dpkg warning about removing",
			output:        "dpkg: warning: while removing google-chrome-stable, directory '/opt/google/chrome' not empty so not removed",
			appName:       "chrome",
			wantProgress:  0.95,
			wantMessage:   "Checking dependencies",
			wantHasResult: true,
		},
		// Completion - removed
		{
			name:          "Package removed",
			output:        "google-chrome-stable removed",
			appName:       "chrome",
			wantProgress:  1.0,
			wantMessage:   "Uninstallation complete",
			wantHasResult: true,
		},
		// Empty app name handling
		{
			name:          "Removing with empty app name",
			output:        "Removing google-chrome-stable",
			appName:       "",
			wantProgress:  0.65,
			wantMessage:   "Removing package",
			wantHasResult: true,
		},
		// No match
		{
			name:          "No match",
			output:        "Some random output",
			appName:       "chrome",
			wantProgress:  0.0,
			wantMessage:   "",
			wantHasResult: false,
		},
		// Empty output
		{
			name:          "Empty output",
			output:        "",
			appName:       "chrome",
			wantProgress:  0.0,
			wantMessage:   "",
			wantHasResult: false,
		},
		// Whitespace only
		{
			name:          "Whitespace only",
			output:        "   \n\t  ",
			appName:       "chrome",
			wantProgress:  0.0,
			wantMessage:   "",
			wantHasResult: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			progress, message, hasResult := parseDpkgUninstallProgress(testCase.output, testCase.appName)

			if progress != testCase.wantProgress {
				t.Errorf("parseDpkgUninstallProgress() progress = %v, want %v", progress, testCase.wantProgress)
			}

			if message != testCase.wantMessage {
				t.Errorf("parseDpkgUninstallProgress() message = %v, want %v", message, testCase.wantMessage)
			}

			if hasResult != testCase.wantHasResult {
				t.Errorf("parseDpkgUninstallProgress() hasResult = %v, want %v", hasResult, testCase.wantHasResult)
			}
		})
	}
}

func TestProgressModel_UninstallStaging(t *testing.T) {
	// Setup isolated test environment
	tempDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempDir)

	// Create uninstall operation
	operations := []SelectedOperation{
		{AppKey: "chrome", Operation: StateUninstall, AppName: "Google Chrome"},
	}

	model := NewProgressWithOperations(context.Background(), styles.New(), operations)

	// Use dry-run mode for testing
	model.uninstaller = uninstall.NewUninstaller(false) // verbose=false for tests

	// Test uninstall stage message handling
	stageMsg := UninstallStageMsg{
		TaskIndex: 0,
		AppKey:    "chrome",
		AppName:   "Google Chrome",
		Stage:     3, // Should be 60% progress - "Removing package files"
	}

	updatedModel, cmd := model.handleUninstallStage(stageMsg)

	// Verify the model state was updated directly
	progressModel, ok := updatedModel.(*Progress)
	if !ok {
		t.Fatalf("Expected *Progress model, got %T", updatedModel)
	}

	// Check that the task progress was updated
	if len(progressModel.tasks) == 0 {
		t.Fatal("Expected at least one task")
	}

	task := progressModel.tasks[0]

	expectedProgress := 0.6
	if task.Progress != expectedProgress {
		t.Errorf("Expected progress %v, got %v", expectedProgress, task.Progress)
	}

	expectedStatus := "Removing package files"
	if task.Status != expectedStatus {
		t.Errorf("Expected status %q, got %q", expectedStatus, task.Status)
	}

	// Verify command is returned for next stage (since stage 3 < 5)
	if cmd == nil {
		t.Error("handleUninstallStage should return command for next stage progression")

		return
	}

	// Execute the command to get the next stage message
	msg := cmd()
	nextStageMsg, isUninstallStageMsg := msg.(UninstallStageMsg)

	if !isUninstallStageMsg {
		t.Errorf("Expected UninstallStageMsg for next stage, got %T", msg)

		return
	}

	// Verify next stage
	expectedNextStage := 4
	if nextStageMsg.Stage != expectedNextStage {
		t.Errorf("Expected next stage %d, got %d", expectedNextStage, nextStageMsg.Stage)
	}
}
