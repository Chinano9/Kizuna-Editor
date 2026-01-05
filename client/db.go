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
	"os"
	"path/filepath"
	"runtime"

	// Import shared models package (defines Song/Track structs used across client and server)
	"kizuna/shared/models"

	_ "modernc.org/sqlite"
)

const (
	InstrumentGuitarID = 1
	defaultDBName      = "kizuna.db"
	appFolderName      = "Kizuna"
)

// DBManager handles all direct database interactions.
type DBManager struct {
	db *sql.DB
}

// NewDBManager initializes the SQLite connection and ensures the schema exists.
// The database file is created inside the user's OS-appropriate data directory
// (e.g. %APPDATA%/Kizuna on Windows, ~/Library/Application Support/Kizuna on macOS,
// or XDG data dir (typically ~/.local/share) on Linux). If the path cannot be
// resolved or created, it falls back to a local file in the current working directory.
func NewDBManager() *DBManager {
	dbPath, err := getDefaultDBPath()
	if err != nil {
		log.Println("Warning: couldn't determine default DB path:", err)
		log.Println("Falling back to local file:", defaultDBName)
		dbPath = defaultDBName
	} else {
		// Ensure the parent directory exists
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			log.Println("Warning: couldn't create DB directory:", err)
			log.Println("Falling back to local file:", defaultDBName)
			dbPath = defaultDBName
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Initialize Schema
	if err := createFullSchema(db); err != nil {
		db.Close()
		log.Fatal("Failed to create schema:", err)
	}

	// Seed Initial Data (non-fatal)
	if err := seedInstruments(db); err != nil {
		log.Println("Warning: Failed to seed instruments:", err)
	}

	return &DBManager{db: db}
}

// Close safely closes the underlying database connection.
func (m *DBManager) Close() error {
	if m == nil || m.db == nil {
		return nil
	}
	return m.db.Close()
}

// SaveQuickIdea handles the "Upsert" logic for the editor.
// It uses transactions to ensure data integrity between Songs and Tracks.
func (m *DBManager) SaveQuickIdea(songID int, title string, content string) (int64, error) {
	if m == nil || m.db == nil {
		return 0, fmt.Errorf("database is not initialized")
	}

	tx, err := m.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		// If an error occurred and tx is still active, ensure rollback.
		_ = tx.Rollback()
	}()

	var finalID int64

	if songID == 0 {
		res, err := tx.Exec("INSERT INTO songs (title, bpm) VALUES (?, ?)", title, 120)
		if err != nil {
			return 0, fmt.Errorf("insert song: %w", err)
		}
		finalID, err = res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("last insert id (song): %w", err)
		}

		_, err = tx.Exec(`
			INSERT INTO tracks (song_id, instrument_id, name, data_content)
			VALUES (?, ?, ?, ?)`,
			finalID, InstrumentGuitarID, "Lead Guitar", content)
		if err != nil {
			return 0, fmt.Errorf("insert initial track: %w", err)
		}
	} else {
		finalID = int64(songID)

		_, err := tx.Exec("UPDATE songs SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", title, songID)
		if err != nil {
			return 0, fmt.Errorf("update song title: %w", err)
		}

		res, err := tx.Exec("UPDATE tracks SET data_content = ? WHERE song_id = ? AND instrument_id = ?", content, songID, InstrumentGuitarID)
		if err != nil {
			return 0, fmt.Errorf("update track content: %w", err)
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("rows affected check: %w", err)
		}
		if rowsAffected == 0 {
			log.Printf("⚠️ Track missing for Song %d. Creating recovery track...", songID)
			_, err = tx.Exec(`
				INSERT INTO tracks (song_id, instrument_id, name, data_content)
				VALUES (?, ?, ?, ?)`,
				songID, InstrumentGuitarID, "Lead Guitar", content)
			if err != nil {
				return 0, fmt.Errorf("create recovery track: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return finalID, nil
}

// GetSong retrieves a song and its associated tracks using the shared models.
func (m *DBManager) GetSong(id int) (*models.Song, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	var s models.Song

	querySong := `
		SELECT id, album_id, title, bpm, time_signature, key_signature, created_at, updated_at
		FROM songs WHERE id = ?`

	row := m.db.QueryRow(querySong, id)

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

	queryTracks := `
		SELECT id, song_id, instrument_id, name, data_content, display_mode, is_muted, created_at
		FROM tracks WHERE song_id = ?`

	rows, err := m.db.Query(queryTracks, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.SongID, &t.InstrumentID, &t.Name, &t.DataContent, &t.DisplayMode, &t.IsMuted, &t.CreatedAt); err != nil {
			// skip malformed row but continue
			continue
		}
		s.Tracks = append(s.Tracks, t)
	}

	return &s, nil
}

// GetRecentSongs returns a lightweight list of recent songs for the dashboard.
func (m *DBManager) GetRecentSongs() ([]models.Song, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	query := `SELECT id, title, updated_at FROM songs ORDER BY updated_at DESC LIMIT 10`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []models.Song
	for rows.Next() {
		var s models.Song
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
		// If the table doesn't exist yet, return nil so caller can decide.
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

// getDefaultDBPath returns a sensible default path for the database depending on OS.
func getDefaultDBPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appdata, appFolderName, defaultDBName), nil
	case "darwin":
		// macOS: ~/Library/Application Support
		configDir, err := os.UserConfigDir()
		if err != nil {
			home, herr := os.UserHomeDir()
			if herr != nil {
				return "", err
			}
			configDir = filepath.Join(home, "Library", "Application Support")
		}
		return filepath.Join(configDir, appFolderName, defaultDBName), nil
	default:
		// Linux / other: prefer XDG_DATA_HOME, otherwise ~/.local/share
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg != "" {
			return filepath.Join(xdg, appFolderName, defaultDBName), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", appFolderName, defaultDBName), nil
	}
}
