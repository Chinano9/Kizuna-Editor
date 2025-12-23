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
package main

import (
	"database/sql"
	"fmt"
	"log"

	// Import shared models package (defines Song/Track structs used across client and server)
	"kizuna/shared/models"

	_ "modernc.org/sqlite"
)

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

		_, err := tx.Exec("UPDATE songs SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", title, songID)
		if err != nil {
			log.Println("Error updating song title:", err)
			return 0
		}

		res, err := tx.Exec("UPDATE tracks SET data_content = ? WHERE song_id = ? AND instrument_id = ?", content, songID, InstrumentGuitarID)
		if err != nil {
			log.Println("Error updating track content:", err)
			return 0
		}

		// Check if the track actually existed
		rowsAffected, _ := res.RowsAffected()
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

	if err := tx.Commit(); err != nil {
		log.Println("Error committing transaction:", err)
		return 0
	}

	return finalID
}

// --- Read helpers using shared models ---

// GetSong retrieves a song and its associated tracks using the shared models.
func (m *DBManager) GetSong(id int) (*models.Song, error) {
	var s models.Song

	// 1. Load song metadata from the songs table
	querySong := `
		SELECT id, album_id, title, bpm, time_signature, key_signature, created_at, updated_at
		FROM songs WHERE id = ?`

	row := m.db.QueryRow(querySong, id)

	// Scan the row into the models.Song struct
	var ts sql.NullString
	var ks sql.NullString

	err := row.Scan(
		&s.ID, &s.AlbumID, &s.Title, &s.BPM,
		&ts, &ks,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable string columns: convert sql.NullString to plain string.
	if ts.Valid {
		s.TimeSignature = ts.String
	} else {
		s.TimeSignature = ""
	}
	if ks.Valid {
		s.KeySignature = ks.String
	} else {
		s.KeySignature = ""
	}

	// 2. Load tracks associated with the song
	queryTracks := `
		SELECT id, song_id, instrument_id, name, data_content, is_muted
		FROM tracks WHERE song_id = ?`

	rows, err := m.db.Query(queryTracks, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t models.Track
		// Scan each row into a models.Track struct
		if err := rows.Scan(&t.ID, &t.SongID, &t.InstrumentID, &t.Name, &t.DataContent, &t.IsMuted); err != nil {
			continue
		}
		s.Tracks = append(s.Tracks, t)
	}

	return &s, nil
}

// GetRecentSongs returns a lightweight list of recent songs for the dashboard.
func (m *DBManager) GetRecentSongs() ([]models.Song, error) {
	query := `SELECT id, title, updated_at FROM songs ORDER BY updated_at DESC LIMIT 10`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []models.Song
	for rows.Next() {
		var s models.Song
		// Only populate fields needed for the dashboard list (id, title, updated_at)
		if err := rows.Scan(&s.ID, &s.Title, &s.UpdatedAt); err != nil {
			continue
		}
		songs = append(songs, s)
	}
	return songs, nil
}

// --- PRIVATE HELPERS ---

func createFullSchema(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS albums (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			artist TEXT,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
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
		`CREATE TABLE IF NOT EXISTS instruments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			default_clef TEXT DEFAULT 'treble'
		);`,
		`CREATE TABLE IF NOT EXISTS tracks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			song_id INTEGER NOT NULL,
			instrument_id INTEGER,
			name TEXT,
			data_content TEXT,
			display_mode TEXT DEFAULT 'BOTH',
			is_muted BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
			FOREIGN KEY (instrument_id) REFERENCES instruments(id)
		);`,
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
