// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package console

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"

	"golang.org/x/term"
)

// OutputState holds global output configuration.
type OutputState struct {
	Verbose bool
	JSON    bool
	Plain   bool
}

// DefaultOutput provides output formatting utilities.
var DefaultOutput = &OutputState{} //nolint:gochecknoglobals

// SetMode configures output mode.
func (o *OutputState) SetMode(verbose, json, plain bool) {
	o.Verbose = verbose
	o.JSON = json
	o.Plain = plain
}

// IsTTY checks if output is going to a terminal (not piped/redirected).
func (o *OutputState) IsTTY(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

// Bold formats text with bold when in TTY, uppercase when piped.
func (o *OutputState) Bold(text string) string {
	if o.JSON || o.Plain {
		return text // No formatting in JSON or plain mode
	}

	// Check no-color.org standards
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return text // Disable color per no-color.org
	}

	// Check if stdout is a TTY
	if o.IsTTY(os.Stdout.Fd()) {
		return "\033[1m" + text + "\033[0m" // ANSI bold
	}

	// Fallback for pipes/redirects - use uppercase
	return strings.ToUpper(text)
}

// Header formats section headers consistently.
func (o *OutputState) Header(text string) string {
	return o.Bold(text)
}

// Progressf writes progress messages to stderr (only if verbose and not JSON/Plain).
func (o *OutputState) Progressf(format string, args ...any) {
	if o.Verbose && !o.JSON && !o.Plain {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// Successf writes success messages to stderr (only if not JSON/Plain).
func (o *OutputState) Successf(format string, args ...any) {
	if !o.JSON && !o.Plain {
		fmt.Fprintf(os.Stderr, "✓ "+format+"\n", args...)
	}
}

// Warningf writes warning messages to stderr (always visible unless plain mode).
func (o *OutputState) Warningf(format string, args ...any) {
	if o.Plain {
		// In plain mode, warnings go to stderr without symbols
		fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
	} else {
		fmt.Fprintf(os.Stderr, "⚠ "+format+"\n", args...)
	}
}

// Errorf writes error messages to stderr (always visible).
func (o *OutputState) Errorf(format string, args ...any) {
	if o.Plain {
		// In plain mode, errors go to stderr without symbols
		fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	} else {
		fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
	}
}

// Result writes command results to stdout (machine-readable primary output).
func (o *OutputState) Result(data any) {
	_, _ = fmt.Fprintf(os.Stdout, "%v\n", data)
}

// JSONResult writes structured JSON results to stdout.
func (o *OutputState) JSONResult(status string, data map[string]any) {
	result := map[string]any{
		"status": status,
	}
	maps.Copy(result, data)

	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		// Best effort - output encoding errors shouldn't crash the program
		fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
	}
}

// SuccessResult outputs success result to stdout with optional stderr message.
func (o *OutputState) SuccessResult(result any, message string) {
	if !o.JSON && !o.Plain && message != "" {
		o.Successf("%s", message)
	}

	if o.JSON {
		o.JSONResult("success", map[string]any{"result": result})
	} else {
		o.Result(result)
	}
}

// ErrorResult outputs error result to stdout (for commands that need to pipe errors).
func (o *OutputState) ErrorResult(err error, code int) {
	if o.JSON {
		o.JSONResult("error", map[string]any{
			"error": err.Error(),
			"code":  code,
		})
	}
	// Error message always goes to stderr regardless
	o.Errorf("%s", err.Error())
}

// PlainKeyValue outputs key:value pairs for machine parsing.
func (o *OutputState) PlainKeyValue(key, value string) {
	_, _ = fmt.Fprintf(os.Stdout, "%s:%s\n", key, value)
}

// PlainStatus outputs status information in key:status format.
func (o *OutputState) PlainStatus(name, status string) {
	_, _ = fmt.Fprintf(os.Stdout, "%s:%s\n", name, status)
}

// PlainList outputs a simple list of items, one per line.
func (o *OutputState) PlainList(items []string) {
	for _, item := range items {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", item)
	}
}

// PlainValue outputs a single value.
func (o *OutputState) PlainValue(value string) {
	_, _ = fmt.Fprintf(os.Stdout, "%s\n", value)
}
