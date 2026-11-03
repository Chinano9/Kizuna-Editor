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

// SongHandler exposes Song domain operations to the frontend via Wails.
type SongHandler struct {
	songService *service.SongService
}

// NewSongHandler instantiates a SongHandler.
func NewSongHandler(songService *service.SongService) *SongHandler {
	return &SongHandler{
		songService: songService,
	}
}

// SaveQuickIdea persists the current editor content rapidly to the database.
func (h *SongHandler) SaveQuickIdea(id int, title string, content string) (int64, error) {
	if title == "" {
		title = "Untitled Idea"
	}
	log.Printf("[SongHandler] Saving song idea: '%s' (ID: %d)", title, id)
	return h.songService.SaveQuickIdea(context.Background(), id, title, content)
}

// GetSong retrieves a song and its associated tracks by ID.
func (h *SongHandler) GetSong(id int) (*models.Song, error) {
	log.Printf("[SongHandler] Fetching song ID %d", id)
	return h.songService.GetSong(context.Background(), id)
}

// SaveSong saves the complete song object, including its tracks collection.
func (h *SongHandler) SaveSong(song *models.Song) (*models.Song, error) {
	if song == nil {
		return nil, fmt.Errorf("cannot save a nil song")
	}
	log.Printf("[SongHandler] Saving full song object '%s' (ID: %d)", song.Title, song.ID)
	return h.songService.SaveSong(context.Background(), song)
}

// GetRecentSongs returns a list of recent songs for the dashboard.
func (h *SongHandler) GetRecentSongs() ([]models.Song, error) {
	return h.songService.GetRecentSongs(context.Background(), 10)
}

// DeleteSong removes a song by ID.
func (h *SongHandler) DeleteSong(id int) error {
	log.Printf("[SongHandler] Deleting song ID %d", id)
	return h.songService.DeleteSong(context.Background(), id)
}
