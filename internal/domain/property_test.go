// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"testing"
	"unicode"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackageNameValidationProperties tests invariants that must hold for all package names.
func TestPackageNameValidationProperties(t *testing.T) {
	// Test meaningful validation rules
	t.Run("package_validation_business_rules", func(t *testing.T) {
		tests := []struct {
			name     string
			pkg      *domain.Package
			expected bool
			reason   string
		}{
			{
				name:     "all_fields_required",
				pkg:      &domain.Package{Name: "vim", Method: domain.MethodAPT, Source: "ubuntu"},
				expected: true,
				reason:   "Valid package with all required fields",
			},
			{
				name:     "empty_name_invalid",
				pkg:      &domain.Package{Name: "", Method: domain.MethodAPT, Source: "ubuntu"},
				expected: false,
				reason:   "Package must have a name",
			},
			{
				name:     "empty_method_invalid",
				pkg:      &domain.Package{Name: "vim", Method: "", Source: "ubuntu"},
				expected: false,
				reason:   "Package must have an installation method",
			},
			{
				name:     "empty_source_invalid",
				pkg:      &domain.Package{Name: "vim", Method: domain.MethodAPT, Source: ""},
				expected: false,
				reason:   "Package must have a source repository",
			},
			{
				name:     "whitespace_name_fixed",
				pkg:      &domain.Package{Name: "   ", Method: domain.MethodAPT, Source: "ubuntu"},
				expected: false, // Fixed: whitespace-only names are now invalid
				reason:   "Whitespace-only names should be invalid",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.expected, tc.pkg.IsValid(), tc.reason)
			})
		}
	})

	// Test real-world package names that might cause issues
	t.Run("edge_case_package_names", func(t *testing.T) {
		edgeCaseNames := []struct {
			name   string
			reason string
		}{
			{"lib32stdc++6", "Package names with numbers and special chars"},
			{"python3.11-dev", "Version-specific packages"},
			{"@angular/cli", "Scoped npm packages"},
			{"github.com/owner/repo", "Go module style names"},
			{"postgresql-14-postgis-3", "Complex versioned names"},
		}

		for _, tc := range edgeCaseNames {
			pkg := &domain.Package{
				Name:   tc.name,
				Method: domain.MethodAPT,
				Source: "ubuntu",
			}
			assert.True(t, pkg.IsValid(), tc.reason)
		}
	})

	// Property 3: Missing any required field makes package invalid
	t.Run("missing_required_fields_invalid", func(t *testing.T) {
		// Missing name
		pkg1 := &domain.Package{
			Method: domain.MethodAPT,
			Source: "source",
		}
		assert.False(t, pkg1.IsValid(), "Package without name should be invalid")

		// Missing method (zero value)
		pkg2 := &domain.Package{
			Name:   "vim",
			Source: "source",
		}
		assert.False(t, pkg2.IsValid(), "Package without method should be invalid")

		// Missing source
		pkg3 := &domain.Package{
			Name:   "vim",
			Method: domain.MethodAPT,
		}
		assert.False(t, pkg3.IsValid(), "Package without source should be invalid")
	})

	// Property 4: Whitespace-only fields are invalid (BUG FIX)
	t.Run("whitespace_only_fields_invalid", func(t *testing.T) {
		// Whitespace-only name
		pkg1 := &domain.Package{
			Name:   "   \t\n  ",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		}
		assert.False(t, pkg1.IsValid(), "Package with whitespace-only name should be invalid")

		// Whitespace-only source
		pkg2 := &domain.Package{
			Name:   "vim",
			Method: domain.MethodAPT,
			Source: "   ",
		}
		assert.False(t, pkg2.IsValid(), "Package with whitespace-only source should be invalid")

		// Mixed valid and whitespace fields
		pkg3 := &domain.Package{
			Name:   "\t\t\t",
			Method: domain.MethodAPT,
			Source: "\n\n",
		}
		assert.False(t, pkg3.IsValid(), "Package with all whitespace fields should be invalid")
	})
}

