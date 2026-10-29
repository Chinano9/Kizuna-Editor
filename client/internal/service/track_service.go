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

// TrackService coordinates track-level operations and instrument associations.
type TrackService struct {
	trackRepo repository.TrackRepository
}

// NewTrackService instantiates a TrackService.
func NewTrackService(trackRepo repository.TrackRepository) *TrackService {
	return &TrackService{
		trackRepo: trackRepo,
	}
}

// AddTrack creates a new standard notation track for a song with default values.
func (s *TrackService) AddTrack(ctx context.Context, songID int, trackName string) (*models.Track, error) {
	if trackName == "" {
		trackName = "New Track"
	}

	track, err := models.NewTrack(songID, models.InstrumentElectricGuitarID, trackName, `\title "New Track"`)
	if err != nil {
		return nil, fmt.Errorf("create track model: %w", err)
	}

	newID, err := s.trackRepo.Create(ctx, nil, track)
	if err != nil {
		return nil, fmt.Errorf("insert new track: %w", err)
	}

	return s.trackRepo.GetByID(ctx, newID)
}

// AddAudioTrack creates a new audio track linked to a recorded WAV file path.
func (s *TrackService) AddAudioTrack(ctx context.Context, songID int, trackName string, filePath string) (*models.Track, error) {
	if trackName == "" {
		trackName = "Audio Track"
	}
	audioContent := models.AudioContentPrefix + filePath

	track, err := models.NewTrack(songID, models.InstrumentVocalsID, trackName, audioContent)
	if err != nil {
		return nil, fmt.Errorf("create audio track model: %w", err)
	}
	track.DisplayMode = models.DisplayModeBoth

	newID, err := s.trackRepo.Create(ctx, nil, track)
	if err != nil {
		return nil, fmt.Errorf("insert audio track: %w", err)
	}

	return s.trackRepo.GetByID(ctx, newID)
}

// UpdateTrack validates and persists modifications to a track.
func (s *TrackService) UpdateTrack(ctx context.Context, track *models.Track) error {
	if track == nil {
		return fmt.Errorf("track cannot be nil")
	}

	if err := track.Validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	return s.trackRepo.Update(ctx, nil, track)
}

// DeleteTrack removes a track by its ID.
func (s *TrackService) DeleteTrack(ctx context.Context, trackID int) error {
	return s.trackRepo.Delete(ctx, trackID)
}

// GetTracksForSong retrieves all tracks belonging to a song.
func (s *TrackService) GetTracksForSong(ctx context.Context, songID int) ([]models.Track, error) {
	return s.trackRepo.GetBySongID(ctx, songID)
}
