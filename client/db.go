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
	"database/sql"
	"fmt"
	"log"

	"kizuna/client/internal/database"
	"kizuna/client/internal/database/migrator"
	"kizuna/client/internal/repository"
	"kizuna/client/internal/service"
	"kizuna/shared/models"

	_ "modernc.org/sqlite"
)

const (
	InstrumentGuitarID = models.InstrumentElectricGuitarID
	defaultDBName      = database.DefaultDBName
	appFolderName      = database.DefaultAppFolder
)

// DBManager serves as a facade coordinating data services and repository abstractions for the Wails desktop app.
type DBManager struct {
	db           *sql.DB
	songService  *service.SongService
	trackService *service.TrackService
	albumService *service.AlbumService
	audioService *service.AudioService
	instRepo     repository.InstrumentRepository
}

// NewDBManager initializes the SQLite connection, executes pending versioned migrations,
// seeds the default instrument catalog, and instantiates the domain service layer.
func NewDBManager() *DBManager {
	dbPath, err := database.ResolveDBPath(appFolderName, defaultDBName)
	if err != nil {
		log.Printf("Warning: couldn't determine default DB path (%v), using local fallback: %s\n", err, defaultDBName)
		dbPath = defaultDBName
	}

	if err := database.EnsureDirectory(dbPath); err != nil {
		log.Printf("Warning: couldn't create DB directory (%v), falling back to local file: %s\n", err, defaultDBName)
		dbPath = defaultDBName
	}

	cfg := database.DefaultConfig(dbPath)
	db, err := database.Open(cfg)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Initialize and execute versioned migrations
	m := migrator.NewMigrator(db)
	if err := m.Migrate(); err != nil {
		db.Close()
		log.Fatal("Failed to execute database migrations:", err)
	}

	// Seed initial instrument catalog (idempotent)
	if err := database.SeedInstruments(db); err != nil {
		log.Println("Warning: Failed to seed instruments:", err)
	}

	return NewDBManagerWithDB(db)
}

// NewDBManagerWithDB instantiates DBManager with an existing database connection (e.g., for testing).
func NewDBManagerWithDB(db *sql.DB) *DBManager {
	songRepo := repository.NewSongRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	albumRepo := repository.NewAlbumRepository(db)
	audioRepo := repository.NewAudioRepository(db)
	instRepo := repository.NewInstrumentRepository(db)

	return &DBManager{
		db:           db,
		songService:  service.NewSongService(db, songRepo, trackRepo),
		trackService: service.NewTrackService(trackRepo),
		albumService: service.NewAlbumService(albumRepo, songRepo),
		audioService: service.NewAudioService(audioRepo),
		instRepo:     instRepo,
	}
}

// Close safely closes the underlying database connection.
func (m *DBManager) Close() error {
	if m == nil || m.db == nil {
		return nil
	}
	return m.db.Close()
}

// SongService returns the underlying SongService for direct access.
func (m *DBManager) SongService() *service.SongService {
	return m.songService
}

// TrackService returns the underlying TrackService for direct access.
func (m *DBManager) TrackService() *service.TrackService {
	return m.trackService
}

// AlbumService returns the underlying AlbumService for direct access.
func (m *DBManager) AlbumService() *service.AlbumService {
	return m.albumService
}

// AudioService returns the underlying AudioService for direct access.
func (m *DBManager) AudioService() *service.AudioService {
	return m.audioService
}

// SaveQuickIdea handles rapid idea creation and updates.
func (m *DBManager) SaveQuickIdea(songID int, title string, content string) (int64, error) {
	if m == nil || m.songService == nil {
		return 0, fmt.Errorf("database service is not initialized")
	}
	return m.songService.SaveQuickIdea(context.Background(), songID, title, content)
}

// GetSong retrieves a song and its associated tracks.
func (m *DBManager) GetSong(id int) (*models.Song, error) {
	if m == nil || m.songService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.songService.GetSong(context.Background(), id)
}