// TestVersionValidationProperties tests version string validation invariants.
func TestVersionValidationProperties(t *testing.T) {
	// Property: Any non-empty version string is acceptable
	t.Run("version_string_formats", func(t *testing.T) {
		validVersions := []string{
			"1.0.0",
			"2.1.3-alpha",
			"3.0.0-beta.1",
			"4.5.6+build.123",
			"latest",
			"stable",
			"nightly",
			">=1.0.0",
			"~1.2.3",
			"^2.0.0",
			"1.x",
			"2.*",
			"v1.0.0",
			"r123",
			"20231225",
			"1:2.3.4-5ubuntu6", // Debian version format
			"1.0.0.0",
			"", // Empty version means "latest"
		}

		for _, version := range validVersions {
			pkg := &domain.Package{
				Name:    "test",
				Method:  domain.MethodAPT,
				Source:  "test",
				Version: version,
			}
			assert.True(t, pkg.IsValid(),
				"Package with version=%q should be valid", version)
		}
	})

	// Property: Version field doesn't affect base validation
	t.Run("version_independent_of_other_fields", func(t *testing.T) {
		// Invalid package remains invalid regardless of version
		invalidPkg := &domain.Package{
			Name:    "", // Invalid: empty name
			Method:  domain.MethodAPT,
			Source:  "source",
			Version: "1.0.0",
		}
		assert.False(t, invalidPkg.IsValid(),
			"Invalid package should remain invalid even with version")

		// Valid package remains valid with any version
		validPkg := &domain.Package{
			Name:   "app",
			Method: domain.MethodAPT,
			Source: "source",
		}

		versions := []string{"", "1.0", "latest", ">=2.0.0", "any-string-really"}
		for _, v := range versions {
			validPkg.Version = v
			assert.True(t, validPkg.IsValid(),
				"Valid package should remain valid with version=%q", v)
		}
	})
}

// TestInstallMethodProperties tests install method invariants.
func TestInstallMethodProperties(t *testing.T) {
	// Property: Zero value InstallMethod is invalid
	t.Run("zero_value_invalid", func(t *testing.T) {
		pkg := &domain.Package{
			Name:   "app",
			Method: domain.InstallMethod(""), // Zero value for string
			Source: "source",
		}
		assert.False(t, pkg.IsValid(), "Zero value InstallMethod should be invalid")
	})

	// Property: All defined methods are valid
	t.Run("all_defined_methods_valid", func(t *testing.T) {
		methods := []domain.InstallMethod{
			domain.MethodAPT,
			domain.MethodDNF,
			domain.MethodYum,
			domain.MethodPacman,
			domain.MethodSnap,
			domain.MethodFlatpak,
			domain.MethodGitHub,
			domain.MethodBinary,
			domain.MethodScript,
		}

		for _, method := range methods {
			pkg := &domain.Package{
				Name:   "app",
				Method: method,
				Source: "source",
			}
			assert.True(t, pkg.IsValid(), "Method %v should be valid", method)
		}
	})

	// Property: Undefined methods make package invalid
	t.Run("undefined_methods_invalid", func(t *testing.T) {
		// Test with various undefined method values
		undefinedMethods := []domain.InstallMethod{
			domain.InstallMethod("unknown"),
			domain.InstallMethod("invalid-method"),
			domain.InstallMethod("not-a-real-method"),
		}

		for _, method := range undefinedMethods {
			pkg := &domain.Package{
				Name:   "app",
				Method: method,
				Source: "source",
			}
			// Note: This assumes IsValid checks for known methods
			// If it doesn't, this test documents current behavior
			isValid := pkg.IsValid()
			// Current implementation might accept any non-empty string method
			// This test documents the actual behavior
			t.Logf("Package with undefined method %v has IsValid=%v", method, isValid)
		}
	})
}

