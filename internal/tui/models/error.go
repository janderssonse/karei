// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements error display and recovery UI.
package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// ErrorType represents different types of errors.
type ErrorType int

const (
	// ErrorGeneral represents a general error type.
	ErrorGeneral ErrorType = iota
	// ErrorNetwork represents a network-related error.
	ErrorNetwork
	// ErrorPermission represents a permission-related error.
	ErrorPermission
	// ErrorDiskSpace represents a disk space error.
	ErrorDiskSpace
	// ErrorDependency represents a dependency error.
	ErrorDependency
	// ErrorConfiguration represents a configuration error.
	ErrorConfiguration
	// ErrorInstallation represents an installation error.
	ErrorInstallation
)

// ErrorDetails contains error information.
type ErrorDetails struct {
	Type        ErrorType
	Title       string
	Message     string
	Details     string
	Suggestions []string
	Timestamp   time.Time
	Recoverable bool
	Recovery    func() tea.Cmd
}

// ErrorScreen represents the error display screen model.
type ErrorScreen struct {
	styles   *styles.Styles
	width    int
	height   int
	error    ErrorDetails
	quitting bool
	keyMap   ErrorKeyMap
}

// ErrorKeyMap defines key bindings for the error screen.
type ErrorKeyMap struct {
	Retry key.Binding
	Back  key.Binding
	Help  key.Binding
	Quit  key.Binding
}

// DefaultErrorKeyMap returns the default key bindings.
func DefaultErrorKeyMap() ErrorKeyMap {
	return ErrorKeyMap{
		Retry: key.NewBinding(
			key.WithKeys("r", "enter"),
			key.WithHelp("r/enter", "retry operation"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "go back"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "show help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// NewErrorScreen creates a new error screen model.
func NewErrorScreen(s *styles.Styles, err ErrorDetails) *ErrorScreen {
	return &ErrorScreen{
		styles: s,
		error:  err,
		keyMap: DefaultErrorKeyMap(),
	}
}

// CreateNetworkError creates a network-related error.
func CreateNetworkError(message string, details string) ErrorDetails {
	return ErrorDetails{
		Type:      ErrorNetwork,
		Title:     "Network Connection Error",
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Suggestions: []string{
			"Check your internet connection",
			"Verify proxy settings if applicable",
			"Try again after network issues are resolved",
			"Use 'karei status' to check system health",
		},
		Recoverable: true,
		Recovery: func() tea.Cmd {
			return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return RetryMsg{}
			})
		},
	}
}

// CreatePermissionError creates a permission-related error.
func CreatePermissionError(message string, details string) ErrorDetails {
	return ErrorDetails{
		Type:      ErrorPermission,
		Title:     "Permission Denied",
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Suggestions: []string{
			"Ensure your user account has sudo privileges",
			"Run 'sudo -v' to refresh sudo session",
			"Check that user is in 'sudo' group",
			"Contact system administrator if needed",
		},
		Recoverable: true,
		Recovery: func() tea.Cmd {
			return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return RetryMsg{}
			})
		},
	}
}

// CreateInstallationError creates an installation-related error.
func CreateInstallationError(appName string, message string, details string) ErrorDetails {
	return ErrorDetails{
		Type:      ErrorInstallation,
		Title:     "Installation Failed: " + appName,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Suggestions: []string{
			"Check available disk space",
			"Verify internet connection",
			"Update package lists: 'sudo apt update'",
			"Try installing manually: 'karei install " + appName + "'",
			"Check installation logs: 'karei logs install'",
		},
		Recoverable: true,
		Recovery: func() tea.Cmd {
			return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return RetryMsg{}
			})
		},
	}
}

// CreateUninstallationError creates an uninstallation-related error.
func CreateUninstallationError(appName string, message string, details string) ErrorDetails {
	suggestions := getUninstallationSuggestions(appName)

	return ErrorDetails{
		Type:        ErrorInstallation, // Same type as installation errors
		Title:       "Uninstallation Failed: " + appName,
		Message:     message,
		Details:     details,
		Timestamp:   time.Now(),
		Suggestions: suggestions,
		Recoverable: true,
		Recovery: func() tea.Cmd {
			return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return RetryMsg{}
			})
		},
	}
}

