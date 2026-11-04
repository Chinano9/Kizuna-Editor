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
	"errors"
	"kizuna/server/internal/collaboration"
	"kizuna/server/internal/sync"
	"kizuna/server/pkg/response"
	"net/http"
	"strconv"
	"time"
)

// ProjectHandler manages cloud project listing, details, and collaborative locking (Issues #15, #16, #22).
type ProjectHandler struct {
	syncEngine *sync.Engine
	locker     *collaboration.Locker
}

// NewProjectHandler instantiates a ProjectHandler.
func NewProjectHandler(syncEngine *sync.Engine, locker *collaboration.Locker) *ProjectHandler {
	return &ProjectHandler{
		syncEngine: syncEngine,
		locker:     locker,
	}
}

// ListProjects returns all cloud projects owned by or shared with the authenticated user.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	projects, err := h.syncEngine.ListUserProjects(user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list projects", err.Error())
		return
	}

	response.Success(w, http.StatusOK, projects)
}

// GetProject returns the full project metadata and tracks by ID.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	project, err := h.syncEngine.Pull(projectID)
	if err != nil {
		if errors.Is(err, sync.ErrProjectNotFound) {
			response.Error(w, http.StatusNotFound, "project not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to retrieve project", err.Error())
		return
	}

	// Attach active lock info if present
	lock, isLocked := h.locker.GetLock(projectID)
	type projectWithLock struct {
		*sync.CloudProject
		Lock *collaboration.ProjectLock `json:"active_lock,omitempty"`
	}

	res := projectWithLock{
		CloudProject: project,
	}
	if isLocked {
		res.Lock = lock
	}

	response.Success(w, http.StatusOK, res)
}

// CheckoutProject requests an exclusive checkout lease for collaboration (Issue #16).
func (h *ProjectHandler) CheckoutProject(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	idStr := r.PathValue("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	lock, err := h.locker.AcquireLock(projectID, user.ID, user.Username, 30*time.Minute)
	if err != nil {
		if errors.Is(err, collaboration.ErrProjectLocked) {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to acquire lock", err.Error())
		return
	}

	response.Success(w, http.StatusOK, lock)
}

// ReleaseProject releases an active checkout lease on a project (Issue #22).
func (h *ProjectHandler) ReleaseProject(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	idStr := r.PathValue("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if err := h.locker.ReleaseLock(projectID, user.ID); err != nil {
		if errors.Is(err, collaboration.ErrLockNotHeld) {
			response.Error(w, http.StatusForbidden, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to release lock", err.Error())
		return
	}

	response.Success(w, http.StatusOK, map[string]string{"message": "lock released successfully"})
}
