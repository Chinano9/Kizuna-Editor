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
	"errors"
	"fmt"
	"io"
	"kizuna/client/internal/audio"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// QuickIdeaConversionService moves a standalone idea into a newly created song.
type QuickIdeaConversionService struct {
	db            *sql.DB
	songRepo      repository.SongRepository
	trackRepo     repository.TrackRepository
	audioRepo     repository.AudioRepository
	quickIdeaRepo repository.QuickIdeaRepository
}

// NewQuickIdeaConversionService creates a QuickIdeaConversionService.
func NewQuickIdeaConversionService(
	db *sql.DB,
	songRepo repository.SongRepository,
	trackRepo repository.TrackRepository,
	audioRepo repository.AudioRepository,
	quickIdeaRepo repository.QuickIdeaRepository,
) *QuickIdeaConversionService {
	return &QuickIdeaConversionService{
		db:            db,
		songRepo:      songRepo,
		trackRepo:     trackRepo,
		audioRepo:     audioRepo,
		quickIdeaRepo: quickIdeaRepo,
	}
}

// ConvertToSong creates a song with the idea as its first audio track and removes it from the inbox.
func (s *QuickIdeaConversionService) ConvertToSong(ctx context.Context, ideaID int, title string) (songID int, resultErr error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return 0, fmt.Errorf("song title cannot be empty")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin quick idea conversion: %w", err)
	}

	var projectDir, destinationPath, sourcePath string
	sourceRemoved := false
	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback quick idea conversion: %w", rollbackErr))
		}
		if sourceRemoved {
			if restoreErr := copyFileExclusive(destinationPath, sourcePath); restoreErr != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("restore original quick idea audio: %w", restoreErr))
				return
			}
			log.Printf("[QuickIdeaConversion] Restored source audio to %q after failed conversion", sourcePath)
		}
		if projectDir != "" {
			if cleanupErr := os.RemoveAll(projectDir); cleanupErr != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("remove incomplete project directory %q: %w", projectDir, cleanupErr))
			}
		}
	}()

	idea, err := s.quickIdeaRepo.GetByID(ctx, tx, ideaID)
	if err != nil {
		return 0, err
	}
	sourcePath = idea.FilePath
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return 0, fmt.Errorf("stat quick idea audio %q: %w", sourcePath, err)
	}
	if !sourceInfo.Mode().IsRegular() {
		return 0, fmt.Errorf("quick idea audio %q is not a regular file", sourcePath)
	}

	song, err := models.NewSong(title, 120, "4/4")
	if err != nil {
		return 0, fmt.Errorf("create song model: %w", err)
	}
	songID, err = s.songRepo.Create(ctx, tx, song)
	if err != nil {
		return 0, fmt.Errorf("create song: %w", err)
	}

	destinationPath, err = audio.ResolveQuickIdeaProjectAudioPath(songID, ideaID)
	if err != nil {
		return 0, fmt.Errorf("resolve project audio path: %w", err)
	}
	projectDirCandidate := filepath.Dir(filepath.Dir(destinationPath))
	projectParent := filepath.Dir(projectDirCandidate)
	if err := os.MkdirAll(projectParent, 0o755); err != nil {
		return 0, fmt.Errorf("create projects directory: %w", err)
	}
	if err := os.Mkdir(projectDirCandidate, 0o755); err != nil {
		return 0, fmt.Errorf("create project directory: %w", err)
	}
	projectDir = projectDirCandidate
	if err := os.Mkdir(filepath.Dir(destinationPath), 0o755); err != nil {
		return 0, fmt.Errorf("create project audio directory: %w", err)
	}
	if err := copyFileExclusive(sourcePath, destinationPath); err != nil {
		return 0, fmt.Errorf("copy quick idea audio into project: %w", err)
	}
	log.Printf("[QuickIdeaConversion] Moving WAV from %s to %s", sourcePath, destinationPath)

	defaultTrack, err := models.NewTrack(songID, models.InstrumentElectricGuitarID, "Track 1", "\\title \"New Song\"\n.")
	if err != nil {
		return 0, fmt.Errorf("create default notation track: %w", err)
	}
	if _, err := s.trackRepo.Create(ctx, tx, defaultTrack); err != nil {
		return 0, fmt.Errorf("create default notation track: %w", err)
	}

	audioTrack, err := models.NewTrack(songID, models.InstrumentVocalsID, "Quick Idea Audio", models.AudioContentPrefix+destinationPath)
	if err != nil {
		return 0, fmt.Errorf("create first audio track model: %w", err)
	}
	audioTrack.DisplayMode = models.DisplayModeBoth
	audioTrackID, err := s.trackRepo.Create(ctx, tx, audioTrack)
	if err != nil {
		return 0, fmt.Errorf("create first audio track: %w", err)
	}

	audioTrackIDPointer := audioTrackID
	audioVersion, err := models.NewAudioVersion(songID, &audioTrackIDPointer, "Quick Idea", destinationPath, "Imported from Quick Ideas")
	if err != nil {
		return 0, fmt.Errorf("create imported audio version model: %w", err)
	}
	if _, err := s.audioRepo.CreateWithTx(ctx, tx, audioVersion); err != nil {
		return 0, fmt.Errorf("register imported audio version: %w", err)
	}

	if err := s.quickIdeaRepo.Delete(ctx, tx, ideaID); err != nil {
		return 0, err
	}

	if err := os.Remove(sourcePath); err != nil {
		return 0, fmt.Errorf("remove original quick idea audio %q: %w", sourcePath, err)
	}
	sourceRemoved = true
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit quick idea conversion: %w", err)
	}
	committed = true
	log.Printf("[QuickIdeaConversion] Converted idea %d to song %d (%q)", ideaID, songID, title)
	return songID, nil
}

func copyFileExclusive(sourcePath, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source %q: %w", sourcePath, err)
	}
	defer source.Close()

	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create destination %q: %w", destinationPath, err)
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		removeErr := os.Remove(destinationPath)
		return errors.Join(copyErr, closeErr, removeErr)
	}
	return nil
}
