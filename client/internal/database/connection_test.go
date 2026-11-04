package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestOpen(t *testing.T) {
	t.Run("open in-memory database successfully", func(t *testing.T) {
		cfg := DefaultConfig(":memory:")
		db, err := Open(cfg)
		if err != nil {
			t.Fatalf("failed to open in-memory database: %v", err)
		}
		defer db.Close()

		var fk int
		if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fk); err != nil {
			t.Fatalf("failed to query foreign_keys pragma: %v", err)
		}
		if fk != 1 {
			t.Errorf("expected foreign_keys to be enabled (1), got %d", fk)
		}
	})

	t.Run("open file-based database with WAL and busy timeout", func(t *testing.T) {
		tempDir := t.TempDir()
		dbFile := filepath.Join(tempDir, "test_wal.db")

		cfg := DefaultConfig(dbFile)
		cfg.BusyTimeout = 3 * time.Second
		db, err := Open(cfg)
		if err != nil {
			t.Fatalf("failed to open file database: %v", err)
		}
		defer db.Close()

		var journalMode string
		if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
			t.Fatalf("failed to query journal_mode: %v", err)
		}
		if journalMode != "wal" {
			t.Errorf("expected journal_mode 'wal', got '%s'", journalMode)
		}

		var busyTimeout int
		if err := db.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
			t.Fatalf("failed to query busy_timeout: %v", err)
		}
		if busyTimeout < 3000 {
			t.Errorf("expected busy_timeout >= 3000ms, got %d", busyTimeout)
		}
	})

	t.Run("fails when path is empty", func(t *testing.T) {
		cfg := DefaultConfig("")
		_, err := Open(cfg)
		if err == nil {
			t.Fatalf("expected error when path is empty, got nil")
		}
	})
}
