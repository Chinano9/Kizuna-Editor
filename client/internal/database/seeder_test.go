package database

import (
	"database/sql"
	"kizuna/client/internal/database/migrator"
	"kizuna/shared/models"
	"testing"
)

func TestSeedInstruments(t *testing.T) {
	t.Run("seeds initial instruments and checks IDs", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("failed to open in-memory db: %v", err)
		}
		defer db.Close()

		// Run migrations first so tables exist
		m := migrator.NewMigrator(db)
		if err := m.Migrate(); err != nil {
			t.Fatalf("failed to migrate db: %v", err)
		}

		// Seed instruments
		if err := SeedInstruments(db); err != nil {
			t.Fatalf("failed to seed instruments: %v", err)
		}

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM instruments").Scan(&count); err != nil {
			t.Fatalf("failed to count instruments: %v", err)
		}
		if count != 5 {
			t.Errorf("expected 5 instruments, got %d", count)
		}

		// Verify electric guitar has ID = 1
		var name, clef string
		err = db.QueryRow("SELECT name, default_clef FROM instruments WHERE id = ?", models.InstrumentElectricGuitarID).Scan(&name, &clef)
		if err != nil {
			t.Fatalf("failed to query instrument ID 1: %v", err)
		}
		if name != "Electric Guitar" || clef != "treble" {
			t.Errorf("expected Electric Guitar with treble clef, got %s / %s", name, clef)
		}
	})

	t.Run("seeding is idempotent", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("failed to open in-memory db: %v", err)
		}
		defer db.Close()

		m := migrator.NewMigrator(db)
		if err := m.Migrate(); err != nil {
			t.Fatalf("failed to migrate db: %v", err)
		}

		if err := SeedInstruments(db); err != nil {
			t.Fatalf("first seed run failed: %v", err)
		}
		// Run again
		if err := SeedInstruments(db); err != nil {
			t.Fatalf("second seed run failed: %v", err)
		}

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM instruments").Scan(&count); err != nil {
			t.Fatalf("failed to count instruments: %v", err)
		}
		if count != 5 {
			t.Errorf("expected still 5 instruments after re-seeding, got %d", count)
		}
	})
}
