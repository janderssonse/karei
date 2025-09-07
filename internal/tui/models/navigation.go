// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models defines shared navigation messages between UI screens.
package models

// NavigateMsg is a message sent to request navigation to a specific screen.
type NavigateMsg struct {
	Screen int
	Data   any // Optional data to pass to the new screen
}

// Screen constants for navigation.
const (
	MenuScreen = iota
	AppsScreen
	ThemeScreen
	ConfigScreen
	StatusScreen
	HelpScreen
	ProgressScreen
	PasswordScreen
)

// Operation constants.
const (
	OperationInstall   = "install"
	OperationUninstall = "uninstall"
)

// Common message constants.
const (
	RefreshStatusData      = "refresh_status"
	TestPassword           = "test_password"
	InstallationFailed     = "installation failed"
	CheckRecentActivityMsg = "Check recent activity for details"
)

// CompletedOperationsMsg contains operations that were just completed.
type CompletedOperationsMsg struct {
	Operations []SelectedOperation
}

// SmoothScrollMsg triggers smooth viewport scrolling after navigation.
type SmoothScrollMsg struct{}

// SearchActivatedMsg indicates search has been activated.
type SearchActivatedMsg struct {
	Active bool
}

// SearchDeactivatedMsg indicates search has been deactivated.
type SearchDeactivatedMsg struct {
	PreserveQuery bool
	Query         string
}

// SearchUpdateMsg carries search query updates.
type SearchUpdateMsg struct {
	Query  string
	Active bool
}

// ContextSwitchMsg indicates a context switch occurred.
type ContextSwitchMsg struct {
	Direction string // "up" or "down"
	Context   string // "search" or "categories"
}
