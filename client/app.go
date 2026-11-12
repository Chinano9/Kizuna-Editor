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
	"kizuna/client/bindings"
	"kizuna/client/internal/audio"
	"kizuna/shared/models"
	"log"
)

// App struct represents the main application state and acts as the Wails presentation facade.
type App struct {
	ctx          context.Context
	db           *DBManager
	audioRec     *audio.AudioRecorder
	audioPlay    *audio.AudioPlayer
	songHandler  *bindings.SongHandler
	trackHandler *bindings.TrackHandler
	albumHandler *bindings.AlbumHandler
	audioHandler *bindings.AudioHandler
	syncHandler  *bindings.SyncHandler
}

// NewApp creates a new App application struct wiring up clean dependency injection.
func NewApp() *App {
	db := NewDBManager()
	audioRec := audio.NewAudioRecorder()
	audioPlay := audio.NewAudioPlayer()

	songHandler := bindings.NewSongHandler(db.SongService())
	trackHandler := bindings.NewTrackHandler(db.TrackService(), db.InstrumentRepository())
	albumHandler := bindings.NewAlbumHandler(db.AlbumService())
	audioHandler := bindings.NewAudioHandler(audioRec, audioPlay, db.AudioService(), db.QuickIdeaService())
	syncHandler := bindings.NewSyncHandler(db.SyncClientService())

	return &App{
		db:           db,
		audioRec:     audioRec,
		audioPlay:    audioPlay,
		songHandler:  songHandler,
		trackHandler: trackHandler,
		albumHandler: albumHandler,
		audioHandler: audioHandler,
		syncHandler:  syncHandler,
	}
}

// startup is called by Wails when the application boots up.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("Kizuna Editor started successfully")
}

// shutdown is called by Wails when the app is terminating to release resources cleanly.
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			log.Println("Error closing database during shutdown:", err)
		}
	}
	audio.TeardownSharedContext()
	log.Println("Kizuna Editor shutdown complete")
}

// Handler accessors for modular Wails binding
func (a *App) SongHandler() *bindings.SongHandler   { return a.songHandler }
func (a *App) TrackHandler() *bindings.TrackHandler { return a.trackHandler }
func (a *App) AlbumHandler() *bindings.AlbumHandler { return a.albumHandler }
func (a *App) AudioHandler() *bindings.AudioHandler { return a.audioHandler }
func (a *App) SyncHandler() *bindings.SyncHandler   { return a.syncHandler }

// PushProjectToCloud packages and backs up the local project to Kizuna Cloud (Issue #13).
func (a *App) PushProjectToCloud(cloudURL string, token string, songID int, baseVersion int) (*models.CloudPushResult, error) {
	return a.syncHandler.PushProjectToCloud(cloudURL, token, songID, baseVersion)
}

// --- Song Operations (Delegated to SongHandler) ---

// SaveQuickIdea persists the current editor content to the local database.
func (a *App) SaveQuickIdea(id int, title string, content string) (int64, error) {
	return a.songHandler.SaveQuickIdea(id, title, content)
}

// GetSong retrieves a song by its ID.
func (a *App) GetSong(id int) (*models.Song, error) {
	return a.songHandler.GetSong(id)
}

// SaveSong saves the entire song object, including all its tracks.
func (a *App) SaveSong(song *models.Song) (*models.Song, error) {
	return a.songHandler.SaveSong(song)
}

// GetRecentSongs returns recent songs from the database.
func (a *App) GetRecentSongs() ([]models.Song, error) {
	return a.songHandler.GetRecentSongs()
}

// --- Track Operations (Delegated to TrackHandler) ---

// AddTrack creates a new track for a given song.
func (a *App) AddTrack(songID int, trackName string) (*models.Track, error) {
	return a.trackHandler.AddTrack(songID, trackName)
}

// DeleteTrack removes a track by ID.
func (a *App) DeleteTrack(trackID int) error {
	return a.trackHandler.DeleteTrack(trackID)
}

// UpdateTrack updates track details.
func (a *App) UpdateTrack(track *models.Track) error {
	return a.trackHandler.UpdateTrack(track)
}

// GetInstruments retrieves all catalog instruments.
func (a *App) GetInstruments() ([]models.Instrument, error) {
	return a.trackHandler.GetInstruments()
}

// --- Audio Capture & Playback Operations (Delegated to AudioHandler) ---

// StartRecording starts capturing audio from the designated input device.
func (a *App) StartRecording(songID int, trackName string, deviceID string) error {
	return a.audioHandler.StartRecording(songID, trackName, deviceID)
}

// StartQuickIdeaRecording starts a standalone audio idea recording.
func (a *App) StartQuickIdeaRecording(deviceID string) error {
	return a.audioHandler.StartQuickIdeaRecording(deviceID)
}

// StopRecording stops capture, saves WAV to disk, and registers the take in the database.
func (a *App) StopRecording(trackID int) (*models.AudioVersion, error) {
	return a.audioHandler.StopRecording(trackID)
}

