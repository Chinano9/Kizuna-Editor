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
	"fmt"
	"kizuna/shared/models"
)

// AudioRepository defines persistence operations for recorded AudioVersion takes.
type AudioRepository interface {
	Create(ctx context.Context, av *models.AudioVersion) (int, error)
	GetByID(ctx context.Context, id int) (*models.AudioVersion, error)
	GetBySongID(ctx context.Context, songID int) ([]models.AudioVersion, error)
	GetByTrackID(ctx context.Context, trackID int) ([]models.AudioVersion, error)
	Delete(ctx context.Context, id int) error
}

// SQLiteAudioRepository implements AudioRepository for SQLite.
type SQLiteAudioRepository struct {
	db *sql.DB
}

// NewAudioRepository instantiates an SQLiteAudioRepository.
func NewAudioRepository(db *sql.DB) *SQLiteAudioRepository {
	return &SQLiteAudioRepository{db: db}
}

// Create inserts a new audio version record into the database.
func (r *SQLiteAudioRepository) Create(ctx context.Context, av *models.AudioVersion) (int, error) {
	if av == nil {
		return 0, fmt.Errorf("audio version cannot be nil")
	}

	query := `
		INSERT INTO audio_versions (song_id, track_id, version_name, file_path, notes, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	res, err := r.db.ExecContext(ctx, query, av.SongID, av.TrackID, av.VersionName, av.FilePath, av.Notes)
	if err != nil {
		return 0, fmt.Errorf("insert audio version: %w", err)
	}

	newID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id (audio version): %w", err)
	}

	return int(newID), nil
}

// GetByID retrieves an audio version by its ID.
func (r *SQLiteAudioRepository) GetByID(ctx context.Context, id int) (*models.AudioVersion, error) {
	query := `
		SELECT id, song_id, track_id, version_name, file_path, notes, created_at
		FROM audio_versions WHERE id = ?`

	var av models.AudioVersion
	var trackIDNull sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&av.ID, &av.SongID, &trackIDNull, &av.VersionName, &av.FilePath, &av.Notes, &av.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("audio version %d not found", id)
		}
		return nil, fmt.Errorf("query audio version: %w", err)
	}

	if trackIDNull.Valid {
		val := int(trackIDNull.Int64)
		av.TrackID = &val
	} else {
		av.TrackID = nil
	}

	return &av, nil
}

// GetBySongID retrieves all takes for a song, including global takes and track takes.
func (r *SQLiteAudioRepository) GetBySongID(ctx context.Context, songID int) ([]models.AudioVersion, error) {
	query := `
		SELECT id, song_id, track_id, version_name, file_path, notes, created_at
		FROM audio_versions WHERE song_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, songID)
	if err != nil {
		return nil, fmt.Errorf("query audio versions for song: %w", err)
	}
	defer rows.Close()

	var versions []models.AudioVersion
	for rows.Next() {
		var av models.AudioVersion
		var trackIDNull sql.NullInt64
		err := rows.Scan(&av.ID, &av.SongID, &trackIDNull, &av.VersionName, &av.FilePath, &av.Notes, &av.CreatedAt)
		if err != nil {
			continue
		}
		if trackIDNull.Valid {
			val := int(trackIDNull.Int64)
			av.TrackID = &val
		} else {
			av.TrackID = nil
		}
		versions = append(versions, av)
	}
	return versions, nil
}

// GetByTrackID retrieves all audio takes associated specifically with a track.
func (r *SQLiteAudioRepository) GetByTrackID(ctx context.Context, trackID int) ([]models.AudioVersion, error) {
	query := `
		SELECT id, song_id, track_id, version_name, file_path, notes, created_at
		FROM audio_versions WHERE track_id = ? ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, trackID)
	if err != nil {
		return nil, fmt.Errorf("query audio versions for track: %w", err)
	}
	defer rows.Close()

	var versions []models.AudioVersion
	for rows.Next() {
		var av models.AudioVersion
		var trackIDNull sql.NullInt64
		err := rows.Scan(&av.ID, &av.SongID, &trackIDNull, &av.VersionName, &av.FilePath, &av.Notes, &av.CreatedAt)
		if err != nil {
			continue
		}
		if trackIDNull.Valid {
			val := int(trackIDNull.Int64)
			av.TrackID = &val
		} else {
			av.TrackID = nil
		}
		versions = append(versions, av)
	}
	return versions, nil
}

// Delete removes an audio version record by its ID.
func (r *SQLiteAudioRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM audio_versions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete audio version %d: %w", id, err)
	}
	return nil
}
