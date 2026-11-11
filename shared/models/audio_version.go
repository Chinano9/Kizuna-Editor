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

// AudioVersion represents an audio recording take or version associated with a song and optionally a track.
type AudioVersion struct {
	ID          int       `db:"id" json:"id"`
	SongID      int       `db:"song_id" json:"song_id"`
	TrackID     *int      `db:"track_id" json:"track_id,omitempty"` // Nullable: can be nil for global project audio
	VersionName string    `db:"version_name" json:"version_name"`
	FilePath    string    `db:"file_path" json:"file_path"`
	Notes       string    `db:"notes" json:"notes"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// NewAudioVersion creates and validates an AudioVersion take.
func NewAudioVersion(songID int, trackID *int, versionName, filePath, notes string) (*AudioVersion, error) {
	if versionName == "" {
		versionName = "Take " + time.Now().Format("2006-01-02 15:04")
	}

	av := &AudioVersion{
		SongID:      songID,
		TrackID:     trackID,
		VersionName: strings.TrimSpace(versionName),
		FilePath:    strings.TrimSpace(filePath),
		Notes:       strings.TrimSpace(notes),
		CreatedAt:   time.Now().UTC(),
	}

	if err := av.Validate(); err != nil {
		return nil, err
	}
	return av, nil
}

// Validate verifies structural constraints on the AudioVersion entity.
func (av *AudioVersion) Validate() error {
	if av.SongID <= 0 {
		return ErrInvalidSongID
	}
	if strings.TrimSpace(av.FilePath) == "" {
		return ErrEmptyFilePath
	}
	return nil
}

// IsGlobal returns true if the audio version is a global project take (not tied to a specific track).
func (av *AudioVersion) IsGlobal() bool {
	return av.TrackID == nil
}

// LinkToTrack associates this take with a specific track ID.
func (av *AudioVersion) LinkToTrack(trackID int) {
	av.TrackID = &trackID
}

// UnlinkTrack converts this take to a global project take.
func (av *AudioVersion) UnlinkTrack() {
	av.TrackID = nil
}
