// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements secure password input UI.
package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/adapters/system"
	"github.com/janderssonse/karei/internal/tui/styles"
)

// Note: KeyCtrlC and other key constants are defined in menu.go

// PasswordPrompt represents a password input screen for sudo authentication.
//
//nolint:containedctx // TUI models require context for proper cancellation propagation
type PasswordPrompt struct {
	styles     *styles.Styles
	width      int
	height     int
	password   string
	operations []SelectedOperation
	message    string
	error      string
	cancelled  bool
	completed  bool
	showCursor bool
	ctx        context.Context // Parent context for cancellation/timeout propagation //nolint:containedctx
}

// PasswordPromptResult carries the result of password input.
type PasswordPromptResult struct {
	Password   string
	Operations []SelectedOperation
	Cancelled  bool
}

// NewPasswordPrompt creates a new password input screen.
func NewPasswordPrompt(ctx context.Context, styleConfig *styles.Styles, operations []SelectedOperation) *PasswordPrompt {
	// Initialize password prompt with operations
	prompt := &PasswordPrompt{
		styles:     styleConfig,
		operations: operations,
		ctx:        ctx, // Store parent context
	}

	appCount := len(operations)
	installCount := 0
	uninstallCount := 0

	for _, op := range operations {
		switch op.Operation {
		case StateInstall:
			installCount++
		case StateUninstall:
			uninstallCount++
		}
	}

	var message string

	switch {
	case installCount > 0 && uninstallCount > 0:
		message = fmt.Sprintf("Installing %d applications and uninstalling %d applications requires administrator privileges.", installCount, uninstallCount)

	case installCount > 0:
		message = fmt.Sprintf("Installing %d applications requires administrator privileges.", installCount)

	case uninstallCount > 0:
		message = fmt.Sprintf("Uninstalling %d applications requires administrator privileges.", uninstallCount)

	default:
		message = fmt.Sprintf("Processing %d applications requires administrator privileges.", appCount)
	}

	prompt.message = message
	prompt.showCursor = true

	return prompt
}

// Init initializes the password prompt.
func (m *PasswordPrompt) Init() tea.Cmd {
	return tea.Tick(500*1000000, func(_ time.Time) tea.Msg {
		return CursorBlinkMsg{}
	})
}

// CursorBlinkMsg represents a cursor blink event.
type CursorBlinkMsg struct{}

// PasswordValidationMsg represents a password validation request.
type PasswordValidationMsg struct {
	Password   string
	Operations []SelectedOperation
}

// PasswordValidationResult represents the result of password validation.
type PasswordValidationResult struct {
	Valid      bool
	Password   string
	Operations []SelectedOperation
	Error      string
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the PasswordPrompt model.
func (m *PasswordPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case CursorBlinkMsg:
		m.showCursor = !m.showCursor

		return m, tea.Tick(500*1000000, func(_ time.Time) tea.Msg {
			return CursorBlinkMsg{}
		})

	case tea.KeyMsg:
		return m.handleKeyInput(msg)

	case PasswordValidationMsg:
		return m.handlePasswordValidation(msg)

	case PasswordValidationResult:
		return m.handlePasswordValidationResult(msg)
	}

	return m, nil
}

// View renders the password prompt screen.
func (m *PasswordPrompt) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var builder strings.Builder

	// Header
	header := m.renderHeader()
	builder.WriteString(header)
	builder.WriteString("\n\n")

	// Main content
	content := m.renderContent()
	builder.WriteString(content)
	builder.WriteString("\n\n")

	// Footer
	footer := m.renderFooter()
	builder.WriteString(footer)

	return builder.String()
}

// handleKeyInput processes keyboard input for the password prompt.
//

func (m *PasswordPrompt) handleKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case KeyCtrlC, "esc":
		return m.handleCancelation()
	case "enter":
		return m.handlePasswordSubmission()
	case "backspace":
		return m.handleBackspace()
	default:
		return m.handleCharacterInput(msg)
	}
}

// handleCancelation handles cancel operations (Ctrl+C, Esc).
//

func (m *PasswordPrompt) handleCancelation() (tea.Model, tea.Cmd) {
	m.cancelled = true
	m.completed = true

	return m, func() tea.Msg {
		return PasswordPromptResult{
			Password:   "",
			Operations: m.operations,
			Cancelled:  true,
		}
	}
}

// handlePasswordSubmission handles Enter key for password submission.
//

func (m *PasswordPrompt) handlePasswordSubmission() (tea.Model, tea.Cmd) {
	if len(m.password) == 0 {
		m.error = "Password cannot be empty"

		return m, nil
	}

	// Validate password immediately with sudo true
	return m, func() tea.Msg {
		return PasswordValidationMsg{
			Password:   m.password,
			Operations: m.operations,
		}
	}
}

// handleBackspace handles backspace key for character deletion.
//

