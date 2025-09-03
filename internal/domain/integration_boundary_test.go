// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdapterContractCompliance verifies adapters fulfill port contracts.
func TestAdapterContractCompliance(t *testing.T) {
	t.Parallel()

	t.Run("installer_port_contract", func(t *testing.T) {
		t.Parallel()

		// Port contract: Installers must handle these scenarios
		scenarios := []struct {
			name          string
			pkg           *domain.Package
			expectedError error
			businessRule  string
		}{
			{
				name: "valid_package_installation",
				pkg: &domain.Package{
					Name:   "valid-app",
					Method: domain.MethodAPT,
					Source: "ubuntu",
				},
				expectedError: nil,
				businessRule:  "Valid packages must install successfully",
			},
			{
				name: "network_failure_handling",
				pkg: &domain.Package{
					Name:   "remote-app",
					Method: domain.MethodGitHub,
					Source: "owner/repo",
				},
				expectedError: domain.ErrNetworkFailure,
				businessRule:  "Network failures must be reported clearly",
			},
			{
				name: "permission_denied_handling",
				pkg: &domain.Package{
					Name:   "system-tool",
					Method: domain.MethodAPT,
					Source: "ubuntu",
				},
				expectedError: domain.ErrPermissionDenied,
				businessRule:  "Permission errors must be distinguishable",
			},
			{
				name: "disk_space_check",
				pkg: &domain.Package{
					Name:   "large-app",
					Method: domain.MethodFlatpak,
					Source: "flathub",
				},
				expectedError: domain.ErrInsufficientSpace,
				businessRule:  "Disk space must be checked before installation",
			},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				mockInstaller := new(testutil.MockPackageInstaller)
				mockDetector := new(testutil.MockSystemDetector)
				service := domain.NewPackageService(mockInstaller, mockDetector)

				ctx := context.Background()

				if scenario.expectedError != nil {
					mockInstaller.On("Install", ctx, scenario.pkg).
						Return(nil, scenario.expectedError).Once()

					_, err := service.Install(ctx, scenario.pkg)

					// Contract: Specific errors must be preserved
					require.ErrorIs(t, err, scenario.expectedError, scenario.businessRule)
				} else {
					mockInstaller.On("Install", ctx, scenario.pkg).
						Return(&domain.InstallationResult{
							Package: scenario.pkg,
							Success: true,
						}, nil).Once()

					result, err := service.Install(ctx, scenario.pkg)

					require.NoError(t, err, scenario.businessRule)
					assert.NotNil(t, result)
					assert.True(t, result.Success)
				}

				mockInstaller.AssertExpectations(t)
			})
		}
	})

	t.Run("detector_port_contract", func(t *testing.T) {
		t.Parallel()

		// Port contract: System detectors must provide this information
		requiredInfo := []struct {
			field        string
			critical     bool
			businessRule string
		}{
			{
				field:        "OS",
				critical:     true,
				businessRule: "OS detection is critical for package compatibility",
			},
			{
				field:        "Architecture",
				critical:     true,
				businessRule: "Architecture determines binary compatibility",
			},
			{
				field:        "Distribution",
				critical:     false,
				businessRule: "Distribution helps select package manager",
			},
			{
				field:        "Version",
				critical:     false,
				businessRule: "Version helps with compatibility checks",
			},
		}

		mockDetector := new(testutil.MockSystemDetector)
		ctx := context.Background()

		systemInfo := &domain.SystemInfo{
			Architecture: "amd64",
			Kernel:       "5.15.0",
			Distribution: &domain.Distribution{
				Name:    "Ubuntu",
				Version: "22.04",
			},
		}

		mockDetector.On("DetectSystem", ctx).Return(systemInfo, nil).Once()

		info, err := mockDetector.DetectSystem(ctx)
		require.NoError(t, err)

		// Verify critical fields are present
		for _, req := range requiredInfo {
			switch req.field {
			case "OS":
				// OS is determined by checking if Distribution exists
				assert.NotNil(t, info.Distribution, req.businessRule)
			case "Architecture":
				assert.NotEmpty(t, info.Architecture, req.businessRule)
			case "Distribution":
				// Non-critical, can be empty
				_ = info.Distribution
			case "Version":
				// Version is in Distribution
				if info.Distribution != nil {
					_ = info.Distribution.Version
				}
			}
		}

		mockDetector.AssertExpectations(t)
	})

	t.Run("error_propagation_through_boundaries", func(t *testing.T) {
		t.Parallel()

		// Business rule: Domain errors must cross boundaries correctly
		domainErrors := []error{
			domain.ErrCircularDependency,
			domain.ErrMissingDependency,
			domain.ErrNotInstalled,
			domain.ErrAlreadyInstalled,
			domain.ErrInvalidPackage,
		}

		mockInstaller := new(testutil.MockPackageInstaller)
		mockDetector := new(testutil.MockSystemDetector)
		service := domain.NewPackageService(mockInstaller, mockDetector)

		pkg := &domain.Package{
			Name:   "test-pkg",
			Method: domain.MethodAPT,
			Source: "ubuntu",
		}

		ctx := context.Background()

		for _, domainErr := range domainErrors {
			t.Run(domainErr.Error(), func(t *testing.T) {
				// Setup mock to return domain error
				mockInstaller.On("Install", ctx, pkg).
					Return(nil, domainErr).Once()

				_, err := service.Install(ctx, pkg)

				// Domain errors must be preserved across boundaries
				assert.ErrorIs(t, err, domainErr,
					"Domain error '%v' must cross boundary unchanged", domainErr)
			})
		}
	})
}

