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

// AlbumRepository defines data persistence operations for Album entities.
type AlbumRepository interface {
	Create(ctx context.Context, album *models.Album) (int, error)
	GetByID(ctx context.Context, id int) (*models.Album, error)
	GetAll(ctx context.Context) ([]models.Album, error)
	Update(ctx context.Context, album *models.Album) error
	Delete(ctx context.Context, id int) error
}

// SQLiteAlbumRepository implements AlbumRepository for SQLite.
type SQLiteAlbumRepository struct {
	db *sql.DB
}

// NewAlbumRepository instantiates an SQLiteAlbumRepository.
func NewAlbumRepository(db *sql.DB) *SQLiteAlbumRepository {
	return &SQLiteAlbumRepository{db: db}
}

// Create inserts a new album into the database.
func (r *SQLiteAlbumRepository) Create(ctx context.Context, album *models.Album) (int, error) {
	if album == nil {
		return 0, fmt.Errorf("album cannot be nil")
	}

	query := `
		INSERT INTO albums (title, artist, description, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`

	res, err := r.db.ExecContext(ctx, query, album.Title, album.Artist, album.Description)
	if err != nil {
		return 0, fmt.Errorf("insert album: %w", err)
	}

	newID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id (album): %w", err)
	}

	return int(newID), nil
}

// GetByID retrieves an album by its primary key.
func (r *SQLiteAlbumRepository) GetByID(ctx context.Context, id int) (*models.Album, error) {
	query := `SELECT id, title, artist, description, created_at FROM albums WHERE id = ?`

	var a models.Album
	var artist, desc sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.Title, &artist, &desc, &a.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("album with id %d not found", id)
		}
		return nil, fmt.Errorf("query album by id: %w", err)
	}

	if artist.Valid {
		a.Artist = artist.String
	}
	if desc.Valid {
		a.Description = desc.String
	}

	return &a, nil
}

// GetAll retrieves all albums ordered by creation date descending.
func (r *SQLiteAlbumRepository) GetAll(ctx context.Context) ([]models.Album, error) {
	query := `SELECT id, title, artist, description, created_at FROM albums ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all albums: %w", err)
	}
	defer rows.Close()

	var albums []models.Album
	for rows.Next() {
		var a models.Album
		var artist, desc sql.NullString
		if err := rows.Scan(&a.ID, &a.Title, &artist, &desc, &a.CreatedAt); err != nil {
			continue
		}
		if artist.Valid {
			a.Artist = artist.String
		}
		if desc.Valid {
			a.Description = desc.String
		}
		albums = append(albums, a)
	}
	return albums, nil
}

// Update updates an album's title, artist, and description.
func (r *SQLiteAlbumRepository) Update(ctx context.Context, album *models.Album) error {
	if album == nil {
		return fmt.Errorf("album cannot be nil")
	}

	query := `UPDATE albums SET title = ?, artist = ?, description = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, album.Title, album.Artist, album.Description, album.ID)
	if err != nil {
		return fmt.Errorf("update album %d: %w", album.ID, err)
	}
	return nil
}

// Delete removes an album by its ID (songs linked will have album_id set to NULL due to SET NULL fk).
func (r *SQLiteAlbumRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM albums WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete album %d: %w", id, err)
	}
	return nil
}
