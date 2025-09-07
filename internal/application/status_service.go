// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package application

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/janderssonse/karei/internal/domain"
)

// SystemStatusData contains real system information for display.
// This is a simple DTO to avoid overengineering.
type SystemStatusData struct {
	InstalledApps int
	AvailableApps int
	CurrentTheme  string
	DiskUsageGB   float64
	DiskAvailGB   float64
	UptimeHours   float64
}

// StatusService provides system status information.
type StatusService struct {
	packageInstaller domain.PackageInstaller
	commandRunner    domain.CommandRunner
}

// NewStatusService creates a new status service.
func NewStatusService(installer domain.PackageInstaller, runner domain.CommandRunner) *StatusService {
	return &StatusService{
		packageInstaller: installer,
		commandRunner:    runner,
	}
}

// GetInstalledApps returns a list of installed applications.
func (s *StatusService) GetInstalledApps(_ context.Context) []string {
	// Check common installed apps from the system
	commonApps := []string{"git", "vim", "docker", "go", "rust", "node", "python"}

	var installed []string

	for _, name := range commonApps {
		// Check if app is actually installed
		if s.commandRunner.CommandExists(name) {
			installed = append(installed, name)
		}
	}

	return installed
}

// GetCurrentTheme returns the currently active theme.
func (s *StatusService) GetCurrentTheme(_ context.Context) string {
	// Read from config file or system settings
	// For now, return empty string until configuration is implemented
	return ""
}

// GetCurrentFont returns the currently active font.
func (s *StatusService) GetCurrentFont(_ context.Context) string {
	// Read from config file or system settings
	// For now, return empty string until configuration is implemented
	return ""
}

// GetSystemStatus fetches real system information.
// Pragmatic approach: use simple shell commands for now.
func (s *StatusService) GetSystemStatus(ctx context.Context) (*SystemStatusData, error) {
	status := &SystemStatusData{
		AvailableApps: 126, // From catalog.go - hardcoded for now
	}

	// Get installed packages count
	packages, err := s.packageInstaller.List(ctx)
	if err == nil {
		status.InstalledApps = len(packages)
	}

	// Get disk usage - simple df command
	if output, err := s.commandRunner.ExecuteWithOutput(ctx, "df", "-BG", "/"); err == nil {
		status.DiskUsageGB, status.DiskAvailGB = parseDiskUsage(output)
	}

	// Get uptime - simple uptime command
	if output, err := s.commandRunner.ExecuteWithOutput(ctx, "uptime", "-p"); err == nil {
		status.UptimeHours = parseUptime(ctx, output)
	}

	// Theme would come from config file - skip for now
	status.CurrentTheme = "System Default"

	return status, nil
}

// parseDiskUsage extracts disk usage from df output.
// Example: /dev/sda1  234G  45G  177G  21% /.
func parseDiskUsage(output string) (used, available float64) {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return 0, 0
	}

	// Parse second line (first is header)
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return 0, 0
	}

	// Fields: filesystem, size, used, available, use%, mount
	// Remove 'G' suffix and convert
	if usedStr := strings.TrimSuffix(fields[2], "G"); usedStr != "" {
		if val, err := strconv.ParseFloat(usedStr, 64); err == nil {
			used = val
		}
	}

	if availStr := strings.TrimSuffix(fields[3], "G"); availStr != "" {
		if val, err := strconv.ParseFloat(availStr, 64); err == nil {
			available = val
		}
	}

	return used, available
}

// parseUptime extracts hours from uptime output.
// Example: "up 2 days, 3 hours, 15 minutes".
func parseUptime(ctx context.Context, output string) float64 {
	// Try simple parsing first
	hours := parseUptimeString(output)
	if hours > 0 {
		return hours
	}

	// Fallback to /proc/uptime
	return getUptimeFromProc(ctx)
}

// parseUptimeString parses uptime command output.
func parseUptimeString(output string) float64 {
	var totalHours float64

	// Extract days
	totalHours += extractTimeUnit(output, "day") * 24

	// Extract hours
	totalHours += extractTimeUnit(output, "hour")

	return totalHours
}

// extractTimeUnit extracts a numeric value before a time unit.
func extractTimeUnit(output, unit string) float64 {
	if !strings.Contains(output, unit) {
		return 0
	}

	parts := strings.Split(output, unit)
	if len(parts) == 0 {
		return 0
	}

	words := strings.Fields(parts[0])
	if len(words) == 0 {
		return 0
	}

	value, err := strconv.Atoi(words[len(words)-1])
	if err != nil {
		return 0
	}

	return float64(value)
}

// getUptimeFromProc reads uptime from /proc/uptime.
func getUptimeFromProc(ctx context.Context) float64 {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "cat", "/proc/uptime").Output()
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return 0
	}

	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}

	return seconds / 3600
}

// FormatDiskSpace formats bytes to human readable string.
func FormatDiskSpace(gb float64) string {
	if gb < 1 {
		return fmt.Sprintf("%.0f MB", gb*1024)
	}

	return fmt.Sprintf("%.1f GB", gb)
}

// FormatUptime formats hours to human readable duration.
func FormatUptime(hours float64) time.Duration {
	return time.Duration(hours * float64(time.Hour))
}
