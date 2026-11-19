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

package bindings

import (
	"context"
	"kizuna/client/internal/service"
	"kizuna/shared/models"
	"log"
)

// SyncHandler provides Wails presentation facade for cloud backup and synchronization (Issue #13).
type SyncHandler struct {
	syncService  *service.SyncClientService
	ctx          context.Context
	lastCloudURL string
	lastToken    string
}

// NewSyncHandler instantiates a SyncHandler.
func NewSyncHandler(syncService *service.SyncClientService) *SyncHandler {
	return &SyncHandler{
		syncService:  syncService,
		ctx:          context.Background(),
		lastCloudURL: "http://localhost:8080",
	}
}

// SetContext assigns the Wails application context to enable desktop runtime event emission (Issue #23).
func (h *SyncHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

// SetCloudSession configures active session parameters for subsequent cloud operations.
func (h *SyncHandler) SetCloudSession(cloudURL string, token string) {
	if cloudURL != "" {
		h.lastCloudURL = cloudURL
	}
	h.lastToken = token
}

// PushProjectToCloud packages local project metadata, tracks, and audio takes, and uploads them to Kizuna Cloud (Issue #13).
func (h *SyncHandler) PushProjectToCloud(cloudURL string, token string, songID int, baseVersion int) (*models.CloudPushResult, error) {
	log.Printf("[SyncHandler] Initiating cloud backup for song ID %d to %s", songID, cloudURL)
	h.SetCloudSession(cloudURL, token)
	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return h.syncService.PushProjectToCloud(ctx, cloudURL, token, songID, baseVersion)
}

// PullAndImportProject downloads a cloud project and unpacks/imports it into local SQLite (Issue #14).
func (h *SyncHandler) PullAndImportProject(cloudURL string, token string, projectID int) (int, error) {
	log.Printf("[SyncHandler] Initiating pull and import for cloud project ID %d from %s", projectID, cloudURL)
	h.SetCloudSession(cloudURL, token)
	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	_, songID, err := h.syncService.PullProjectFromCloud(ctx, cloudURL, token, projectID)
	return songID, err
}

// ListCloudProjects retrieves and locally decrypts the user's cloud projects.
func (h *SyncHandler) ListCloudProjects(cloudURL string, token string) ([]models.CloudProject, error) {
	return h.syncService.ListCloudProjects(context.Background(), cloudURL, token)
}

// GetEncryptionKeyFingerprint returns the short SHA-256 fingerprint of the local E2EE key.
func (h *SyncHandler) GetEncryptionKeyFingerprint() string {
	return h.syncService.GetKeyFingerprint()
}

// ShareProject invites a collaborator by email or username to a remote cloud project (Issue #15).
func (h *SyncHandler) ShareProject(cloudID string, collaboratorIdentifier string) error {
	log.Printf("[SyncHandler] Sharing cloud project %s with %s", cloudID, collaboratorIdentifier)
	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return h.syncService.ShareProject(ctx, h.lastCloudURL, h.lastToken, cloudID, collaboratorIdentifier)
}
