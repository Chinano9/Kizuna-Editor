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

package handlers

import (
	"encoding/json"
	"errors"
	"kizuna/server/internal/collaboration"
	"kizuna/server/internal/sync"
	"kizuna/server/pkg/response"
	"net/http"
	"strconv"
)

// SyncHandler manages push and pull synchronization between desktop and cloud (Issues #13, #14, #21, #23).
type SyncHandler struct {
	syncEngine *sync.Engine
}

// NewSyncHandler instantiates a SyncHandler.
func NewSyncHandler(syncEngine *sync.Engine) *SyncHandler {
	return &SyncHandler{syncEngine: syncEngine}
}

// Push uploads local changes to the cloud.
func (h *SyncHandler) Push(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var req sync.PushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	res, err := h.syncEngine.Push(user.ID, req)
	if err != nil {
		if errors.Is(err, sync.ErrVersionConflict) || errors.Is(err, collaboration.ErrProjectLocked) {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, sync.ErrProjectNotFound) {
			response.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, sync.ErrUnauthorizedSync) {
			response.Error(w, http.StatusForbidden, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to push project", err.Error())
		return
	}

	response.Success(w, http.StatusOK, res)
}

// Pull downloads the latest cloud project version.
func (h *SyncHandler) Pull(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil || projectID <= 0 {
		response.Error(w, http.StatusBadRequest, "missing or invalid project_id query parameter")
		return
	}

	project, err := h.syncEngine.Pull(projectID)
	if err != nil {
		if errors.Is(err, sync.ErrProjectNotFound) {
			response.Error(w, http.StatusNotFound, "cloud project not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to pull project", err.Error())
		return
	}

	response.Success(w, http.StatusOK, project)
}

// Diff compares client version with cloud version to display pending changes (Issue #21).
func (h *SyncHandler) Diff(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil || projectID <= 0 {
		response.Error(w, http.StatusBadRequest, "missing or invalid project_id query parameter")
		return
	}

	verStr := r.URL.Query().Get("client_version")
	clientVersion, _ := strconv.Atoi(verStr)

	diff, err := h.syncEngine.GetDiff(projectID, clientVersion)
	if err != nil {
		if errors.Is(err, sync.ErrProjectNotFound) {
			response.Error(w, http.StatusNotFound, "cloud project not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to compute diff", err.Error())
		return
	}

	response.Success(w, http.StatusOK, diff)
}