func (m *PasswordPrompt) handleBackspace() (tea.Model, tea.Cmd) {
	if len(m.password) > 0 {
		m.password = m.password[:len(m.password)-1]
		m.error = "" // Clear error when user starts typing
	}

	return m, nil
}

// handleCharacterInput handles regular character input.
//

func (m *PasswordPrompt) handleCharacterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Add character to password (only printable characters)
	if len(msg.Runes) == 1 {
		char := msg.Runes[0]
		if char >= 32 && char <= 126 { // Printable ASCII
			m.password += string(char)
			m.error = "" // Clear error when user starts typing
		}
	}

	return m, nil
}

// handlePasswordValidation validates the password using sudo true.
//

func (m *PasswordPrompt) handlePasswordValidation(msg PasswordValidationMsg) (tea.Model, tea.Cmd) {
	// Show validating message
	m.error = "Validating password..."

	return m, func() tea.Msg {
		// Test password with sudo true
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		defer cancel()

		// Use sudo -v to validate password - this always requires password verification
		// even if recent sudo cache exists, unlike 'sudo true'
		err := system.RunWithPassword(ctx, false, msg.Password, "-v")
		if err != nil {
			return PasswordValidationResult{
				Valid:      false,
				Password:   msg.Password,
				Operations: msg.Operations,
				Error:      "Invalid password. Please try again.",
			}
		}

		return PasswordValidationResult{
			Valid:      true,
			Password:   msg.Password,
			Operations: msg.Operations,
			Error:      "",
		}
	}
}

// handlePasswordValidationResult handles the result of password validation.
//

func (m *PasswordPrompt) handlePasswordValidationResult(msg PasswordValidationResult) (tea.Model, tea.Cmd) {
	if msg.Valid {
		// Password is valid - proceed with operations
		m.completed = true

		return m, func() tea.Msg {
			return PasswordPromptResult{
				Password:   msg.Password,
				Operations: msg.Operations,
				Cancelled:  false,
			}
		}
	}

	// Password is invalid - show error and let user try again
	m.error = msg.Error
	m.password = "" // Clear the invalid password

	return m, nil
}

// renderHeader creates the header with clean style matching other screens.
func (m *PasswordPrompt) renderHeader() string {
	// Left side: App name » Current location
	location := "Karei » Authentication"
	leftSide := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Primary).
		Render(location)

	// Right side: Status
	status := "🔒 Required"
	rightSide := lipgloss.NewStyle().
		Foreground(m.styles.Warning).
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

	// Style the header with subtle border
	return lipgloss.NewStyle().
		Padding(0, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width).
		Render(headerLine)
}

// renderContent creates the main content area with password input.
func (m *PasswordPrompt) renderContent() string {
	var builder strings.Builder

	// Explanation message
	messageStyle := lipgloss.NewStyle().
		Foreground(m.styles.Primary).
		MarginBottom(2)
	builder.WriteString(messageStyle.Render(m.message))
	builder.WriteString("\n\n")

	// Applications list
	builder.WriteString(m.styles.Title.Render("Applications to process:"))
	builder.WriteString("\n")

	for _, operation := range m.operations {
		var icon string

		switch operation.Operation {
		case StateInstall:
			icon = "✓"

		case StateUninstall:
			icon = "✗"

		default:
			icon = "○"
		}

		line := fmt.Sprintf("  %s %s", icon, operation.AppName)
		appStyle := lipgloss.NewStyle().Foreground(m.styles.Muted)
		builder.WriteString(appStyle.Render(line))
		builder.WriteString("\n")
	}

	builder.WriteString("\n")

	// Password input field
	builder.WriteString(m.styles.Title.Render("Password:"))
	builder.WriteString("\n")

	// Create password field with masked characters
	passwordDisplay := strings.Repeat("●", len(m.password))
	if m.showCursor {
		passwordDisplay += "│"
	} else {
		passwordDisplay += " "
	}

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Primary).
		Padding(0, 1).
		Width(30).
		Foreground(m.styles.Primary)

	builder.WriteString(inputStyle.Render(passwordDisplay))
	builder.WriteString("\n")

	// Error message
	if m.error != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(m.styles.Error).
			Bold(true).
			MarginTop(1)
		builder.WriteString(errorStyle.Render("❌ " + m.error))
		builder.WriteString("\n")
	}

	// Security notice
	securityNotice := "🛡️ Your password is used only for this session and is not stored."
	noticeStyle := lipgloss.NewStyle().
		Foreground(m.styles.Success).
		MarginTop(2)
	builder.WriteString(noticeStyle.Render(securityNotice))

	return m.styles.Card.Render(builder.String())
}

// renderFooter creates the footer with clean style matching other screens.
func (m *PasswordPrompt) renderFooter() string {
	actions := []FooterAction{
		{Key: "Enter", Action: "Confirm"},
		{Key: "Esc", Action: "Cancel"},
		{Key: "Backspace", Action: "Delete"},
	}

	// Use the shared RenderFooter function for consistency
	return RenderFooter(m.styles, m.width, actions, false) // No help button for password screen
}
