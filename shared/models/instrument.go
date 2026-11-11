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

package models

import (
	"strings"
)

// Standard instrument IDs seeded in the database catalog.
const (
	InstrumentElectricGuitarID = 1
	InstrumentBassID           = 2
	InstrumentPianoID          = 3
	InstrumentDrumsID          = 4
	InstrumentVocalsID         = 5
)

// Instrument represents an entry in the catalog of available instruments.
type Instrument struct {
	ID          int    `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Type        string `db:"type" json:"type"` // e.g. 'String', 'Keys', 'Percussion', 'Voice'
	DefaultClef string `db:"default_clef" json:"default_clef"`
}

// NewInstrument instantiates a new Instrument entity.
func NewInstrument(id int, name, instType, defaultClef string) (*Instrument, error) {
	if defaultClef == "" {
		defaultClef = "treble"
	}

	inst := &Instrument{
		ID:          id,
		Name:        strings.TrimSpace(name),
		Type:        strings.TrimSpace(instType),
		DefaultClef: strings.ToLower(strings.TrimSpace(defaultClef)),
	}

	if err := inst.Validate(); err != nil {
		return nil, err
	}
	return inst, nil
}

// Validate verifies structural constraints on the Instrument entity.
func (i *Instrument) Validate() error {
	if strings.TrimSpace(i.Name) == "" {
		return ErrEmptyName
	}
	if i.DefaultClef != "" && !IsValidClef(i.DefaultClef) {
		return ErrInvalidClef
	}
	return nil
}
