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

package service

import (
	"context"
	"fmt"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
)

// AlbumService coordinates operations for Albums and project grouping (Issues #6, #7, #8, #17, #18).
type AlbumService struct {
	albumRepo repository.AlbumRepository
	songRepo  repository.SongRepository
}

// NewAlbumService instantiates an AlbumService.
func NewAlbumService(albumRepo repository.AlbumRepository, songRepo repository.SongRepository) *AlbumService {
	return &AlbumService{
		albumRepo: albumRepo,
		songRepo:  songRepo,
	}
}

// CreateAlbum creates a new empty album container (Issue #6).
func (s *AlbumService) CreateAlbum(ctx context.Context, title, artist, description string) (*models.Album, error) {
	album, err := models.NewAlbum(title, artist, description)
	if err != nil {
		return nil, fmt.Errorf("create album model: %w", err)
	}

	newID, err := s.albumRepo.Create(ctx, album)
	if err != nil {
		return nil, fmt.Errorf("persist album: %w", err)
	}
	album.ID = newID

	return album, nil
}

// GetAlbum retrieves an album by its ID (Issue #8).
func (s *AlbumService) GetAlbum(ctx context.Context, id int) (*models.Album, error) {
	return s.albumRepo.GetByID(ctx, id)
}

// ListAlbums lists all albums in the library (Issue #8).
func (s *AlbumService) ListAlbums(ctx context.Context) ([]models.Album, error) {
	return s.albumRepo.GetAll(ctx)
}

// UpdateAlbum updates an album's metadata (Issue #18).
func (s *AlbumService) UpdateAlbum(ctx context.Context, album *models.Album) error {
	if album == nil {
		return fmt.Errorf("album cannot be nil")
	}

	if err := album.Validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	return s.albumRepo.Update(ctx, album)
}

// DeleteAlbum deletes an album (linked songs will remain with album_id = NULL).
func (s *AlbumService) DeleteAlbum(ctx context.Context, id int) error {
	return s.albumRepo.Delete(ctx, id)
}

// AssignSongToAlbum links a song to a specified album (Issue #7).
func (s *AlbumService) AssignSongToAlbum(ctx context.Context, songID int, albumID int) error {
	// Verify that the album exists
	if _, err := s.albumRepo.GetByID(ctx, albumID); err != nil {
		return fmt.Errorf("target album %d does not exist: %w", albumID, err)
	}

	// Verify that the song exists
	if _, err := s.songRepo.GetByID(ctx, songID); err != nil {
		return fmt.Errorf("target song %d does not exist: %w", songID, err)
	}

	return s.songRepo.SetAlbum(ctx, songID, &albumID)
}

// RemoveSongFromAlbum removes a song from its current album (Issue #17).
func (s *AlbumService) RemoveSongFromAlbum(ctx context.Context, songID int) error {
	return s.songRepo.SetAlbum(ctx, songID, nil)
}

// GetAlbumSongs returns all songs associated with an album.
func (s *AlbumService) GetAlbumSongs(ctx context.Context, albumID int) ([]models.Song, error) {
	return s.songRepo.GetByAlbumID(ctx, albumID)
}
