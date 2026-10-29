package migrator

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	return db
}

func TestMigrator(t *testing.T) {
	t.Run("successfully runs all migrations on empty db", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		m := NewMigrator(db)
		if err := m.Migrate(); err != nil {
			t.Fatalf("migration failed: %v", err)
		}

		currentVersion, err := m.CurrentVersion()
		if err != nil {
			t.Fatalf("failed to get current version: %v", err)
		}
		if currentVersion < 2 {
			t.Errorf("expected current version >= 2, got %d", currentVersion)
		}

		applied, err := m.GetAppliedMigrations()
		if err != nil {
			t.Fatalf("failed to get applied migrations: %v", err)
		}
		if len(applied) < 2 {
			t.Fatalf("expected at least 2 applied migrations, got %d", len(applied))
		}

		// Verify tables were created
		expectedTables := []string{"albums", "songs", "instruments", "tracks", "audio_versions", "schema_migrations"}
		for _, table := range expectedTables {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
			if err != nil || count != 1 {
				t.Errorf("expected table '%s' to exist", table)
			}
		}

		// Verify indexes were created
		expectedIndexes := []string{"idx_songs_album_id", "idx_songs_updated_at", "idx_tracks_song_id", "idx_audio_versions_track_id"}
		for _, idx := range expectedIndexes {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&count)
			if err != nil || count != 1 {
				t.Errorf("expected index '%s' to exist", idx)
			}
		}
	})

	t.Run("migration is idempotent", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		m := NewMigrator(db)
		if err := m.Migrate(); err != nil {
			t.Fatalf("first migration run failed: %v", err)
		}

		// Run again immediately
		if err := m.Migrate(); err != nil {
			t.Fatalf("second migration run failed: %v", err)
		}

		applied, err := m.GetAppliedMigrations()
		if err != nil {
			t.Fatalf("failed to get applied migrations: %v", err)
		}
		if len(applied) < 2 {
			t.Errorf("expected at least 2 applied migrations, got %d", len(applied))
		}
	})

	t.Run("handles legacy schema by adding track_id if missing", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		// Simulate legacy database where audio_versions table existed without track_id
		legacySQL := `
			CREATE TABLE audio_versions (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				song_id INTEGER NOT NULL,
				version_name TEXT,
				file_path TEXT NOT NULL,
				notes TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`
		if _, err := db.Exec(legacySQL); err != nil {
			t.Fatalf("failed to create legacy table: %v", err)
		}

		m := NewMigrator(db)
		if err := m.Migrate(); err != nil {
			t.Fatalf("migrator failed on legacy table: %v", err)
		}

		// Verify track_id now exists in table
		rows, err := db.Query("PRAGMA table_info(audio_versions)")
		if err != nil {
			t.Fatalf("pragma table_info failed: %v", err)
		}
		defer rows.Close()

		foundTrackID := false
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dfltValue sql.NullString
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
				t.Fatalf("scan row failed: %v", err)
			}
			if name == "track_id" {
				foundTrackID = true
				break
			}
		}

		if !foundTrackID {
			t.Errorf("expected legacy table to have 'track_id' column patched in")
		}
	})
}
