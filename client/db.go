package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite" // CGO-free SQLite driver
)

// Constants for known Seed IDs (To avoid magic numbers)
const (
	InstrumentGuitarID = 1
)

// DBManager handles all direct database interactions.
type DBManager struct {
	db *sql.DB
}

// NewDBManager initializes the SQLite connection and ensures the schema exists.
func NewDBManager() *DBManager {
	db, err := sql.Open("sqlite", "kizuna.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// 1. Initialize Schema
	if err := createFullSchema(db); err != nil {
		log.Fatal("Failed to create schema:", err)
	}

	// 2. Seed Initial Data
	if err := seedInstruments(db); err != nil {
		log.Println("Warning: Failed to seed instruments:", err)
	}

	return &DBManager{db: db}
}

// SaveQuickIdea handles the "Upsert" logic for the editor.
// It uses transactions to ensure data integrity between Songs and Tracks.
func (m *DBManager) SaveQuickIdea(songID int, title string, content string) int64 {
	// Start a transaction. If anything fails, we Rollback.
	tx, err := m.db.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		return 0
	}
	// Defer a rollback in case of panic or error (ignored if Commit is called)
	defer tx.Rollback()

	var finalID int64

	// --- CASE 1: NEW SONG (INSERT) ---
	if songID == 0 {
		// A. Create Song
		res, err := tx.Exec("INSERT INTO songs (title, bpm) VALUES (?, ?)", title, 120)
		if err != nil {
			log.Println("Error inserting song:", err)
			return 0
		}

		finalID, _ = res.LastInsertId()

		// B. Create Default Track (Guitar)
		_, err = tx.Exec(`
			INSERT INTO tracks (song_id, instrument_id, name, data_content)
			VALUES (?, ?, ?, ?)`,
			finalID, InstrumentGuitarID, "Lead Guitar", content)

		if err != nil {
			log.Println("Error inserting initial track:", err)
			return 0
		}

	} else {
		// --- CASE 2: EXISTING SONG (UPDATE) ---
		finalID = int64(songID)

		// A. Update Song Metadata
		_, err := tx.Exec("UPDATE songs SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", title, songID)
		if err != nil {
			log.Println("Error updating song title:", err)
			return 0
		}

		// B. Update Track Content
		res, err := tx.Exec("UPDATE tracks SET data_content = ? WHERE song_id = ? AND instrument_id = ?", content, songID, InstrumentGuitarID)
		if err != nil {
			log.Println("Error updating track content:", err)
			return 0
		}

		// Check if the track actually existed
		rowsAffected, _ := res.RowsAffected()

		// C. Edge Case: Song exists but has no tracks (Ghost Song) -> Create Track
		if rowsAffected == 0 {
			log.Printf("⚠️ Track missing for Song %d. Creating recovery track...", songID)
			_, err = tx.Exec(`
				INSERT INTO tracks (song_id, instrument_id, name, data_content)
				VALUES (?, ?, ?, ?)`,
				songID, InstrumentGuitarID, "Lead Guitar", content)

			if err != nil {
				log.Println("Error creating recovery track:", err)
				return 0
			}
		}
	}

	// Commit the transaction to save changes to disk
	if err := tx.Commit(); err != nil {
		log.Println("Error committing transaction:", err)
		return 0
	}

	return finalID
}

// --- PRIVATE HELPERS ---

func createFullSchema(db *sql.DB) error {
	queries := []string{
		// 1. ALBUMS
		`CREATE TABLE IF NOT EXISTS albums (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			artist TEXT,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		// 2. SONGS (The central unit)
		`CREATE TABLE IF NOT EXISTS songs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			album_id INTEGER,
			title TEXT NOT NULL,
			bpm INTEGER DEFAULT 120,
			time_signature TEXT DEFAULT '4/4',
			key_signature TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (album_id) REFERENCES albums(id) ON DELETE SET NULL
		);`,

		// 3. INSTRUMENTS (Catalog)
		`CREATE TABLE IF NOT EXISTS instruments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL, -- 'String', 'Keys', 'Percussion'
			default_clef TEXT DEFAULT 'treble'
		);`,

		// 4. TRACKS (Where the music lives: AlphaTex)
		`CREATE TABLE IF NOT EXISTS tracks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			song_id INTEGER NOT NULL,
			instrument_id INTEGER,
			name TEXT,
			data_content TEXT, -- ALPHATEX CODE GOES HERE
			display_mode TEXT DEFAULT 'BOTH',
			is_muted BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
			FOREIGN KEY (instrument_id) REFERENCES instruments(id)
		);`,

		// 5. AUDIO VERSIONS (Exported MP3s/WAVs)
		`CREATE TABLE IF NOT EXISTS audio_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			song_id INTEGER NOT NULL,
			version_name TEXT,
			file_path TEXT NOT NULL,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("error executing schema query: %w", err)
		}
	}
	return nil
}

func seedInstruments(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM instruments").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		log.Println("Seeding initial instruments...")
		instruments := []string{
			"INSERT INTO instruments (id, name, type, default_clef) VALUES (1, 'Electric Guitar', 'String', 'treble')",
			"INSERT INTO instruments (name, type, default_clef) VALUES ('Bass', 'String', 'bass')",
			"INSERT INTO instruments (name, type, default_clef) VALUES ('Piano', 'Keys', 'treble')",
			"INSERT INTO instruments (name, type, default_clef) VALUES ('Drums', 'Percussion', 'percussion')",
			"INSERT INTO instruments (name, type, default_clef) VALUES ('Vocals', 'Voice', 'treble')",
		}
		for _, ins := range instruments {
			if _, err := db.Exec(ins); err != nil {
				return err
			}
		}
	}
	return nil
}
