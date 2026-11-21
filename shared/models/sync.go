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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

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

// ComputeProjectHash calculates a deterministic 16-char hexadecimal SHA-256 content hash
// (Git-style short commit hash) over canonical project metadata, tracks, and audio takes (Issue #21).
func ComputeProjectHash(
	title string,
	bpm int,
	timeSignature string,
	keySignature string,
	tracks []Track,
	audioVersions []AudioVersion,
) string {
	normTitle := strings.TrimSpace(title)
	normBPM := bpm
	if normBPM <= 0 {
		normBPM = 120
	}
	normTS := strings.TrimSpace(timeSignature)
	if normTS == "" {
		normTS = "4/4"
	}
	normKS := strings.TrimSpace(keySignature)

	var b strings.Builder
	fmt.Fprintf(&b, "song:%s|bpm:%d|ts:%s|ks:%s\n", normTitle, normBPM, normTS, normKS)

	for i, tr := range tracks {
		instID := tr.InstrumentID
		if instID <= 0 {
			instID = InstrumentElectricGuitarID
		}
		mode := strings.TrimSpace(tr.DisplayMode)
		if mode == "" {
			mode = DisplayModeBoth
		}
		fmt.Fprintf(
			&b,
			"track[%d]:inst=%d|name=%s|mode=%s|muted=%t|content=%s\n",
			i,
			instID,
			strings.TrimSpace(tr.Name),
			mode,
			tr.IsMuted,
			strings.TrimSpace(tr.DataContent),
		)
	}

	if len(audioVersions) > 0 {
		canonicalTakes := make([]string, 0, len(audioVersions))
		for _, av := range audioVersions {
			entry := fmt.Sprintf(
				"ver=%s|path=%s|notes=%s",
				strings.TrimSpace(av.VersionName),
				strings.TrimSpace(av.FilePath),
				strings.TrimSpace(av.Notes),
			)
			canonicalTakes = append(canonicalTakes, entry)
		}
		sort.Strings(canonicalTakes)
		for i, entry := range canonicalTakes {
			fmt.Fprintf(&b, "audio[%d]:%s\n", i, entry)
		}
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])[:16]
}

// ProjectLock represents an exclusive checkout lease on a song project (Issues #16, #22).
type ProjectLock struct {
	ProjectID  int       `json:"project_id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	AcquiredAt time.Time `json:"acquired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// CloudPushPayload encapsulates the full data snapshot required to backup a project to Kizuna Cloud (Issue #13, #21).
type CloudPushPayload struct {
	ProjectID        int            `json:"project_id"`
	Title            string         `json:"title"`
	BPM              int            `json:"bpm"`
	TimeSignature    string         `json:"time_signature"`
	KeySignature     string         `json:"key_signature"`
	BaseVersion      int            `json:"base_version"`
	ContentHash      string         `json:"content_hash,omitempty"`
	Tracks           []Track        `json:"tracks"`
	AudioVersions    []AudioVersion `json:"audio_versions"`
	EncryptedPayload string         `json:"encrypted_payload,omitempty"`
	EncryptionAlgo   string         `json:"encryption_algo,omitempty"`
	IsEncrypted      bool           `json:"is_encrypted,omitempty"`
}

// CloudPushResult contains the confirmation response received from Kizuna Cloud upon successful sync (Issue #13, #21, #22).
type CloudPushResult struct {
	ProjectID    int       `json:"project_id"`
	NewVersion   int       `json:"new_version"`
	SyncedAt     time.Time `json:"synced_at"`
	Status       string    `json:"status"` // "created", "synced"
	ContentHash  string    `json:"content_hash,omitempty"`
	LockReleased bool      `json:"lock_released,omitempty"`
}

// CloudProject represents a project snapshot stored in Kizuna Cloud (Issues #13, #14, #16, #21).
type CloudProject struct {
	ID               int            `json:"id"`
	OwnerID          string         `json:"owner_id"`
	Title            string         `json:"title"`
	BPM              int            `json:"bpm"`
	TimeSignature    string         `json:"time_signature"`
	KeySignature     string         `json:"key_signature"`
	Version          int            `json:"version"`
	ContentHash      string         `json:"content_hash,omitempty"`
	Tracks           []Track        `json:"tracks"`
	AudioVersions    []AudioVersion `json:"audio_versions"`
	ActiveLock       *ProjectLock   `json:"active_lock,omitempty"`
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
