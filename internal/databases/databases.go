// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package databases provides database management functionality.
package databases

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

var (
	// ErrUnknownDatabase is returned when the requested database is not found.
	ErrUnknownDatabase = errors.New("unknown database")
)

// Database represents a database configuration.
type Database struct {
	Name      string
	Container string
	Port      string
	Command   []string
}

// Databases contains available database configurations.
var Databases = map[string]Database{ //nolint:gochecknoglobals
	"mysql": {
		Name:      "MySQL",
		Container: "mysql8",
		Port:      "3306",
		Command: []string{
			"docker", "run", "-d", "--restart", "unless-stopped",
			"-p", "127.0.0.1:3306:3306", "--name=mysql8",
			"-e", "MYSQL_ROOT_PASSWORD=", "-e", "MYSQL_ALLOW_EMPTY_PASSWORD=true",
			"mysql:8.4",
		},
	},
	"redis": {
		Name:      "Redis",
		Container: "redis",
		Port:      "6379",
		Command: []string{
			"docker", "run", "-d", "--restart", "unless-stopped",
			"-p", "127.0.0.1:6379:6379", "--name=redis",
			"redis:7",
		},
	},
	"postgresql": {
		Name:      "PostgreSQL",
		Container: "postgres16",
		Port:      "5432",
		Command: []string{
			"docker", "run", "-d", "--restart", "unless-stopped",
			"-p", "127.0.0.1:5432:5432", "--name=postgres16",
			"-e", "POSTGRES_HOST_AUTH_METHOD=trust",
			"postgres:16",
		},
	},
}

// Manager handles database operations.
type Manager struct {
	verbose bool
	dryRun  bool
}

// NewManager creates a new database manager.
func NewManager(verbose bool) *Manager {
	return &Manager{verbose: verbose, dryRun: false}
}

// NewManagerWithDryRun creates a new database manager with dry run option.
func NewManagerWithDryRun(verbose bool, dryRun bool) *Manager {
	return &Manager{verbose: verbose, dryRun: dryRun}
}

// InstallDatabase installs a database by name.
func (m *Manager) InstallDatabase(ctx context.Context, dbName string) error {
	database, exists := Databases[dbName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownDatabase, dbName)
	}

	// Check if container already exists
	if m.isContainerRunning(ctx, database.Container) {
		if m.verbose {
			fmt.Printf("%s container already running\n", database.Name)
		}

		return nil
	}

	if m.verbose {
		fmt.Printf("Installing %s database container...\n", database.Name)
	}

	if m.dryRun {
		fmt.Printf("DRY RUN: sudo %v\n", database.Command)

		return nil
	}

	cmd := exec.CommandContext(ctx, "sudo", database.Command...) //nolint:gosec

	return cmd.Run()
}

// InstallDatabases installs multiple databases.
func (m *Manager) InstallDatabases(ctx context.Context, dbNames []string) error {
	for _, dbName := range dbNames {
		if err := m.InstallDatabase(ctx, dbName); err != nil {
			fmt.Printf("Warning: Failed to install %s: %v\n", dbName, err)
		}
	}

	return nil
}

// StopDatabase stops a running database container.
func (m *Manager) StopDatabase(ctx context.Context, dbName string) error {
	database, exists := Databases[dbName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownDatabase, dbName)
	}

	if m.dryRun {
		fmt.Printf("DRY RUN: sudo docker stop %s\n", database.Container)

		return nil
	}

	cmd := exec.CommandContext(ctx, "sudo", "docker", "stop", database.Container) //nolint:gosec

	return cmd.Run()
}

// RemoveDatabase stops and removes a database container.
func (m *Manager) RemoveDatabase(ctx context.Context, dbName string) error {
	database, exists := Databases[dbName]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownDatabase, dbName)
	}

	if m.dryRun {
		fmt.Printf("DRY RUN: sudo docker stop %s\n", database.Container)
		fmt.Printf("DRY RUN: sudo docker rm %s\n", database.Container)

		return nil
	}

	// Stop and remove container
	_ = exec.CommandContext(ctx, "sudo", "docker", "stop", database.Container).Run() //nolint:gosec
	cmd := exec.CommandContext(ctx, "sudo", "docker", "rm", database.Container)      //nolint:gosec

	return cmd.Run()
}

// ListDatabases returns all available databases.
func (m *Manager) ListDatabases() []Database {
	dbs := make([]Database, 0, len(Databases))
	for _, database := range Databases {
		dbs = append(dbs, database)
	}

	return dbs
}

// GetDatabaseStatus returns the running status of all databases.
func (m *Manager) GetDatabaseStatus(ctx context.Context) map[string]bool {
	status := make(map[string]bool)
	for name, db := range Databases {
		status[name] = m.isContainerRunning(ctx, db.Container)
	}

	return status
}

func (m *Manager) isContainerRunning(ctx context.Context, containerName string) bool {
	if m.dryRun {
		// In dry run mode, return false to simulate no containers running
		return false
	}

	cmd := exec.CommandContext(ctx, "sudo", "docker", "ps", "--filter", "name="+containerName, "--format", "{{.Names}}") //nolint:gosec

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return len(output) > 0
}
