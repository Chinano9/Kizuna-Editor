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

// AudioService coordinates business logic for recorded takes and versions.
type AudioService struct {
	audioRepo repository.AudioRepository
}

// NewAudioService instantiates an AudioService.
func NewAudioService(audioRepo repository.AudioRepository) *AudioService {
	return &AudioService{
		audioRepo: audioRepo,
	}
}

// RegisterAudioVersion validates and logs a new recorded take into the database.
func (s *AudioService) RegisterAudioVersion(ctx context.Context, songID int, trackID *int, versionName, filePath, notes string) (*models.AudioVersion, error) {
	av, err := models.NewAudioVersion(songID, trackID, versionName, filePath, notes)
	if err != nil {
		return nil, fmt.Errorf("create audio version model: %w", err)
	}

	newID, err := s.audioRepo.Create(ctx, av)
	if err != nil {
		return nil, fmt.Errorf("persist audio version: %w", err)
	}
	av.ID = newID

	return av, nil
}

// GetAudioVersionsForTrack returns all takes associated with a track.
func (s *AudioService) GetAudioVersionsForTrack(ctx context.Context, trackID int) ([]models.AudioVersion, error) {
	return s.audioRepo.GetByTrackID(ctx, trackID)
}

// GetAudioVersionsForSong returns all takes for a song (including global takes).
func (s *AudioService) GetAudioVersionsForSong(ctx context.Context, songID int) ([]models.AudioVersion, error) {
	return s.audioRepo.GetBySongID(ctx, songID)
}

// DeleteAudioVersion removes an audio take record.
func (s *AudioService) DeleteAudioVersion(ctx context.Context, id int) error {
	return s.audioRepo.Delete(ctx, id)
}
