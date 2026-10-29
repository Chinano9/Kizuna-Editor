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

// InstrumentRepository defines read operations for the instrument catalog.
type InstrumentRepository interface {
	GetAll(ctx context.Context) ([]models.Instrument, error)
	GetByID(ctx context.Context, id int) (*models.Instrument, error)
}

// SQLiteInstrumentRepository implements InstrumentRepository for SQLite.
type SQLiteInstrumentRepository struct {
	db *sql.DB
}

// NewInstrumentRepository instantiates an SQLiteInstrumentRepository.
func NewInstrumentRepository(db *sql.DB) *SQLiteInstrumentRepository {
	return &SQLiteInstrumentRepository{db: db}
}

// GetAll retrieves all catalog instruments ordered alphabetically by name.
func (r *SQLiteInstrumentRepository) GetAll(ctx context.Context) ([]models.Instrument, error) {
	query := `SELECT id, name, type, default_clef FROM instruments ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query instruments: %w", err)
	}
	defer rows.Close()

	var instruments []models.Instrument
	for rows.Next() {
		var i models.Instrument
		if err := rows.Scan(&i.ID, &i.Name, &i.Type, &i.DefaultClef); err != nil {
			continue
		}
		instruments = append(instruments, i)
	}
	return instruments, nil
}

// GetByID retrieves a single instrument by its ID.
func (r *SQLiteInstrumentRepository) GetByID(ctx context.Context, id int) (*models.Instrument, error) {
	query := `SELECT id, name, type, default_clef FROM instruments WHERE id = ?`

	var i models.Instrument
	err := r.db.QueryRowContext(ctx, query, id).Scan(&i.ID, &i.Name, &i.Type, &i.DefaultClef)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("instrument with id %d not found", id)
		}
		return nil, fmt.Errorf("query instrument by id: %w", err)
	}
	return &i, nil
}
