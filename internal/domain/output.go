// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain

import "time"

// OutputPort defines the interface for presenting command results.
// This is a domain port that adapters implement for different output formats.
type OutputPort interface {
	// Success outputs a success message with optional structured data
	Success(message string, data interface{}) error

	// Error outputs an error message
	Error(message string) error

	// Info outputs an informational message
	Info(message string) error

	// Progress outputs progress information for long-running operations
	Progress(message string) error

	// Table outputs tabular data
	Table(headers []string, rows [][]string) error

	// IsQuiet returns true if output should be suppressed
	IsQuiet() bool
}

// InstallResult represents the outcome of an installation operation.
type InstallResult struct {
	Installed []string      `json:"installed"`
	Failed    []string      `json:"failed,omitempty"`
	Skipped   []string      `json:"skipped,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// UninstallResult represents the outcome of an uninstallation operation.
type UninstallResult struct {
	Uninstalled []string      `json:"uninstalled"`
	Failed      []string      `json:"failed,omitempty"`
	NotFound    []string      `json:"not_found,omitempty"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
}

// ListResult represents installed packages and their metadata.
type ListResult struct {
	Packages  []PackageInfo `json:"packages"`
	Total     int           `json:"total"`
	Timestamp time.Time     `json:"timestamp"`
}

// PackageInfo contains information about an installed package.
type PackageInfo struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Type        string    `json:"type"` // "app", "font", "theme", "tool"
	Installed   time.Time `json:"installed"`
	Size        int64     `json:"size,omitempty"`
	Description string    `json:"description,omitempty"`
}

// StatusResult represents system status information.
type StatusResult struct {
	Version      string            `json:"version"`
	Platform     string            `json:"platform"`
	Architecture string            `json:"architecture"`
	Installed    int               `json:"installed_packages"`
	Theme        string            `json:"current_theme"`
	Font         string            `json:"current_font"`
	Environment  map[string]string `json:"environment,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// VerifyResult represents system verification results.
type VerifyResult struct {
	Valid     bool          `json:"valid"`
	Checks    []VerifyCheck `json:"checks"`
	Errors    []string      `json:"errors,omitempty"`
	Warnings  []string      `json:"warnings,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// VerifyCheck represents a single verification check.
type VerifyCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warning", "skip"
	Message string `json:"message,omitempty"`
}