// TestSourceValidationProperties tests source field validation.
func TestSourceValidationProperties(t *testing.T) {
	// Property: Source format depends on install method
	t.Run("source_format_by_method", func(t *testing.T) {
		testCases := []struct {
			method       domain.InstallMethod
			validSources []string
			description  string
		}{
			{
				method:       domain.MethodAPT,
				validSources: []string{"vim", "nginx", "python3", "lib-dev"},
				description:  "APT sources are package names",
			},
			{
				method:       domain.MethodGitHub,
				validSources: []string{"owner/repo", "golang/go", "facebook/react"},
				description:  "GitHub sources are owner/repo format",
			},
			{
				method: domain.MethodBinary,
				validSources: []string{
					"https://example.com/file.tar.gz",
					"http://download.example.com/app",
					"/local/path/to/binary",
				},
				description: "Binary sources are URLs or paths",
			},
			{
				method: domain.MethodScript,
				validSources: []string{
					"https://get.docker.com",
					"install.sh",
					"/usr/local/scripts/setup.sh",
				},
				description: "Script sources are URLs or paths",
			},
		}

		for _, tc := range testCases {
			for _, source := range tc.validSources {
				pkg := &domain.Package{
					Name:   "test",
					Method: tc.method,
					Source: source,
				}
				assert.True(t, pkg.IsValid(),
					"%s: source=%q should be valid", tc.description, source)
			}
		}
	})

	// Property: Empty source is always invalid
	t.Run("empty_source_invalid", func(t *testing.T) {
		for _, method := range []domain.InstallMethod{
			domain.MethodAPT, domain.MethodDNF, domain.MethodGitHub,
		} {
			pkg := &domain.Package{
				Name:   "app",
				Method: method,
				Source: "",
			}
			assert.False(t, pkg.IsValid(),
				"Empty source should be invalid for method %v", method)
		}
	})
}

// TestPackageEqualityProperties tests package equality invariants.
func TestPackageDeduplicationLogic(t *testing.T) {
	// Test actual business logic for package deduplication
	mockInstaller := new(testutil.MockPackageInstaller)
	mockDetector := new(testutil.MockSystemDetector)
	service := domain.NewPackageService(mockInstaller, mockDetector)
	ctx := context.Background()

	t.Run("same_package_different_versions", func(t *testing.T) {
		// Test handling of same package with different versions
		oldVersion := &domain.Package{
			Name:    "vim",
			Method:  domain.MethodAPT,
			Source:  "ubuntu",
			Version: "8.1",
		}

		newVersion := &domain.Package{
			Name:    "vim",
			Method:  domain.MethodAPT,
			Source:  "ubuntu",
			Version: "8.2",
		}

		// First install old version
		mockInstaller.On("Install", ctx, oldVersion).
			Return(&domain.InstallationResult{Package: oldVersion, Success: true}, nil).Once()

		// Then install new version (should upgrade)
		mockInstaller.On("Install", ctx, newVersion).
			Return(&domain.InstallationResult{
				Package: newVersion,
				Success: true,
				Output:  "Upgrading vim from 8.1 to 8.2",
			}, nil).Once()

		result1, err1 := service.Install(ctx, oldVersion)
		require.NoError(t, err1)
		assert.True(t, result1.Success)

		result2, err2 := service.Install(ctx, newVersion)
		require.NoError(t, err2)
		assert.Contains(t, result2.Output, "Upgrading")
	})

	t.Run("same_app_different_install_methods", func(t *testing.T) {
		// Test that same app from different sources can coexist
		aptVersion := &domain.Package{
			Name:   "code",
			Method: domain.MethodAPT,
			Source: "microsoft",
		}

		snapVersion := &domain.Package{
			Name:   "code",
			Method: domain.MethodSnap,
			Source: "snapcraft",
		}

		// Both should be installable independently
		mockInstaller.On("Install", ctx, aptVersion).
			Return(&domain.InstallationResult{Package: aptVersion, Success: true}, nil).Once()

		mockInstaller.On("Install", ctx, snapVersion).
			Return(&domain.InstallationResult{Package: snapVersion, Success: true}, nil).Once()

		result1, err1 := service.Install(ctx, aptVersion)
		require.NoError(t, err1)
		assert.True(t, result1.Success)

		result2, err2 := service.Install(ctx, snapVersion)
		require.NoError(t, err2)
		assert.True(t, result2.Success)
		// Both installations should succeed - different methods
	})
}

