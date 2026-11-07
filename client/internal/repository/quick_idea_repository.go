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

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"kizuna/shared/models"
)

// QuickIdeaRepository defines persistence operations for independent audio ideas.
type QuickIdeaRepository interface {
	Create(ctx context.Context, idea *models.QuickIdea) (int, error)
	List(ctx context.Context) ([]models.QuickIdea, error)
}

// List retrieves all independent audio ideas, newest first.
func (r *SQLiteQuickIdeaRepository) List(ctx context.Context) ([]models.QuickIdea, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, file_path, created_at FROM quick_ideas ORDER BY created_at DESC, id DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("query quick ideas: %w", err)
	}
	defer rows.Close()

	ideas := make([]models.QuickIdea, 0)
	for rows.Next() {
		var idea models.QuickIdea
		if err := rows.Scan(&idea.ID, &idea.FilePath, &idea.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan quick idea: %w", err)
		}
		ideas = append(ideas, idea)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quick ideas: %w", err)
	}
	return ideas, nil
}

// SQLiteQuickIdeaRepository implements QuickIdeaRepository for SQLite.
type SQLiteQuickIdeaRepository struct {
	db *sql.DB
}

// NewQuickIdeaRepository instantiates an SQLiteQuickIdeaRepository.
func NewQuickIdeaRepository(db *sql.DB) *SQLiteQuickIdeaRepository {
	return &SQLiteQuickIdeaRepository{db: db}
}

// Create inserts a new independent audio idea.
func (r *SQLiteQuickIdeaRepository) Create(ctx context.Context, idea *models.QuickIdea) (int, error) {
	if idea == nil {
		return 0, fmt.Errorf("quick idea cannot be nil")
	}
	if err := idea.Validate(); err != nil {
		return 0, fmt.Errorf("validate quick idea: %w", err)
	}

	result, err := r.db.ExecContext(ctx,
		"INSERT INTO quick_ideas (file_path, created_at) VALUES (?, ?)",
		idea.FilePath, idea.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert quick idea: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id (quick idea): %w", err)
	}
	return int(id), nil
}