// AddTrack creates a new track for a song with default values.
func (m *DBManager) AddTrack(songID int, trackName string) (*models.Track, error) {
	if m == nil || m.trackService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.trackService.AddTrack(context.Background(), songID, trackName)
}

// AddAudioTrack creates a new track for a song linked to a recorded audio file.
func (m *DBManager) AddAudioTrack(songID int, trackName string, filePath string) (*models.Track, error) {
	if m == nil || m.trackService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.trackService.AddAudioTrack(context.Background(), songID, trackName, filePath)
}

// DeleteTrack removes a track by its ID.
func (m *DBManager) DeleteTrack(trackID int) error {
	if m == nil || m.trackService == nil {
		return fmt.Errorf("database service is not initialized")
	}
	return m.trackService.DeleteTrack(context.Background(), trackID)
}

// UpdateTrack updates all fields of a given track.
func (m *DBManager) UpdateTrack(track *models.Track) error {
	if m == nil || m.trackService == nil {
		return fmt.Errorf("database service is not initialized")
	}
	return m.trackService.UpdateTrack(context.Background(), track)
}

// GetRecentSongs returns a list of recent songs for the dashboard.
func (m *DBManager) GetRecentSongs() ([]models.Song, error) {
	if m == nil || m.songService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.songService.GetRecentSongs(context.Background(), 10)
}

// GetInstruments retrieves all available catalog instruments.
func (m *DBManager) GetInstruments() ([]models.Instrument, error) {
	if m == nil || m.instRepo == nil {
		return nil, fmt.Errorf("database repository is not initialized")
	}
	return m.instRepo.GetAll(context.Background())
}

// SaveSong executes an atomic upsert of a song and its associated tracks.
func (m *DBManager) SaveSong(song *models.Song) (*models.Song, error) {
	if m == nil || m.songService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.songService.SaveSong(context.Background(), song)
}

// AddAudioVersion logs a new recorded take linked to a song and optionally a track.
func (m *DBManager) AddAudioVersion(songID int, trackID *int, versionName string, filePath string, notes string) (*models.AudioVersion, error) {
	if m == nil || m.audioService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.audioService.RegisterAudioVersion(context.Background(), songID, trackID, versionName, filePath, notes)
}

// GetAudioVersionsForTrack retrieves all takes for a specific track.
func (m *DBManager) GetAudioVersionsForTrack(trackID int) ([]models.AudioVersion, error) {
	if m == nil || m.audioService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.audioService.GetAudioVersionsForTrack(context.Background(), trackID)
}

// GetAudioVersionsForSong retrieves all takes for a song (including global takes).
func (m *DBManager) GetAudioVersionsForSong(songID int) ([]models.AudioVersion, error) {
	if m == nil || m.audioService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.audioService.GetAudioVersionsForSong(context.Background(), songID)
}

// CreateAlbum creates a new album container (Issue #6).
func (m *DBManager) CreateAlbum(title, artist, description string) (*models.Album, error) {
	if m == nil || m.albumService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.albumService.CreateAlbum(context.Background(), title, artist, description)
}

// GetAlbum retrieves an album by its ID (Issue #8).
func (m *DBManager) GetAlbum(id int) (*models.Album, error) {
	if m == nil || m.albumService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.albumService.GetAlbum(context.Background(), id)
}

// ListAlbums lists all albums in the library (Issue #8).
func (m *DBManager) ListAlbums() ([]models.Album, error) {
	if m == nil || m.albumService == nil {
		return nil, fmt.Errorf("database service is not initialized")
	}
	return m.albumService.ListAlbums(context.Background())
}

// AssignSongToAlbum links a song to an album (Issue #7).
func (m *DBManager) AssignSongToAlbum(songID int, albumID int) error {
	if m == nil || m.albumService == nil {
		return fmt.Errorf("database service is not initialized")
	}
	return m.albumService.AssignSongToAlbum(context.Background(), songID, albumID)
}
