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

// AudioFileBlob represents a physical .wav audio file packaged inside an E2EE project bundle.
type AudioFileBlob struct {
	TrackIndex   int    `json:"track_index"`
	VersionIndex int    `json:"version_index"`
	FileName     string `json:"file_name"`
	OriginalPath string `json:"original_path"`
	Data         []byte `json:"data"`
}

// EncryptedProjectBundle contains the full SQLite project data and raw .wav audio files
// before AES-256-GCM encryption on the client.
type EncryptedProjectBundle struct {
	Title         string          `json:"title"`
	BPM           int             `json:"bpm"`
	TimeSignature string          `json:"time_signature"`
	KeySignature  string          `json:"key_signature"`
	Tracks        []Track         `json:"tracks"`
	AudioVersions []AudioVersion  `json:"audio_versions"`
	AudioFiles    []AudioFileBlob `json:"audio_files,omitempty"`
}

// CloudPushPayload encapsulates the full data snapshot required to backup a project to Kizuna Cloud (Issue #13).
type CloudPushPayload struct {
	ProjectID        int            `json:"project_id"`
	Title            string         `json:"title"`
	BPM              int            `json:"bpm"`
	TimeSignature    string         `json:"time_signature"`
	KeySignature     string         `json:"key_signature"`
	BaseVersion      int            `json:"base_version"`
	Tracks           []Track        `json:"tracks"`
	AudioVersions    []AudioVersion `json:"audio_versions"`
	EncryptedPayload string         `json:"encrypted_payload,omitempty"`
	EncryptionAlgo   string         `json:"encryption_algo,omitempty"`
	IsEncrypted      bool           `json:"is_encrypted,omitempty"`
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
	ID               int            `json:"id"`
	OwnerID          string         `json:"owner_id"`
	Title            string         `json:"title"`
	BPM              int            `json:"bpm"`
	TimeSignature    string         `json:"time_signature"`
	KeySignature     string         `json:"key_signature"`
	Version          int            `json:"version"`
	Tracks           []Track        `json:"tracks"`
	AudioVersions    []AudioVersion `json:"audio_versions"`
	EncryptedPayload string         `json:"encrypted_payload,omitempty"`
	EncryptionAlgo   string         `json:"encryption_algo,omitempty"`
	IsEncrypted      bool           `json:"is_encrypted,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// SyncProgressPayload encapsulates real-time byte transfer progress emitted to Wails (Issue #23).
type SyncProgressPayload struct {
	Operation        string  `json:"operation"`         // "push" | "pull"
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes"`       // -1 if unknown
	Percentage       float64 `json:"percentage"`        // 0.0 to 100.0
	Phase            string  `json:"phase"`             // "transferring" | "retrying" | "completed" | "failed"
	Attempt          int     `json:"attempt"`
	MaxAttempts      int     `json:"max_attempts"`
}

// SyncRetryPayload notifies the frontend of an unstable connection and scheduled retry backoff (Issue #23).
type SyncRetryPayload struct {
	Operation   string `json:"operation"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts"`
	WaitSeconds int    `json:"wait_seconds"`
	Reason      string `json:"reason"`
}
