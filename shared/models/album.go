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

// Album represents a collection of songs/projects.
type Album struct {
	ID          int       `db:"id" json:"id"`
	Title       string    `db:"title" json:"title"`
	Artist      string    `db:"artist" json:"artist"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// NewAlbum creates a validated Album instance with default timestamp.
func NewAlbum(title, artist, description string) (*Album, error) {
	album := &Album{
		Title:       strings.TrimSpace(title),
		Artist:      strings.TrimSpace(artist),
		Description: strings.TrimSpace(description),
		CreatedAt:   time.Now().UTC(),
	}

	if err := album.Validate(); err != nil {
		return nil, err
	}
	return album, nil
}

// Validate checks for required fields and business constraints.
func (a *Album) Validate() error {
	if strings.TrimSpace(a.Title) == "" {
		return ErrEmptyTitle
	}
	return nil
}
