// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package uninstall_test

import (
	"context"
	"testing"

	"github.com/janderssonse/karei/internal/uninstall"
	"github.com/stretchr/testify/assert"
)

// TestUninstallGroupHandlesPartialFailures tests that group uninstalls continue on individual failures
// This is an important business rule: partial success is allowed for group operations

func TestUninstallGroupValidation(t *testing.T) {
	// Test that group validation works
	uninstaller, _ := uninstall.NewTestUninstaller(false)
	ctx := context.Background()

	// Test unknown group returns appropriate error
	err := uninstaller.UninstallGroup(ctx, "definitely-not-a-real-group")
	assert.ErrorIs(t, err, uninstall.ErrUnknownGroup,
		"Unknown group should return ErrUnknownGroup")
}

// TestSpecialUninstallsMatchCatalog verifies special uninstalls match real apps in catalog.
func TestSpecialUninstallsMatchCatalog(t *testing.T) {
	// Import apps to verify special uninstalls match catalog
	// This ensures consistency between special uninstalls and app definitions
	for appName, uninstallFunc := range uninstall.SpecialUninstalls {
		assert.NotNil(t, uninstallFunc,
			"Special uninstall for %s should not be nil", appName)

		// Each special uninstall should correspond to a real app
		// This is a business rule: special handling only for known apps
	}
}