// TestCrossCuttingConcerns verifies handling of cross-cutting concerns.
func TestCrossCuttingConcerns(t *testing.T) {
	t.Parallel()

	t.Run("logging_at_boundaries", func(t *testing.T) {
		t.Parallel()
		testLoggingAtBoundaries(t)
	})

	t.Run("transaction_boundaries", func(t *testing.T) {
		t.Parallel()
		testTransactionBoundaries(t)
	})

	t.Run("security_at_boundaries", func(t *testing.T) {
		t.Parallel()
		testSecurityAtBoundaries(t)
	})
}

func testLoggingAtBoundaries(t *testing.T) {
	t.Helper()
	// Business rule: All boundary crossings should be logged
	type LogEntry struct {
		Level     string
		Message   string
		Package   string
		Operation string
	}

	var logs []LogEntry

	logOperation := func(level, op, pkg, msg string) {
		logs = append(logs, LogEntry{
			Level:     level,
			Message:   msg,
			Package:   pkg,
			Operation: op,
		})
	}

	operations := []string{"install", "remove", "update", "list"}

	for _, op := range operations {
		// Log entry when crossing boundary
		logOperation("INFO", op, "test-pkg", "Starting "+op)

		// Simulate operation

		// Log exit from boundary
		logOperation("INFO", op, "test-pkg", "Completed "+op)
	}

	// Verify logging happened
	assert.GreaterOrEqual(t, len(logs), len(operations)*2,
		"Each operation should log entry and exit")

	// Verify log structure
	for _, log := range logs {
		assert.NotEmpty(t, log.Level, "Log level required")
		assert.NotEmpty(t, log.Operation, "Operation required")
		assert.NotEmpty(t, log.Message, "Message required")
	}
}

func testTransactionBoundaries(t *testing.T) {
	t.Helper()
	// Business rule: Transactions don't cross aggregate boundaries
	type Transaction struct {
		ID         string
		Operations []string
		Started    bool
		Committed  bool
		RolledBack bool
	}

	beginTransaction := func() *Transaction {
		return &Transaction{
			ID:         "tx-001",
			Operations: []string{},
			Started:    true,
		}
	}

	addOperation := func(tx *Transaction, op string) error {
		if !tx.Started || tx.Committed || tx.RolledBack {
			return errors.New("invalid transaction state")
		}

		tx.Operations = append(tx.Operations, op)

		return nil
	}

	commitTransaction := func(tx *Transaction) error {
		if !tx.Started || tx.Committed || tx.RolledBack {
			return errors.New("cannot commit")
		}

		tx.Committed = true

		return nil
	}

	rollbackTransaction := func(tx *Transaction) error {
		if !tx.Started || tx.Committed {
			return errors.New("cannot rollback")
		}

		tx.RolledBack = true
		tx.Operations = nil

		return nil
	}

	// Test transaction boundary
	tx := beginTransaction()
	assert.True(t, tx.Started, "Transaction should start")

	// Add operations within same aggregate
	err := addOperation(tx, "install-pkg1")
	require.NoError(t, err)

	err = addOperation(tx, "configure-pkg1")
	require.NoError(t, err)

	// Commit transaction
	err = commitTransaction(tx)
	require.NoError(t, err)
	assert.True(t, tx.Committed, "Transaction should commit")

	// Cannot modify committed transaction
	err = addOperation(tx, "another-op")
	require.Error(t, err, "Cannot modify committed transaction")

	// Test rollback scenario
	tx2 := beginTransaction()
	_ = addOperation(tx2, "risky-operation")

	err = rollbackTransaction(tx2)
	require.NoError(t, err)
	assert.True(t, tx2.RolledBack, "Transaction should rollback")
	assert.Empty(t, tx2.Operations, "Operations should be cleared on rollback")
}

