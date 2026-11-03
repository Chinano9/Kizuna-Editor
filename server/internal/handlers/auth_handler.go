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
	"kizuna/server/internal/auth"
	"kizuna/server/pkg/response"
	"net/http"
)

// AuthHandler handles user registration, login, and profile verification (Issues #12, #20).
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler instantiates an AuthHandler.
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User  *auth.User `json:"user"`
	Token string     `json:"token"`
}

// Register handles user sign up.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	user, token, err := h.authService.Register(req.Username, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(w, http.StatusCreated, authResponse{
		User:  user,
		Token: token,
	})
}

// Login handles user authentication and session creation.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	user, token, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) || errors.Is(err, auth.ErrInvalidPassword) {
			response.Error(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		response.Error(w, http.StatusInternalServerError, "login error", err.Error())
		return
	}

	response.Success(w, http.StatusOK, authResponse{
		User:  user,
		Token: token,
	})
}

// Me returns the currently authenticated user's profile.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	response.Success(w, http.StatusOK, user)
}
