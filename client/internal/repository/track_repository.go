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

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"kizuna/shared/models"
	"log"
	"os"
	"strings"
)

// TrackRepository defines data persistence operations for Track entities.
type TrackRepository interface {
	Create(ctx context.Context, tx *sql.Tx, track *models.Track) (int, error)
	GetByID(ctx context.Context, id int) (*models.Track, error)
	GetBySongID(ctx context.Context, songID int) ([]models.Track, error)
	Update(ctx context.Context, tx *sql.Tx, track *models.Track) error
	Delete(ctx context.Context, id int) error
	DeleteBySongIDExcept(ctx context.Context, tx *sql.Tx, songID int, keepIDs []int) error
	UpdateContentBySongAndInstrument(ctx context.Context, tx *sql.Tx, songID int, instrumentID int, content string) (int64, error)
}

// SQLiteTrackRepository implements TrackRepository for SQLite.
type SQLiteTrackRepository struct {
	db *sql.DB
}

// NewTrackRepository instantiates an SQLiteTrackRepository.
func NewTrackRepository(db *sql.DB) *SQLiteTrackRepository {
	return &SQLiteTrackRepository{db: db}
}

// Create inserts a new track into the database, optionally within a transaction.
func (r *SQLiteTrackRepository) Create(ctx context.Context, tx *sql.Tx, track *models.Track) (int, error) {
	if track == nil {
		return 0, fmt.Errorf("track cannot be nil")
	}

	mode := track.DisplayMode
	if mode == "" {
		mode = models.DisplayModeBoth
	}

	query := `
		INSERT INTO tracks (song_id, instrument_id, name, data_content, display_mode, is_muted, created_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	var res sql.Result
	var err error

	if tx != nil {
		res, err = tx.ExecContext(ctx, query, track.SongID, track.InstrumentID, track.Name, track.DataContent, mode, track.IsMuted)
	} else {
		res, err = r.db.ExecContext(ctx, query, track.SongID, track.InstrumentID, track.Name, track.DataContent, mode, track.IsMuted)
	}

	if err != nil {
		return 0, fmt.Errorf("insert track: %w", err)
	}

	newID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id (track): %w", err)
	}

	return int(newID), nil
}

// GetByID retrieves a track by its primary key.
func (r *SQLiteTrackRepository) GetByID(ctx context.Context, id int) (*models.Track, error) {
	query := `
		SELECT id, song_id, instrument_id, name, data_content, display_mode, is_muted, created_at
		FROM tracks WHERE id = ?`

	var t models.Track
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.SongID, &t.InstrumentID, &t.Name, &t.DataContent, &t.DisplayMode, &t.IsMuted, &t.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("track with id %d not found", id)
		}
		return nil, fmt.Errorf("query track by id: %w", err)
	}
	return &t, nil
}

// GetBySongID returns all tracks associated with a specific song.
func (r *SQLiteTrackRepository) GetBySongID(ctx context.Context, songID int) ([]models.Track, error) {
	query := `
		SELECT id, song_id, instrument_id, name, data_content, display_mode, is_muted, created_at
		FROM tracks WHERE song_id = ? ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query, songID)
	if err != nil {
		return nil, fmt.Errorf("query tracks by song id: %w", err)
	}
	defer rows.Close()

	tracks := make([]models.Track, 0)
	for rows.Next() {
		var t models.Track
		if err := rows.Scan(&t.ID, &t.SongID, &t.InstrumentID, &t.Name, &t.DataContent, &t.DisplayMode, &t.IsMuted, &t.CreatedAt); err != nil {
			continue
		}
		tracks = append(tracks, t)
	}
	return tracks, nil
}

// Update updates the track fields, optionally within a transaction.
func (r *SQLiteTrackRepository) Update(ctx context.Context, tx *sql.Tx, track *models.Track) error {
	if track == nil {
		return fmt.Errorf("track cannot be nil")
	}

	query := `
		UPDATE tracks SET
			name = ?,
			data_content = ?,
			instrument_id = ?,
			display_mode = ?,
			is_muted = ?
		WHERE id = ?`

	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, track.Name, track.DataContent, track.InstrumentID, track.DisplayMode, track.IsMuted, track.ID)
	} else {
		_, err = r.db.ExecContext(ctx, query, track.Name, track.DataContent, track.InstrumentID, track.DisplayMode, track.IsMuted, track.ID)
	}

	if err != nil {
		return fmt.Errorf("update track %d: %w", track.ID, err)
	}
	return nil
}

// Delete removes a track by its ID and deletes any associated physical .wav files on disk.
func (r *SQLiteTrackRepository) Delete(ctx context.Context, id int) error {
	audioPaths, err := r.collectTrackAudioFiles(ctx, nil, []int{id})
	if err != nil {
		return fmt.Errorf("collect audio files for track %d: %w", id, err)
	}

	_, err = r.db.ExecContext(ctx, "DELETE FROM tracks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete track %d: %w", id, err)
	}

	if err := r.removeUnreferencedAudioFiles(ctx, nil, audioPaths); err != nil {
		return err
	}
	return nil
}