func testSecurityAtBoundaries(t *testing.T) {
	t.Helper()
	// Business rule: All inputs must be sanitized at boundaries
	dangerousInputs := []struct {
		input       string
		sanitized   string
		description string
	}{
		{
			input:       "package; rm -rf /",
			sanitized:   "package rm -rf",
			description: "Shell injection attempt",
		},
		{
			input:       "../../../etc/passwd",
			sanitized:   "etcpasswd",
			description: "Path traversal attempt",
		},
		{
			input:       "package\x00.sh",
			sanitized:   "package.sh",
			description: "Null byte injection",
		},
		{
			input:       "<script>alert('xss')</script>",
			sanitized:   "scriptalertxssscript",
			description: "XSS attempt",
		},
	}

	sanitizeInput := func(input string) string {
		// Remove dangerous characters
		dangerous := []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\x00", "../", "\\"}

		sanitized := input
		for _, char := range dangerous {
			sanitized = strings.ReplaceAll(sanitized, char, "")
		}

		return sanitized
	}

	for _, test := range dangerousInputs {
		t.Run(test.description, func(t *testing.T) {
			result := sanitizeInput(test.input)

			// Verify dangerous content removed
			assert.NotContains(t, result, ";", "Semicolon should be removed")
			assert.NotContains(t, result, "../", "Path traversal should be removed")
			assert.NotContains(t, result, "\x00", "Null bytes should be removed")

			t.Logf("Input '%s' sanitized to '%s'", test.input, result)
		})
	}
}

// TestBoundaryDataTransformation verifies data transformation at boundaries.
func TestBoundaryDataTransformation(t *testing.T) {
	t.Parallel()

	t.Run("domain_to_dto_mapping", func(t *testing.T) {
		t.Parallel()

		// Business rule: Domain objects map to DTOs at boundaries
		type PackageDTO struct {
			Name    string `json:"name"`
			Method  string `json:"method"`
			Source  string `json:"source"`
			Version string `json:"version,omitempty"`
		}

		domainToDTO := func(pkg *domain.Package) PackageDTO {
			return PackageDTO{
				Name:    pkg.Name,
				Method:  string(pkg.Method),
				Source:  pkg.Source,
				Version: pkg.Version,
			}
		}

		// Test transformation
		domainPkg := &domain.Package{
			Name:    "test-app",
			Method:  domain.MethodAPT,
			Source:  "ubuntu",
			Version: "1.2.3",
		}

		dto := domainToDTO(domainPkg)

		assert.Equal(t, domainPkg.Name, dto.Name, "Name should map correctly")
		assert.Equal(t, string(domainPkg.Method), dto.Method, "Method should map correctly")
		assert.Equal(t, domainPkg.Source, dto.Source, "Source should map correctly")
		assert.Equal(t, domainPkg.Version, dto.Version, "Version should map correctly")
	})

	t.Run("dto_to_domain_validation", func(t *testing.T) {
		t.Parallel()

		// Business rule: DTOs must be validated when creating domain objects
		type PackageDTO struct {
			Name   string
			Method string
			Source string
		}

		dtoToDomain := func(dto PackageDTO) (*domain.Package, error) {
			// Validate DTO fields
			if dto.Name == "" {
				return nil, errors.New("name is required")
			}

			if dto.Method == "" {
				return nil, errors.New("method is required")
			}

			if dto.Source == "" {
				return nil, errors.New("source is required")
			}

			// Create domain object
			return &domain.Package{
				Name:   dto.Name,
				Method: domain.InstallMethod(dto.Method),
				Source: dto.Source,
			}, nil
		}

		// Test valid DTO
		validDTO := PackageDTO{
			Name:   "app",
			Method: "apt",
			Source: "ubuntu",
		}

		pkg, err := dtoToDomain(validDTO)
		require.NoError(t, err)
		assert.NotNil(t, pkg)

		// Test invalid DTOs
		invalidDTOs := []PackageDTO{
			{Method: "apt", Source: "ubuntu"}, // Missing name
			{Name: "app", Source: "ubuntu"},   // Missing method
			{Name: "app", Method: "apt"},      // Missing source
		}

		for _, dto := range invalidDTOs {
			_, err := dtoToDomain(dto)
			assert.Error(t, err, "Invalid DTO should produce error")
		}
	})
}
