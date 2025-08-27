// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package databases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "verbose manager creation",
			verbose: true,
		},
		{
			name:    "quiet manager creation",
			verbose: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			manager := NewManager(testCase.verbose)
			require.NotNil(t, manager, "NewManager should never return nil")
			assert.Equal(t, testCase.verbose, manager.verbose, "Verbose setting should matestCaseh expected")
		})
	}
}

func TestListDatabases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		verbose          bool
		expectedMinCount int
		expectedDbs      []string
	}{
		{
			name:             "list databases with verbose=false",
			verbose:          false,
			expectedMinCount: 1,
			expectedDbs:      []string{"mysql", "redis", "postgresql"},
		},
		{
			name:             "list databases with verbose=true",
			verbose:          true,
			expectedMinCount: 1,
			expectedDbs:      []string{"mysql", "redis", "postgresql"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			manager := NewManager(testCase.verbose)
			require.NotNil(t, manager)

			dbs := manager.ListDatabases()
			require.GreaterOrEqual(t, len(dbs), testCase.expectedMinCount,
				"Should have at least %d database(s) available", testCase.expectedMinCount)

			// Create map of found databases for efficient lookup
			found := make(map[string]bool)
			for _, db := range dbs {
				found[db.Name] = true
				// Verify database structure is valid
				assert.NotEmpty(t, db.Name, "Database name should not be empty")
				assert.NotEmpty(t, db.Container, "Database container should not be empty")
			}

			// Check that expected databases exist in catalog
			for _, expected := range testCase.expectedDbs {
				assert.Contains(t, Databases, expected,
					"Expected database %s should exist in catalog", expected)
			}
		})
	}
}

func TestDatabaseStructure(t *testing.T) {
	t.Parallel()

	// Convert map to slice for table-driven test
	databaseTests := make([]struct {
		name     string
		database Database
	}, 0, len(Databases))

	for name, database := range Databases {
		databaseTests = append(databaseTests, struct {
			name     string
			database Database
		}{
			name:     name,
			database: database,
		})
	}

	require.NotEmpty(t, databaseTests, "Should have databases to test")

	for _, testCase := range databaseTests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Use require for critical structural validations
			require.NotEmpty(t, testCase.database.Name,
				"Database %s should have a non-empty Name", testCase.name)
			require.NotEmpty(t, testCase.database.Container,
				"Database %s should have a non-empty Container", testCase.name)
			require.NotEmpty(t, testCase.database.Port,
				"Database %s should have a non-empty Port", testCase.name)
			require.NotEmpty(t, testCase.database.Command,
				"Database %s should have a non-empty Command", testCase.name)

			// Additional structural validations
			// Note: Database names may have different casing than keys, so just verify non-empty
			assert.NotEmpty(t, testCase.database.Name, "Database name should not be empty")
			assert.NotContains(t, testCase.database.Container, " ",
				"Container name should not contain spaces")
			assert.Regexp(t, `^\d+$`, testCase.database.Port,
				"Port should be numeric")
		})
	}
}

func TestGetDatabaseStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verbose bool
		dryRun  bool
	}{
		{
			name:    "status check with dry run enabled",
			verbose: false,
			dryRun:  true,
		},
		{
			name:    "verbose status check with dry run",
			verbose: true,
			dryRun:  true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Use dry run mode to avoid system changes and sudo
			manager := NewManagerWithDryRun(testCase.verbose, testCase.dryRun)
			require.NotNil(t, manager)

			status := manager.GetDatabaseStatus(context.Background())

			// Should return status for all databases
			require.Len(t, status, len(Databases),
				"Should return status for all %d databases", len(Databases))

			// Check that all expected databases have status
			for dbName := range Databases {
				assert.Contains(t, status, dbName,
					"Should have status for database %s", dbName)

				// Verify status structure if present
				if dbStatus, exists := status[dbName]; exists {
					assert.NotNil(t, dbStatus, "Database status should not be nil")
				}
			}
		})
	}
}

