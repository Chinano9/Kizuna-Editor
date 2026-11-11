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
	"sync"
	"time"
)

var (
	ErrProjectNotFound  = errors.New("cloud project not found")
	ErrVersionConflict  = errors.New("cloud version mismatch: local changes are based on an outdated version")
	ErrUnauthorizedSync = errors.New("unauthorized to sync this project")
)

// CloudProject represents a remote project stored in Kizuna Cloud (Issues #13, #14).
type CloudProject struct {
	ID            int                   `json:"id"`
	OwnerID       string                `json:"owner_id"`
	Title         string                `json:"title"`
	BPM           int                   `json:"bpm"`
	TimeSignature string                `json:"time_signature"`
	KeySignature  string                `json:"key_signature"`
	Version       int                   `json:"version"`
	Tracks        []models.Track        `json:"tracks"`
	AudioVersions []models.AudioVersion `json:"audio_versions"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// PushRequest represents a sync push from desktop to cloud (Issue #13).
type PushRequest struct {
	ProjectID     int                   `json:"project_id"`
	Title         string                `json:"title"`
	BPM           int                   `json:"bpm"`
	TimeSignature string                `json:"time_signature"`
	KeySignature  string                `json:"key_signature"`
	BaseVersion   int                   `json:"base_version"` // 0 for initial upload
	Tracks        []models.Track        `json:"tracks"`
	AudioVersions []models.AudioVersion `json:"audio_versions"`
}

// PushResponse returns the newly assigned cloud version and sync timestamp.
type PushResponse struct {
	ProjectID  int       `json:"project_id"`
	NewVersion int       `json:"new_version"`
	SyncedAt   time.Time `json:"synced_at"`
	Status     string    `json:"status"` // "synced", "created"
}

// DiffResponse summarizes changes between client and cloud versions (Issue #21).
type DiffResponse struct {
	ProjectID     int  `json:"project_id"`
	ClientVersion int  `json:"client_version"`
	CloudVersion  int  `json:"cloud_version"`
	HasChanges    bool `json:"has_changes"`
	TracksCount   int  `json:"tracks_count"`
	AudioCount    int  `json:"audio_count"`
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

// Push receives local project changes and stores them in the cloud (Issue #13, #23).
func (e *Engine) Push(userID string, req PushRequest) (*PushResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now().UTC()

	// 1. Initial push (new cloud project)
	if req.ProjectID == 0 {
		newID := e.nextID
		e.nextID++

		project := &CloudProject{
			ID:            newID,
			OwnerID:       userID,
			Title:         req.Title,
			BPM:           req.BPM,
			TimeSignature: req.TimeSignature,
			KeySignature:  req.KeySignature,
			Version:       1,
			Tracks:        req.Tracks,
			AudioVersions: req.AudioVersions,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		e.projects[newID] = project

		return &PushResponse{
			ProjectID:  newID,
			NewVersion: 1,
			SyncedAt:   now,
			Status:     "created",
		}, nil
	}

	// 2. Incremental sync on existing project
	project, exists := e.projects[req.ProjectID]
	if !exists {
		return nil, ErrProjectNotFound
	}

	if project.OwnerID != userID {
		return nil, ErrUnauthorizedSync
	}

	// Verify checkout lock if active
	if lock, isLocked := e.locker.GetLock(project.ID); isLocked && lock.UserID != userID {
		return nil, collaboration.ErrProjectLocked
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
	project.Version++
	project.UpdatedAt = now

	return &PushResponse{
		ProjectID:  project.ID,
		NewVersion: project.Version,
		SyncedAt:   now,
		Status:     "synced",
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

	return project, nil
}

// ListUserProjects returns all projects belonging to a user.
func (e *Engine) ListUserProjects(userID string) ([]*CloudProject, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var list []*CloudProject
	for _, p := range e.projects {
		if p.OwnerID == userID {
			list = append(list, p)
		}
	}
	return list, nil
}

// GetDiff compares client version against cloud version (Issue #21).
func (e *Engine) GetDiff(projectID int, clientVersion int) (*DiffResponse, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	project, exists := e.projects[projectID]
	if !exists {
		return nil, ErrProjectNotFound
	}

	return &DiffResponse{
		ProjectID:     project.ID,
		ClientVersion: clientVersion,
		CloudVersion:  project.Version,
		HasChanges:    clientVersion != project.Version,
		TracksCount:   len(project.Tracks),
		AudioCount:    len(project.AudioVersions),
	}, nil
}