// getUninstallationSuggestions provides method-specific troubleshooting suggestions for uninstallation failures.
func getUninstallationSuggestions(appName string) []string {
	// Look up the app in the catalog to determine the installation method
	app, exists := apps.Apps[appName]
	if !exists {
		// If app not found in catalog, provide general suggestions
		return []string{
			"Check if the application is currently running and close it",
			"Verify administrator permissions",
			"Try uninstalling manually based on how it was installed",
			"Check uninstallation logs: 'karei logs uninstall'",
		}
	}

	// Provide method-specific suggestions
	switch app.Method {
	case domain.MethodAPT:
		return []string{
			"Check if the application is currently running and close it",
			"Verify administrator permissions",
			"Update package lists: 'sudo apt update'",
			"Try uninstalling manually: 'sudo apt remove " + appName + "'",
			"Check uninstallation logs: 'karei logs uninstall'",
		}
	case domain.MethodSnap:
		return []string{
			"Check if the application is currently running and close it",
			"Verify snap daemon is running: 'systemctl status snapd'",
			"Try uninstalling manually: 'sudo snap remove " + appName + "'",
			"List installed snaps: 'snap list'",
			"Check uninstallation logs: 'karei logs uninstall'",
		}
	case domain.MethodFlatpak:
		return []string{
			"Check if the application is currently running and close it",
			"Verify flatpak is available: 'flatpak --version'",
			"Try uninstalling manually: 'flatpak uninstall " + app.Source + "'",
			"List installed flatpaks: 'flatpak list --app'",
			"Check uninstallation logs: 'karei logs uninstall'",
		}
	case domain.MethodDEB:
		return []string{
			"Check if the application is currently running and close it",
			"Verify administrator permissions",
			"Try uninstalling manually: 'sudo apt remove " + appName + "'",
			"Check installed packages: 'dpkg -l | grep " + appName + "'",
			"Check uninstallation logs: 'karei logs uninstall'",
		}
	case domain.MethodMise:
		return []string{
			"Check if the application is currently running and close it",
			"Verify mise is available: 'mise --version'",
			"Try uninstalling manually: 'mise uninstall " + appName + "'",
			"Check mise installations: 'mise list'",
			"Check uninstallation logs: 'karei logs uninstall'",
		}
	case domain.MethodAqua:
		return []string{
			"Check if the application is currently running and close it",
			"Verify aqua is available: 'aqua --version'",
			"Remove from aqua config: edit ~/.config/aqua/aqua.yaml",
			"Check aqua installations: 'aqua list'",
			"Check uninstallation logs: 'karei logs uninstall'",
		}
	case domain.MethodGitHub, domain.MethodScript:
		return []string{
			"Check if the application is currently running and close it",
			"Remove binary from PATH: 'rm ~/.local/bin/" + appName + "'",
			"Check for additional installation files in ~/.local/",
			"Remove desktop entries if any: 'rm ~/.local/share/applications/" + appName + ".desktop'",
			"Check uninstallation logs: 'karei logs uninstall'",
		}
	default:
		return []string{
			"Check if the application is currently running and close it",
			"Verify administrator permissions",
			"Try uninstalling manually based on how it was installed",
			"Check uninstallation logs: 'karei logs uninstall'",
		}
	}
}

// CreateConfigurationError creates a configuration-related error.
func CreateConfigurationError(message string, details string) ErrorDetails {
	return ErrorDetails{
		Type:      ErrorConfiguration,
		Title:     "Configuration Error",
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Suggestions: []string{
			"Check configuration file permissions",
			"Verify configuration syntax",
			"Reset to defaults if needed",
			"Check logs for detailed error information",
		},
		Recoverable: true,
		Recovery: func() tea.Cmd {
			return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
				return RetryMsg{}
			})
		},
	}
}

// RetryMsg is sent when the user wants to retry an operation.
type RetryMsg struct{}

// Init initializes the error screen model.
func (m *ErrorScreen) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the ErrorScreen model.
func (m *ErrorScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			m.quitting = true

			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Back):
			// Navigate back to apps screen
			return m, func() tea.Msg {
				return NavigateMsg{Screen: AppsScreen}
			}

		case key.Matches(msg, m.keyMap.Retry):
			// Attempt to retry the operation if recoverable
			if m.error.Recoverable && m.error.Recovery != nil {
				return m, m.error.Recovery()
			}

		case key.Matches(msg, m.keyMap.Help):
			// Show context-sensitive help (navigate to help screen with error context)
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case RetryMsg:
		// Handle retry attempt
		// Implement actual retry logic (re-execute failed operation)
		return m, nil
	}

	return m, nil
}

// View renders the error screen.
func (m *ErrorScreen) View() string {
	if m.quitting {
		return GoodbyeMessage
	}

	var builder strings.Builder

	// Header with error type
	header := m.renderHeader()
	builder.WriteString(header)
	builder.WriteString("\n\n")

	// Main error display
	errorDisplay := m.renderErrorDisplay()
	builder.WriteString(errorDisplay)
	builder.WriteString("\n\n")

	// Footer with actions
	footer := m.renderFooter()
	builder.WriteString(footer)

	return builder.String()
}

