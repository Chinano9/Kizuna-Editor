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
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"log"
	"os"
	"path/filepath"
)

// QuickIdeaService coordinates registration of independent recorded audio ideas.
type QuickIdeaService struct {
	db            *sql.DB
	quickIdeaRepo repository.QuickIdeaRepository
}

// NewQuickIdeaService instantiates a QuickIdeaService.
func NewQuickIdeaService(db *sql.DB, quickIdeaRepo repository.QuickIdeaRepository) *QuickIdeaService {
	return &QuickIdeaService{db: db, quickIdeaRepo: quickIdeaRepo}
}

// RegisterQuickIdea validates and persists a saved audio idea.
func (s *QuickIdeaService) RegisterQuickIdea(ctx context.Context, filePath string) (*models.QuickIdea, error) {
	idea, err := models.NewQuickIdea(filePath)
	if err != nil {
		return nil, fmt.Errorf("create quick idea model: %w", err)
	}

	id, err := s.quickIdeaRepo.Create(ctx, idea)
	if err != nil {
		return nil, fmt.Errorf("persist quick idea: %w", err)
	}
	idea.ID = id
	return idea, nil
}

// ListQuickIdeas returns all saved audio ideas, newest first.
func (s *QuickIdeaService) ListQuickIdeas(ctx context.Context) ([]models.QuickIdea, error) {
	ideas, err := s.quickIdeaRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list quick ideas: %w", err)
	}
	return ideas, nil
}

// DeleteQuickIdea removes a saved idea and its WAV file from disk.
func (s *QuickIdeaService) DeleteQuickIdea(ctx context.Context, id int) (resultErr error) {
	if id <= 0 {
		return fmt.Errorf("quick idea ID must be positive")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin quick idea deletion: %w", err)
	}
	committed := false
	var idea *models.QuickIdea
	var stagedPath string
	var sourceStaged bool
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				resultErr = errors.Join(resultErr, fmt.Errorf("rollback quick idea deletion: %w", rollbackErr))
			}
			if sourceStaged {
				if restoreErr := os.Rename(stagedPath, idea.FilePath); restoreErr != nil {
					resultErr = errors.Join(resultErr, fmt.Errorf("restore quick idea audio %q: %w", idea.FilePath, restoreErr))
				}
			}
		}
	}()

	idea, err = s.quickIdeaRepo.GetByID(ctx, tx, id)
	if err != nil {
		return err
	}
	stagedPath, sourceStaged, err = stageQuickIdeaForDeletion(idea.FilePath)
	if err != nil {
		return fmt.Errorf("stage quick idea audio %q for deletion: %w", idea.FilePath, err)
	}
	if sourceStaged {
		log.Printf("[QuickIdeaService] Staged WAV for deletion: %s", idea.FilePath)
	}

	if err := s.quickIdeaRepo.Delete(ctx, tx, id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit quick idea deletion: %w", err)
	}
	committed = true

	if sourceStaged {
		if err := os.Remove(stagedPath); err != nil {
			restoreFileErr := os.Rename(stagedPath, idea.FilePath)
			if restoreFileErr == nil {
				sourceStaged = false
			}
			restoreRecordErr := s.restoreQuickIdeaRecord(ctx, idea)
			return errors.Join(
				fmt.Errorf("delete staged quick idea audio %q: %w", idea.FilePath, err),
				wrapOptionalError("restore quick idea audio", restoreFileErr),
				restoreRecordErr,
			)
		}
		log.Printf("[QuickIdeaService] Deleted WAV: %s", idea.FilePath)
	}
	log.Printf("[QuickIdeaService] Deleted Quick Idea %d", id)
	return nil
}

func stageQuickIdeaForDeletion(path string) (string, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("audio path is not a regular file")
	}

	stagingFile, err := os.CreateTemp(filepath.Dir(path), ".quick-idea-delete-*")
	if err != nil {
		return "", false, fmt.Errorf("create staging path: %w", err)
	}
	stagedPath := stagingFile.Name()
	if err := stagingFile.Close(); err != nil {
		_ = os.Remove(stagedPath)
		return "", false, fmt.Errorf("close staging file: %w", err)
	}
	if err := os.Remove(stagedPath); err != nil {
		return "", false, fmt.Errorf("prepare staging path: %w", err)
	}
	if err := os.Rename(path, stagedPath); err != nil {
		return "", false, fmt.Errorf("move audio to staging path: %w", err)
	}
	return stagedPath, true, nil
}

func (s *QuickIdeaService) restoreQuickIdeaRecord(ctx context.Context, idea *models.QuickIdea) error {
	if _, err := s.quickIdeaRepo.Create(ctx, idea); err != nil {
		return fmt.Errorf("restore Quick Idea record after audio deletion failure: %w", err)
	}
	log.Printf("[QuickIdeaService] Restored Quick Idea record for %q after WAV deletion failure", idea.FilePath)
	return nil
}

func wrapOptionalError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
