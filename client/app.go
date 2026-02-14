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
	"kizuna/shared/models"
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
	savedID, err := a.db.SaveQuickIdea(id, title, content)
	if err != nil {
		return 0, err
	}

	// In the future, if DBManager returns an error, we should return it here
	// like: return 0, fmt.Errorf("database error")

	return savedID, nil
}

// GetSong retrieves a song by its ID from the database.
// Exposed to Wails (Frontend).
func (a *App) GetSong(id int) (*models.Song, error) {
	return a.db.GetSong(id)
}

// AddTrack creates a new track for a given song.
// Exposed to Wails (Frontend).
func (a *App) AddTrack(songID int, trackName string) (*models.Track, error) {
	log.Printf("Adding new track '%s' to song ID %d", trackName, songID)
	return a.db.AddTrack(songID, trackName)
}

// DeleteTrack removes a track from the database.
// Exposed to Wails (Frontend).
func (a *App) DeleteTrack(trackID int) error {
	log.Printf("Deleting track ID %d", trackID)
	return a.db.DeleteTrack(trackID)
}

// UpdateTrack updates the details of an existing track.
// Exposed to Wails (Frontend).
func (a *App) UpdateTrack(track *models.Track) error {
	if track == nil {
		log.Println("UpdateTrack called with nil track")
		return nil // Or return an error
	}
	log.Printf("Updating track ID %d with name '%s'", track.ID, track.Name)
	return a.db.UpdateTrack(track)
}

// GetInstruments retrieves all available instruments from the database.
// Exposed to Wails (Frontend).
func (a *App) GetInstruments() ([]models.Instrument, error) {
	return a.db.GetInstruments()
}

// SaveSong saves the entire song object, including all its tracks.
// This is the new primary save method.
// Exposed to Wails (Frontend).
func (a *App) SaveSong(song *models.Song) (*models.Song, error) {
	if song == nil {
		log.Println("SaveSong called with a nil song object.")
		// Or return an error: return nil, errors.New("cannot save a nil song")
		return nil, nil
	}
	log.Printf("Saving full song object for '%s' (ID: %d)", song.Title, song.ID)
	return a.db.SaveSong(song)
}

// Gets all recent songs from the database.
// Exposed to Wails (Frontend).
func (a *App) GetRecentSongs() ([]models.Song, error) {
	return a.db.GetRecentSongs()
}

// shutdown is called when the app is shutting down.
// It attempts to close the DB connection cleanly. Wails will call this if
// you set OnShutdown to this method in main.
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			log.Println("Error closing DB during shutdown:", err)
		}
	}
	log.Println("App shutdown complete")
}
