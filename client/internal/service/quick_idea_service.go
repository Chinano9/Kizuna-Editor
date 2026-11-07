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

// QuickIdeaService coordinates registration of independent recorded audio ideas.
type QuickIdeaService struct {
	quickIdeaRepo repository.QuickIdeaRepository
}

// NewQuickIdeaService instantiates a QuickIdeaService.
func NewQuickIdeaService(quickIdeaRepo repository.QuickIdeaRepository) *QuickIdeaService {
	return &QuickIdeaService{quickIdeaRepo: quickIdeaRepo}
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
