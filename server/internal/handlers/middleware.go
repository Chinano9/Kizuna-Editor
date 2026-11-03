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
	"context"
	"kizuna/server/internal/auth"
	"kizuna/server/pkg/response"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey = contextKey("user")

// AuthMiddleware validates the Authorization Bearer JWT token on protected endpoints.
func AuthMiddleware(authService *auth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Error(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Error(w, http.StatusUnauthorized, "invalid authorization format, expected 'Bearer <token>'")
			return
		}

		user, err := authService.ValidateSession(parts[1])
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid or expired token", err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// GetUserFromContext retrieves the authenticated User from the request context.
func GetUserFromContext(ctx context.Context) (*auth.User, bool) {
	u, ok := ctx.Value(UserContextKey).(*auth.User)
	return u, ok
}

// CORSMiddleware handles cross-origin requests for web/desktop API access.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
