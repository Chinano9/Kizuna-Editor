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
	"context"
	"log"
)

// App struct represents the main application state.
type App struct {
	ctx context.Context
	db  *DBManager
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{
		db: NewDBManager(),
	}
}

// startup is called when the app starts.
// The context is saved so we can call the runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("App started successfully")
}

// SaveQuickIdea persists the current editor content to the local database.
// Returns the ID of the saved song or an error if the operation fails.
//
// Exposed to Wails (Frontend).
func (a *App) SaveQuickIdea(id int, title string, content string) (int64, error) {
	// 1. Basic Validation
	if title == "" {
		title = "Untitled Idea"
	}

	log.Printf("Saving song: '%s' (Target ID: %d)", title, id)

	// 2. Database Operation
	// We delegate the logic to the DBManager.
	// Note: If id is 0, the DBManager should interpret it as a CREATE operation.
	savedID := a.db.SaveQuickIdea(id, title, content)

	// In the future, if DBManager returns an error, we should return it here
	// like: return 0, fmt.Errorf("database error")

	return savedID, nil
}
