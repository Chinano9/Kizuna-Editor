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
	"errors"
	"fmt"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"log"
	"os"
	"strings"
)

// TrackService coordinates track-level operations, instrument associations, and disk cleanup.
type TrackService struct {
	trackRepo repository.TrackRepository
	audioRepo repository.AudioRepository
}

// NewTrackService instantiates a TrackService.
func NewTrackService(trackRepo repository.TrackRepository, audioRepo ...repository.AudioRepository) *TrackService {
	var ar repository.AudioRepository
	if len(audioRepo) > 0 {
		ar = audioRepo[0]
	}
	return &TrackService{
		trackRepo: trackRepo,
		audioRepo: ar,
	}
}

// AddTrack creates a new standard notation track for a song with default values.
func (s *TrackService) AddTrack(ctx context.Context, songID int, trackName string) (*models.Track, error) {
	trackName = strings.TrimSpace(trackName)
	if trackName == "" {
		trackName = "New Track"
	}

	track, err := models.NewTrack(songID, models.InstrumentElectricGuitarID, trackName, "\\title \"New Song\"\n.")
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
	trackName = strings.TrimSpace(trackName)
	if trackName == "" {
		trackName = "Audio Track"
	}
	audioContent := models.AudioContentPrefix + strings.TrimSpace(filePath)

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

// UpdateTrack validates and persists modifications to a track (including individual renaming and instrument changes).
func (s *TrackService) UpdateTrack(ctx context.Context, track *models.Track) error {
	if track == nil {
		return fmt.Errorf("track cannot be nil")
	}

	track.Name = strings.TrimSpace(track.Name)
	if err := track.Validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	return s.trackRepo.Update(ctx, nil, track)
}

// DeleteTrack removes a track by its ID in SQLite and guarantees physical deletion of associated .wav files on disk.
func (s *TrackService) DeleteTrack(ctx context.Context, trackID int) error {
	if trackID <= 0 {
		return fmt.Errorf("invalid track ID: %d", trackID)
	}

	// Collect any associated .wav file paths before the SQLite cascade delete executes.
	seenPaths := make(map[string]struct{})
	var wavPaths []string
	addPath := func(p string) {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			return
		}
		if _, ok := seenPaths[trimmed]; !ok {
			seenPaths[trimmed] = struct{}{}
			wavPaths = append(wavPaths, trimmed)
		}
	}

	if existingTrack, err := s.trackRepo.GetByID(ctx, trackID); err == nil && existingTrack != nil {
		if audioPath, ok := existingTrack.GetAudioPath(); ok {
			addPath(audioPath)
		}
	}

	if s.audioRepo != nil {
		if versions, err := s.audioRepo.GetByTrackID(ctx, trackID); err == nil {
			for _, v := range versions {
				addPath(v.FilePath)
			}
		}
	}

	if err := s.trackRepo.Delete(ctx, trackID); err != nil {
		return err
	}

	// Ensure physical .wav files are removed if SQLiteTrackRepository did not already remove them
	for _, p := range wavPaths {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove track audio file %q: %w", p, err)
		} else if err == nil {
			log.Printf("[TrackService] Deleted physical WAV file: %s", p)
		}
	}

	return nil
}

// GetTracksForSong retrieves all tracks belonging to a song.
func (s *TrackService) GetTracksForSong(ctx context.Context, songID int) ([]models.Track, error) {
	return s.trackRepo.GetBySongID(ctx, songID)
}
