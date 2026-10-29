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

package database

import (
	"database/sql"
	"fmt"
	"kizuna/shared/models"
	"log"
)

// DefaultInstrument defines a standard instrument for seeding the database catalog.
type DefaultInstrument struct {
	ID          int
	Name        string
	Type        string
	DefaultClef string
}

// InitialInstruments lists the standard instrument catalog required by Kizuna Editor.
var InitialInstruments = []DefaultInstrument{
	{ID: models.InstrumentElectricGuitarID, Name: "Electric Guitar", Type: "String", DefaultClef: "treble"},
	{ID: models.InstrumentBassID, Name: "Bass", Type: "String", DefaultClef: "bass"},
	{ID: models.InstrumentPianoID, Name: "Piano", Type: "Keys", DefaultClef: "treble"},
	{ID: models.InstrumentDrumsID, Name: "Drums", Type: "Percussion", DefaultClef: "percussion"},
	{ID: models.InstrumentVocalsID, Name: "Vocals", Type: "Voice", DefaultClef: "treble"},
}

// SeedInstruments inserts or updates standard instruments in the instruments table.
// It is completely idempotent and runs safely within a database transaction.
func SeedInstruments(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin seeder transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query := `
		INSERT INTO instruments (id, name, type, default_clef)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			type = excluded.type,
			default_clef = excluded.default_clef;
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare seeder statement: %w", err)
	}
	defer stmt.Close()

	for _, inst := range InitialInstruments {
		if _, err := stmt.Exec(inst.ID, inst.Name, inst.Type, inst.DefaultClef); err != nil {
			return fmt.Errorf("failed to seed instrument '%s' (ID %d): %w", inst.Name, inst.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit seeder transaction: %w", err)
	}

	log.Printf("Successfully seeded %d default instruments.", len(InitialInstruments))
	return nil
}
