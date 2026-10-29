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

// Allowed Track display modes for AlphaTab and rendering views.
const (
	DisplayModeBoth     = "BOTH"
	DisplayModeTab      = "TAB"
	DisplayModeNotation = "NOTATION"
	DisplayModeScore    = "SCORE"
)

// AudioContentPrefix denotes tracks whose data represents recorded audio rather than AlphaTex.
const AudioContentPrefix = "audio:"

// Track represents a specific instrumental or vocal track within a song.
type Track struct {
	ID           int       `db:"id" json:"id"`
	SongID       int       `db:"song_id" json:"song_id"`
	InstrumentID int       `db:"instrument_id" json:"instrument_id"`
	Name         string    `db:"name" json:"name"`
	DataContent  string    `db:"data_content" json:"data_content"` // AlphaTex code or "audio:<path>"
	DisplayMode  string    `db:"display_mode" json:"display_mode"`
	IsMuted      bool      `db:"is_muted" json:"is_muted"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// NewTrack instantiates a Track with standard defaults.
func NewTrack(songID, instrumentID int, name, dataContent string) (*Track, error) {
	if name == "" {
		name = "New Track"
	}

	track := &Track{
		SongID:       songID,
		InstrumentID: instrumentID,
		Name:         strings.TrimSpace(name),
		DataContent:  dataContent,
		DisplayMode:  DisplayModeBoth,
		IsMuted:      false,
		CreatedAt:    time.Now().UTC(),
	}

	if err := track.Validate(); err != nil {
		return nil, err
	}
	return track, nil
}

// Validate verifies structural constraints on the Track entity.
func (t *Track) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return ErrEmptyName
	}
	if t.SongID < 0 {
		return ErrInvalidSongID
	}
	if t.InstrumentID <= 0 {
		return ErrInvalidInstrumentID
	}

	mode := strings.ToUpper(strings.TrimSpace(t.DisplayMode))
	if mode == "" {
		t.DisplayMode = DisplayModeBoth
	} else if mode != DisplayModeBoth && mode != DisplayModeTab && mode != DisplayModeNotation && mode != DisplayModeScore {
		return ErrInvalidDisplayMode
	}
	return nil
}

// IsAudioTrack returns true if the track points to a native recorded audio file.
func (t *Track) IsAudioTrack() bool {
	return strings.HasPrefix(t.DataContent, AudioContentPrefix)
}

// GetAudioPath returns the audio file path if this is an audio track.
func (t *Track) GetAudioPath() (string, bool) {
	if !t.IsAudioTrack() {
		return "", false
	}
	return strings.TrimPrefix(t.DataContent, AudioContentPrefix), true
}
