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
	"time"
)

// QuickIdea represents an audio idea that is not associated with a song or track.
type QuickIdea struct {
	ID              int       `db:"id" json:"id"`
	FilePath        string    `db:"file_path" json:"file_path"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	DurationSeconds *float64  `db:"-" json:"duration_seconds,omitempty"`
}

// NewQuickIdea creates and validates an independently recorded audio idea.
func NewQuickIdea(filePath string) (*QuickIdea, error) {
	idea := &QuickIdea{
		FilePath:  strings.TrimSpace(filePath),
		CreatedAt: time.Now().UTC(),
	}
	if err := idea.Validate(); err != nil {
		return nil, err
	}
	return idea, nil
}

// Validate verifies structural constraints on the QuickIdea entity.
func (idea *QuickIdea) Validate() error {
	if strings.TrimSpace(idea.FilePath) == "" {
		return ErrEmptyFilePath
	}
	return nil
}
