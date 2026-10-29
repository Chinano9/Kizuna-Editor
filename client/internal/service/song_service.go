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
	"database/sql"
	"fmt"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"log"
	"strings"
)

// SongService coordinates business logic, data validation, and transaction boundaries for Songs.
type SongService struct {
	db        *sql.DB
	songRepo  repository.SongRepository
	trackRepo repository.TrackRepository
}

// NewSongService instantiates a SongService.
func NewSongService(db *sql.DB, songRepo repository.SongRepository, trackRepo repository.TrackRepository) *SongService {
	return &SongService{
		db:        db,
		songRepo:  songRepo,
		trackRepo: trackRepo,
	}
}

// SaveQuickIdea handles rapid capture of a song concept.
// If songID is 0, creates a new song with an initial Lead Guitar track.
// If songID > 0, updates the title and the Lead Guitar track content (creating a recovery track if missing).
func (s *SongService) SaveQuickIdea(ctx context.Context, songID int, title string, content string) (int64, error) {
	if strings.TrimSpace(title) == "" {
		title = "Untitled Idea"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var finalID int64

	if songID == 0 {
		newSong, err := models.NewSong(title, 120, "4/4")
		if err != nil {
			return 0, fmt.Errorf("create song model: %w", err)
		}

		newID, err := s.songRepo.Create(ctx, tx, newSong)
		if err != nil {
			return 0, fmt.Errorf("insert quick idea song: %w", err)
		}
		finalID = int64(newID)

		track, err := models.NewTrack(newID, models.InstrumentElectricGuitarID, "Lead Guitar", content)
		if err != nil {
			return 0, fmt.Errorf("create track model: %w", err)
		}

		if _, err := s.trackRepo.Create(ctx, tx, track); err != nil {
			return 0, fmt.Errorf("insert initial track: %w", err)
		}
	} else {
		finalID = int64(songID)

		if err := s.songRepo.UpdateTitle(ctx, tx, songID, title); err != nil {
			return 0, fmt.Errorf("update song title: %w", err)
		}

		rowsAffected, err := s.trackRepo.UpdateContentBySongAndInstrument(ctx, tx, songID, models.InstrumentElectricGuitarID, content)
		if err != nil {
			return 0, fmt.Errorf("update track content: %w", err)
		}

		if rowsAffected == 0 {
			log.Printf("⚠️ Track missing for Song %d. Creating recovery track...", songID)
			recTrack, err := models.NewTrack(songID, models.InstrumentElectricGuitarID, "Lead Guitar", content)
			if err != nil {
				return 0, fmt.Errorf("create recovery track model: %w", err)
			}
			if _, err := s.trackRepo.Create(ctx, tx, recTrack); err != nil {
				return 0, fmt.Errorf("create recovery track: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return finalID, nil
}

// GetSong retrieves a song and populates its associated tracks.
func (s *SongService) GetSong(ctx context.Context, id int) (*models.Song, error) {
	song, err := s.songRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tracks, err := s.trackRepo.GetBySongID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get tracks for song %d: %w", id, err)
	}

	song.Tracks = tracks
	return song, nil
}

// GetRecentSongs returns a list of recent songs.
func (s *SongService) GetRecentSongs(ctx context.Context, limit int) ([]models.Song, error) {
	return s.songRepo.GetRecent(ctx, limit)
}

// SaveSong executes an atomic upsert of a full Song entity and its nested tracks collection.
func (s *SongService) SaveSong(ctx context.Context, song *models.Song) (*models.Song, error) {
	if song == nil {
		return nil, fmt.Errorf("song cannot be nil")
	}

	if err := song.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Upsert Song header
	if song.ID == 0 {
		newID, err := s.songRepo.Create(ctx, tx, song)
		if err != nil {
			return nil, fmt.Errorf("insert song: %w", err)
		}
		song.ID = newID
	} else {
		if err := s.songRepo.Update(ctx, tx, song); err != nil {
			return nil, fmt.Errorf("update song: %w", err)
		}
	}

	// 2. Determine tracks to keep vs delete
	var incomingTrackIDs []int
	for _, track := range song.Tracks {
		if track.ID != 0 {
			incomingTrackIDs = append(incomingTrackIDs, track.ID)
		}
	}

	if err := s.trackRepo.DeleteBySongIDExcept(ctx, tx, song.ID, incomingTrackIDs); err != nil {
		return nil, fmt.Errorf("clean up deleted tracks: %w", err)
	}

	// 3. Upsert incoming tracks
	for i := range song.Tracks {
		track := &song.Tracks[i]
		track.SongID = song.ID

		if track.ID == 0 {
			newTrackID, err := s.trackRepo.Create(ctx, tx, track)
			if err != nil {
				return nil, fmt.Errorf("insert track: %w", err)
			}
			track.ID = newTrackID
		} else {
			if err := s.trackRepo.Update(ctx, tx, track); err != nil {
				return nil, fmt.Errorf("update track %d: %w", track.ID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return song, nil
}

// DeleteSong removes a song and cascades to tracks and audio takes.
func (s *SongService) DeleteSong(ctx context.Context, id int) error {
	return s.songRepo.Delete(ctx, id)
}