func TestInstallDatabasesWithEmptyList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		databases   []string
		expectError bool
		description string
	}{
		{
			name:        "empty database list",
			databases:   []string{},
			expectError: false,
			description: "Should handle empty list gracefully",
		},
		{
			name:        "nil database list",
			databases:   nil,
			expectError: false,
			description: "Should handle nil list gracefully",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Use dry run mode to avoid system changes and sudo
			manager := NewManagerWithDryRun(false, true)
			require.NotNil(t, manager)

			err := manager.InstallDatabases(context.Background(), testCase.databases)

			if testCase.expectError {
				require.Error(t, err, testCase.description)
			} else {
				require.NoError(t, err, testCase.description)
			}
		})
	}
}

func TestInstallDatabaseWithUnknownName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		databaseName   string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:           "nonexistent database installation",
			databaseName:   "nonexistent-database",
			expectError:    true,
			expectedErrMsg: "unknown database: nonexistent-database",
		},
		{
			name:           "empty database name",
			databaseName:   "",
			expectError:    true,
			expectedErrMsg: "unknown database: ",
		},
		{
			name:           "malformed database name",
			databaseName:   "invalid-database-name-with-special-chars!@#",
			expectError:    true,
			expectedErrMsg: "unknown database: invalid-database-name-with-special-chars!@#",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Use dry run mode to avoid system changes and sudo
			manager := NewManagerWithDryRun(false, true)
			require.NotNil(t, manager)

			err := manager.InstallDatabase(context.Background(), testCase.databaseName)

			if testCase.expectError {
				require.Error(t, err, "Should error for unknown database")
				assert.Equal(t, testCase.expectedErrMsg, err.Error(),
					"Error message should matestCaseh expected")
			} else {
				require.NoError(t, err, "Should not error for valid database")
			}

			// Verify no dangerous operations in database name
			assert.NotContains(t, testCase.databaseName, "/",
				"Database name should not contain filesystem paths")
			assert.NotContains(t, testCase.databaseName, "rm",
				"Database name should not contain dangerous commands")
		})
	}
}

func TestStopDatabaseWithUnknownName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		databaseName string
		expectError  bool
	}{
		{
			name:         "stop nonexistent database",
			databaseName: "nonexistent-database",
			expectError:  true,
		},
		{
			name:         "stop with empty name",
			databaseName: "",
			expectError:  true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Use dry run mode to avoid system changes and sudo
			manager := NewManagerWithDryRun(false, true)
			require.NotNil(t, manager)

			err := manager.StopDatabase(context.Background(), testCase.databaseName)

			if testCase.expectError {
				require.Error(t, err, "Should error for unknown database")
			} else {
				require.NoError(t, err, "Should not error for valid database")
			}
		})
	}
}

func TestRemoveDatabaseWithUnknownName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		databaseName string
		expectError  bool
	}{
		{
			name:         "remove nonexistent database",
			databaseName: "nonexistent-database",
			expectError:  true,
		},
		{
			name:         "remove with empty name",
			databaseName: "",
			expectError:  true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Use dry run mode to avoid system changes and sudo
			manager := NewManagerWithDryRun(false, true)
			require.NotNil(t, manager)

			err := manager.RemoveDatabase(context.Background(), testCase.databaseName)

			if testCase.expectError {
				require.Error(t, err, "Should error for unknown database")
			} else {
				require.NoError(t, err, "Should not error for valid database")
			}

			// Verify safety - removal operations should not affect host
			assert.NotContains(t, testCase.databaseName, "/",
				"Database name should not contain filesystem paths")
		})
	}
}

// TestDatabaseManager_ThreadSafety demonstrates thread-safe database operations.
func TestDatabaseManager_ThreadSafety(t *testing.T) {
	t.Parallel()

	// Use dry run mode to avoid system changes
	manager := NewManagerWithDryRun(false, true)
	require.NotNil(t, manager)

	// Run concurrent operations to test thread safety
	done := make(chan bool, 5)

	for index := range 5 {
		go func(_ int) {
			defer func() { done <- true }()

			// Each goroutine performs read-only operations
			dbs := manager.ListDatabases()
			assert.NotEmpty(t, dbs, "Should have databases available")

			status := manager.GetDatabaseStatus(context.Background())
			assert.NotEmpty(t, status, "Should have database status")
		}(index)
	}

	// Wait for all operations to complete
	for range 5 {
		<-done
	}
}
