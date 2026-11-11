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

package bindings

import (
	"context"
	"fmt"
	"kizuna/client/internal/service"
	"kizuna/shared/models"
	"log"
)

// AlbumHandler exposes album management operations to the frontend via Wails (Issues #6, #7, #8, #17, #18).
type AlbumHandler struct {
	albumService *service.AlbumService
}

// NewAlbumHandler instantiates an AlbumHandler.
func NewAlbumHandler(albumService *service.AlbumService) *AlbumHandler {
	return &AlbumHandler{
		albumService: albumService,
	}
}

// CreateAlbum creates a new empty album (Issue #6).
func (h *AlbumHandler) CreateAlbum(title, artist, description string) (*models.Album, error) {
	log.Printf("[AlbumHandler] Creating album '%s' by '%s'", title, artist)
	return h.albumService.CreateAlbum(context.Background(), title, artist, description)
}

// GetAlbum retrieves an album by its ID (Issue #8).
func (h *AlbumHandler) GetAlbum(id int) (*models.Album, error) {
	log.Printf("[AlbumHandler] Fetching album ID %d", id)
	return h.albumService.GetAlbum(context.Background(), id)
}

// ListAlbums lists all albums in the library (Issue #8).
func (h *AlbumHandler) ListAlbums() ([]models.Album, error) {
	return h.albumService.ListAlbums(context.Background())
}

// UpdateAlbum updates an album's metadata (Issue #18).
func (h *AlbumHandler) UpdateAlbum(album *models.Album) error {
	if album == nil {
		return fmt.Errorf("album cannot be nil")
	}
	log.Printf("[AlbumHandler] Updating album ID %d ('%s')", album.ID, album.Title)
	return h.albumService.UpdateAlbum(context.Background(), album)
}

// DeleteAlbum deletes an album (linked songs will remain with album_id = NULL).
func (h *AlbumHandler) DeleteAlbum(id int) error {
	log.Printf("[AlbumHandler] Deleting album ID %d", id)
	return h.albumService.DeleteAlbum(context.Background(), id)
}

// AssignSongToAlbum assigns a song to an album (Issue #7).
func (h *AlbumHandler) AssignSongToAlbum(songID int, albumID int) error {
	log.Printf("[AlbumHandler] Assigning song ID %d to album ID %d", songID, albumID)
	return h.albumService.AssignSongToAlbum(context.Background(), songID, albumID)
}

// RemoveSongFromAlbum removes a song from an album (Issue #17).
func (h *AlbumHandler) RemoveSongFromAlbum(songID int) error {
	log.Printf("[AlbumHandler] Removing song ID %d from album", songID)
	return h.albumService.RemoveSongFromAlbum(context.Background(), songID)
}

// GetAlbumSongs retrieves all songs in a given album.
func (h *AlbumHandler) GetAlbumSongs(albumID int) ([]models.Song, error) {
	return h.albumService.GetAlbumSongs(context.Background(), albumID)
}
