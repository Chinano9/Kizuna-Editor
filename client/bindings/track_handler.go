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
	"kizuna/client/internal/repository"
	"kizuna/client/internal/service"
	"kizuna/shared/models"
	"log"
)

// TrackHandler exposes track management operations to the frontend via Wails.
type TrackHandler struct {
	trackService *service.TrackService
	instRepo     repository.InstrumentRepository
}

// NewTrackHandler instantiates a TrackHandler.
func NewTrackHandler(trackService *service.TrackService, instRepo repository.InstrumentRepository) *TrackHandler {
	return &TrackHandler{
		trackService: trackService,
		instRepo:     instRepo,
	}
}

// AddTrack creates a new notation track for a song with default values.
func (h *TrackHandler) AddTrack(songID int, trackName string) (*models.Track, error) {
	log.Printf("[TrackHandler] Adding track '%s' to song ID %d", trackName, songID)
	return h.trackService.AddTrack(context.Background(), songID, trackName)
}

// AddAudioTrack creates a new audio track for a song linked to a recorded WAV file path.
func (h *TrackHandler) AddAudioTrack(songID int, trackName string, filePath string) (*models.Track, error) {
	log.Printf("[TrackHandler] Adding audio track '%s' (file: %s) to song ID %d", trackName, filePath, songID)
	return h.trackService.AddAudioTrack(context.Background(), songID, trackName, filePath)
}

// UpdateTrack updates track details (name, data_content, instrument_id, display_mode, is_muted).
func (h *TrackHandler) UpdateTrack(track *models.Track) error {
	if track == nil {
		return fmt.Errorf("track cannot be nil")
	}
	log.Printf("[TrackHandler] Updating track ID %d ('%s')", track.ID, track.Name)
	return h.trackService.UpdateTrack(context.Background(), track)
}

// DeleteTrack removes a track by ID.
func (h *TrackHandler) DeleteTrack(trackID int) error {
	log.Printf("[TrackHandler] Deleting track ID %d", trackID)
	return h.trackService.DeleteTrack(context.Background(), trackID)
}

// GetInstruments retrieves all available instruments from the catalog.
func (h *TrackHandler) GetInstruments() ([]models.Instrument, error) {
	return h.instRepo.GetAll(context.Background())
}
