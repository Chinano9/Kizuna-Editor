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

package sync

import (
	"errors"
	"fmt"
	"kizuna/server/internal/collaboration"
	"kizuna/shared/models"
	"strings"
	"sync"
	"time"
)

var (
	ErrProjectNotFound  = errors.New("cloud project not found")
	ErrVersionConflict  = errors.New("cloud version mismatch: local changes are based on an outdated version")
	ErrUnauthorizedSync = errors.New("unauthorized to sync this project")
)

// CloudProject represents a remote project stored in Kizuna Cloud (Issues #13, #14, #16, #21, #22).
type CloudProject struct {
	ID               int                        `json:"id"`
	OwnerID          string                     `json:"owner_id"`
	Collaborators    []string                   `json:"collaborators,omitempty"`
	Title            string                     `json:"title"`
	BPM              int                        `json:"bpm"`
	TimeSignature    string                     `json:"time_signature"`
	KeySignature     string                     `json:"key_signature"`
	Version          int                        `json:"version"`
	ContentHash      string                     `json:"content_hash,omitempty"`
	Tracks           []models.Track             `json:"tracks"`
	AudioVersions    []models.AudioVersion      `json:"audio_versions"`
	ActiveLock       *collaboration.ProjectLock `json:"active_lock,omitempty"`
	EncryptedPayload string                     `json:"encrypted_payload,omitempty"`
	EncryptionAlgo   string                     `json:"encryption_algo,omitempty"`
	IsEncrypted      bool                       `json:"is_encrypted,omitempty"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

// PushRequest represents a sync push from desktop to cloud (Issue #13, #21).
type PushRequest struct {
	ProjectID        int                   `json:"project_id"`
	Title            string                `json:"title"`
	BPM              int                   `json:"bpm"`
	TimeSignature    string                `json:"time_signature"`
	KeySignature     string                `json:"key_signature"`
	BaseVersion      int                   `json:"base_version"` // 0 for initial upload
	ContentHash      string                `json:"content_hash,omitempty"`
	Tracks           []models.Track        `json:"tracks"`
	AudioVersions    []models.AudioVersion `json:"audio_versions"`
	EncryptedPayload string                `json:"encrypted_payload,omitempty"`
	EncryptionAlgo   string                `json:"encryption_algo,omitempty"`
	IsEncrypted      bool                  `json:"is_encrypted,omitempty"`
}

// PushResponse returns the newly assigned cloud version, content hash, and sync timestamp (Issues #13, #21, #22).
type PushResponse struct {
	ProjectID    int       `json:"project_id"`
	NewVersion   int       `json:"new_version"`
	SyncedAt     time.Time `json:"synced_at"`
	Status       string    `json:"status"` // "synced", "created"
	ContentHash  string    `json:"content_hash,omitempty"`
	LockReleased bool      `json:"lock_released,omitempty"`
}

// DiffResponse summarizes changes between client and cloud versions and content hashes (Issue #21).
type DiffResponse struct {
	ProjectID      int       `json:"project_id"`
	ClientVersion  int       `json:"client_version"`
	CloudVersion   int       `json:"cloud_version"`
	ClientHash     string    `json:"client_hash,omitempty"`
	CloudHash      string    `json:"cloud_hash,omitempty"`
	CloudUpdatedAt time.Time `json:"cloud_updated_at"`
	HasChanges     bool      `json:"has_changes"`
	TracksCount    int       `json:"tracks_count"`
	AudioCount     int       `json:"audio_count"`
}

// Engine coordinates cloud sync, version conflict resolution, and differential comparison.
type Engine struct {
	mu       sync.RWMutex
	projects map[int]*CloudProject
	locker   *collaboration.Locker
	nextID   int
}

// NewEngine instantiates a cloud sync engine.
func NewEngine(locker *collaboration.Locker) *Engine {
	return &Engine{
		projects: make(map[int]*CloudProject),
		locker:   locker,
		nextID:   1001,
	}
}

// isProjectCollaborator checks whether a user is either the project owner or an invited collaborator.
func isProjectCollaborator(project *CloudProject, userID string, identities ...string) bool {
	if project.OwnerID == userID {
		return true
	}
	for _, c := range project.Collaborators {
		if strings.EqualFold(c, userID) {
			return true
		}
		for _, id := range identities {
			if id != "" && strings.EqualFold(c, id) {
				return true
			}
		}
	}
	return false
}

// Push receives local project changes and stores them in the cloud (Issues #13, #16, #22, #23).
func (e *Engine) Push(userID string, req PushRequest, identities ...string) (*PushResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now().UTC()
	contentHash := models.ComputeProjectHash(
		req.Title,
		req.BPM,
		req.TimeSignature,
		req.KeySignature,
		req.Tracks,
		req.AudioVersions,
	)

	// 1. Initial push (new cloud project)
	if req.ProjectID == 0 {
		newID := e.nextID
		e.nextID++

		project := &CloudProject{
			ID:               newID,
			OwnerID:          userID,
			Title:            req.Title,
			BPM:              req.BPM,
			TimeSignature:    req.TimeSignature,
			KeySignature:     req.KeySignature,
			Version:          1,
			ContentHash:      contentHash,
			Tracks:           req.Tracks,
			AudioVersions:    req.AudioVersions,
			EncryptedPayload: req.EncryptedPayload,
			EncryptionAlgo:   req.EncryptionAlgo,
			IsEncrypted:      req.IsEncrypted,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		e.projects[newID] = project

		return &PushResponse{
			ProjectID:   newID,
			NewVersion:  1,
			SyncedAt:    now,
			Status:      "created",
			ContentHash: contentHash,
		}, nil
	}

	// 2. Incremental sync on existing project
	project, exists := e.projects[req.ProjectID]
	if !exists {
		return nil, ErrProjectNotFound
	}

	if !isProjectCollaborator(project, userID, identities...) {
		return nil, ErrUnauthorizedSync
	}

	// Verify checkout lock if active (Issue #16)
	if lock, isLocked := e.locker.GetLock(project.ID); isLocked && lock.UserID != userID {
		return nil, fmt.Errorf("%w: user '%s' holds the lock until %s", collaboration.ErrProjectLocked, lock.Username, lock.ExpiresAt.Format(time.RFC3339))
	}

	// Conflict detection: client must base changes on current cloud version (Issue #21)
	if req.BaseVersion > 0 && req.BaseVersion != project.Version {
		return nil, fmt.Errorf("%w: cloud is at v%d, client sent base v%d", ErrVersionConflict, project.Version, req.BaseVersion)
	}

	project.Title = req.Title
	project.BPM = req.BPM
	project.TimeSignature = req.TimeSignature
	project.KeySignature = req.KeySignature
	project.Tracks = req.Tracks
	project.AudioVersions = req.AudioVersions
	project.ContentHash = contentHash
	project.EncryptedPayload = req.EncryptedPayload
	project.EncryptionAlgo = req.EncryptionAlgo
	project.IsEncrypted = req.IsEncrypted
	project.Version++
	project.UpdatedAt = now

	// Automatically release checkout lock after successful push (Issue #22)
	lockReleased := false
	if e.locker != nil {
		if lock, isLocked := e.locker.GetLock(project.ID); isLocked && lock.UserID == userID {
			_ = e.locker.ReleaseLock(project.ID, userID)
			lockReleased = true
		}
	}

	return &PushResponse{
		ProjectID:    project.ID,
		NewVersion:   project.Version,
		SyncedAt:     now,
		Status:       "synced",
		ContentHash:  contentHash,
		LockReleased: lockReleased,
	}, nil
}

// Pull retrieves the latest project data from the cloud (Issue #14).
func (e *Engine) Pull(projectID int) (*CloudProject, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	project, exists := e.projects[projectID]
	if !exists {
		return nil, ErrProjectNotFound
	}

	cp := *project
	if e.locker != nil {
		if lock, isLocked := e.locker.GetLock(projectID); isLocked {
			cp.ActiveLock = lock
		} else {
			cp.ActiveLock = nil
		}
	}

	return &cp, nil
}

// Share adds a collaborator identifier to a cloud project (Issue #15).
func (e *Engine) Share(projectID int, ownerID string, collaboratorIdentifier string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	project, exists := e.projects[projectID]
	if !exists {
		return ErrProjectNotFound
	}
	if project.OwnerID != ownerID {
		return ErrUnauthorizedSync
	}
	for _, c := range project.Collaborators {
		if strings.EqualFold(c, collaboratorIdentifier) {
			return errors.New("el usuario ya es colaborador de este proyecto")
		}
	}
	project.Collaborators = append(project.Collaborators, collaboratorIdentifier)
	return nil
}

// ListUserProjects returns all projects belonging to or shared with a user.
func (e *Engine) ListUserProjects(userID string, identities ...string) ([]*CloudProject, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var list []*CloudProject
	for _, p := range e.projects {
		if isProjectCollaborator(p, userID, identities...) {
			cp := *p
			if e.locker != nil {
				if lock, isLocked := e.locker.GetLock(p.ID); isLocked {
					cp.ActiveLock = lock
				}
			}
			list = append(list, &cp)
		}
	}
	return list, nil
}

// GetDiff compares client version and optional content hash against cloud version (Issue #21).
func (e *Engine) GetDiff(projectID int, clientVersion int, clientHash ...string) (*DiffResponse, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	project, exists := e.projects[projectID]
	if !exists {
		return nil, ErrProjectNotFound
	}

	cHash := ""
	if len(clientHash) > 0 {
		cHash = strings.TrimSpace(clientHash[0])
	}

	hasChanges := clientVersion != project.Version
	if cHash != "" && project.ContentHash != "" {
		if cHash != project.ContentHash {
			hasChanges = true
		} else if clientVersion == project.Version {
			hasChanges = false
		}
	}

	return &DiffResponse{
		ProjectID:      project.ID,
		ClientVersion:  clientVersion,
		CloudVersion:   project.Version,
		ClientHash:     cHash,
		CloudHash:      project.ContentHash,
		CloudUpdatedAt: project.UpdatedAt,
		HasChanges:     hasChanges,
		TracksCount:    len(project.Tracks),
		AudioCount:     len(project.AudioVersions),
	}, nil
}