// StopQuickIdeaRecording saves and registers the active standalone audio idea.
func (a *App) StopQuickIdeaRecording() (*models.QuickIdea, error) {
	return a.audioHandler.StopQuickIdeaRecording()
}

// GetQuickIdeas lists saved standalone audio ideas.
func (a *App) GetQuickIdeas() ([]models.QuickIdea, error) {
	return a.audioHandler.GetQuickIdeas()
}

// ConvertQuickIdeaToSong creates a song from a saved idea and returns the new song ID.
func (a *App) ConvertQuickIdeaToSong(ideaID int, title string) (int, error) {
	return a.db.ConvertQuickIdeaToSong(ideaID, title)
}

// DeleteQuickIdea removes a saved idea and its WAV file.
func (a *App) DeleteQuickIdea(ideaID int) error {
	return a.db.DeleteQuickIdea(ideaID)
}

// GetAudioLevels returns the current real-time normalized audio peak amplitude [0.0, 1.0].
func (a *App) GetAudioLevels() float64 {
	return a.audioHandler.GetAudioLevels()
}

// GetInputDevices queries all available physical capture devices.
func (a *App) GetInputDevices() ([]audio.AudioDevice, error) {
	return a.audioHandler.GetInputDevices()
}

// GetAvailableDrivers returns the list of audio driver backends for the current OS.
func (a *App) GetAvailableDrivers() []string {
	return a.audioHandler.GetAvailableDrivers()
}

// GetCurrentDriver returns the name of the active audio driver backend.
func (a *App) GetCurrentDriver() string {
	return a.audioHandler.GetCurrentDriver()
}

// SetAudioDriver updates the active audio driver backend.
func (a *App) SetAudioDriver(driverName string) error {
	return a.audioHandler.SetAudioDriver(driverName)
}

// PlayAudio loads the WAV file and starts playback.
func (a *App) PlayAudio(filePath string) error {
	return a.audioHandler.PlayAudio(filePath)
}

// PauseAudio pauses or resumes active audio playback.
func (a *App) PauseAudio() error {
	return a.audioHandler.PauseAudio()
}

// StopAudio stops playback and releases the output device.
func (a *App) StopAudio() error {
	return a.audioHandler.StopAudio()
}

// SeekAudio shifts playback progress to the given second offset.
func (a *App) SeekAudio(seconds float64) error {
	return a.audioHandler.SeekAudio(seconds)
}

// SetAudioVolume updates playback volume in range [0.0, 1.0].
func (a *App) SetAudioVolume(volume float64) error {
	return a.audioHandler.SetAudioVolume(volume)
}

// GetPlaybackPosition returns current playback progress in seconds.
func (a *App) GetPlaybackPosition() float64 {
	return a.audioHandler.GetPlaybackPosition()
}

// GetPlaybackDuration returns total duration of loaded WAV in seconds.
func (a *App) GetPlaybackDuration() float64 {
	return a.audioHandler.GetPlaybackDuration()
}

// GetAudioVersionsForTrack retrieves all audio takes associated with a track.
func (a *App) GetAudioVersionsForTrack(trackID int) ([]models.AudioVersion, error) {
	return a.audioHandler.GetAudioVersionsForTrack(trackID)
}

// GetAudioVersionsForSong retrieves all audio takes for a song.
func (a *App) GetAudioVersionsForSong(songID int) ([]models.AudioVersion, error) {
	return a.audioHandler.GetAudioVersionsForSong(songID)
}

// --- Album Operations (Delegated to AlbumHandler - Issues #6, #7, #8, #17, #18) ---

// CreateAlbum creates a new empty album (Issue #6).
func (a *App) CreateAlbum(title, artist, description string) (*models.Album, error) {
	return a.albumHandler.CreateAlbum(title, artist, description)
}

// GetAlbum retrieves an album by ID (Issue #8).
func (a *App) GetAlbum(id int) (*models.Album, error) {
	return a.albumHandler.GetAlbum(id)
}

// ListAlbums lists all albums in the library (Issue #8).
func (a *App) ListAlbums() ([]models.Album, error) {
	return a.albumHandler.ListAlbums()
}

// UpdateAlbum updates an album's metadata (Issue #18).
func (a *App) UpdateAlbum(album *models.Album) error {
	return a.albumHandler.UpdateAlbum(album)
}

// DeleteAlbum deletes an album (Issue #18).
func (a *App) DeleteAlbum(id int) error {
	return a.albumHandler.DeleteAlbum(id)
}

// AssignSongToAlbum assigns a song to an album (Issue #7).
func (a *App) AssignSongToAlbum(songID int, albumID int) error {
	return a.albumHandler.AssignSongToAlbum(songID, albumID)
}

// RemoveSongFromAlbum removes a song from its album (Issue #17).
func (a *App) RemoveSongFromAlbum(songID int) error {
	return a.albumHandler.RemoveSongFromAlbum(songID)
}

// GetAlbumSongs retrieves all songs in an album (Issue #8).
func (a *App) GetAlbumSongs(albumID int) ([]models.Song, error) {
	return a.albumHandler.GetAlbumSongs(albumID)
}
