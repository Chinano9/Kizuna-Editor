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

package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Config encapsulates configuration parameters for opening an SQLite database connection.
type Config struct {
	Path            string
	BusyTimeout     time.Duration
	JournalMode     string
	ForeignKeys     bool
	Synchronous     string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns an optimal configuration for local-first desktop usage.
func DefaultConfig(path string) Config {
	return Config{
		Path:            path,
		BusyTimeout:     5 * time.Second,
		JournalMode:     "WAL",
		ForeignKeys:     true,
		Synchronous:     "NORMAL",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}
}

// Open initializes and configures a robust SQLite database connection.
// It applies essential PRAGMAs (WAL mode, foreign key enforcement, busy timeout)
// to prevent "database is locked" errors and ensure referential integrity.
func Open(cfg Config) (*sql.DB, error) {
	if strings.TrimSpace(cfg.Path) == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}

	fkValue := "OFF"
	if cfg.ForeignKeys {
		fkValue = "ON"
	}

	busyMs := cfg.BusyTimeout.Milliseconds()
	if busyMs <= 0 {
		busyMs = 5000
	}

	journalMode := cfg.JournalMode
	if journalMode == "" {
		journalMode = "WAL"
	}

	syncMode := cfg.Synchronous
	if syncMode == "" {
		syncMode = "NORMAL"
	}

	// Build DSN with parameters supported by modernc.org/sqlite
	separator := "?"
	if strings.Contains(cfg.Path, "?") {
		separator = "&"
	}
	dsn := fmt.Sprintf("%s%s_pragma=busy_timeout(%d)&_pragma=journal_mode(%s)&_pragma=foreign_keys(%s)&_pragma=synchronous(%s)",
		cfg.Path, separator, busyMs, journalMode, fkValue, syncMode)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite connection: %w", err)
	}

	// Configure connection pooling
	isMemory := cfg.Path == ":memory:" || strings.Contains(cfg.Path, "mode=memory")
	if isMemory {
		// In-memory databases must use 1 max connection to prevent losing schema across pools
		db.SetMaxOpenConns(1)
	} else {
		maxOpen := cfg.MaxOpenConns
		if maxOpen <= 0 {
			maxOpen = 10
		}
		db.SetMaxOpenConns(maxOpen)
	}

	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}
	db.SetMaxIdleConns(maxIdle)

	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	// Explicitly verify and set pragmas via Exec for guaranteed enforcement across connection pools
	if cfg.ForeignKeys {
		if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to enable foreign keys pragma: %w", err)
		}
	}

	if busyMs > 0 {
		if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d;", busyMs)); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set busy_timeout pragma: %w", err)
		}
	}

	// Verify connection with Ping
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("database ping check failed: %w", err)
	}

	return db, nil
}
