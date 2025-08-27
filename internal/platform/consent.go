// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Constants for consent responses.
const (
	ConsentYes = "yes"
	ConsentY   = "y"
)

// AutoYes is set by the --yes flag to auto-accept prompts.
var AutoYes bool //nolint:gochecknoglobals // CLI flag state needs to be global

// AskConsent prompts the user for consent before modifying configs.
// Returns true if user agrees, false otherwise.
func AskConsent(appName string, configPath string) bool {
	// If --yes flag is set, auto-accept
	if AutoYes {
		fmt.Printf("Auto-accepting: Modifying %s configuration at %s\n", appName, configPath)
		return true
	}

	// If not a TTY (piped/scripted), skip config modifications
	if !DefaultOutput.IsTTY(os.Stdin.Fd()) {
		return false
	}

	fmt.Printf("\nKarei needs to modify %s configuration:\n", appName)
	fmt.Printf("  File: %s\n", configPath)
	fmt.Print("Continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))

	return response == ConsentY || response == ConsentYes
}

// AskProcessRestart prompts before killing a process.
func AskProcessRestart(processName string) bool {
	// If --yes flag is set, auto-accept
	if AutoYes {
		fmt.Printf("Auto-accepting: Restarting %s\n", processName)
		return true
	}

	// If not a TTY, don't kill processes
	if !DefaultOutput.IsTTY(os.Stdin.Fd()) {
		return false
	}

	fmt.Printf("\n%s needs to be restarted for changes to take effect.\n", processName)
	fmt.Printf("Close %s now? [y/N]: ", processName)

	reader := bufio.NewReader(os.Stdin)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))

	return response == ConsentY || response == ConsentYes
}

// AddConfigMarker adds a dated comment to track modifications.
// For JSON files, adds it as a neighboring field comment.
func AddConfigMarker(content string, format string) string {
	marker := "Modified by Karei on " + GetTimestamp()

	switch format {
	case "json":
		// For JSON, add as inline comment (non-standard but commonly accepted)
		// Many tools ignore comments in JSON
		return content // JSON doesn't support comments, return as-is
	default:
		// For shell/conf files, add as comment
		return fmt.Sprintf("%s # %s", content, marker)
	}
}

// GetTimestamp returns current date in YYYY-MM-DD format.
func GetTimestamp() string {
	return time.Now().Format("2006-01-02")
}

// PromptConsentWithReader is a testable version of prompt consent.
// It accepts custom reader and writer for testing.
func PromptConsentWithReader(prompt string, autoYes bool, reader io.Reader, writer io.Writer) (bool, error) {
	// If auto-yes is set, immediately return true
	if autoYes {
		_, _ = fmt.Fprintf(writer, "Auto-accepting: %s\n", prompt)
		return true, nil
	}

	// Show prompt
	_, _ = fmt.Fprintf(writer, "%s [y/N]: ", prompt)

	// Read response
	bufReader := bufio.NewReader(reader)

	response, err := bufReader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))

	return response == ConsentY || response == ConsentYes, nil
}
