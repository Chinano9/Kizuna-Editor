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
	"fmt"
	"kizuna/client/recorder"
	"kizuna/shared/models"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// App struct represents the main application state.
type App struct {
	ctx           context.Context
	db            *DBManager
	audioRecorder *recorder.AudioRecorder
	audioPlayer   *recorder.AudioPlayer
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{
		db:            NewDBManager(),
		audioRecorder: recorder.NewAudioRecorder(),
		audioPlayer:   recorder.NewAudioPlayer(),
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
	recorder.TeardownSharedContext()
	log.Println("App shutdown complete")
}

// StartRecording starts capturing audio from the designated input device.
// Exposed to Wails (Frontend).
func (a *App) StartRecording(songID int, trackName string, deviceID string) error {
	log.Printf("StartRecording called for Song ID: %d, Track Name: '%s', Device ID: %s", songID, trackName, deviceID)
	if a.audioRecorder == nil {
		return fmt.Errorf("audio recorder is not initialized")
	}
	return a.audioRecorder.StartRecording(songID, trackName, deviceID)
}

// StopRecording stops active audio capture, saves the PCM data directly to disk as a WAV file,
// registers the take as an AudioVersion (tied to trackID if > 0, otherwise global), and returns the new take struct.
// Exposed to Wails (Frontend).
func (a *App) StopRecording(trackID int) (*models.AudioVersion, error) {
	log.Printf("StopRecording called with Target Track ID: %d", trackID)
	if a.audioRecorder == nil {
		return nil, fmt.Errorf("audio recorder is not initialized")
	}

	// 1. Get state before stopping
	songID := a.audioRecorder.SongID
	trackName := a.audioRecorder.TrackName

	// 2. Stop hardware capture and retrieve PCM frames
	pcmData, sampleRate, channels, bitsPerSample, err := a.audioRecorder.StopRecording()
	if err != nil {
		return nil, err
	}

	// 3. Resolve absolute project WAV path
	wavPath, err := a.getProjectAudioPath(songID, trackName, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve audio path: %w", err)
	}

	// 4. Save raw PCM bytes directly to disk in WAV format
	log.Printf("Saving audio file to: %s", wavPath)
	if err := recorder.WriteWavFile(wavPath, pcmData, sampleRate, channels, bitsPerSample); err != nil {
		return nil, fmt.Errorf("failed to save WAV file to disk: %w", err)
	}

	// 5. Register the take inside SQLite 'audio_versions' table
	var dbTrackID *int
	var versionName string
	if trackID > 0 {
		dbTrackID = &trackID
		versionName = fmt.Sprintf("Take: %s", trackName)
	} else {
		dbTrackID = nil
		versionName = fmt.Sprintf("Global take: %s", trackName)
	}

	log.Printf("Registering AudioVersion '%s' in database (Track ID pointer: %v)...", versionName, dbTrackID)
	av, err := a.db.AddAudioVersion(songID, dbTrackID, versionName, wavPath, "Native Go recorded take")
	if err != nil {
		return nil, fmt.Errorf("failed to save audio version to database: %w", err)
	}

	log.Printf("Audio version successfully recorded and registered: ID %d", av.ID)
	return av, nil
}

// GetAudioLevels retrieves the current real-time normalized audio peak amplitude [0.0, 1.0].
// Exposed to Wails (Frontend).
func (a *App) GetAudioLevels() float64 {
	if a.audioRecorder == nil {
		return 0.0
	}
	return a.audioRecorder.GetAudioLevels()
}

// GetInputDevices queries malgo to list all available physical microphone/capture devices.
// Exposed to Wails (Frontend).
func (a *App) GetInputDevices() ([]recorder.AudioDevice, error) {
	return recorder.GetInputDevices()
}

// GetAvailableDrivers returns the list of audio driver backends supported by the active host OS.
// Exposed to Wails (Frontend).
func (a *App) GetAvailableDrivers() []string {
	return recorder.GetAvailableDrivers()
}

// GetCurrentDriver returns the name of the currently active audio driver backend.
// Exposed to Wails (Frontend).
func (a *App) GetCurrentDriver() string {
	return recorder.GetCurrentDriver()
}

// SetAudioDriver updates the active audio backend in Go and reinitializes the shared context.
// Exposed to Wails (Frontend).
func (a *App) SetAudioDriver(driverName string) error {
	log.Printf("SetAudioDriver called: transitioning to driver '%s'", driverName)
	return recorder.SetAudioDriver(driverName)
}

// PlayAudio loads the WAV audio file and starts native CGO playback.
// Exposed to Wails (Frontend).
func (a *App) PlayAudio(filePath string) error {
	log.Printf("PlayAudio called for file: %s", filePath)
	if a.audioPlayer == nil {
		return fmt.Errorf("audio player is not initialized")
	}
	return a.audioPlayer.Play(filePath)
}

// PauseAudio pauses or resumes the active audio playback.
// Exposed to Wails (Frontend).
func (a *App) PauseAudio() error {
	log.Println("PauseAudio called")
	if a.audioPlayer == nil {
		return fmt.Errorf("audio player is not initialized")
	}
	return a.audioPlayer.Pause()
}

// SeekAudio shifts the playback progress to the designated timestamp in seconds.
// Exposed to Wails (Frontend).
func (a *App) SeekAudio(seconds float64) error {
	log.Printf("SeekAudio called for offset: %.2fs", seconds)
	if a.audioPlayer == nil {
		return fmt.Errorf("audio player is not initialized")
	}
	return a.audioPlayer.Seek(seconds)
}

// SetAudioVolume updates the real-time gain multiplier in Go in the range [0.0, 1.0].
// Exposed to Wails (Frontend).
func (a *App) SetAudioVolume(volume float64) error {
	if a.audioPlayer == nil {
		return fmt.Errorf("audio player is not initialized")
	}
	return a.audioPlayer.SetVolume(volume)
}

// GetPlaybackPosition returns the current playback progress in seconds.
// Exposed to Wails (Frontend).
func (a *App) GetPlaybackPosition() float64 {
	if a.audioPlayer == nil {
		return 0.0
	}
	return a.audioPlayer.GetPosition()
}

// GetPlaybackDuration returns the total duration of the currently loaded WAV in seconds.
// Exposed to Wails (Frontend).
func (a *App) GetPlaybackDuration() float64 {
	if a.audioPlayer == nil {
		return 0.0
	}
	return a.audioPlayer.GetDuration()
}

// GetAudioVersionsForTrack retrieves all audio versions associated with a specific track.
// Exposed to Wails (Frontend).
func (a *App) GetAudioVersionsForTrack(trackID int) ([]models.AudioVersion, error) {
	return a.db.GetAudioVersionsForTrack(trackID)
}

// GetAudioVersionsForSong retrieves all audio versions for a song, including those that are global (track_id IS NULL).
// Exposed to Wails (Frontend).
func (a *App) GetAudioVersionsForSong(songID int) ([]models.AudioVersion, error) {
	return a.db.GetAudioVersionsForSong(songID)
}

// getProjectAudioPath determines the absolute OS-appropriate project path to save the .wav file.
func (a *App) getProjectAudioPath(songID int, trackName string, trackID int) (string, error) {
	safeTrackName := cleanFileName(trackName)
	timestamp := time.Now().Format("20060102_150405")
	
	var filename string
	if trackID > 0 {
		filename = fmt.Sprintf("track_%d_%s_%s.wav", trackID, safeTrackName, timestamp)
	} else {
		filename = fmt.Sprintf("global_%s_%s.wav", safeTrackName, timestamp)
	}

	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appdata, "Kizuna", "projects", fmt.Sprintf("%d", songID), "audio", filename), nil
	case "darwin":
		configDir, err := os.UserConfigDir()
		if err != nil {
			home, herr := os.UserHomeDir()
			if herr != nil {
				return "", err
			}
			configDir = filepath.Join(home, "Library", "Application Support")
		}
		return filepath.Join(configDir, "Kizuna", "projects", fmt.Sprintf("%d", songID), "audio", filename), nil
	default:
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg != "" {
			return filepath.Join(xdg, "Kizuna", "projects", fmt.Sprintf("%d", songID), "audio", filename), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "Kizuna", "projects", fmt.Sprintf("%d", songID), "audio", filename), nil
	}
}

// cleanFileName removes characters that are illegal or unsafe for filenames.
func cleanFileName(name string) string {
	// Simple sanitize logic: replace illegal filename characters with underscores
	var result []rune
	for _, r := range name {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			result = append(result, '_')
		} else {
			result = append(result, r)
		}
	}
	cleaned := string(result)
	if cleaned == "" {
		cleaned = "recorded_track"
	}
	return cleaned
}
