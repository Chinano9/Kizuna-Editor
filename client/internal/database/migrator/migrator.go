/*
Kizuna Editor - A local-first songwriting environment.
Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package migrator

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed sql/*.sql
var migrationFS embed.FS

// Migration represents a single parsed database migration script.
type Migration struct {
	Version  int
	Name     string
	Filename string
	SQL      string
}

// MigrationRecord represents an entry in the schema_migrations table.
type MigrationRecord struct {
	Version   int
	Name      string
	AppliedAt time.Time
}

// Migrator manages and executes database migrations safely inside transactions.
type Migrator struct {
	db *sql.DB
	fs embed.FS
}

// NewMigrator returns an initialized Migrator instance using embedded SQL scripts.
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db: db,
		fs: migrationFS,
	}
}

// Migrate executes all pending migrations sequentially inside transactions.
func (m *Migrator) Migrate() error {
	if m == nil || m.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// 1. Ensure schema_migrations table exists
	if err := m.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// 2. Handle legacy schema patch (if database was created by previous legacy versions)
	if err := m.patchLegacySchema(); err != nil {
		return fmt.Errorf("failed to patch legacy schema: %w", err)
	}

	// 3. Load all applied migration versions
	applied, err := m.getAppliedVersions()
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}

	// 4. Discover and parse available migration scripts
	migrations, err := m.discoverMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migration scripts: %w", err)
	}

	// 5. Execute unapplied migrations in ascending order
	for _, mig := range migrations {
		if applied[mig.Version] {
			continue // Already applied
		}

		log.Printf("Applying database migration %06d: %s...", mig.Version, mig.Name)
		if err := m.applyMigration(mig); err != nil {
			return fmt.Errorf("migration %06d (%s) failed: %w", mig.Version, mig.Name, err)
		}
		log.Printf("Successfully applied migration %06d: %s", mig.Version, mig.Name)
	}

	return nil
}

// CurrentVersion returns the highest migration version currently applied, or 0 if none.
func (m *Migrator) CurrentVersion() (int, error) {
	if m == nil || m.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	var currentVersion int
	query := "SELECT COALESCE(MAX(version), 0) FROM schema_migrations"
	err := m.db.QueryRow(query).Scan(&currentVersion)
	if err != nil {
		// Table might not exist yet
		return 0, nil
	}
	return currentVersion, nil
}

// GetAppliedMigrations returns all migration records currently logged in the database.
func (m *Migrator) GetAppliedMigrations() ([]MigrationRecord, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	rows, err := m.db.Query("SELECT version, name, applied_at FROM schema_migrations ORDER BY version ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var r MigrationRecord
		if err := rows.Scan(&r.Version, &r.Name, &r.AppliedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

func (m *Migrator) ensureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) patchLegacySchema() error {
	// Check if legacy audio_versions table exists
	var tableName string
	err := m.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='audio_versions'").Scan(&tableName)
	if err != nil {
		// Table does not exist yet; standard migration will create it with track_id included
		return nil
	}

	// Table exists, check if track_id column is present
	rows, err := m.db.Query("PRAGMA table_info(audio_versions)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasTrackID := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "track_id" {
			hasTrackID = true
			break
		}
	}

	if !hasTrackID {
		log.Println("Patching legacy database: Adding missing 'track_id' column to audio_versions...")
		_, err := m.db.Exec("ALTER TABLE audio_versions ADD COLUMN track_id INTEGER REFERENCES tracks(id) ON DELETE CASCADE")
		if err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return err
		}
	}
	return nil
}

func (m *Migrator) getAppliedVersions() (map[int]bool, error) {
	applied := make(map[int]bool)
	rows, err := m.db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, nil
}

func (m *Migrator) discoverMigrations() ([]Migration, error) {
	entries, err := m.fs.ReadDir("sql")
	if err != nil {
		return nil, fmt.Errorf("unable to read migration sql directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(parts[1], ".up.sql")
		content, err := m.fs.ReadFile("sql/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", entry.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			Filename: entry.Name(),
			SQL:      string(content),
		})
	}

	// Sort ascending by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (m *Migrator) applyMigration(mig Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Execute migration queries
	if _, err := tx.Exec(mig.SQL); err != nil {
		return fmt.Errorf("failed to execute SQL statements: %w", err)
	}

	// Record migration in tracking table
	insertQuery := "INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, CURRENT_TIMESTAMP)"
	if _, err := tx.Exec(insertQuery, mig.Version, mig.Name); err != nil {
		return fmt.Errorf("failed to record migration log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}
	return nil
}