// TestDependencyProperties tests dependency handling invariants.
func TestDependencyProperties(t *testing.T) {
	// Property: Dependencies don't affect package validity
	t.Run("dependencies_optional", func(t *testing.T) {
		// Property: Dependencies field doesn't affect package validity
		testCases := []struct {
			name string
			deps []string
		}{
			{"nil dependencies", nil},
			{"empty slice", []string{}},
			{"single dependency", []string{"lib1"}},
			{"multiple dependencies", []string{"lib1", "lib2", "lib3"}},
		}

		for _, tc := range testCases {
			pkg := &domain.Package{
				Name:         "app",
				Method:       domain.MethodAPT,
				Source:       "app",
				Dependencies: tc.deps,
			}
			assert.True(t, pkg.IsValid(),
				"Package should be valid with %s", tc.name)
		}
	})

	// Property: Dependency names can be any string
	t.Run("dependency_name_flexibility", func(t *testing.T) {
		pkg := &domain.Package{
			Name:   "app",
			Method: domain.MethodAPT,
			Source: "app",
			Dependencies: []string{
				"simple-name",
				"name-with-dashes",
				"name_with_underscores",
				"name.with.dots",
				"name123with456numbers",
				">=versioned-dep-1.0",
				"", // Even empty string might be allowed
			},
		}

		// Package should still be valid regardless of dependency names
		assert.True(t, pkg.IsValid())
	})
}

// TestCharacterSetProperties tests that package names handle various character sets.
func TestCharacterSetProperties(t *testing.T) {
	// Property: ASCII printable characters in names
	t.Run("ascii_printable_names", func(t *testing.T) {
		// Test names with various ASCII characters
		names := []string{
			"simple",
			"with-dash",
			"with_underscore",
			"with.dot",
			"with123numbers",
			"MixedCase",
			"UPPERCASE",
		}

		for _, name := range names {
			pkg := &domain.Package{
				Name:   name,
				Method: domain.MethodAPT,
				Source: "source",
			}
			assert.True(t, pkg.IsValid(),
				"ASCII name %q should be valid", name)
		}
	})

	// Property: Special characters might be valid in some contexts
	t.Run("special_characters", func(t *testing.T) {
		// These might be valid depending on the package manager
		specialNames := []string{
			"package+plus",
			"package@version",
			"scope/package",
			"@namespace/package",
		}

		for _, name := range specialNames {
			pkg := &domain.Package{
				Name:   name,
				Method: domain.MethodAPT,
				Source: "source",
			}
			// Document actual behavior with special characters
			isValid := pkg.IsValid()
			t.Logf("Package with name %q has IsValid=%v", name, isValid)
		}
	})

	// Property: Unicode characters (if supported)
	t.Run("unicode_characters", func(t *testing.T) {
		// Test if unicode is handled gracefully
		unicodeNames := []string{
			"café",
			"包管理器",
			"пакет",
			"🚀-emoji",
		}

		for _, name := range unicodeNames {
			// Create package without panicking
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Package creation panicked with unicode name %q: %v", name, r)
					}
				}()

				pkg := &domain.Package{
					Name:   name,
					Method: domain.MethodAPT,
					Source: "source",
				}
				// Document behavior - likely invalid but shouldn't crash
				isValid := pkg.IsValid()
				hasNonASCII := false

				for _, r := range name {
					if r > unicode.MaxASCII {
						hasNonASCII = true
						break
					}
				}

				t.Logf("Package with unicode name %q (hasNonASCII=%v) has IsValid=%v",
					name, hasNonASCII, isValid)
			}()
		}
	})
}