// renderHeader creates the header with clean style matching other screens.
func (m *ErrorScreen) renderHeader() string {
	// Left side: App name » Current location
	location := "Karei » Error"
	leftSide := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Error). // Use error color for error screen
		Render(location)

	// Right side: Error type with icon
	var icon string

	switch m.error.Type {
	case ErrorNetwork:
		icon = "🌐"
	case ErrorPermission:
		icon = "🔒"
	case ErrorInstallation:
		icon = "📦"
	case ErrorConfiguration:
		icon = "⚙️"
	default:
		icon = "❌"
	}

	status := fmt.Sprintf("%s %s", icon, m.error.GetErrorType())
	rightSide := lipgloss.NewStyle().
		Foreground(m.styles.Muted).
		Render(status)

	// Calculate spacing
	totalWidth := m.width
	leftWidth := lipgloss.Width(leftSide)
	rightWidth := lipgloss.Width(rightSide)
	spacerWidth := totalWidth - leftWidth - rightWidth - 4 // Account for padding

	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := strings.Repeat(" ", spacerWidth)

	// Combine with spacing
	headerLine := leftSide + spacer + rightSide

	// Style the header with subtle border (use error color for border)
	return lipgloss.NewStyle().
		Padding(0, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(m.styles.Error). // Error color for border
		Width(m.width).
		Render(headerLine)
}

// renderErrorDisplay creates the main error information display.
func (m *ErrorScreen) renderErrorDisplay() string {
	var builder strings.Builder

	// Main content container using full width
	containerStyle := m.styles.Card.
		Width(m.width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Error)

	var content strings.Builder

	// Error message
	content.WriteString(m.styles.Title.Render("Error Details"))
	content.WriteString("\n\n")

	messageStyle := lipgloss.NewStyle().
		Foreground(m.styles.Error).
		Bold(true)
	content.WriteString(messageStyle.Render(m.error.Message))
	content.WriteString("\n\n")

	// Detailed information
	if m.error.Details != "" {
		content.WriteString(m.styles.Title.Render("Additional Information"))
		content.WriteString("\n")

		detailsStyle := lipgloss.NewStyle().
			Foreground(m.styles.Muted).
			MarginLeft(2)
		content.WriteString(detailsStyle.Render(m.error.Details))
		content.WriteString("\n\n")
	}

	// Suggestions
	if len(m.error.Suggestions) > 0 {
		content.WriteString(m.styles.Title.Render("💡 Suggested Solutions"))
		content.WriteString("\n")

		for i, suggestion := range m.error.Suggestions {
			suggestionStyle := lipgloss.NewStyle().
				Foreground(m.styles.Success).
				MarginLeft(2)
			line := fmt.Sprintf("%d. %s", i+1, suggestion)
			content.WriteString(suggestionStyle.Render(line))
			content.WriteString("\n")
		}

		content.WriteString("\n")
	}

	// Recovery status
	if m.error.Recoverable {
		recoveryStyle := lipgloss.NewStyle().
			Foreground(m.styles.Success).
			Bold(true)
		content.WriteString(recoveryStyle.Render("🔄 This error is recoverable. Press 'r' or Enter to retry."))
	} else {
		finalStyle := lipgloss.NewStyle().
			Foreground(m.styles.Warning).
			Bold(true)
		content.WriteString(finalStyle.Render("⚠️ This error requires manual intervention."))
	}

	builder.WriteString(containerStyle.Render(content.String()))

	return builder.String()
}

// renderFooter creates the footer with clean style matching other screens.
func (m *ErrorScreen) renderFooter() string {
	var actions []FooterAction

	if m.error.Recoverable {
		actions = append(actions, FooterAction{Key: "Enter", Action: "Retry"})
	}

	actions = append(actions,
		FooterAction{Key: "Esc", Action: "Back"},
		FooterAction{Key: "q", Action: "Quit"},
	)

	// Use the shared RenderFooter function for consistency, include help
	return RenderFooter(m.styles, m.width, actions, true)
}

// GetErrorType returns a human-readable error type string.
func (e ErrorDetails) GetErrorType() string {
	switch e.Type {
	case ErrorNetwork:
		return "Network Error"
	case ErrorPermission:
		return "Permission Error"
	case ErrorInstallation:
		return "Installation Error"
	case ErrorConfiguration:
		return "Configuration Error"
	case ErrorDiskSpace:
		return "Disk Space Error"
	case ErrorDependency:
		return "Dependency Error"
	default:
		return "General Error"
	}
}
