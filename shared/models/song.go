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

// Song represents the fundamental unit of songwriting in Kizuna Editor.
type Song struct {
	ID            int       `db:"id" json:"id"`
	AlbumID       *int      `db:"album_id" json:"album_id,omitempty"` // Nullable pointer
	Title         string    `db:"title" json:"title"`
	BPM           int       `db:"bpm" json:"bpm"`
	TimeSignature string    `db:"time_signature" json:"time_signature"`
	KeySignature  string    `db:"key_signature" json:"key_signature"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`

	// Relational associations (ignored in direct simple SQL table queries)
	Tracks []Track `db:"-" json:"tracks,omitempty"`
}

// NewSong instantiates a new Song with default settings and timestamp.
func NewSong(title string, bpm int, timeSignature string) (*Song, error) {
	if bpm == 0 {
		bpm = 120
	}
	if strings.TrimSpace(timeSignature) == "" {
		timeSignature = "4/4"
	}

	song := &Song{
		Title:         strings.TrimSpace(title),
		BPM:           bpm,
		TimeSignature: timeSignature,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		Tracks:        make([]Track, 0),
	}

	if err := song.Validate(); err != nil {
		return nil, err
	}
	return song, nil
}

// Validate verifies structural constraints on the Song entity.
func (s *Song) Validate() error {
	if strings.TrimSpace(s.Title) == "" {
		return ErrEmptyTitle
	}
	if s.BPM < 20 || s.BPM > 320 {
		return ErrInvalidBPM
	}
	if s.TimeSignature != "" && !IsValidTimeSignature(s.TimeSignature) {
		return ErrInvalidSignature
	}
	return nil
}

// AddTrack attaches a track to the song's in-memory track collection.
func (s *Song) AddTrack(t Track) {
	s.Tracks = append(s.Tracks, t)
	s.UpdatedAt = time.Now().UTC()
}

// RemoveTrack removes a track by its ID and returns true if found.
func (s *Song) RemoveTrack(trackID int) bool {
	for i, t := range s.Tracks {
		if t.ID == trackID {
			s.Tracks = append(s.Tracks[:i], s.Tracks[i+1:]...)
			s.UpdatedAt = time.Now().UTC()
			return true
		}
	}
	return false
}

// GetTrack returns a pointer to the track matching trackID, or nil if not found.
func (s *Song) GetTrack(trackID int) *Track {
	for i := range s.Tracks {
		if s.Tracks[i].ID == trackID {
			return &s.Tracks[i]
		}
	}
	return nil
}
