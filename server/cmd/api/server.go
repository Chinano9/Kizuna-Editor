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

package main

import (
	"encoding/json"
	"kizuna/server/internal/auth"
	"kizuna/server/internal/collaboration"
	"kizuna/server/internal/handlers"
	"kizuna/server/internal/sync"
	"net/http"
	"time"
)

// Config encapsulates server startup parameters.
type Config struct {
	Port      string
	JWTSecret string
	TokenTTL  time.Duration
}

// AppContainer holds all initialized services and handlers for the server.
type AppContainer struct {
	AuthService    *auth.Service
	Locker         *collaboration.Locker
	SyncEngine     *sync.Engine
	AuthHandler    *handlers.AuthHandler
	ProjectHandler *handlers.ProjectHandler
	SyncHandler    *handlers.SyncHandler
}

// NewAppContainer wires together services and handlers.
func NewAppContainer(cfg Config) *AppContainer {
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "kizuna-default-dev-secret-change-in-prod"
	}
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = 24 * time.Hour
	}

	authService := auth.NewService(cfg.JWTSecret, cfg.TokenTTL)
	locker := collaboration.NewLocker()
	syncEngine := sync.NewEngine(locker)

	return &AppContainer{
		AuthService:    authService,
		Locker:         locker,
		SyncEngine:     syncEngine,
		AuthHandler:    handlers.NewAuthHandler(authService),
		ProjectHandler: handlers.NewProjectHandler(syncEngine, locker),
		SyncHandler:    handlers.NewSyncHandler(syncEngine),
	}
}

// SetupRouter registers all HTTP routes and middleware using Go standard library routing.
func SetupRouter(c *AppContainer) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"app":    "kizuna-cloud-api",
			"time":   time.Now().UTC(),
		})
	})

	// Public Auth endpoints
	mux.HandleFunc("POST /api/v1/auth/register", c.AuthHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", c.AuthHandler.Login)

	// Protected Auth endpoints
	mux.HandleFunc("GET /api/v1/auth/me", handlers.AuthMiddleware(c.AuthService, c.AuthHandler.Me))

	// Protected Collaboration & Projects endpoints (Issues #15, #16, #22)
	mux.HandleFunc("GET /api/v1/projects", handlers.AuthMiddleware(c.AuthService, c.ProjectHandler.ListProjects))
	mux.HandleFunc("GET /api/v1/projects/{id}", handlers.AuthMiddleware(c.AuthService, c.ProjectHandler.GetProject))
	mux.HandleFunc("POST /api/v1/projects/{id}/checkout", handlers.AuthMiddleware(c.AuthService, c.ProjectHandler.CheckoutProject))
	mux.HandleFunc("POST /api/v1/projects/{id}/release", handlers.AuthMiddleware(c.AuthService, c.ProjectHandler.ReleaseProject))

	// Protected Sync endpoints (Issues #13, #14, #21, #23)
	mux.HandleFunc("POST /api/v1/sync/push", handlers.AuthMiddleware(c.AuthService, c.SyncHandler.Push))
	mux.HandleFunc("GET /api/v1/sync/pull", handlers.AuthMiddleware(c.AuthService, c.SyncHandler.Pull))
	mux.HandleFunc("GET /api/v1/sync/diff", handlers.AuthMiddleware(c.AuthService, c.SyncHandler.Diff))

	return handlers.CORSMiddleware(mux)
}
