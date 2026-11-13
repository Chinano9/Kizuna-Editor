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

import "time"

// CloudPushPayload encapsulates the full data snapshot required to backup a project to Kizuna Cloud (Issue #13).
type CloudPushPayload struct {
	ProjectID     int            `json:"project_id"`
	Title         string         `json:"title"`
	BPM           int            `json:"bpm"`
	TimeSignature string         `json:"time_signature"`
	KeySignature  string         `json:"key_signature"`
	BaseVersion   int            `json:"base_version"`
	Tracks        []Track        `json:"tracks"`
	AudioVersions []AudioVersion `json:"audio_versions"`
}

// CloudPushResult contains the confirmation response received from Kizuna Cloud upon successful sync (Issue #13).
type CloudPushResult struct {
	ProjectID  int       `json:"project_id"`
	NewVersion int       `json:"new_version"`
	SyncedAt   time.Time `json:"synced_at"`
	Status     string    `json:"status"` // "created", "synced"
}

// CloudProject represents a project snapshot stored in Kizuna Cloud (Issues #13, #14).
type CloudProject struct {
	ID            int            `json:"id"`
	OwnerID       string         `json:"owner_id"`
	Title         string         `json:"title"`
	BPM           int            `json:"bpm"`
	TimeSignature string         `json:"time_signature"`
	KeySignature  string         `json:"key_signature"`
	Version       int            `json:"version"`
	Tracks        []Track        `json:"tracks"`
	AudioVersions []AudioVersion `json:"audio_versions"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
