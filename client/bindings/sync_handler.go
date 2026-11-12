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
	syncService *service.SyncClientService
}

// NewSyncHandler instantiates a SyncHandler.
func NewSyncHandler(syncService *service.SyncClientService) *SyncHandler {
	return &SyncHandler{syncService: syncService}
}

// PushProjectToCloud packages local project metadata, tracks, and audio takes, and uploads them to Kizuna Cloud (Issue #13).
func (h *SyncHandler) PushProjectToCloud(cloudURL string, token string, songID int, baseVersion int) (*models.CloudPushResult, error) {
	log.Printf("[SyncHandler] Initiating cloud backup for song ID %d to %s", songID, cloudURL)
	return h.syncService.PushProjectToCloud(context.Background(), cloudURL, token, songID, baseVersion)
}
