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

// SongRepository defines data persistence operations for Song entities.
type SongRepository interface {
	Create(ctx context.Context, tx *sql.Tx, song *models.Song) (int, error)
	GetByID(ctx context.Context, id int) (*models.Song, error)
	GetRecent(ctx context.Context, limit int) ([]models.Song, error)
	GetByAlbumID(ctx context.Context, albumID int) ([]models.Song, error)
	Update(ctx context.Context, tx *sql.Tx, song *models.Song) error
	UpdateTitle(ctx context.Context, tx *sql.Tx, id int, title string) error
	Delete(ctx context.Context, id int) error
	SetAlbum(ctx context.Context, songID int, albumID *int) error
}

// SQLiteSongRepository implements SongRepository for SQLite.
type SQLiteSongRepository struct {
	db *sql.DB
}

// NewSongRepository instantiates an SQLiteSongRepository.
func NewSongRepository(db *sql.DB) *SQLiteSongRepository {
	return &SQLiteSongRepository{db: db}
}

// Create inserts a new song into the database, optionally inside an existing transaction.
func (r *SQLiteSongRepository) Create(ctx context.Context, tx *sql.Tx, song *models.Song) (int, error) {
	if song == nil {
		return 0, fmt.Errorf("song cannot be nil")
	}

	query := `
		INSERT INTO songs (album_id, title, bpm, time_signature, key_signature, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	var res sql.Result
	var err error

	if tx != nil {
		res, err = tx.ExecContext(ctx, query, song.AlbumID, song.Title, song.BPM, song.TimeSignature, song.KeySignature)
	} else {
		res, err = r.db.ExecContext(ctx, query, song.AlbumID, song.Title, song.BPM, song.TimeSignature, song.KeySignature)
	}

	if err != nil {
		return 0, fmt.Errorf("insert song: %w", err)
	}

	newID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id (song): %w", err)
	}

	return int(newID), nil
}

// GetByID retrieves a single song by its ID.
func (r *SQLiteSongRepository) GetByID(ctx context.Context, id int) (*models.Song, error) {
	query := `
		SELECT id, album_id, title, bpm, time_signature, key_signature, created_at, updated_at
		FROM songs WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)

	var s models.Song
	var ts sql.NullString
	var ks sql.NullString

	err := row.Scan(
		&s.ID, &s.AlbumID, &s.Title, &s.BPM,
		&ts, &ks,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("song with id %d not found", id)
		}
		return nil, fmt.Errorf("query song by id: %w", err)
	}

	if ts.Valid {
		s.TimeSignature = ts.String
	} else {
		s.TimeSignature = "4/4"
	}
	if ks.Valid {
		s.KeySignature = ks.String
	} else {
		s.KeySignature = ""
	}

	return &s, nil
}

// GetRecent returns the most recently updated songs up to the requested limit.
func (r *SQLiteSongRepository) GetRecent(ctx context.Context, limit int) ([]models.Song, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `SELECT id, title, updated_at FROM songs ORDER BY updated_at DESC LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent songs: %w", err)
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

// GetByAlbumID returns all songs belonging to a given album.
func (r *SQLiteSongRepository) GetByAlbumID(ctx context.Context, albumID int) ([]models.Song, error) {
	query := `
		SELECT id, album_id, title, bpm, time_signature, key_signature, created_at, updated_at
		FROM songs WHERE album_id = ? ORDER BY title ASC`

	rows, err := r.db.QueryContext(ctx, query, albumID)
	if err != nil {
		return nil, fmt.Errorf("query songs by album id: %w", err)
	}
	defer rows.Close()

	var songs []models.Song
	for rows.Next() {
		var s models.Song
		var ts, ks sql.NullString
		if err := rows.Scan(&s.ID, &s.AlbumID, &s.Title, &s.BPM, &ts, &ks, &s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		if ts.Valid {
			s.TimeSignature = ts.String
		}
		if ks.Valid {
			s.KeySignature = ks.String
		}
		songs = append(songs, s)
	}
	return songs, nil
}

// Update updates an existing song's metadata, optionally inside an existing transaction.
func (r *SQLiteSongRepository) Update(ctx context.Context, tx *sql.Tx, song *models.Song) error {
	if song == nil {
		return fmt.Errorf("song cannot be nil")
	}

	query := `
		UPDATE songs SET
			album_id = ?,
			title = ?,
			bpm = ?,
			time_signature = ?,
			key_signature = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, song.AlbumID, song.Title, song.BPM, song.TimeSignature, song.KeySignature, song.ID)
	} else {
		_, err = r.db.ExecContext(ctx, query, song.AlbumID, song.Title, song.BPM, song.TimeSignature, song.KeySignature, song.ID)
	}

	if err != nil {
		return fmt.Errorf("update song %d: %w", song.ID, err)
	}
	return nil
}

// UpdateTitle updates only the title of a song, optionally inside an existing transaction.
func (r *SQLiteSongRepository) UpdateTitle(ctx context.Context, tx *sql.Tx, id int, title string) error {
	query := "UPDATE songs SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, title, id)
	} else {
		_, err = r.db.ExecContext(ctx, query, title, id)
	}
	if err != nil {
		return fmt.Errorf("update song title for %d: %w", id, err)
	}
	return nil
}

// Delete removes a song by its ID.
func (r *SQLiteSongRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM songs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete song %d: %w", id, err)
	}
	return nil
}

// SetAlbum assigns or removes a song from an album.
func (r *SQLiteSongRepository) SetAlbum(ctx context.Context, songID int, albumID *int) error {
	query := "UPDATE songs SET album_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := r.db.ExecContext(ctx, query, albumID, songID)
	if err != nil {
		return fmt.Errorf("set album for song %d: %w", songID, err)
	}
	return nil
}