// DeleteBySongIDExcept deletes tracks for a song whose IDs are not in keepIDs and removes their physical .wav files.
func (r *SQLiteTrackRepository) DeleteBySongIDExcept(ctx context.Context, tx *sql.Tx, songID int, keepIDs []int) error {
	var selectIDsQuery string
	var query string
	var args []interface{}

	if len(keepIDs) == 0 {
		selectIDsQuery = "SELECT id FROM tracks WHERE song_id = ?"
		query = "DELETE FROM tracks WHERE song_id = ?"
		args = append(args, songID)
	} else {
		placeholders := make([]string, len(keepIDs))
		for i, id := range keepIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		inClause := strings.Join(placeholders, ",")
		selectIDsQuery = fmt.Sprintf("SELECT id FROM tracks WHERE song_id = ? AND id NOT IN (%s)", inClause)
		query = fmt.Sprintf("DELETE FROM tracks WHERE song_id = ? AND id NOT IN (%s)", inClause)
		args = append([]interface{}{songID}, args...)
	}

	var rows *sql.Rows
	var err error
	if tx != nil {
		rows, err = tx.QueryContext(ctx, selectIDsQuery, args...)
	} else {
		rows, err = r.db.QueryContext(ctx, selectIDsQuery, args...)
	}
	if err != nil {
		return fmt.Errorf("query tracks to delete for song %d: %w", songID, err)
	}
	var trackIDsToDelete []int
	for rows.Next() {
		var tid int
		if scanErr := rows.Scan(&tid); scanErr == nil {
			trackIDsToDelete = append(trackIDsToDelete, tid)
		}
	}
	_ = rows.Close()

	audioPaths, err := r.collectTrackAudioFiles(ctx, tx, trackIDsToDelete)
	if err != nil {
		return fmt.Errorf("collect audio files for deleted tracks in song %d: %w", songID, err)
	}

	if tx != nil {
		_, err = tx.ExecContext(ctx, query, args...)
	} else {
		_, err = r.db.ExecContext(ctx, query, args...)
	}

	if err != nil {
		return fmt.Errorf("delete tracks except keepIDs for song %d: %w", songID, err)
	}

	if err := r.removeUnreferencedAudioFiles(ctx, tx, audioPaths); err != nil {
		return err
	}
	return nil
}

func (r *SQLiteTrackRepository) collectTrackAudioFiles(ctx context.Context, tx *sql.Tx, trackIDs []int) ([]string, error) {
	if len(trackIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(trackIDs))
	args := make([]interface{}, len(trackIDs))
	for i, id := range trackIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	seen := make(map[string]struct{})
	var paths []string
	addPath := func(raw string) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		if _, exists := seen[trimmed]; !exists {
			seen[trimmed] = struct{}{}
			paths = append(paths, trimmed)
		}
	}

	// 1. Check tracks.data_content for "audio:<filePath>"
	trackQuery := fmt.Sprintf("SELECT data_content FROM tracks WHERE id IN (%s)", inClause)
	var trackRows *sql.Rows
	var err error
	if tx != nil {
		trackRows, err = tx.QueryContext(ctx, trackQuery, args...)
	} else {
		trackRows, err = r.db.QueryContext(ctx, trackQuery, args...)
	}
	if err != nil {
		return nil, fmt.Errorf("query track data_content for audio cleanup: %w", err)
	}
	for trackRows.Next() {
		var content string
		if scanErr := trackRows.Scan(&content); scanErr == nil {
			if strings.HasPrefix(content, models.AudioContentPrefix) {
				addPath(strings.TrimPrefix(content, models.AudioContentPrefix))
			}
		}
	}
	_ = trackRows.Close()

	// 2. Check audio_versions.file_path for track_id IN (...)
	avQuery := fmt.Sprintf("SELECT file_path FROM audio_versions WHERE track_id IN (%s)", inClause)
	var avRows *sql.Rows
	if tx != nil {
		avRows, err = tx.QueryContext(ctx, avQuery, args...)
	} else {
		avRows, err = r.db.QueryContext(ctx, avQuery, args...)
	}
	if err != nil {
		return nil, fmt.Errorf("query audio_versions for track cleanup: %w", err)
	}
	for avRows.Next() {
		var filePath string
		if scanErr := avRows.Scan(&filePath); scanErr == nil {
			addPath(filePath)
		}
	}
	_ = avRows.Close()

	return paths, nil
}

func (r *SQLiteTrackRepository) removeUnreferencedAudioFiles(ctx context.Context, tx *sql.Tx, paths []string) error {
	for _, path := range paths {
		var refCount int
		countQuery := `
			SELECT
				(SELECT COUNT(1) FROM audio_versions WHERE file_path = ?) +
				(SELECT COUNT(1) FROM tracks WHERE data_content = ?)`
		audioContent := models.AudioContentPrefix + path

		var err error
		if tx != nil {
			err = tx.QueryRowContext(ctx, countQuery, path, audioContent).Scan(&refCount)
		} else {
			err = r.db.QueryRowContext(ctx, countQuery, path, audioContent).Scan(&refCount)
		}
		if err == nil && refCount > 0 {
			continue
		}

		if rmErr := os.Remove(path); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return fmt.Errorf("remove track audio file %q: %w", path, rmErr)
		} else if rmErr == nil {
			log.Printf("[TrackRepository] Deleted physical WAV file: %s", path)
		}
	}
	return nil
}

// UpdateContentBySongAndInstrument updates data_content for a song's specific instrument (used in quick idea).
func (r *SQLiteTrackRepository) UpdateContentBySongAndInstrument(ctx context.Context, tx *sql.Tx, songID int, instrumentID int, content string) (int64, error) {
	query := "UPDATE tracks SET data_content = ? WHERE song_id = ? AND instrument_id = ?"

	var res sql.Result
	var err error

	if tx != nil {
		res, err = tx.ExecContext(ctx, query, content, songID, instrumentID)
	} else {
		res, err = r.db.ExecContext(ctx, query, content, songID, instrumentID)
	}

	if err != nil {
		return 0, fmt.Errorf("update track content: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("check rows affected: %w", err)
	}

	return rowsAffected, nil
}
